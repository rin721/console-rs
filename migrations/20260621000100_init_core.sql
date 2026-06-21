-- Aoi[葵] 新 Rust 平台的首个 schema。旧插件、RPC、WS、远程协议相关表不得迁入。
create table if not exists setup_state (
  key text primary key,
  value integer not null,
  updated_at text not null
);

create table if not exists setup_runs (
  id text primary key,
  status text not null,
  reason text,
  created_at text not null,
  updated_at text not null
);

create table if not exists setup_step_logs (
  id integer primary key autoincrement,
  run_id text not null references setup_runs(id) on delete cascade,
  step_key text not null,
  status text not null,
  message text not null,
  created_at text not null
);

create table if not exists iam_organizations (
  id integer primary key autoincrement,
  code text not null unique,
  name text not null,
  scope text not null check (scope in ('platform', 'tenant', 'product')),
  status text not null,
  created_at text not null,
  updated_at text not null,
  deleted_at text
);

create table if not exists iam_users (
  id integer primary key autoincrement,
  email text not null unique,
  display_name text not null,
  password_hash text not null,
  status text not null,
  email_verified_at text,
  created_at text not null,
  updated_at text not null,
  deleted_at text
);

create table if not exists iam_roles (
  id integer primary key autoincrement,
  org_id integer references iam_organizations(id) on delete cascade,
  code text not null,
  name text not null,
  scope text not null check (scope in ('platform', 'tenant', 'product')),
  system_builtin integer not null default 0,
  created_at text not null,
  updated_at text not null,
  unique(org_id, code)
);

create table if not exists iam_permissions (
  id integer primary key autoincrement,
  product_code text not null,
  scope text not null check (scope in ('platform', 'tenant', 'product')),
  code text not null,
  name text not null,
  created_at text not null,
  unique(product_code, scope, code)
);

create table if not exists iam_role_permissions (
  role_id integer not null references iam_roles(id) on delete cascade,
  permission_id integer not null references iam_permissions(id) on delete cascade,
  primary key(role_id, permission_id)
);

create table if not exists iam_memberships (
  id integer primary key autoincrement,
  org_id integer references iam_organizations(id) on delete cascade,
  user_id integer not null references iam_users(id) on delete cascade,
  role_code text not null,
  status text not null,
  created_at text not null,
  updated_at text not null,
  unique(org_id, user_id, role_code)
);

create table if not exists iam_sessions (
  id text primary key,
  session_token_hash text not null unique,
  user_id integer not null references iam_users(id) on delete cascade,
  org_id integer not null references iam_organizations(id) on delete cascade,
  product_code text not null,
  client_type text not null,
  status text not null,
  expires_at text not null,
  revoked_at text,
  created_at text not null,
  updated_at text not null
);

create table if not exists iam_api_tokens (
  id integer primary key autoincrement,
  org_id integer not null references iam_organizations(id) on delete cascade,
  user_id integer not null references iam_users(id) on delete cascade,
  token_hash text not null unique,
  token_prefix text not null,
  status text not null,
  expires_at text,
  created_at text not null,
  revoked_at text
);

create table if not exists iam_invitations (
  id integer primary key autoincrement,
  org_id integer not null references iam_organizations(id) on delete cascade,
  email text not null,
  token_hash text not null unique,
  status text not null,
  expires_at text not null,
  created_at text not null,
  accepted_at text,
  revoked_at text
);

create table if not exists iam_password_resets (
  id integer primary key autoincrement,
  user_id integer not null references iam_users(id) on delete cascade,
  token_hash text not null unique,
  status text not null,
  expires_at text not null,
  created_at text not null,
  used_at text
);

create table if not exists iam_email_verifications (
  id integer primary key autoincrement,
  user_id integer not null references iam_users(id) on delete cascade,
  email text not null,
  token_hash text not null unique,
  status text not null,
  expires_at text not null,
  created_at text not null,
  verified_at text
);

create table if not exists iam_mfa_factors (
  id integer primary key autoincrement,
  user_id integer not null references iam_users(id) on delete cascade,
  kind text not null,
  secret_ciphertext text not null,
  status text not null,
  created_at text not null,
  verified_at text,
  revoked_at text
);

create table if not exists iam_audit_logs (
  id integer primary key autoincrement,
  org_id integer references iam_organizations(id) on delete set null,
  user_id integer references iam_users(id) on delete set null,
  action text not null,
  scope text not null,
  product_code text not null,
  detail text not null,
  created_at text not null
);

create table if not exists system_apis (
  id text not null,
  method text not null,
  path text not null,
  tag text not null,
  summary text not null,
  access text not null,
  permission text,
  scope text not null,
  product_code text not null,
  created_at text not null,
  updated_at text not null,
  primary key(method, path)
);

create table if not exists system_menus (
  id integer primary key autoincrement,
  code text not null unique,
  title text not null,
  path text not null,
  permission text,
  scope text not null,
  sort_order integer not null default 0,
  created_at text not null
);

create table if not exists system_configs (
  key text primary key,
  value_json text not null,
  updated_at text not null
);

create table if not exists system_dictionaries (
  id integer primary key autoincrement,
  code text not null unique,
  name text not null,
  created_at text not null,
  deleted_at text
);

create table if not exists system_parameters (
  id integer primary key autoincrement,
  key text not null unique,
  name text not null,
  value text not null,
  created_at text not null,
  updated_at text not null,
  deleted_at text
);

create table if not exists system_operation_records (
  id integer primary key autoincrement,
  actor_user_id integer,
  method text not null,
  path text not null,
  status integer not null,
  created_at text not null
);

create table if not exists system_server_metrics (
  id integer primary key autoincrement,
  source text not null,
  payload_json text not null,
  collected_at text not null
);

create table if not exists system_version_packages (
  id integer primary key autoincrement,
  version_name text not null,
  version_code text not null,
  manifest_json text not null,
  created_at text not null,
  deleted_at text
);

create table if not exists system_media_assets (
  id integer primary key autoincrement,
  category text,
  display_name text not null,
  storage_key text not null,
  mime_type text not null,
  size_bytes integer not null,
  created_at text not null,
  deleted_at text
);

create table if not exists system_traffic_probe_targets (
  id integer primary key autoincrement,
  name text not null,
  url text not null,
  expected_status integer not null,
  status text not null,
  created_at text not null,
  deleted_at text
);

create table if not exists system_traffic_probe_results (
  id integer primary key autoincrement,
  target_id integer not null references system_traffic_probe_targets(id) on delete cascade,
  status text not null,
  detail_json text not null,
  probed_at text not null
);
