-- Add minor-version voucher snapshots without rewriting historical 1.0 evidence.
ALTER TABLE orders DROP CONSTRAINT core_order_shape;
ALTER TABLE orders ADD CONSTRAINT core_order_shape CHECK(cohort<>'order-core-1' OR
 (snapshot_id IS NOT NULL AND snapshot_schema_version IN ('1.0','1.1') AND snapshot_schema_version IS NOT NULL
  AND snapshot_hash IS NOT NULL AND snapshot_hash ~ '^[0-9a-f]{64}$' AND core_state_cipher IS NOT NULL
  AND ((owner_kind='CUSTOMER' AND customer_id IS NOT NULL AND guest_session_id IS NULL)
    OR (owner_kind='GUEST' AND guest_session_id IS NOT NULL AND customer_id IS NULL)) AND owner_kind IS NOT NULL
  AND ((method='VIETQR' AND payment_expires_at IS NOT NULL) OR (method='COD' AND payment_expires_at IS NULL))));

CREATE TABLE order_voucher_finalizations (
 order_id uuid PRIMARY KEY REFERENCES orders(id),
 hold_id uuid NOT NULL UNIQUE,
 decision_id text NOT NULL,
 recorded_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
