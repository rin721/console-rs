-- IAM 通知 outbox 是邮件/队列基础设施的边界。
-- 这里保存可投递的元数据和 pending 记录关联，禁止保存邀请、重置、邮箱验证 raw token。
create table if not exists iam_notification_outbox (
  id integer primary key autoincrement,
  org_id integer references iam_organizations(id) on delete set null,
  user_id integer references iam_users(id) on delete set null,
  product_code text not null,
  channel text not null,
  template_code text not null,
  recipient text not null,
  related_kind text not null,
  related_id integer not null,
  payload_json text not null,
  status text not null check (status in ('pending', 'delivered', 'failed', 'cancelled')),
  available_at text not null,
  created_at text not null,
  locked_at text,
  delivered_at text,
  failed_at text,
  failure_reason text
);

create index if not exists idx_iam_notification_outbox_status
  on iam_notification_outbox(status, available_at);

create index if not exists idx_iam_notification_outbox_related
  on iam_notification_outbox(related_kind, related_id);
