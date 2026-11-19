"""LLM provider abstraction.

Every provider speaks the same canonical types (LLMRequest / LLMResponse /
LLMStreamEvent), shaped after the Bedrock Converse API. Business code must
never branch on which provider is active.
"""
from __future__ import annotations

import asyncio
import json
import re
import uuid
from collections.abc import AsyncIterator, Callable
from dataclasses import dataclass, field
from typing import Protocol

from ..config import Settings
from ..observability import LLM_CALLS, LLM_TOKENS


@dataclass
class Message:
    role: str  # user | assistant
    content: str


@dataclass
class ToolSpec:
    name: str
    description: str
    input_schema: dict


@dataclass
class ToolCall:
    id: str
    name: str
    arguments: dict


@dataclass
class GenerationConfig:
    temperature: float = 0.3
    max_tokens: int = 700


@dataclass
class LLMRequest:
    messages: list[Message]
    system_prompt: str | None = None
    tools: list[ToolSpec] = field(default_factory=list)
    config: GenerationConfig = field(default_factory=GenerationConfig)
    metadata: dict = field(default_factory=dict)


@dataclass
class Usage:
    input_tokens: int = 0
    output_tokens: int = 0


@dataclass
class LLMResponse:
    content: str = ""
    tool_calls: list[ToolCall] = field(default_factory=list)
    finish_reason: str = "stop"
    usage: Usage = field(default_factory=Usage)


@dataclass
class LLMStreamEvent:
    kind: str  # text_delta | tool_call | usage | completed
    text: str = ""
    tool_call: ToolCall | None = None
    usage: Usage | None = None


class LLMProvider(Protocol):
    async def generate(self, request: LLMRequest) -> LLMResponse: ...

    def stream(self, request: LLMRequest) -> AsyncIterator[LLMStreamEvent]: ...


class ProviderError(RuntimeError):
    pass


class ObservedProvider:
    """Metrics decorator that keeps provider concerns out of business workflows."""

    def __init__(self, provider: str, inner: LLMProvider):
        self.provider = provider
        self.inner = inner

    def _usage(self, usage: Usage) -> None:
        LLM_TOKENS.labels(provider=self.provider, direction="input").inc(
            usage.input_tokens
        )
        LLM_TOKENS.labels(provider=self.provider, direction="output").inc(
            usage.output_tokens
        )

    async def generate(self, request: LLMRequest) -> LLMResponse:
        try:
            response = await self.inner.generate(request)
        except asyncio.CancelledError:
            LLM_CALLS.labels(provider=self.provider, result="error").inc()
            raise
        except Exception:
            LLM_CALLS.labels(provider=self.provider, result="error").inc()
            raise
        LLM_CALLS.labels(provider=self.provider, result="ok").inc()
        self._usage(response.usage)
        return response

    async def stream(self, request: LLMRequest) -> AsyncIterator[LLMStreamEvent]:
        try:
            async for event in self.inner.stream(request):
                if event.kind == "usage" and event.usage is not None:
                    self._usage(event.usage)
                yield event
        except asyncio.CancelledError:
            LLM_CALLS.labels(provider=self.provider, result="error").inc()
            raise
        except Exception:
            LLM_CALLS.labels(provider=self.provider, result="error").inc()
            raise
        LLM_CALLS.labels(provider=self.provider, result="ok").inc()


# ---------------------------------------------------------------------------
# FakeProvider — deterministic; used by all automated tests and CI.
# ---------------------------------------------------------------------------

_BUDGET_RE = re.compile(r"(\d+(?:[.,]\d+)?)\s*(triệu|trieu|nghìn|nghin|k\b|đ\b|vnd)", re.IGNORECASE)
_RECIPIENTS = ("bố mẹ", "ba mẹ", "mẹ", "bố", "ba", "ông", "bà", "sếp", "bạn gái", "bạn trai",
               "vợ", "chồng", "thầy", "cô", "đồng nghiệp", "khách hàng", "bạn")
