-- +goose Up
CREATE TABLE IF NOT EXISTS system_traffic_probe_targets (
    id BIGINT PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    url TEXT NOT NULL,
    http_method VARCHAR(8) NOT NULL DEFAULT 'GET',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    interval_seconds INTEGER NOT NULL DEFAULT 30,
    timeout_seconds INTEGER NOT NULL DEFAULT 5,
    expected_status_codes VARCHAR(128) NOT NULL DEFAULT '200-399',
    expected_final_host VARCHAR(255) NOT NULL DEFAULT '',
    expected_content_keyword TEXT NOT NULL DEFAULT '',
    expected_ip_cidrs TEXT NOT NULL DEFAULT '',
    expected_tls_fingerprint VARCHAR(128) NOT NULL DEFAULT '',
    allow_private_network BOOLEAN NOT NULL DEFAULT FALSE,
    alert_channels VARCHAR(128) NOT NULL DEFAULT 'event',
    email_recipients TEXT NOT NULL DEFAULT '',
    last_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    last_severity VARCHAR(32) NOT NULL DEFAULT 'ok',
    last_reason TEXT NOT NULL DEFAULT '',
    last_probed_at TIMESTAMP NULL,
    next_probe_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP NULL
);

CREATE INDEX idx_system_traffic_probe_targets_enabled ON system_traffic_probe_targets (enabled);
CREATE INDEX idx_system_traffic_probe_targets_next_probe ON system_traffic_probe_targets (next_probe_at);
CREATE INDEX idx_system_traffic_probe_targets_deleted ON system_traffic_probe_targets (deleted_at);

CREATE TABLE IF NOT EXISTS system_traffic_probe_results (
    id BIGINT PRIMARY KEY,
    target_id BIGINT NOT NULL,
    target_name VARCHAR(128) NOT NULL DEFAULT '',
    url TEXT NOT NULL,
    http_method VARCHAR(8) NOT NULL DEFAULT 'GET',
    status VARCHAR(32) NOT NULL,
    severity VARCHAR(32) NOT NULL,
    reason VARCHAR(255) NOT NULL DEFAULT '',
    stage VARCHAR(64) NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    dns_ips TEXT NOT NULL DEFAULT '',
    final_url TEXT NOT NULL DEFAULT '',
    status_code INTEGER NOT NULL DEFAULT 0,
    tls_subject VARCHAR(255) NOT NULL DEFAULT '',
    tls_issuer VARCHAR(255) NOT NULL DEFAULT '',
    tls_not_after TIMESTAMP NULL,
    tls_fingerprint_sha256 VARCHAR(128) NOT NULL DEFAULT '',
    dns_duration_ms BIGINT NOT NULL DEFAULT 0,
    connect_duration_ms BIGINT NOT NULL DEFAULT 0,
    tls_duration_ms BIGINT NOT NULL DEFAULT 0,
    ttfb_ms BIGINT NOT NULL DEFAULT 0,
    total_duration_ms BIGINT NOT NULL DEFAULT 0,
    evidence_json TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_system_traffic_probe_results_target ON system_traffic_probe_results (target_id);
CREATE INDEX idx_system_traffic_probe_results_status ON system_traffic_probe_results (status);
CREATE INDEX idx_system_traffic_probe_results_severity ON system_traffic_probe_results (severity);
CREATE INDEX idx_system_traffic_probe_results_created_at ON system_traffic_probe_results (created_at);

CREATE TABLE IF NOT EXISTS system_traffic_hijack_events (
    id BIGINT PRIMARY KEY,
    target_id BIGINT NOT NULL,
    target_name VARCHAR(128) NOT NULL DEFAULT '',
    reason VARCHAR(255) NOT NULL,
    severity VARCHAR(32) NOT NULL,
    state VARCHAR(32) NOT NULL,
    evidence_hash VARCHAR(64) NOT NULL,
    evidence_json TEXT NOT NULL DEFAULT '',
    first_seen_at TIMESTAMP NOT NULL,
    last_seen_at TIMESTAMP NOT NULL,
    resolved_at TIMESTAMP NULL,
    occurrences INTEGER NOT NULL DEFAULT 1,
    notification_status VARCHAR(64) NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_system_traffic_hijack_events_target ON system_traffic_hijack_events (target_id);
CREATE INDEX idx_system_traffic_hijack_events_state ON system_traffic_hijack_events (state);
CREATE INDEX idx_system_traffic_hijack_events_severity ON system_traffic_hijack_events (severity);
CREATE INDEX idx_system_traffic_hijack_events_hash ON system_traffic_hijack_events (evidence_hash);
CREATE INDEX idx_system_traffic_hijack_events_last_seen ON system_traffic_hijack_events (last_seen_at);

-- +goose Down
DROP TABLE IF EXISTS system_traffic_hijack_events;
DROP TABLE IF EXISTS system_traffic_probe_results;
DROP TABLE IF EXISTS system_traffic_probe_targets;
