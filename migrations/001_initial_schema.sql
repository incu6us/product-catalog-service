CREATE TABLE products (
    product_id STRING(36) NOT NULL,
    name STRING(255) NOT NULL,
    description STRING(MAX),
    category STRING(100) NOT NULL,
    base_price_numerator INT64 NOT NULL,
    base_price_denominator INT64 NOT NULL,
    discount_percent NUMERIC,
    discount_start_date TIMESTAMP,
    discount_end_date TIMESTAMP,
    status STRING(20) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    archived_at TIMESTAMP,
    version INT64 NOT NULL,
) PRIMARY KEY (product_id);

CREATE TABLE outbox_events (
    event_id STRING(36) NOT NULL,
    event_type STRING(100) NOT NULL,
    aggregate_id STRING(36) NOT NULL,
    payload JSON NOT NULL,
    status STRING(20) NOT NULL,
    created_at TIMESTAMP NOT NULL OPTIONS (allow_commit_timestamp = true),
    processed_at TIMESTAMP,
) PRIMARY KEY (event_id);

CREATE INDEX idx_outbox_status ON outbox_events(status, created_at);
CREATE INDEX idx_products_category ON products(category, status);
