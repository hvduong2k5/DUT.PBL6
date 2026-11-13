import json
import os
import subprocess
import sys
import time
from datetime import UTC, datetime
from datetime import time as wall_time
from urllib.parse import urlparse
from uuid import uuid4

import httpx
import psycopg
import pytest
from psycopg import sql
from test_analytics import API, contract, headers

from analytics_service.events import TZ
from analytics_service.report import DAY, days

ADMIN_SESSION = "f47ac10b-58cc-4372-a567-0e02b2c3d479"
RESTRICTED_SESSION = "f47ac10b-58cc-4372-a567-0e02b2c3d478"
CUSTOMER_SESSION = "f47ac10b-58cc-4372-a567-0e02b2c3d477"
WAREHOUSE = "warehouse-supply-pilot"
INVENTORY_WAREHOUSE = "a7277eb6-0fd2-4850-b640-169bc446387e"


@pytest.mark.integration
@pytest.mark.skipif(
    os.getenv("RUN_ANALYTICS_INTEGRATION") != "1", reason="Requires Compose test stack"
)
def test_two_sku_pilot_from_owner_databases(tmp_path):
    admin_dsn = os.environ["ANALYTICS_PILOT_PG_DSN"]
    owner_databases = {owner: f"{owner}_{uuid4().hex}" for owner in ("order", "inventory")}
    with psycopg.connect(admin_dsn, autocommit=True) as admin:
        for name in owner_databases.values():
            admin.execute(sql.SQL("CREATE DATABASE {}").format(sql.Identifier(name)))

    now = datetime.now(UTC)
    today = now.astimezone(TZ).date()
    start = today - 14 * DAY
    order_rows = []
    inventory_rows = []
    with psycopg.connect(admin_dsn, dbname=owner_databases["order"]) as owner:
        owner.execute(
            """CREATE TABLE orders(id int PRIMARY KEY, warehouse_id text, cohort text);
               CREATE TABLE order_items(order_id int, sku text, quantity int);
               CREATE TABLE order_milestones(order_id int, kind text, occurred_at timestamptz);"""
        )
        for number, day in enumerate(days(start, today), 1):
            sold_at = datetime.combine(day, wall_time(12), TZ)
            owner.execute("INSERT INTO orders VALUES (%s, %s, 'order-core-1')", (number, WAREHOUSE))
            owner.execute(
                "INSERT INTO order_milestones VALUES (%s,'COMPLETED',%s)", (number, sold_at)
            )
            for sku, sold in (("PILOT-FAST", 10), ("PILOT-SLOW", 2)):
                owner.execute("INSERT INTO order_items VALUES (%s,%s,%s)", (number, sku, sold))
                daily = {
                    "skuCode": sku,
                    "businessDate": str(day),
                    "revision": 1,
                    "sourceAsOf": now.isoformat(),
                    "evidenceRef": f"local-fixture/{day}/{sku}",
                }
                order_rows.append(
                    {
                        **daily,
                        "kind": "order_day",
                        "expectedSoldUnits": sold,
                        "returnedUnits": int(sku == "PILOT-FAST" and day == start),
                    }
                )
                inventory_rows.append(
                    {
                        **daily,
                        "kind": "inventory_day",
                        "stockoutMinutes": 60 if sku == "PILOT-FAST" and day == start else 0,
                    }
                )

    with psycopg.connect(admin_dsn, dbname=owner_databases["inventory"]) as owner:
        owner.execute(
            """CREATE TABLE inventory_items(sku text PRIMARY KEY, warehouse_id uuid);
               CREATE TABLE batches(sku text, batch_code text, exp_date date,
                                    physical_qty int, reserved_qty int, status text);"""
        )
        for sku in ("PILOT-FAST", "PILOT-SLOW"):
            owner.execute("INSERT INTO inventory_items VALUES (%s,%s)", (sku, INVENTORY_WAREHOUSE))
            inventory_rows.append(
                {
                    "kind": "inventory_batches",
                    "skuCode": sku,
                    "businessDate": str(today),
                    "revision": 1,
                    "evidenceRef": "local-fixture/inventory-snapshot",
                }
            )
        for sku, batch, expires, physical, reserved, status in (
            ("PILOT-FAST", "FAST-A", 3, 30, 5, "NEAR_EXPIRY"),
            ("PILOT-FAST", "FAST-B", 5, 50, 0, "ACTIVE"),
            ("PILOT-FAST", "HELD", 4, 80, 0, "QUARANTINED"),
            ("PILOT-FAST", "EXPIRED", 0, 50, 0, "ACTIVE"),
            ("PILOT-SLOW", "SLOW-A", 5, 20, 0, "ACTIVE"),
        ):
            owner.execute(
                "INSERT INTO batches VALUES (%s,%s,%s,%s,%s,%s)",
                (sku, batch, today + expires * DAY, physical, reserved, status),
            )

    def publish(owner, rows, succeeds=True):
        parsed = urlparse(admin_dsn)
        source = "order-service" if owner == "order" else "inventory-service"
        manifest = tmp_path / f"{owner}.json"
        manifest.write_text(
            json.dumps(
                {
                    "warehouseId": WAREHOUSE,
                    "sourceWarehouseId": WAREHOUSE if owner == "order" else INVENTORY_WAREHOUSE,
                    "approvedBy": "local-fixture-owner",
                    "exports": rows,
                }
            ),
            encoding="utf-8",
        )
        result = subprocess.run(
            [
                sys.executable,
                "/app/scripts/publish-analytics-supply.py",
                source,
                str(manifest),
                str(inbox),
            ],
            env={
                **os.environ,
                "PGHOST": parsed.hostname,
                "PGUSER": parsed.username,
                "PGPASSWORD": parsed.password,
                "PGDATABASE": owner_databases[owner],
            },
            capture_output=True,
            text=True,
            timeout=30,
        )
        if not succeeds:
            assert result.returncode != 0
            return result.stderr
        assert result.returncode == 0, result.stderr
        return json.loads(result.stdout)

    inbox = tmp_path / "inbox"
    (inbox / "order-service").mkdir(parents=True)
    (inbox / "inventory-service").mkdir()
    (inbox / "inventory-service" / "bad.json").write_text("{}", encoding="utf-8")
    query = {"warehouseId": WAREHOUSE, "from": str(start), "to": str(today), "horizonDays": 5}
    url = f"{API}/supply/workbench"
    assert httpx.get(url, params=query).status_code == 401
    auth = headers("admin")

    def until_ready(predicate):
        with httpx.Client() as client:
            for _ in range(60):
                response = client.get(url, params=query, headers=auth)
                assert response.status_code == 200, response.text
                if predicate(response.json()):
                    return response
                time.sleep(0.25)
        pytest.fail("Supply polling did not import the expected owner revision")

    worker = subprocess.Popen(
        [sys.executable, "-m", "analytics_service", "supply-sync"],
        env={
            **os.environ,
            "ANALYTICS_SUPPLY_INBOX": str(inbox),
            "ANALYTICS_SUPPLY_SYNC_INTERVAL_SECONDS": "1",
        },
    )
    try:
        assert publish("order", order_rows) == {"published": 28, "duplicate": 0}
        assert publish("inventory", inventory_rows) == {"published": 30, "duplicate": 0}
        assert publish("order", order_rows[:1]) == {"published": 0, "duplicate": 1}
        assert publish("inventory", inventory_rows[-2:]) == {"published": 0, "duplicate": 2}
        assert "does not reconcile" in publish(
            "order",
            [{**order_rows[0], "expectedSoldUnits": 9}],
            succeeds=False,
        )
        workbench = until_ready(
            lambda result: len(result["ranking"]) == 2 and result["batchCount"] == 3
        )
        contract("SupplyWorkbench", workbench)
        data = workbench.json()
        assert [(r["skuCode"], r["netUnits"], r["unitsPerDay"]) for r in data["ranking"]] == [
            ("PILOT-FAST", 139, "9.93"),
            ("PILOT-SLOW", 28, "2.00"),
        ]
        assert all(
            r["salesCoveredDays"] == r["inventoryCoveredDays"] == 14 for r in data["ranking"]
        )
        assert data["ranking"][0]["stockoutDays"] == 1
        assert [
            (r["batchCode"], r["sellableUnits"], r["unitsAtRisk"]) for r in data["batches"]
        ] == [
            ("FAST-A", 25, 5),
            ("FAST-B", 50, 30),
            ("SLOW-A", 20, 12),
        ]
        with httpx.Client() as client:
            for credential, status in (
                (ADMIN_SESSION, 200),
                (RESTRICTED_SESSION, 403),
                (CUSTOMER_SESSION, 403),
            ):
                response = client.get(
                    url, params=query, headers={"Cookie": f"OMAMA_SESSION={credential}"}
                )
                assert response.status_code == status, response.text
            velocity = client.get(
                f"{API}/supply/velocity", params={**query, "skuCode": "PILOT-FAST"}, headers=auth
            ).json()
            assert (velocity["soldUnits"], velocity["returnedUnits"]) == (140, 1)
            metrics = client.get("http://127.0.0.1:8014/metrics").text
            assert (
                'analytics_supply_files_total{result="rejected",source="inventory-service"} 1.0'
                in metrics
            )
            assert (
                'analytics_supply_source_as_of_seconds{kind="inventory_batches",source="inventory-service"}'
                in metrics
            )

        correction = {**order_rows[0], "revision": 2, "returnedUnits": 2}
        assert publish("order", [correction]) == {"published": 1, "duplicate": 0}
        updated = until_ready(lambda result: result["ranking"][0]["netUnits"] == 138)
        assert updated.json()["ranking"][0]["unitsPerDay"] == "9.86"
    finally:
        worker.terminate()
        worker.wait(timeout=10)
