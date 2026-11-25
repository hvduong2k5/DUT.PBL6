-- Convert only carts participating in a merge. SQL remains authoritative for
-- these scopes even if the merge admission flag is subsequently disabled.
CREATE TABLE storefront_carts (
 scope text PRIMARY KEY CHECK(scope ~ '^(customer|guest):[0-9a-f-]{36}$'),
 cart_id uuid NOT NULL,
 revision bigint NOT NULL CHECK(revision>=0),
 items jsonb NOT NULL CHECK(jsonb_typeof(items)='array'),
 expires_at timestamptz NOT NULL DEFAULT clock_timestamp()+interval '30 days',
 updated_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TABLE guest_cart_merges (
 customer_scope text NOT NULL,
 merge_key text NOT NULL CHECK(length(merge_key) BETWEEN 16 AND 128),
 guest_scope text NOT NULL UNIQUE,
 response jsonb NOT NULL,
 merged_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 PRIMARY KEY(customer_scope,merge_key)
);
CREATE FUNCTION immutable_cart_merge() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'cart merge results are immutable'; END $$;
CREATE TRIGGER immutable_cart_merge BEFORE UPDATE OR DELETE ON guest_cart_merges
 FOR EACH ROW EXECUTE FUNCTION immutable_cart_merge();
