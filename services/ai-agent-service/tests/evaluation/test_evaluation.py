"""Golden-set evaluation: ~30 Vietnamese cases in 4 groups
(Functional / Retrieval / Grounding / Safety), run against the full API
with the deterministic FakeProvider and fixture backends.

This is the phase-1 quality gate. When Bedrock IAM credentials arrive, the
same set must be re-run against Nova Pro as the production acceptance.
"""
import json
import pathlib
import sys

import pytest
from fastapi.testclient import TestClient

sys.path.insert(0, str(pathlib.Path(__file__).parent.parent))

from test_shopping_flow import chat, create_conversation, done, reply_text  # noqa: E402

from ai_agent_service.config import Settings  # noqa: E402
from ai_agent_service.main import create_app  # noqa: E402
from ai_agent_service.platform.auth import Principal  # noqa: E402

CASES = json.loads((pathlib.Path(__file__).parent / "golden_cases.json").read_text("utf-8"))


@pytest.fixture(scope="module")
def client():
    app = create_app(Settings())

    def fake_auth(header):
        if header and header.startswith("Bearer "):
            return Principal(customer_id=header[7:].strip(), is_guest=False)
        return Principal(customer_id=None, is_guest=True)

    app.state.authenticator = fake_auth
    with TestClient(app) as test_client:
        yield test_client


def _cards(events) -> list[dict]:
    found = [e["cards"] for e in events if e["type"] == "product_cards"]
    return found[0] if found else []


@pytest.mark.parametrize("case", CASES, ids=[c["id"] for c in CASES])
def test_golden_case(client, case):
    headers = (
        {"Authorization": f"Bearer {case['auth']}"} if case["auth"] != "guest" else {}
    )
    conversation_id = create_conversation(client, headers)
    events: list[dict] = []
    for turn in case["turns"]:
        events = chat(client, conversation_id, turn, headers)

    expect = case["expect"]
    text = reply_text(events)
    final = done(events)
    product_cards = _cards(events)
    card_ids = {card["id"] for card in product_cards}

    if "intent" in expect:
        assert final["intent"] == expect["intent"], f"{case['id']}: intent"
    for needle in expect.get("must_contain", []):
        assert needle in text, f"{case['id']}: reply must contain {needle!r}"
    for needle in expect.get("must_not_contain", []):
        assert needle not in text, f"{case['id']}: reply must not contain {needle!r}"
    for product_id in expect.get("cards_include", []):
        assert product_id in card_ids, f"{case['id']}: cards must include {product_id}"
    for product_id in expect.get("cards_exclude", []):
        assert product_id not in card_ids, f"{case['id']}: cards must exclude {product_id}"
    if "cards_max_price" in expect:
        assert all(
            card["price_vnd"] <= expect["cards_max_price"] for card in product_cards
        ), f"{case['id']}: card above budget"
    if expect.get("fallback_url"):
        assert final.get("fallback_url", "").startswith("/danh-muc"), case["id"]
    if expect.get("needs_login"):
        assert final.get("needs_login") is True, case["id"]
    if "citation_contains" in expect:
        citations = [e for e in events if e["type"] == "citations"]
        assert citations and any(
            expect["citation_contains"] in item for item in citations[0]["items"]
        ), f"{case['id']}: citation"
