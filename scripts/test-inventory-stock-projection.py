"""Disposable PostgreSQL regression for the Inventory -> Catalog stock contract."""
import json
import os
from pathlib import Path
import secrets
import subprocess
import uuid

root = Path(__file__).resolve().parents[1]
name = "inventory-projection-test-" + uuid.uuid4().hex[:10]
env = dict(os.environ, POSTGRES_PASSWORD=secrets.token_urlsafe(24))

def run(args, **kwargs):
    return subprocess.run(args, check=True, env=env, **kwargs)

def sql(statement):
    return run(["docker", "exec", "-i", name, "psql", "-U", "postgres", "-v", "ON_ERROR_STOP=1", "-At"], input=statement, text=True, capture_output=True).stdout.strip()

def latest():
    return json.loads(sql("SELECT payload FROM outbox_events WHERE event_type='inventory.stock-changed.v1' ORDER BY (payload->>'inventory_version')::bigint DESC LIMIT 1"))

try:
    run(["docker", "run", "-d", "--name", name, "-e", "POSTGRES_PASSWORD", "postgres:16-alpine"], stdout=subprocess.DEVNULL)
    import time
    for _ in range(60):
        ready = subprocess.run(["docker", "exec", name, "pg_isready", "-U", "postgres"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        if ready.returncode == 0:
            # Postgres init has a temporary server; exec waits for the final server.
            try:
                sql("SELECT 1")
                break
            except subprocess.CalledProcessError:
                pass
        time.sleep(1)
    else:
        raise RuntimeError("test database did not start")
    migrations = root / "services/inventory-service/migrations"
    for file in ["000001_init_inventory_schema.up.sql", "000002_terminal_profile.up.sql"]:
        sql((migrations / file).read_text(encoding="utf-8"))
    sql("""INSERT INTO inventory_items(sku,physical_qty) VALUES('SKU-A',10);
    INSERT INTO batches(batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,status)
    VALUES('LOT-A','SKU-A',gen_random_uuid(),CURRENT_DATE-1,CURRENT_DATE+30,10,'ACTIVE');""")
    sql((migrations / "000003_stock_projection.up.sql").read_text(encoding="utf-8"))
    assert latest()["sellable_quantity"] == 10
    def change(statement, quantity):
        old = latest()["inventory_version"]
        sql(statement)
        event = latest()
        assert event["sellable_quantity"] == quantity, event
        assert event["inventory_version"] > old, event
        assert set(event) == {"sku_code", "sellable_quantity", "inventory_version", "source_updated_at"}, event
    change("BEGIN; UPDATE inventory_items SET reserved_qty=2 WHERE sku='SKU-A'; UPDATE batches SET reserved_qty=2 WHERE sku='SKU-A'; COMMIT;", 8)
    # Release restores availability; commit consumes physical stock without restoring it.
    change("BEGIN; UPDATE inventory_items SET reserved_qty=0 WHERE sku='SKU-A'; UPDATE batches SET reserved_qty=0 WHERE sku='SKU-A'; COMMIT;", 10)
    change("BEGIN; UPDATE inventory_items SET physical_qty=8 WHERE sku='SKU-A'; UPDATE batches SET physical_qty=8 WHERE sku='SKU-A'; COMMIT;", 8)
    before = latest()
    sql("BEGIN; UPDATE inventory_items SET reserved_qty=1 WHERE sku='SKU-A'; UPDATE batches SET reserved_qty=1 WHERE sku='SKU-A'; ROLLBACK;")
    assert latest() == before, "rollback leaked a snapshot/version"
    change("UPDATE batches SET status='QUARANTINE' WHERE sku='SKU-A'", 0)
    change("UPDATE batches SET status='ACTIVE' WHERE sku='SKU-A'", 8)
    change("UPDATE batches SET exp_date=CURRENT_DATE WHERE sku='SKU-A'", 0)
    change("UPDATE batches SET exp_date=CURRENT_DATE+30 WHERE sku='SKU-A'", 8)
    change("UPDATE inventory_items SET status='SUSPENDED' WHERE sku='SKU-A'", 0)
    change("UPDATE inventory_items SET status='ACTIVE' WHERE sku='SKU-A'", 8)
    change("SELECT inventory_emit_stock_snapshot('SKU-A')", 8)
    # Migration rollback and reapply must remain executable without losing stock.
    previous_version = latest()["inventory_version"]
    sql((migrations / "000003_stock_projection.down.sql").read_text(encoding="utf-8"))
    sql((migrations / "000003_stock_projection.up.sql").read_text(encoding="utf-8"))
    assert latest()["sellable_quantity"] == 8
    assert latest()["inventory_version"] > previous_version
    print("PASS: bootstrap, reserve, release, deduction, rollback, quarantine, expiry, item status, rebuild and migration round-trip")
finally:
    subprocess.run(["docker", "rm", "-f", name], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
