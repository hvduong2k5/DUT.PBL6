import json

import pytest
from fastapi.testclient import TestClient

from ai_agent_service.config import Settings
from ai_agent_service.main import create_app
from ai_agent_service.platform.auth import Principal


def make_app():
    app = create_app(Settings())

    def fake_auth(header):
        if header and header.startswith("Bearer "):
            return Principal(customer_id=header[7:].strip(), is_guest=False)
        return Principal(customer_id=None, is_guest=True)

    app.state.authenticator = fake_auth
    return app


@pytest.fixture()
def client():
    with TestClient(make_app()) as test_client:
        yield test_client


def create_conversation(client, headers=None) -> str:
    response = client.post("/api/v1/ai/shopping/conversations", headers=headers or {})
    assert response.status_code == 201
    return response.json()["conversation_id"]


def chat(client, conversation_id, text, headers=None) -> list[dict]:
    with client.stream(
        "POST",
        f"/api/v1/ai/shopping/conversations/{conversation_id}/messages",
        json={"content": text},
        headers=headers or {},
    ) as response:
        assert response.status_code == 200
        events = []
        for line in response.iter_lines():
            if line.startswith("data: "):
                events.append(json.loads(line[6:]))
    return events


def reply_text(events) -> str:
    return "".join(e["text"] for e in events if e["type"] == "token")


def done(events) -> dict:
    return next(e for e in events if e["type"] == "done")


def cards(events) -> list[dict]:
    found = [e["cards"] for e in events if e["type"] == "product_cards"]
    return found[0] if found else []


def test_guest_clarify_then_recommendations(client):
    cid = create_conversation(client)
    first = chat(client, cid, "Mình muốn mua quà Tết")
    assert done(first)["intent"] == "clarification"

    second = chat(client, cid, "Ngân sách 200k, tặng bố mẹ, thích ngọt")
    assert done(second)["intent"] == "shopping"
    product_cards = cards(second)
    ids = {card["id"] for card in product_cards}
    assert "p-deo" in ids  # in stock, within budget
    assert "p-gion" not in ids  # out of stock (mandatory stock check)
    assert "p-old" not in ids  # discontinued never surfaces
    assert all(card["price_vnd"] <= 200_000 for card in product_cards)
    assert "49.000đ" in reply_text(second)


def test_order_status_owner(client):
    headers = {"Authorization": "Bearer cust-1"}
    cid = create_conversation(client, headers)
    events = chat(client, cid, "Đơn DH1001 của mình tới đâu rồi?", headers)
    assert done(events)["intent"] == "order_status"
    assert "SHIPPED" in reply_text(events)


def test_order_status_other_customer_hidden(client):
    headers = {"Authorization": "Bearer cust-1"}
    cid = create_conversation(client, headers)
    events = chat(client, cid, "Xem giúp đơn DH1002", headers)
    text = reply_text(events)
    assert "DELIVERED" not in text
    assert "không tìm thấy" in text


def test_order_status_guest_needs_login(client):
    cid = create_conversation(client)
    events = chat(client, cid, "Đơn DH1001 giao chưa?")
    assert done(events).get("needs_login") is True
    assert "đăng nhập" in reply_text(events)


def test_shipping_fee_grounded(client):
    cid = create_conversation(client)
    events = chat(client, cid, "Phí ship về Huế bao nhiêu?")
    assert done(events)["intent"] == "shipping_fee"
    assert "22.000đ" in reply_text(events)


def test_faq_shelf_life(client):
    cid = create_conversation(client)
    events = chat(client, cid, "Mè xửng để được bao lâu, bảo quản sao?")
    assert done(events)["intent"] == "faq"
    assert "6 tháng" in reply_text(events)
    citations = [e for e in events if e["type"] == "citations"]
    assert citations and any("Hạn sử dụng" in item for item in citations[0]["items"])


def test_conversation_ownership_isolation(client):
    headers = {"Authorization": "Bearer cust-1"}
    cid = create_conversation(client, headers)
    other = client.get(
        f"/api/v1/ai/shopping/conversations/{cid}",
        headers={"Authorization": "Bearer cust-2"},
    )
    assert other.status_code == 404


def test_recommendations_route_product_detail(client):
    response = client.get(
        "/api/v1/ai/recommendations",
        params={"placement": "product_detail", "product_id": "p-deo"},
    )
    assert response.status_code == 200
    ids = {item["id"] for item in response.json()["items"]}
    assert ids <= {"p-gion", "p-gung"}
    assert ids  # non-empty from the Analytics fixture contract
