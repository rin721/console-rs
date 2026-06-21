use chrono::{Duration, Utc};
use sqlx::mysql::MySqlRow;
use sqlx::postgres::PgRow;
use sqlx::{MySql, Pool, Postgres, Row};

use crate::app::AppResult;
use crate::domain::notification::{
    NotificationDeadLetterRecord, NotificationFailureDisposition, NotificationOutboxItem,
};
use crate::infrastructure::sql_templates::SqlDialect;
use crate::repository::NotificationRepository;

#[derive(Clone)]
pub enum ExternalNotificationRepository {
    Postgres(Pool<Postgres>),
    Mysql(Pool<MySql>),
}

impl ExternalNotificationRepository {
    pub fn postgres(pool: Pool<Postgres>) -> Self {
        Self::Postgres(pool)
    }

    pub fn mysql(pool: Pool<MySql>) -> Self {
        Self::Mysql(pool)
    }

    fn dialect(&self) -> SqlDialect {
        match self {
            Self::Postgres(_) => SqlDialect::Postgres,
            Self::Mysql(_) => SqlDialect::Mysql,
        }
    }

    fn now() -> String {
        Utc::now().to_rfc3339()
    }
}

#[async_trait::async_trait]
impl NotificationRepository for ExternalNotificationRepository {
    async fn claim_due_notifications(
        &self,
        limit: i64,
        lock_ttl_seconds: i64,
    ) -> AppResult<Vec<NotificationOutboxItem>> {
        let now = Utc::now();
        let now_text = now.to_rfc3339();
        let stale_locked_before = (now - Duration::seconds(lock_ttl_seconds.max(1))).to_rfc3339();

        match self {
            Self::Postgres(pool) => {
                let mut tx = pool.begin().await?;
                let rows = sqlx::query(self.dialect().due_notifications())
                    .bind(&now_text)
                    .bind(&stale_locked_before)
                    .bind(limit)
                    .fetch_all(&mut *tx)
                    .await?;
                for row in &rows {
                    let id: i64 = row.get("id");
                    sqlx::query(self.dialect().claim_notification())
                        .bind(&now_text)
                        .bind(id)
                        .execute(&mut *tx)
                        .await?;
                }
                tx.commit().await?;
                Ok(rows.into_iter().map(notification_from_pg_row).collect())
            }
            Self::Mysql(pool) => {
                let mut tx = pool.begin().await?;
                let rows = sqlx::query(self.dialect().due_notifications())
                    .bind(&now_text)
                    .bind(&stale_locked_before)
                    .bind(limit)
                    .fetch_all(&mut *tx)
                    .await?;
                for row in &rows {
                    let id: i64 = row.get("id");
                    sqlx::query(self.dialect().claim_notification())
                        .bind(&now_text)
                        .bind(id)
                        .execute(&mut *tx)
                        .await?;
                }
                tx.commit().await?;
                Ok(rows.into_iter().map(notification_from_mysql_row).collect())
            }
        }
    }

    async fn mark_notification_delivered(&self, notification_id: i64) -> AppResult<bool> {
        let now = Self::now();
        match self {
            Self::Postgres(pool) => {
                let mut tx = pool.begin().await?;
                let result = sqlx::query(self.dialect().mark_notification_delivered())
                    .bind(&now)
                    .bind(notification_id)
                    .execute(&mut *tx)
                    .await?;
                sqlx::query(self.dialect().purge_notification_delivery_secret())
                    .bind(&now)
                    .bind(notification_id)
                    .execute(&mut *tx)
                    .await?;
                tx.commit().await?;
                Ok(result.rows_affected() > 0)
            }
            Self::Mysql(pool) => {
                let mut tx = pool.begin().await?;
                let result = sqlx::query(self.dialect().mark_notification_delivered())
                    .bind(&now)
                    .bind(notification_id)
                    .execute(&mut *tx)
                    .await?;
                sqlx::query(self.dialect().purge_notification_delivery_secret())
                    .bind(&now)
                    .bind(notification_id)
                    .execute(&mut *tx)
                    .await?;
                tx.commit().await?;
                Ok(result.rows_affected() > 0)
            }
        }
    }

