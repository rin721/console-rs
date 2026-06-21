-- +goose Up
ALTER TABLE plugin_instances
    ADD COLUMN transport VARCHAR(32) NOT NULL DEFAULT '';

UPDATE plugin_instances
SET transport = protocol
WHERE transport = ''
  AND protocol IN ('http', 'websocket', 'rpc');

CREATE INDEX IF NOT EXISTS idx_plugin_instances_transport
    ON plugin_instances (transport);

-- +goose Down
DROP INDEX IF EXISTS idx_plugin_instances_transport;

