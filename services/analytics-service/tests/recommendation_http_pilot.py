"""Repeatable load check of the real HTTP engine with two isolated fixture customers."""

import argparse
import json
import math
import time
from collections import Counter
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path

import httpx


def run(base, duration, rps):
    cookies = [
        "f47ac10b-58cc-4372-a567-0e02b2c3d477",
        "f47ac10b-58cc-4372-a567-0e02b2c3d476",
    ]
    queries = [
        "placement=product_detail&productId=00000000-0000-0000-0000-000000000001&skuCode=SKU-1",
        "placement=cart&skuCodes=SKU-4",
        "placement=home",
    ]
    with httpx.Client(base_url=base, timeout=3, limits=httpx.Limits(max_connections=32)) as client:

        def request(number):
            start = time.monotonic()
            try:
                response = client.get(
                    "/api/v1/products/recommended?" + queries[number % 3],
                    headers={"Cookie": "OMAMA_SESSION=" + cookies[number % 2]},
                )
                payload = response.json()
                valid = response.status_code == 200 and bool(payload.get("items"))
                if valid and number % 3 == 1:
                    valid = (
                        payload["items"][0]["skuCode"] == "SKU-5"
                        or payload["strategy"] == "FALLBACK"
                    ) and all(item["skuCode"] != "SKU-4" for item in payload["items"])
                if valid:
                    items = payload["items"]
                    valid = (
                        len(items) <= 6
                        and len({i["productId"] for i in items}) == len(items)
                        and [i["rank"] for i in items] == list(range(1, len(items) + 1))
                        and all(i["priceVnd"] > 0 for i in items)
                    )
                return response.status_code, time.monotonic() - start, valid
            except (httpx.HTTPError, ValueError):
                return 0, time.monotonic() - start, False

        start = time.monotonic()
        with ThreadPoolExecutor(max_workers=24) as pool:
            futures = []
            for number in range(round(duration * rps)):
                delay = start + number / rps - time.monotonic()
                if delay > 0:
                    time.sleep(delay)
                futures.append(pool.submit(request, number))
            results = [f.result() for f in futures]
        elapsed = time.monotonic() - start
    latencies = sorted(row[1] * 1000 for row in results)
    report = {
        "requests": len(results),
        "durationSeconds": round(elapsed, 2),
        "targetRps": rps,
        "actualRps": round(len(results) / elapsed, 2),
        "statuses": dict(Counter(str(r[0]) for r in results)),
        "invalidResults": sum(not r[2] for r in results),
        "p95Ms": round(latencies[math.ceil(len(latencies) * 0.95) - 1], 2),
        "p99Ms": round(latencies[math.ceil(len(latencies) * 0.99) - 1], 2),
    }
    print(json.dumps(report, indent=2), flush=True)
    return report


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base", default="http://127.0.0.1:18112")
    parser.add_argument("--duration", type=int, default=30)
    parser.add_argument("--rps", type=int, default=20)
    parser.add_argument("--output", type=Path)
    args = parser.parse_args()
    if args.duration < 1 or not 1 <= args.rps <= 100:
        parser.error("duration >=1 and rps between 1 and 100 required")
    report = run(args.base, args.duration, args.rps)
    if args.output:
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    if report["invalidResults"] or report["p95Ms"] >= 300:
        raise SystemExit(1)
