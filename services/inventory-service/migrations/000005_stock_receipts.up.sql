CREATE TABLE stock_receipts (
    actor_id uuid NOT NULL,
    idem_key varchar(128) NOT NULL,
    sku varchar(50) NOT NULL REFERENCES inventory_items(sku),
    source_reference varchar(220) NOT NULL,
    request jsonb NOT NULL,
    response jsonb NOT NULL,
    PRIMARY KEY (actor_id, idem_key),
    UNIQUE (sku, source_reference)
);
