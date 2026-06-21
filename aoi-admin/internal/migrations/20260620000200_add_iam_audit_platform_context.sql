-- +goose Up
ALTER TABLE iam_audit_logs ADD COLUMN product_code VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE iam_audit_logs ADD COLUMN client_type VARCHAR(32) NOT NULL DEFAULT '';
CREATE INDEX idx_iam_audit_logs_product_code ON iam_audit_logs (product_code);
CREATE INDEX idx_iam_audit_logs_client_type ON iam_audit_logs (client_type);

-- +goose Down
DROP INDEX IF EXISTS idx_iam_audit_logs_client_type;
DROP INDEX IF EXISTS idx_iam_audit_logs_product_code;
ALTER TABLE iam_audit_logs DROP COLUMN client_type;
ALTER TABLE iam_audit_logs DROP COLUMN product_code;
