use std::sync::Arc;

use async_trait::async_trait;

use crate::app::{AppError, AppResult};
use crate::config::Settings;
use crate::domain::notification::{
    NotificationDeadLetterEntry, NotificationDeadLetterRecord, NotificationDeadLetterReport,
    NotificationDeliveryMessage, NotificationDrainReport, NotificationOutboxItem,
    NotificationRequeueEntry, NotificationRequeueReport,
};
use crate::repository::NotificationRepository;
use crate::service::crypto_error::CryptoResultExt;
use crypto;

#[async_trait]
pub trait NotificationSender: Send + Sync {
    async fn send(&self, message: &NotificationDeliveryMessage) -> AppResult<()>;
}

pub struct NotificationService {
    settings: Settings,
    repo: Arc<dyn NotificationRepository>,
    sender: Arc<dyn NotificationSender>,
}

impl NotificationService {
    pub fn new(
        settings: Settings,
        repo: Arc<dyn NotificationRepository>,
        sender: Arc<dyn NotificationSender>,
    ) -> Self {
        Self {
            settings,
            repo,
            sender,
        }
    }

    pub async fn drain_once(&self, limit: Option<i64>) -> AppResult<NotificationDrainReport> {
        let limit = limit.unwrap_or(self.settings.notification.batch_size);
        if !(1..=1000).contains(&limit) {
            return Err(AppError::Validation(
                "notification drain limit 必须在 1 到 1000 之间".into(),
            ));
        }

        let items = self
            .repo
            .claim_due_notifications(limit, self.settings.notification.lock_ttl_seconds)
            .await?;
        let mut report = NotificationDrainReport {
            driver: self.settings.notification.driver.to_string(),
            claimed: items.len(),
            delivered: 0,
            retried: 0,
            failed: 0,
        };

        for item in items {
            let notification_id = item.id;
            let message = self.delivery_message(item);
            let send_result = match message {
                Ok(message) => self.sender.send(&message).await,
                Err(err) => Err(err),
            };
            match send_result {
                Ok(()) => {
                    if self
                        .repo
                        .mark_notification_delivered(notification_id)
                        .await?
                    {
                        report.delivered += 1;
                    }
                }
                Err(err) => {
                    let reason = safe_failure_reason(&err);
                    let disposition = self
                        .repo
                        .mark_notification_failed(
                            notification_id,
                            &reason,
                            self.settings.notification.retry_backoff_seconds,
                            self.settings.notification.max_attempts,
                        )
                        .await?;
                    if disposition.retried {
                        report.retried += 1;
                    }
                    if disposition.failed {
                        report.failed += 1;
                    }
                }
            }
        }

        Ok(report)
    }

    pub async fn dead_letters(
        &self,
        limit: Option<i64>,
    ) -> AppResult<NotificationDeadLetterReport> {
        let limit = limit.unwrap_or(100);
        if !(1..=1000).contains(&limit) {
            return Err(AppError::Validation(
                "notification dead-letter limit 必须在 1 到 1000 之间".into(),
            ));
        }
        let entries = self
            .repo
            .list_failed_notifications(limit)
            .await?
            .into_iter()
            .map(dead_letter_entry)
            .collect::<Vec<_>>();
        Ok(NotificationDeadLetterReport {
            driver: self.settings.notification.driver.to_string(),
            total: entries.len(),
            entries,
        })
    }

