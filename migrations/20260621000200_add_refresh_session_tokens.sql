-- refresh token 只保存哈希值；原始值只通过 HttpOnly Cookie 下发，不进入响应体、日志或测试快照。
alter table iam_sessions add column refresh_token_hash text;
alter table iam_sessions add column refresh_expires_at text;

create unique index if not exists idx_iam_sessions_refresh_token_hash
  on iam_sessions(refresh_token_hash);
