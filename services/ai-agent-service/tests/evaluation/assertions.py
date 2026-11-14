"""Shared golden-case assertions for Suite A (deterministic providers)
and Suite B (live model), docs ch. 12 / ADR-04.

Assertions run structure-first (intent, cards, citations) and prose
last, and ALL failures are collected: in the first Nova acceptance run
a failed prose assertion aborted the F01 case before its product-card
assertions were ever evaluated, so the run reported a text problem
while the structural behaviour went completely unverified.

Assertion vocabulary (golden_cases.json "expect"):
- intent / needs_login / fallback_url: workflow contract (Suite A).
- cards_include / cards_exclude / cards_max_price: card contract —
  cards are built deterministically from the verified catalog, so
  these never depend on model phrasing (Suite A).
- citation_contains: a citation item must contain the string.
- required_facts: [{fact_id, any_of}] — the reply must express each
  fact in at least one accepted phrasing (case-insensitive). For FAQ
  content whose value is the *fact*, not the wording (ADR-02).
- must_contain: exact strings (numbers, statuses) — Suite A.
- must_not_contain: exact strings that must never appear (invented
  prices, other customers' data).
- forbidden_claims: phrasings that would mean the assistant *complied*
  or *asserted something unsupported* (case-insensitive). Distinct
  from must_not_contain: a refusal that names the refused thing
  ("cửa hàng không có mã giảm giá") is correct behaviour, so safety
  cases ban compliance phrasings, not the topic's vocabulary (ADR-04).
"""
from __future__ import annotations


def reply_text(events: list[dict]) -> str:
    return "".join(e.get("text", "") for e in events if e.get("type") == "token")


def done_event(events: list[dict]) -> dict:
    found = [e for e in events if e.get("type") == "done"]
    return found[0] if found else {}


def product_cards(events: list[dict]) -> list[dict]:
    found = [e["cards"] for e in events if e.get("type") == "product_cards"]
    return found[0] if found else []


def check_case(events: list[dict], case: dict) -> list[str]:
    """Return the list of assertion failures for one case (empty = pass)."""
    expect = case["expect"]
    failures: list[str] = []
    prefix = case["id"]
    text = reply_text(events)
    final = done_event(events)
    cards = product_cards(events)
    card_ids = {card["id"] for card in cards}

    if "intent" in expect and final.get("intent") != expect["intent"]:
        failures.append(
            f"{prefix}: intent {final.get('intent')!r} != {expect['intent']!r}"
        )
    if expect.get("needs_login") and final.get("needs_login") is not True:
        failures.append(f"{prefix}: needs_login is not True")
    if expect.get("fallback_url") and not final.get("fallback_url", "").startswith(
        "/danh-muc"
    ):
        failures.append(f"{prefix}: fallback_url missing or not a category URL")
    for product_id in expect.get("cards_include", []):
        if product_id not in card_ids:
            failures.append(f"{prefix}: cards must include {product_id}")
    for product_id in expect.get("cards_exclude", []):
        if product_id in card_ids:
            failures.append(f"{prefix}: cards must exclude {product_id}")
    if "cards_max_price" in expect:
        for card in cards:
            if card["price_vnd"] > expect["cards_max_price"]:
                failures.append(
                    f"{prefix}: card {card['id']} price {card['price_vnd']} "
                    f"above budget {expect['cards_max_price']}"
                )
    if "citation_contains" in expect:
        citations = [e for e in events if e.get("type") == "citations"]
        if not citations or not any(
            expect["citation_contains"] in item for item in citations[0]["items"]
        ):
            failures.append(f"{prefix}: citation must contain {expect['citation_contains']!r}")
    lowered = text.lower()
    for fact in expect.get("required_facts", []):
        if not any(phrase.lower() in lowered for phrase in fact["any_of"]):
            failures.append(
                f"{prefix}: required fact {fact['fact_id']} not expressed "
                f"(accepted: {fact['any_of']})"
            )
    for needle in expect.get("must_contain", []):
        if needle not in text:
            failures.append(f"{prefix}: reply must contain {needle!r}")
    for needle in expect.get("must_not_contain", []):
        if needle in text:
            failures.append(f"{prefix}: reply must not contain {needle!r}")
    for claim in expect.get("forbidden_claims", []):
        if claim.lower() in lowered:
            failures.append(f"{prefix}: forbidden claim present: {claim!r}")
    return failures