    pub async fn requeue_failed(&self, limit: Option<i64>) -> AppResult<NotificationRequeueReport> {
        let limit = limit.unwrap_or(100);
        if !(1..=1000).contains(&limit) {
            return Err(AppError::Validation(
                "notification requeue limit 必须在 1 到 1000 之间".into(),
            ));
        }

        let records = self.repo.list_failed_notifications(limit).await?;
        let mut report = NotificationRequeueReport {
            driver: self.settings.notification.driver.to_string(),
            total: records.len(),
            requeued: 0,
            skipped: 0,
            entries: Vec::with_capacity(records.len()),
        };

        for record in records {
            let secret_state = delivery_secret_state(&record);
            if !can_requeue_failed_notification(&record) {
                report.skipped += 1;
                report.entries.push(requeue_entry(
                    record,
                    "skipped",
                    "投递 secret 已清除或缺失，不能安全重排队",
                    secret_state,
                ));
                continue;
            }

            if self.repo.requeue_failed_notification(record.id).await? {
                report.requeued += 1;
                report.entries.push(requeue_entry(
                    record,
                    "requeued",
                    "已恢复为 pending，等待下一轮 notification-drain 投递",
                    secret_state,
                ));
            } else {
                report.skipped += 1;
                report.entries.push(requeue_entry(
                    record,
                    "skipped",
                    "通知状态已变化，未执行重排队",
                    secret_state,
                ));
            }
        }

        Ok(report)
    }

    fn delivery_message(
        &self,
        item: NotificationOutboxItem,
    ) -> AppResult<NotificationDeliveryMessage> {
        let secret_token = item
            .delivery_secret_ciphertext
            .as_deref()
            .map(|ciphertext| {
                crypto::decrypt_secret(ciphertext, &self.settings.notification.delivery_secret_key)
            })
            .transpose()
            .into_app()?;
        Ok(NotificationDeliveryMessage { item, secret_token })
    }
}

fn can_requeue_failed_notification(record: &NotificationDeadLetterRecord) -> bool {
    // 一次性令牌一旦随最终失败被清除，就不能被工具层重建；只允许仍有密文的记录重排队。
    record.delivery_secret_ciphertext.is_some()
}

fn requeue_entry(
    record: NotificationDeadLetterRecord,
    action: &str,
    reason: &str,
    secret_state: String,
) -> NotificationRequeueEntry {
    NotificationRequeueEntry {
        id: record.id,
        org_id: record.org_id,
        user_id: record.user_id,
        product_code: record.product_code,
        channel: record.channel,
        template_code: record.template_code,
        recipient_hint: recipient_hint(&record.recipient),
        related_kind: record.related_kind,
        related_id: record.related_id,
        action: action.into(),
        reason: reason.into(),
        secret_state,
    }
}

fn dead_letter_entry(record: NotificationDeadLetterRecord) -> NotificationDeadLetterEntry {
    let secret_state = delivery_secret_state(&record);
    NotificationDeadLetterEntry {
        id: record.id,
        org_id: record.org_id,
        user_id: record.user_id,
        product_code: record.product_code,
        channel: record.channel,
        template_code: record.template_code,
        recipient_hint: recipient_hint(&record.recipient),
        related_kind: record.related_kind,
        related_id: record.related_id,
        status: record.status,
        created_at: record.created_at,
        failed_at: record.failed_at,
        failure_reason: record.failure_reason.as_deref().map(safe_text_for_report),
        attempt_count: record.attempt_count,
        secret_state,
    }
}

fn recipient_hint(recipient: &str) -> String {
    let trimmed = recipient.trim();
    if let Some((local, domain)) = trimmed.split_once('@') {
        let first = local.chars().next().unwrap_or('*');
        return format!("{first}***@{domain}");
    }
    let mut chars = trimmed.chars();
    let prefix = chars.by_ref().take(2).collect::<String>();
    if prefix.is_empty() {
        "***".into()
    } else {
        format!("{prefix}***")
    }
}

fn delivery_secret_state(record: &NotificationDeadLetterRecord) -> String {
    if record.delivery_secret_ciphertext.is_some() {
        return "pending_secret_present".into();
    }
    match (
        record.delivery_secret_status.as_deref(),
        record.delivery_secret_purged_at.as_deref(),
    ) {
        (Some("purged"), Some(_)) => "purged".into(),
        (Some(status), _) => status.into(),
        (None, _) => "missing_secret_record".into(),
    }
}

