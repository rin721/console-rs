-- +goose Up
ALTER TABLE iam_organizations ADD COLUMN kind VARCHAR(32) NOT NULL DEFAULT 'tenant';
CREATE INDEX idx_iam_organizations_kind ON iam_organizations (kind);
UPDATE iam_organizations
SET kind = 'platform'
WHERE id = (
    SELECT id FROM (
        SELECT MIN(id) AS id FROM iam_organizations
    ) AS platform_org
);

ALTER TABLE system_apis ADD COLUMN scope VARCHAR(32) NOT NULL DEFAULT 'tenant';
CREATE INDEX idx_system_apis_scope ON system_apis (scope);

CREATE TABLE iam_permissions_next (
    id BIGINT PRIMARY KEY,
    product_code VARCHAR(64) NOT NULL DEFAULT '',
    scope VARCHAR(32) NOT NULL DEFAULT 'tenant',
    code VARCHAR(128) NOT NULL,
    name VARCHAR(128) NOT NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    UNIQUE (product_code, scope, code)
);
INSERT INTO iam_permissions_next (id, product_code, scope, code, name, description, created_at, updated_at)
SELECT
    MIN(id),
    COALESCE(NULLIF(product_code, ''), 'aoi-admin') AS normalized_product_code,
    'tenant',
    code,
    MIN(name),
    MIN(description),
    MIN(created_at),
    MAX(updated_at)
FROM iam_permissions
GROUP BY COALESCE(NULLIF(product_code, ''), 'aoi-admin'), code;
DROP TABLE iam_permissions;
ALTER TABLE iam_permissions_next RENAME TO iam_permissions;
CREATE INDEX idx_iam_permissions_product_code ON iam_permissions (product_code);
CREATE INDEX idx_iam_permissions_scope ON iam_permissions (scope);

DELETE FROM iam_casbin_rules
WHERE ptype = 'p'
  AND v0 = 'role:owner'
  AND v2 = '*'
  AND v3 = '*';

UPDATE iam_casbin_rules
SET v5 = v3,
    v4 = v2,
    v3 = 'tenant',
    v2 = 'aoi-admin'
WHERE ptype = 'p'
  AND v4 = ''
  AND v5 = '';

-- +goose Down
UPDATE iam_casbin_rules
SET v2 = v4,
    v3 = v5,
    v4 = '',
    v5 = ''
WHERE ptype = 'p'
  AND v2 = 'aoi-admin'
  AND v3 IN ('platform', 'tenant', 'product');

CREATE TABLE iam_permissions_prev (
    id BIGINT PRIMARY KEY,
    product_code VARCHAR(64) NOT NULL DEFAULT '',
    code VARCHAR(128) NOT NULL UNIQUE,
    name VARCHAR(128) NOT NULL,
    description TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
INSERT INTO iam_permissions_prev (id, product_code, code, name, description, created_at, updated_at)
SELECT MIN(id), MIN(product_code), code, MIN(name), MIN(description), MIN(created_at), MAX(updated_at)
FROM iam_permissions
GROUP BY code;
DROP TABLE iam_permissions;
ALTER TABLE iam_permissions_prev RENAME TO iam_permissions;
CREATE INDEX idx_iam_permissions_product_code ON iam_permissions (product_code);

DROP INDEX IF EXISTS idx_system_apis_scope;
ALTER TABLE system_apis DROP COLUMN scope;

DROP INDEX IF EXISTS idx_iam_organizations_kind;
ALTER TABLE iam_organizations DROP COLUMN kind;