    async fn mark_notification_failed(
        &self,
        notification_id: i64,
        failure_reason: &str,
        retry_backoff_seconds: i64,
        max_attempts: i64,
    ) -> AppResult<NotificationFailureDisposition> {
        let now = Utc::now();
        let now_text = now.to_rfc3339();
        let retry_at = (now + Duration::seconds(retry_backoff_seconds.max(1))).to_rfc3339();
        let max_attempts = max_attempts.max(1);

        match self {
            Self::Postgres(pool) => {
                let mut tx = pool.begin().await?;
                let attempt_count = sqlx::query_scalar::<_, Option<i64>>(
                    self.dialect().notification_attempt_count(),
                )
                .bind(notification_id)
                .fetch_optional(&mut *tx)
                .await?
                .flatten();
                let Some(attempt_count) = attempt_count else {
                    tx.commit().await?;
                    return Ok(NotificationFailureDisposition {
                        retried: false,
                        failed: false,
                    });
                };
                let update = NotificationFailureUpdate {
                    notification_id,
                    failure_reason,
                    now_text: &now_text,
                    retry_at: &retry_at,
                    next_attempt_count: attempt_count + 1,
                    max_attempts,
                };
                let disposition =
                    mark_failed_in_postgres_tx(&mut tx, self.dialect(), update).await?;
                tx.commit().await?;
                Ok(disposition)
            }
            Self::Mysql(pool) => {
                let mut tx = pool.begin().await?;
                let attempt_count = sqlx::query_scalar::<_, Option<i64>>(
                    self.dialect().notification_attempt_count(),
                )
                .bind(notification_id)
                .fetch_optional(&mut *tx)
                .await?
                .flatten();
                let Some(attempt_count) = attempt_count else {
                    tx.commit().await?;
                    return Ok(NotificationFailureDisposition {
                        retried: false,
                        failed: false,
                    });
                };
                let update = NotificationFailureUpdate {
                    notification_id,
                    failure_reason,
                    now_text: &now_text,
                    retry_at: &retry_at,
                    next_attempt_count: attempt_count + 1,
                    max_attempts,
                };
                let disposition = mark_failed_in_mysql_tx(&mut tx, self.dialect(), update).await?;
                tx.commit().await?;
                Ok(disposition)
            }
        }
    }

    async fn list_failed_notifications(
        &self,
        limit: i64,
    ) -> AppResult<Vec<NotificationDeadLetterRecord>> {
        match self {
            Self::Postgres(pool) => {
                let rows = sqlx::query(self.dialect().failed_notifications())
                    .bind(limit)
                    .fetch_all(pool)
                    .await?;
                Ok(rows.into_iter().map(dead_letter_from_pg_row).collect())
            }
            Self::Mysql(pool) => {
                let rows = sqlx::query(self.dialect().failed_notifications())
                    .bind(limit)
                    .fetch_all(pool)
                    .await?;
                Ok(rows.into_iter().map(dead_letter_from_mysql_row).collect())
            }
        }
    }

    async fn requeue_failed_notification(&self, notification_id: i64) -> AppResult<bool> {
        let now = Self::now();
        match self {
            Self::Postgres(pool) => {
                let result = sqlx::query(self.dialect().requeue_failed_notification())
                    .bind(&now)
                    .bind(notification_id)
                    .execute(pool)
                    .await?;
                Ok(result.rows_affected() > 0)
            }
            Self::Mysql(pool) => {
                let result = sqlx::query(self.dialect().requeue_failed_notification())
                    .bind(&now)
                    .bind(notification_id)
                    .execute(pool)
                    .await?;
                Ok(result.rows_affected() > 0)
            }
        }
    }
}

struct NotificationFailureUpdate<'a> {
    notification_id: i64,
    failure_reason: &'a str,
    now_text: &'a str,
    retry_at: &'a str,
    next_attempt_count: i64,
    max_attempts: i64,
}

async fn mark_failed_in_postgres_tx(
    tx: &mut sqlx::Transaction<'_, Postgres>,
    dialect: SqlDialect,
    update: NotificationFailureUpdate<'_>,
) -> AppResult<NotificationFailureDisposition> {
    if update.next_attempt_count < update.max_attempts {
        sqlx::query(dialect.mark_notification_retry())
            .bind(update.next_attempt_count)
            .bind(update.retry_at)
            .bind(update.now_text)
            .bind(update.failure_reason)
            .bind(update.notification_id)
            .execute(&mut **tx)
            .await?;
        return Ok(NotificationFailureDisposition {
            retried: true,
            failed: false,
        });
    }

    sqlx::query(dialect.mark_notification_final_failed())
        .bind(update.next_attempt_count)
        .bind(update.now_text)
        .bind(update.failure_reason)
        .bind(update.notification_id)
        .execute(&mut **tx)
        .await?;
    sqlx::query(dialect.purge_notification_delivery_secret())
        .bind(update.now_text)
        .bind(update.notification_id)
        .execute(&mut **tx)
        .await?;
    Ok(NotificationFailureDisposition {
        retried: false,
        failed: true,
    })
}

