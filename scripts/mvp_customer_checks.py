"""Acceptance assertions for the real Web/Profile fixture."""

import hashlib
import hmac
import json
import time
import uuid
from datetime import datetime, timezone
from pathlib import Path
import yaml
from openapi_schema_validator import OAS30Validator
from jsonschema import FormatChecker

_spec = yaml.safe_load(
    (
        Path(__file__).resolve().parents[1]
        / "docs/03_api_specs/mvp-customer.openapi.yaml"
    ).read_text(encoding="utf-8")
)


def contract(name, payload):
    OAS30Validator(
        {"$ref": "#/components/schemas/" + name, "components": _spec["components"]},
        format_checker=FormatChecker(),
    ).validate(payload)


def profile_addresses(request, web, opener, profile, foreign_id=None):
    def call(path, body=None, method=None, version=None):
        headers = {"Origin": web}
        if version is not None:
            headers["If-Match"] = f'"{version}"'
        return request(
            web + "/api/customer/" + path,
            body,
            method=method,
            opener=opener,
            headers=headers,
        )

    update = {
        "full_name": "Synthetic Updated",
        "phone_number": "0905123456",
        "email": profile["email"],
        "version": profile["version"],
    }
    status, changed, _ = call("profile", update, "PUT", profile["version"])
    assert status == 200 and changed["version"] == profile["version"] + 1, (
        "profile update",
        status,
        changed,
    )
    assert call("profile", update, "PUT", profile["version"])[0] == 409
    status, locations, _ = call("locations")
    assert status == 200 and locations["provinces"], ("locations", status, locations)
    province = locations["provinces"][0]
    ward = province["wards"][0]
    body = {
        "recipient_name": "Synthetic Recipient",
        "phone_number": "0905123456",
        "street_address": "1 Synthetic Street",
        "ward_code": ward["wardCode"],
        "province_code": province["provinceCode"],
        "ward_name": "untrusted label",
        "province_name": "untrusted label",
        "label": "HOME",
    }
    status, first, _ = call("addresses", body, "POST")
    assert (
        status == 201
        and first["is_default"]
        and first["ward_name"] != "untrusted label"
    ), ("create first", status, first)
    status, second, _ = call(
        "addresses", dict(body, street_address="2 Synthetic Street"), "POST"
    )
    assert status == 201 and not second["is_default"], ("create second", status, second)
    status, listed, _ = call(
        "addresses/" + second["id"] + "/default",
        method="PUT",
        version=second["version"],
    )
    assert (
        status == 200
        and sum(a["is_default"] for a in listed) == 1
        and next(a for a in listed if a["id"] == second["id"])["is_default"]
    ), ("default", status, listed)
    status, current, _ = call("addresses/" + first["id"])
    assert status == 200
    status, changed, _ = call(
        "addresses/" + first["id"],
        dict(body, street_address="3 Synthetic Street"),
        "PUT",
        current["version"],
    )
    assert status == 200 and changed["version"] == current["version"] + 1, (
        "address update",
        status,
        changed,
    )
    assert call("addresses/" + first["id"], body, "PUT", current["version"])[0] == 409
    assert (
        call("addresses/" + first["id"], method="DELETE", version=current["version"])[0]
        == 409
    )
    if foreign_id:
        for method, payload in [("GET", None), ("PUT", body), ("DELETE", None)]:
            assert (
                call("addresses/" + foreign_id, payload, method, 1)[0] == 404
            ), "Cross-owner address access"
        assert (
            call("addresses/" + foreign_id + "/default", method="PUT", version=1)[0]
            == 404
        )
    status, listed, _ = call("addresses")
    selected = next(a for a in listed if a["id"] == second["id"])
    status, listed, _ = call(
        "addresses/" + second["id"], method="DELETE", version=selected["version"]
    )
    assert (
        status == 200
        and len(listed) == 1
        and listed[0]["id"] == first["id"]
        and listed[0]["is_default"]
    ), ("delete/default repair", status, listed)
    assert call("addresses/" + second["id"])[0] == 404
    print(
        "PASS: Profile update/CAS, canonical locations, address CRUD/default/delete and owner isolation",
        flush=True,
    )
    return first["id"]


