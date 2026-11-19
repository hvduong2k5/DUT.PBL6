WITH captured AS (SELECT transaction_timestamp() AS at), stock AS (
 SELECT i.*, coalesce(sum(b.physical_qty),0) AS batch_physical,
   coalesce(sum(b.reserved_qty),0) AS batch_reserved,
   coalesce(sum(b.physical_qty-b.reserved_qty) FILTER (
     WHERE b.status IN ('ACTIVE','NEAR_EXPIRY')
       AND b.exp_date>(t.at AT TIME ZONE 'Asia/Ho_Chi_Minh')::date),0) AS sellable
 FROM inventory_items i CROSS JOIN captured t LEFT JOIN batches b ON b.sku=i.sku
 WHERE i.warehouse_id=:'warehouse'::uuid
 GROUP BY i.sku,i.warehouse_id,i.physical_qty,i.reserved_qty,i.status,i.updated_at
), rows AS (
 SELECT jsonb_build_object('recordKey','sku:'||s.sku,'skuCode',s.sku,
   'warehouseId',s.warehouse_id,'skuActive',s.status='ACTIVE',
   'sellableUnits',CASE WHEN s.status='ACTIVE' THEN s.sellable ELSE 0 END,
   'observedAt',t.at,
   'eligibilityValidUntil',least(t.at+interval '120 seconds',
     (((t.at AT TIME ZONE 'Asia/Ho_Chi_Minh')::date+1)::timestamp AT TIME ZONE 'Asia/Ho_Chi_Minh'))
 ) AS body,s.sku AS sort_key FROM stock s CROSS JOIN captured t
)
SELECT jsonb_build_object('sourceAsOf',t.at,'window',NULL,
 'violations',(SELECT count(*) FROM stock WHERE physical_qty<>batch_physical OR reserved_qty<>batch_reserved),
 'records',coalesce((SELECT jsonb_agg(body ORDER BY sort_key COLLATE "C") FROM rows),'[]'::jsonb))
FROM captured t;
