"""F-53 checks against the disposable real Web/Kong/Identity/Order stack.

These TEST statement transfers are local integration evidence, never live proofs.
"""
import http.cookiejar
import time
import urllib.request
import uuid


def guest_purchases(request, web, control):
    for method in ["COD", "VIETQR"]:
        jar = http.cookiejar.CookieJar()
        opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(jar))

        def call(path, body=None, verb=None, headers=None):
            return request(web + path, body, method=verb, opener=opener,
                           headers={"Origin": web, **(headers or {})})

        status, cart, _ = call("/api/cart")
        assert status == 200 and not cart["items"] and not list(jar), "page read created a guest session"
        status, cart, _ = call("/api/cart/items", {"sku_code": "PEER", "quantity": 1, "revision": 0}, "POST")
        assert status == 200, ("guest cart", method, status, cart)
        assert any(c.name == "oma_guest_session" and c.has_nonstandard_attr("HttpOnly") for c in jar)
        assert not any(c.name == "OMAMA_SESSION" for c in jar), "guest acquired identity material"
        quote_input = {"cart_revision": cart["revision"], "payment_method": method,
                       "address": {"recipient_name": "Guest TEST", "phone": "0905123456",
                                   "street": "Test only", "province_code": "75", "ward_code": "HUE-01"}}
        status, quote, _ = call("/api/checkout/quote", quote_input, "POST")
        assert status == 200, ("guest quote", method, status, quote)
        key = str(uuid.uuid4())
        accepted = {"quote_id": quote["quote_id"], "cart_revision": cart["revision"], "payment_method": method}
        status, operation, _ = call("/api/checkout", accepted, "POST", {"Idempotency-Key": key})
        assert status in [201, 202], ("guest checkout", method, status, operation)
        for _ in range(30):
            if operation["status"] == "SUCCEEDED":
                break
            status, operation, _ = call("/api/checkout/operations/" + operation["operation_id"])
            assert status == 200, ("guest operation", status, operation)
        assert operation["status"] == "SUCCEEDED", operation
        order_id = operation["order_id"]
        status, order, _ = call("/api/orders/" + order_id)
        assert status == 200 and order["order_code"], ("guest order detail", method, status, order)
        if method == "COD":
            assert order["status"] == "CONFIRMED_COD" and order["payment_expires_at"] is None
        else:
            status, missing, _ = call(f"/api/payments/orders/{order_id}/status")
            assert status == 401 and missing["code"] == "GUEST_PAYMENT_ACCESS_REQUIRED"
            status, _, _ = call(f"/api/payments/orders/{order_id}/guest-access", {}, "POST")
            assert status == 204, ("guest payment grant", status)
            assert any(c.name.startswith("oma_guest_payment_") and c.has_nonstandard_attr("HttpOnly") for c in jar)
            for _ in range(20):
                status, payment, _ = call(f"/api/payments/orders/{order_id}/status")
                assert status == 200, ("guest payment poll", status, payment)
                if payment["can_pay"]:
                    break
            assert payment["can_pay"], payment
            assert call(f"/api/payments/orders/{order_id}/instructions")[0] == 200
            assert request(control + "/pay?order=" + order_id, {}, method="POST")[0] == 202
            for _ in range(25):
                status, payment, _ = call(f"/api/payments/orders/{order_id}/status")
                if status == 200 and payment["order_status"] == "PAID":
                    break
            assert payment["order_status"] == "PAID", ("guest TEST settlement", payment)
        print("PASS: logged-out real Web BFF -> gateway -> guest " + method + " checkout/owned reads" +
              ("/order-grant polling/TEST settlement" if method == "VIETQR" else "/CONFIRMED_COD"), flush=True)


def prepare_merge_source(request, web, jar, opener):
    revision = 0
    for sku, quantity in [("PEER", 2), ("DELETED-SKU", 1)]:
        status, cart, _ = request(web + "/api/cart/items", {"sku_code": sku, "quantity": quantity, "revision": revision},
                                  method="POST", opener=opener, headers={"Origin": web})
        assert status == 200, ("guest merge source", status, cart)
        revision = cart["revision"]
    return next(c.value for c in jar if c.name == "oma_guest_session")


def merge_after_login(request, web, gateway, jar, opener, source, cart):
    if source is None:
        assert request(web + "/api/cart/merge", {}, method="POST", opener=opener, headers={"Origin": web})[0] == 204
        return
    status, cart, _ = request(web + "/api/cart/items", {"sku_code": "PEER", "quantity": 3, "revision": cart["revision"]},
                              method="POST", opener=opener, headers={"Origin": web})
    assert status == 200, ("customer target cart", status, cart)
    status, result, _ = request(web + "/api/cart/merge", {}, method="POST", opener=opener, headers={"Origin": web})
    assert status == 200, ("merge through existing identity proxy", status, result)
    assert next(i for i in result["cart"]["items"] if i["sku_code"] == "PEER")["quantity"] == 5
    assert any(i["sku_code"] == "DELETED-SKU" and i["status"] == "dropped" for i in result["items"])
    assert not any(c.name == "oma_guest_session" for c in jar), "successful merge did not clear source cookie"
    status, retry, _ = request(gateway + "/api/v1/customer-api/cart", {"guest_token": source}, method="POST", opener=opener,
                               headers={"Origin": web, "Idempotency-Key": result["merge_key"]})
    assert status == 200 and retry == result, ("merge replay after revocation", status, retry)
    assert request(gateway + "/api/v1/guest-sessions/current", token=source, method="DELETE")[0] == 401
    status, fetched, _ = request(web + "/api/cart/merge?merge_key=" + result["merge_key"], opener=opener)
    assert status == 200 and fetched == result, "customer cannot retrieve persisted result"
    print("PASS: guest x2 + Customer x3 -> x5 through real OIDC/Identity proxy; dropped SKU, replay and revocation", flush=True)
