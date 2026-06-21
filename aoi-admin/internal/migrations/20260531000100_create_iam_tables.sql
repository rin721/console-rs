-- +goose Up
CREATE TABLE IF NOT EXISTS iam_organizations (
    id BIGINT PRIMARY KEY,
    code VARCHAR(64) NOT NULL UNIQUE,
    name VARCHAR(128) NOT NULL,
    status VARCHAR(32) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP NULL
);

CREATE TABLE IF NOT EXISTS iam_users (
    id BIGINT PRIMARY KEY,
    username VARCHAR(64) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    display_name VARCHAR(128) NOT NULL,
    status VARCHAR(32) NOT NULL,
    mfa_enabled BOOLEAN NOT NULL DEFAULT false,
    failed_login_attempts INTEGER NOT NULL DEFAULT 0,
    locked_until TIMESTAMP NULL,
    last_login_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP NULL
);

CREATE TABLE IF NOT EXISTS iam_memberships (
    id BIGINT PRIMARY KEY,
    org_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    status VARCHAR(32) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP NULL,
    UNIQUE (org_id, user_id)
);

CREATE TABLE IF NOT EXISTS iam_roles (
    id BIGINT PRIMARY KEY,
    org_id BIGINT NOT NULL,
    code VARCHAR(64) NOT NULL,
    name VARCHAR(128) NOT NULL,
    description TEXT NOT NULL,
    system BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    deleted_at TIMESTAMP NULL,
    UNIQUE (org_id, code)
);

CREATE TABLE IF NOT EXISTS iam_permissions (
    id BIGINT PRIMARY KEY,
    code VARCHAR(128) NOT NULL UNIQUE,
    name VARCHAR(128) NOT NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS iam_sessions (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    org_id BIGINT NOT NULL,
    refresh_token_hash VARCHAR(128) NOT NULL UNIQUE,
    user_agent TEXT NOT NULL,
    ip_address VARCHAR(64) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    revoked_at TIMESTAMP NULL,
    last_used_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS iam_invitations (
    id BIGINT PRIMARY KEY,
    org_id BIGINT NOT NULL,
    email VARCHAR(255) NOT NULL,
    role_code VARCHAR(64) NOT NULL,
    token_hash VARCHAR(128) NOT NULL UNIQUE,
    status VARCHAR(32) NOT NULL,
    invited_by BIGINT NOT NULL,
    accepted_by BIGINT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS iam_password_resets (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    token_hash VARCHAR(128) NOT NULL UNIQUE,
    status VARCHAR(32) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    used_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS iam_mfa_factors (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    type VARCHAR(32) NOT NULL,
    secret TEXT NOT NULL,
    status VARCHAR(32) NOT NULL,
    confirmed_at TIMESTAMP NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS iam_audit_logs (
    id BIGINT PRIMARY KEY,
    org_id BIGINT NULL,
    user_id BIGINT NULL,
    action VARCHAR(128) NOT NULL,
    resource VARCHAR(128) NOT NULL,
    resource_id VARCHAR(128) NOT NULL,
    ip_address VARCHAR(64) NOT NULL,
    user_agent TEXT NOT NULL,
    metadata TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS iam_casbin_rules (
    id BIGINT PRIMARY KEY,
    ptype VARCHAR(8) NOT NULL,
    v0 VARCHAR(128) NOT NULL,
    v1 VARCHAR(128) NOT NULL,
    v2 VARCHAR(256) NOT NULL,
    v3 VARCHAR(256) NOT NULL,
    v4 VARCHAR(256) NOT NULL,
    v5 VARCHAR(256) NOT NULL,
    created_at TIMESTAMP NOT NULL,
    UNIQUE (ptype, v0, v1, v2, v3, v4, v5)
);

CREATE INDEX idx_iam_memberships_user ON iam_memberships (user_id);
CREATE INDEX idx_iam_sessions_user ON iam_sessions (user_id);
CREATE INDEX idx_iam_audit_logs_org_created ON iam_audit_logs (org_id, created_at);

-- +goose Down
DROP TABLE IF EXISTS iam_casbin_rules;
DROP TABLE IF EXISTS iam_audit_logs;
DROP TABLE IF EXISTS iam_mfa_factors;
DROP TABLE IF EXISTS iam_password_resets;
DROP TABLE IF EXISTS iam_invitations;
DROP TABLE IF EXISTS iam_sessions;
DROP TABLE IF EXISTS iam_permissions;
DROP TABLE IF EXISTS iam_roles;
DROP TABLE IF EXISTS iam_memberships;
DROP TABLE IF EXISTS iam_users;
DROP TABLE IF EXISTS iam_organizations;
