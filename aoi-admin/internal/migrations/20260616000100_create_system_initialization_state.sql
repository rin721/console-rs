-- +goose Up
CREATE TABLE IF NOT EXISTS system_initialization_runs (
    id BIGINT PRIMARY KEY,
    run_key VARCHAR(64) NOT NULL,
    source VARCHAR(32) NOT NULL,
    mode VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL,
    current_step VARCHAR(128) NOT NULL DEFAULT '',
    started_by_user_id BIGINT NOT NULL DEFAULT 0,
    ip_address VARCHAR(64) NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL,
    config_path VARCHAR(512) NOT NULL DEFAULT '',
    config_fingerprint VARCHAR(128) NOT NULL DEFAULT '',
    last_error TEXT NOT NULL,
    started_at TIMESTAMP NULL,
    finished_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    UNIQUE (run_key)
);

CREATE INDEX idx_system_initialization_runs_status ON system_initialization_runs (status);
CREATE INDEX idx_system_initialization_runs_source ON system_initialization_runs (source);
CREATE INDEX idx_system_initialization_runs_created_at ON system_initialization_runs (created_at);

CREATE TABLE IF NOT EXISTS system_initialization_steps (
    id BIGINT PRIMARY KEY,
    run_id BIGINT NOT NULL,
    step_key VARCHAR(128) NOT NULL,
    phase VARCHAR(64) NOT NULL,
    step_order INTEGER NOT NULL,
    status VARCHAR(32) NOT NULL,
    required BOOLEAN NOT NULL DEFAULT true,
    retryable BOOLEAN NOT NULL DEFAULT true,
    idempotent BOOLEAN NOT NULL DEFAULT true,
    attempt INTEGER NOT NULL DEFAULT 0,
    input_summary TEXT NOT NULL,
    output_summary TEXT NOT NULL,
    test_status VARCHAR(32) NOT NULL DEFAULT '',
    test_summary TEXT NOT NULL DEFAULT '',
    test_error TEXT NOT NULL DEFAULT '',
    skipped_reason TEXT NOT NULL DEFAULT '',
    repair_hint TEXT NOT NULL DEFAULT '',
    restart_required BOOLEAN NOT NULL DEFAULT false,
    error_code VARCHAR(128) NOT NULL DEFAULT '',
    error_message TEXT NOT NULL,
    started_at TIMESTAMP NULL,
    finished_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    UNIQUE (run_id, step_key)
);

CREATE INDEX idx_system_initialization_steps_run ON system_initialization_steps (run_id);
CREATE INDEX idx_system_initialization_steps_status ON system_initialization_steps (status);
CREATE INDEX idx_system_initialization_steps_key ON system_initialization_steps (step_key);

-- +goose Down
DROP TABLE IF EXISTS system_initialization_steps;
DROP TABLE IF EXISTS system_initialization_runs;
