-- MySQL bootstrap schema draft for the Rust platform.
-- This file mirrors the current SQLite schema without enabling MySQL runtime.

create table if not exists setup_state (
  `key` varchar(191) primary key,
  value integer not null,
  updated_at varchar(64) not null
) engine=InnoDB default charset=utf8mb4;

create table if not exists setup_runs (
  id varchar(191) primary key,
  status varchar(64) not null,
  reason text,
  created_at varchar(64) not null,
  updated_at varchar(64) not null
) engine=InnoDB default charset=utf8mb4;

create table if not exists setup_step_logs (
  id integer primary key auto_increment,
  run_id varchar(191) not null,
  step_key varchar(191) not null,
  status varchar(64) not null,
  message text not null,
  created_at varchar(64) not null,
  foreign key(run_id) references setup_runs(id) on delete cascade
) engine=InnoDB default charset=utf8mb4;

create table if not exists iam_organizations (
  id integer primary key auto_increment,
  code varchar(191) not null unique,
  name varchar(255) not null,
  scope varchar(32) not null check (scope in ('platform', 'tenant', 'product')),
  status varchar(64) not null,
  created_at varchar(64) not null,
  updated_at varchar(64) not null,
  deleted_at varchar(64)
) engine=InnoDB default charset=utf8mb4;

create table if not exists iam_users (
  id integer primary key auto_increment,
  email varchar(320) not null unique,
  display_name varchar(255) not null,
  password_hash varchar(255) not null,
  status varchar(64) not null,
  email_verified_at varchar(64),
  created_at varchar(64) not null,
  updated_at varchar(64) not null,
  deleted_at varchar(64)
) engine=InnoDB default charset=utf8mb4;

create table if not exists iam_roles (
  id integer primary key auto_increment,
  org_id integer,
  code varchar(191) not null,
  name varchar(255) not null,
  scope varchar(32) not null check (scope in ('platform', 'tenant', 'product')),
  system_builtin integer not null default 0,
  created_at varchar(64) not null,
  updated_at varchar(64) not null,
  unique(org_id, code),
  foreign key(org_id) references iam_organizations(id) on delete cascade
) engine=InnoDB default charset=utf8mb4;

create table if not exists iam_permissions (
  id integer primary key auto_increment,
  product_code varchar(191) not null,
  scope varchar(32) not null check (scope in ('platform', 'tenant', 'product')),
  code varchar(191) not null,
  name varchar(255) not null,
  created_at varchar(64) not null,
  unique(product_code, scope, code)
) engine=InnoDB default charset=utf8mb4;

create table if not exists iam_role_permissions (
  role_id integer not null,
  permission_id integer not null,
  primary key(role_id, permission_id),
  foreign key(role_id) references iam_roles(id) on delete cascade,
  foreign key(permission_id) references iam_permissions(id) on delete cascade
) engine=InnoDB default charset=utf8mb4;

create table if not exists iam_memberships (
  id integer primary key auto_increment,
  org_id integer,
  user_id integer not null,
  role_code varchar(191) not null,
  status varchar(64) not null,
  created_at varchar(64) not null,
  updated_at varchar(64) not null,
  unique(org_id, user_id, role_code),
  foreign key(org_id) references iam_organizations(id) on delete cascade,
  foreign key(user_id) references iam_users(id) on delete cascade
) engine=InnoDB default charset=utf8mb4;

create table if not exists iam_sessions (
  id varchar(191) primary key,
  session_token_hash varchar(255) not null unique,
  refresh_token_hash varchar(255) unique,
  user_id integer not null,
  org_id integer not null,
  product_code varchar(191) not null,
  client_type varchar(64) not null,
  status varchar(64) not null,
  expires_at varchar(64) not null,
  refresh_expires_at varchar(64),
  revoked_at varchar(64),
  created_at varchar(64) not null,
  updated_at varchar(64) not null,
  foreign key(user_id) references iam_users(id) on delete cascade,
  foreign key(org_id) references iam_organizations(id) on delete cascade
) engine=InnoDB default charset=utf8mb4;

create table if not exists iam_api_tokens (
  id integer primary key auto_increment,
  org_id integer not null,
  user_id integer not null,
  token_hash varchar(255) not null unique,
  token_prefix varchar(32) not null,
  status varchar(64) not null,
  expires_at varchar(64),
  created_at varchar(64) not null,
  revoked_at varchar(64),
  foreign key(org_id) references iam_organizations(id) on delete cascade,
  foreign key(user_id) references iam_users(id) on delete cascade
) engine=InnoDB default charset=utf8mb4;

