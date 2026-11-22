WITH captured AS (SELECT transaction_timestamp() AS at), rows AS (
 SELECT jsonb_build_object('recordKey','order:'||o.id,'orderId',o.id,'orderVersion',o.version,
   'cohort',o.cohort,'channel',o.channel,'warehouseId',o.warehouse_id,
   'completedAt',m.occurred_at,'eligible',true,'eligibilityReason',NULL,
   'customerPrincipalId',CASE WHEN o.owner_kind='CUSTOMER' THEN o.principal_id ELSE NULL END,
   'identityMappingVersion',CASE WHEN o.owner_kind='CUSTOMER' THEN :'identity_mapping' ELSE NULL END,
   'items',(SELECT jsonb_agg(jsonb_build_object('skuCode',i.sku,'quantityAtCompletion',i.quantity)
      ORDER BY i.sku COLLATE "C") FROM order_items i WHERE i.order_id=o.id)
 ) AS body,o.id::text AS sort_key
 FROM orders o JOIN order_milestones m ON m.order_id=o.id AND m.kind='COMPLETED'
 CROSS JOIN captured t
 WHERE o.cohort='order-core-1' AND o.channel='D2C_WEB' AND o.warehouse_id=:'warehouse'
   AND m.occurred_at>=t.at-interval '90 days' AND m.occurred_at<t.at
   AND NOT o.id::text=ANY(string_to_array(:'excluded_orders',','))
)
SELECT jsonb_build_object('sourceAsOf',t.at,
 'window',jsonb_build_object('from',t.at-interval '90 days','to',t.at),
 'violations',0,
 'records',coalesce((SELECT jsonb_agg(body ORDER BY sort_key COLLATE "C") FROM rows),'[]'::jsonb))
FROM captured t;
