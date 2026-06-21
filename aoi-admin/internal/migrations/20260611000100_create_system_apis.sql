-- +goose Up
CREATE TABLE IF NOT EXISTS system_apis (
    id BIGINT PRIMARY KEY,
    code VARCHAR(256) NOT NULL,
    api_group VARCHAR(64) NOT NULL,
    http_method VARCHAR(16) NOT NULL,
    path VARCHAR(512) NOT NULL,
    description TEXT NOT NULL,
    permission VARCHAR(128) NOT NULL,
    status VARCHAR(32) NOT NULL,
    source VARCHAR(64) NOT NULL,
    synced_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    UNIQUE (http_method, path)
);

CREATE INDEX idx_system_apis_group ON system_apis (api_group);
CREATE INDEX idx_system_apis_status ON system_apis (status);

-- +goose Down
DROP TABLE IF EXISTS system_apis;
