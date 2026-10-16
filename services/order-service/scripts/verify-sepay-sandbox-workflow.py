"""Exercise replay and manual cancellation/refund against a real sandbox receipt.

Only the explicitly selected sandbox test order is changed. Bank refund evidence
is synthetic: this verifies the local workflow, never an actual bank transfer.
Credentials, tokens and provider payloads are never written to the report.
"""
import argparse
import json
import subprocess
import time
import uuid
from datetime import datetime, timezone
from pathlib import Path
from urllib.error import HTTPError
from urllib.request import Request, urlopen

ROOT = Path(__file__).resolve().parents[3]
BASE = "http://127.0.0.1:18004"
CHECKS = []


def sql(query):
    return subprocess.check_output(["docker", "compose", "-f", str(ROOT / "services/order-service/docker-compose.test.yml"), "exec", "-T", "db", "psql", "-U", "order_test", "-d", "order_mvp_sandbox", "-At", "-v", "ON_ERROR_STOP=1", "-c", query], cwd=ROOT, text=True).strip()


def call(url, method="GET", body=None, token=None, headers=None, expected=200):
    h = {"Content-Type": "application/json", **(headers or {})}
    if token:
        h["Authorization"] = "Bearer " + token
    req = Request(url, method=method, data=None if body is None else json.dumps(body).encode(), headers=h)
    try:
        with urlopen(req, timeout=25) as r:
            status, result = r.status, json.load(r)
    except HTTPError as error:
        status = error.code
        try:
            result = json.load(error)
        except (ValueError, UnicodeError):
            result = {}
    if status != expected:
        raise AssertionError(f"Expected HTTP {expected}, got {status}")
    return result


def account(role, user=None):
    return call("http://127.0.0.1:18104/test/token", "POST", {"role": role, "user_id": user or str(uuid.uuid4())}, headers={"X-Internal-Token": "order-sandbox-internal-token-2026"})["token"]


def passed(name, mode, **details):
    CHECKS.append({"scenario": name, "mode": mode, "status": "PASS", **details})
    print(name + ": PASS", flush=True)


