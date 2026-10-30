"""Read-only smoke test. Tokens come from environment and are never printed."""
import os
import urllib.error
import urllib.request
from urllib.parse import urlparse
from uuid import UUID


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


def main():
    base = os.environ.get("ORDER_TEST_URL", "http://127.0.0.1:18004").rstrip("/")
    parsed = urlparse(base)
    if parsed.scheme != "https" and not (parsed.scheme == "http" and parsed.hostname in ("localhost", "127.0.0.1", "::1")):
        raise SystemExit("Use HTTPS or a loopback HTTP test URL")
    tokens = [os.environ.get("ORDER_TEST_TOKEN_A", ""), os.environ.get("ORDER_TEST_TOKEN_B", "")]
    if not all(tokens) or tokens[0] == tokens[1]:
        raise SystemExit("Set distinct ORDER_TEST_TOKEN_A and ORDER_TEST_TOKEN_B access tokens")
    opener = urllib.request.build_opener(NoRedirect())

    def check(label, path, token, expected, method="GET"):
        headers = {"Authorization": "Bearer " + token} if token else {}
        request = urllib.request.Request(base + path, headers=headers, method=method)
        try:
            with opener.open(request, timeout=5) as response:
                status = response.status
        except urllib.error.HTTPError as error:
            status = error.code
            error.close()
        if status not in expected:
            raise SystemExit(f"FAIL {label}: HTTP {status}, expected {expected}")
        print(f"PASS {label}: HTTP {status}")

    for label, token in zip(("customer A cart", "customer B cart"), tokens):
        check(label, "/api/v1/cart", token, (200,))
    check("missing token", "/api/v1/cart", "", (401,))
    parts = tokens[0].split(".")
    if len(parts) != 3 or not parts[2]:
        raise SystemExit("Token A is not a JWT")
    parts[2] = ("A" if parts[2][0] != "A" else "B") + parts[2][1:]
    check("tampered signature", "/api/v1/cart", ".".join(parts), (401,))
    check("guest disabled", "/api/v1/guest-sessions", "", (403,), "POST")
    order = os.environ.get("ORDER_TEST_ORDER_A")
    if order:
        path = "/api/v1/orders/" + str(UUID(order))
        check("A reads own order", path, tokens[0], (200,))
        check("B cannot read A order", path, tokens[1], (403, 404))
    else:
        print("SKIP live order ownership: set ORDER_TEST_ORDER_A to an order owned by A")


if __name__ == "__main__":
    main()
