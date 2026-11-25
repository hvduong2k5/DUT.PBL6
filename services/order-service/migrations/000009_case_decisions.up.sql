-- F-55: local payment case authority. Decision records for reconciliation
-- cases live in this service's own database. A record is created by a
-- PAYMENT_CASE_DECIDE holder who is not the case assignee, and the assignee
-- consumes it by reference when resolving (maker-checker). This replaces
-- the never-deployed external care authority for case decisions only;
-- refund approvals and the evidence vault keep their external contract.
CREATE TABLE payment_case_decisions (
 decision_reference text PRIMARY KEY,
 case_id uuid NOT NULL REFERENCES reconciliation_cases(id),
 connection_id uuid NOT NULL,
 receiver_profile_id uuid NOT NULL,
 resolution text NOT NULL CHECK(resolution IN ('REFUNDED','NO_PAYMENT_CONFIRMED','EXTERNAL_FUNDS_RESOLVED')),
 allowed boolean NOT NULL DEFAULT true,
 evidence_ids jsonb NOT NULL,
 policy_version text NOT NULL,
 decided_by uuid NOT NULL,
 reason text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX payment_case_decisions_case ON payment_case_decisions(case_id,created_at);
