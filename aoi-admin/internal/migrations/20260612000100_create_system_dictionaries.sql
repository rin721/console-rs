-- +goose Up
CREATE TABLE IF NOT EXISTS system_dictionaries (
    id BIGINT PRIMARY KEY,
    code VARCHAR(128) NOT NULL,
    name VARCHAR(128) NOT NULL,
    description TEXT NOT NULL,
    status VARCHAR(32) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP NULL,
    UNIQUE (code)
);

CREATE TABLE IF NOT EXISTS system_dictionary_items (
    id BIGINT PRIMARY KEY,
    dictionary_id BIGINT NOT NULL,
    label VARCHAR(128) NOT NULL,
    value VARCHAR(128) NOT NULL,
    extra TEXT NOT NULL,
    status VARCHAR(32) NOT NULL,
    sort_order INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP NULL,
    UNIQUE (dictionary_id, value)
);

CREATE INDEX idx_system_dictionaries_status ON system_dictionaries (status);
CREATE INDEX idx_system_dictionary_items_dictionary ON system_dictionary_items (dictionary_id, sort_order);
CREATE INDEX idx_system_dictionary_items_status ON system_dictionary_items (status);

-- +goose Down
DROP TABLE IF EXISTS system_dictionary_items;
DROP TABLE IF EXISTS system_dictionaries;
