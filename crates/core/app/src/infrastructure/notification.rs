use std::path::PathBuf;

use async_trait::async_trait;
use lettre::message::{Mailbox, header::ContentType};
use lettre::transport::smtp::authentication::Credentials;
use lettre::{AsyncSmtpTransport, AsyncTransport, Message, Tokio1Executor};
use serde::Serialize;
use tokio::fs;
use tracing::info;
use uuid::Uuid;

use crate::app::{AppError, AppResult};
use crate::config::{QueueNotificationConfig, SmtpNotificationConfig, SmtpTlsMode};
use crate::domain::notification::NotificationDeliveryMessage;
use crate::service::notification::NotificationSender;

#[derive(Default)]
pub struct LogNotificationSender;

#[async_trait]
impl NotificationSender for LogNotificationSender {
    async fn send(&self, message: &NotificationDeliveryMessage) -> AppResult<()> {
        let item = &message.item;
        // 本地 log driver 只确认投递元数据，不输出 payload、raw token 或可恢复的秘密。
        info!(
            notification_id = item.id,
            product_code = %item.product_code,
            channel = %item.channel,
            template_code = %item.template_code,
            recipient_hint = %recipient_hint(&item.recipient),
            related_kind = %item.related_kind,
            related_id = item.related_id,
            "通知 outbox 已由本地 log driver 接收"
        );
        Ok(())
    }
}

pub struct FileNotificationSender {
    local_dir: PathBuf,
}

impl FileNotificationSender {
    pub fn new(local_dir: impl Into<PathBuf>) -> Self {
        Self {
            local_dir: local_dir.into(),
        }
    }
}

pub struct SmtpNotificationSender {
    from: Mailbox,
    mailer: AsyncSmtpTransport<Tokio1Executor>,
}

pub struct QueueNotificationSender {
    dir: PathBuf,
    secret_key: String,
}

impl QueueNotificationSender {
    pub fn new(config: QueueNotificationConfig) -> Self {
        Self {
            dir: config.dir.into(),
            secret_key: config.secret_key,
        }
    }
}

impl SmtpNotificationSender {
    pub fn new(config: SmtpNotificationConfig) -> AppResult<Self> {
        let from = parse_mailbox("notification.smtp.from", &config.from)?;
        let mut builder = match config.tls {
            SmtpTlsMode::StartTls => {
                AsyncSmtpTransport::<Tokio1Executor>::starttls_relay(config.host.trim())
                    .map_err(|err| AppError::Internal(format!("SMTP STARTTLS 配置无效：{err}")))?
            }
            SmtpTlsMode::Tls => AsyncSmtpTransport::<Tokio1Executor>::relay(config.host.trim())
                .map_err(|err| AppError::Internal(format!("SMTP TLS 配置无效：{err}")))?,
            SmtpTlsMode::None => {
                // 只有配置校验允许的非生产环境可以使用明文 SMTP；这里仍然显式表达风险边界。
                AsyncSmtpTransport::<Tokio1Executor>::builder_dangerous(config.host.trim())
            }
        }
        .port(config.port);

        if !config.username.trim().is_empty() {
            builder = builder.credentials(Credentials::new(
                config.username.trim().to_string(),
                config.password.clone(),
            ));
        }

        Ok(Self {
            from,
            mailer: builder.build(),
        })
    }
}

#[async_trait]
impl NotificationSender for SmtpNotificationSender {
    async fn send(&self, message: &NotificationDeliveryMessage) -> AppResult<()> {
        if message.item.channel != "email" {
            return Err(AppError::Internal(format!(
                "SMTP driver 只能投递 email channel，当前为 {}",
                message.item.channel
            )));
        }
        let email = smtp_email(&self.from, message)?;
        self.mailer
            .send(email)
            .await
            .map_err(|err| AppError::Internal(format!("SMTP 通知投递失败：{err}")))?;
        Ok(())
    }
}

