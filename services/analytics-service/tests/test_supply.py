import json
import os
import subprocess
import sys
import traceback
from datetime import UTC, datetime, timedelta
from pathlib import Path

import httpx
import pytest
from jsonschema import Draft202012Validator, FormatChecker
from test_analytics import API, contract, headers

from analytics_service import db, supply
from analytics_service.config import Settings
from analytics_service.events import TZ

SCHEMA = Path("/app/contracts/analytics-supply-source.schema.json")
if not SCHEMA.exists():
    SCHEMA = (
        Path(__file__).resolve().parents[3]
        / "docs/03_api_specs/analytics-supply-source.v1.schema.json"
    )

SKU = "SKU-ANALYTICS"
WAREHOUSE = "warehouse-hue"


def manifest(kind, day, source_as_of, **data):
    return {
        "schemaVersion": "1",
        "kind": kind,
        "source": "order-service" if kind == "order_day" else "inventory-service",
        "skuCode": SKU,
        "warehouseId": WAREHOUSE,
        "businessDate": day.isoformat(),
        "revision": 1,
        "sourceAsOf": source_as_of.isoformat(),
        "approvedBy": "integration-owner",
        "evidenceRef": "sandbox/closed-day-v1",
        **data,
    }


def encoded(payload):
    return json.dumps(payload).encode()


def test_export_schema_matches_runtime():
    declared = json.loads(SCHEMA.read_text(encoding="utf-8"))
    Draft202012Validator.check_schema(declared)
    assert declared.pop("$schema") == "https://json-schema.org/draft/2020-12/schema"
    assert declared == supply.EXPORT.json_schema()


@pytest.mark.parametrize(
    "changes",
    [
        {"source": "inventory-service"},
        {"soldUnits": True},
        {"businessDate": "2026-11-06"},
        {"customerEmail": "unexpected"},
        {"sourceAsOf": "2026-11-07T03:00:00Z"},
        {"sourceAsOf": "2026-11-06T03:00:00"},
    ],
)
def test_rejects_unapproved_shape_and_open_day(changes):
    now = datetime(2026, 11, 6, 3, tzinfo=UTC)
    payload = manifest(
        "order_day", now.date() - timedelta(days=1), now, soldUnits=10, returnedUnits=0
    )
    with pytest.raises(db.Conflict):
        supply.parse_export(encoded({**payload, **changes}), now)


def test_fefo_conserves_fractional_demand_and_respects_expiry_boundary():
    today = datetime(2026, 11, 6, 3, tzinfo=UTC).date()
    batches = [
        {"batchCode": "B", "expiresOn": str(today + 3 * supply.DAY), "sellableUnits": 3},
        {"batchCode": "A", "expiresOn": str(today + 3 * supply.DAY), "sellableUnits": 2},
        {"batchCode": "C", "expiresOn": str(today + 4 * supply.DAY), "sellableUnits": 2},
    ]
    actual = supply.allocate_fefo(batches, today, today + 4 * supply.DAY, 5, 3)
    assert [(batch["batchCode"], batch["unitsAtRisk"]) for batch in actual] == [
        ("A", 0),
        ("B", 2),
        ("C", 0),
    ]


def test_invalid_export_error_omits_raw_input():
    now = datetime(2026, 11, 6, 3, tzinfo=UTC)
    payload = manifest("order_day", now.date() - supply.DAY, now, soldUnits=10, returnedUnits=0)
    payload["customerEmail"] = "test@example.com"
    with pytest.raises(db.Conflict) as error:
        supply.parse_export(encoded(payload), now)
    assert "test@example.com" not in "".join(traceback.format_exception(error.value))


def test_expiry_reports_stale_snapshot():
    now = datetime(2026, 11, 6, 3, tzinfo=UTC)
    snapshot = {
        "business_date": now.date() - supply.DAY,
        "source_as_of": now - 2 * supply.DAY,
        "revision": 1,
        "payload": {"batches": []},
    }
    assert (
        supply.expiry_risk(snapshot, {}, 30, now, supply.DAY)["reasonCode"]
        == "BATCH_SNAPSHOT_STALE"
    )


