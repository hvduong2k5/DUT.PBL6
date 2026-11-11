WITH captured AS (SELECT transaction_timestamp() AS at), rows AS (
 SELECT jsonb_build_object(
   'recordKey','sku:'||v.sku_code,'skuCode',v.sku_code,'productId',p.id,'categoryId',p.category_id,
   'productVersion',p.version,'skuVersion',v.version,'priceVersion',c.version,
   'publiclyVisible',p.approval_status='APPROVED' AND p.listing_status='ACTIVE',
   'skuActive',v.listing_status='ACTIVE','slug',p.slug,'name',p.name,
   'thumbnailUrl','', 'priceVnd',CASE WHEN c.currency_code='VND' AND c.amount_nanos=0 AND c.amount_units>0 THEN c.amount_units ELSE NULL END,
   'weightGrams',CASE WHEN v.weight_value>0 AND
      (v.weight_value * CASE WHEN v.weight_unit='KILOGRAM' THEN 1000 ELSE 1 END) =
       trunc(v.weight_value * CASE WHEN v.weight_unit='KILOGRAM' THEN 1000 ELSE 1 END)
      AND v.weight_unit IN ('GRAM','KILOGRAM')
      THEN (v.weight_value * CASE WHEN v.weight_unit='KILOGRAM' THEN 1000 ELSE 1 END)::bigint ELSE NULL END,
   'packagingType',v.packaging_type,'fallbackRank',NULL,
   'sourceUpdatedAt',greatest(p.updated_at,v.updated_at,c.updated_at)
 ) AS body, v.sku_code AS sort_key
 FROM product_variants v JOIN products p ON p.id=v.product_id CROSS JOIN captured t
 LEFT JOIN channel_prices c ON c.variant_id=v.id AND c.channel='D2C_WEB' AND c.status='ACTIVE'
   AND c.effective_from<=t.at AND (c.effective_to IS NULL OR c.effective_to>t.at)
)
SELECT jsonb_build_object('sourceAsOf',t.at,'window',NULL,'violations',0,
 'records',coalesce((SELECT jsonb_agg(body ORDER BY sort_key COLLATE "C") FROM rows),'[]'::jsonb))
FROM captured t;
