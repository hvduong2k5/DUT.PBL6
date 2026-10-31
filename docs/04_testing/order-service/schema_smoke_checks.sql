-- Runtime 3.1 smoke checks against migrated Order database. All fixtures rolled back.
BEGIN;
INSERT INTO checkout_operations(id,scope,principal_id,planned_order_id,status,input_cipher,request_hash)
VALUES ('00000000-0000-4000-8000-000000000001','smoke','00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000003','ACCEPTED',decode('00','hex'),'hash');
INSERT INTO idempotency_records(scope,kind,key,request_hash,operation_id)
VALUES ('smoke','CHECKOUT','smoke-key','hash','00000000-0000-4000-8000-000000000001');
INSERT INTO orders(id,operation_id,order_code,scope,principal_id,method,status,stock_status,subtotal,shipping_fee,final_amount,snapshot_cipher,reservation_id,payment_expires_at,provider_reference)
VALUES ('00000000-0000-4000-8000-000000000003','00000000-0000-4000-8000-000000000001','SCHEMA-SMOKE','smoke','00000000-0000-4000-8000-000000000002','VIETQR','PENDING_PAYMENT','RESERVED',100000,25000,125000,decode('00','hex'),'schema-reserve',clock_timestamp()+interval '5 minutes','schema-smoke-ref');
INSERT INTO order_items(order_id,sku,quantity,unit_price,line_total,product_name)
VALUES ('00000000-0000-4000-8000-000000000003','SMOKE',1,100000,100000,'Synthetic');
DO $$
BEGIN
 BEGIN
  INSERT INTO idempotency_records(scope,kind,key,request_hash,operation_id) VALUES ('smoke','CHECKOUT','smoke-key','hash','00000000-0000-4000-8000-000000000001');
  RAISE EXCEPTION 'registry uniqueness not enforced';
 EXCEPTION WHEN unique_violation THEN NULL; END;
 BEGIN
  UPDATE orders SET status='PAID',payment_status='CONFIRMED' WHERE order_code='SCHEMA-SMOKE';
  RAISE EXCEPTION 'paid stock guard not enforced';
 EXCEPTION WHEN check_violation THEN NULL; END;
 BEGIN
  UPDATE orders SET status='CONFIRMED_COD',stock_status='COMMITTED' WHERE order_code='SCHEMA-SMOKE';
  RAISE EXCEPTION 'COD method guard not enforced';
 EXCEPTION WHEN check_violation THEN NULL; END;
 BEGIN
  UPDATE orders SET final_amount=1 WHERE order_code='SCHEMA-SMOKE';
  RAISE EXCEPTION 'commercial snapshot mutation not rejected';
 EXCEPTION WHEN raise_exception THEN
  IF SQLERRM <> 'commercial snapshot is immutable' THEN RAISE; END IF;
 END;
 BEGIN
  UPDATE order_items SET quantity=2 WHERE sku='SMOKE';
  RAISE EXCEPTION 'item mutation not rejected';
 EXCEPTION WHEN raise_exception THEN
  IF SQLERRM <> 'order snapshots are immutable' THEN RAISE; END IF;
 END;
 IF NOT EXISTS(SELECT 1 FROM checkout_operations WHERE scope='smoke' AND order_id IS NULL) THEN
  RAISE EXCEPTION 'operation must allow no public order yet';
 END IF;
END $$;
ROLLBACK;
SELECT 'PASS: operation-first, registry uniqueness, PAID/COD guards, immutable commercial/items; fixtures rolled back' AS result;
