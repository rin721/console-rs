-- +goose Up
CREATE TABLE IF NOT EXISTS system_operation_records (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL DEFAULT 0,
    username VARCHAR(128) NOT NULL DEFAULT '',
    ip_address VARCHAR(64) NOT NULL DEFAULT '',
    http_method VARCHAR(16) NOT NULL,
    path VARCHAR(512) NOT NULL,
    status INTEGER NOT NULL DEFAULT 0,
    latency_ms BIGINT NOT NULL DEFAULT 0,
    user_agent TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    body TEXT NOT NULL DEFAULT '',
    response TEXT NOT NULL DEFAULT '',
    trace_id VARCHAR(128) NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_system_operation_records_user ON system_operation_records (user_id);
CREATE INDEX idx_system_operation_records_ip ON system_operation_records (ip_address);
CREATE INDEX idx_system_operation_records_method ON system_operation_records (http_method);
CREATE INDEX idx_system_operation_records_path ON system_operation_records (path);
CREATE INDEX idx_system_operation_records_status ON system_operation_records (status);
CREATE INDEX idx_system_operation_records_trace ON system_operation_records (trace_id);
CREATE INDEX idx_system_operation_records_created_at ON system_operation_records (created_at);

-- +goose Down
DROP TABLE IF EXISTS system_operation_records;
