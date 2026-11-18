import asyncio

import pytest
from pydantic import BaseModel

from ai_agent_service.platform.gateway import (
    Tool,
    ToolContext,
    ToolError,
    ToolGateway,
    ToolRegistry,
)
from ai_agent_service.platform.runtime import BudgetExceeded, TurnState


class EchoArgs(BaseModel):
    text: str


async def echo(ctx, args):
    return {"echo": args.text, "customer": ctx.customer_id}


async def slow(ctx, args):
    await asyncio.sleep(5)
    return {}


def make_gateway(timeout=1.0, recorder=None):
    registry = ToolRegistry()
    registry.register(Tool("echo", "echo", EchoArgs, echo))
    registry.register(Tool("mine", "customer only", EchoArgs, echo, requires_customer=True))
    registry.register(Tool("slow", "slow", EchoArgs, slow))
    return ToolGateway(registry, timeout_seconds=timeout, recorder=recorder)


@pytest.mark.asyncio
async def test_gateway_executes_and_records():
    records = []
    gateway = make_gateway(recorder=records.append)
    result = await gateway.execute("echo", {"text": "hi"}, ToolContext(customer_id="c1"))
    assert result == {"echo": "hi", "customer": "c1"}
    assert records and records[0]["status"] == "ok"


@pytest.mark.asyncio
async def test_gateway_rejects_unregistered_tool():
    gateway = make_gateway()
    with pytest.raises(ToolError) as exc:
        await gateway.execute("delete_everything", {}, ToolContext())
    assert exc.value.code == "tool_not_allowed"


@pytest.mark.asyncio
async def test_gateway_validates_arguments():
    gateway = make_gateway()
    with pytest.raises(ToolError) as exc:
        await gateway.execute("echo", {"wrong": 1}, ToolContext())
    assert exc.value.code == "invalid_arguments"


@pytest.mark.asyncio
async def test_gateway_requires_customer():
    gateway = make_gateway()
    with pytest.raises(ToolError) as exc:
        await gateway.execute("mine", {"text": "x"}, ToolContext(customer_id=None))
    assert exc.value.code == "customer_required"


@pytest.mark.asyncio
async def test_gateway_timeout():
    gateway = make_gateway(timeout=0.1)
    with pytest.raises(ToolError) as exc:
        await gateway.execute("slow", {"text": "x"}, ToolContext())
    assert exc.value.code == "tool_timeout"


@pytest.mark.asyncio
async def test_turn_budget_enforced():
    gateway = make_gateway()
    state = TurnState(max_steps=2, max_tool_calls=1)
    ctx = ToolContext()
    await state.call_tool(gateway, "echo", {"text": "a"}, ctx)
    with pytest.raises(BudgetExceeded):
        await state.call_tool(gateway, "echo", {"text": "b"}, ctx)
