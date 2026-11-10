"""Budget evidence gate (docs ch. 14, ADR-01).

The LLM extractor once turned a user's "Ngân sách của tôi là 5đ" into
budget_vnd=5000000 — a value inside the valid range, so the old range
check passed it and the workflow shopped with an invented budget
(golden case S06). Range validation checks the model's number; it
cannot check the number's *provenance*. This module is the provenance
gate: a small deterministic, domain-scoped parser reads the user's own
words and decides what monetary evidence actually exists.

Evidence states (per conversation, user messages in order, the latest
budget statement wins):

- EXACT: an explicit amount the parser can tie to the *budget role*
  ("ngân sách 200k", "tối đa 500.000đ", "2 triệu rưỡi", "có thể chi
  300k", "Tôi có 200k, mua quà cho mẹ"). Seeing an amount is never
  enough — its role must be evidenced. The parser is authoritative.
- INVALID: an explicit budget amount outside the accepted range
  ("5đ", "5 tỷ"). The LLM is forbidden from overriding it — an invalid
  statement is evidence, not absence of evidence.
- AMBIGUOUS: a budget statement without one determinate value
  ("2-3 trăm nghìn", "vài trăm", "ngân sách 500" without a unit).
  Never resolved by guessing: the workflow asks.
- ABSENT: no budget-role amount anywhere. A number the LLM returns
  may become a *candidate* when it matches an amount the user
  literally wrote in a non-price, non-suppressed position — but a
  candidate is never committed by the gate (đợt 1.7 audit: existence
  of a money mention is not evidence of its role; "tặng bạn 200k" is
  money given away, not a shopping budget). The workflow asks the
  customer to confirm the candidate; only that confirmation commits
  it. An amount the user never wrote can not even become a candidate
  (e.g. a budget invented from "ngân sách sinh viên").

Cap phrasings ("dưới 200k", "không quá 200k", "tối đa 200k") are EXACT
with the cap as value: the workflow treats budget as a maximum price
anyway, so cap semantics and budget semantics coincide operationally.

Known limits: word numerals ("năm trăm nghìn") are not parsed; the
product-word list used to spot price positions is shop-domain specific.
"""
from __future__ import annotations

import re
from dataclasses import dataclass

BUDGET_MIN_VND = 10_000
BUDGET_MAX_VND = 1_000_000_000

EXACT = "exact"
INVALID = "invalid"
AMBIGUOUS = "ambiguous"
ABSENT = "absent"

_UNITS = {
    "tỷ": 1_000_000_000,
    "triệu": 1_000_000,
    "tr": 1_000_000,
    "nghìn": 1_000,
    "ngàn": 1_000,
    "k": 1_000,
    "trăm": 100,
    "đồng": 1,
    "vnđ": 1,
    "vnd": 1,
    "đ": 1,
    "₫": 1,
}

# One money expression: optional "nửa" (half), a number, an optional
# unit, an optional "rưỡi" (+half a unit). Numbers are either grouped
# ("500.000"), decimal ("1.5" — only meaningful with a multiplier unit)
# or plain digits.
_NUM = r"\d{1,3}(?:[.,]\d{3})+|\d+(?:[.,]\d+)?"
_UNIT = r"tỷ|triệu|tr|nghìn|ngàn|trăm|k|đồng|vnđ|vnd|đ|₫"
_EXPR = re.compile(
    rf"(?P<half>nửa\s+)?(?P<num>{_NUM})(?:\s*(?P<unit>{_UNIT}))?(?:\s*(?P<ruoi>rưỡi))?",
    re.IGNORECASE,
)
_HALF_ONLY = re.compile(rf"nửa\s+(?P<unit>{_UNIT})\b", re.IGNORECASE)
_RANGE = re.compile(
    rf"(?P<n1>{_NUM})\s*(?:[-–~]|đến)\s*(?P<n2>{_NUM})\s*"
    rf"(?P<unit>trăm\s*(?:nghìn|ngàn)|{_UNIT})",
    re.IGNORECASE,
)
_VAGUE = re.compile(rf"(?:vài|mấy)\s+(?:{_UNIT})", re.IGNORECASE)

