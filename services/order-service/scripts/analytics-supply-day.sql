-- Run only against the owning Order database with a read-only role.
WITH totals AS (
    SELECT coalesce(sum(i.quantity), 0) AS sold
    FROM order_milestones m
    JOIN orders o ON o.id = m.order_id
    JOIN order_items i ON i.order_id = o.id
    WHERE m.kind = 'COMPLETED' AND o.cohort = 'order-core-1'
      AND o.warehouse_id = :'source_warehouse' AND i.sku = :'sku'
      AND m.occurred_at >= (:'day'::date::timestamp AT TIME ZONE 'Asia/Ho_Chi_Minh')
      AND m.occurred_at < ((:'day'::date + 1)::timestamp AT TIME ZONE 'Asia/Ho_Chi_Minh')
)
SELECT json_build_object('soldUnits', sold, 'returnedUnits', :'returned'::int)
FROM totals
WHERE sold = :'expected_sold'::bigint
  AND :'as_of'::timestamptz <= statement_timestamp()
  AND :'as_of'::timestamptz >= ((:'day'::date + 1)::timestamp AT TIME ZONE 'Asia/Ho_Chi_Minh');
