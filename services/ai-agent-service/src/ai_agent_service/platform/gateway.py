"""Tool Execution Gateway — the single mandatory path for every tool call.

Order of control: registry allowlist -> customer requirement -> argument
schema validation -> timeout -> execution -> result validation -> audit.
The LLM never calls HTTP or a database directly.
"""
from __future__ import annotations

import asyncio
import time
from collections.abc import Awaitable, Callable
from dataclasses import dataclass

from pydantic import BaseModel, ValidationError

from ..observability import TOOL_CALLS, TOOL_SECONDS


class ToolError(Exception):
    def __init__(self, code: str, message: str = ""):
        super().__init__(message or code)
        self.code = code


@dataclass
class ToolContext:
    customer_id: str | None = None  # None for guests
    run_id: str | None = None


Handler = Callable[[ToolContext, BaseModel], Awaitable[dict]]


@dataclass
class Tool:
    name: str
    description: str
    args_model: type[BaseModel]
    handler: Handler
    requires_customer: bool = False


class ToolRegistry:
    def __init__(self):
        self._tools: dict[str, Tool] = {}

    def register(self, tool: Tool) -> None:
        self._tools[tool.name] = tool

    def get(self, name: str) -> Tool | None:
        return self._tools.get(name)

    def names(self) -> list[str]:
        return sorted(self._tools)


class ToolGateway:
    def __init__(
        self,
        registry: ToolRegistry,
        timeout_seconds: float = 2.0,
        recorder: Callable[[dict], None] | None = None,
    ):
        self.registry = registry
        self.timeout_seconds = timeout_seconds
        self.recorder = recorder

    async def execute(self, name: str, arguments: dict, ctx: ToolContext) -> dict:
        started = time.monotonic()
        status = "ok"
        try:
            tool = self.registry.get(name)
            if tool is None:
                raise ToolError("tool_not_allowed", f"tool is not registered: {name}")
            if tool.requires_customer and not ctx.customer_id:
                raise ToolError("customer_required", f"tool {name} requires a logged-in customer")
            try:
                args = tool.args_model.model_validate(arguments or {})
            except ValidationError as exc:
                raise ToolError("invalid_arguments", str(exc)) from exc
            try:
                result = await asyncio.wait_for(
                    tool.handler(ctx, args), timeout=self.timeout_seconds
                )
            except TimeoutError as exc:
                raise ToolError("tool_timeout", f"tool {name} exceeded timeout") from exc
            if not isinstance(result, dict):
                raise ToolError("invalid_tool_result", f"tool {name} must return an object")
            return result
        except ToolError as exc:
            status = exc.code
            raise
        except Exception:
            status = "error"
            raise
        finally:
            latency = time.monotonic() - started
            TOOL_CALLS.labels(tool=name, result=status).inc()
            TOOL_SECONDS.labels(tool=name).observe(latency)
            if self.recorder is not None:
                self.recorder(
                    {
                        "run_id": ctx.run_id,
                        "tool_name": name,
                        "args": arguments,
                        "status": status,
                        "latency_ms": int(latency * 1000),
                    }
                )
