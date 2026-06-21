-- 通知投递 secret vault 只保存 worker 必需的一次性令牌密文。
-- raw token 禁止进入 outbox payload、HTTP 响应、日志和 URL；投递完成后密文会被清空。
create table if not exists iam_notification_delivery_secrets (
  id integer primary key autoincrement,
  outbox_id integer not null unique references iam_notification_outbox(id) on delete cascade,
  secret_kind text not null check (secret_kind in ('raw_token')),
  secret_ciphertext text,
  status text not null check (status in ('pending', 'purged')),
  created_at text not null,
  purged_at text
);

create index if not exists idx_iam_notification_delivery_secrets_outbox
  on iam_notification_delivery_secrets(outbox_id, status);
