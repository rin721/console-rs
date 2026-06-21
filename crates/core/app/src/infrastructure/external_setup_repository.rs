use chrono::Utc;
use sqlx::{MySql, Pool, Postgres, Row};

use crate::app::AppResult;
use crate::domain::setup::{SetupRun, SetupStepLog};
use crate::infrastructure::sql_templates::SqlDialect;
use crate::repository::SetupRepository;

#[derive(Clone)]
pub enum ExternalSetupRepository {
    Postgres(Pool<Postgres>),
    Mysql(Pool<MySql>),
}

impl ExternalSetupRepository {
    pub fn postgres(pool: Pool<Postgres>) -> Self {
        Self::Postgres(pool)
    }

    pub fn mysql(pool: Pool<MySql>) -> Self {
        Self::Mysql(pool)
    }

    fn now() -> String {
        Utc::now().to_rfc3339()
    }

    fn dialect(&self) -> SqlDialect {
        match self {
            Self::Postgres(_) => SqlDialect::Postgres,
            Self::Mysql(_) => SqlDialect::Mysql,
        }
    }
}

#[async_trait::async_trait]
impl SetupRepository for ExternalSetupRepository {
    async fn setup_completed(&self) -> AppResult<bool> {
        let completed = match self {
            Self::Postgres(pool) => {
                sqlx::query_scalar::<_, Option<i64>>(self.dialect().setup_completed_value())
                    .fetch_optional(pool)
                    .await?
            }
            Self::Mysql(pool) => {
                sqlx::query_scalar::<_, Option<i64>>(self.dialect().setup_completed_value())
                    .fetch_optional(pool)
                    .await?
            }
        }
        .flatten()
        .unwrap_or_default();
        Ok(completed == 1)
    }

    async fn complete_setup(&self, run_id: Option<&str>) -> AppResult<bool> {
        let now = Self::now();
        match self {
            Self::Postgres(pool) => {
                let mut tx = pool.begin().await?;
                if let Some(run_id) = run_id {
                    let run_update = sqlx::query(self.dialect().complete_setup_run())
                        .bind(&now)
                        .bind(run_id)
                        .execute(&mut *tx)
                        .await?;
                    if run_update.rows_affected() == 0 {
                        tx.rollback().await?;
                        return Ok(false);
                    }
                    sqlx::query(self.dialect().append_setup_complete_log())
                        .bind(run_id)
                        .bind(&now)
                        .execute(&mut *tx)
                        .await?;
                }
                sqlx::query(self.dialect().setup_state_completed_upsert())
                    .bind(&now)
                    .execute(&mut *tx)
                    .await?;
                tx.commit().await?;
            }
            Self::Mysql(pool) => {
                let mut tx = pool.begin().await?;
                if let Some(run_id) = run_id {
                    let run_update = sqlx::query(self.dialect().complete_setup_run())
                        .bind(&now)
                        .bind(run_id)
                        .execute(&mut *tx)
                        .await?;
                    if run_update.rows_affected() == 0 {
                        tx.rollback().await?;
                        return Ok(false);
                    }
                    sqlx::query(self.dialect().append_setup_complete_log())
                        .bind(run_id)
                        .bind(&now)
                        .execute(&mut *tx)
                        .await?;
                }
                sqlx::query(self.dialect().setup_state_completed_upsert())
                    .bind(&now)
                    .execute(&mut *tx)
                    .await?;
                tx.commit().await?;
            }
        }
        Ok(true)
    }

    async fn create_setup_run(&self, id: &str, reason: Option<&str>) -> AppResult<SetupRun> {
        let now = Self::now();
        match self {
            Self::Postgres(pool) => {
                sqlx::query(self.dialect().create_setup_run())
                    .bind(id)
                    .bind(reason)
                    .bind(&now)
                    .bind(&now)
                    .execute(pool)
                    .await?;
            }
            Self::Mysql(pool) => {
                sqlx::query(self.dialect().create_setup_run())
                    .bind(id)
                    .bind(reason)
                    .bind(&now)
                    .bind(&now)
                    .execute(pool)
                    .await?;
            }
        }
        Ok(SetupRun {
            id: id.into(),
            status: "running".into(),
            reason: reason.map(str::to_owned),
            created_at: now.clone(),
            updated_at: now,
        })
    }

    async fn list_setup_runs(&self, limit: i64) -> AppResult<Vec<SetupRun>> {
        match self {
            Self::Postgres(pool) => {
                let rows = sqlx::query(self.dialect().setup_runs_list())
                    .bind(limit)
                    .fetch_all(pool)
                    .await?;
                Ok(rows
                    .into_iter()
                    .map(|row| SetupRun {
                        id: row.get("id"),
                        status: row.get("status"),
                        reason: row.get("reason"),
                        created_at: row.get("created_at"),
                        updated_at: row.get("updated_at"),
                    })
                    .collect())
            }
            Self::Mysql(pool) => {
                let rows = sqlx::query(self.dialect().setup_runs_list())
                    .bind(limit)
                    .fetch_all(pool)
                    .await?;
                Ok(rows
                    .into_iter()
                    .map(|row| SetupRun {
                        id: row.get("id"),
                        status: row.get("status"),
                        reason: row.get("reason"),
                        created_at: row.get("created_at"),
                        updated_at: row.get("updated_at"),
                    })
                    .collect())
            }
        }
    }

    async fn append_setup_log(
        &self,
        run_id: &str,
        step_key: &str,
        status: &str,
        message: &str,
    ) -> AppResult<()> {
        match self {
            Self::Postgres(pool) => {
                sqlx::query(self.dialect().append_setup_step_log())
                    .bind(run_id)
                    .bind(step_key)
                    .bind(status)
                    .bind(message)
                    .bind(Self::now())
                    .execute(pool)
                    .await?;
            }
            Self::Mysql(pool) => {
                sqlx::query(self.dialect().append_setup_step_log())
                    .bind(run_id)
                    .bind(step_key)
                    .bind(status)
                    .bind(message)
                    .bind(Self::now())
                    .execute(pool)
                    .await?;
            }
        }
        Ok(())
    }

    async fn list_setup_logs(&self, run_id: &str) -> AppResult<Vec<SetupStepLog>> {
        match self {
            Self::Postgres(pool) => {
                let rows = sqlx::query(self.dialect().setup_step_logs_for_run())
                    .bind(run_id)
                    .fetch_all(pool)
                    .await?;
                Ok(rows
                    .into_iter()
                    .map(|row| SetupStepLog {
                        step_key: row.get("step_key"),
                        status: row.get("status"),
                        message: row.get("message"),
                        created_at: row.get("created_at"),
                    })
                    .collect())
            }
            Self::Mysql(pool) => {
                let rows = sqlx::query(self.dialect().setup_step_logs_for_run())
                    .bind(run_id)
                    .fetch_all(pool)
                    .await?;
                Ok(rows
                    .into_iter()
                    .map(|row| SetupStepLog {
                        step_key: row.get("step_key"),
                        status: row.get("status"),
                        message: row.get("message"),
                        created_at: row.get("created_at"),
                    })
                    .collect())
            }
        }
    }
}
