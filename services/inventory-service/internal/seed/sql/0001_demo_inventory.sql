-- Demo-only job. Never included in schema migrations.
SELECT pg_advisory_xact_lock(2026112041);
CREATE TABLE IF NOT EXISTS inventory_demo_seed_000007 (
 sku text PRIMARY KEY, batch_id uuid NOT NULL UNIQUE,
 initial_quantity integer NOT NULL CHECK(initial_quantity>0),
 item_created boolean NOT NULL DEFAULT false, batch_created boolean NOT NULL DEFAULT false,
 initial_inventory_version bigint
);
-- Adopt markers from the unreleased startup seed without claiming any new rows.
ALTER TABLE inventory_demo_seed_000007 ADD COLUMN IF NOT EXISTS initial_inventory_version bigint;
DO $$
DECLARE r record;
BEGIN
 FOR r IN SELECT * FROM (VALUES
('OM-MX-01','9b200005-2026-4020-8020-000000000001'::uuid,80),
('OM-MX-02','9b200005-2026-4020-8020-000000000002'::uuid,80),
('OM-MX-03','9b200005-2026-4020-8020-000000000003'::uuid,80),
('OM-MX-04','9b200005-2026-4020-8020-000000000004'::uuid,80),
('OM-MX-05','9b200005-2026-4020-8020-000000000005'::uuid,80),
('OM-MX-06','9b200005-2026-4020-8020-000000000006'::uuid,80),
('OM-MX-07','9b200005-2026-4020-8020-000000000007'::uuid,80),
('OM-MX-08','9b200005-2026-4020-8020-000000000008'::uuid,80),
('OM-MU-01','9b200005-2026-4020-8020-000000000009'::uuid,80),
('OM-MU-02','9b200005-2026-4020-8020-000000000010'::uuid,20),
('OM-MU-03','9b200005-2026-4020-8020-000000000011'::uuid,80),
('OM-MU-04','9b200005-2026-4020-8020-000000000012'::uuid,80),
('OM-MU-05','9b200005-2026-4020-8020-000000000013'::uuid,80),
('OM-MU-06','9b200005-2026-4020-8020-000000000014'::uuid,20),
('OM-MU-07','9b200005-2026-4020-8020-000000000015'::uuid,80),
('OM-TR-01','9b200005-2026-4020-8020-000000000016'::uuid,80),
('OM-TR-02','9b200005-2026-4020-8020-000000000017'::uuid,20),
('OM-TR-03','9b200005-2026-4020-8020-000000000018'::uuid,80),
('OM-TR-04','9b200005-2026-4020-8020-000000000019'::uuid,80),
('OM-TR-05','9b200005-2026-4020-8020-000000000020'::uuid,80),
('OM-TR-06','9b200005-2026-4020-8020-000000000021'::uuid,80),
('OM-TR-07','9b200005-2026-4020-8020-000000000022'::uuid,80),
('OM-BA-01','9b200005-2026-4020-8020-000000000023'::uuid,80),
('OM-BA-02','9b200005-2026-4020-8020-000000000024'::uuid,80),
('OM-BA-03','9b200005-2026-4020-8020-000000000025'::uuid,80),
('OM-BA-04','9b200005-2026-4020-8020-000000000026'::uuid,80),
('OM-BA-05','9b200005-2026-4020-8020-000000000027'::uuid,80),
('OM-KS-01','9b200005-2026-4020-8020-000000000028'::uuid,80),
('OM-KS-02','9b200005-2026-4020-8020-000000000029'::uuid,80),
('OM-KS-03','9b200005-2026-4020-8020-000000000030'::uuid,80),
('OM-KS-04','9b200005-2026-4020-8020-000000000031'::uuid,20),
('OM-KS-05','9b200005-2026-4020-8020-000000000032'::uuid,80),
('OM-KS-06','9b200005-2026-4020-8020-000000000033'::uuid,20),
('OM-HQ-01','9b200005-2026-4020-8020-000000000034'::uuid,20),
('OM-HQ-02','9b200005-2026-4020-8020-000000000035'::uuid,20),
('OM-HQ-03','9b200005-2026-4020-8020-000000000036'::uuid,20),
('OM-HQ-04','9b200005-2026-4020-8020-000000000037'::uuid,20),
('OM-DS-01','9b200005-2026-4020-8020-000000000038'::uuid,20),
('OM-DS-02','9b200005-2026-4020-8020-000000000039'::uuid,80),
('OM-DS-03','9b200005-2026-4020-8020-000000000040'::uuid,80)
 ) AS demo(sku,batch_id,quantity) LOOP
  IF EXISTS(SELECT 1 FROM inventory_items WHERE sku=r.sku) OR EXISTS(SELECT 1 FROM batches WHERE id=r.batch_id) THEN
   RAISE NOTICE 'Skipping existing demo SKU %',r.sku;
   CONTINUE;
  END IF;
  INSERT INTO inventory_items(sku,warehouse_id,physical_qty,status,inventory_version)
  VALUES(r.sku,'9b200006-2026-4020-8020-000000000001',r.quantity,'ACTIVE',COALESCE((SELECT max((payload->>'inventory_version')::bigint) FROM outbox_events WHERE aggregate_id=r.sku AND event_type='inventory.stock-changed.v1'),0));
  INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,status)
  VALUES(r.batch_id,'LOT-PBL6-'||r.sku||'-01',r.sku,'9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,r.quantity,'ACTIVE');
  INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity,item_created,batch_created)
  VALUES(r.sku,r.batch_id,r.quantity,true,true);
 END LOOP;
END $$;
-- Flush deferred snapshots before recording the post-seed version.
SET CONSTRAINTS ALL IMMEDIATE;
UPDATE inventory_demo_seed_000007 s SET initial_inventory_version=i.inventory_version
FROM inventory_items i WHERE i.sku=s.sku AND s.initial_inventory_version IS NULL;