@pytest.mark.integration
@pytest.mark.skipif(
    os.getenv("RUN_ANALYTICS_INTEGRATION") != "1", reason="Requires Compose test stack"
)
def test_supply_import_and_http(tmp_path):
    settings = Settings.load()
    now = datetime.now(UTC)
    today = now.astimezone(TZ).date()
    start = today - 2 * supply.DAY
    scope = {"skuCode": SKU, "warehouseId": WAREHOUSE}
    period = {**scope, "from": str(start), "to": str(today), "comparison": "previous_period"}
    expiry_params = {**scope, "horizonDays": 5}
    auth = headers("admin")
    schema = json.loads(SCHEMA.read_text(encoding="utf-8"))
    source_contract = Draft202012Validator(schema, format_checker=FormatChecker())

    def request(path, params, schema_name):
        response = httpx.get(f"{API}/supply/{path}", headers=auth, params=params)
        assert response.status_code == 200, response.text
        contract(schema_name, response)
        return response.json()

    def cli(payload, succeeds=True):
        source_contract.validate(payload)
        path = tmp_path / "supply.json"
        path.write_bytes(encoded(payload))
        result = subprocess.run(
            [sys.executable, "-m", "analytics_service", "import-supply", str(path)],
            capture_output=True,
            text=True,
            timeout=15,
        )
        if not succeeds:
            assert result.returncode != 0
            return result.stderr
        assert result.returncode == 0, result.stderr
        return json.loads(result.stdout)["status"]

    missing = request("velocity", period, "SupplyVelocity")
    assert missing["status"] == "UNAVAILABLE" and missing["unitsPerDay"] is None
    assert missing["series"][0]["stockoutMinutes"] is None
    assert (
        request("expiry-risk", expiry_params, "SupplyExpiryRisk")["reasonCode"]
        == "BATCH_SNAPSHOT_MISSING"
    )

    order = manifest("order_day", start, now, soldUnits=10, returnedUnits=2)
    inventory = manifest("inventory_day", start, now, stockoutMinutes=120)
    lots = manifest(
        "inventory_batches",
        today,
        now,
        batches=[
            {"batchCode": "B", "expiresOn": str(today + 5 * supply.DAY), "sellableUnits": 50},
            {"batchCode": "A", "expiresOn": str(today + 3 * supply.DAY), "sellableUnits": 25},
        ],
    )
    assert cli(order) == "imported"
    partial = request("velocity", period, "SupplyVelocity")
    assert partial["status"] == "PARTIAL" and partial["unitsPerDay"] == "8.00"
    assert partial["stockoutDays"] is None and partial["netUnitsChange"] is None
    assert cli(inventory) == "imported"
    assert cli(lots) == "imported"
    assert cli({**lots, "batches": list(reversed(lots["batches"]))}) == "duplicate"
    assert (
        request("expiry-risk", expiry_params, "SupplyExpiryRisk")["reasonCode"]
        == "BASELINE_COVERAGE_GAP"
    )

    with db.connect(settings) as conn:
        for day in supply.days(today - supply.BASELINE_DAYS * supply.DAY, today):
            if day == start:
                continue
            for payload in (
                manifest("order_day", day, now, soldUnits=10, returnedUnits=0),
                manifest("inventory_day", day, now, stockoutMinutes=0),
            ):
                supply.import_export(
                    conn, settings, supply.parse_export(encoded(payload), datetime.now(UTC))
                )

    result = request("velocity", period, "SupplyVelocity")
    assert result["status"] == "READY"
    assert (result["netUnits"], result["unitsPerDay"], result["stockoutDays"]) == (18, "9.00", 1)
    assert result["netUnitsChange"] == -2 and result["series"][0]["orderRevision"] == 1
    assert (
        cli({**order, "soldUnits": 11}, succeeds=False).count(
            "content changed at the same revision"
        )
        >= 1
    )
    assert cli({**order, "soldUnits": 11, "revision": 2}) == "imported"
    assert cli(order) == "duplicate"
    corrected = request("velocity", period, "SupplyVelocity")
    assert corrected["netUnits"] == 19 and corrected["series"][0]["orderRevision"] == 2

    expiry = request("expiry-risk", expiry_params, "SupplyExpiryRisk")
    assert (expiry["status"], expiry["inStockDays"], expiry["demandUnitsPerDay"]) == (
        "READY",
        13,
        "10.00",
    )
    assert {lot["batchCode"]: lot["unitsAtRisk"] for lot in expiry["batches"]} == {"A": 5, "B": 30}

    different = request("velocity", {**period, "warehouseId": "other"}, "SupplyVelocity")
    assert different["status"] == "UNAVAILABLE"
    denied = httpx.get(f"{API}/supply/velocity", params=period, headers=headers("customer"))
    assert denied.status_code == 403
    invalid = httpx.get(
        f"{API}/supply/expiry-risk", headers=auth, params={**scope, "horizonDays": 91}
    )
    assert invalid.status_code == 400
