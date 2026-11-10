from __future__ import annotations

import logging

from prometheus_client import CONTENT_TYPE_LATEST, Counter, Histogram, generate_latest

CHAT_TURNS = Counter(
    "ai_chat_turns", "Shopping chat turns processed", ["outcome"]
)
LLM_CALLS = Counter(
    "ai_llm_calls", "LLM provider calls", ["provider", "result"]
)
LLM_TOKENS = Counter(
    "ai_llm_tokens", "LLM tokens consumed", ["provider", "direction"]
)
TOOL_CALLS = Counter(
    "ai_tool_calls", "Tool Gateway executions", ["tool", "result"]
)
TOOL_SECONDS = Histogram(
    "ai_tool_seconds", "Tool Gateway execution seconds", ["tool"]
)
RETRIEVAL_RESULTS = Histogram(
    "ai_retrieval_results", "Knowledge retrieval result count", buckets=(0, 1, 2, 3, 5, 8, 13)
)
ELIGIBILITY_REJECTIONS = Counter(
    "ai_eligibility_rejections",
    "Product cards rejected by the unified eligibility layer",
    ["source", "outcome"],
)
RECOMMENDATION_OUTCOMES = Counter(
    "ai_recommendation_outcomes",
    "Analytics recommendation fetch outcomes",
    ["placement", "outcome"],
)


def setup_logging() -> None:
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s %(levelname)s %(name)s %(message)s",
    )


def metrics_response() -> tuple[bytes, str]:
    return generate_latest(), CONTENT_TYPE_LATEST
BUDGET_GATE = Counter(
    "ai_budget_gate", "Budget evidence gate decisions", ["decision"]
)
