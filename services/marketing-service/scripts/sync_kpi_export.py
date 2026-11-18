"""Submit one normalized Marketing KPI export to an already-bound execution."""

import argparse
import csv
from contextlib import closing
import hashlib
import io
import json
import os
import re
import sqlite3
import time
from pathlib import Path
from urllib.error import HTTPError, URLError
from urllib.parse import urlsplit
from urllib.request import HTTPRedirectHandler, Request, build_opener
from uuid import UUID

from normalize_kpi_export import TARGET_HEADER, integer

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
    versions = {integer(row[2], 1) for row in rows[1:]}
    if len(versions) != 1:
        raise ValueError("Expected one source version per export batch")
    expected = [
        {"date": row[1], "status": "REPORTED", "sourceVersion": integer(row[2], 1),
         "impressions": integer(row[3], 0), "clicks": integer(row[4], 0)}
        for row in rows[1:]
    ]
    return payload, key, versions.pop(), expected


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


def sync(checkpoint, path, token, payload, key, version, expected):
    digest = hashlib.sha256(payload).hexdigest()
    failure = None
    status = "IMPORTED"
    with closing(sqlite3.connect(checkpoint, timeout=5)) as db, db:
        db.execute("""
            CREATE TABLE IF NOT EXISTS kpi_batches (
                destination TEXT NOT NULL,
                version INTEGER NOT NULL,
                sha256 TEXT NOT NULL,
                status TEXT NOT NULL CHECK (status IN ('IMPORTED', 'FAILED')),
                error TEXT,
                updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
                PRIMARY KEY (destination, version)
            )
            """)
        db.execute("BEGIN IMMEDIATE")
        previous = db.execute(
            "SELECT sha256, status FROM kpi_batches WHERE destination = ? AND version = ?",
            (path, version),
        ).fetchone()
        if previous and previous[0] != digest:
            raise ValueError("Checkpoint already has a different payload for this version")
        if previous and previous[1] == "IMPORTED":
            return "ALREADY_IMPORTED", digest
        latest = db.execute(
            "SELECT max(version) FROM kpi_batches WHERE destination = ? AND status = 'IMPORTED'",
            (path,),
        ).fetchone()[0]
        if latest is not None and version < latest:
            raise ValueError("Export version precedes the last imported version")
        try:
            opener = build_opener(NoRedirect())
            source = call(opener, path + "/source", token)
            if not isinstance(source, dict) or source.get("type") != "CSV" or source.get("externalExecutionKey") != key:
                raise ValueError("Execution is not bound to this CSV key")
            receipt = call(opener, path + "/csv", token, payload)
            if receipt != expected:
                raise ValueError("Import receipt did not match the submitted snapshots")
        except (OSError, ValueError) as error:
            failure = error
            status = "FAILED"
        db.execute("""
            INSERT INTO kpi_batches (destination, version, sha256, status, error)
            VALUES (?, ?, ?, ?, ?)
            ON CONFLICT (destination, version) DO UPDATE SET
                status = excluded.status, error = excluded.error, updated_at = CURRENT_TIMESTAMP
            """, (path, version, digest, status, str(failure) if failure else None))
    if failure:
        raise failure
    return status, digest


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("csv", type=Path)
    parser.add_argument("--gateway", required=True, type=gateway)
    parser.add_argument("--checkpoint", required=True, type=Path)
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
        payload, key, version, expected = read_batch(args.csv)
        status, digest = sync(args.checkpoint, path, token, payload, key, version, expected)
    except (OSError, ValueError, csv.Error, sqlite3.Error) as error:
        parser.exit(2, f"KPI sync failed: {error}\n")
    print(json.dumps({
        "status": status,
        "executionId": str(args.execution),
        "sourceVersion": version,
        "sha256": digest,
        "days": len(expected),
    }))


if __name__ == "__main__":
    main()
