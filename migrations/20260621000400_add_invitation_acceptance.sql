-- 邀请接受流程需要保留目标租户角色，但仍不保存 raw token。
alter table iam_invitations
  add column role_code text not null default 'owner';

create index if not exists idx_iam_invitations_token_hash
  on iam_invitations(token_hash);