_OCCASIONS = ("tết", "sinh nhật", "trung thu", "20/10", "8/3", "giáng sinh", "noel",
              "tri ân", "khai trương", "đám cưới", "thôi nôi")
_TASTES = ("ngọt", "mặn", "ít đường", "không đường", "cay", "truyền thống", "giòn", "dẻo")


def _extract_slots(text: str) -> dict:
    low = text.lower()
    slots: dict = {}
    match = _BUDGET_RE.search(low)
    if match:
        value = float(match.group(1).replace(",", "."))
        unit = match.group(2).lower()
        amount = value * 1_000_000 if unit.startswith("tri") else value * 1_000
        slots["budget_vnd"] = int(amount)
    for word in _RECIPIENTS:
        if word in low:
            slots["recipient"] = word
            break
    for word in _OCCASIONS:
        if word in low:
            slots["occasion"] = word
            break
    for word in _TASTES:
        if word in low:
            slots.setdefault("tastes", []).append(word)
    return slots


class FakeProvider:
    """Deterministic provider.

    - If the request offers an `extract_needs` tool, answers with a tool call
      whose arguments are extracted from the conversation by simple rules.
    - Otherwise answers with text: a custom responder if given, else a reply
      composed from the [GROUNDING] block the workflow put in the prompt.
    - Can simulate provider errors and latency for resilience tests.
    """

    def __init__(
        self,
        responder: Callable[[LLMRequest], str] | None = None,
        fail_with: str | None = None,  # "error" | "timeout"
        delay_seconds: float = 0.0,
    ):
        self.responder = responder
        self.fail_with = fail_with
        self.delay_seconds = delay_seconds

    async def _pre(self) -> None:
        if self.delay_seconds:
            await asyncio.sleep(self.delay_seconds)
        if self.fail_with == "timeout":
            raise TimeoutError("fake provider timeout")
        if self.fail_with == "error":
            raise ProviderError("fake provider error")

    async def generate(self, request: LLMRequest) -> LLMResponse:
        await self._pre()
        tool_names = {tool.name for tool in request.tools}
        if "extract_needs" in tool_names:
            joined = "\n".join(m.content for m in request.messages if m.role == "user")
            slots = _extract_slots(joined)
            return LLMResponse(
                tool_calls=[ToolCall(id=uuid.uuid4().hex, name="extract_needs", arguments=slots)],
                finish_reason="tool_use",
                usage=Usage(input_tokens=len(joined) // 4, output_tokens=16),
            )
        if self.responder is not None:
            text = self.responder(request)
        else:
            text = self._grounded_reply(request)
        return LLMResponse(
            content=text,
            finish_reason="stop",
            usage=Usage(input_tokens=32, output_tokens=max(1, len(text) // 4)),
        )

    def _grounded_reply(self, request: LLMRequest) -> str:
        prompt = request.system_prompt or ""
        if "[GROUNDING]" in prompt:
            block = prompt.split("[GROUNDING]", 1)[1].strip()
            return "Dạ, em tra cứu trực tiếp từ cửa hàng và gợi ý như sau:\n" + block
        return (
            "Dạ, em có thể hỗ trợ anh/chị chọn đặc sản làm quà. "
            "Anh/chị cho em biết thêm ngân sách và người nhận nhé ạ."
        )

    async def stream(self, request: LLMRequest) -> AsyncIterator[LLMStreamEvent]:
        response = await self.generate(request)
        for call in response.tool_calls:
            yield LLMStreamEvent(kind="tool_call", tool_call=call)
        if response.content:
            for i in range(0, len(response.content), 24):
                yield LLMStreamEvent(kind="text_delta", text=response.content[i:i + 24])
        yield LLMStreamEvent(kind="usage", usage=response.usage)
        yield LLMStreamEvent(kind="completed")


# ---------------------------------------------------------------------------
# GeminiProvider — dev substitute via the OpenAI-compatible endpoint.
# ---------------------------------------------------------------------------

class GeminiProvider:
    def __init__(self, settings: Settings):
        self.settings = settings

    def _client(self):
        if not self.settings.gemini_api_key:
            raise ProviderError("GEMINI_API_KEY is not configured")
        import httpx

        return httpx.AsyncClient(
            base_url=self.settings.gemini_base_url,
            headers={"Authorization": f"Bearer {self.settings.gemini_api_key}"},
            timeout=30.0,
        )

    def _payload(self, request: LLMRequest, stream: bool) -> dict:
        messages = []
        if request.system_prompt:
            messages.append({"role": "system", "content": request.system_prompt})
        messages += [{"role": m.role, "content": m.content} for m in request.messages]
        payload: dict = {
            "model": self.settings.gemini_model,
            "messages": messages,
            "temperature": request.config.temperature,
            "max_tokens": request.config.max_tokens,
            "stream": stream,
        }
        if request.tools:
            payload["tools"] = [
                {
                    "type": "function",
                    "function": {
                        "name": t.name,
                        "description": t.description,
                        "parameters": t.input_schema,
                    },
                }
                for t in request.tools
            ]
        return payload

    @staticmethod
    def _parse_message(message: dict) -> LLMResponse:
        calls = []
        for raw in message.get("tool_calls") or []:
            fn = raw.get("function", {})
            try:
                args = json.loads(fn.get("arguments") or "{}")
            except json.JSONDecodeError:
                args = {}
            calls.append(ToolCall(id=raw.get("id", uuid.uuid4().hex),
                                  name=fn.get("name", ""), arguments=args))
        return LLMResponse(content=message.get("content") or "", tool_calls=calls)

    async def generate(self, request: LLMRequest) -> LLMResponse:
        try:
            async with self._client() as client:
                resp = await client.post("/chat/completions", json=self._payload(request, False))
                resp.raise_for_status()
                data = resp.json()
            choice = data["choices"][0]
            result = self._parse_message(choice.get("message", {}))
            result.finish_reason = choice.get("finish_reason") or "stop"
            usage = data.get("usage") or {}
            result.usage = Usage(
                input_tokens=usage.get("prompt_tokens", 0),
                output_tokens=usage.get("completion_tokens", 0),
            )
            return result
        except ProviderError:
            raise
        except Exception as exc:
            raise ProviderError("Gemini generation failed") from exc

    async def stream(self, request: LLMRequest) -> AsyncIterator[LLMStreamEvent]:
        async with self._client() as client:
            async with client.stream(
                "POST", "/chat/completions", json=self._payload(request, True)
            ) as resp:
                resp.raise_for_status()
                async for line in resp.aiter_lines():
                    if not line.startswith("data:"):
                        continue
                    data = line[5:].strip()
                    if data == "[DONE]":
                        break
                    chunk = json.loads(data)
                    delta = chunk["choices"][0].get("delta", {})
                    if delta.get("content"):
                        yield LLMStreamEvent(kind="text_delta", text=delta["content"])
                    usage = chunk.get("usage")
                    if usage:
                        yield LLMStreamEvent(
                            kind="usage",
                            usage=Usage(
                                input_tokens=usage.get("prompt_tokens", 0),
                                output_tokens=usage.get("completion_tokens", 0),
                            ),
                        )
        yield LLMStreamEvent(kind="completed")


# ---------------------------------------------------------------------------
# BedrockProvider — production (Converse API, model decided: Nova Pro).
# ---------------------------------------------------------------------------

class BedrockProvider:
    def __init__(self, settings: Settings):
        self.settings = settings
        self._session = None

    def _client(self):
        import aioboto3

        if self._session is None:
            self._session = aioboto3.Session()
        return self._session.client(
            "bedrock-runtime", region_name=self.settings.aws_region
        )

    def _converse_kwargs(self, request: LLMRequest) -> dict:
        kwargs: dict = {
            "modelId": self.settings.model_id,
            "messages": [
                {"role": m.role, "content": [{"text": m.content}]} for m in request.messages
            ],
            "inferenceConfig": {
                "temperature": request.config.temperature,
                "maxTokens": request.config.max_tokens,
            },
        }
        if request.system_prompt:
            kwargs["system"] = [{"text": request.system_prompt}]
        if request.tools:
            kwargs["toolConfig"] = {
                "tools": [
                    {
                        "toolSpec": {
                            "name": t.name,
                            "description": t.description,
                            "inputSchema": {"json": t.input_schema},
                        }
                    }
                    for t in request.tools
                ]
            }
        return kwargs

    @staticmethod
    def _parse_output(data: dict) -> LLMResponse:
        content_parts: list[str] = []
        calls: list[ToolCall] = []
        message = (data.get("output") or {}).get("message") or {}
        for block in message.get("content") or []:
            if "text" in block:
                content_parts.append(block["text"])
            if "toolUse" in block:
                use = block["toolUse"]
                calls.append(
                    ToolCall(
                        id=use.get("toolUseId", uuid.uuid4().hex),
                        name=use.get("name", ""),
                        arguments=use.get("input") or {},
                    )
                )
        usage = data.get("usage") or {}
        return LLMResponse(
            content="".join(content_parts),
            tool_calls=calls,
            finish_reason=data.get("stopReason") or "stop",
            usage=Usage(
                input_tokens=usage.get("inputTokens", 0),
                output_tokens=usage.get("outputTokens", 0),
            ),
        )

    async def generate(self, request: LLMRequest) -> LLMResponse:
        try:
            async with self._client() as client:
                data = await client.converse(**self._converse_kwargs(request))
            return self._parse_output(data)
        except Exception as exc:
            raise ProviderError("Bedrock generation failed") from exc

    async def stream(self, request: LLMRequest) -> AsyncIterator[LLMStreamEvent]:
        async with self._client() as client:
            response = await client.converse_stream(**self._converse_kwargs(request))
            pending: dict[int, dict] = {}
            async for event in response["stream"]:
                if "contentBlockDelta" in event:
                    delta = event["contentBlockDelta"].get("delta") or {}
                    if "text" in delta:
                        yield LLMStreamEvent(kind="text_delta", text=delta["text"])
                    if "toolUse" in event["contentBlockDelta"]:
                        pass
                if "contentBlockStart" in event:
                    start = event["contentBlockStart"]
                    idx = start.get("contentBlockIndex", 0)
                    tool_use = (start.get("start") or {}).get("toolUse") or {}
                    if tool_use:
                        pending[idx] = {
                            "id": tool_use.get("toolUseId", uuid.uuid4().hex),
                            "name": tool_use.get("name", ""),
                            "json": "",
                        }
                if "contentBlockDelta" in event:
                    delta = event["contentBlockDelta"].get("delta") or {}
                    tool_delta = delta.get("toolUse") or {}
                    idx = event["contentBlockDelta"].get("contentBlockIndex", 0)
                    if tool_delta and idx in pending:
                        pending[idx]["json"] += tool_delta.get("input", "")
                if "contentBlockStop" in event:
                    idx = event["contentBlockStop"].get("contentBlockIndex", 0)
                    if idx in pending:
                        raw = pending.pop(idx)
                        try:
                            args = json.loads(raw["json"] or "{}")
                        except json.JSONDecodeError:
                            args = {}
                        yield LLMStreamEvent(
                            kind="tool_call",
                            tool_call=ToolCall(id=raw["id"], name=raw["name"], arguments=args),
                        )
                if "metadata" in event:
                    usage = (event["metadata"] or {}).get("usage") or {}
                    if usage:
                        yield LLMStreamEvent(
                            kind="usage",
                            usage=Usage(
                                input_tokens=usage.get("inputTokens", 0),
                                output_tokens=usage.get("outputTokens", 0),
                            ),
                        )
        yield LLMStreamEvent(kind="completed")
