"""Submit one normalized Marketing KPI export to an already-bound execution."""

import argparse
import csv
import hashlib
import io
import json
import os
import re
import sys
import time
from pathlib import Path
from urllib.error import HTTPError, URLError
from urllib.parse import urlsplit
from urllib.request import HTTPRedirectHandler, Request, build_opener
from uuid import UUID

from normalize_kpi_export import TARGET_HEADER

MAX_CSV_BYTES = 8192
MAX_RESPONSE_BYTES = 65536


class NoRedirect(HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


def gateway(value):
    url = urlsplit(value)
    if (url.scheme not in ("https", "http") or not url.hostname or url.username or url.password
            or url.path not in ("", "/") or url.query or url.fragment):
        raise argparse.ArgumentTypeError("Gateway must be an HTTPS origin or an HTTP loopback origin")
    if url.scheme == "http" and url.hostname not in ("127.0.0.1", "::1", "localhost"):
        raise argparse.ArgumentTypeError("HTTP is only supported on loopback")
    return value.rstrip("/")


def read_batch(path):
    with path.open("rb") as handle:
        payload = handle.read(MAX_CSV_BYTES + 1)
    if len(payload) > MAX_CSV_BYTES:
        raise ValueError("Normalized CSV exceeds 8192 bytes")
    rows = list(csv.reader(io.StringIO(payload.decode("utf-8-sig")), strict=True))
    if not 2 <= len(rows) <= 32 or rows[0] != TARGET_HEADER:
        raise ValueError("Expected a normalized CSV with 1 to 31 daily rows")
    key = rows[1][0] if rows[1] else ""
    if not re.fullmatch(r"[A-Za-z0-9._:-]{1,160}", key) or any(len(row) != 5 or row[0] != key for row in rows[1:]):
        raise ValueError("Expected exactly one valid external execution key")
    return payload, key, len(rows) - 1


def call(opener, url, token, payload=None):
    headers = {"Authorization": f"Bearer {token}", "Accept": "application/json"}
    if payload is not None:
        headers["Content-Type"] = "text/csv; charset=utf-8"
    for attempt in range(3):
        request = Request(url, data=payload, headers=headers, method="GET" if payload is None else "PUT")
        try:
            with opener.open(request, timeout=15) as response:
                if response.status != 200:
                    raise ValueError(f"Unexpected gateway HTTP {response.status}")
                body = response.read(MAX_RESPONSE_BYTES + 1)
                if len(body) > MAX_RESPONSE_BYTES:
                    raise ValueError("Gateway response exceeded size limit")
                return json.loads(body)
        except HTTPError as error:
            with error:
                if error.code not in (502, 503, 504) or attempt == 2:
                    code = ""
                    try:
                        value = json.loads(error.read(MAX_RESPONSE_BYTES)).get("code", "")
                        if isinstance(value, str) and re.fullmatch(r"[A-Z][A-Z0-9_]{0,63}", value):
                            code = f" ({value})"
                    except (AttributeError, ValueError):
                        pass
                    raise ValueError(f"Gateway HTTP {error.code}{code}") from None
        except (URLError, TimeoutError, ConnectionError):
            if attempt == 2:
                raise ValueError("Gateway transport retry limit reached") from None
        time.sleep(2 ** attempt)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("csv", type=Path)
    parser.add_argument("--gateway", required=True, type=gateway)
    for field in ("campaign", "revision", "execution"):
        parser.add_argument(f"--{field}", required=True, type=UUID)
    args = parser.parse_args()
    token = os.getenv("MARKETING_KPI_BEARER_TOKEN", "")
    if not re.fullmatch(r"[A-Za-z0-9._~-]{1,8192}", token):
        parser.exit(2, "Set MARKETING_KPI_BEARER_TOKEN to a valid staff bearer token\n")
    path = (
        f"{args.gateway}/api/v1/admin/marketing/campaigns/{args.campaign}/revisions/{args.revision}"
        f"/executions/{args.execution}/metrics"
    )
    try:
        payload, key, row_count = read_batch(args.csv)
        opener = build_opener(NoRedirect())
        source = call(opener, path + "/source", token)
        if not isinstance(source, dict) or source.get("type") != "CSV" or source.get("externalExecutionKey") != key:
            raise ValueError("Execution is not bound to this CSV key")
        rows = call(opener, path + "/csv", token, payload)
        if not isinstance(rows, list) or len(rows) != row_count:
            raise ValueError("Unexpected import receipt")
    except (OSError, ValueError, csv.Error) as error:
        parser.exit(2, f"KPI sync failed: {error}\n")
    print(json.dumps({
        "status": "IMPORTED",
        "executionId": str(args.execution),
        "sha256": hashlib.sha256(payload).hexdigest(),
        "days": row_count,
    }))


if __name__ == "__main__":
    main()