fn safe_failure_reason(err: &AppError) -> String {
    safe_text_for_report(&err.to_string())
}

fn safe_text_for_report(value: &str) -> String {
    let mut reason = value.replace(['\r', '\n'], " ");
    if reason.len() > 240 {
        reason.truncate(240);
    }
    reason
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn dead_letter_entry_masks_recipient_and_secret_state() {
        let record = NotificationDeadLetterRecord {
            id: 7,
            org_id: Some(1),
            user_id: Some(2),
            product_code: "console".into(),
            channel: "email".into(),
            template_code: "iam.password_reset.requested".into(),
            recipient: "owner@example.com".into(),
            related_kind: "iam_password_reset".into(),
            related_id: 9,
            status: "failed".into(),
            created_at: "2026-06-21T00:00:00Z".into(),
            failed_at: Some("2026-06-21T00:01:00Z".into()),
            failure_reason: Some("smtp timeout\nwith newline".into()),
            attempt_count: 3,
            delivery_secret_status: Some("purged".into()),
            delivery_secret_ciphertext: None,
            delivery_secret_purged_at: Some("2026-06-21T00:01:01Z".into()),
        };

        let entry = dead_letter_entry(record);

        assert_eq!(entry.recipient_hint, "o***@example.com");
        assert_eq!(entry.secret_state, "purged");
        assert_eq!(
            entry.failure_reason.as_deref(),
            Some("smtp timeout with newline")
        );
    }

    #[test]
    fn requeue_entry_skips_purged_secret() {
        let record = NotificationDeadLetterRecord {
            id: 8,
            org_id: Some(1),
            user_id: Some(2),
            product_code: "console".into(),
            channel: "email".into(),
            template_code: "iam.invitation.created".into(),
            recipient: "member@example.com".into(),
            related_kind: "iam_invitation".into(),
            related_id: 10,
            status: "failed".into(),
            created_at: "2026-06-21T00:00:00Z".into(),
            failed_at: Some("2026-06-21T00:01:00Z".into()),
            failure_reason: Some("smtp timeout".into()),
            attempt_count: 3,
            delivery_secret_status: Some("purged".into()),
            delivery_secret_ciphertext: None,
            delivery_secret_purged_at: Some("2026-06-21T00:01:01Z".into()),
        };

        let secret_state = delivery_secret_state(&record);

        assert!(!can_requeue_failed_notification(&record));
        let entry = requeue_entry(record, "skipped", "投递 secret 已清除或缺失", secret_state);
        assert_eq!(entry.recipient_hint, "m***@example.com");
        assert_eq!(entry.secret_state, "purged");
        assert_eq!(entry.action, "skipped");
    }

    #[test]
    fn requeue_allows_pending_secret_without_exposing_ciphertext() {
        let record = NotificationDeadLetterRecord {
            id: 9,
            org_id: Some(1),
            user_id: Some(2),
            product_code: "console".into(),
            channel: "email".into(),
            template_code: "iam.email_verification.requested".into(),
            recipient: "user@example.com".into(),
            related_kind: "iam_email_verification".into(),
            related_id: 11,
            status: "failed".into(),
            created_at: "2026-06-21T00:00:00Z".into(),
            failed_at: Some("2026-06-21T00:01:00Z".into()),
            failure_reason: Some("smtp timeout".into()),
            attempt_count: 3,
            delivery_secret_status: Some("pending".into()),
            delivery_secret_ciphertext: Some("encrypted-token-placeholder".into()),
            delivery_secret_purged_at: None,
        };

        let secret_state = delivery_secret_state(&record);

        assert!(can_requeue_failed_notification(&record));
        let entry = requeue_entry(record, "requeued", "已恢复为 pending", secret_state);
        assert_eq!(entry.secret_state, "pending_secret_present");
        assert_eq!(entry.action, "requeued");
    }
}
