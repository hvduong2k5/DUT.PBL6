"""Controlled agent runtime: budget enforcement around a deterministic workflow.

The workflow (code) owns the step order; the LLM only decides inside its
allowance. This class is the allowance: step and tool-call budgets per turn.
"""
from __future__ import annotations

from dataclasses import dataclass, field

from .gateway import ToolContext, ToolGateway


class BudgetExceeded(Exception):
    pass


@dataclass
class TurnState:
    max_steps: int = 8
    max_tool_calls: int = 6
    steps: int = 0
    tool_calls: int = 0
    tool_log: list[dict] = field(default_factory=list)

    def step(self) -> None:
        self.steps += 1
        if self.steps > self.max_steps:
            raise BudgetExceeded(f"step budget exceeded ({self.max_steps})")

    async def call_tool(
        self, gateway: ToolGateway, name: str, arguments: dict, ctx: ToolContext
    ) -> dict:
        self.tool_calls += 1
        if self.tool_calls > self.max_tool_calls:
            raise BudgetExceeded(f"tool-call budget exceeded ({self.max_tool_calls})")
        result = await gateway.execute(name, arguments, ctx)
        self.tool_log.append({"tool": name, "args": arguments, "result": result})
        return result