#[async_trait]
impl NotificationSender for QueueNotificationSender {
    async fn send(&self, message: &NotificationDeliveryMessage) -> AppResult<()> {
        let item = &message.item;
        let token = message.secret_token.as_deref().ok_or_else(|| {
            AppError::Internal("队列通知投递缺少加密的一次性令牌，已拒绝写入队列".into())
        })?;
        let secret_ciphertext = crypto::encrypt_secret(token, &self.secret_key)
            .map_err(|err| AppError::Internal(format!("队列通知 secret 重新加密失败：{err}")))?;
        let envelope = QueueNotificationEnvelope {
            schema_version: 1,
            notification_id: item.id,
            product_code: &item.product_code,
            channel: &item.channel,
            template_code: &item.template_code,
            recipient: &item.recipient,
            related_kind: &item.related_kind,
            related_id: item.related_id,
            payload_json: &item.payload_json,
            secret_kind: "raw_token",
            secret_ciphertext: &secret_ciphertext,
            created_at: &item.created_at,
        };
        let bytes = serde_json::to_vec_pretty(&envelope)
            .map_err(|err| AppError::Internal(format!("队列通知 JSON 构建失败：{err}")))?;
        fs::create_dir_all(&self.dir).await?;
        let name = format!(
            "notification-{}-{}-{}.json",
            item.id,
            safe_file_segment(&item.template_code),
            Uuid::new_v4().simple()
        );
        let tmp_path = self.dir.join(format!("{name}.tmp"));
        let final_path = self.dir.join(name);
        fs::write(&tmp_path, bytes).await?;
        fs::rename(tmp_path, final_path).await?;
        Ok(())
    }
}

#[async_trait]
impl NotificationSender for FileNotificationSender {
    async fn send(&self, message: &NotificationDeliveryMessage) -> AppResult<()> {
        let item = &message.item;
        let token = message.secret_token.as_deref().ok_or_else(|| {
            AppError::Internal("通知投递缺少加密的一次性令牌，已拒绝写入本地投递文件".into())
        })?;
        fs::create_dir_all(&self.local_dir).await?;
        let path = self.local_dir.join(format!(
            "notification-{}-{}.txt",
            item.id,
            safe_file_segment(&item.template_code)
        ));
        let body = format!(
            "Aoi[葵] 本地通知投递\n\
             模板: {template}\n\
             产品: {product}\n\
             收件人: {recipient}\n\
             关联对象: {related_kind}#{related_id}\n\
             一次性令牌: {token}\n\
             安全提示: 请在账号页面对应表单中输入令牌，不要把令牌放入 URL、日志或截图。\n",
            template = item.template_code,
            product = item.product_code,
            recipient = item.recipient,
            related_kind = item.related_kind,
            related_id = item.related_id,
            token = token,
        );
        fs::write(path, body).await?;
        Ok(())
    }
}

#[derive(Serialize)]
struct QueueNotificationEnvelope<'a> {
    schema_version: u8,
    notification_id: i64,
    product_code: &'a str,
    channel: &'a str,
    template_code: &'a str,
    recipient: &'a str,
    related_kind: &'a str,
    related_id: i64,
    payload_json: &'a str,
    secret_kind: &'a str,
    secret_ciphertext: &'a str,
    created_at: &'a str,
}

fn smtp_email(from: &Mailbox, message: &NotificationDeliveryMessage) -> AppResult<Message> {
    let item = &message.item;
    let to = parse_mailbox("notification recipient", &item.recipient)?;
    let subject = smtp_subject(&item.product_code, &item.template_code);
    let body = smtp_body(message)?;
    Message::builder()
        .from(from.clone())
        .to(to)
        .subject(subject)
        .header(ContentType::TEXT_PLAIN)
        .body(body)
        .map_err(|err| AppError::Internal(format!("SMTP 邮件构建失败：{err}")))
}

fn smtp_subject(product_code: &str, template_code: &str) -> String {
    let action = match template_code {
        "iam.registration.email_verification" => "注册邮箱验证",
        "iam.invitation.created" => "组织邀请",
        "iam.password_reset.requested" => "密码重置",
        "iam.email_verification.requested" => "邮箱验证",
        _ => "通知",
    };
    format!("[{product_code}] {action}")
}

fn smtp_body(message: &NotificationDeliveryMessage) -> AppResult<String> {
    let item = &message.item;
    let token = message.secret_token.as_deref().ok_or_else(|| {
        AppError::Internal("通知投递缺少加密的一次性令牌，已拒绝构建 SMTP 邮件".into())
    })?;
    Ok(format!(
        "产品: {product}\n\
         模板: {template}\n\
         收件人: {recipient}\n\
         关联对象: {related_kind}#{related_id}\n\
         一次性令牌: {token}\n\
         \n\
         安全提示: 请在账号页面对应表单中输入令牌；不要把令牌放入 URL、日志、截图或工单。\n",
        product = item.product_code,
        template = item.template_code,
        recipient = item.recipient,
        related_kind = item.related_kind,
        related_id = item.related_id,
        token = token,
    ))
}