# Budget-position anchors looked for in the ~32 chars before an amount.
_ANCHORS = (
    "ngân sách", "tối đa", "không quá", "không vượt quá", "khoảng",
    "tầm", "dưới", "chỉ có", "trong vòng", "budget",
    "có thể chi", "chỉ mang theo", "mang theo",
)
# Possession-only markers right after an amount veto the possessive
# reading ("Tôi có 200k trong ví" states funds, not a budget).
_POSSESSION_VETOES = ("ví", "tài khoản", "tiền mặt", "chưa biết", "có nên")
# Statements that must NOT become budget evidence. Checked before the
# anchors ("không quá" is an anchor, not a negation — it is excluded
# from this list deliberately). "nhầm" is NOT a suppressor: "À nhầm,
# ngân sách 500k" is how a correction STARTS (case B28).
_SUPPRESSORS = (
    "không phải", "không tới", "không đến", "chưa tới", "nếu",
    "giả sử",
)
# Words marking a price position rather than a budget position.
_PRICE_MARKERS = ("giá",)
_PRODUCT_WORDS = ("hộp", "gói", "sản phẩm", "món", "combo", "set", "chai", "túi")


@dataclass(frozen=True)
class _Amount:
    value: int | None  # None when the expression has no determinate value
    low: int | None
    high: int | None
    start: int
    end: int
    text: str
    has_unit: bool


def _number_value(raw: str, unit: str | None) -> float:
    """Numeric value of the digit part, before the unit multiplier.
    Grouped separators ("500.000") read as vi-VN thousand grouping; a
    short fraction ("1.5") reads as a decimal."""
    if re.fullmatch(r"\d{1,3}(?:[.,]\d{3})+", raw):
        return float(int(raw.replace(".", "").replace(",", "")))
    return float(raw.replace(",", "."))


def _unit_multiplier(unit: str) -> int:
    unit = " ".join(unit.lower().split())
    if unit.startswith("trăm ") or unit in ("trăm nghìn", "trăm ngàn"):
        return 100_000
    return _UNITS[unit]


def _expression_value(match: re.Match) -> _Amount:
    num_raw = match.group("num")
    unit = (match.group("unit") or "").lower() or None
    multiplier = _UNITS.get(unit, 1) if unit else 1
    base = _number_value(num_raw, unit)
    value = int(base * multiplier)
    if match.group("half"):
        # "nửa" before an explicit number is rare; read it as half of
        # the whole expression.
        value //= 2
    if match.group("ruoi") and unit:
        value += multiplier // 2
    has_unit = unit is not None or match.group("ruoi") is not None
    return _Amount(
        value=value, low=None, high=None,
        start=match.start(), end=match.end(), text=match.group(0),
        has_unit=has_unit,
    )


def _amounts_in(text: str) -> list[_Amount]:
    """All determinate money expressions in one message (any position)."""
    found: list[_Amount] = []
    low_text = text.lower()
    range_spans: list[tuple[int, int]] = []
    for match in _RANGE.finditer(low_text):
        unit = match.group("unit").lower()
        multiplier = _unit_multiplier(unit)
        lo = int(_number_value(match.group("n1"), unit) * multiplier)
        hi = int(_number_value(match.group("n2"), unit) * multiplier)
        found.append(_Amount(
            value=None, low=min(lo, hi), high=max(lo, hi),
            start=match.start(), end=match.end(), text=match.group(0),
            has_unit=True,
        ))
        range_spans.append((match.start(), match.end()))
    for match in _HALF_ONLY.finditer(low_text):
        unit = match.group("unit").lower()
        if any(s <= match.start() < e for s, e in range_spans):
            continue
        found.append(_Amount(
            value=_UNITS[unit] // 2, low=None, high=None,
            start=match.start(), end=match.end(), text=match.group(0),
            has_unit=True,
        ))
        range_spans.append((match.start(), match.end()))
    for match in _EXPR.finditer(low_text):
        if any(s <= match.start() < e for s, e in range_spans):
            continue
        # Digits glued to a letter belong to a code or word
        # ("DH1001", "200kẹo"), not to a money expression.
        if match.start() > 0 and low_text[match.start() - 1].isalnum():
            continue
        found.append(_expression_value(match))
    return sorted(found, key=lambda a: a.start)


