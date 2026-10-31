-- Preserve canonical source history independently from event delivery retries.
CREATE TABLE core_source_evidence(
 source text NOT NULL,event_id text NOT NULL,order_id uuid NOT NULL REFERENCES orders(id),
 resource_id text NOT NULL,source_version bigint NOT NULL,evidence_hash text NOT NULL,evidence_cipher bytea NOT NULL,
 occurred_at timestamptz NOT NULL,created_at timestamptz NOT NULL DEFAULT clock_timestamp(),PRIMARY KEY(source,event_id)
);
CREATE TRIGGER core_source_evidence_immutable BEFORE UPDATE OR DELETE ON core_source_evidence FOR EACH ROW EXECUTE FUNCTION immutable_audit();
CREATE TABLE core_lifecycle_facts(order_id uuid PRIMARY KEY REFERENCES orders(id),delivered_at timestamptz NOT NULL,source text NOT NULL,event_id text NOT NULL,FOREIGN KEY(source,event_id) REFERENCES core_source_evidence(source,event_id));
CREATE TRIGGER core_lifecycle_facts_immutable BEFORE UPDATE OR DELETE ON core_lifecycle_facts FOR EACH ROW EXECUTE FUNCTION immutable_audit();
