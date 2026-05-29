CREATE TABLE IF NOT EXISTS audit_logs (
    id          BIGSERIAL    PRIMARY KEY,
    event_type  VARCHAR(50)  NOT NULL,
    resource    VARCHAR(50)  NOT NULL,
    resource_id BIGINT,
    actor       VARCHAR(100),           -- nullable until auth is added
    request_id  VARCHAR(36),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