create table if not exists iam_invitations (
  id integer primary key auto_increment,
  org_id integer not null,
  email varchar(320) not null,
  token_hash varchar(255) not null unique,
  role_code varchar(191) not null default 'owner',
  status varchar(64) not null,
  expires_at varchar(64) not null,
  created_at varchar(64) not null,
  accepted_at varchar(64),
  revoked_at varchar(64),
  foreign key(org_id) references iam_organizations(id) on delete cascade
) engine=InnoDB default charset=utf8mb4;

create table if not exists iam_password_resets (
  id integer primary key auto_increment,
  user_id integer not null,
  token_hash varchar(255) not null unique,
  status varchar(64) not null,
  expires_at varchar(64) not null,
  created_at varchar(64) not null,
  used_at varchar(64),
  foreign key(user_id) references iam_users(id) on delete cascade
) engine=InnoDB default charset=utf8mb4;

create table if not exists iam_email_verifications (
  id integer primary key auto_increment,
  user_id integer not null,
  email varchar(320) not null,
  token_hash varchar(255) not null unique,
  status varchar(64) not null,
  expires_at varchar(64) not null,
  created_at varchar(64) not null,
  verified_at varchar(64),
  foreign key(user_id) references iam_users(id) on delete cascade
) engine=InnoDB default charset=utf8mb4;

create table if not exists iam_mfa_factors (
  id integer primary key auto_increment,
  user_id integer not null,
  kind varchar(64) not null,
  secret_ciphertext text not null,
  status varchar(64) not null,
  created_at varchar(64) not null,
  verified_at varchar(64),
  revoked_at varchar(64),
  foreign key(user_id) references iam_users(id) on delete cascade
) engine=InnoDB default charset=utf8mb4;

create table if not exists iam_mfa_recovery_codes (
  id integer primary key auto_increment,
  user_id integer not null,
  code_hash varchar(255) not null unique,
  code_prefix varchar(32) not null,
  status varchar(64) not null check (status in ('active', 'used', 'revoked')),
  created_at varchar(64) not null,
  used_at varchar(64),
  revoked_at varchar(64),
  foreign key(user_id) references iam_users(id) on delete cascade
) engine=InnoDB default charset=utf8mb4;

create table if not exists iam_audit_logs (
  id integer primary key auto_increment,
  org_id integer,
  user_id integer,
  action varchar(191) not null,
  scope varchar(32) not null,
  product_code varchar(191) not null,
  detail text not null,
  created_at varchar(64) not null,
  foreign key(org_id) references iam_organizations(id) on delete set null,
  foreign key(user_id) references iam_users(id) on delete set null
) engine=InnoDB default charset=utf8mb4;

create table if not exists iam_notification_outbox (
  id integer primary key auto_increment,
  org_id integer,
  user_id integer,
  product_code varchar(191) not null,
  channel varchar(64) not null,
  template_code varchar(191) not null,
  recipient varchar(320) not null,
  related_kind varchar(191) not null,
  related_id integer not null,
  payload_json text not null,
  status varchar(64) not null check (status in ('pending', 'delivered', 'failed', 'cancelled')),
  attempt_count integer not null default 0,
  available_at varchar(64) not null,
  created_at varchar(64) not null,
  locked_at varchar(64),
  delivered_at varchar(64),
  failed_at varchar(64),
  failure_reason text,
  foreign key(org_id) references iam_organizations(id) on delete set null,
  foreign key(user_id) references iam_users(id) on delete set null
) engine=InnoDB default charset=utf8mb4;

create table if not exists iam_notification_delivery_secrets (
  id integer primary key auto_increment,
  outbox_id integer not null unique,
  secret_kind varchar(64) not null check (secret_kind in ('raw_token')),
  secret_ciphertext text,
  status varchar(64) not null check (status in ('pending', 'purged')),
  created_at varchar(64) not null,
  purged_at varchar(64),
  foreign key(outbox_id) references iam_notification_outbox(id) on delete cascade
) engine=InnoDB default charset=utf8mb4;

create table if not exists system_apis (
  id varchar(191) not null,
  method varchar(16) not null,
  path varchar(512) not null,
  tag varchar(191) not null,
  summary varchar(512) not null,
  access varchar(64) not null,
  permission varchar(191),
  scope varchar(32) not null,
  product_code varchar(191) not null,
  created_at varchar(64) not null,
  updated_at varchar(64) not null,
  primary key(method, path)
) engine=InnoDB default charset=utf8mb4;

create table if not exists system_menus (
  id integer primary key auto_increment,
  code varchar(191) not null unique,
  title varchar(255) not null,
  path varchar(512) not null,
  permission varchar(191),
  scope varchar(32) not null,
  sort_order integer not null default 0,
  created_at varchar(64) not null
) engine=InnoDB default charset=utf8mb4;

