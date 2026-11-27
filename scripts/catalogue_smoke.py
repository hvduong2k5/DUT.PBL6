#!/usr/bin/env python3
"""Catalogue consistency smoke for the administrative dataset (PR-B).

Public mode (default; used by the CD smoke job after a deploy):
  1. Fetch the locations catalogue the storefront serves and compare its
     canonical checksum with the checksum of the canonical dataset file
     in this repo (same algorithm as order-service's AdministrativeValidator:
     SHA-256 over sorted "LEVEL|CODE|PARENT" lines joined by newlines).
  2. Probe guest checkout quotes through the real BFF: a valid pair must
     quote 200, a retired pair and a cross pair must answer
     422 INVALID_ADMINISTRATIVE_UNIT. While the catalogue-maintenance
     barrier is on, the valid probe answers 503 CATALOGUE_MAINTENANCE
     instead (reported distinctly; pass --expect-maintenance during a
     catalogue-change window).

Internal mode (--internal-base + --token): read the simulator's
/catalogue/status (dataset version, runtime checksum, maintenance flag)
and compare the runtime checksum with the repo dataset.

Exit codes: 0 = consistent, 1 = mismatch/failure, 3 = maintenance active
(public mode without --expect-maintenance).
"""
import argparse
import hashlib
import json
import subprocess
import sys
import tempfile
import os

sys.stdout.reconfigure(encoding="utf-8", errors="replace")
UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Safari/537.36"


def dataset_checksum(units):
    lines = []
    for u in units:
        parent = u.get("parent_code") or ""
        lines.append(f"{u['level']}|{u['code']}|{parent}")
    lines.sort()
    return hashlib.sha256("\n".join(lines).encode("utf-8")).hexdigest()


def served_checksum(provinces):
    lines = []
    for p in provinces:
        lines.append(f"PROVINCE|{p['provinceCode']}|")
        for w in p.get("wards", []):
            lines.append(f"WARD|{w['wardCode']}|{p['provinceCode']}")
    lines.sort()
    return hashlib.sha256("\n".join(lines).encode("utf-8")).hexdigest()


def curl(base, jar, path, body=None, headers=None):
    cmd = ["curl", "-sS", "-A", UA, "-b", jar, "-c", jar, "-H", f"Origin: {base}"]
    for k, v in (headers or {}).items():
        cmd += ["-H", f"{k}: {v}"]
    if body is not None:
        cmd += ["-H", "Content-Type: application/json", "--data", json.dumps(body)]
    cmd += ["-w", "\n%{http_code}", base + path]
    out = subprocess.run(cmd, capture_output=True, text=True, timeout=60).stdout
    raw, _, status = out.rpartition("\n")
    try:
        payload = json.loads(raw) if raw.strip() else {}
    except json.JSONDecodeError:
        payload = {"raw": raw[:200]}
    return int(status), payload


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--base", default="https://pbl.npanhbao.dev")
    ap.add_argument("--dataset", default="services/order-service/internal/simulator/administrative_units.json")
    ap.add_argument("--sku", default="MX-500")
    ap.add_argument("--valid-pair", default="46,19753")
    ap.add_argument("--retired-pair", default="75,WARD-TH-001")
    ap.add_argument("--cross-pair", default="46,20242")
    ap.add_argument("--expect-maintenance", action="store_true")
    ap.add_argument("--internal-base", default="")
    ap.add_argument("--token", default="")
    args = ap.parse_args()

    with open(args.dataset, encoding="utf-8") as f:
        dataset = json.load(f)
    want = dataset_checksum(dataset["units"])
    print(f"repo dataset: version={dataset['version']} units={len(dataset['units'])} checksum={want}")

    if args.internal_base:
        if not args.token:
            print("FAIL: --token is required with --internal-base")
            return 1
        status, payload = curl(args.internal_base, os.devnull, "/catalogue/status",
                               headers={"X-Internal-Token": args.token})
        print(f"simulator status: HTTP {status} {payload}")
        if status != 200:
            print("FAIL: cannot read simulator catalogue status")
            return 1
        if payload.get("dataset_checksum") != want:
            print(f"FAIL: runtime checksum {payload.get('dataset_checksum')} != repo {want}")
            return 1
        if payload.get("dataset_version") != dataset["version"]:
            print(f"FAIL: runtime dataset version {payload.get('dataset_version')} != repo {dataset['version']}")
            return 1
        print("PASS: simulator runtime dataset matches the repo dataset")
        return 0

    jar = os.path.join(tempfile.mkdtemp(), "cookies.txt")
    status, locations = curl(args.base, jar, "/api/customer/locations")
    if status != 200 or not locations.get("provinces"):
        print(f"FAIL: locations endpoint HTTP {status}")
        return 1
    got = served_checksum(locations["provinces"])
    n_wards = sum(len(p.get("wards", [])) for p in locations["provinces"])
    print(f"served catalogue: {len(locations['provinces'])} provinces, {n_wards} wards, checksum={got}")
    if got != want:
        print("FAIL: served catalogue checksum != repo dataset checksum (profile/order drift)")
        return 1
    print("PASS: served catalogue matches the repo dataset")

    status, cart = curl(args.base, jar, "/api/cart")
    if status != 200:
        print(f"FAIL: guest cart bootstrap HTTP {status}")
        return 1
    status, cart = curl(args.base, jar, "/api/cart/items",
                        {"sku_code": args.sku, "quantity": 1, "revision": cart["revision"]})
    if status not in (200, 201):
        print(f"FAIL: add to cart HTTP {status}: {cart}")
        return 1

    def quote(pair):
        province, ward = pair.split(",")
        body = {"cart_revision": cart["revision"], "payment_method": "COD",
                "address": {"recipient_name": "Catalogue Smoke", "phone": "0905000111",
                            "street": "1 Smoke Test", "province_code": province, "ward_code": ward}}
        return curl(args.base, jar, "/api/checkout/quote", body)

    status, payload = quote(args.valid_pair)
    code = (payload.get("error") or {}).get("code") or payload.get("code")
    if status == 503 and code == "CATALOGUE_MAINTENANCE":
        print("maintenance barrier is ON (valid-pair quote answered 503 CATALOGUE_MAINTENANCE)")
        return 0 if args.expect_maintenance else 3
    if status != 200:
        print(f"FAIL: valid pair {args.valid_pair} quote HTTP {status}: {payload}")
        return 1
    print(f"PASS: valid pair {args.valid_pair} quotes 200")
    for label, pair in (("retired", args.retired_pair), ("cross", args.cross_pair)):
        status, payload = quote(pair)
        code = (payload.get("error") or {}).get("code") or payload.get("code")
        if status != 422 or code != "INVALID_ADMINISTRATIVE_UNIT":
            print(f"FAIL: {label} pair {pair} -> HTTP {status} code={code}, want 422 INVALID_ADMINISTRATIVE_UNIT")
            return 1
        print(f"PASS: {label} pair {pair} rejected 422 INVALID_ADMINISTRATIVE_UNIT")
    print("PASS: catalogue smoke complete")
    return 0


if __name__ == "__main__":
    sys.exit(main())
