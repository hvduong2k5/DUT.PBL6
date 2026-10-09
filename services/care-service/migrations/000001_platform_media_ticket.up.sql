BEGIN;

CREATE TABLE idempotency_records (
    id UUID PRIMARY KEY,
    actor_id UUID NOT NULL,
    operation VARCHAR(100) NOT NULL,
    idempotency_key VARCHAR(200) NOT NULL,
    request_hash CHAR(64) NOT NULL,
    resource_id VARCHAR(100),
    response_status INTEGER,
    response_body JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT uq_idempotency_actor_operation_key UNIQUE (actor_id, operation, idempotency_key)
);

CREATE INDEX idx_idempotency_expiry ON idempotency_records (expires_at);

CREATE TABLE inbox_events (
    event_id VARCHAR(200) NOT NULL,
    consumer_name VARCHAR(100) NOT NULL,
    event_type VARCHAR(200) NOT NULL,
    payload JSONB NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    last_error TEXT,
    PRIMARY KEY (event_id, consumer_name)
);

CREATE TABLE outbox_events (
    id UUID PRIMARY KEY,
    aggregate_type VARCHAR(50) NOT NULL,
    aggregate_id VARCHAR(100) NOT NULL,
    event_type VARCHAR(200) NOT NULL,
    topic VARCHAR(100) NOT NULL,
    partition_key VARCHAR(200) NOT NULL,
    payload JSONB NOT NULL,
    traceparent VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    last_error TEXT,
    published_at TIMESTAMPTZ
);

CREATE INDEX idx_outbox_pending
    ON outbox_events (next_attempt_at, created_at)
    WHERE published_at IS NULL;

