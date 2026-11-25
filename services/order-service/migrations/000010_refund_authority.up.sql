-- F-42 slice 2: local refund authority. Approval records and bank-statement
-- evidence certificates for manual refunds live in this service's own
-- database, replacing the never-deployed external care/finance authorities
-- for refund approvals and refund evidence. An approval is created by a
-- PAYMENT_REFUND_APPROVE holder; execution and verification are enforced
-- against different actors by the refund lifecycle itself (approver never
-- executes, verifier is neither approver nor executor). Evidence
-- certificates are issued by a PAYMENT_REFUND_VERIFY holder who is neither
-- the refund's approver nor its executor, after checking the bank
-- statement; the executor seals a certificate by attaching it, and the
-- verifier's bank presentation must match the sealed certificate exactly.
-- The evidence file vault (upload/download) keeps its external contract.
CREATE TABLE payment_refund_approvals (
 approval_id text PRIMARY KEY,
 receipt_id uuid NOT NULL,
 connection_id uuid NOT NULL,
 receiver_profile_id uuid NOT NULL,
 approver_id uuid NOT NULL,
 amount bigint NOT NULL CHECK(amount >= 1),
 currency text NOT NULL DEFAULT 'VND',
 state text NOT NULL DEFAULT 'APPROVED' CHECK(state IN ('APPROVED','WITHDRAWN')),
 beneficiary_account text NOT NULL,
 beneficiary_bank text NOT NULL,
 beneficiary_holder text NOT NULL,
 policy_version text NOT NULL,
 version bigint NOT NULL DEFAULT 1,
 withdrawal_reference text,
 reason text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 withdrawn_at timestamptz
);
CREATE INDEX payment_refund_approvals_receipt ON payment_refund_approvals(receipt_id,created_at);
CREATE TABLE payment_refund_evidence_records (
 reference text PRIMARY KEY,
 refund_id uuid NOT NULL,
 attempt_id uuid NOT NULL,
 sha256 text NOT NULL,
 source text NOT NULL DEFAULT 'BANK_STATEMENT',
 scan_state text NOT NULL DEFAULT 'CLEAN',
 decision text NOT NULL CHECK(decision IN ('SENT','NOT_SENT','INCONCLUSIVE')),
 definitive boolean NOT NULL DEFAULT false,
 amount bigint NOT NULL,
 currency text NOT NULL DEFAULT 'VND',
 beneficiary_account text NOT NULL,
 bank text NOT NULL,
 beneficiary_holder text NOT NULL,
 source_bank_account_id text NOT NULL,
 bank_identity text NOT NULL,
 bank_reference text NOT NULL,
 occurred_at timestamptz NOT NULL,
 issued_by uuid NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX payment_refund_evidence_records_refund ON payment_refund_evidence_records(refund_id,created_at);
