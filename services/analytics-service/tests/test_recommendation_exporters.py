import importlib.util
import json
import os
from dataclasses import replace
from datetime import UTC, datetime, timedelta
from pathlib import Path
from uuid import UUID, uuid4

import psycopg
import pytest
from psycopg import sql
from test_recommendation import CUSTOMERS, WAREHOUSE, config, grant, store  # noqa: F401

from analytics_service.recommendation.sources import read_snapshot

ROOT = Path(__file__).resolve().parents[1]
if not (ROOT / "services").exists():
    ROOT = Path(__file__).resolve().parents[3]


@pytest.fixture
def owners(monkeypatch):
    if os.getenv("RUN_ANALYTICS_INTEGRATION") != "1":
        pytest.skip("Requires PostgreSQL owner databases and psql")
    dsn = os.environ["ANALYTICS_PILOT_PG_DSN"]
    databases = {name: "rec_owner_" + uuid4().hex for name in ("catalog", "inventory", "order")}
    with psycopg.connect(dsn, autocommit=True) as admin:
        for database in databases.values():
            admin.execute(sql.SQL("CREATE DATABASE {}").format(sql.Identifier(database)))
    try:
        paths = {
            "catalog": sorted(
                (ROOT / "services/catalog-service/src/main/resources/db/migration").glob("V*.sql"),
                key=lambda p: int(p.name.split("__")[0][1:]),
            ),
            "inventory": sorted((ROOT / "services/inventory-service/migrations").glob("*.up.sql")),
            "order": sorted((ROOT / "services/order-service/migrations").glob("*.up.sql")),
        }
        for name, database in databases.items():
            with psycopg.connect(dsn, dbname=database) as conn:
                for path in paths[name]:
                    conn.execute(path.read_text(encoding="utf-8"))
        options = psycopg.conninfo.conninfo_to_dict(dsn)
        for field, env in (
            ("host", "PGHOST"),
            ("user", "PGUSER"),
            ("password", "PGPASSWORD"),
            ("port", "PGPORT"),
        ):
            if field in options:
                monkeypatch.setenv(env, options[field])
        monkeypatch.setenv("PGCLIENTENCODING", "UTF8")
        yield dsn, databases
    finally:
        with psycopg.connect(dsn, autocommit=True) as admin:
            for database in databases.values():
                admin.execute(
                    sql.SQL("DROP DATABASE {} WITH (FORCE)").format(sql.Identifier(database))
                )


