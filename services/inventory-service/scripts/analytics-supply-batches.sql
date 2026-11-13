-- Run only against the owning Inventory database with a read-only role.
WITH stock AS (
    SELECT i.sku
    FROM inventory_items i
    WHERE i.sku = :'sku' AND i.warehouse_id::text = :'source_warehouse'
), cutoff AS (
    SELECT statement_timestamp() AS captured_at,
           (statement_timestamp() AT TIME ZONE 'Asia/Ho_Chi_Minh')::date AS business_date
)
SELECT json_build_object(
    'businessDate', c.business_date, 'sourceAsOf', c.captured_at,
    'batches', coalesce((
        SELECT json_agg(json_build_object(
            'batchCode', b.batch_code, 'expiresOn', b.exp_date,
            'sellableUnits', b.physical_qty - b.reserved_qty
        ) ORDER BY b.exp_date, b.batch_code)
        FROM batches b
        WHERE b.sku = s.sku AND b.status IN ('ACTIVE','NEAR_EXPIRY')
          AND b.exp_date > c.business_date AND b.physical_qty > b.reserved_qty
    ), '[]'::json)
)
FROM stock s CROSS JOIN cutoff c
WHERE c.business_date = :'day'::date;
