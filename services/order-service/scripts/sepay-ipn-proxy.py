"""Expose only the SePay IPN route while keeping sandbox admin APIs local."""
import argparse
import hmac
import json
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.error import HTTPError, URLError
from urllib.parse import urlsplit
from urllib.request import Request, urlopen

ENV_FILE = Path(__file__).resolve().parents[1] / ".env.local"
IPN_PATH = "/api/v1/payments/sepay/ipn"


def ipn_secret():
    if not ENV_FILE.exists():
        return ""
    for line in ENV_FILE.read_text(encoding="utf-8-sig").splitlines():
        key, separator, value = line.partition("=")
        if separator and key.strip() == "SEPAY_IPN_SECRET":
            return value.strip().strip("\"'")
    return ""


class Handler(BaseHTTPRequestHandler):
    def log_message(self, *_):
        pass  # Never log credentials or callback bodies.

    def respond(self, status, body):
        if self.command == "POST" and urlsplit(self.path).path == IPN_PATH:
            print(json.dumps({"event": "ipn_response", "time_unix": int(time.time()), "http_status": status}), flush=True)
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Cache-Control", "no-store")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):
        if self.path == "/healthz":
            self.respond(200, b'{"ipn_proxy":true}')
        elif urlsplit(self.path).path == "/payment-result":
            # Static display only: never treat a query parameter as payment proof.
            body = ('<!doctype html><html lang="vi"><meta charset="utf-8">'
                    '<title>SePay Sandbox</title><h1>Da quay ve tu SePay</h1>'
                    '<p>Backend se xac nhan thanh toan bang IPN hoac tra cuu SePay. '
                    'Trang quay ve khong phai bang chung da tra tien. '
                    'Quay lai chat de kiem tra trang thai don.</p></html>').encode()
            self.send_response(200)
            self.send_header("Content-Type", "text/html; charset=utf-8")
            self.send_header("Cache-Control", "no-store")
            self.send_header("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
        else:
            self.respond(404, b'{"success":false}')

    def do_POST(self):
        if urlsplit(self.path).path != IPN_PATH:
            self.respond(404, b'{"success":false}')
            return
        secret = ipn_secret()
        if not secret:
            self.respond(503, b'{"success":false,"code":"SETUP_PENDING"}')
            return
        if not hmac.compare_digest(secret.encode(), self.headers.get("X-Secret-Key", "").encode()):
            self.respond(401, b'{"success":false}')
            return
        try:
            length = int(self.headers.get("Content-Length", "0"))
        except ValueError:
            length = 0
        if not 0 < length <= 65536:
            self.respond(413, b'{"success":false}')
            return
        self.connection.settimeout(10)
        try:
            body = self.rfile.read(length)
            if len(body) != length or not isinstance(json.loads(body), dict):
                raise ValueError("invalid body")
        except (ValueError, OSError):
            self.respond(400, b'{"success":false}')
            return
        request = Request("http://127.0.0.1:18004" + IPN_PATH, data=body, headers={"Content-Type": "application/json", "X-Secret-Key": secret}, method="POST")
        try:
            with urlopen(request, timeout=5) as response:
                self.respond(response.status, response.read(65536))
        except HTTPError as error:
            self.respond(error.code, b'{"success":false}')
        except (URLError, TimeoutError, OSError):
            self.respond(503, b'{"success":false,"code":"BACKEND_UNAVAILABLE"}')


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--port", type=int, default=18005)
    args = parser.parse_args()
    ThreadingHTTPServer(("127.0.0.1", args.port), Handler).serve_forever()
