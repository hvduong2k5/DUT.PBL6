-- 40 demo finished-goods SKUs. Images and Catalog data are supplied separately.
-- Stock quantities and dates are synthetic demo values.
-- Applied transactionally by the migration tool or the Inventory startup seed.
SELECT pg_advisory_xact_lock(2026112041);
CREATE TABLE IF NOT EXISTS inventory_demo_seed_000007 (
    sku text PRIMARY KEY,
    batch_id uuid NOT NULL UNIQUE,
    initial_quantity integer NOT NULL CHECK (initial_quantity > 0),
    item_created boolean NOT NULL DEFAULT false,
    batch_created boolean NOT NULL DEFAULT false
);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000001' AND sku<>'OM-MX-01') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-MX-01') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000001' AND sku='OM-MX-01') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-MX-01' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-MX-01';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-MX-01','9b200005-2026-4020-8020-000000000001',80) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-MX-01','9b200006-2026-4020-8020-000000000001',80,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000001','LOT-PBL6-OM-MX-01-01','OM-MX-01','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,80,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000002' AND sku<>'OM-MX-02') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-MX-02') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000002' AND sku='OM-MX-02') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-MX-02' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-MX-02';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-MX-02','9b200005-2026-4020-8020-000000000002',80) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-MX-02','9b200006-2026-4020-8020-000000000001',80,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000002','LOT-PBL6-OM-MX-02-01','OM-MX-02','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,80,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000003' AND sku<>'OM-MX-03') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-MX-03') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000003' AND sku='OM-MX-03') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-MX-03' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-MX-03';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-MX-03','9b200005-2026-4020-8020-000000000003',80) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-MX-03','9b200006-2026-4020-8020-000000000001',80,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000003','LOT-PBL6-OM-MX-03-01','OM-MX-03','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,80,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000004' AND sku<>'OM-MX-04') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-MX-04') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000004' AND sku='OM-MX-04') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-MX-04' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-MX-04';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-MX-04','9b200005-2026-4020-8020-000000000004',80) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-MX-04','9b200006-2026-4020-8020-000000000001',80,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000004','LOT-PBL6-OM-MX-04-01','OM-MX-04','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,80,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000005' AND sku<>'OM-MX-05') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-MX-05') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000005' AND sku='OM-MX-05') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-MX-05' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-MX-05';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-MX-05','9b200005-2026-4020-8020-000000000005',80) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-MX-05','9b200006-2026-4020-8020-000000000001',80,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000005','LOT-PBL6-OM-MX-05-01','OM-MX-05','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,80,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000006' AND sku<>'OM-MX-06') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-MX-06') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000006' AND sku='OM-MX-06') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-MX-06' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-MX-06';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-MX-06','9b200005-2026-4020-8020-000000000006',80) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-MX-06','9b200006-2026-4020-8020-000000000001',80,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000006','LOT-PBL6-OM-MX-06-01','OM-MX-06','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,80,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000007' AND sku<>'OM-MX-07') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-MX-07') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000007' AND sku='OM-MX-07') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-MX-07' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-MX-07';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-MX-07','9b200005-2026-4020-8020-000000000007',80) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-MX-07','9b200006-2026-4020-8020-000000000001',80,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000007','LOT-PBL6-OM-MX-07-01','OM-MX-07','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,80,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000008' AND sku<>'OM-MX-08') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-MX-08') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000008' AND sku='OM-MX-08') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-MX-08' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-MX-08';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-MX-08','9b200005-2026-4020-8020-000000000008',80) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-MX-08','9b200006-2026-4020-8020-000000000001',80,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000008','LOT-PBL6-OM-MX-08-01','OM-MX-08','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,80,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000009' AND sku<>'OM-MU-01') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-MU-01') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000009' AND sku='OM-MU-01') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-MU-01' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-MU-01';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-MU-01','9b200005-2026-4020-8020-000000000009',80) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-MU-01','9b200006-2026-4020-8020-000000000001',80,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000009','LOT-PBL6-OM-MU-01-01','OM-MU-01','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,80,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000010' AND sku<>'OM-MU-02') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-MU-02') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000010' AND sku='OM-MU-02') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-MU-02' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-MU-02';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-MU-02','9b200005-2026-4020-8020-000000000010',20) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-MU-02','9b200006-2026-4020-8020-000000000001',20,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000010','LOT-PBL6-OM-MU-02-01','OM-MU-02','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,20,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000011' AND sku<>'OM-MU-03') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-MU-03') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000011' AND sku='OM-MU-03') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-MU-03' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-MU-03';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-MU-03','9b200005-2026-4020-8020-000000000011',80) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-MU-03','9b200006-2026-4020-8020-000000000001',80,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000011','LOT-PBL6-OM-MU-03-01','OM-MU-03','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,80,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000012' AND sku<>'OM-MU-04') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-MU-04') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000012' AND sku='OM-MU-04') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-MU-04' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-MU-04';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-MU-04','9b200005-2026-4020-8020-000000000012',80) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-MU-04','9b200006-2026-4020-8020-000000000001',80,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000012','LOT-PBL6-OM-MU-04-01','OM-MU-04','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,80,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000013' AND sku<>'OM-MU-05') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-MU-05') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000013' AND sku='OM-MU-05') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-MU-05' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-MU-05';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-MU-05','9b200005-2026-4020-8020-000000000013',80) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-MU-05','9b200006-2026-4020-8020-000000000001',80,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000013','LOT-PBL6-OM-MU-05-01','OM-MU-05','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,80,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000014' AND sku<>'OM-MU-06') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-MU-06') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000014' AND sku='OM-MU-06') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-MU-06' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-MU-06';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-MU-06','9b200005-2026-4020-8020-000000000014',20) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-MU-06','9b200006-2026-4020-8020-000000000001',20,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000014','LOT-PBL6-OM-MU-06-01','OM-MU-06','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,20,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000015' AND sku<>'OM-MU-07') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-MU-07') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000015' AND sku='OM-MU-07') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-MU-07' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-MU-07';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-MU-07','9b200005-2026-4020-8020-000000000015',80) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-MU-07','9b200006-2026-4020-8020-000000000001',80,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000015','LOT-PBL6-OM-MU-07-01','OM-MU-07','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,80,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000016' AND sku<>'OM-TR-01') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-TR-01') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000016' AND sku='OM-TR-01') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-TR-01' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-TR-01';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-TR-01','9b200005-2026-4020-8020-000000000016',80) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-TR-01','9b200006-2026-4020-8020-000000000001',80,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000016','LOT-PBL6-OM-TR-01-01','OM-TR-01','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,80,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000017' AND sku<>'OM-TR-02') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-TR-02') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000017' AND sku='OM-TR-02') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-TR-02' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-TR-02';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-TR-02','9b200005-2026-4020-8020-000000000017',20) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-TR-02','9b200006-2026-4020-8020-000000000001',20,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000017','LOT-PBL6-OM-TR-02-01','OM-TR-02','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,20,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000018' AND sku<>'OM-TR-03') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-TR-03') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000018' AND sku='OM-TR-03') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-TR-03' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-TR-03';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-TR-03','9b200005-2026-4020-8020-000000000018',80) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-TR-03','9b200006-2026-4020-8020-000000000001',80,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000018','LOT-PBL6-OM-TR-03-01','OM-TR-03','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,80,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000019' AND sku<>'OM-TR-04') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-TR-04') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000019' AND sku='OM-TR-04') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-TR-04' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-TR-04';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-TR-04','9b200005-2026-4020-8020-000000000019',80) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-TR-04','9b200006-2026-4020-8020-000000000001',80,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000019','LOT-PBL6-OM-TR-04-01','OM-TR-04','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,80,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000020' AND sku<>'OM-TR-05') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-TR-05') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000020' AND sku='OM-TR-05') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-TR-05' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-TR-05';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-TR-05','9b200005-2026-4020-8020-000000000020',80) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-TR-05','9b200006-2026-4020-8020-000000000001',80,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000020','LOT-PBL6-OM-TR-05-01','OM-TR-05','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,80,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000021' AND sku<>'OM-TR-06') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-TR-06') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000021' AND sku='OM-TR-06') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-TR-06' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-TR-06';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-TR-06','9b200005-2026-4020-8020-000000000021',80) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-TR-06','9b200006-2026-4020-8020-000000000001',80,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000021','LOT-PBL6-OM-TR-06-01','OM-TR-06','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,80,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000022' AND sku<>'OM-TR-07') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-TR-07') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000022' AND sku='OM-TR-07') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-TR-07' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-TR-07';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-TR-07','9b200005-2026-4020-8020-000000000022',80) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-TR-07','9b200006-2026-4020-8020-000000000001',80,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000022','LOT-PBL6-OM-TR-07-01','OM-TR-07','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,80,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000023' AND sku<>'OM-BA-01') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-BA-01') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000023' AND sku='OM-BA-01') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-BA-01' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-BA-01';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-BA-01','9b200005-2026-4020-8020-000000000023',80) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-BA-01','9b200006-2026-4020-8020-000000000001',80,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000023','LOT-PBL6-OM-BA-01-01','OM-BA-01','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,80,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000024' AND sku<>'OM-BA-02') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-BA-02') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000024' AND sku='OM-BA-02') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-BA-02' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-BA-02';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-BA-02','9b200005-2026-4020-8020-000000000024',80) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-BA-02','9b200006-2026-4020-8020-000000000001',80,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000024','LOT-PBL6-OM-BA-02-01','OM-BA-02','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,80,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000025' AND sku<>'OM-BA-03') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-BA-03') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000025' AND sku='OM-BA-03') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-BA-03' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-BA-03';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-BA-03','9b200005-2026-4020-8020-000000000025',80) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-BA-03','9b200006-2026-4020-8020-000000000001',80,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000025','LOT-PBL6-OM-BA-03-01','OM-BA-03','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,80,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000026' AND sku<>'OM-BA-04') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-BA-04') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000026' AND sku='OM-BA-04') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-BA-04' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-BA-04';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-BA-04','9b200005-2026-4020-8020-000000000026',80) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-BA-04','9b200006-2026-4020-8020-000000000001',80,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000026','LOT-PBL6-OM-BA-04-01','OM-BA-04','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,80,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000027' AND sku<>'OM-BA-05') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-BA-05') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000027' AND sku='OM-BA-05') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-BA-05' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-BA-05';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-BA-05','9b200005-2026-4020-8020-000000000027',80) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-BA-05','9b200006-2026-4020-8020-000000000001',80,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000027','LOT-PBL6-OM-BA-05-01','OM-BA-05','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,80,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000028' AND sku<>'OM-KS-01') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-KS-01') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000028' AND sku='OM-KS-01') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-KS-01' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-KS-01';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-KS-01','9b200005-2026-4020-8020-000000000028',80) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-KS-01','9b200006-2026-4020-8020-000000000001',80,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000028','LOT-PBL6-OM-KS-01-01','OM-KS-01','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,80,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000029' AND sku<>'OM-KS-02') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-KS-02') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000029' AND sku='OM-KS-02') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-KS-02' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-KS-02';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-KS-02','9b200005-2026-4020-8020-000000000029',80) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-KS-02','9b200006-2026-4020-8020-000000000001',80,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000029','LOT-PBL6-OM-KS-02-01','OM-KS-02','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,80,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000030' AND sku<>'OM-KS-03') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-KS-03') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000030' AND sku='OM-KS-03') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-KS-03' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-KS-03';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-KS-03','9b200005-2026-4020-8020-000000000030',80) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-KS-03','9b200006-2026-4020-8020-000000000001',80,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000030','LOT-PBL6-OM-KS-03-01','OM-KS-03','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,80,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000031' AND sku<>'OM-KS-04') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-KS-04') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000031' AND sku='OM-KS-04') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-KS-04' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-KS-04';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-KS-04','9b200005-2026-4020-8020-000000000031',20) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-KS-04','9b200006-2026-4020-8020-000000000001',20,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000031','LOT-PBL6-OM-KS-04-01','OM-KS-04','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,20,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000032' AND sku<>'OM-KS-05') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-KS-05') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000032' AND sku='OM-KS-05') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-KS-05' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-KS-05';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-KS-05','9b200005-2026-4020-8020-000000000032',80) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-KS-05','9b200006-2026-4020-8020-000000000001',80,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000032','LOT-PBL6-OM-KS-05-01','OM-KS-05','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,80,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000033' AND sku<>'OM-KS-06') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-KS-06') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000033' AND sku='OM-KS-06') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-KS-06' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-KS-06';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-KS-06','9b200005-2026-4020-8020-000000000033',20) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-KS-06','9b200006-2026-4020-8020-000000000001',20,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000033','LOT-PBL6-OM-KS-06-01','OM-KS-06','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,20,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000034' AND sku<>'OM-HQ-01') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-HQ-01') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000034' AND sku='OM-HQ-01') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-HQ-01' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-HQ-01';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-HQ-01','9b200005-2026-4020-8020-000000000034',20) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-HQ-01','9b200006-2026-4020-8020-000000000001',20,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000034','LOT-PBL6-OM-HQ-01-01','OM-HQ-01','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,20,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000035' AND sku<>'OM-HQ-02') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-HQ-02') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000035' AND sku='OM-HQ-02') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-HQ-02' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-HQ-02';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-HQ-02','9b200005-2026-4020-8020-000000000035',20) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-HQ-02','9b200006-2026-4020-8020-000000000001',20,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000035','LOT-PBL6-OM-HQ-02-01','OM-HQ-02','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,20,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000036' AND sku<>'OM-HQ-03') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-HQ-03') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000036' AND sku='OM-HQ-03') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-HQ-03' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-HQ-03';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-HQ-03','9b200005-2026-4020-8020-000000000036',20) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-HQ-03','9b200006-2026-4020-8020-000000000001',20,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000036','LOT-PBL6-OM-HQ-03-01','OM-HQ-03','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,20,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000037' AND sku<>'OM-HQ-04') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-HQ-04') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000037' AND sku='OM-HQ-04') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-HQ-04' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-HQ-04';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-HQ-04','9b200005-2026-4020-8020-000000000037',20) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-HQ-04','9b200006-2026-4020-8020-000000000001',20,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000037','LOT-PBL6-OM-HQ-04-01','OM-HQ-04','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,20,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000038' AND sku<>'OM-DS-01') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-DS-01') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000038' AND sku='OM-DS-01') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-DS-01' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-DS-01';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-DS-01','9b200005-2026-4020-8020-000000000038',20) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-DS-01','9b200006-2026-4020-8020-000000000001',20,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000038','LOT-PBL6-OM-DS-01-01','OM-DS-01','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,20,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000039' AND sku<>'OM-DS-02') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-DS-02') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000039' AND sku='OM-DS-02') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-DS-02' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-DS-02';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-DS-02','9b200005-2026-4020-8020-000000000039',80) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-DS-02','9b200006-2026-4020-8020-000000000001',80,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000039','LOT-PBL6-OM-DS-02-01','OM-DS-02','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,80,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);