async fn mark_failed_in_mysql_tx(
    tx: &mut sqlx::Transaction<'_, MySql>,
    dialect: SqlDialect,
    update: NotificationFailureUpdate<'_>,
) -> AppResult<NotificationFailureDisposition> {
    if update.next_attempt_count < update.max_attempts {
        sqlx::query(dialect.mark_notification_retry())
            .bind(update.next_attempt_count)
            .bind(update.retry_at)
            .bind(update.now_text)
            .bind(update.failure_reason)
            .bind(update.notification_id)
            .execute(&mut **tx)
            .await?;
        return Ok(NotificationFailureDisposition {
            retried: true,
            failed: false,
        });
    }

    sqlx::query(dialect.mark_notification_final_failed())
        .bind(update.next_attempt_count)
        .bind(update.now_text)
        .bind(update.failure_reason)
        .bind(update.notification_id)
        .execute(&mut **tx)
        .await?;
    sqlx::query(dialect.purge_notification_delivery_secret())
        .bind(update.now_text)
        .bind(update.notification_id)
        .execute(&mut **tx)
        .await?;
    Ok(NotificationFailureDisposition {
        retried: false,
        failed: true,
    })
}

fn notification_from_pg_row(row: PgRow) -> NotificationOutboxItem {
    NotificationOutboxItem {
        id: row.get("id"),
        org_id: row.get("org_id"),
        user_id: row.get("user_id"),
        product_code: row.get("product_code"),
        channel: row.get("channel"),
        template_code: row.get("template_code"),
        recipient: row.get("recipient"),
        related_kind: row.get("related_kind"),
        related_id: row.get("related_id"),
        payload_json: row.get("payload_json"),
        status: row.get("status"),
        available_at: row.get("available_at"),
        created_at: row.get("created_at"),
        locked_at: row.get("locked_at"),
        delivered_at: row.get("delivered_at"),
        failed_at: row.get("failed_at"),
        failure_reason: row.get("failure_reason"),
        attempt_count: row.get("attempt_count"),
        delivery_secret_ciphertext: row.get("delivery_secret_ciphertext"),
    }
}

fn notification_from_mysql_row(row: MySqlRow) -> NotificationOutboxItem {
    NotificationOutboxItem {
        id: row.get("id"),
        org_id: row.get("org_id"),
        user_id: row.get("user_id"),
        product_code: row.get("product_code"),
        channel: row.get("channel"),
        template_code: row.get("template_code"),
        recipient: row.get("recipient"),
        related_kind: row.get("related_kind"),
        related_id: row.get("related_id"),
        payload_json: row.get("payload_json"),
        status: row.get("status"),
        available_at: row.get("available_at"),
        created_at: row.get("created_at"),
        locked_at: row.get("locked_at"),
        delivered_at: row.get("delivered_at"),
        failed_at: row.get("failed_at"),
        failure_reason: row.get("failure_reason"),
        attempt_count: row.get("attempt_count"),
        delivery_secret_ciphertext: row.get("delivery_secret_ciphertext"),
    }
}

fn dead_letter_from_pg_row(row: PgRow) -> NotificationDeadLetterRecord {
    NotificationDeadLetterRecord {
        id: row.get("id"),
        org_id: row.get("org_id"),
        user_id: row.get("user_id"),
        product_code: row.get("product_code"),
        channel: row.get("channel"),
        template_code: row.get("template_code"),
        recipient: row.get("recipient"),
        related_kind: row.get("related_kind"),
        related_id: row.get("related_id"),
        status: row.get("status"),
        created_at: row.get("created_at"),
        failed_at: row.get("failed_at"),
        failure_reason: row.get("failure_reason"),
        attempt_count: row.get("attempt_count"),
        delivery_secret_status: row.get("delivery_secret_status"),
        delivery_secret_ciphertext: row.get("delivery_secret_ciphertext"),
        delivery_secret_purged_at: row.get("delivery_secret_purged_at"),
    }
}

fn dead_letter_from_mysql_row(row: MySqlRow) -> NotificationDeadLetterRecord {
    NotificationDeadLetterRecord {
        id: row.get("id"),
        org_id: row.get("org_id"),
        user_id: row.get("user_id"),
        product_code: row.get("product_code"),
        channel: row.get("channel"),
        template_code: row.get("template_code"),
        recipient: row.get("recipient"),
        related_kind: row.get("related_kind"),
        related_id: row.get("related_id"),
        status: row.get("status"),
        created_at: row.get("created_at"),
        failed_at: row.get("failed_at"),
        failure_reason: row.get("failure_reason"),
        attempt_count: row.get("attempt_count"),
        delivery_secret_status: row.get("delivery_secret_status"),
        delivery_secret_ciphertext: row.get("delivery_secret_ciphertext"),
        delivery_secret_purged_at: row.get("delivery_secret_purged_at"),
    }
}