fn parse_mailbox(label: &str, value: &str) -> AppResult<Mailbox> {
    value
        .trim()
        .parse::<Mailbox>()
        .map_err(|err| AppError::Validation(format!("{label} 邮箱地址无效：{err}")))
}

fn recipient_hint(recipient: &str) -> String {
    if let Some((name, domain)) = recipient.split_once('@') {
        let first = name.chars().next().unwrap_or('*');
        return format!("{first}***@{domain}");
    }
    "redacted".into()
}

fn safe_file_segment(value: &str) -> String {
    value
        .chars()
        .map(|ch| {
            if ch.is_ascii_alphanumeric() || matches!(ch, '-' | '_' | '.') {
                ch
            } else {
                '-'
            }
        })
        .collect()
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::domain::notification::{NotificationDeliveryMessage, NotificationOutboxItem};

    #[test]
    fn smtp_body_contains_secret_token_only_after_delivery_decrypts_it() {
        let message = NotificationDeliveryMessage {
            item: notification_item(),
            secret_token: Some("invitation_token_visible_only_to_delivery".into()),
        };

        let body = smtp_body(&message).expect("smtp body");

        assert!(body.contains("invitation_token_visible_only_to_delivery"));
        assert!(body.contains("不要把令牌放入 URL"));
        assert!(!body.contains("secret_ciphertext"));
    }

    #[test]
    fn smtp_body_rejects_missing_secret_token() {
        let message = NotificationDeliveryMessage {
            item: notification_item(),
            secret_token: None,
        };

        let error = smtp_body(&message).expect_err("missing token must fail");

        assert!(error.to_string().contains("缺少加密的一次性令牌"));
    }

    #[test]
    fn smtp_email_rejects_invalid_recipient() {
        let from = parse_mailbox("from", "Console <noreply@example.com>").expect("from");
        let mut item = notification_item();
        item.recipient = "not-an-email".into();
        let message = NotificationDeliveryMessage {
            item,
            secret_token: Some("email_verify_token_sample".into()),
        };

        let error = smtp_email(&from, &message).expect_err("invalid recipient must fail");

        assert!(error.to_string().contains("邮箱地址无效"));
    }

    #[tokio::test]
    async fn queue_sender_writes_encrypted_envelope_without_raw_token() {
        let queue_dir = std::env::temp_dir().join(format!(
            "console-queue-notification-test-{}",
            Uuid::new_v4().simple()
        ));
        let queue_secret = "queue-secret-for-test-at-least-32-bytes";
        let sender = QueueNotificationSender::new(QueueNotificationConfig {
            dir: queue_dir.to_string_lossy().to_string(),
            secret_key: queue_secret.into(),
        });
        let raw_token = "queue_delivery_secret_visible_after_decrypt";
        let message = NotificationDeliveryMessage {
            item: notification_item(),
            secret_token: Some(raw_token.into()),
        };

        sender.send(&message).await.expect("queue send");

        let mut entries = fs::read_dir(&queue_dir).await.expect("read queue dir");
        let entry = entries
            .next_entry()
            .await
            .expect("read queue entry")
            .expect("queue file");
        let body = fs::read_to_string(entry.path()).await.expect("queue body");
        assert!(!body.contains(raw_token));
        assert!(body.contains("\"secret_ciphertext\""));
        let value: serde_json::Value = serde_json::from_str(&body).expect("queue json");
        let ciphertext = value["secret_ciphertext"].as_str().expect("ciphertext");
        let decrypted = crypto::decrypt_secret(ciphertext, queue_secret).expect("decrypt queue");
        assert_eq!(decrypted, raw_token);
        fs::remove_dir_all(&queue_dir)
            .await
            .expect("cleanup queue dir");
    }

    fn notification_item() -> NotificationOutboxItem {
        NotificationOutboxItem {
            id: 1,
            org_id: Some(1),
            user_id: Some(1),
            product_code: "console".into(),
            channel: "email".into(),
            template_code: "iam.invitation.created".into(),
            recipient: "user@example.com".into(),
            related_kind: "iam_invitation".into(),
            related_id: 10,
            payload_json: "{}".into(),
            status: "processing".into(),
            available_at: "2026-06-21T00:00:00Z".into(),
            created_at: "2026-06-21T00:00:00Z".into(),
            locked_at: None,
            delivered_at: None,
            failed_at: None,
            failure_reason: None,
            attempt_count: 0,
            delivery_secret_ciphertext: None,
        }
    }
}
