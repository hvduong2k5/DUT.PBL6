"""Unit tests for the budget evidence gate (docs ch. 14, ADR-01).

The parser expectations come straight from the adversarial case file
(tests/evaluation/budget_cases.json) that also drives the recorded
Nova replay — one source of truth for what the user's words support.
"""
import json
import pathlib

import pytest

from ai_agent_service.platform.budget_evidence import (
    ABSENT,
    AMBIGUOUS,
    INVALID,
    parse_budget_evidence,
    resolve_budget,
)

CASES = json.loads(
    (pathlib.Path(__file__).parent / "evaluation" / "budget_cases.json").read_text("utf-8")
)


@pytest.mark.parametrize("case", CASES, ids=[c["id"] for c in CASES])
def test_parser_matches_expected_evidence(case):
    evidence = parse_budget_evidence(case["turns"])
    expected = case["expected"]
    assert evidence.status == expected["status"], (
        f"{case['id']}: status {evidence.status} != {expected['status']}"
    )
    if "value_vnd" in expected:
        assert evidence.value_vnd == expected["value_vnd"], case["id"]
    if "low_vnd" in expected:
        assert evidence.low_vnd == expected["low_vnd"], case["id"]
        assert evidence.high_vnd == expected["high_vnd"], case["id"]


def test_gate_blocks_model_repair_of_invalid_budget():
    # The S06 recording: user wrote 5đ, Nova Pro answered 5000000.
    turns = ["Ngân sách của tôi là 5đ, mua quà cho mẹ"]
    evidence = parse_budget_evidence(turns)
    assert evidence.status == INVALID
    budget, decision = resolve_budget(evidence, 5_000_000, turns)
    assert budget is None
    assert decision == "invalid_blocked"


def test_gate_evidence_overrides_model():
    turns = ["Ngân sách 200k, mua quà cho mẹ"]
    evidence = parse_budget_evidence(turns)
    budget, decision = resolve_budget(evidence, 999_999, turns)
    assert (budget, decision) == (200_000, "evidence")


def test_gate_absent_rejects_invented_budget():
    turns = ["Ngân sách sinh viên thôi, mua quà cho mẹ"]
    evidence = parse_budget_evidence(turns)
    assert evidence.status == ABSENT
    budget, decision = resolve_budget(evidence, 5_000_000, turns)
    assert budget is None
    assert decision == "model_unsupported"


def test_gate_absent_accepts_literal_supported_budget():
    turns = ["Cho mẹ 200k nhé, tính giúp em"]
    evidence = parse_budget_evidence(turns)
    assert evidence.status == ABSENT
    budget, decision = resolve_budget(evidence, 200_000, turns)
    assert (budget, decision) == (200_000, "model_supported")


def test_gate_absent_rejects_product_price_as_budget():
    turns = ["Mua hộp quà 169.000đ này thì còn gợi ý gì nữa không?"]
    evidence = parse_budget_evidence(turns)
    assert evidence.status == ABSENT
    budget, decision = resolve_budget(evidence, 169_000, turns)
    assert budget is None
    assert decision == "model_unsupported"


def test_gate_ambiguous_never_guesses():
    turns = ["Ngân sách tầm 2-3 trăm nghìn, mua quà cho mẹ"]
    evidence = parse_budget_evidence(turns)
    assert evidence.status == AMBIGUOUS
    budget, decision = resolve_budget(evidence, 250_000, turns)
    assert budget is None
    assert decision == "ambiguous_clarify"


def test_gate_no_budget_anywhere():
    turns = ["Mua quà cho mẹ"]
    evidence = parse_budget_evidence(turns)
    budget, decision = resolve_budget(evidence, None, turns)
    assert (budget, decision) == (None, "no_budget")
