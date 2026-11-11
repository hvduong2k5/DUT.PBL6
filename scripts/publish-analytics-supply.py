"""Owner-side Supply exporter. PostgreSQL credentials come from the caller's libpq environment."""

import argparse
import json
import os
import re
import subprocess
import tempfile
from datetime import date, datetime
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SOURCES = {
    "order_day": "order-service",
    "inventory_day": "inventory-service",
    "inventory_batches": "inventory-service",
}
QUERIES = {
    "order_day": "analytics-supply-day.sql",
    "inventory_batches": "analytics-supply-batches.sql",
}
CODE = re.compile(r"[A-Za-z0-9][A-Za-z0-9_.-]{0,63}")


def checked_code(value):
    if not isinstance(value, str) or not CODE.fullmatch(value):
        raise ValueError("Invalid warehouse or SKU code")
    return value


def quantity(value, maximum=2_147_483_647):
    if type(value) is not int or not 0 <= value <= maximum:
        raise ValueError("Expected a non-negative integer from an approved source")
    return value


def approved_text(value, maximum):
    if not isinstance(value, str) or not 1 <= len(value.strip()) <= maximum:
        raise ValueError("Approved actor and evidence reference are required")
    return value.strip()


def owner_query(owner, kind, variables):
    query = ROOT / "services" / owner / "scripts" / QUERIES[kind]
    args = ["psql", "-X", "-qAt", "-v", "ON_ERROR_STOP=1", "-f", str(query)]
    for key, value in variables.items():
        args.extend(("-v", f"{key}={value}"))
    result = subprocess.run(
        args,
        capture_output=True,
        text=True,
        timeout=20,
        env={
            **os.environ,
            "PGOPTIONS": "-c default_transaction_read_only=on -c statement_timeout=10000",
        },
    )
    if result.returncode or not result.stdout.strip():
        raise ValueError("Owner query failed or evidence does not reconcile with source data")
    return json.loads(result.stdout)


def export(owner, item, manifest):
    kind = item["kind"]
    if SOURCES.get(kind) != owner:
        raise ValueError("Export kind belongs to a different data owner")
    sku = checked_code(item["skuCode"])
    warehouse = checked_code(manifest["warehouseId"])
    day = date.fromisoformat(item["businessDate"])
    revision = quantity(item["revision"])
    if revision == 0:
        raise ValueError("Revision must be positive")
    result = {
        "schemaVersion": "1",
        "kind": kind,
        "source": owner,
        "skuCode": sku,
        "warehouseId": warehouse,
        "businessDate": day.isoformat(),
        "revision": revision,
        "approvedBy": approved_text(manifest["approvedBy"], 128),
        "evidenceRef": approved_text(item["evidenceRef"], 256),
    }
    if kind != "inventory_batches":
        as_of = datetime.fromisoformat(item["sourceAsOf"])
        if as_of.tzinfo is None:
            raise ValueError("Evidence sourceAsOf must include an offset")
        result["sourceAsOf"] = as_of.isoformat()
    if kind == "inventory_day":
        # Current Inventory schema has no complete historical zero-stock intervals.
        # Accept only the owner's explicit, reconciled count for the full day.
        return {**result, "stockoutMinutes": quantity(item["stockoutMinutes"], 1440)}
    params = {
        "source_warehouse": checked_code(manifest["sourceWarehouseId"]),
        "sku": sku,
        "day": day,
    }
    if kind == "order_day":
        params.update(
            expected_sold=quantity(item["expectedSoldUnits"]),
            returned=quantity(item["returnedUnits"]),
            as_of=result["sourceAsOf"],
        )
    return {**result, **owner_query(owner, kind, params)}


def publish(folder, document):
    target = folder / (
        "{kind}--{warehouseId}--{skuCode}--{businessDate}--{revision:010}.json".format(**document)
    )
    data = (json.dumps(document, ensure_ascii=True, sort_keys=True) + "\n").encode()
    with tempfile.NamedTemporaryFile(dir=folder, delete=False) as temp:
        temp.write(data)
        source = Path(temp.name)
    try:
        # Link a complete file into the inbox without replacing an existing revision.
        os.chmod(source, 0o640)
        try:
            os.link(source, target)
        except FileExistsError:
            previous = json.loads(target.read_text(encoding="utf-8"))
            if document["kind"] == "inventory_batches":
                previous["sourceAsOf"] = document["sourceAsOf"]
            if previous != document:
                raise ValueError("Published revision already exists with different data")
            return "duplicate"
        return "published"
    finally:
        source.unlink()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("owner", choices=sorted(set(SOURCES.values())))
    parser.add_argument("manifest", type=Path)
    parser.add_argument(
        "outbox", type=Path, help="Shared export root, with a writable owner subfolder"
    )
    args = parser.parse_args()
    folder = args.outbox / args.owner
    manifest = json.loads(args.manifest.read_text(encoding="utf-8"))
    results = {"published": 0, "duplicate": 0}
    for item in manifest["exports"]:
        status = publish(folder, export(args.owner, item, manifest))
        results[status] += 1
    print(json.dumps(results))


if __name__ == "__main__":
    main()
