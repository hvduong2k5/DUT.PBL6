-- Deprecated 2026-11-21: destructive legacy seed is disabled.
DO $$ BEGIN RAISE EXCEPTION 'Legacy seed disabled. Run inventory-migrate, then INVENTORY_SEED_DEMO=true inventory-seed-demo. See services/inventory-service/MIGRATIONS.md'; END $$;
