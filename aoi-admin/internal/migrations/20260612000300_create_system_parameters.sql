-- +goose Up
CREATE TABLE IF NOT EXISTS system_parameters (
    id BIGINT PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    param_key VARCHAR(128) NOT NULL,
    param_value TEXT NOT NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP NULL,
    UNIQUE (param_key)
);

CREATE INDEX idx_system_parameters_name ON system_parameters (name);
CREATE INDEX idx_system_parameters_created_at ON system_parameters (created_at);

-- +goose Down
DROP TABLE IF EXISTS system_parameters;
