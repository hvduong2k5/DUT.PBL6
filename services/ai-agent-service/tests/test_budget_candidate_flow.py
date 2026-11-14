"""Budget candidate confirmation flow (docs ch. 14, dot 1.7).

End-to-end over the HTTP API with the deterministic Fake provider:
under ABSENT, a model budget that matches a literal amount becomes a
candidate the bot names and asks about; only the customer's direct
confirmation commits it.
"""
import json
import uuid

import pytest
from fastapi.testclient import TestClient

from ai_agent_service.config import Settings
from ai_agent_service.main import create_app

_GUEST_TOKENS: dict[str, str] = {}


@pytest.fixture(scope="module")
def client():
    with TestClient(create_app(Settings(fixtures_mode=True))) as test_client:
        yield test_client


def _conversation(client) -> str:
    resp = client.post(
        "/api/v1/ai/shopping/conversations",
        headers={"Idempotency-Key": str(uuid.uuid4())},
    )
    assert resp.status_code == 201
    payload = resp.json()
    conversation_id = payload["conversation_id"]
    _GUEST_TOKENS[conversation_id] = payload["guest_token"]
    return conversation_id


def _chat(client, conv_id: str, text: str) -> dict:
    with client.stream(
        "POST",
        f"/api/v1/ai/shopping/conversations/{conv_id}/messages",
        json={"content": text},
        headers={"X-Guest-Conversation-Token": _GUEST_TOKENS[conv_id]},
    ) as resp:
        assert resp.status_code == 200
        body = resp.read().decode("utf-8")
    events = []
    for block in body.split("\n\n"):
        for line in block.splitlines():
            if line.startswith("data: "):
                events.append(json.loads(line[6:]))
    text_out = "".join(e.get("text", "") for e in events if e["type"] == "token")
    done = next(e for e in events if e["type"] == "done")
    cards = next((e["cards"] for e in events if e["type"] == "product_cards"), [])
    return {"text": text_out, "intent": done["intent"], "cards": cards}


def test_candidate_is_named_and_waits_for_confirmation(client):
    conv = _conversation(client)
    first = _chat(client, conv, "Cho mẹ 200k nhé, tính giúp em")
    assert first["intent"] == "clarification"
    assert "200.000" in first["text"]
    assert "ngân sách" in first["text"]


def test_confirmation_commits_the_candidate(client):
    conv = _conversation(client)
    _chat(client, conv, "Cho mẹ 200k nhé, tính giúp em")
    second = _chat(client, conv, "Đúng rồi")
    assert second["intent"] == "shopping"
    # The committed budget constrains the follow-up search.
    third = _chat(client, conv, "Cho em xem mè xửng dẻo")
    assert third["cards"], "expected product cards after confirmation"
    assert all(card["price_vnd"] <= 200_000 for card in third["cards"])


def test_new_budget_supersedes_the_candidate(client):
    conv = _conversation(client)
    _chat(client, conv, "Cho mẹ 200k nhé, tính giúp em")
    second = _chat(client, conv, "Không, ngân sách 100k thôi")
    assert second["intent"] == "shopping"
    third = _chat(client, conv, "Cho em xem mè xửng dẻo")
    assert third["cards"]
    assert all(card["price_vnd"] <= 100_000 for card in third["cards"])


def test_candidate_expires_across_a_detour(client):
    conv = _conversation(client)
    _chat(client, conv, "Cho mẹ 200k nhé, tính giúp em")
    detour = _chat(client, conv, "Phí ship về Huế bao nhiêu?")
    assert detour["intent"] == "shipping_fee"
    late = _chat(client, conv, "Đúng rồi")
    # The stale "yes" must not commit the old candidate: the bot is
    # back to asking, not shopping with a 200k budget.
    assert late["intent"] == "clarification"
    assert not late["cards"]