create table if not exists system_configs (
  `key` varchar(191) primary key,
  value_json text not null,
  updated_at varchar(64) not null
) engine=InnoDB default charset=utf8mb4;

create table if not exists system_dictionaries (
  id integer primary key auto_increment,
  code varchar(191) not null unique,
  name varchar(255) not null,
  created_at varchar(64) not null,
  deleted_at varchar(64)
) engine=InnoDB default charset=utf8mb4;

create table if not exists system_parameters (
  id integer primary key auto_increment,
  `key` varchar(191) not null unique,
  name varchar(255) not null,
  value text not null,
  created_at varchar(64) not null,
  updated_at varchar(64) not null,
  deleted_at varchar(64)
) engine=InnoDB default charset=utf8mb4;

create table if not exists system_operation_records (
  id integer primary key auto_increment,
  actor_user_id integer,
  method varchar(16) not null,
  path varchar(512) not null,
  status integer not null,
  created_at varchar(64) not null
) engine=InnoDB default charset=utf8mb4;

create table if not exists system_server_metrics (
  id integer primary key auto_increment,
  source varchar(191) not null,
  payload_json text not null,
  collected_at varchar(64) not null
) engine=InnoDB default charset=utf8mb4;

create table if not exists system_version_packages (
  id integer primary key auto_increment,
  version_name varchar(255) not null,
  version_code varchar(191) not null,
  manifest_json text not null,
  status varchar(64) not null default 'draft',
  created_at varchar(64) not null,
  published_at varchar(64),
  retired_at varchar(64),
  deleted_at varchar(64)
) engine=InnoDB default charset=utf8mb4;

create table if not exists system_version_release_events (
  id integer primary key auto_increment,
  package_id integer not null,
  previous_active_id integer,
  action varchar(64) not null,
  status varchar(64) not null,
  reason text,
  created_at varchar(64) not null,
  foreign key(package_id) references system_version_packages(id),
  foreign key(previous_active_id) references system_version_packages(id)
) engine=InnoDB default charset=utf8mb4;

create table if not exists system_media_assets (
  id integer primary key auto_increment,
  category varchar(191),
  display_name varchar(255) not null,
  storage_key text not null,
  mime_type varchar(191) not null,
  size_bytes bigint not null,
  created_at varchar(64) not null,
  deleted_at varchar(64)
) engine=InnoDB default charset=utf8mb4;

create table if not exists system_traffic_probe_targets (
  id integer primary key auto_increment,
  name varchar(255) not null,
  url text not null,
  expected_status integer not null,
  status varchar(64) not null,
  created_at varchar(64) not null,
  deleted_at varchar(64)
) engine=InnoDB default charset=utf8mb4;

create table if not exists system_traffic_probe_results (
  id integer primary key auto_increment,
  target_id integer not null,
  status varchar(64) not null,
  detail_json text not null,
  probed_at varchar(64) not null,
  foreign key(target_id) references system_traffic_probe_targets(id) on delete cascade
) engine=InnoDB default charset=utf8mb4;

create table if not exists system_traffic_probe_alerts (
  id integer primary key auto_increment,
  target_id integer not null,
  result_id integer not null unique,
  severity varchar(64) not null check (severity in ('warning', 'critical')),
  status varchar(64) not null check (status in ('open', 'acknowledged', 'resolved')),
  reason text not null,
  detail_json text not null,
  opened_at varchar(64) not null,
  acknowledged_at varchar(64),
  resolved_at varchar(64),
  foreign key(target_id) references system_traffic_probe_targets(id) on delete cascade,
  foreign key(result_id) references system_traffic_probe_results(id) on delete cascade
) engine=InnoDB default charset=utf8mb4;

create index idx_iam_sessions_refresh_token_hash on iam_sessions(refresh_token_hash);
create index idx_iam_invitations_token_hash on iam_invitations(token_hash);
create index idx_iam_mfa_recovery_codes_user_status on iam_mfa_recovery_codes(user_id, status);
create index idx_iam_notification_outbox_status on iam_notification_outbox(status, available_at);
create index idx_iam_notification_outbox_related on iam_notification_outbox(related_kind, related_id);
create index idx_iam_notification_outbox_retry on iam_notification_outbox(status, available_at, attempt_count);
create index idx_iam_notification_delivery_secrets_outbox on iam_notification_delivery_secrets(outbox_id, status);
create index idx_system_traffic_probe_alerts_status on system_traffic_probe_alerts(status, opened_at);
create index idx_system_traffic_probe_alerts_target on system_traffic_probe_alerts(target_id, status);
create index idx_system_version_release_events_package_id on system_version_release_events(package_id);
