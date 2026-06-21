//! 共享类型占位 crate。
//!
//! 当前 Aoi[葵] 的应用生命周期错误与结果类型位于 `crates/core/app/src/error.rs`，
//! 与 app 启动、装配和 HTTP 边界保持同层。这里暂不导出业务模型、HTTP DTO、
//! 数据库行结构或前端展示类型，避免 `types` 退化为跨层杂物仓库。
