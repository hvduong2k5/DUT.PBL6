BEGIN;

DROP TABLE IF EXISTS media_references;
DROP TABLE IF EXISTS ticket_sla_milestones;
DROP TABLE IF EXISTS ticket_status_history;
DROP TABLE IF EXISTS ticket_assignments;
DROP TABLE IF EXISTS ticket_internal_notes;
DROP TABLE IF EXISTS ticket_messages;
DROP TABLE IF EXISTS support_tickets;
DROP TABLE IF EXISTS media_scan_results;
DROP TABLE IF EXISTS media_files;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS outbox_events;
DROP TABLE IF EXISTS inbox_events;
DROP TABLE IF EXISTS idempotency_records;

COMMIT;