def _context_before(text: str, start: int, width: int = 32) -> str:
    window = text[max(0, start - width):start].lower()
    # Anchors, suppressors and price markers only count within the
    # same sentence: "Tôi có ngân sách 500k. Hộp quà đó giá 200k" must
    # not lend its budget anchor to the price (case B21).
    cut = max(window.rfind(mark) for mark in (". ", "? ", "! ", "; ", "\n"))
    if cut >= 0:
        window = window[cut + 2:]
    return window


def _is_suppressed(text: str, start: int) -> bool:
    window = _context_before(text, start, 24)
    return any(marker in window for marker in _SUPPRESSORS)


def _is_budget_position(text: str, amount: _Amount, budget_requested: bool = False) -> bool:
    window = _context_before(text, amount.start)
    if any(anchor in window for anchor in _ANCHORS):
        return True
    # "dành X để mua" states an allocation for buying; "dành tặng" is
    # money given away, the opposite role.
    if "dành" in window and "dành tặng" not in window:
        return True
    # Possessive: the amount directly follows "có". That is budget
    # evidence only when the amount is tied to the purchase — the
    # message links it to buying/gifting, or it directly answers a
    # budget question the workflow just asked. Bare possession
    # ("trong ví", "tiền mặt") is a candidate at most (đợt 1.7).
    before = text[:amount.start].rstrip().lower()
    if not before.endswith("có"):
        return False
    after = text[amount.end:amount.end + 24].lower()
    if any(veto in after for veto in _POSSESSION_VETOES):
        return False
    if budget_requested:
        return True
    return any(marker in text.lower() for marker in ("mua", "quà"))


def _is_price_position(text: str, amount: _Amount) -> bool:
    window = _context_before(text, amount.start, 24)
    if any(marker in window for marker in _PRICE_MARKERS):
        return True
    return any(word in window for word in _PRODUCT_WORDS)


@dataclass(frozen=True)
class BudgetEvidence:
    status: str
    value_vnd: int | None = None
    low_vnd: int | None = None
    high_vnd: int | None = None
    message_index: int | None = None
    source_text: str | None = None


def _message_outcome(
    text: str, index: int, budget_requested: bool = False
) -> BudgetEvidence | None:
    """Budget outcome of a single message, or None when the message
    carries no budget-position statement."""
    low_text = text.lower()
    outcome: BudgetEvidence | None = None
    for amount in _amounts_in(text):
        if _is_suppressed(text, amount.start):
            continue
        if not _is_budget_position(low_text, amount, budget_requested):
            continue
        span = text[amount.start:amount.end]
        if amount.value is None:
            outcome = BudgetEvidence(
                status=AMBIGUOUS, low_vnd=amount.low, high_vnd=amount.high,
                message_index=index, source_text=span,
            )
            continue
        if not amount.has_unit and amount.value < BUDGET_MIN_VND:
            # "ngân sách 500" — a number with no unit is not yet money.
            outcome = BudgetEvidence(
                status=AMBIGUOUS, message_index=index, source_text=span,
            )
            continue
        if not (BUDGET_MIN_VND <= amount.value <= BUDGET_MAX_VND):
            outcome = BudgetEvidence(
                status=INVALID, value_vnd=amount.value,
                message_index=index, source_text=span,
            )
            continue
        outcome = BudgetEvidence(
            status=EXACT, value_vnd=amount.value,
            message_index=index, source_text=span,
        )
    if outcome is None:
        # A vague quantity in budget position ("vài trăm nghìn").
        for match in _VAGUE.finditer(low_text):
            probe = _Amount(
                value=None, low=None, high=None,
                start=match.start(), end=match.end(), text=match.group(0),
                has_unit=True,
            )
            if _is_suppressed(low_text, probe.start):
                continue
            if _is_budget_position(low_text, probe, budget_requested):
                outcome = BudgetEvidence(
                    status=AMBIGUOUS, message_index=index,
                    source_text=text[match.start():match.end()],
                )
    return outcome


