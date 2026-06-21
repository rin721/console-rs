-- +goose Up
CREATE TABLE IF NOT EXISTS iam_api_tokens (
    id BIGINT PRIMARY KEY,
    org_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    role_code VARCHAR(64) NOT NULL,
    token_prefix VARCHAR(32) NOT NULL,
    token_hash VARCHAR(128) NOT NULL UNIQUE,
    status VARCHAR(32) NOT NULL,
    expires_at TIMESTAMP NULL,
    last_used_at TIMESTAMP NULL,
    last_used_ip_address VARCHAR(64) NOT NULL DEFAULT '',
    revoked_at TIMESTAMP NULL,
    revoked_by BIGINT NULL,
    remark TEXT NOT NULL,
    created_by BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_iam_api_tokens_org_created ON iam_api_tokens (org_id, created_at);
CREATE INDEX idx_iam_api_tokens_org_user ON iam_api_tokens (org_id, user_id);
CREATE INDEX idx_iam_api_tokens_status_expires ON iam_api_tokens (status, expires_at);

-- +goose Down
DROP TABLE IF EXISTS iam_api_tokens;
