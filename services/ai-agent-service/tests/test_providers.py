import pytest

from ai_agent_service.providers.llm import (
    FakeProvider,
    LLMRequest,
    Message,
    ProviderError,
    ToolSpec,
)

EXTRACT_TOOL = ToolSpec(
    name="extract_needs",
    description="Extract shopping needs",
    input_schema={"type": "object", "properties": {}},
)


@pytest.mark.asyncio
async def test_fake_extracts_slots_via_tool_call():
    provider = FakeProvider()
    response = await provider.generate(
        LLMRequest(
            messages=[Message("user", "Mua quà tết cho bố mẹ, ngân sách 500k, thích đồ ngọt")],
            tools=[EXTRACT_TOOL],
        )
    )
    assert response.finish_reason == "tool_use"
    assert len(response.tool_calls) == 1
    args = response.tool_calls[0].arguments
    assert args["budget_vnd"] == 500_000
    assert args["recipient"] == "bố mẹ"
    assert args["occasion"] == "tết"
    assert "ngọt" in args["tastes"]


@pytest.mark.asyncio
async def test_fake_grounded_reply_uses_grounding_block():
    provider = FakeProvider()
    response = await provider.generate(
        LLMRequest(
            messages=[Message("user", "gợi ý quà")],
            system_prompt="Rules...\n[GROUNDING]\n- Mè xửng dẻo — 49.000đ — còn hàng",
        )
    )
    assert "49.000đ" in response.content
    assert response.tool_calls == []


@pytest.mark.asyncio
async def test_fake_streaming_events():
    provider = FakeProvider()
    events = [event async for event in provider.stream(LLMRequest(messages=[Message("user", "hi")]))]
    kinds = [e.kind for e in events]
    assert kinds[-1] == "completed"
    assert "text_delta" in kinds
    assert "usage" in kinds


@pytest.mark.asyncio
async def test_fake_error_modes():
    with pytest.raises(ProviderError):
        await FakeProvider(fail_with="error").generate(LLMRequest(messages=[]))
    with pytest.raises(TimeoutError):
        await FakeProvider(fail_with="timeout").generate(LLMRequest(messages=[]))
