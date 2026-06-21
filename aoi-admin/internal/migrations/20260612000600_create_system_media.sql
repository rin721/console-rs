-- +goose Up
CREATE TABLE IF NOT EXISTS system_media_categories (
    id BIGINT PRIMARY KEY,
    parent_id BIGINT NOT NULL DEFAULT 0,
    name VARCHAR(128) NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP NULL
);

CREATE INDEX idx_system_media_categories_parent ON system_media_categories (parent_id);
CREATE INDEX idx_system_media_categories_name ON system_media_categories (name);
CREATE INDEX idx_system_media_categories_deleted_at ON system_media_categories (deleted_at);

CREATE TABLE IF NOT EXISTS system_media_assets (
    id BIGINT PRIMARY KEY,
    category_id BIGINT NOT NULL DEFAULT 0,
    display_name VARCHAR(255) NOT NULL,
    original_name VARCHAR(255) NOT NULL,
    storage_key VARCHAR(512) NOT NULL,
    url TEXT NOT NULL,
    mime_type VARCHAR(128) NOT NULL,
    extension VARCHAR(32) NOT NULL,
    size_bytes BIGINT NOT NULL,
    source VARCHAR(32) NOT NULL,
    external BOOLEAN NOT NULL DEFAULT FALSE,
    uploaded_by BIGINT NOT NULL DEFAULT 0,
    uploaded_by_username VARCHAR(128) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP NULL
);

CREATE INDEX idx_system_media_assets_category ON system_media_assets (category_id);
CREATE INDEX idx_system_media_assets_display_name ON system_media_assets (display_name);
CREATE INDEX idx_system_media_assets_source ON system_media_assets (source);
CREATE INDEX idx_system_media_assets_external ON system_media_assets (external);
CREATE INDEX idx_system_media_assets_created_at ON system_media_assets (created_at);
CREATE INDEX idx_system_media_assets_deleted_at ON system_media_assets (deleted_at);

-- +goose Down
DROP TABLE IF EXISTS system_media_assets;
DROP TABLE IF EXISTS system_media_categories;
