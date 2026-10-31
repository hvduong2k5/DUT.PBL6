CREATE TABLE IF NOT EXISTS catalog(sku text PRIMARY KEY,price bigint NOT NULL,weight bigint NOT NULL,name text NOT NULL,active boolean NOT NULL DEFAULT true,version bigint NOT NULL DEFAULT 1);
CREATE TABLE IF NOT EXISTS stock(sku text PRIMARY KEY REFERENCES catalog(sku),physical integer NOT NULL,available integer NOT NULL CHECK(available>=0));
CREATE TABLE IF NOT EXISTS reservations(order_id uuid PRIMARY KEY,id uuid NOT NULL UNIQUE,operation_key text NOT NULL UNIQUE,input_hash text NOT NULL,items jsonb NOT NULL,state text NOT NULL,expires_at timestamptz NOT NULL);
CREATE TABLE IF NOT EXISTS profiles(user_id uuid PRIMARY KEY,customer_id uuid NOT NULL UNIQUE,address_id uuid NOT NULL UNIQUE);
CREATE TABLE IF NOT EXISTS fulfillment(order_id uuid PRIMARY KEY,generation bigint NOT NULL,state text NOT NULL,task_id uuid);
CREATE TABLE IF NOT EXISTS care(order_id uuid PRIMARY KEY,active boolean NOT NULL DEFAULT false,finalized boolean NOT NULL DEFAULT false);
CREATE TABLE IF NOT EXISTS resources(id uuid PRIMARY KEY,order_id uuid NOT NULL,source text NOT NULL,version bigint NOT NULL,generation bigint NOT NULL DEFAULT 0);
CREATE TABLE IF NOT EXISTS approvals(id text PRIMARY KEY,payload jsonb NOT NULL);
CREATE TABLE IF NOT EXISTS provider_payments(transaction_id text PRIMARY KEY,reference text NOT NULL,payload jsonb NOT NULL);
CREATE TABLE IF NOT EXISTS provider_refunds(operation_key text PRIMARY KEY,transaction_id text NOT NULL REFERENCES provider_payments(transaction_id),amount bigint NOT NULL,status text NOT NULL,reference uuid NOT NULL,input_hash text NOT NULL);
CREATE TABLE IF NOT EXISTS faults(name text PRIMARY KEY,mode text NOT NULL,remaining integer NOT NULL);
INSERT INTO catalog(sku,price,weight,name) VALUES('MX-GION-500G',110000,500,'Me xung gion 500g'),('MX-DEO-300G',130000,300,'Me xung deo 300g') ON CONFLICT DO NOTHING;
INSERT INTO stock(sku,physical,available) SELECT sku,100,100 FROM catalog ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS catalog_quotes(id uuid PRIMARY KEY,snapshot jsonb NOT NULL,expires_at timestamptz NOT NULL);

ALTER TABLE provider_payments ADD COLUMN IF NOT EXISTS sequence bigserial UNIQUE;
CREATE TABLE IF NOT EXISTS core_inventory(order_id uuid PRIMARY KEY,id uuid NOT NULL UNIQUE,digest text NOT NULL,warehouse text NOT NULL,items jsonb NOT NULL,state text NOT NULL,version bigint NOT NULL,expires_at timestamptz NOT NULL);
CREATE TABLE IF NOT EXISTS core_inventory_commands(order_id uuid NOT NULL REFERENCES core_inventory(order_id),operation_key text NOT NULL,request_hash text NOT NULL,action text NOT NULL,decision_id uuid NOT NULL,result text NOT NULL,resulting_state text NOT NULL,decided_at timestamptz NOT NULL,PRIMARY KEY(order_id,operation_key));
CREATE TABLE IF NOT EXISTS core_source_history(source text NOT NULL,resource_id uuid NOT NULL,version bigint NOT NULL,order_id uuid NOT NULL,proof jsonb NOT NULL,PRIMARY KEY(source,resource_id,version));
CREATE TABLE IF NOT EXISTS core_completion_decisions(order_id uuid NOT NULL,order_version bigint NOT NULL,decision_id uuid NOT NULL,expires_at timestamptz NOT NULL,PRIMARY KEY(order_id,order_version));
CREATE TABLE IF NOT EXISTS core_admission_barriers(order_id uuid PRIMARY KEY,fenced_generation bigint NOT NULL);