DO $$ BEGIN
IF EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000040' AND sku<>'OM-DS-03') OR
(EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-DS-03') AND (
NOT EXISTS(SELECT 1 FROM batches WHERE id='9b200005-2026-4020-8020-000000000040' AND sku='OM-DS-03') OR
EXISTS(SELECT 1 FROM inventory_items WHERE sku='OM-DS-03' AND warehouse_id<>'9b200006-2026-4020-8020-000000000001'))) THEN
RAISE EXCEPTION 'Conflicting existing Inventory SKU: OM-DS-03';
END IF;
END $$;
INSERT INTO inventory_demo_seed_000007(sku,batch_id,initial_quantity)
VALUES ('OM-DS-03','9b200005-2026-4020-8020-000000000040',80) ON CONFLICT (sku) DO NOTHING;
WITH inserted AS (
    INSERT INTO inventory_items(sku,warehouse_id,physical_qty,reserved_qty,status)
    VALUES ('OM-DS-03','9b200006-2026-4020-8020-000000000001',80,0,'ACTIVE')
    ON CONFLICT (sku) DO NOTHING RETURNING sku
)
UPDATE inventory_demo_seed_000007 SET item_created=true
WHERE sku IN (SELECT sku FROM inserted);
WITH inserted AS (
    INSERT INTO batches(id,batch_code,sku,supplier_id,mfg_date,exp_date,physical_qty,reserved_qty,status)
    VALUES ('9b200005-2026-4020-8020-000000000040','LOT-PBL6-OM-DS-03-01','OM-DS-03','9b200007-2026-4020-8020-000000000001',CURRENT_DATE-7,CURRENT_DATE+90,80,0,'ACTIVE')
    ON CONFLICT (id) DO NOTHING RETURNING id
)
UPDATE inventory_demo_seed_000007 SET batch_created=true
WHERE batch_id IN (SELECT id FROM inserted);
