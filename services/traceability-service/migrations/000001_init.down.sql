-- Destructive rollback is restricted to isolated test schemas by the migration runner.
DROP TABLE eligibility_observations, idempotency_records, outbox_events, audit_entries, review_decisions CASCADE;
DROP TABLE traceability_revisions, traceability_profiles, service_metadata CASCADE;
DROP FUNCTION protect_outbox_payload(), check_public_pointer(), check_review_consistency(), validate_decision(), protect_revision(), protect_profile(), reject_history_mutation();
