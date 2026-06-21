-- +goose Up
CREATE TABLE IF NOT EXISTS plugin_instances (
    plugin_id VARCHAR(128) NOT NULL,
    instance_id VARCHAR(128) NOT NULL,
    name VARCHAR(128) NOT NULL,
    version VARCHAR(64) NOT NULL,
    protocol VARCHAR(32) NOT NULL,
    endpoint TEXT NOT NULL,
    status VARCHAR(32) NOT NULL,
    runtime_status VARCHAR(32) NOT NULL DEFAULT '',
    schema_version VARCHAR(32) NOT NULL DEFAULT '',
    owner_host VARCHAR(255) NOT NULL DEFAULT '',
    lease_ttl_seconds INTEGER NOT NULL DEFAULT 30,
    lease_expires_at TIMESTAMP NOT NULL,
    registered_at TIMESTAMP NOT NULL,
    last_heartbeat_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    permissions_json TEXT NOT NULL,
    hooks_json TEXT NOT NULL,
    metadata_json TEXT NOT NULL,
    capabilities_json TEXT NOT NULL,
    PRIMARY KEY (plugin_id, instance_id)
);

CREATE INDEX IF NOT EXISTS idx_plugin_instances_plugin_status
    ON plugin_instances (plugin_id, status);
CREATE INDEX IF NOT EXISTS idx_plugin_instances_lease
    ON plugin_instances (status, lease_expires_at);
CREATE INDEX IF NOT EXISTS idx_plugin_instances_owner
    ON plugin_instances (owner_host);

CREATE TABLE IF NOT EXISTS plugin_instance_capabilities (
    plugin_id VARCHAR(128) NOT NULL,
    instance_id VARCHAR(128) NOT NULL,
    capability VARCHAR(255) NOT NULL,
    capability_version VARCHAR(64) NOT NULL DEFAULT '',
    scope VARCHAR(32) NOT NULL DEFAULT '',
    permissions_json TEXT NOT NULL,
    input_schema TEXT NOT NULL,
    output_schema TEXT NOT NULL,
    secret_policy VARCHAR(32) NOT NULL DEFAULT '',
    description TEXT NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    PRIMARY KEY (plugin_id, instance_id, capability)
);

CREATE INDEX IF NOT EXISTS idx_plugin_instance_capabilities_capability
    ON plugin_instance_capabilities (capability);

CREATE TABLE IF NOT EXISTS plugin_event_subscriptions (
    plugin_id VARCHAR(128) NOT NULL,
    instance_id VARCHAR(128) NOT NULL,
    event VARCHAR(255) NOT NULL,
    filters_json TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    PRIMARY KEY (plugin_id, instance_id, event)
);

CREATE INDEX IF NOT EXISTS idx_plugin_event_subscriptions_event
    ON plugin_event_subscriptions (event);

-- +goose Down
DROP TABLE IF EXISTS plugin_event_subscriptions;
DROP TABLE IF EXISTS plugin_instance_capabilities;
DROP TABLE IF EXISTS plugin_instances;
