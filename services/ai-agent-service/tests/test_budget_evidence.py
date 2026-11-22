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
    EXACT,
    INVALID,
    is_affirmation,
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
    # Đợt 1.7: a literal match proves existence, not role — the gate
    # yields a candidate; only a customer confirmation may commit it.
    assert (budget, decision) == (None, "model_candidate")


def test_gate_absent_misattributed_money_is_only_a_candidate():
    # The audit's P0 cases: real money, wrong role (given away / past
    # purchase). Never a silent commit.
    for text, amount in (
        ("Tôi tặng bạn 200k rồi, giờ mua quà cho mẹ", 200_000),
        ("Tôi từng mua quà 500k, lần này muốn tiết kiệm hơn", 500_000),
    ):
        evidence = parse_budget_evidence([text])
        assert evidence.status == ABSENT
        budget, decision = resolve_budget(evidence, amount, [text])
        assert budget is None
        assert decision == "model_candidate"


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


# --- Đợt 1.7: role-based EXACT, candidate confirmation --------------------

def test_possessive_purchase_linked_is_exact():
    evidence = parse_budget_evidence(["Tôi có 200k, mua quà cho mẹ"])
    assert (evidence.status, evidence.value_vnd) == (EXACT, 200_000)


def test_possessive_bare_possession_is_not_evidence():
    for text in (
        "Tôi có 200k trong ví",
        "Tôi có 200k tiền mặt, còn tiền trong tài khoản nữa",
        "Tôi có 200k, nhưng chưa biết có nên tiêu không",
    ):
        assert parse_budget_evidence([text]).status == ABSENT, text


def test_possessive_direct_answer_to_budget_question_is_exact():
    turns = ["Anh/chị cho em biết ngân sách nhé", "Tôi có 200k"]
    assert parse_budget_evidence(turns).status == ABSENT  # no question pending
    evidence = parse_budget_evidence(turns, budget_requested=True)
    assert (evidence.status, evidence.value_vnd) == (EXACT, 200_000)


def test_role_predicates_are_budget_evidence():
    cases = [
        ("Tôi định mua hộp quà 200k, nhưng có thể chi đến 300k nếu cần", 300_000),
        ("Mình dành 300k để mua quà cho bố", 300_000),
        ("Em chỉ mang theo 500k thôi", 500_000),
    ]
    for text, value in cases:
        evidence = parse_budget_evidence([text])
        assert (evidence.status, evidence.value_vnd) == (EXACT, value), text


def test_danh_tang_is_giving_not_budget():
    assert parse_budget_evidence(["Tôi dành tặng bạn 200k"]).status == ABSENT


@pytest.mark.parametrize(
    "first,second",
    [(500, 200), (200, 500), (100, 300), (300, 100), (250, 250)],
)
def test_correction_ordering_property(first, second):
    turns = [f"Ngân sách {first}k", f"À thôi, ngân sách {second}k thôi"]
    evidence = parse_budget_evidence(turns)
    assert (evidence.status, evidence.value_vnd) == (EXACT, second * 1000)


@pytest.mark.parametrize(
    "text",
    ["Đúng", "đúng rồi", "Ừ", "ừm đúng", "Ok", "Phải ạ", "Vâng, đúng rồi"],
)
def test_affirmation_accepts_plain_yes(text):
    assert is_affirmation(text)


@pytest.mark.parametrize(
    "text",
    [
        "Đúng, nhưng tối đa 150k thôi",   # contains a new amount -> evidence path
        "Đúng, nhưng 200k là tiền tặng thêm",
        "Không phải",                      # negation
        "Đúng rồi, giao hàng bao nhiêu?",  # a question, not a bare yes
        "Ừ, mà sản phẩm có giao trong ngày không?",
        "Chưa chắc",
        "",
    ],
)
def test_affirmation_rejects_non_plain(text):
    assert not is_affirmation(text)
