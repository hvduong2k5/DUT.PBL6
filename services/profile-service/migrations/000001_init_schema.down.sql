-- =============================================================================
-- ROLLBACK SCHEMA MIGRATION: MS-15 PROFILE SERVICE
-- =============================================================================

DROP TABLE IF EXISTS outbox_events CASCADE;
DROP TABLE IF EXISTS employee_profiles CASCADE;
DROP TABLE IF EXISTS departments CASCADE;
DROP TABLE IF EXISTS guest_order_claims CASCADE;
DROP TABLE IF EXISTS shipping_addresses CASCADE;
DROP TABLE IF EXISTS customer_profiles CASCADE;
DROP TABLE IF EXISTS administrative_units CASCADE;
