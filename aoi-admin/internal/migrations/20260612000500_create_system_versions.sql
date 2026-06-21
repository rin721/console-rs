-- +goose Up
CREATE TABLE IF NOT EXISTS system_versions (
    id BIGINT PRIMARY KEY,
    version_name VARCHAR(128) NOT NULL,
    version_code VARCHAR(128) NOT NULL,
    description TEXT NOT NULL,
    version_data TEXT NOT NULL,
    menu_count INTEGER NOT NULL,
    api_count INTEGER NOT NULL,
    dictionary_count INTEGER NOT NULL,
    source VARCHAR(32) NOT NULL,
    created_by BIGINT NOT NULL,
    created_by_username VARCHAR(128) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP NULL
);

CREATE INDEX idx_system_versions_version_name ON system_versions (version_name);
CREATE INDEX idx_system_versions_version_code ON system_versions (version_code);
CREATE INDEX idx_system_versions_source ON system_versions (source);
CREATE INDEX idx_system_versions_created_at ON system_versions (created_at);

-- +goose Down
DROP TABLE IF EXISTS system_versions;
