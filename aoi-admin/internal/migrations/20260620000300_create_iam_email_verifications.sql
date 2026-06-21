-- +goose Up
CREATE TABLE iam_email_verifications (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    org_id BIGINT NOT NULL,
    token_hash VARCHAR(128) NOT NULL UNIQUE,
    status VARCHAR(32) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    verified_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE INDEX idx_iam_email_verifications_user_id ON iam_email_verifications (user_id);
CREATE INDEX idx_iam_email_verifications_org_id ON iam_email_verifications (org_id);
CREATE INDEX idx_iam_email_verifications_status ON iam_email_verifications (status);
CREATE INDEX idx_iam_email_verifications_expires_at ON iam_email_verifications (expires_at);

-- +goose Down
DROP INDEX IF EXISTS idx_iam_email_verifications_expires_at;
DROP INDEX IF EXISTS idx_iam_email_verifications_status;
DROP INDEX IF EXISTS idx_iam_email_verifications_org_id;
DROP INDEX IF EXISTS idx_iam_email_verifications_user_id;
DROP TABLE IF EXISTS iam_email_verifications;
