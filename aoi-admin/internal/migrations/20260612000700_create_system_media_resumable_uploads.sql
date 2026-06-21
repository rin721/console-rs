-- +goose Up
CREATE TABLE IF NOT EXISTS system_media_upload_sessions (
    id BIGINT PRIMARY KEY,
    category_id BIGINT NOT NULL DEFAULT 0,
    file_hash VARCHAR(128) NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    mime_type VARCHAR(128) NOT NULL DEFAULT '',
    extension VARCHAR(32) NOT NULL DEFAULT '',
    size_bytes BIGINT NOT NULL,
    chunk_size BIGINT NOT NULL,
    chunk_total INTEGER NOT NULL,
    status VARCHAR(32) NOT NULL,
    final_asset_id BIGINT NOT NULL DEFAULT 0,
    uploaded_by BIGINT NOT NULL DEFAULT 0,
    uploaded_by_username VARCHAR(128) NOT NULL DEFAULT '',
    expires_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP NULL
);

CREATE INDEX IF NOT EXISTS idx_system_media_upload_sessions_lookup
    ON system_media_upload_sessions (file_hash, file_name, category_id, uploaded_by, status);
CREATE INDEX IF NOT EXISTS idx_system_media_upload_sessions_expires
    ON system_media_upload_sessions (status, expires_at);

CREATE TABLE IF NOT EXISTS system_media_upload_chunks (
    id BIGINT PRIMARY KEY,
    session_id BIGINT NOT NULL,
    chunk_index INTEGER NOT NULL,
    chunk_hash VARCHAR(128) NOT NULL,
    storage_key VARCHAR(512) NOT NULL,
    size_bytes BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_system_media_upload_chunks_session_index
    ON system_media_upload_chunks (session_id, chunk_index);
CREATE INDEX IF NOT EXISTS idx_system_media_upload_chunks_session
    ON system_media_upload_chunks (session_id);

-- +goose Down
DROP TABLE IF EXISTS system_media_upload_chunks;
DROP TABLE IF EXISTS system_media_upload_sessions;
