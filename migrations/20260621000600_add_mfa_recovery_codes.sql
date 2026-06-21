create table if not exists iam_mfa_recovery_codes (
  id integer primary key autoincrement,
  user_id integer not null references iam_users(id) on delete cascade,
  code_hash text not null unique,
  code_prefix text not null,
  status text not null check (status in ('active', 'used', 'revoked')),
  created_at text not null,
  used_at text,
  revoked_at text
);

create index if not exists idx_iam_mfa_recovery_codes_user_status
  on iam_mfa_recovery_codes(user_id, status);
