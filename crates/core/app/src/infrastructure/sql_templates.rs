#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum SqlDialect {
    Sqlite,
    Postgres,
    Mysql,
}

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum InsertIdStrategy {
    ReturningId,
    DialectSpecificPostInsertRead,
}

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum InsertIdRead {
    ReturningIdInStatement,
    PostInsertQuery(&'static str),
}

impl SqlDialect {
    pub fn insert_id_strategy(self) -> InsertIdStrategy {
        match self {
            Self::Sqlite | Self::Postgres => InsertIdStrategy::ReturningId,
            Self::Mysql => InsertIdStrategy::DialectSpecificPostInsertRead,
        }
    }

    pub fn insert_id_read(self) -> InsertIdRead {
        match self {
            // SQLite 和 PostgreSQL 当前模板会把 ID 放在 insert ... returning id 结果集中。
            Self::Sqlite | Self::Postgres => InsertIdRead::ReturningIdInStatement,
            // MySQL 必须在同一连接/事务内读取 last_insert_id()，不能复用 returning id 语义。
            Self::Mysql => InsertIdRead::PostInsertQuery("select last_insert_id()"),
        }
    }

    pub fn role_permissions_for_role(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select p.code
                 from iam_role_permissions rp
                 join iam_permissions p on p.id = rp.permission_id
                 where rp.role_id = ?
                 order by p.scope asc, p.code asc"
            }
            Self::Postgres => {
                "select p.code
                 from iam_role_permissions rp
                 join iam_permissions p on p.id = rp.permission_id
                 where rp.role_id = $1
                 order by p.scope asc, p.code asc"
            }
        }
    }

    pub fn tenant_permission_id_by_code(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select id
                 from iam_permissions
                 where product_code = ? and code = ? and scope in ('tenant', 'product')
                 limit 1"
            }
            Self::Postgres => {
                "select id
                 from iam_permissions
                 where product_code = $1 and code = $2 and scope in ('tenant', 'product')
                 limit 1"
            }
        }
    }

    pub fn delete_role_permissions(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => "delete from iam_role_permissions where role_id = ?",
            Self::Postgres => "delete from iam_role_permissions where role_id = $1",
        }
    }

    pub fn role_permission_values(self) -> &'static str {
        match self {
            Self::Sqlite => {
                "insert into iam_role_permissions(role_id, permission_id)
                 values (?, ?)
                 on conflict(role_id, permission_id) do nothing"
            }
            Self::Postgres => {
                "insert into iam_role_permissions(role_id, permission_id)
                 values ($1, $2)
                 on conflict(role_id, permission_id) do nothing"
            }
            Self::Mysql => {
                "insert ignore into iam_role_permissions(role_id, permission_id)
                 values (?, ?)"
            }
        }
    }

    pub fn role_permissions_for_tenant_scopes(self) -> &'static str {
        match self {
            Self::Sqlite => {
                "insert into iam_role_permissions(role_id, permission_id)
                 select ?, id from iam_permissions
                 where product_code = ? and scope in ('tenant', 'product')
                 on conflict(role_id, permission_id) do nothing"
            }
            Self::Postgres => {
                "insert into iam_role_permissions(role_id, permission_id)
                 select $1, id from iam_permissions
                 where product_code = $2 and scope in ('tenant', 'product')
                 on conflict(role_id, permission_id) do nothing"
            }
            Self::Mysql => {
                "insert ignore into iam_role_permissions(role_id, permission_id)
                 select ?, id from iam_permissions
                 where product_code = ? and scope in ('tenant', 'product')"
            }
        }
    }

    pub fn role_permissions_for_platform_scope(self) -> &'static str {
        match self {
            Self::Sqlite => {
                "insert into iam_role_permissions(role_id, permission_id)
                 select ?, id from iam_permissions
                 where product_code = ? and scope = 'platform'
                 on conflict(role_id, permission_id) do nothing"
            }
            Self::Postgres => {
                "insert into iam_role_permissions(role_id, permission_id)
                 select $1, id from iam_permissions
                 where product_code = $2 and scope = 'platform'
                 on conflict(role_id, permission_id) do nothing"
            }
            Self::Mysql => {
                "insert ignore into iam_role_permissions(role_id, permission_id)
                 select ?, id from iam_permissions
                 where product_code = ? and scope = 'platform'"
            }
        }
    }

    pub fn setup_state_completed_upsert(self) -> &'static str {
        match self {
            Self::Sqlite => {
                "insert into setup_state(key, value, updated_at)
                 values ('completed', 1, ?)
                 on conflict(key) do update set value = excluded.value, updated_at = excluded.updated_at"
            }
            Self::Postgres => {
                "insert into setup_state(key, value, updated_at)
                 values ('completed', 1, $1)
                 on conflict(key) do update set value = excluded.value, updated_at = excluded.updated_at"
            }
            Self::Mysql => {
                "insert into setup_state(`key`, value, updated_at)
                 values ('completed', 1, ?)
                 on duplicate key update value = values(value), updated_at = values(updated_at)"
            }
        }
    }

    pub fn setup_completed_value(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Postgres => {
                "select value from setup_state where key = 'completed'"
            }
            Self::Mysql => "select value from setup_state where `key` = 'completed'",
        }
    }

    pub fn complete_setup_run(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "update setup_runs
                 set status = 'completed', updated_at = ?
                 where id = ?"
            }
            Self::Postgres => {
                "update setup_runs
                 set status = 'completed', updated_at = $1
                 where id = $2"
            }
        }
    }

    pub fn append_setup_complete_log(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "insert into setup_step_logs(run_id, step_key, status, message, created_at)
                 values (?, 'complete', 'ok', '初始化完成状态已写入', ?)"
            }
            Self::Postgres => {
                "insert into setup_step_logs(run_id, step_key, status, message, created_at)
                 values ($1, 'complete', 'ok', '初始化完成状态已写入', $2)"
            }
        }
    }

    pub fn create_setup_run(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "insert into setup_runs(id, status, reason, created_at, updated_at)
                 values (?, 'running', ?, ?, ?)"
            }
            Self::Postgres => {
                "insert into setup_runs(id, status, reason, created_at, updated_at)
                 values ($1, 'running', $2, $3, $4)"
            }
        }
    }

    pub fn append_setup_step_log(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "insert into setup_step_logs(run_id, step_key, status, message, created_at)
                 values (?, ?, ?, ?, ?)"
            }
            Self::Postgres => {
                "insert into setup_step_logs(run_id, step_key, status, message, created_at)
                 values ($1, $2, $3, $4, $5)"
            }
        }
    }

    pub fn setup_step_logs_for_run(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select step_key, status, message, created_at
                 from setup_step_logs
                 where run_id = ?
                 order by id asc"
            }
            Self::Postgres => {
                "select step_key, status, message, created_at
                 from setup_step_logs
                 where run_id = $1
                 order by id asc"
            }
        }
    }

    pub fn system_api_upsert(self) -> &'static str {
        match self {
            Self::Sqlite => {
                "insert into system_apis(id, method, path, tag, summary, access, permission, scope, product_code, created_at, updated_at)
                 values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                 on conflict(method, path) do update set
                   id = excluded.id,
                   tag = excluded.tag,
                   summary = excluded.summary,
                   access = excluded.access,
                   permission = excluded.permission,
                   scope = excluded.scope,
                   product_code = excluded.product_code,
                   updated_at = excluded.updated_at"
            }
            Self::Postgres => {
                "insert into system_apis(id, method, path, tag, summary, access, permission, scope, product_code, created_at, updated_at)
                 values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
                 on conflict(method, path) do update set
                   id = excluded.id,
                   tag = excluded.tag,
                   summary = excluded.summary,
                   access = excluded.access,
                   permission = excluded.permission,
                   scope = excluded.scope,
                   product_code = excluded.product_code,
                   updated_at = excluded.updated_at"
            }
            Self::Mysql => {
                "insert into system_apis(id, method, path, tag, summary, access, permission, scope, product_code, created_at, updated_at)
                 values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                 on duplicate key update
                   id = values(id),
                   tag = values(tag),
                   summary = values(summary),
                   access = values(access),
                   permission = values(permission),
                   scope = values(scope),
                   product_code = values(product_code),
                   updated_at = values(updated_at)"
            }
        }
    }

    pub fn system_apis_list(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Postgres | Self::Mysql => {
                "select id, method, path, tag, summary, access, permission, scope, product_code
                 from system_apis
                 order by tag asc, path asc, method asc"
            }
        }
    }

    pub fn permission_upsert(self) -> &'static str {
        match self {
            Self::Sqlite => {
                "insert into iam_permissions(product_code, scope, code, name, created_at)
                 values (?, ?, ?, ?, ?)
                 on conflict(product_code, scope, code) do update set name = excluded.name"
            }
            Self::Postgres => {
                "insert into iam_permissions(product_code, scope, code, name, created_at)
                 values ($1, $2, $3, $4, $5)
                 on conflict(product_code, scope, code) do update set name = excluded.name"
            }
            Self::Mysql => {
                "insert into iam_permissions(product_code, scope, code, name, created_at)
                 values (?, ?, ?, ?, ?)
                 on duplicate key update name = values(name)"
            }
        }
    }

    pub fn permissions_list(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select id, product_code, scope, code, name
                 from iam_permissions
                 where product_code = ?
                 order by scope asc, code asc"
            }
            Self::Postgres => {
                "select id, product_code, scope, code, name
                 from iam_permissions
                 where product_code = $1
                 order by scope asc, code asc"
            }
        }
    }

    pub fn platform_builtin_role_permissions(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Postgres => {
                "insert into iam_role_permissions(role_id, permission_id)
                 select r.id, p.id
                 from iam_roles r
                 join iam_permissions p on p.scope = 'platform'
                 where r.system_builtin = 1 and r.scope = 'platform' and r.org_id is null
                 on conflict(role_id, permission_id) do nothing"
            }
            Self::Mysql => {
                "insert ignore into iam_role_permissions(role_id, permission_id)
                 select r.id, p.id
                 from iam_roles r
                 join iam_permissions p on p.scope = 'platform'
                 where r.system_builtin = 1 and r.scope = 'platform' and r.org_id is null"
            }
        }
    }

    pub fn tenant_builtin_role_permissions(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Postgres => {
                "insert into iam_role_permissions(role_id, permission_id)
                 select r.id, p.id
                 from iam_roles r
                 join iam_permissions p on p.scope in ('tenant', 'product')
                 where r.system_builtin = 1 and r.scope = 'tenant' and r.org_id is not null
                 on conflict(role_id, permission_id) do nothing"
            }
            Self::Mysql => {
                "insert ignore into iam_role_permissions(role_id, permission_id)
                 select r.id, p.id
                 from iam_roles r
                 join iam_permissions p on p.scope in ('tenant', 'product')
                 where r.system_builtin = 1 and r.scope = 'tenant' and r.org_id is not null"
            }
        }
    }

    pub fn create_tenant_organization(self) -> &'static str {
        match self {
            Self::Sqlite => {
                "insert into iam_organizations(code, name, scope, status, created_at, updated_at)
                 values (?, ?, 'tenant', 'active', ?, ?)
                 returning id"
            }
            Self::Postgres => {
                "insert into iam_organizations(code, name, scope, status, created_at, updated_at)
                 values ($1, $2, 'tenant', 'active', $3, $4)
                 returning id"
            }
            Self::Mysql => {
                "insert into iam_organizations(code, name, scope, status, created_at, updated_at)
                 values (?, ?, 'tenant', 'active', ?, ?)"
            }
        }
    }

    pub fn create_active_user(self) -> &'static str {
        match self {
            Self::Sqlite => {
                "insert into iam_users(email, display_name, password_hash, status, email_verified_at, created_at, updated_at)
                 values (?, ?, ?, 'active', ?, ?, ?)
                 returning id"
            }
            Self::Postgres => {
                "insert into iam_users(email, display_name, password_hash, status, email_verified_at, created_at, updated_at)
                 values ($1, $2, $3, 'active', $4, $5, $6)
                 returning id"
            }
            Self::Mysql => {
                "insert into iam_users(email, display_name, password_hash, status, email_verified_at, created_at, updated_at)
                 values (?, ?, ?, 'active', ?, ?, ?)"
            }
        }
    }

    pub fn create_pending_verification_user(self) -> &'static str {
        match self {
            Self::Sqlite => {
                "insert into iam_users(email, display_name, password_hash, status, email_verified_at, created_at, updated_at)
                 values (?, ?, ?, 'pending_verification', null, ?, ?)
                 returning id"
            }
            Self::Postgres => {
                "insert into iam_users(email, display_name, password_hash, status, email_verified_at, created_at, updated_at)
                 values ($1, $2, $3, 'pending_verification', null, $4, $5)
                 returning id"
            }
            Self::Mysql => {
                "insert into iam_users(email, display_name, password_hash, status, email_verified_at, created_at, updated_at)
                 values (?, ?, ?, 'pending_verification', null, ?, ?)"
            }
        }
    }

    pub fn create_active_membership(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "insert into iam_memberships(org_id, user_id, role_code, status, created_at, updated_at)
                 values (?, ?, ?, 'active', ?, ?)"
            }
            Self::Postgres => {
                "insert into iam_memberships(org_id, user_id, role_code, status, created_at, updated_at)
                 values ($1, $2, $3, 'active', $4, $5)"
            }
        }
    }

    pub fn users_count(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Postgres | Self::Mysql => "select count(*) from iam_users",
        }
    }

    pub fn user_email_count(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select count(*)
                 from iam_users
                 where lower(email) = lower(?) and deleted_at is null"
            }
            Self::Postgres => {
                "select count(*)
                 from iam_users
                 where lower(email) = lower($1) and deleted_at is null"
            }
        }
    }

    pub fn organization_code_count(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => "select count(*) from iam_organizations where code = ?",
            Self::Postgres => "select count(*) from iam_organizations where code = $1",
        }
    }

    pub fn user_by_identifier(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select id, email, display_name, password_hash, status
                 from iam_users
                 where lower(email) = lower(?) and deleted_at is null"
            }
            Self::Postgres => {
                "select id, email, display_name, password_hash, status
                 from iam_users
                 where lower(email) = lower($1) and deleted_at is null"
            }
        }
    }

    pub fn primary_organization_for_user(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select o.id, o.code, o.name, o.scope
                 from iam_organizations o
                 join iam_memberships m on m.org_id = o.id
                 where m.user_id = ? and m.status = 'active' and o.deleted_at is null
                 order by o.id asc
                 limit 1"
            }
            Self::Postgres => {
                "select o.id, o.code, o.name, o.scope
                 from iam_organizations o
                 join iam_memberships m on m.org_id = o.id
                 where m.user_id = $1 and m.status = 'active' and o.deleted_at is null
                 order by o.id asc
                 limit 1"
            }
        }
    }

    pub fn organizations_list(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Postgres | Self::Mysql => {
                "select id, code, name, scope, status, created_at
                 from iam_organizations
                 where deleted_at is null
                 order by scope asc, code asc"
            }
        }
    }

    pub fn org_users_list(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select u.id, u.email, u.display_name, u.status, u.email_verified_at,
                    group_concat(distinct m.role_code) as role_codes
                 from iam_memberships m
                 join iam_users u on u.id = m.user_id
                 where m.org_id = ? and m.status = 'active' and u.deleted_at is null
                 group by u.id, u.email, u.display_name, u.status, u.email_verified_at
                 order by u.id asc"
            }
            Self::Postgres => {
                "select u.id, u.email, u.display_name, u.status, u.email_verified_at,
                    string_agg(distinct m.role_code, ',') as role_codes
                 from iam_memberships m
                 join iam_users u on u.id = m.user_id
                 where m.org_id = $1 and m.status = 'active' and u.deleted_at is null
                 group by u.id, u.email, u.display_name, u.status, u.email_verified_at
                 order by u.id asc"
            }
        }
    }

    pub fn org_user_membership_context(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select u.id, u.email, u.display_name, u.status, u.email_verified_at,
                    case when exists(
                      select 1 from iam_memberships m
                      where m.org_id = ? and m.user_id = u.id and m.status = 'active'
                    ) then 1 else 0 end as in_org,
                    case when exists(
                      select 1 from iam_memberships m
                      where m.org_id = ? and m.user_id = u.id and m.role_code = 'owner' and m.status = 'active'
                    ) then 1 else 0 end as is_owner
                 from iam_users u
                 where u.id = ? and u.deleted_at is null
                 limit 1"
            }
            Self::Postgres => {
                "select u.id, u.email, u.display_name, u.status, u.email_verified_at,
                    case when exists(
                      select 1 from iam_memberships m
                      where m.org_id = $1 and m.user_id = u.id and m.status = 'active'
                    ) then 1 else 0 end as in_org,
                    case when exists(
                      select 1 from iam_memberships m
                      where m.org_id = $2 and m.user_id = u.id and m.role_code = 'owner' and m.status = 'active'
                    ) then 1 else 0 end as is_owner
                 from iam_users u
                 where u.id = $3 and u.deleted_at is null
                 limit 1"
            }
        }
    }

    pub fn org_active_owner_count_except_user(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select count(distinct m.user_id)
                 from iam_memberships m
                 join iam_users u on u.id = m.user_id
                 where m.org_id = ?
                   and m.role_code = 'owner'
                   and m.status = 'active'
                   and u.status = 'active'
                   and u.deleted_at is null
                   and m.user_id != ?"
            }
            Self::Postgres => {
                "select count(distinct m.user_id)
                 from iam_memberships m
                 join iam_users u on u.id = m.user_id
                 where m.org_id = $1
                   and m.role_code = 'owner'
                   and m.status = 'active'
                   and u.status = 'active'
                   and u.deleted_at is null
                   and m.user_id != $2"
            }
        }
    }

    pub fn update_org_user_profile_status(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "update iam_users set display_name = ?, status = ?, updated_at = ? where id = ?"
            }
            Self::Postgres => {
                "update iam_users set display_name = $1, status = $2, updated_at = $3 where id = $4"
            }
        }
    }

    pub fn delete_org_user_memberships(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "delete from iam_memberships where org_id = ? and user_id = ?"
            }
            Self::Postgres => "delete from iam_memberships where org_id = $1 and user_id = $2",
        }
    }

    pub fn create_audit_log(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "insert into iam_audit_logs(org_id, user_id, action, scope, product_code, detail, created_at)
                 values (?, ?, ?, ?, ?, ?, ?)"
            }
            Self::Postgres => {
                "insert into iam_audit_logs(org_id, user_id, action, scope, product_code, detail, created_at)
                 values ($1, $2, $3, $4, $5, $6, $7)"
            }
        }
    }

    pub fn create_tenant_owner_role(self) -> &'static str {
        match self {
            Self::Sqlite => {
                "insert into iam_roles(org_id, code, name, scope, system_builtin, created_at, updated_at)
                 values (?, 'owner', '组织所有者', 'tenant', 1, ?, ?)
                 returning id"
            }
            Self::Postgres => {
                "insert into iam_roles(org_id, code, name, scope, system_builtin, created_at, updated_at)
                 values ($1, 'owner', '组织所有者', 'tenant', 1, $2, $3)
                 returning id"
            }
            Self::Mysql => {
                "insert into iam_roles(org_id, code, name, scope, system_builtin, created_at, updated_at)
                 values (?, 'owner', '组织所有者', 'tenant', 1, ?, ?)"
            }
        }
    }

    pub fn create_platform_owner_role(self) -> &'static str {
        match self {
            Self::Sqlite => {
                "insert into iam_roles(org_id, code, name, scope, system_builtin, created_at, updated_at)
                 values (null, 'platform_owner', '平台所有者', 'platform', 1, ?, ?)
                 returning id"
            }
            Self::Postgres => {
                "insert into iam_roles(org_id, code, name, scope, system_builtin, created_at, updated_at)
                 values (null, 'platform_owner', '平台所有者', 'platform', 1, $1, $2)
                 returning id"
            }
            Self::Mysql => {
                "insert into iam_roles(org_id, code, name, scope, system_builtin, created_at, updated_at)
                 values (null, 'platform_owner', '平台所有者', 'platform', 1, ?, ?)"
            }
        }
    }

    pub fn create_org_role(self) -> &'static str {
        match self {
            Self::Sqlite => {
                "insert into iam_roles(org_id, code, name, scope, system_builtin, created_at, updated_at)
                 values (?, ?, ?, 'tenant', 0, ?, ?)
                 returning id"
            }
            Self::Postgres => {
                "insert into iam_roles(org_id, code, name, scope, system_builtin, created_at, updated_at)
                 values ($1, $2, $3, 'tenant', 0, $4, $5)
                 returning id"
            }
            Self::Mysql => {
                "insert into iam_roles(org_id, code, name, scope, system_builtin, created_at, updated_at)
                 values (?, ?, ?, 'tenant', 0, ?, ?)"
            }
        }
    }

    pub fn org_roles_list(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select id, org_id, code, name, scope, system_builtin
                 from iam_roles
                 where org_id = ?
                 order by system_builtin desc, code asc"
            }
            Self::Postgres => {
                "select id, org_id, code, name, scope, system_builtin
                 from iam_roles
                 where org_id = $1
                 order by system_builtin desc, code asc"
            }
        }
    }

    pub fn org_role_code_count(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select count(*)
                 from iam_roles
                 where org_id = ? and code = ?"
            }
            Self::Postgres => {
                "select count(*)
                 from iam_roles
                 where org_id = $1 and code = $2"
            }
        }
    }

    pub fn tenant_org_role_code_count(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select count(*)
                 from iam_roles
                 where org_id = ? and scope = 'tenant' and code = ?"
            }
            Self::Postgres => {
                "select count(*)
                 from iam_roles
                 where org_id = $1 and scope = 'tenant' and code = $2"
            }
        }
    }

    pub fn tenant_org_role_by_id(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select id, code, system_builtin
                 from iam_roles
                 where id = ? and org_id = ? and scope = 'tenant'
                 limit 1"
            }
            Self::Postgres => {
                "select id, code, system_builtin
                 from iam_roles
                 where id = $1 and org_id = $2 and scope = 'tenant'
                 limit 1"
            }
        }
    }

    pub fn update_org_role_name(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "update iam_roles set name = ?, updated_at = ? where id = ?"
            }
            Self::Postgres => "update iam_roles set name = $1, updated_at = $2 where id = $3",
        }
    }

    pub fn org_role_active_member_count(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select count(*)
                 from iam_memberships
                 where org_id = ? and role_code = ? and status = 'active'"
            }
            Self::Postgres => {
                "select count(*)
                 from iam_memberships
                 where org_id = $1 and role_code = $2 and status = 'active'"
            }
        }
    }

    pub fn org_role_pending_invitation_count(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select count(*)
                 from iam_invitations
                 where org_id = ? and role_code = ? and status = 'pending'"
            }
            Self::Postgres => {
                "select count(*)
                 from iam_invitations
                 where org_id = $1 and role_code = $2 and status = 'pending'"
            }
        }
    }

    pub fn delete_org_role(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => "delete from iam_roles where id = ? and org_id = ?",
            Self::Postgres => "delete from iam_roles where id = $1 and org_id = $2",
        }
    }

    pub fn create_api_token(self) -> &'static str {
        match self {
            Self::Sqlite => {
                "insert into iam_api_tokens(org_id, user_id, token_hash, token_prefix, status, expires_at, created_at)
                 values (?, ?, ?, ?, 'active', ?, ?)
                 returning id"
            }
            Self::Postgres => {
                "insert into iam_api_tokens(org_id, user_id, token_hash, token_prefix, status, expires_at, created_at)
                 values ($1, $2, $3, $4, 'active', $5, $6)
                 returning id"
            }
            Self::Mysql => {
                "insert into iam_api_tokens(org_id, user_id, token_hash, token_prefix, status, expires_at, created_at)
                 values (?, ?, ?, ?, 'active', ?, ?)"
            }
        }
    }

    pub fn create_invitation(self) -> &'static str {
        match self {
            Self::Sqlite => {
                "insert into iam_invitations(org_id, email, role_code, token_hash, status, expires_at, created_at)
                 values (?, ?, ?, ?, 'pending', ?, ?)
                 returning id"
            }
            Self::Postgres => {
                "insert into iam_invitations(org_id, email, role_code, token_hash, status, expires_at, created_at)
                 values ($1, $2, $3, $4, 'pending', $5, $6)
                 returning id"
            }
            Self::Mysql => {
                "insert into iam_invitations(org_id, email, role_code, token_hash, status, expires_at, created_at)
                 values (?, ?, ?, ?, 'pending', ?, ?)"
            }
        }
    }

    pub fn create_password_reset(self) -> &'static str {
        match self {
            Self::Sqlite => {
                "insert into iam_password_resets(user_id, token_hash, status, expires_at, created_at)
                 values (?, ?, 'pending', ?, ?)
                 returning id"
            }
            Self::Postgres => {
                "insert into iam_password_resets(user_id, token_hash, status, expires_at, created_at)
                 values ($1, $2, 'pending', $3, $4)
                 returning id"
            }
            Self::Mysql => {
                "insert into iam_password_resets(user_id, token_hash, status, expires_at, created_at)
                 values (?, ?, 'pending', ?, ?)"
            }
        }
    }

    pub fn create_email_verification(self) -> &'static str {
        match self {
            Self::Sqlite => {
                "insert into iam_email_verifications(user_id, email, token_hash, status, expires_at, created_at)
                 values (?, ?, ?, 'pending', ?, ?)
                 returning id"
            }
            Self::Postgres => {
                "insert into iam_email_verifications(user_id, email, token_hash, status, expires_at, created_at)
                 values ($1, $2, $3, 'pending', $4, $5)
                 returning id"
            }
            Self::Mysql => {
                "insert into iam_email_verifications(user_id, email, token_hash, status, expires_at, created_at)
                 values (?, ?, ?, 'pending', ?, ?)"
            }
        }
    }

    pub fn invitations_for_org(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select id, org_id, email, role_code, status, expires_at, created_at, accepted_at, revoked_at
                 from iam_invitations
                 where org_id = ?
                 order by id desc"
            }
            Self::Postgres => {
                "select id, org_id, email, role_code, status, expires_at, created_at, accepted_at, revoked_at
                 from iam_invitations
                 where org_id = $1
                 order by id desc"
            }
        }
    }

    pub fn revoke_invitation(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "update iam_invitations
                 set status = 'revoked', revoked_at = ?
                 where org_id = ? and id = ? and status = 'pending'"
            }
            Self::Postgres => {
                "update iam_invitations
                 set status = 'revoked', revoked_at = $1
                 where org_id = $2 and id = $3 and status = 'pending'"
            }
        }
    }

    pub fn invitation_by_hash(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select id, org_id, email, role_code, status, expires_at, accepted_at, revoked_at
                 from iam_invitations
                 where token_hash = ?
                 limit 1"
            }
            Self::Postgres => {
                "select id, org_id, email, role_code, status, expires_at, accepted_at, revoked_at
                 from iam_invitations
                 where token_hash = $1
                 limit 1"
            }
        }
    }

    pub fn organization_by_id(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select id, code, name, scope
                 from iam_organizations
                 where id = ? and deleted_at is null
                 limit 1"
            }
            Self::Postgres => {
                "select id, code, name, scope
                 from iam_organizations
                 where id = $1 and deleted_at is null
                 limit 1"
            }
        }
    }

    pub fn accept_invitation(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "update iam_invitations
                 set status = 'accepted', accepted_at = ?
                 where id = ? and status = 'pending' and accepted_at is null and revoked_at is null"
            }
            Self::Postgres => {
                "update iam_invitations
                 set status = 'accepted', accepted_at = $1
                 where id = $2 and status = 'pending' and accepted_at is null and revoked_at is null"
            }
        }
    }

    pub fn password_reset_by_hash(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select id, user_id, status, expires_at, used_at
                 from iam_password_resets
                 where token_hash = ?
                 limit 1"
            }
            Self::Postgres => {
                "select id, user_id, status, expires_at, used_at
                 from iam_password_resets
                 where token_hash = $1
                 limit 1"
            }
        }
    }

    pub fn mark_password_reset_used(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "update iam_password_resets
                 set status = 'used', used_at = ?
                 where id = ? and user_id = ? and status = 'pending' and used_at is null"
            }
            Self::Postgres => {
                "update iam_password_resets
                 set status = 'used', used_at = $1
                 where id = $2 and user_id = $3 and status = 'pending' and used_at is null"
            }
        }
    }

    pub fn update_user_password_hash(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "update iam_users set password_hash = ?, updated_at = ? where id = ?"
            }
            Self::Postgres => {
                "update iam_users set password_hash = $1, updated_at = $2 where id = $3"
            }
        }
    }

    pub fn email_verification_by_hash(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select id, user_id, status, expires_at, verified_at
                 from iam_email_verifications
                 where token_hash = ?
                 limit 1"
            }
            Self::Postgres => {
                "select id, user_id, status, expires_at, verified_at
                 from iam_email_verifications
                 where token_hash = $1
                 limit 1"
            }
        }
    }

    pub fn mark_email_verification_verified(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "update iam_email_verifications
                 set status = 'verified', verified_at = ?
                 where id = ? and user_id = ? and status = 'pending' and verified_at is null"
            }
            Self::Postgres => {
                "update iam_email_verifications
                 set status = 'verified', verified_at = $1
                 where id = $2 and user_id = $3 and status = 'pending' and verified_at is null"
            }
        }
    }

    pub fn mark_user_email_verified(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "update iam_users
                 set email_verified_at = ?,
                     status = case when status = 'pending_verification' then 'active' else status end,
                     updated_at = ?
                 where id = ?"
            }
            Self::Postgres => {
                "update iam_users
                 set email_verified_at = $1,
                     status = case when status = 'pending_verification' then 'active' else status end,
                     updated_at = $2
                 where id = $3"
            }
        }
    }

    pub fn create_mfa_factor(self) -> &'static str {
        match self {
            Self::Sqlite => {
                "insert into iam_mfa_factors(user_id, kind, secret_ciphertext, status, created_at)
                 values (?, ?, ?, 'pending', ?)
                 returning id"
            }
            Self::Postgres => {
                "insert into iam_mfa_factors(user_id, kind, secret_ciphertext, status, created_at)
                 values ($1, $2, $3, 'pending', $4)
                 returning id"
            }
            Self::Mysql => {
                "insert into iam_mfa_factors(user_id, kind, secret_ciphertext, status, created_at)
                 values (?, ?, ?, 'pending', ?)"
            }
        }
    }

    pub fn revoke_pending_mfa_factors(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "update iam_mfa_factors
                 set status = 'revoked', revoked_at = ?
                 where user_id = ? and kind = ? and status = 'pending' and revoked_at is null"
            }
            Self::Postgres => {
                "update iam_mfa_factors
                 set status = 'revoked', revoked_at = $1
                 where user_id = $2 and kind = $3 and status = 'pending' and revoked_at is null"
            }
        }
    }

    pub fn mfa_factors_for_user(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select id, kind, status, created_at, verified_at, revoked_at
                 from iam_mfa_factors
                 where user_id = ? and kind = 'totp'
                 order by id desc"
            }
            Self::Postgres => {
                "select id, kind, status, created_at, verified_at, revoked_at
                 from iam_mfa_factors
                 where user_id = $1 and kind = 'totp'
                 order by id desc"
            }
        }
    }

    pub fn pending_mfa_factor_for_user(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select id, user_id, kind, secret_ciphertext, status, created_at, verified_at, revoked_at
                 from iam_mfa_factors
                 where user_id = ? and kind = 'totp' and status = 'pending' and revoked_at is null
                 order by id desc
                 limit 1"
            }
            Self::Postgres => {
                "select id, user_id, kind, secret_ciphertext, status, created_at, verified_at, revoked_at
                 from iam_mfa_factors
                 where user_id = $1 and kind = 'totp' and status = 'pending' and revoked_at is null
                 order by id desc
                 limit 1"
            }
        }
    }

    pub fn verified_mfa_factor_for_user(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select id, user_id, kind, secret_ciphertext, status, created_at, verified_at, revoked_at
                 from iam_mfa_factors
                 where user_id = ? and kind = 'totp' and status = 'active' and verified_at is not null and revoked_at is null
                 order by id desc
                 limit 1"
            }
            Self::Postgres => {
                "select id, user_id, kind, secret_ciphertext, status, created_at, verified_at, revoked_at
                 from iam_mfa_factors
                 where user_id = $1 and kind = 'totp' and status = 'active' and verified_at is not null and revoked_at is null
                 order by id desc
                 limit 1"
            }
        }
    }

    pub fn activate_mfa_factor(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "update iam_mfa_factors
                 set status = 'active', verified_at = ?
                 where id = ? and user_id = ? and kind = 'totp' and status = 'pending' and revoked_at is null"
            }
            Self::Postgres => {
                "update iam_mfa_factors
                 set status = 'active', verified_at = $1
                 where id = $2 and user_id = $3 and kind = 'totp' and status = 'pending' and revoked_at is null"
            }
        }
    }

    pub fn revoke_active_mfa_recovery_codes(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "update iam_mfa_recovery_codes
                 set status = 'revoked', revoked_at = ?
                 where user_id = ? and status = 'active'"
            }
            Self::Postgres => {
                "update iam_mfa_recovery_codes
                 set status = 'revoked', revoked_at = $1
                 where user_id = $2 and status = 'active'"
            }
        }
    }

    pub fn revoke_other_mfa_factors(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "update iam_mfa_factors
                 set status = 'revoked', revoked_at = ?
                 where user_id = ? and kind = 'totp' and id != ? and revoked_at is null"
            }
            Self::Postgres => {
                "update iam_mfa_factors
                 set status = 'revoked', revoked_at = $1
                 where user_id = $2 and kind = 'totp' and id != $3 and revoked_at is null"
            }
        }
    }

    pub fn revoke_mfa_factor(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "update iam_mfa_factors
                 set status = 'revoked', revoked_at = ?
                 where id = ? and user_id = ? and revoked_at is null"
            }
            Self::Postgres => {
                "update iam_mfa_factors
                 set status = 'revoked', revoked_at = $1
                 where id = $2 and user_id = $3 and revoked_at is null"
            }
        }
    }

    pub fn create_mfa_recovery_code(self) -> &'static str {
        match self {
            Self::Sqlite => {
                "insert into iam_mfa_recovery_codes(user_id, code_hash, code_prefix, status, created_at)
                 values (?, ?, ?, 'active', ?)
                 returning id"
            }
            Self::Postgres => {
                "insert into iam_mfa_recovery_codes(user_id, code_hash, code_prefix, status, created_at)
                 values ($1, $2, $3, 'active', $4)
                 returning id"
            }
            Self::Mysql => {
                "insert into iam_mfa_recovery_codes(user_id, code_hash, code_prefix, status, created_at)
                 values (?, ?, ?, 'active', ?)"
            }
        }
    }

    pub fn mfa_recovery_codes_for_user(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select id, code_prefix, status, created_at, used_at, revoked_at
                 from iam_mfa_recovery_codes
                 where user_id = ?
                 order by id desc"
            }
            Self::Postgres => {
                "select id, code_prefix, status, created_at, used_at, revoked_at
                 from iam_mfa_recovery_codes
                 where user_id = $1
                 order by id desc"
            }
        }
    }

    pub fn consume_mfa_recovery_code(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "update iam_mfa_recovery_codes
                 set status = 'used', used_at = ?
                 where user_id = ? and code_hash = ? and status = 'active'"
            }
            Self::Postgres => {
                "update iam_mfa_recovery_codes
                 set status = 'used', used_at = $1
                 where user_id = $2 and code_hash = $3 and status = 'active'"
            }
        }
    }

    pub fn system_menu_upsert(self) -> &'static str {
        match self {
            Self::Sqlite => {
                "insert into system_menus(code, title, path, permission, scope, sort_order, created_at)
                 values (?, ?, ?, ?, ?, ?, ?)
                 on conflict(code) do update set
                   title = excluded.title,
                   path = excluded.path,
                   permission = excluded.permission,
                   scope = excluded.scope,
                   sort_order = excluded.sort_order"
            }
            Self::Postgres => {
                "insert into system_menus(code, title, path, permission, scope, sort_order, created_at)
                 values ($1, $2, $3, $4, $5, $6, $7)
                 on conflict(code) do update set
                   title = excluded.title,
                   path = excluded.path,
                   permission = excluded.permission,
                   scope = excluded.scope,
                   sort_order = excluded.sort_order"
            }
            Self::Mysql => {
                "insert into system_menus(code, title, path, permission, scope, sort_order, created_at)
                 values (?, ?, ?, ?, ?, ?, ?)
                 on duplicate key update
                   title = values(title),
                   path = values(path),
                   permission = values(permission),
                   scope = values(scope),
                   sort_order = values(sort_order)"
            }
        }
    }

    pub fn system_menus_list(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Postgres | Self::Mysql => {
                "select code, title, path, permission, scope, sort_order
                 from system_menus
                 order by sort_order asc, code asc"
            }
        }
    }

    pub fn system_config_upsert(self) -> &'static str {
        match self {
            Self::Sqlite => {
                "insert into system_configs(key, value_json, updated_at)
                 values (?, ?, ?)
                 on conflict(key) do update set
                   value_json = excluded.value_json,
                   updated_at = excluded.updated_at"
            }
            Self::Postgres => {
                "insert into system_configs(key, value_json, updated_at)
                 values ($1, $2, $3)
                 on conflict(key) do update set
                   value_json = excluded.value_json,
                   updated_at = excluded.updated_at"
            }
            Self::Mysql => {
                "insert into system_configs(`key`, value_json, updated_at)
                 values (?, ?, ?)
                 on duplicate key update
                   value_json = values(value_json),
                   updated_at = values(updated_at)"
            }
        }
    }

    pub fn system_configs_list(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Postgres => {
                "select key, value_json, updated_at
                 from system_configs
                 order by key asc"
            }
            Self::Mysql => {
                "select `key` as `key`, value_json, updated_at
                 from system_configs
                 order by `key` asc"
            }
        }
    }

    pub fn delete_system_config(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => "delete from system_configs where key = ?",
            Self::Postgres => "delete from system_configs where key = $1",
        }
    }

    pub fn system_dictionary_upsert(self) -> &'static str {
        match self {
            Self::Sqlite => {
                "insert into system_dictionaries(code, name, created_at, deleted_at)
                 values (?, ?, ?, null)
                 on conflict(code) do update set
                   name = excluded.name,
                   deleted_at = null"
            }
            Self::Postgres => {
                "insert into system_dictionaries(code, name, created_at, deleted_at)
                 values ($1, $2, $3, null)
                 on conflict(code) do update set
                   name = excluded.name,
                   deleted_at = null"
            }
            Self::Mysql => {
                "insert into system_dictionaries(code, name, created_at, deleted_at)
                 values (?, ?, ?, null)
                 on duplicate key update
                   name = values(name),
                   deleted_at = null"
            }
        }
    }

    pub fn system_dictionaries_list(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Postgres | Self::Mysql => {
                "select id, code, name, created_at
                 from system_dictionaries
                 where deleted_at is null
                 order by code asc"
            }
        }
    }

    pub fn system_dictionary_by_code(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select id, code, name, created_at
                 from system_dictionaries
                 where code = ? and deleted_at is null
                 limit 1"
            }
            Self::Postgres => {
                "select id, code, name, created_at
                 from system_dictionaries
                 where code = $1 and deleted_at is null
                 limit 1"
            }
        }
    }

    pub fn delete_system_dictionary(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "update system_dictionaries
                 set deleted_at = ?
                 where code = ? and deleted_at is null"
            }
            Self::Postgres => {
                "update system_dictionaries
                 set deleted_at = $1
                 where code = $2 and deleted_at is null"
            }
        }
    }

    pub fn system_parameter_upsert(self) -> &'static str {
        match self {
            Self::Sqlite => {
                "insert into system_parameters(key, name, value, created_at, updated_at, deleted_at)
                 values (?, ?, ?, ?, ?, null)
                 on conflict(key) do update set
                   name = excluded.name,
                   value = excluded.value,
                   updated_at = excluded.updated_at,
                   deleted_at = null"
            }
            Self::Postgres => {
                "insert into system_parameters(key, name, value, created_at, updated_at, deleted_at)
                 values ($1, $2, $3, $4, $5, null)
                 on conflict(key) do update set
                   name = excluded.name,
                   value = excluded.value,
                   updated_at = excluded.updated_at,
                   deleted_at = null"
            }
            Self::Mysql => {
                "insert into system_parameters(`key`, name, value, created_at, updated_at, deleted_at)
                 values (?, ?, ?, ?, ?, null)
                 on duplicate key update
                   name = values(name),
                   value = values(value),
                   updated_at = values(updated_at),
                   deleted_at = null"
            }
        }
    }

    pub fn system_parameters_list(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Postgres => {
                "select id, key, name, value, created_at, updated_at
                 from system_parameters
                 where deleted_at is null
                 order by key asc"
            }
            Self::Mysql => {
                "select id, `key` as `key`, name, value, created_at, updated_at
                 from system_parameters
                 where deleted_at is null
                 order by `key` asc"
            }
        }
    }

    pub fn system_parameter_by_key(self) -> &'static str {
        match self {
            Self::Sqlite => {
                "select id, key, name, value, created_at, updated_at
                 from system_parameters
                 where key = ? and deleted_at is null
                 limit 1"
            }
            Self::Postgres => {
                "select id, key, name, value, created_at, updated_at
                 from system_parameters
                 where key = $1 and deleted_at is null
                 limit 1"
            }
            Self::Mysql => {
                "select id, `key` as `key`, name, value, created_at, updated_at
                 from system_parameters
                 where `key` = ? and deleted_at is null
                 limit 1"
            }
        }
    }

    pub fn delete_system_parameter(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "update system_parameters
                 set deleted_at = ?, updated_at = ?
                 where key = ? and deleted_at is null"
            }
            Self::Postgres => {
                "update system_parameters
                 set deleted_at = $1, updated_at = $2
                 where key = $3 and deleted_at is null"
            }
        }
    }

    pub fn create_session(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "insert into iam_sessions(id, session_token_hash, refresh_token_hash, user_id, org_id, product_code, client_type, status, expires_at, refresh_expires_at, created_at, updated_at)
                 values (?, ?, ?, ?, ?, ?, ?, 'active', ?, ?, ?, ?)"
            }
            Self::Postgres => {
                "insert into iam_sessions(id, session_token_hash, refresh_token_hash, user_id, org_id, product_code, client_type, status, expires_at, refresh_expires_at, created_at, updated_at)
                 values ($1, $2, $3, $4, $5, $6, $7, 'active', $8, $9, $10, $11)"
            }
        }
    }

    pub fn session_by_token_hash(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select s.id, s.session_token_hash, s.refresh_token_hash, s.product_code, s.client_type, s.expires_at, s.refresh_expires_at, s.revoked_at,
                        u.id as user_id, u.email, u.display_name, u.status as user_status,
                        o.id as org_id, o.code as org_code, o.name as org_name, o.scope as org_scope
                 from iam_sessions s
                 join iam_users u on u.id = s.user_id
                 join iam_organizations o on o.id = s.org_id
                 where s.session_token_hash = ? and u.status = 'active' and u.deleted_at is null
                 limit ?"
            }
            Self::Postgres => {
                "select s.id, s.session_token_hash, s.refresh_token_hash, s.product_code, s.client_type, s.expires_at, s.refresh_expires_at, s.revoked_at,
                        u.id as user_id, u.email, u.display_name, u.status as user_status,
                        o.id as org_id, o.code as org_code, o.name as org_name, o.scope as org_scope
                 from iam_sessions s
                 join iam_users u on u.id = s.user_id
                 join iam_organizations o on o.id = s.org_id
                 where s.session_token_hash = $1 and u.status = 'active' and u.deleted_at is null
                 limit $2"
            }
        }
    }

    pub fn session_by_refresh_hash(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select s.id, s.session_token_hash, s.refresh_token_hash, s.product_code, s.client_type, s.expires_at, s.refresh_expires_at, s.revoked_at,
                        u.id as user_id, u.email, u.display_name, u.status as user_status,
                        o.id as org_id, o.code as org_code, o.name as org_name, o.scope as org_scope
                 from iam_sessions s
                 join iam_users u on u.id = s.user_id
                 join iam_organizations o on o.id = s.org_id
                 where s.refresh_token_hash = ? and u.status = 'active' and u.deleted_at is null
                 limit ?"
            }
            Self::Postgres => {
                "select s.id, s.session_token_hash, s.refresh_token_hash, s.product_code, s.client_type, s.expires_at, s.refresh_expires_at, s.revoked_at,
                        u.id as user_id, u.email, u.display_name, u.status as user_status,
                        o.id as org_id, o.code as org_code, o.name as org_name, o.scope as org_scope
                 from iam_sessions s
                 join iam_users u on u.id = s.user_id
                 join iam_organizations o on o.id = s.org_id
                 where s.refresh_token_hash = $1 and u.status = 'active' and u.deleted_at is null
                 limit $2"
            }
        }
    }

    pub fn rotate_session_tokens(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "update iam_sessions
                 set session_token_hash = ?, refresh_token_hash = ?, expires_at = ?, refresh_expires_at = ?, updated_at = ?
                 where id = ? and refresh_token_hash = ? and revoked_at is null"
            }
            Self::Postgres => {
                "update iam_sessions
                 set session_token_hash = $1, refresh_token_hash = $2, expires_at = $3, refresh_expires_at = $4, updated_at = $5
                 where id = $6 and refresh_token_hash = $7 and revoked_at is null"
            }
        }
    }

    pub fn revoke_session_by_token_hash(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "update iam_sessions
                 set status = 'revoked', revoked_at = ?, updated_at = ?
                 where session_token_hash = ?"
            }
            Self::Postgres => {
                "update iam_sessions
                 set status = 'revoked', revoked_at = $1, updated_at = $2
                 where session_token_hash = $3"
            }
        }
    }

    pub fn revoke_session_by_refresh_hash(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "update iam_sessions
                 set status = 'revoked', revoked_at = ?, updated_at = ?
                 where refresh_token_hash = ?"
            }
            Self::Postgres => {
                "update iam_sessions
                 set status = 'revoked', revoked_at = $1, updated_at = $2
                 where refresh_token_hash = $3"
            }
        }
    }

    pub fn permissions_for_user(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select distinct p.code
                 from iam_memberships m
                 join iam_roles r on r.code = m.role_code
                    and (
                      (m.org_id is not null and r.org_id = m.org_id)
                      or (m.org_id is null and r.org_id is null)
                    )
                 join iam_role_permissions rp on rp.role_id = r.id
                 join iam_permissions p on p.id = rp.permission_id
                 where m.user_id = ?
                   and m.status = 'active'
                   and p.product_code = ?
                   and (
                     (p.scope in ('tenant', 'product') and m.org_id = ?)
                     or (? = 1 and p.scope = 'platform' and m.org_id is null)
                   )
                 order by p.code asc"
            }
            Self::Postgres => {
                "select distinct p.code
                 from iam_memberships m
                 join iam_roles r on r.code = m.role_code
                    and (
                      (m.org_id is not null and r.org_id = m.org_id)
                      or (m.org_id is null and r.org_id is null)
                    )
                 join iam_role_permissions rp on rp.role_id = r.id
                 join iam_permissions p on p.id = rp.permission_id
                 where m.user_id = $1
                   and m.status = 'active'
                   and p.product_code = $2
                   and (
                     (p.scope in ('tenant', 'product') and m.org_id = $3)
                     or ($4 = 1 and p.scope = 'platform' and m.org_id is null)
                   )
                 order by p.code asc"
            }
        }
    }

    pub fn api_token_by_hash(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select t.id, t.status, t.expires_at, t.revoked_at,
                        u.id as user_id, u.email, u.display_name, u.status as user_status,
                        o.id as org_id, o.code as org_code, o.name as org_name, o.scope as org_scope
                 from iam_api_tokens t
                 join iam_users u on u.id = t.user_id
                 join iam_organizations o on o.id = t.org_id
                 where t.token_hash = ? and u.status = 'active' and u.deleted_at is null
                 limit ?"
            }
            Self::Postgres => {
                "select t.id, t.status, t.expires_at, t.revoked_at,
                        u.id as user_id, u.email, u.display_name, u.status as user_status,
                        o.id as org_id, o.code as org_code, o.name as org_name, o.scope as org_scope
                 from iam_api_tokens t
                 join iam_users u on u.id = t.user_id
                 join iam_organizations o on o.id = t.org_id
                 where t.token_hash = $1 and u.status = 'active' and u.deleted_at is null
                 limit $2"
            }
        }
    }

    pub fn api_tokens_for_org(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select id, org_id, user_id, token_prefix, status, expires_at, created_at, revoked_at
                 from iam_api_tokens
                 where org_id = ?
                 order by id desc"
            }
            Self::Postgres => {
                "select id, org_id, user_id, token_prefix, status, expires_at, created_at, revoked_at
                 from iam_api_tokens
                 where org_id = $1
                 order by id desc"
            }
        }
    }

    pub fn revoke_api_token(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "update iam_api_tokens
                 set status = 'revoked', revoked_at = ?
                 where org_id = ? and id = ? and revoked_at is null"
            }
            Self::Postgres => {
                "update iam_api_tokens
                 set status = 'revoked', revoked_at = $1
                 where org_id = $2 and id = $3 and revoked_at is null"
            }
        }
    }

    pub fn setup_runs_list(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select id, status, reason, created_at, updated_at
                 from setup_runs
                 order by updated_at desc, created_at desc
                 limit ?"
            }
            Self::Postgres => {
                "select id, status, reason, created_at, updated_at
                 from setup_runs
                 order by updated_at desc, created_at desc
                 limit $1"
            }
        }
    }

    pub fn due_notifications(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select o.id, o.org_id, o.user_id, o.product_code, o.channel, o.template_code,
                        o.recipient, o.related_kind, o.related_id, o.payload_json, o.status,
                        o.available_at, o.created_at, o.locked_at, o.delivered_at, o.failed_at,
                        o.failure_reason, o.attempt_count,
                        s.secret_ciphertext as delivery_secret_ciphertext
                 from iam_notification_outbox o
                 left join iam_notification_delivery_secrets s
                   on s.outbox_id = o.id and s.status = 'pending'
                 where o.status = 'pending'
                   and o.available_at <= ?
                   and (o.locked_at is null or o.locked_at <= ?)
                 order by o.id asc
                 limit ?"
            }
            Self::Postgres => {
                "select o.id, o.org_id, o.user_id, o.product_code, o.channel, o.template_code,
                        o.recipient, o.related_kind, o.related_id, o.payload_json, o.status,
                        o.available_at, o.created_at, o.locked_at, o.delivered_at, o.failed_at,
                        o.failure_reason, o.attempt_count,
                        s.secret_ciphertext as delivery_secret_ciphertext
                 from iam_notification_outbox o
                 left join iam_notification_delivery_secrets s
                   on s.outbox_id = o.id and s.status = 'pending'
                 where o.status = 'pending'
                   and o.available_at <= $1
                   and (o.locked_at is null or o.locked_at <= $2)
                 order by o.id asc
                 limit $3"
            }
        }
    }

    pub fn create_notification_outbox(self) -> &'static str {
        match self {
            Self::Sqlite => {
                "insert into iam_notification_outbox(
                   org_id, user_id, product_code, channel, template_code, recipient,
                   related_kind, related_id, payload_json, status, available_at, created_at
                 )
                 values (?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', ?, ?)
                 returning id"
            }
            Self::Postgres => {
                "insert into iam_notification_outbox(
                   org_id, user_id, product_code, channel, template_code, recipient,
                   related_kind, related_id, payload_json, status, available_at, created_at
                 )
                 values ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'pending', $10, $11)
                 returning id"
            }
            Self::Mysql => {
                "insert into iam_notification_outbox(
                   org_id, user_id, product_code, channel, template_code, recipient,
                   related_kind, related_id, payload_json, status, available_at, created_at
                 )
                 values (?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', ?, ?)"
            }
        }
    }

    pub fn create_notification_delivery_secret(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "insert into iam_notification_delivery_secrets(
                   outbox_id, secret_kind, secret_ciphertext, status, created_at
                 )
                 values (?, 'raw_token', ?, 'pending', ?)"
            }
            Self::Postgres => {
                "insert into iam_notification_delivery_secrets(
                   outbox_id, secret_kind, secret_ciphertext, status, created_at
                 )
                 values ($1, 'raw_token', $2, 'pending', $3)"
            }
        }
    }

    pub fn purge_notification_delivery_secret(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "update iam_notification_delivery_secrets
                 set status = 'purged',
                     secret_ciphertext = null,
                     purged_at = ?
                 where outbox_id = ? and status = 'pending'"
            }
            Self::Postgres => {
                "update iam_notification_delivery_secrets
                 set status = 'purged',
                     secret_ciphertext = null,
                     purged_at = $1
                 where outbox_id = $2 and status = 'pending'"
            }
        }
    }

    pub fn claim_notification(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "update iam_notification_outbox
                 set locked_at = ?
                 where id = ? and status = 'pending'"
            }
            Self::Postgres => {
                "update iam_notification_outbox
                 set locked_at = $1
                 where id = $2 and status = 'pending'"
            }
        }
    }

    pub fn mark_notification_delivered(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "update iam_notification_outbox
                 set status = 'delivered',
                     delivered_at = ?,
                     failed_at = null,
                     failure_reason = null
                 where id = ? and status = 'pending'"
            }
            Self::Postgres => {
                "update iam_notification_outbox
                 set status = 'delivered',
                     delivered_at = $1,
                     failed_at = null,
                     failure_reason = null
                 where id = $2 and status = 'pending'"
            }
        }
    }

    pub fn notification_attempt_count(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select attempt_count
                 from iam_notification_outbox
                 where id = ? and status = 'pending'
                 limit 1"
            }
            Self::Postgres => {
                "select attempt_count
                 from iam_notification_outbox
                 where id = $1 and status = 'pending'
                 limit 1"
            }
        }
    }

    pub fn mark_notification_retry(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "update iam_notification_outbox
                 set status = 'pending',
                     attempt_count = ?,
                     available_at = ?,
                     locked_at = null,
                     failed_at = ?,
                     failure_reason = ?
                 where id = ? and status = 'pending'"
            }
            Self::Postgres => {
                "update iam_notification_outbox
                 set status = 'pending',
                     attempt_count = $1,
                     available_at = $2,
                     locked_at = null,
                     failed_at = $3,
                     failure_reason = $4
                 where id = $5 and status = 'pending'"
            }
        }
    }

    pub fn mark_notification_final_failed(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "update iam_notification_outbox
                 set status = 'failed',
                     attempt_count = ?,
                     locked_at = null,
                     failed_at = ?,
                     failure_reason = ?
                 where id = ? and status = 'pending'"
            }
            Self::Postgres => {
                "update iam_notification_outbox
                 set status = 'failed',
                     attempt_count = $1,
                     locked_at = null,
                     failed_at = $2,
                     failure_reason = $3
                 where id = $4 and status = 'pending'"
            }
        }
    }

    pub fn failed_notifications(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select o.id, o.org_id, o.user_id, o.product_code, o.channel, o.template_code,
                        o.recipient, o.related_kind, o.related_id, o.status, o.created_at,
                        o.failed_at, o.failure_reason, o.attempt_count,
                        s.status as delivery_secret_status,
                        s.secret_ciphertext as delivery_secret_ciphertext,
                        s.purged_at as delivery_secret_purged_at
                 from iam_notification_outbox o
                 left join iam_notification_delivery_secrets s on s.outbox_id = o.id
                 where o.status = 'failed'
                 order by o.failed_at desc, o.id desc
                 limit ?"
            }
            Self::Postgres => {
                "select o.id, o.org_id, o.user_id, o.product_code, o.channel, o.template_code,
                        o.recipient, o.related_kind, o.related_id, o.status, o.created_at,
                        o.failed_at, o.failure_reason, o.attempt_count,
                        s.status as delivery_secret_status,
                        s.secret_ciphertext as delivery_secret_ciphertext,
                        s.purged_at as delivery_secret_purged_at
                 from iam_notification_outbox o
                 left join iam_notification_delivery_secrets s on s.outbox_id = o.id
                 where o.status = 'failed'
                 order by o.failed_at desc, o.id desc
                 limit $1"
            }
        }
    }

    pub fn requeue_failed_notification(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "update iam_notification_outbox
                 set status = 'pending',
                     attempt_count = 0,
                     available_at = ?,
                     locked_at = null,
                     delivered_at = null,
                     failed_at = null,
                     failure_reason = null
                 where id = ?
                   and status = 'failed'
                   and exists (
                     select 1
                     from iam_notification_delivery_secrets s
                     where s.outbox_id = iam_notification_outbox.id
                       and s.status = 'pending'
                       and s.secret_ciphertext is not null
                   )"
            }
            Self::Postgres => {
                "update iam_notification_outbox
                 set status = 'pending',
                     attempt_count = 0,
                     available_at = $1,
                     locked_at = null,
                     delivered_at = null,
                     failed_at = null,
                     failure_reason = null
                 where id = $2
                   and status = 'failed'
                   and exists (
                     select 1
                     from iam_notification_delivery_secrets s
                     where s.outbox_id = iam_notification_outbox.id
                       and s.status = 'pending'
                       and s.secret_ciphertext is not null
                   )"
            }
        }
    }

    pub fn operation_records_list(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select id, actor_user_id, method, path, status, created_at
                 from system_operation_records
                 where (? is null or upper(method) = ?)
                   and (? is null or path = ?)
                   and (? is null or status = ?)
                   and (? is null or actor_user_id = ?)
                   and (? is null or created_at >= ?)
                   and (? is null or created_at <= ?)
                 order by id desc
                 limit ? offset ?"
            }
            Self::Postgres => {
                "select id, actor_user_id, method, path, status, created_at
                 from system_operation_records
                 where ($1::text is null or upper(method) = $2)
                   and ($3::text is null or path = $4)
                   and ($5::bigint is null or status::bigint = $6::bigint)
                   and ($7::bigint is null or actor_user_id::bigint = $8::bigint)
                   and ($9::text is null or created_at >= $10)
                   and ($11::text is null or created_at <= $12)
                 order by id desc
                 limit $13 offset $14"
            }
        }
    }

    pub fn operation_records_summary_counts(self) -> &'static str {
        match self {
            Self::Sqlite => {
                "select count(*) as total_count,
                        coalesce(sum(case when status between 200 and 299 then 1 else 0 end), 0) as success_count,
                        coalesce(sum(case when status between 300 and 399 then 1 else 0 end), 0) as redirect_count,
                        coalesce(sum(case when status between 400 and 499 then 1 else 0 end), 0) as client_error_count,
                        coalesce(sum(case when status between 500 and 599 then 1 else 0 end), 0) as server_error_count,
                        coalesce(sum(case when status < 200 or status > 599 then 1 else 0 end), 0) as other_count
                 from system_operation_records
                 where (? is null or upper(method) = ?)
                   and (? is null or path = ?)
                   and (? is null or status = ?)
                   and (? is null or actor_user_id = ?)
                   and (? is null or created_at >= ?)
                   and (? is null or created_at <= ?)"
            }
            Self::Mysql => {
                "select cast(count(*) as signed) as total_count,
                        cast(coalesce(sum(case when status between 200 and 299 then 1 else 0 end), 0) as signed) as success_count,
                        cast(coalesce(sum(case when status between 300 and 399 then 1 else 0 end), 0) as signed) as redirect_count,
                        cast(coalesce(sum(case when status between 400 and 499 then 1 else 0 end), 0) as signed) as client_error_count,
                        cast(coalesce(sum(case when status between 500 and 599 then 1 else 0 end), 0) as signed) as server_error_count,
                        cast(coalesce(sum(case when status < 200 or status > 599 then 1 else 0 end), 0) as signed) as other_count
                 from system_operation_records
                 where (? is null or upper(method) = ?)
                   and (? is null or path = ?)
                   and (? is null or status = ?)
                   and (? is null or actor_user_id = ?)
                   and (? is null or created_at >= ?)
                   and (? is null or created_at <= ?)"
            }
            Self::Postgres => {
                "select count(*) as total_count,
                        coalesce(sum(case when status between 200 and 299 then 1 else 0 end), 0) as success_count,
                        coalesce(sum(case when status between 300 and 399 then 1 else 0 end), 0) as redirect_count,
                        coalesce(sum(case when status between 400 and 499 then 1 else 0 end), 0) as client_error_count,
                        coalesce(sum(case when status between 500 and 599 then 1 else 0 end), 0) as server_error_count,
                        coalesce(sum(case when status < 200 or status > 599 then 1 else 0 end), 0) as other_count
                 from system_operation_records
                 where ($1::text is null or upper(method) = $2)
                   and ($3::text is null or path = $4)
                   and ($5::bigint is null or status::bigint = $6::bigint)
                   and ($7::bigint is null or actor_user_id::bigint = $8::bigint)
                   and ($9::text is null or created_at >= $10)
                   and ($11::text is null or created_at <= $12)"
            }
        }
    }

    pub fn operation_records_summary_methods(self) -> &'static str {
        match self {
            Self::Sqlite => {
                "select method as key, count(*) as count
                 from system_operation_records
                 where (? is null or upper(method) = ?)
                   and (? is null or path = ?)
                   and (? is null or status = ?)
                   and (? is null or actor_user_id = ?)
                   and (? is null or created_at >= ?)
                   and (? is null or created_at <= ?)
                 group by method
                 order by count desc, method asc"
            }
            Self::Mysql => {
                "select method as key, cast(count(*) as signed) as count
                 from system_operation_records
                 where (? is null or upper(method) = ?)
                   and (? is null or path = ?)
                   and (? is null or status = ?)
                   and (? is null or actor_user_id = ?)
                   and (? is null or created_at >= ?)
                   and (? is null or created_at <= ?)
                 group by method
                 order by count desc, method asc"
            }
            Self::Postgres => {
                "select method as key, count(*) as count
                 from system_operation_records
                 where ($1::text is null or upper(method) = $2)
                   and ($3::text is null or path = $4)
                   and ($5::bigint is null or status::bigint = $6::bigint)
                   and ($7::bigint is null or actor_user_id::bigint = $8::bigint)
                   and ($9::text is null or created_at >= $10)
                   and ($11::text is null or created_at <= $12)
                 group by method
                 order by count desc, method asc"
            }
        }
    }

    pub fn operation_records_summary_status_classes(self) -> &'static str {
        match self {
            Self::Sqlite => {
                "select status_class as key, count(*) as count
                 from (
                   select case
                            when status between 100 and 199 then '1xx'
                            when status between 200 and 299 then '2xx'
                            when status between 300 and 399 then '3xx'
                            when status between 400 and 499 then '4xx'
                            when status between 500 and 599 then '5xx'
                            else 'other'
                          end as status_class
                   from system_operation_records
                   where (? is null or upper(method) = ?)
                     and (? is null or path = ?)
                     and (? is null or status = ?)
                     and (? is null or actor_user_id = ?)
                     and (? is null or created_at >= ?)
                     and (? is null or created_at <= ?)
                 ) as grouped_status
                 group by status_class
                 order by status_class asc"
            }
            Self::Mysql => {
                "select status_class as key, cast(count(*) as signed) as count
                 from (
                   select case
                            when status between 100 and 199 then '1xx'
                            when status between 200 and 299 then '2xx'
                            when status between 300 and 399 then '3xx'
                            when status between 400 and 499 then '4xx'
                            when status between 500 and 599 then '5xx'
                            else 'other'
                          end as status_class
                   from system_operation_records
                   where (? is null or upper(method) = ?)
                     and (? is null or path = ?)
                     and (? is null or status = ?)
                     and (? is null or actor_user_id = ?)
                     and (? is null or created_at >= ?)
                     and (? is null or created_at <= ?)
                 ) as grouped_status
                 group by status_class
                 order by status_class asc"
            }
            Self::Postgres => {
                "select status_class as key, count(*) as count
                 from (
                   select case
                            when status between 100 and 199 then '1xx'
                            when status between 200 and 299 then '2xx'
                            when status between 300 and 399 then '3xx'
                            when status between 400 and 499 then '4xx'
                            when status between 500 and 599 then '5xx'
                            else 'other'
                          end as status_class
                   from system_operation_records
                   where ($1::text is null or upper(method) = $2)
                     and ($3::text is null or path = $4)
                     and ($5::bigint is null or status::bigint = $6::bigint)
                     and ($7::bigint is null or actor_user_id::bigint = $8::bigint)
                     and ($9::text is null or created_at >= $10)
                     and ($11::text is null or created_at <= $12)
                 ) as grouped_status
                 group by status_class
                 order by status_class asc"
            }
        }
    }

    pub fn operation_records_summary_paths(self) -> &'static str {
        match self {
            Self::Sqlite => {
                "select path,
                        count(*) as count,
                        coalesce(sum(case when status >= 400 then 1 else 0 end), 0) as error_count,
                        max(created_at) as last_seen_at
                 from system_operation_records
                 where (? is null or upper(method) = ?)
                   and (? is null or path = ?)
                   and (? is null or status = ?)
                   and (? is null or actor_user_id = ?)
                   and (? is null or created_at >= ?)
                   and (? is null or created_at <= ?)
                 group by path
                 order by count desc, error_count desc, path asc
                 limit ?"
            }
            Self::Mysql => {
                "select path,
                        cast(count(*) as signed) as count,
                        cast(coalesce(sum(case when status >= 400 then 1 else 0 end), 0) as signed) as error_count,
                        max(created_at) as last_seen_at
                 from system_operation_records
                 where (? is null or upper(method) = ?)
                   and (? is null or path = ?)
                   and (? is null or status = ?)
                   and (? is null or actor_user_id = ?)
                   and (? is null or created_at >= ?)
                   and (? is null or created_at <= ?)
                 group by path
                 order by count desc, error_count desc, path asc
                 limit ?"
            }
            Self::Postgres => {
                "select path,
                        count(*) as count,
                        coalesce(sum(case when status >= 400 then 1 else 0 end), 0) as error_count,
                        max(created_at) as last_seen_at
                 from system_operation_records
                 where ($1::text is null or upper(method) = $2)
                   and ($3::text is null or path = $4)
                   and ($5::bigint is null or status::bigint = $6::bigint)
                   and ($7::bigint is null or actor_user_id::bigint = $8::bigint)
                   and ($9::text is null or created_at >= $10)
                   and ($11::text is null or created_at <= $12)
                 group by path
                 order by count desc, error_count desc, path asc
                 limit $13"
            }
        }
    }

    pub fn create_operation_record(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "insert into system_operation_records(actor_user_id, method, path, status, created_at)
                 values (?, ?, ?, ?, ?)"
            }
            Self::Postgres => {
                "insert into system_operation_records(actor_user_id, method, path, status, created_at)
                 values ($1, $2, $3, $4, $5)"
            }
        }
    }

    pub fn prune_operation_records(self) -> &'static str {
        match self {
            Self::Sqlite => {
                "delete from system_operation_records
                 where id in (
                   select id
                   from system_operation_records
                   where created_at < ?
                   order by id asc
                   limit ?
                 )"
            }
            Self::Postgres => {
                "delete from system_operation_records
                 where id in (
                   select id
                   from system_operation_records
                   where created_at < $1
                   order by id asc
                   limit $2
                 )"
            }
            Self::Mysql => {
                "delete from system_operation_records
                 where id in (
                   select id
                   from (
                     select id
                     from system_operation_records
                     where created_at < ?
                     order by id asc
                     limit ?
                   ) as prune_ids
                 )"
            }
        }
    }

    pub fn version_release_events_list(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Postgres | Self::Mysql => {
                "select id, package_id, previous_active_id, action, status, reason, created_at
                 from system_version_release_events
                 order by id desc
                 limit 100"
            }
        }
    }

    pub fn create_version_release_event(self) -> &'static str {
        match self {
            Self::Sqlite => {
                "insert into system_version_release_events(
                   package_id, previous_active_id, action, status, reason, created_at
                 )
                 values (?, ?, ?, 'succeeded', ?, ?)
                 returning id"
            }
            Self::Postgres => {
                "insert into system_version_release_events(
                   package_id, previous_active_id, action, status, reason, created_at
                 )
                 values ($1, $2, $3, 'succeeded', $4, $5)
                 returning id"
            }
            Self::Mysql => {
                "insert into system_version_release_events(
                   package_id, previous_active_id, action, status, reason, created_at
                 )
                 values (?, ?, ?, 'succeeded', ?, ?)"
            }
        }
    }

    pub fn create_version_package(self) -> &'static str {
        match self {
            Self::Sqlite => {
                "insert into system_version_packages(version_name, version_code, manifest_json, status, created_at)
                 values (?, ?, ?, 'draft', ?)
                 returning id"
            }
            Self::Postgres => {
                "insert into system_version_packages(version_name, version_code, manifest_json, status, created_at)
                 values ($1, $2, $3, 'draft', $4)
                 returning id"
            }
            Self::Mysql => {
                "insert into system_version_packages(version_name, version_code, manifest_json, status, created_at)
                 values (?, ?, ?, 'draft', ?)"
            }
        }
    }

    pub fn version_packages_list(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Postgres | Self::Mysql => {
                "select id, version_name, version_code, manifest_json, status, created_at,
                        published_at, retired_at
                 from system_version_packages
                 where deleted_at is null
                 order by id desc"
            }
        }
    }

    pub fn version_package_by_id(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select id, version_name, version_code, manifest_json, status, created_at,
                        published_at, retired_at
                 from system_version_packages
                 where id = ? and deleted_at is null
                 limit 1"
            }
            Self::Postgres => {
                "select id, version_name, version_code, manifest_json, status, created_at,
                        published_at, retired_at
                 from system_version_packages
                 where id = $1 and deleted_at is null
                 limit 1"
            }
        }
    }

    pub fn active_version_package_id(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Postgres | Self::Mysql => {
                "select id
                 from system_version_packages
                 where status = 'active' and deleted_at is null
                 order by id desc
                 limit 1"
            }
        }
    }

    pub fn retire_version_package(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "update system_version_packages
                 set status = 'retired', retired_at = ?
                 where id = ?"
            }
            Self::Postgres => {
                "update system_version_packages
                 set status = 'retired', retired_at = $1
                 where id = $2"
            }
        }
    }

    pub fn activate_version_package(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "update system_version_packages
                 set status = 'active', published_at = ?, retired_at = null
                 where id = ? and deleted_at is null"
            }
            Self::Postgres => {
                "update system_version_packages
                 set status = 'active', published_at = $1, retired_at = null
                 where id = $2 and deleted_at is null"
            }
        }
    }

    pub fn delete_version_package(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "update system_version_packages
                 set deleted_at = ?
                 where id = ? and deleted_at is null and status != 'active'"
            }
            Self::Postgres => {
                "update system_version_packages
                 set deleted_at = $1
                 where id = $2 and deleted_at is null and status != 'active'"
            }
        }
    }

    pub fn version_package_status_by_id(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select status
                 from system_version_packages
                 where id = ? and deleted_at is null
                 limit 1"
            }
            Self::Postgres => {
                "select status
                 from system_version_packages
                 where id = $1 and deleted_at is null
                 limit 1"
            }
        }
    }

    pub fn create_media_asset(self) -> &'static str {
        match self {
            Self::Sqlite => {
                "insert into system_media_assets(category, display_name, storage_key, mime_type, size_bytes, created_at)
                 values (?, ?, ?, ?, ?, ?)
                 returning id"
            }
            Self::Postgres => {
                "insert into system_media_assets(category, display_name, storage_key, mime_type, size_bytes, created_at)
                 values ($1, $2, $3, $4, $5, $6)
                 returning id"
            }
            Self::Mysql => {
                "insert into system_media_assets(category, display_name, storage_key, mime_type, size_bytes, created_at)
                 values (?, ?, ?, ?, ?, ?)"
            }
        }
    }

    pub fn media_assets_list(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Postgres | Self::Mysql => {
                "select id, category, display_name, storage_key, mime_type, size_bytes, created_at
                 from system_media_assets
                 where deleted_at is null
                 order by id desc"
            }
        }
    }

    pub fn delete_media_asset(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "update system_media_assets
                 set deleted_at = ?
                 where id = ? and deleted_at is null"
            }
            Self::Postgres => {
                "update system_media_assets
                 set deleted_at = $1
                 where id = $2 and deleted_at is null"
            }
        }
    }

    pub fn create_traffic_probe_target(self) -> &'static str {
        match self {
            Self::Sqlite => {
                "insert into system_traffic_probe_targets(name, url, expected_status, status, created_at)
                 values (?, ?, ?, 'pending', ?)
                 returning id"
            }
            Self::Postgres => {
                "insert into system_traffic_probe_targets(name, url, expected_status, status, created_at)
                 values ($1, $2, $3, 'pending', $4)
                 returning id"
            }
            Self::Mysql => {
                "insert into system_traffic_probe_targets(name, url, expected_status, status, created_at)
                 values (?, ?, ?, 'pending', ?)"
            }
        }
    }

    pub fn create_traffic_probe_result(self) -> &'static str {
        match self {
            Self::Sqlite => {
                "insert into system_traffic_probe_results(target_id, status, detail_json, probed_at)
                 values (?, ?, ?, ?)
                 returning id"
            }
            Self::Postgres => {
                "insert into system_traffic_probe_results(target_id, status, detail_json, probed_at)
                 values ($1, $2, $3, $4)
                 returning id"
            }
            Self::Mysql => {
                "insert into system_traffic_probe_results(target_id, status, detail_json, probed_at)
                 values (?, ?, ?, ?)"
            }
        }
    }

    pub fn create_traffic_probe_alert(self) -> &'static str {
        match self {
            Self::Sqlite => {
                "insert into system_traffic_probe_alerts(
                   target_id, result_id, severity, status, reason, detail_json, opened_at
                 )
                 values (?, ?, ?, 'open', ?, ?, ?)
                 returning id"
            }
            Self::Postgres => {
                "insert into system_traffic_probe_alerts(
                   target_id, result_id, severity, status, reason, detail_json, opened_at
                 )
                 values ($1, $2, $3, 'open', $4, $5, $6)
                 returning id"
            }
            Self::Mysql => {
                "insert into system_traffic_probe_alerts(
                   target_id, result_id, severity, status, reason, detail_json, opened_at
                 )
                 values (?, ?, ?, 'open', ?, ?, ?)"
            }
        }
    }

    pub fn traffic_probe_results_for_target(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select id, target_id, status, detail_json, probed_at
                 from system_traffic_probe_results
                 where target_id = ?
                 order by id desc
                 limit ?"
            }
            Self::Postgres => {
                "select id, target_id, status, detail_json, probed_at
                 from system_traffic_probe_results
                 where target_id = $1
                 order by id desc
                 limit $2"
            }
        }
    }

    pub fn traffic_probe_targets_list(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Postgres | Self::Mysql => {
                "select id, name, url, expected_status, status, created_at
                 from system_traffic_probe_targets
                 where deleted_at is null
                 order by id desc"
            }
        }
    }

    pub fn traffic_probe_target_by_id(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select id, name, url, expected_status, status, created_at
                 from system_traffic_probe_targets
                 where id = ? and deleted_at is null
                 limit 1"
            }
            Self::Postgres => {
                "select id, name, url, expected_status, status, created_at
                 from system_traffic_probe_targets
                 where id = $1 and deleted_at is null
                 limit 1"
            }
        }
    }

    pub fn delete_traffic_probe_target(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "update system_traffic_probe_targets
                 set deleted_at = ?
                 where id = ? and deleted_at is null"
            }
            Self::Postgres => {
                "update system_traffic_probe_targets
                 set deleted_at = $1
                 where id = $2 and deleted_at is null"
            }
        }
    }

    pub fn update_traffic_probe_target_status(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "update system_traffic_probe_targets
                 set status = ?
                 where id = ? and deleted_at is null"
            }
            Self::Postgres => {
                "update system_traffic_probe_targets
                 set status = $1
                 where id = $2 and deleted_at is null"
            }
        }
    }

    pub fn traffic_probe_results_all(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select id, target_id, status, detail_json, probed_at
                 from system_traffic_probe_results
                 order by id desc
                 limit ?"
            }
            Self::Postgres => {
                "select id, target_id, status, detail_json, probed_at
                 from system_traffic_probe_results
                 order by id desc
                 limit $1"
            }
        }
    }

    pub fn acknowledge_traffic_probe_alert(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "update system_traffic_probe_alerts
                 set status = 'acknowledged', acknowledged_at = ?
                 where id = ? and status = 'open'"
            }
            Self::Postgres => {
                "update system_traffic_probe_alerts
                 set status = 'acknowledged', acknowledged_at = $1
                 where id = $2 and status = 'open'"
            }
        }
    }

    pub fn resolve_traffic_probe_alert(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "update system_traffic_probe_alerts
                 set status = 'resolved', resolved_at = ?
                 where id = ? and status in ('open', 'acknowledged')"
            }
            Self::Postgres => {
                "update system_traffic_probe_alerts
                 set status = 'resolved', resolved_at = $1
                 where id = $2 and status in ('open', 'acknowledged')"
            }
        }
    }

    pub fn resolve_traffic_probe_alerts_for_target(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "update system_traffic_probe_alerts
                 set status = 'resolved', resolved_at = ?
                 where target_id = ? and status in ('open', 'acknowledged')"
            }
            Self::Postgres => {
                "update system_traffic_probe_alerts
                 set status = 'resolved', resolved_at = $1
                 where target_id = $2 and status in ('open', 'acknowledged')"
            }
        }
    }

    pub fn traffic_probe_alerts_target_status(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select id, target_id, result_id, severity, status, reason, detail_json,
                        opened_at, acknowledged_at, resolved_at
                 from system_traffic_probe_alerts
                 where target_id = ? and status = ?
                 order by id desc
                 limit ?"
            }
            Self::Postgres => {
                "select id, target_id, result_id, severity, status, reason, detail_json,
                        opened_at, acknowledged_at, resolved_at
                 from system_traffic_probe_alerts
                 where target_id = $1 and status = $2
                 order by id desc
                 limit $3"
            }
        }
    }

    pub fn traffic_probe_alerts_target(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select id, target_id, result_id, severity, status, reason, detail_json,
                        opened_at, acknowledged_at, resolved_at
                 from system_traffic_probe_alerts
                 where target_id = ?
                 order by id desc
                 limit ?"
            }
            Self::Postgres => {
                "select id, target_id, result_id, severity, status, reason, detail_json,
                        opened_at, acknowledged_at, resolved_at
                 from system_traffic_probe_alerts
                 where target_id = $1
                 order by id desc
                 limit $2"
            }
        }
    }

    pub fn traffic_probe_alerts_status(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select id, target_id, result_id, severity, status, reason, detail_json,
                        opened_at, acknowledged_at, resolved_at
                 from system_traffic_probe_alerts
                 where status = ?
                 order by id desc
                 limit ?"
            }
            Self::Postgres => {
                "select id, target_id, result_id, severity, status, reason, detail_json,
                        opened_at, acknowledged_at, resolved_at
                 from system_traffic_probe_alerts
                 where status = $1
                 order by id desc
                 limit $2"
            }
        }
    }

    pub fn traffic_probe_alerts_all(self) -> &'static str {
        match self {
            Self::Sqlite | Self::Mysql => {
                "select id, target_id, result_id, severity, status, reason, detail_json,
                        opened_at, acknowledged_at, resolved_at
                 from system_traffic_probe_alerts
                 order by id desc
                 limit ?"
            }
            Self::Postgres => {
                "select id, target_id, result_id, severity, status, reason, detail_json,
                        opened_at, acknowledged_at, resolved_at
                 from system_traffic_probe_alerts
                 order by id desc
                 limit $1"
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::{InsertIdRead, InsertIdStrategy, SqlDialect};

    #[test]
    fn insert_id_strategy_keeps_mysql_off_returning_id_assumption() {
        assert_eq!(
            SqlDialect::Sqlite.insert_id_strategy(),
            InsertIdStrategy::ReturningId
        );
        assert_eq!(
            SqlDialect::Postgres.insert_id_strategy(),
            InsertIdStrategy::ReturningId
        );
        assert_eq!(
            SqlDialect::Mysql.insert_id_strategy(),
            InsertIdStrategy::DialectSpecificPostInsertRead
        );
    }

    #[test]
    fn insert_id_read_exposes_mysql_post_insert_query() {
        assert_eq!(
            SqlDialect::Sqlite.insert_id_read(),
            InsertIdRead::ReturningIdInStatement
        );
        assert_eq!(
            SqlDialect::Postgres.insert_id_read(),
            InsertIdRead::ReturningIdInStatement
        );
        assert_eq!(
            SqlDialect::Mysql.insert_id_read(),
            InsertIdRead::PostInsertQuery("select last_insert_id()")
        );
    }

    #[test]
    fn mysql_generated_id_templates_require_post_insert_id_read() {
        let templates = [
            SqlDialect::Mysql.create_tenant_organization(),
            SqlDialect::Mysql.create_active_user(),
            SqlDialect::Mysql.create_pending_verification_user(),
            SqlDialect::Mysql.create_tenant_owner_role(),
            SqlDialect::Mysql.create_platform_owner_role(),
            SqlDialect::Mysql.create_org_role(),
            SqlDialect::Mysql.create_api_token(),
            SqlDialect::Mysql.create_invitation(),
            SqlDialect::Mysql.create_password_reset(),
            SqlDialect::Mysql.create_email_verification(),
            SqlDialect::Mysql.create_mfa_factor(),
            SqlDialect::Mysql.create_mfa_recovery_code(),
            SqlDialect::Mysql.create_notification_outbox(),
            SqlDialect::Mysql.create_version_release_event(),
            SqlDialect::Mysql.create_version_package(),
            SqlDialect::Mysql.create_media_asset(),
            SqlDialect::Mysql.create_traffic_probe_target(),
            SqlDialect::Mysql.create_traffic_probe_result(),
            SqlDialect::Mysql.create_traffic_probe_alert(),
        ];

        assert_eq!(
            SqlDialect::Mysql.insert_id_read(),
            InsertIdRead::PostInsertQuery("select last_insert_id()")
        );
        for template in templates {
            assert!(!template.to_ascii_lowercase().contains("returning id"));
        }
    }

    #[test]
    fn mysql_templates_do_not_use_sqlite_upsert_syntax() {
        let templates = [
            SqlDialect::Mysql.role_permission_values(),
            SqlDialect::Mysql.role_permissions_for_tenant_scopes(),
            SqlDialect::Mysql.role_permissions_for_platform_scope(),
            SqlDialect::Mysql.setup_state_completed_upsert(),
            SqlDialect::Mysql.system_api_upsert(),
            SqlDialect::Mysql.permission_upsert(),
            SqlDialect::Mysql.platform_builtin_role_permissions(),
            SqlDialect::Mysql.tenant_builtin_role_permissions(),
            SqlDialect::Mysql.system_menu_upsert(),
            SqlDialect::Mysql.system_config_upsert(),
            SqlDialect::Mysql.system_dictionary_upsert(),
            SqlDialect::Mysql.system_parameter_upsert(),
        ];

        for template in templates {
            let lower = template.to_ascii_lowercase();
            assert!(!lower.contains("on conflict"));
            assert!(!lower.contains("excluded."));
            assert!(lower.contains("insert ignore") || lower.contains("on duplicate key update"));
        }
    }

    #[test]
    fn postgres_templates_use_numbered_placeholders_where_values_are_bound() {
        assert!(
            SqlDialect::Postgres
                .role_permissions_for_role()
                .contains("role_id = $1")
        );
        assert!(
            SqlDialect::Postgres
                .tenant_permission_id_by_code()
                .contains("code = $2")
        );
        assert!(
            SqlDialect::Postgres
                .delete_role_permissions()
                .contains("role_id = $1")
        );
        assert!(SqlDialect::Postgres.role_permission_values().contains("$1"));
        assert!(
            SqlDialect::Postgres
                .complete_setup_run()
                .contains("id = $2")
        );
        assert!(
            SqlDialect::Postgres
                .append_setup_complete_log()
                .contains("values ($1")
        );
        assert!(SqlDialect::Postgres.create_setup_run().contains("$4"));
        assert!(SqlDialect::Postgres.append_setup_step_log().contains("$5"));
        assert!(
            SqlDialect::Postgres
                .setup_step_logs_for_run()
                .contains("run_id = $1")
        );
        assert!(SqlDialect::Postgres.system_api_upsert().contains("$11"));
        assert!(
            SqlDialect::Postgres
                .system_apis_list()
                .contains("from system_apis")
        );
        assert!(
            SqlDialect::Postgres
                .system_parameter_upsert()
                .contains("$5")
        );
        assert!(
            SqlDialect::Postgres
                .delete_system_parameter()
                .contains("key = $3")
        );
        assert!(
            SqlDialect::Postgres
                .system_dictionary_by_code()
                .contains("code = $1")
        );
        assert!(
            SqlDialect::Postgres
                .due_notifications()
                .contains("limit $3")
        );
        assert!(
            SqlDialect::Postgres
                .purge_notification_delivery_secret()
                .contains("outbox_id = $2")
        );
        assert!(
            SqlDialect::Postgres
                .claim_notification()
                .contains("id = $2")
        );
        assert!(
            SqlDialect::Postgres
                .mark_notification_delivered()
                .contains("delivered_at = $1")
        );
        assert!(
            SqlDialect::Postgres
                .notification_attempt_count()
                .contains("id = $1")
        );
        assert!(
            SqlDialect::Postgres
                .mark_notification_retry()
                .contains("id = $5")
        );
        assert!(
            SqlDialect::Postgres
                .mark_notification_final_failed()
                .contains("id = $4")
        );
        assert!(
            SqlDialect::Postgres
                .failed_notifications()
                .contains("limit $1")
        );
        assert!(
            SqlDialect::Postgres
                .traffic_probe_alerts_target_status()
                .contains("limit $3")
        );
        assert!(
            SqlDialect::Postgres
                .traffic_probe_target_by_id()
                .contains("id = $1")
        );
        assert!(
            SqlDialect::Postgres
                .resolve_traffic_probe_alerts_for_target()
                .contains("target_id = $2")
        );
        assert!(
            SqlDialect::Postgres
                .delete_media_asset()
                .contains("id = $2")
        );
        assert!(
            SqlDialect::Postgres
                .version_package_status_by_id()
                .contains("id = $1")
        );
        assert!(
            SqlDialect::Postgres
                .version_package_by_id()
                .contains("id = $1")
        );
        assert!(
            SqlDialect::Postgres
                .retire_version_package()
                .contains("id = $2")
        );
        assert!(
            SqlDialect::Postgres
                .activate_version_package()
                .contains("published_at = $1")
        );
        assert!(SqlDialect::Postgres.create_media_asset().contains("$6"));
        assert!(
            SqlDialect::Postgres
                .create_traffic_probe_alert()
                .contains("$6")
        );
        assert!(
            SqlDialect::Postgres
                .create_tenant_organization()
                .contains("$4")
        );
        assert!(SqlDialect::Postgres.create_active_user().contains("$6"));
        assert!(
            SqlDialect::Postgres
                .create_pending_verification_user()
                .contains("$5")
        );
        assert!(
            SqlDialect::Postgres
                .create_active_membership()
                .contains("$5")
        );
        assert!(
            SqlDialect::Postgres
                .users_count()
                .contains("from iam_users")
        );
        assert!(
            SqlDialect::Postgres
                .user_email_count()
                .contains("lower($1)")
        );
        assert!(
            SqlDialect::Postgres
                .organization_code_count()
                .contains("code = $1")
        );
        assert!(
            SqlDialect::Postgres
                .user_by_identifier()
                .contains("lower($1)")
        );
        assert!(
            SqlDialect::Postgres
                .primary_organization_for_user()
                .contains("m.user_id = $1")
        );
        assert!(SqlDialect::Postgres.org_users_list().contains("string_agg"));
        assert!(
            SqlDialect::Postgres
                .org_user_membership_context()
                .contains("u.id = $3")
        );
        assert!(
            SqlDialect::Postgres
                .org_active_owner_count_except_user()
                .contains("m.user_id != $2")
        );
        assert!(
            SqlDialect::Postgres
                .update_org_user_profile_status()
                .contains("updated_at = $3")
        );
        assert!(
            SqlDialect::Postgres
                .delete_org_user_memberships()
                .contains("org_id = $1")
        );
        assert!(SqlDialect::Postgres.create_audit_log().contains("$7"));
        assert!(SqlDialect::Postgres.create_api_token().contains("$6"));
        assert!(SqlDialect::Postgres.create_org_role().contains("$5"));
        assert!(
            SqlDialect::Postgres
                .org_roles_list()
                .contains("org_id = $1")
        );
        assert!(
            SqlDialect::Postgres
                .org_role_code_count()
                .contains("code = $2")
        );
        assert!(
            SqlDialect::Postgres
                .tenant_org_role_code_count()
                .contains("scope = 'tenant'")
        );
        assert!(
            SqlDialect::Postgres
                .tenant_org_role_by_id()
                .contains("id = $1")
        );
        assert!(
            SqlDialect::Postgres
                .update_org_role_name()
                .contains("updated_at = $2")
        );
        assert!(
            SqlDialect::Postgres
                .org_role_active_member_count()
                .contains("role_code = $2")
        );
        assert!(
            SqlDialect::Postgres
                .org_role_pending_invitation_count()
                .contains("role_code = $2")
        );
        assert!(SqlDialect::Postgres.delete_org_role().contains("id = $1"));
        assert!(
            SqlDialect::Postgres
                .create_email_verification()
                .contains("$5")
        );
        assert!(
            SqlDialect::Postgres
                .invitations_for_org()
                .contains("org_id = $1")
        );
        assert!(SqlDialect::Postgres.revoke_invitation().contains("id = $3"));
        assert!(
            SqlDialect::Postgres
                .invitation_by_hash()
                .contains("token_hash = $1")
        );
        assert!(
            SqlDialect::Postgres
                .organization_by_id()
                .contains("id = $1")
        );
        assert!(SqlDialect::Postgres.accept_invitation().contains("id = $2"));
        assert!(
            SqlDialect::Postgres
                .password_reset_by_hash()
                .contains("token_hash = $1")
        );
        assert!(
            SqlDialect::Postgres
                .mark_password_reset_used()
                .contains("user_id = $3")
        );
        assert!(
            SqlDialect::Postgres
                .update_user_password_hash()
                .contains("updated_at = $2")
        );
        assert!(
            SqlDialect::Postgres
                .email_verification_by_hash()
                .contains("token_hash = $1")
        );
        assert!(
            SqlDialect::Postgres
                .mark_email_verification_verified()
                .contains("user_id = $3")
        );
        assert!(
            SqlDialect::Postgres
                .mark_user_email_verified()
                .contains("updated_at = $2")
        );
        assert!(
            SqlDialect::Postgres
                .create_notification_outbox()
                .contains("$11")
        );
        assert!(
            SqlDialect::Postgres
                .create_notification_delivery_secret()
                .contains("$3")
        );
        assert!(
            SqlDialect::Postgres
                .create_mfa_recovery_code()
                .contains("$4")
        );
        assert!(
            SqlDialect::Postgres
                .revoke_pending_mfa_factors()
                .contains("kind = $3")
        );
        assert!(
            SqlDialect::Postgres
                .pending_mfa_factor_for_user()
                .contains("user_id = $1")
        );
        assert!(
            SqlDialect::Postgres
                .activate_mfa_factor()
                .contains("user_id = $3")
        );
        assert!(
            SqlDialect::Postgres
                .revoke_active_mfa_recovery_codes()
                .contains("user_id = $2")
        );
        assert!(
            SqlDialect::Postgres
                .consume_mfa_recovery_code()
                .contains("code_hash = $3")
        );
        assert!(
            SqlDialect::Postgres
                .create_version_release_event()
                .contains("$5")
        );
        assert!(SqlDialect::Postgres.create_session().contains("$11"));
        assert!(
            SqlDialect::Postgres
                .session_by_token_hash()
                .contains("limit $2")
        );
        assert!(
            SqlDialect::Postgres
                .rotate_session_tokens()
                .contains("refresh_token_hash = $7")
        );
        assert!(
            SqlDialect::Postgres
                .permissions_for_user()
                .contains("$4 = 1")
        );
        assert!(
            SqlDialect::Postgres
                .permissions_list()
                .contains("product_code = $1")
        );
        assert!(
            SqlDialect::Postgres
                .api_token_by_hash()
                .contains("limit $2")
        );
        assert!(
            SqlDialect::Postgres
                .api_tokens_for_org()
                .contains("org_id = $1")
        );
        assert!(SqlDialect::Postgres.revoke_api_token().contains("id = $3"));
        assert!(
            SqlDialect::Postgres
                .create_operation_record()
                .contains("$5")
        );
    }

    #[test]
    fn sqlite_templates_keep_current_repository_semantics() {
        assert!(
            SqlDialect::Sqlite
                .role_permissions_for_role()
                .contains("order by p.scope asc")
        );
        assert!(
            SqlDialect::Sqlite
                .tenant_permission_id_by_code()
                .contains("limit 1")
        );
        assert!(SqlDialect::Sqlite.role_permission_values().contains("?"));
        assert!(
            SqlDialect::Mysql
                .setup_completed_value()
                .contains("`key` = 'completed'")
        );
        assert!(
            SqlDialect::Sqlite
                .complete_setup_run()
                .contains("updated_at = ?")
        );
        assert!(
            SqlDialect::Sqlite
                .append_setup_complete_log()
                .contains("初始化完成状态已写入")
        );
        assert!(SqlDialect::Sqlite.create_setup_run().contains("'running'"));
        assert!(
            SqlDialect::Sqlite
                .append_setup_step_log()
                .contains("values (?, ?, ?, ?, ?)")
        );
        assert!(
            SqlDialect::Sqlite
                .setup_step_logs_for_run()
                .contains("order by id asc")
        );
        assert!(
            SqlDialect::Sqlite
                .purge_notification_delivery_secret()
                .contains("secret_ciphertext = null")
        );
        assert!(
            SqlDialect::Sqlite
                .claim_notification()
                .contains("locked_at = ?")
        );
        assert!(
            SqlDialect::Sqlite
                .mark_notification_delivered()
                .contains("status = 'delivered'")
        );
        assert!(
            SqlDialect::Sqlite
                .notification_attempt_count()
                .contains("limit 1")
        );
        assert!(
            SqlDialect::Sqlite
                .mark_notification_retry()
                .contains("available_at = ?")
        );
        assert!(
            SqlDialect::Sqlite
                .mark_notification_final_failed()
                .contains("status = 'failed'")
        );
        assert!(
            SqlDialect::Sqlite
                .failed_notifications()
                .contains("o.status = 'failed'")
        );
        assert!(
            SqlDialect::Sqlite
                .system_config_upsert()
                .contains("excluded.value_json")
        );
        assert!(
            SqlDialect::Sqlite
                .delete_system_config()
                .contains("key = ?")
        );
        assert!(
            SqlDialect::Sqlite
                .system_menus_list()
                .contains("from system_menus")
        );
        assert!(SqlDialect::Sqlite.setup_runs_list().contains("limit ?"));
        assert!(
            SqlDialect::Mysql
                .traffic_probe_results_for_target()
                .contains("limit ?")
        );
        assert!(
            SqlDialect::Sqlite
                .delete_traffic_probe_target()
                .contains("id = ?")
        );
        assert!(
            SqlDialect::Sqlite
                .acknowledge_traffic_probe_alert()
                .contains("status = 'open'")
        );
        assert!(
            SqlDialect::Sqlite
                .delete_version_package()
                .contains("status != 'active'")
        );
        assert!(
            SqlDialect::Sqlite
                .media_assets_list()
                .contains("from system_media_assets")
        );
        assert!(
            SqlDialect::Sqlite
                .version_packages_list()
                .contains("where deleted_at is null")
        );
        assert!(
            SqlDialect::Sqlite
                .active_version_package_id()
                .contains("status = 'active'")
        );
        assert!(
            SqlDialect::Sqlite
                .create_version_package()
                .contains("returning id")
        );
        assert!(
            !SqlDialect::Mysql
                .create_traffic_probe_target()
                .contains("returning id")
        );
        assert!(
            SqlDialect::Sqlite
                .create_invitation()
                .contains("returning id")
        );
        assert!(
            !SqlDialect::Mysql
                .create_mfa_factor()
                .contains("returning id")
        );
        assert!(
            !SqlDialect::Mysql
                .create_tenant_organization()
                .contains("returning id")
        );
        assert!(
            !SqlDialect::Mysql
                .create_active_user()
                .contains("returning id")
        );
        assert!(
            !SqlDialect::Mysql
                .create_pending_verification_user()
                .contains("returning id")
        );
        assert!(
            SqlDialect::Mysql
                .create_active_membership()
                .contains("values (?, ?, ?, 'active', ?, ?)")
        );
        assert!(SqlDialect::Sqlite.users_count().contains("from iam_users"));
        assert!(
            SqlDialect::Sqlite
                .user_email_count()
                .contains("deleted_at is null")
        );
        assert!(
            SqlDialect::Sqlite
                .organization_code_count()
                .contains("code = ?")
        );
        assert!(SqlDialect::Sqlite.org_users_list().contains("group_concat"));
        assert!(
            SqlDialect::Sqlite
                .org_user_membership_context()
                .contains("then 1 else 0")
        );
        assert!(
            SqlDialect::Sqlite
                .permissions_list()
                .contains("order by scope asc, code asc")
        );
        assert!(
            SqlDialect::Mysql
                .delete_org_user_memberships()
                .contains("user_id = ?")
        );
        assert!(
            SqlDialect::Mysql
                .create_audit_log()
                .contains("values (?, ?, ?, ?, ?, ?, ?)")
        );
        assert!(
            !SqlDialect::Mysql
                .create_notification_outbox()
                .contains("returning id")
        );
        assert!(
            !SqlDialect::Mysql
                .create_mfa_recovery_code()
                .contains("returning id")
        );
        assert!(
            SqlDialect::Sqlite
                .mfa_factors_for_user()
                .contains("kind = 'totp'")
        );
        assert!(
            SqlDialect::Sqlite
                .pending_mfa_factor_for_user()
                .contains("limit 1")
        );
        assert!(
            SqlDialect::Sqlite
                .activate_mfa_factor()
                .contains("status = 'pending'")
        );
        assert!(
            SqlDialect::Sqlite
                .mfa_recovery_codes_for_user()
                .contains("order by id desc")
        );
        assert!(
            SqlDialect::Sqlite
                .consume_mfa_recovery_code()
                .contains("code_hash = ?")
        );
        assert!(
            SqlDialect::Sqlite
                .create_version_release_event()
                .contains("returning id")
        );
        assert!(
            SqlDialect::Mysql
                .system_parameters_list()
                .contains("`key` as `key`")
        );
        assert!(SqlDialect::Sqlite.create_session().contains("values (?, ?"));
        assert!(
            SqlDialect::Sqlite
                .session_by_refresh_hash()
                .contains("limit ?")
        );
        assert!(SqlDialect::Sqlite.permissions_for_user().contains("? = 1"));
        assert!(SqlDialect::Sqlite.api_token_by_hash().contains("limit ?"));
        assert!(
            SqlDialect::Sqlite
                .api_tokens_for_org()
                .contains("org_id = ?")
        );
        assert!(
            SqlDialect::Sqlite
                .tenant_org_role_by_id()
                .contains("limit 1")
        );
        assert!(SqlDialect::Sqlite.invitation_by_hash().contains("limit 1"));
        assert!(
            SqlDialect::Sqlite
                .mark_password_reset_used()
                .contains("status = 'pending'")
        );
        assert!(
            SqlDialect::Sqlite
                .mark_user_email_verified()
                .contains("pending_verification")
        );
        assert!(
            SqlDialect::Mysql
                .delete_org_role()
                .contains("id = ? and org_id = ?")
        );
        assert!(
            SqlDialect::Sqlite
                .prune_operation_records()
                .contains("where created_at < ?")
        );
        assert!(
            SqlDialect::Sqlite
                .operation_records_summary_counts()
                .contains("success_count")
        );
        assert!(
            SqlDialect::Postgres
                .operation_records_summary_paths()
                .contains("limit $13")
        );
        assert!(
            SqlDialect::Mysql
                .operation_records_summary_counts()
                .contains("cast(count(*) as signed)")
        );
        assert!(
            SqlDialect::Mysql
                .operation_records_summary_paths()
                .contains("cast(coalesce(sum")
        );
        assert!(
            SqlDialect::Postgres
                .prune_operation_records()
                .contains("limit $2")
        );
        assert!(
            SqlDialect::Mysql
                .prune_operation_records()
                .contains("as prune_ids")
        );
    }
}