def parse_budget_evidence(
    messages: list[str], budget_requested: bool = False
) -> BudgetEvidence:
    """Evidence over the whole conversation: the latest budget
    statement wins (a later correction supersedes an earlier amount,
    including a later invalid one).

    `budget_requested` is true when the workflow has already asked for
    the budget (a clarification is in flight): a possessive amount in
    the latest message then directly answers that question and counts
    as budget evidence (đợt 1.7).
    """
    result = BudgetEvidence(status=ABSENT)
    for index, message in enumerate(messages):
        outcome = _message_outcome(
            message, index, budget_requested and index == len(messages) - 1
        )
        if outcome is not None:
            result = outcome
    return result


def literal_amounts(messages: list[str]) -> set[int]:
    """Every determinate amount the user literally wrote, any position."""
    values: set[int] = set()
    for message in messages:
        for amount in _amounts_in(message):
            if amount.value is not None:
                values.add(amount.value)
    return values


_AFFIRMATIONS = {
    "đúng", "đúng rồi", "đúng vậy", "ừ", "ừ đúng", "ừm", "ừm đúng",
    "ok", "oke", "ok đúng", "phải", "vâng", "chuẩn", "đồng ý",
}
_AFFIRM_HEADS = {"đúng", "ừ", "ừm", "ok", "oke", "phải", "vâng", "chuẩn"}
_AFFIRM_BLOCKERS = {"không", "chưa", "đừng", "nhưng"}


def is_affirmation(text: str) -> bool:
    """True only for a short, unambiguous yes with no new content.

    Used to confirm a budget candidate (đợt 1.7): the reply must carry
    no digits (a new amount is evidence, handled by the parser), no
    negation or concession ("đúng nhưng..." is not a confirmation),
    and at most a short affirmation phrase.
    """
    cleaned = re.sub(r"[.!?…~]", " ", text.lower()).replace(",", " ")
    cleaned = " ".join(cleaned.split())
    if not cleaned or any(ch.isdigit() for ch in cleaned):
        return False
    tokens = cleaned.split()
    if any(token in _AFFIRM_BLOCKERS for token in tokens):
        return False
    if cleaned in _AFFIRMATIONS:
        return True
    return tokens[0] in _AFFIRM_HEADS and len(tokens) <= 4


def _supported_literals(messages: list[str]) -> set[int]:
    """Literal amounts that may support a model-extracted budget when
    no budget-position evidence exists: any position except a price
    position (a product's price is not a budget) and except suppressed
    statements (a hypothetical "giả sử ngân sách 500k" is not a budget
    either, case B35)."""
    values: set[int] = set()
    for message in messages:
        low = message.lower()
        for amount in _amounts_in(message):
            if amount.value is None:
                continue
            if _is_suppressed(low, amount.start):
                continue
            if _is_price_position(low, amount):
                continue
            values.add(amount.value)
    return values


def resolve_budget(
    evidence: BudgetEvidence,
    model_budget: int | None,
    messages: list[str],
) -> tuple[int | None, str]:
    """The gate. Returns (budget_vnd, decision) — decision names the
    provenance of the outcome for metrics and audit:
    evidence | invalid_blocked | ambiguous_clarify |
    model_candidate | model_unsupported | no_budget.

    "model_candidate" (đợt 1.7) commits nothing: under ABSENT, a model
    budget that matches a literal amount the user wrote becomes a
    candidate the workflow must confirm with the customer — existence
    of a money mention is not evidence of its budget role.
    """
    if evidence.status == EXACT:
        return evidence.value_vnd, "evidence"
    if evidence.status == INVALID:
        return None, "invalid_blocked"
    if evidence.status == AMBIGUOUS:
        return None, "ambiguous_clarify"
    if model_budget is None:
        return None, "no_budget"
    if not (BUDGET_MIN_VND <= model_budget <= BUDGET_MAX_VND):
        return None, "model_unsupported"
    if model_budget in _supported_literals(messages):
        return None, "model_candidate"
    return None, "model_unsupported"
