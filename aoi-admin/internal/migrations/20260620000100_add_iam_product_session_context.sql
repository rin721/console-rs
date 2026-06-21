-- +goose Up
ALTER TABLE iam_sessions ADD COLUMN product_code VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE iam_sessions ADD COLUMN client_type VARCHAR(32) NOT NULL DEFAULT '';
CREATE INDEX idx_iam_sessions_product_client ON iam_sessions (user_id, org_id, product_code, client_type, revoked_at);

ALTER TABLE iam_permissions ADD COLUMN product_code VARCHAR(64) NOT NULL DEFAULT '';
CREATE INDEX idx_iam_permissions_product_code ON iam_permissions (product_code);

ALTER TABLE system_apis ADD COLUMN product_code VARCHAR(64) NOT NULL DEFAULT '';
CREATE INDEX idx_system_apis_product_code ON system_apis (product_code);

-- +goose Down
DROP INDEX IF EXISTS idx_system_apis_product_code;
ALTER TABLE system_apis DROP COLUMN product_code;

DROP INDEX IF EXISTS idx_iam_permissions_product_code;
ALTER TABLE iam_permissions DROP COLUMN product_code;

DROP INDEX IF EXISTS idx_iam_sessions_product_client;
ALTER TABLE iam_sessions DROP COLUMN client_type;
ALTER TABLE iam_sessions DROP COLUMN product_code;