def checkout_shipping(request, web, opener, control, shipping, token, webhook):
    def call(path, body=None, method=None, headers=None):
        return request(
            web + path,
            body,
            method=method,
            opener=opener,
            headers={"Origin": web, **(headers or {})},
        )

    def order_until(order_id, predicate):
        for _ in range(35):
            status, order, _ = call("/api/orders/" + order_id)
            assert status == 200, ("order read", status, order)
            contract("OrderOrder", order)
            if predicate(order):
                return order
            time.sleep(1)
        raise AssertionError(("Order projection did not converge", order))

    def place():
        status, cart, _ = call("/api/cart")
        assert status == 200
        status, cart, _ = call(
            "/api/cart/items",
            {"sku_code": "PEER", "quantity": 2, "revision": cart["revision"]},
            "POST",
        )
        assert status == 200, ("cart add", status, cart)
        status, addresses, _ = call("/api/customer/addresses")
        assert status == 200 and addresses
        saved = next(a for a in addresses if a["is_default"])
        assert (
            saved["id"].split("-")[2].startswith("7")
        ), "Profile must exercise its UUID v7 producer contract"
        body = {
            "cart_revision": cart["revision"],
            "payment_method": "VIETQR",
            "shipping_address_id": saved["id"],
        }
        status, quote, _ = call("/api/checkout/quote", body, "POST")
        assert status == 200, ("quote", status, quote)
        contract("OrderQuote", quote)
        accept = {
            "quote_id": quote["quote_id"],
            "cart_revision": quote["cart_revision"],
            "payment_method": "VIETQR",
        }
        key = uuid.uuid4().hex
        status, operation, _ = call(
            "/api/checkout", accept, "POST", {"Idempotency-Key": key}
        )
        assert status in [201, 202], ("accept", status, operation)
        contract("OrderCheckoutOperation", operation)
        operation_id = operation["operation_id"]
        status, replay, _ = call(
            "/api/checkout", accept, "POST", {"Idempotency-Key": key}
        )
        assert status in [201, 202] and replay["operation_id"] == operation_id, (
            "checkout replay",
            status,
            replay,
        )
        for _ in range(25):
            status, operation, _ = call("/api/checkout/operations/" + operation_id)
            assert status == 200, ("poll", status, operation)
            contract("OrderCheckoutOperation", operation)
            if operation["status"] == "SUCCEEDED":
                break
            assert operation["status"] not in ["FAILED", "MANUAL_REVIEW"], operation
            time.sleep(1)
        assert operation["status"] == "SUCCEEDED" and operation["order_id"]
        return operation["order_id"]

    status, cart, _ = call("/api/cart")
    for item in cart["items"]:
        status, cart, _ = call(
            "/api/cart/items/" + item["item_id"],
            method="DELETE",
            headers={"If-Match": f'"{cart["revision"]}"'},
        )
        assert status == 200 and not cart["items"], ("cart delete", status, cart)
    pending = place()
    current = order_until(pending, lambda o: o["status"] == "PENDING_PAYMENT")
    status, cancel, _ = call(
        "/api/orders/" + pending + "/cancel",
        {"active": True, "reason": "Synthetic cancellation"},
        "POST",
        {"Idempotency-Key": uuid.uuid4().hex, "If-Match": str(current["version"])},
    )
    assert status in [200, 202], ("cancel", status, cancel)
    contract("OrderCancellation", cancel)
    order_until(
        pending,
        lambda o: o["status"] == "CANCELLED_BY_USER"
        and o["stock_status"] == "RELEASED",
    )
    # Checkout cleanup has completed before placing a second order.
    for _ in range(20):
        status, cart, _ = call("/api/cart")
        if not cart["items"]:
            break
        time.sleep(0.25)
    assert not cart["items"], "Checkout cart cleanup did not finish"
    paid = place()
    order_until(paid, lambda o: o["status"] == "PENDING_PAYMENT")
    assert request(control + "/pay?order=" + paid, {}, method="POST")[0] == 202
    order_until(
        paid,
        lambda o: o["status"] == "PAID"
        and o["stock_status"] == "COMMITTED"
        and o["ready_authorized"],
    )
    status, shipment, _ = request(control + "/pack?order=" + paid, {}, method="POST")
    assert status == 200, ("shipping create", status, shipment)
    order_until(paid, lambda o: o.get("shipment_id") == shipment["shipment_id"])
    status, _, _ = request(
        shipping
        + "/api/v1/internal/shipments/"
        + shipment["shipment_id"]
        + "/handover",
        {},
        token,
        "POST",
    )
    assert status == 200, ("handover", status)
    order_until(paid, lambda o: o["status"] == "SHIPPED")
    payload = {
        "event_id": uuid.uuid4().hex,
        "tracking_code": shipment["tracking_code"],
        "status": "DELIVERED",
        "event_at": datetime.now(timezone.utc).isoformat(),
    }
    signature = (
        "sha256="
        + hmac.new(
            webhook.encode(), json.dumps(payload).encode(), hashlib.sha256
        ).hexdigest()
    )
    status, _, _ = request(
        shipping + "/api/v1/webhooks/logistics/fake",
        payload,
        headers={"X-Shipping-Signature": "wrong"},
    )
    assert status == 401
    status, _, _ = request(
        shipping + "/api/v1/webhooks/logistics/fake",
        payload,
        headers={"X-Shipping-Signature": signature},
    )
    assert status == 200, ("delivery", status)
    delivered = order_until(paid, lambda o: o["status"] == "DELIVERED")
    assert (
        request(
            shipping + "/api/v1/webhooks/logistics/fake",
            payload,
            headers={"X-Shipping-Signature": signature},
        )[0]
        == 200
    )
    status, tracking, _ = call("/api/orders/" + paid + "/tracking")
    assert (
        status == 200
        and tracking["shipment_id"] == shipment["shipment_id"]
        and tracking["order_status"] == "DELIVERED"
    ), ("tracking", status, tracking)
    status, history, _ = call("/api/orders?limit=20")
    assert status == 200 and {pending, paid}.issubset(
        {o["order_id"] for o in history["items"]}
    ), ("history", status, history)
    print(
        "PASS: Web quote/accept/replay/poll -> Order -> real TLS Inventory reserve/release/commit -> TEST payment -> TEST packing -> real Shipping/Kafka/revision -> delivered/history",
        flush=True,
    )
    return paid
