"""Create a synthetic CUSTOMER order; write a private, loopback-only demo page.

Run after starting the mvp1 sandbox stack. No merchant secret is read or emitted.
Generated state (including the local test token) stays in ignored .system_generated,
outside the directory served by the demo HTTP server.
"""
import html
import json
import time
import uuid
from pathlib import Path
from urllib.request import Request, urlopen

ROOT = Path(__file__).resolve().parents[3]
STATE = ROOT / ".system_generated/order-sepay"
BASE = "http://127.0.0.1:18004"


def request(url, body=None, headers=None):
    req = Request(url, data=None if body is None else json.dumps(body).encode(),
                  headers={"Content-Type": "application/json", **(headers or {})})
    with urlopen(req, timeout=10) as response:
        return json.load(response)


def main():
    account = request("http://127.0.0.1:18104/test/token", {"role": "CUSTOMER"},
                      {"X-Internal-Token": "order-sandbox-internal-token-2026"})
    auth = {"Authorization": "Bearer " + account["token"]}
    cart = request(BASE + "/api/v1/cart/items", {"sku_code": "MX-GION-500G", "quantity": 1, "revision": 0}, auth)
    quote = request(BASE + "/api/v1/checkout/quote", {"payment_method": "VIETQR", "cart_revision": cart["revision"], "shipping_address_id": account["address_id"]}, auth)
    operation = request(BASE + "/api/v1/checkout", {"payment_method": "VIETQR", "cart_revision": cart["revision"], "quote_id": quote["quote_id"]}, {**auth, "X-Idempotency-Key": str(uuid.uuid4())})
    deadline = time.monotonic() + 30
    while operation["status"] != "SUCCEEDED":
        if operation["status"] in ("FAILED", "MANUAL_REVIEW") or time.monotonic() >= deadline:
            raise RuntimeError("Synthetic checkout did not complete: " + operation["status"])
        time.sleep(.25)
        operation = request(BASE + "/api/v1/checkout-operations/" + operation["operation_id"], headers=auth)
    order_id = operation["order_id"]
    checkout = request(BASE + "/api/v1/orders/" + order_id + "/payment-checkout", {}, auth)
    if checkout["action"] != "https://pay-sandbox.sepay.vn/v1/checkout/init":
        raise RuntimeError("Refusing a checkout outside SePay sandbox")
    STATE.mkdir(parents=True, exist_ok=True)
    (STATE / "session.json").write_text(json.dumps({"order_id": order_id, "token": account["token"], "checkout": checkout}), encoding="utf-8")
    public = STATE / "public"
    (public / "result").mkdir(parents=True, exist_ok=True)
    fields = "\n".join('<input type="hidden" name="{}" value="{}">'.format(html.escape(f["name"], quote=True), html.escape(f["value"], quote=True)) for f in checkout["fields"])
    amount = next(f["value"] for f in checkout["fields"] if f["name"] == "order_amount")
    shell = '<!doctype html><html lang="vi"><meta charset="utf-8"><meta name="referrer" content="no-referrer"><title>Order — SePay Sandbox</title><style>body{font:18px system-ui;max-width:740px;margin:60px auto;padding:20px;line-height:1.6}button{font:inherit;padding:12px 20px;cursor:pointer}</style>'
    page = shell + '<h1>Thử thanh toán SePay Sandbox</h1><p>Đơn mẫu: <b>' + html.escape(checkout["invoice"]) + '</b></p><p>Số tiền giả lập: <b>' + html.escape(amount) + ' VND</b></p><p>Hạn thanh toán: ' + html.escape(checkout["expires_at"]) + ' (UTC). Chỉ dùng thao tác mô phỏng trên trang sandbox; không chuyển khoản tiền thật.</p><form method="post" action="' + checkout["action"] + '">' + fields + '<button>Mở trang thanh toán thử SePay</button></form></html>'
    (public / "index.html").write_text(page, encoding="utf-8")
    (public / "result/index.html").write_text(shell + '<h1>Đã quay về từ SePay</h1><p>Trang quay về không phải bằng chứng đã trả tiền. Backend sẽ xác nhận bằng IPN hoặc tra cứu SePay. Hãy quay lại chat để mình kiểm tra trạng thái đơn thực tế.</p></html>', encoding="utf-8")
    print(json.dumps({"order_id": order_id, "invoice": checkout["invoice"], "amount_vnd": amount, "expires_at": checkout["expires_at"], "demo_url": "http://127.0.0.1:18006/"}))


if __name__ == "__main__":
    main()