CREATE TABLE audit_logs (
    id UUID PRIMARY KEY,
    actor_id UUID,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id VARCHAR(100) NOT NULL,
    reason TEXT,
    before_state JSONB,
    after_state JSONB,
    request_id VARCHAR(100),
    trace_id VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_resource ON audit_logs (resource_type, resource_id, created_at DESC);

CREATE TABLE media_files (
    id UUID PRIMARY KEY,
    owner_id UUID NOT NULL,
    purpose VARCHAR(40) NOT NULL CHECK (purpose IN ('RETURN_EVIDENCE', 'TICKET_ATTACHMENT', 'REVIEW_MEDIA')),
    provider VARCHAR(20) NOT NULL DEFAULT 'S3',
    bucket VARCHAR(100) NOT NULL,
    object_key VARCHAR(500) NOT NULL UNIQUE,
    original_filename VARCHAR(255) NOT NULL,
    declared_mime_type VARCHAR(100) NOT NULL,
    detected_mime_type VARCHAR(100),
    declared_size_bytes BIGINT NOT NULL CHECK (declared_size_bytes > 0),
    actual_size_bytes BIGINT CHECK (actual_size_bytes > 0),
    checksum_sha256 CHAR(64),
    storage_etag VARCHAR(200),
    status VARCHAR(30) NOT NULL CHECK (status IN ('PENDING_UPLOAD', 'UPLOADED', 'SCANNING', 'READY', 'ATTACHED', 'REJECTED', 'EXPIRED', 'DELETED')),
    scan_status VARCHAR(20) NOT NULL CHECK (scan_status IN ('PENDING', 'CLEAN', 'INFECTED', 'FAILED')),
    version INTEGER NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    uploaded_at TIMESTAMPTZ,
    attached_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_media_owner_created ON media_files (owner_id, created_at DESC);
CREATE INDEX idx_media_orphan_cleanup ON media_files (expires_at)
    WHERE status IN ('PENDING_UPLOAD', 'UPLOADED', 'SCANNING', 'READY');

CREATE TABLE media_scan_results (
    id UUID PRIMARY KEY,
    media_id UUID NOT NULL REFERENCES media_files(id),
    scanner VARCHAR(100) NOT NULL,
    scanner_version VARCHAR(100),
    result VARCHAR(20) NOT NULL CHECK (result IN ('CLEAN', 'INFECTED', 'FAILED')),
    details JSONB,
    scanned_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_media_scan_media ON media_scan_results (media_id, scanned_at DESC);

CREATE TABLE support_tickets (
    id UUID PRIMARY KEY,
    code VARCHAR(30) NOT NULL UNIQUE,
    customer_id UUID NOT NULL,
    subject VARCHAR(200) NOT NULL,
    category VARCHAR(40) NOT NULL CHECK (category IN ('ORDER', 'PAYMENT', 'SHIPPING', 'PRODUCT_QUALITY', 'RETURN_REFUND', 'ACCOUNT', 'OTHER')),
    priority VARCHAR(20) NOT NULL CHECK (priority IN ('LOW', 'NORMAL', 'HIGH', 'URGENT')),
    status VARCHAR(30) NOT NULL CHECK (status IN ('NEW', 'TRIAGED', 'ASSIGNED', 'IN_PROGRESS', 'WAITING_CUSTOMER', 'WAITING_INTERNAL', 'RESOLVED', 'CLOSED')),
    order_id VARCHAR(100),
    assigned_staff_id UUID,
    assigned_queue VARCHAR(100),
    first_response_due_at TIMESTAMPTZ NOT NULL,
    first_responded_at TIMESTAMPTZ,
    resolution_due_at TIMESTAMPTZ NOT NULL,
    resolved_at TIMESTAMPTZ,
    closed_at TIMESTAMPTZ,
    version INTEGER NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tickets_customer_created ON support_tickets (customer_id, created_at DESC, id DESC);
CREATE INDEX idx_tickets_queue ON support_tickets (status, priority, created_at)
    WHERE status NOT IN ('CLOSED');

CREATE TABLE ticket_messages (
    id UUID PRIMARY KEY,
    ticket_id UUID NOT NULL REFERENCES support_tickets(id),
    sender_id UUID NOT NULL,
    sender_type VARCHAR(20) NOT NULL CHECK (sender_type IN ('CUSTOMER', 'STAFF', 'SYSTEM')),
    direction VARCHAR(20) NOT NULL CHECK (direction IN ('INBOUND', 'OUTBOUND', 'INTERNAL')),
    content TEXT NOT NULL CHECK (char_length(content) BETWEEN 1 AND 10000),
    delivery_status VARCHAR(20) NOT NULL DEFAULT 'STORED' CHECK (delivery_status IN ('STORED', 'PENDING', 'SENT', 'FAILED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ticket_messages_timeline ON ticket_messages (ticket_id, created_at, id);

CREATE TABLE ticket_internal_notes (
    id UUID PRIMARY KEY,
    ticket_id UUID NOT NULL REFERENCES support_tickets(id),
    author_id UUID NOT NULL,
    content TEXT NOT NULL CHECK (char_length(content) BETWEEN 1 AND 10000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ticket_notes_timeline ON ticket_internal_notes (ticket_id, created_at, id);

CREATE TABLE ticket_assignments (
    id UUID PRIMARY KEY,
    ticket_id UUID NOT NULL REFERENCES support_tickets(id),
    staff_id UUID,
    queue VARCHAR(100),
    assigned_by UUID NOT NULL,
    reason TEXT,
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    CHECK (staff_id IS NOT NULL OR queue IS NOT NULL)
);

CREATE INDEX idx_ticket_assignments_active ON ticket_assignments (ticket_id, assigned_at DESC)
    WHERE ended_at IS NULL;

CREATE TABLE ticket_status_history (
    id UUID PRIMARY KEY,
    ticket_id UUID NOT NULL REFERENCES support_tickets(id),
    from_status VARCHAR(30),
    to_status VARCHAR(30) NOT NULL,
    actor_id UUID NOT NULL,
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_ticket_status_history ON ticket_status_history (ticket_id, created_at, id);

CREATE TABLE ticket_sla_milestones (
    id UUID PRIMARY KEY,
    ticket_id UUID NOT NULL REFERENCES support_tickets(id),
    milestone_type VARCHAR(30) NOT NULL CHECK (milestone_type IN ('FIRST_RESPONSE', 'RESOLUTION')),
    due_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    paused_at TIMESTAMPTZ,
    total_paused_seconds BIGINT NOT NULL DEFAULT 0 CHECK (total_paused_seconds >= 0),
    breached_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_ticket_sla_type UNIQUE (ticket_id, milestone_type)
);

CREATE TABLE media_references (
    id UUID PRIMARY KEY,
    media_id UUID NOT NULL REFERENCES media_files(id),
    aggregate_type VARCHAR(50) NOT NULL CHECK (aggregate_type IN ('TICKET_MESSAGE', 'RETURN_CASE', 'REVIEW')),
    aggregate_id UUID NOT NULL,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_media_reference UNIQUE (media_id, aggregate_type, aggregate_id)
);

CREATE INDEX idx_media_reference_aggregate ON media_references (aggregate_type, aggregate_id);

COMMIT;