def main(order_id):
    order_id = str(uuid.UUID(order_id))
    config = dict(line.split("=", 1) for line in (ROOT / "services/order-service/.env.local").read_text(encoding="utf-8").splitlines() if "=" in line and not line.startswith("#"))
    assert config["SEPAY_MERCHANT_ID"].startswith("SP-TEST-"), "Sandbox merchant required"
    principal = str(uuid.UUID(sql("SELECT principal_id FROM orders WHERE id='" + order_id + "'")))
    owner, admin, verifier = account("CUSTOMER", principal), account("ADMIN"), account("ADMIN")
    order_path = BASE + "/api/v1/orders/" + order_id
    order = call(order_path, token=owner)
    assert order["status"] == "PAID", "Choose a dedicated PAID sandbox fixture"
    invoice = "OMAMA-" + order_id
    notification = None
    for notification_id in sql("SELECT id FROM sepay_notifications WHERE status='APPLIED' ORDER BY received_at").splitlines():
        entry = call(BASE + "/api/v1/admin/payments/notifications/" + str(uuid.UUID(notification_id)), token=admin)
        if entry["payload"].get("order", {}).get("order_invoice_number") == invoice and entry["payload"].get("timestamp", 0) > 1:
            notification = entry["payload"]
            break
    assert notification is not None, "A real, accepted provider IPN is required"
    public_ipn = config["SEPAY_RETURN_URL"].split("/payment-result")[0] + "/api/v1/payments/sepay/ipn"
    for name, headers in [("public_ipn_missing_key", {}), ("public_ipn_wrong_key", {"X-Secret-Key": "wrong-synthetic-key"})]:
        call(public_ipn, "POST", notification, headers=headers, expected=401)
        passed(name, "PUBLIC_HTTP_NEGATIVE")
    secret_header = {"X-Secret-Key": config["SEPAY_IPN_SECRET"]}
    for _ in range(3):
        call(public_ipn, "POST", notification, headers=secret_header)
    notification["timestamp"] += 1
    call(public_ipn, "POST", notification, headers=secret_header)
    # Wait for all delivery variants to be interpreted before checking counts.
    time.sleep(3)
    assert sql("SELECT count(*) FROM receipts WHERE order_id='" + order_id + "'") == "1"
    assert sql("SELECT count(*) FROM allocations WHERE order_id='" + order_id + "'") == "1"
    assert call(order_path, token=owner)["status"] == "PAID"
    passed("replay_real_ipn_same_and_changed_timestamp", "REPLAY_CAPTURED_PROVIDER_PAYLOAD", receipt_count=1, allocation_count=1)
    call(order_path + "/payment-checkout", "POST", {}, token=owner, expected=409)
    passed("paid_order_cannot_checkout_again", "LOCAL_HTTP_ON_REAL_SANDBOX_RECEIPT")
    other = account("CUSTOMER")
    call(order_path, token=other, expected=404)
    passed("other_customer_cannot_read_order", "LOCAL_HTTP_ON_REAL_SANDBOX_RECEIPT")
    review = call(BASE + "/api/v1/admin/payments/review", token=admin)
    receipt = next(r for r in review["receipts"] if r["order_id"] == order_id)["receipt_id"]
    refund_input = {"receipt_id": receipt, "amount": order["final_amount"], "reason": "Sandbox workflow test; no actual bank refund", "payee": {"bank_code": "SYNTHETIC", "account_number": "TEST-NO-REAL-ACCOUNT", "account_name": "Synthetic test recipient", "verification_reference": "SANDBOX-NOT-A-BANK-VERIFICATION"}}
    call(BASE + "/api/v1/admin/manual-refunds", "POST", refund_input, token=admin, headers={"X-Idempotency-Key": str(uuid.uuid4())}, expected=409)
    passed("refund_blocked_before_stock_reversal", "LOCAL_MANUAL_WORKFLOW_SYNTHETIC_EVIDENCE")
    order = call(order_path, token=owner)
    request = call(order_path + "/cancellation-requests", "POST", {"reason": "Sandbox cancellation test"}, token=owner, headers={"If-Match": str(order["version"]), "X-Idempotency-Key": str(uuid.uuid4())}, expected=202)
    decision_path = BASE + "/api/v1/admin/cancellation-requests/" + request["request_id"] + "/decision"
    decision = {"approve": True, "reason": "Sandbox test before packing"}
    for _ in range(2):
        call(decision_path, "POST", decision, token=admin)
    deadline = time.monotonic() + 30
    while time.monotonic() < deadline:
        order = call(order_path, token=owner)
        if order["status"] == "CANCELLED_BY_ADMIN" and order["stock_status"] == "RELEASED":
            break
        time.sleep(.5)
    assert order["status"] == "CANCELLED_BY_ADMIN" and order["stock_status"] == "RELEASED"
    passed("paid_cancel_review_replay_and_reverse_stock", "LOCAL_HTTP_WITH_SIMULATED_INVENTORY")
    refund_key = str(uuid.uuid4())
    refund_path = BASE + "/api/v1/admin/manual-refunds"
    refund = call(refund_path, "POST", refund_input, token=admin, headers={"X-Idempotency-Key": refund_key})
    replay = call(refund_path, "POST", refund_input, token=admin, headers={"X-Idempotency-Key": refund_key})
    assert refund["refund_id"] == replay["refund_id"]
    passed("refund_approval_replay", "LOCAL_MANUAL_WORKFLOW_SYNTHETIC_EVIDENCE")
    refund_path += "/" + refund["refund_id"]
    call(refund_path + "/claim", "POST", {}, token=admin)
    call(refund_path + "/claim", "POST", {}, token=verifier, expected=409)
    call(refund_path + "/unknown", "POST", {}, token=admin)
    call(BASE + "/api/v1/admin/manual-refunds", "POST", refund_input, token=verifier, headers={"X-Idempotency-Key": str(uuid.uuid4())}, expected=409)
    passed("exclusive_refund_claim_unknown_keeps_budget", "LOCAL_MANUAL_WORKFLOW_SYNTHETIC_EVIDENCE")
    evidence = {"source_bank_code": "SYNTHETIC", "source_account_number": "TEST-NO-REAL-ACCOUNT", "bank_transaction_id": "SYNTHETIC-" + str(uuid.uuid4()), "amount": refund_input["amount"], "paid_at": datetime.now(timezone.utc).isoformat(), "statement_reference": "SYNTHETIC-SANDBOX-NO-ACTUAL-TRANSFER"}
    refund = call(refund_path + "/evidence", "POST", evidence, token=admin)
    proof = {"evidence_hash": refund["evidence_hash"]}
    call(refund_path + "/confirm", "POST", proof, token=admin, expected=403)
    for _ in range(2):
        settled = call(refund_path + "/confirm", "POST", proof, token=verifier)
        assert settled["status"] == "SUCCEEDED"
    assert call(order_path, token=owner)["payment_status"] == "REFUNDED"
    assert sql("SELECT count(*) FROM outbox WHERE topic='order.manual_refunds.v1' AND aggregate_id='" + str(uuid.UUID(refund["refund_id"])) + "'") == "1"
    passed("independent_refund_verification_replay_single_event", "LOCAL_MANUAL_WORKFLOW_SYNTHETIC_EVIDENCE", actual_bank_transfer=False)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--order-id", required=True, help="Dedicated PAID SePay sandbox test order; will be cancelled/refunded locally")
    args = parser.parse_args()
    report = {"verified_at_utc": datetime.now(timezone.utc).isoformat(), "order_id": args.order_id, "checks": CHECKS, "actual_bank_refund": False}
    try:
        main(args.order_id)
        report["status"] = "PASS"
    except Exception as error:
        report["status"] = "FAIL"
        report["error_type"] = type(error).__name__
        raise
    finally:
        (ROOT / "docs/04_testing/order-service/live-sepay-additional-workflow.json").write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
