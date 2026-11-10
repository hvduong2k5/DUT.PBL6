"""Offline replay of the recorded Nova extractions (budget_recordings.json)
through the budget evidence gate — Suite A, deterministic, no model calls.

Asserts the hybrid gate's safety metrics on the adversarial set:
unsupported acceptance == 0 and invalid acceptance == 0, plus the
per-group outcomes the debate (ADR-01) decided.
"""
import json
import pathlib

import pytest

from ai_agent_service.platform.budget_evidence import (
    ABSENT,
    AMBIGUOUS,
    EXACT,
    INVALID,
    literal_amounts,
    parse_budget_evidence,
    resolve_budget,
)

HERE = pathlib.Path(__file__).parent
CASES = {c["id"]: c for c in json.loads((HERE / "budget_cases.json").read_text("utf-8"))}
RECORDINGS = {
    r["case_id"]: r
    for r in json.loads((HERE / "budget_recordings.json").read_text("utf-8"))["recordings"]
}


def _recorded_budget(case_id: str) -> int | None:
    raw = RECORDINGS[case_id]["raw_arguments"].get("budget_vnd")
    if raw is None:
        return None
    try:
        return int(raw)
    except (TypeError, ValueError):
        return None


@pytest.mark.parametrize("case_id", sorted(CASES))
def test_gate_replay(case_id):
    case = CASES[case_id]
    turns = case["turns"]
    evidence = parse_budget_evidence(turns)
    budget, decision = resolve_budget(evidence, _recorded_budget(case_id), turns)
    expected = case["expected"]

    if expected["status"] == EXACT:
        assert evidence.status == EXACT
        assert budget == expected["value_vnd"], f"{case_id}: gate -> {budget} ({decision})"
        assert decision == "evidence"
    elif expected["status"] == INVALID:
        assert budget is None, f"{case_id}: invalid budget accepted: {budget}"
        assert decision == "invalid_blocked"
    elif expected["status"] == AMBIGUOUS:
        assert budget is None, f"{case_id}: ambiguous budget resolved by guessing: {budget}"
    elif expected["status"] == ABSENT:
        if budget is not None:
            # A model budget may survive only with literal support from
            # the user's own words (never an invented number).
            assert budget in literal_amounts(turns), (
                f"{case_id}: unsupported budget accepted: {budget}"
            )
            assert decision == "model_supported"


def test_replay_safety_metrics():
    unsupported = 0
    invalid_accepted = 0
    for case_id, case in CASES.items():
        turns = case["turns"]
        evidence = parse_budget_evidence(turns)
        budget, _decision = resolve_budget(evidence, _recorded_budget(case_id), turns)
        if budget is not None and evidence.status != EXACT:
            if budget not in literal_amounts(turns):
                unsupported += 1
        if case["expected"]["status"] == INVALID and budget is not None:
            invalid_accepted += 1
    assert unsupported == 0
    assert invalid_accepted == 0
