BEGIN;

CREATE TABLE return_cases (
    id UUID PRIMARY KEY,
    code VARCHAR(30) NOT NULL UNIQUE,
    customer_id UUID NOT NULL,
    order_id VARCHAR(100) NOT NULL,
    linked_ticket_id UUID REFERENCES support_tickets(id),
    parent_case_id UUID REFERENCES return_cases(id),
    requested_resolution VARCHAR(30) NOT NULL CHECK (requested_resolution IN ('RETURN_AND_REFUND', 'REFUND_WITHOUT_RETURN')),
    reason_code VARCHAR(50) NOT NULL CHECK (char_length(reason_code) BETWEEN 1 AND 50),
    customer_note TEXT NOT NULL DEFAULT '' CHECK (char_length(customer_note) <= 2000),
    status VARCHAR(30) NOT NULL CHECK (status IN (
        'REQUESTED', 'NEEDS_INFORMATION', 'REVIEWING', 'APPROVED',
        'PARTIALLY_APPROVED', 'REJECTED', 'IN_FULFILLMENT', 'COMPLETED', 'CLOSED'
    )),
    order_snapshot JSONB NOT NULL CHECK (
        jsonb_typeof(order_snapshot) = 'object' AND octet_length(order_snapshot::text) <= 65536
    ),
    source_contract_version VARCHAR(50) NOT NULL,
    source_captured_at TIMESTAMPTZ NOT NULL,
    return_eligible_at TIMESTAMPTZ NOT NULL,
    eligibility_expires_at TIMESTAMPTZ NOT NULL,
    version INTEGER NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMPTZ,
    CHECK (eligibility_expires_at >= return_eligible_at),
    CHECK (source_captured_at <= created_at),
    CHECK (parent_case_id IS NULL OR parent_case_id <> id)
);

CREATE INDEX idx_return_cases_customer_created
    ON return_cases (customer_id, created_at DESC, id DESC);
CREATE INDEX idx_return_cases_queue
    ON return_cases (status, created_at, id)
    WHERE status NOT IN ('CLOSED');
CREATE INDEX idx_return_cases_order
    ON return_cases (order_id, created_at DESC);

