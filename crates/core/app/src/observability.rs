use crate::config::ObservabilityConfig;

pub fn init(cfg: &ObservabilityConfig) -> anyhow::Result<()> {
    let filter = tracing_subscriber::EnvFilter::try_from_default_env()
        .unwrap_or_else(|_| tracing_subscriber::EnvFilter::new(cfg.level.clone()));

    let builder = tracing_subscriber::fmt().with_env_filter(filter);
    if cfg.format.eq_ignore_ascii_case("json") {
        builder
            .json()
            .try_init()
            .map_err(|err| anyhow::anyhow!("初始化 JSON 日志失败：{err}"))?;
    } else {
        builder
            .try_init()
            .map_err(|err| anyhow::anyhow!("初始化日志失败：{err}"))?;
    }
    Ok(())
}