def test_owner_queries_from_real_migrations_and_purchase_mapping(owners, store, monkeypatch):  # noqa: F811
    dsn, databases = owners
    store.config = replace(
        store.config,
        scope="d2c-export-test",
        export_root=store.config.export_root / "owner-published",
    )
    module_path = ROOT / "scripts/publish-analytics-recommendation.py"
    spec = importlib.util.spec_from_file_location("recommendation_publisher", module_path)
    publisher = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(publisher)
    scope = {
        "scopeId": store.config.scope,
        "channel": "D2C_WEB",
        "warehouseId": WAREHOUSE,
        "orderCohort": "order-core-1",
        "identityMappingVersion": "identity-core-v1",
    }
    category = str(UUID(int=101))
    with psycopg.connect(dsn, dbname=databases["catalog"]) as conn:
        conn.execute(
            "INSERT INTO categories(id,name,slug) VALUES(%s,'Sesame','sesame')", (category,)
        )
        for n in range(1, 7):
            product, variant = str(UUID(int=n)), str(UUID(int=n + 1000))
            conn.execute(
                """INSERT INTO products(id,category_id,name,slug,approval_status,created_by)
                         VALUES(%s,%s,%s,%s,'APPROVED','test')""",
                (product, category, f"Product {n}", f"product-{n}"),
            )
            conn.execute(
                """INSERT INTO product_variants(id,product_id,sku_code,variant_name,weight_value,weight_unit)
                         VALUES(%s,%s,%s,'Variant',0.5,'KILOGRAM')""",
                (variant, product, f"SKU-{n}"),
            )
            conn.execute(
                """INSERT INTO channel_prices(variant_id,channel,currency_code,amount_units,created_by)
                         VALUES(%s,'D2C_WEB','VND',95000,'test')""",
                (variant,),
            )
    with psycopg.connect(dsn, dbname=databases["inventory"]) as conn:
        for n in range(1, 7):
            conn.execute(
                "INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty) VALUES(%s,%s,20,3)",
                (f"SKU-{n}", WAREHOUSE),
            )
            conn.execute(
                """INSERT INTO batches(batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
                         VALUES(%s,%s,%s,CURRENT_DATE-1,CURRENT_DATE+30,20,3,'ACTIVE')""",
                (f"LOT-{n}", f"SKU-{n}", uuid4()),
            )
        conn.execute("UPDATE batches SET status='QUARANTINE' WHERE sku='SKU-6'")
    principals = CUSTOMERS[:3]
    subjects = [grant(store, p) for p in principals]
    completed = datetime.now(UTC) - timedelta(minutes=5)
    with psycopg.connect(dsn, dbname=databases["order"]) as conn:
        for n in range(20):
            order, operation, principal = uuid4(), uuid4(), principals[n % 3]
            owner_scope = "customer:" + principal
            # Relational fixture uses the real migration constraints. No payment action is run here.
            conn.execute(
                """INSERT INTO checkout_operations(id,scope,principal_id,planned_order_id,status,input_cipher,request_hash)
                         VALUES(%s,%s,%s,%s,'SUCCEEDED','x','fixture')""",
                (operation, owner_scope, principal, order),
            )
            conn.execute(
                """INSERT INTO orders(id,operation_id,order_code,scope,principal_id,customer_id,warehouse_id,
                method,status,payment_status,stock_status,subtotal,shipping_fee,final_amount,snapshot_cipher,
                reservation_id,cohort,owner_kind,snapshot_id,snapshot_schema_version,snapshot_hash,core_state_cipher)
                VALUES(%s,%s,%s,%s,%s,%s,%s,'COD','COMPLETED','UNPAID','COMMITTED',95000,0,95000,'x',
                'fixture','order-core-1','CUSTOMER',%s,'1.0',%s,'x')""",
                (
                    order,
                    operation,
                    f"REC-{n}",
                    owner_scope,
                    principal,
                    uuid4(),
                    WAREHOUSE,
                    uuid4(),
                    "0" * 64,
                ),
            )
            skus = ["SKU-4", "SKU-5"] if n < 5 else ["SKU-2"]
            for sku in skus:
                conn.execute(
                    "INSERT INTO order_items VALUES(%s,%s,1,95000,95000,'Product')", (order, sku)
                )
            conn.execute(
                "INSERT INTO order_milestones VALUES(%s,'COMPLETED','fixture',%s)",
                (order, completed),
            )
    results = {}
    for dataset, owner in (
        ("catalog", "catalog"),
        ("sellability", "inventory"),
        ("completed_baskets", "order"),
    ):
        monkeypatch.setenv("PGDATABASE", databases[owner])
        published = publisher.export(dataset, scope, store.config.export_root)
        owner_name = {
            "catalog": "catalog-service",
            "inventory": "inventory-service",
            "order": "order-service",
        }[owner]
        path = store.config.export_root / owner_name / published["snapshotId"]
        manifest, records = read_snapshot(path, store.config, datetime.now(UTC))
        assert manifest.recordCount == len(records)
        results[dataset] = records
        store.import_snapshot(manifest, records, datetime.now(UTC))
    assert results["catalog"][0]["weightGrams"] == 500
    assert results["sellability"][-1]["sellableUnits"] == 0
    store.build(datetime.now(UTC))
    result = store.recommend("cart", {"skuCodes": ["SKU-4"]}, 6, None, datetime.now(UTC))
    assert result["items"][0]["skuCode"] == "SKU-5"
    assert result["strategy"] == "BOUGHT_TOGETHER"
    result = store.recommend("home", {}, 6, subjects[0], datetime.now(UTC))
    assert result["strategy"] == "PERSONALIZED"
    with store.connect() as conn:
        raw = json.dumps(store.sources(conn)["completed_baskets"]["records"])
    assert all(p not in raw for p in principals)
    # A second export allocates a durable new revision, including after a fresh helper import.
    monkeypatch.setenv("PGDATABASE", databases["catalog"])
    assert publisher.export("catalog", scope, store.config.export_root)["revision"] == 2
    # A reconciliation failure never publishes an inflated sellable snapshot.
    with psycopg.connect(dsn, dbname=databases["inventory"]) as conn:
        conn.execute("UPDATE inventory_items SET reserved_qty=4 WHERE sku='SKU-1'")
    monkeypatch.setenv("PGDATABASE", databases["inventory"])
    with pytest.raises(ValueError, match="reconciliation"):
        publisher.export("sellability", scope, store.config.export_root)
