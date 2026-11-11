"""Synthetic dev fixture for browser/load checks against the real MS-11 HTTP runtime."""

import argparse
import os
import tempfile
import time
from datetime import UTC, datetime, timedelta
from pathlib import Path
from uuid import uuid4

import httpx
from test_recommendation import basket_row, catalog_row, inventory_row, snapshot

from analytics_service.recommendation import engine
from analytics_service.recommendation.models import Config
from analytics_service.recommendation.sources import read_snapshot
from analytics_service.recommendation.store import Store

SESSIONS = (
    ("f47ac10b-58cc-4372-a567-0e02b2c3d477", "27bc0f3a-69e5-4aa7-8644-3c7fcb7b3660"),
    ("f47ac10b-58cc-4372-a567-0e02b2c3d476", "00000000-0000-0000-0000-000000002711"),
    ("f47ac10b-58cc-4372-a567-0e02b2c3d475", "00000000-0000-0000-0000-000000002712"),
)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--watch", action="store_true")
    args = parser.parse_args()
    api = os.getenv("RECOMMENDATION_API", "http://recommendation-api:8011")
    config = Config(
        "postgresql://analytics_rec_worker:local-rec-worker@postgres/analytics",
        "d2c-hue",
        "a7277eb6-0fd2-4850-b640-169bc446387e",
        "warehouse-supply-pilot",
        b"isolated-recommendation-test-key-32-bytes",
        export_root=Path(tempfile.mkdtemp()),
    )
    store = Store(config)
    grants = []
    for cookie, _ in SESSIONS:
        with httpx.Client(
            base_url=api,
            cookies={"OMAMA_SESSION": cookie},
            headers={"Origin": "http://127.0.0.1:13046"},
        ) as client:
            current = client.get("/api/v1/recommendations/consent")
            current.raise_for_status()
            value = current.json()
            if not value["enabled"]:
                result = client.put(
                    "/api/v1/recommendations/consent",
                    headers={"If-Match": current.headers["etag"], "Idempotency-Key": str(uuid4())},
                    json={"enabled": True, "policyVersion": "personalization-v1"},
                )
                result.raise_for_status()
                value = result.json()
            grants.append(engine.timestamp(value["grantedAt"]))
    completed = max(grants) + timedelta(microseconds=1)
    time.sleep(6.1)
    while True:
        now = datetime.now(UTC)
        catalog = [catalog_row(n, category=1 if n < 4 else 2) for n in range(1, 7)]
        inventory = [inventory_row(n, now) for n in range(1, 7)]
        baskets = [
            basket_row(
                n,
                ["SKU-4", "SKU-5"] if n < 5 else ["SKU-2"],
                SESSIONS[n % 3][1],
                completed + timedelta(seconds=5),
            )
            for n in range(20)
        ]
        for name, records in (
            ("catalog", catalog),
            ("sellability", inventory),
            ("completed_baskets", baskets),
        ):
            with store.connect() as conn:
                revision = conn.execute(
                    "SELECT coalesce(max(source_revision),0)+1 AS n FROM source_snapshot WHERE scope_id=%s AND dataset=%s",
                    (config.scope, name),
                ).fetchone()["n"]
            folder = snapshot(config, name, records, revision, now)
            manifest, rows = read_snapshot(folder, config, now)
            store.import_snapshot(manifest, rows, now)
        generation = store.build(now)
        print(f"Synthetic pilot generation={generation} catalog=6 baskets=20", flush=True)
        if not args.watch:
            break
        time.sleep(30)


if __name__ == "__main__":
    main()