CREATE TABLE return_case_lines (
    id UUID PRIMARY KEY,
    return_case_id UUID NOT NULL REFERENCES return_cases(id),
    order_line_id VARCHAR(100) NOT NULL,
    product_id VARCHAR(100) NOT NULL,
    sku VARCHAR(100) NOT NULL,
    product_name VARCHAR(255) NOT NULL,
    purchased_quantity INTEGER NOT NULL CHECK (purchased_quantity > 0),
    previously_committed_return_quantity INTEGER NOT NULL DEFAULT 0 CHECK (previously_committed_return_quantity >= 0),
    requested_quantity INTEGER NOT NULL CHECK (requested_quantity > 0),
    refundable_ceiling_minor BIGINT NOT NULL CHECK (refundable_ceiling_minor >= 0),
    currency CHAR(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    source_snapshot JSONB NOT NULL CHECK (
        jsonb_typeof(source_snapshot) = 'object' AND octet_length(source_snapshot::text) <= 32768
    ),
    source_captured_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_return_case_order_line UNIQUE (return_case_id, order_line_id),
    CONSTRAINT uq_return_case_line_case UNIQUE (id, return_case_id),
    CHECK (previously_committed_return_quantity + requested_quantity <= purchased_quantity)
);

CREATE INDEX idx_return_case_lines_order_line
    ON return_case_lines (order_line_id, return_case_id);

CREATE TABLE return_evidence (
    id UUID PRIMARY KEY,
    return_case_id UUID NOT NULL REFERENCES return_cases(id),
    return_case_line_id UUID REFERENCES return_case_lines(id),
    media_id UUID NOT NULL REFERENCES media_files(id),
    submitted_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_return_evidence_media UNIQUE (media_id),
    CONSTRAINT fk_return_evidence_line_case FOREIGN KEY (return_case_line_id, return_case_id)
        REFERENCES return_case_lines(id, return_case_id)
);

CREATE INDEX idx_return_evidence_case
    ON return_evidence (return_case_id, created_at, id);

CREATE TABLE return_decisions (
    id UUID PRIMARY KEY,
    return_case_id UUID NOT NULL REFERENCES return_cases(id),
    decision_version INTEGER NOT NULL CHECK (decision_version > 0),
    outcome VARCHAR(30) NOT NULL CHECK (outcome IN ('APPROVED', 'PARTIALLY_APPROVED', 'REJECTED')),
    decided_by UUID NOT NULL,
    reason_code VARCHAR(50) NOT NULL CHECK (char_length(reason_code) BETWEEN 1 AND 50),
    reason TEXT NOT NULL CHECK (char_length(reason) BETWEEN 1 AND 2000),
    return_required BOOLEAN NOT NULL,
    approved_total_minor BIGINT NOT NULL CHECK (approved_total_minor >= 0),
    authoritative_ceiling_minor BIGINT NOT NULL CHECK (authoritative_ceiling_minor >= 0),
    currency CHAR(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    source_snapshot JSONB NOT NULL CHECK (
        jsonb_typeof(source_snapshot) = 'object' AND octet_length(source_snapshot::text) <= 65536
    ),
    source_captured_at TIMESTAMPTZ NOT NULL,
    decided_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT uq_return_decision_version UNIQUE (return_case_id, decision_version),
    CONSTRAINT uq_return_decision_case UNIQUE (id, return_case_id),
    CHECK (approved_total_minor <= authoritative_ceiling_minor),
    CHECK (
        (outcome = 'REJECTED' AND approved_total_minor = 0)
        OR outcome IN ('APPROVED', 'PARTIALLY_APPROVED')
    )
);

CREATE TABLE return_decision_lines (
    id UUID PRIMARY KEY,
    return_case_id UUID NOT NULL REFERENCES return_cases(id),
    return_decision_id UUID NOT NULL,
    return_case_line_id UUID NOT NULL,
    outcome VARCHAR(20) NOT NULL CHECK (outcome IN ('APPROVED', 'REJECTED')),
    approved_quantity INTEGER NOT NULL CHECK (approved_quantity >= 0),
    approved_amount_minor BIGINT NOT NULL CHECK (approved_amount_minor >= 0),
    reason_code VARCHAR(50) NOT NULL CHECK (char_length(reason_code) BETWEEN 1 AND 50),
    CONSTRAINT uq_return_decision_line UNIQUE (return_decision_id, return_case_line_id),
    CONSTRAINT fk_return_decision_line_decision_case FOREIGN KEY (return_decision_id, return_case_id)
        REFERENCES return_decisions(id, return_case_id),
    CONSTRAINT fk_return_decision_line_case FOREIGN KEY (return_case_line_id, return_case_id)
        REFERENCES return_case_lines(id, return_case_id),
    CHECK (
        (outcome = 'REJECTED' AND approved_quantity = 0 AND approved_amount_minor = 0)
        OR outcome = 'APPROVED'
    )
);

CREATE TABLE return_status_history (
    id UUID PRIMARY KEY,
    return_case_id UUID NOT NULL REFERENCES return_cases(id),
    from_status VARCHAR(30),
    to_status VARCHAR(30) NOT NULL,
    actor_id UUID NOT NULL,
    reason TEXT NOT NULL CHECK (char_length(reason) BETWEEN 1 AND 2000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_return_status_history
    ON return_status_history (return_case_id, created_at, id);

CREATE TABLE returned_goods_tracking (
    id UUID PRIMARY KEY,
    return_case_id UUID NOT NULL REFERENCES return_cases(id),
    return_decision_id UUID NOT NULL,
    status VARCHAR(40) NOT NULL CHECK (status IN (
        'NOT_REQUIRED', 'AWAITING_CUSTOMER_SHIPMENT', 'IN_TRANSIT', 'RECEIVED', 'INSPECTED'
    )),
    reverse_shipment_id VARCHAR(100),
    inspection_id VARCHAR(100),
    disposition VARCHAR(30) CHECK (disposition IN ('QUARANTINE', 'DISPOSE', 'RETURN_TO_VENDOR', 'RESTOCKABLE')),
    version INTEGER NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    received_at TIMESTAMPTZ,
    inspected_at TIMESTAMPTZ,
    CONSTRAINT uq_returned_goods_decision UNIQUE (return_decision_id),
    CONSTRAINT fk_returned_goods_decision_case FOREIGN KEY (return_decision_id, return_case_id)
        REFERENCES return_decisions(id, return_case_id),
    CHECK (status = 'INSPECTED' OR disposition IS NULL),
    CHECK (status <> 'RECEIVED' OR received_at IS NOT NULL),
    CHECK (status <> 'INSPECTED' OR (received_at IS NOT NULL AND inspected_at IS NOT NULL AND disposition IS NOT NULL))
);

CREATE TABLE refund_obligations (
    id UUID PRIMARY KEY,
    return_case_id UUID NOT NULL REFERENCES return_cases(id),
    return_decision_id UUID NOT NULL,
    order_id VARCHAR(100) NOT NULL,
    original_payment_reference VARCHAR(200) NOT NULL,
    method VARCHAR(30) NOT NULL CHECK (method IN ('ORIGINAL_PAYMENT')),
    status VARCHAR(30) NOT NULL CHECK (status IN (
        'NOT_REQUIRED', 'PENDING_APPROVAL', 'APPROVED', 'REQUESTED',
        'PROCESSING', 'SUCCEEDED', 'FAILED'
    )),
    approved_amount_minor BIGINT NOT NULL CHECK (approved_amount_minor >= 0),
    authoritative_ceiling_minor BIGINT NOT NULL CHECK (authoritative_ceiling_minor >= 0),
    currency CHAR(3) NOT NULL CHECK (currency ~ '^[A-Z]{3}$'),
    executor_reference VARCHAR(200),
    failure_code VARCHAR(100),
    version INTEGER NOT NULL DEFAULT 1 CHECK (version > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    requested_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    CONSTRAINT uq_refund_obligation_decision UNIQUE (return_decision_id),
    CONSTRAINT fk_refund_obligation_decision_case FOREIGN KEY (return_decision_id, return_case_id)
        REFERENCES return_decisions(id, return_case_id),
    CHECK (approved_amount_minor <= authoritative_ceiling_minor),
    CHECK (status <> 'NOT_REQUIRED' OR approved_amount_minor = 0),
    CHECK (status <> 'SUCCEEDED' OR completed_at IS NOT NULL)
);

CREATE INDEX idx_refund_obligations_pending
    ON refund_obligations (status, updated_at, id)
    WHERE status IN ('REQUESTED', 'PROCESSING', 'FAILED');

CREATE TABLE refund_status_history (
    id UUID PRIMARY KEY,
    refund_obligation_id UUID NOT NULL REFERENCES refund_obligations(id),
    from_status VARCHAR(30),
    to_status VARCHAR(30) NOT NULL,
    event_id VARCHAR(200),
    actor_id UUID,
    reason TEXT NOT NULL CHECK (char_length(reason) BETWEEN 1 AND 2000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_refund_history_event UNIQUE (event_id)
);

CREATE INDEX idx_refund_status_history
    ON refund_status_history (refund_obligation_id, created_at, id);

COMMIT;
