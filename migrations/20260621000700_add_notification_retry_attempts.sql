-- 通知 outbox 重试计数只记录投递尝试次数，不记录 raw token、SMTP 密码或响应原文。
alter table iam_notification_outbox
  add column attempt_count integer not null default 0;

create index if not exists idx_iam_notification_outbox_retry
  on iam_notification_outbox(status, available_at, attempt_count);
