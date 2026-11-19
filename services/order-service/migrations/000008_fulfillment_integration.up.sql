CREATE TABLE fulfillment_permits (
 id uuid PRIMARY KEY,
 order_id uuid NOT NULL REFERENCES orders(id),
 generation bigint NOT NULL CHECK(generation>0),
 task_request_key text NOT NULL,
 items_digest text NOT NULL,
 warehouse_id text NOT NULL,
 order_version bigint NOT NULL,
 policy_version text NOT NULL,
 expires_at timestamptz NOT NULL,
 UNIQUE(order_id,generation,task_request_key)
);
CREATE TABLE fulfillment_shipping_handoffs (
 order_id uuid PRIMARY KEY REFERENCES orders(id),
 package_id uuid NOT NULL UNIQUE,
 task_id uuid NOT NULL,
 sealed_version bigint NOT NULL,
 request_hash text NOT NULL,
 request_cipher bytea NOT NULL,
 shipment_id uuid,
 lease_owner text,
 lease_until timestamptz,
 lease_epoch bigint NOT NULL DEFAULT 0,
 attempts integer NOT NULL DEFAULT 0,
 retry_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
