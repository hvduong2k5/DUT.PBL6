-- Synthetic fixtures only. Re-running resets just these two customers' test data.
BEGIN;
DELETE FROM shipping_addresses WHERE customer_id IN ('c0000000-0000-0000-0000-000000000001','c0000000-0000-0000-0000-000000000002');
DELETE FROM guest_order_claims WHERE customer_id IN ('c0000000-0000-0000-0000-000000000001','c0000000-0000-0000-0000-000000000002');
INSERT INTO customer_profiles(id,user_id,full_name,phone_number,email,gender,status,version)
VALUES
('c0000000-0000-0000-0000-000000000001','d0000000-0000-0000-0000-000000000001','Test Customer A','+84905111111','a@example.test','UNSPECIFIED','ACTIVE',1),
('c0000000-0000-0000-0000-000000000002','d0000000-0000-0000-0000-000000000002','Test Customer B','+84905222222','b@example.test','UNSPECIFIED','ACTIVE',1)
ON CONFLICT(id) DO UPDATE SET full_name=EXCLUDED.full_name,phone_number=EXCLUDED.phone_number,email=EXCLUDED.email,version=1;
INSERT INTO guest_order_claims(id,customer_id,order_id,phone_number,claim_status)
VALUES('b0000000-0000-0000-0000-000000000001','c0000000-0000-0000-0000-000000000002','TEST-CLAIM-B','+84905222222','VERIFIED');
COMMIT;
