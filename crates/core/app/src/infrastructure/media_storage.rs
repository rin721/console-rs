use std::path::{Path, PathBuf};
use std::sync::Arc;

use async_trait::async_trait;
use chrono::{DateTime, Utc};
use futures_util::StreamExt;
use object_store::aws::AmazonS3Builder;
use object_store::path::Path as ObjectPath;
use object_store::{ObjectStore, ObjectStoreExt, PutPayload};
use tokio::fs;
use uuid::Uuid;

use crate::app::{AppError, AppResult};
use crate::config::StorageConfig;
use crate::domain::system::{StorageObjectEntry, StorageObjectQuery};
use crate::service::system::{MediaStorage, StoreMediaObjectInput, StoredMediaObject};

#[derive(Clone)]
pub struct LocalMediaStorage {
    root: PathBuf,
}

impl LocalMediaStorage {
    pub fn new(config: StorageConfig) -> Self {
        Self {
            root: PathBuf::from(config.local_dir),
        }
    }
}

#[async_trait]
impl MediaStorage for LocalMediaStorage {
    async fn put(&self, input: StoreMediaObjectInput) -> AppResult<StoredMediaObject> {
        let extension = safe_extension(&input.file_name)
            .or_else(|| extension_from_mime(&input.mime_type))
            .unwrap_or("bin");
        let storage_key = format!("local/{}.{}", Uuid::new_v4(), extension);
        let path = safe_local_path(&self.root, &storage_key)?;
        if let Some(parent) = path.parent() {
            fs::create_dir_all(parent).await?;
        }
        fs::write(&path, &input.bytes).await?;
        Ok(StoredMediaObject {
            storage_key,
            size_bytes: input.bytes.len() as i64,
        })
    }

    async fn delete(&self, storage_key: &str) -> AppResult<()> {
        if !storage_key.starts_with("local/") {
            return Err(AppError::Validation(
                "当前 local storage driver 只能删除 local/ 对象".into(),
            ));
        }
        let path = safe_local_path(&self.root, storage_key)?;
        match fs::remove_file(path).await {
            Ok(()) => Ok(()),
            Err(err) if err.kind() == std::io::ErrorKind::NotFound => Ok(()),
            Err(err) => Err(err.into()),
        }
    }

    async fn list_objects(&self, query: StorageObjectQuery) -> AppResult<Vec<StorageObjectEntry>> {
        let root = self.root.clone();
        tokio::task::spawn_blocking(move || list_local_objects(root, query))
            .await
            .map_err(|err| AppError::Internal(format!("本地对象遍历任务失败：{err}")))?
    }
}

pub struct S3MediaStorage {
    store: Arc<dyn ObjectStore>,
    prefix: String,
}

impl S3MediaStorage {
    pub fn new(config: StorageConfig) -> AppResult<Self> {
        let s3 = config.s3;
        let builder = AmazonS3Builder::new()
            .with_endpoint(s3.endpoint.trim())
            .with_region(s3.region.trim())
            .with_bucket_name(s3.bucket.trim())
            .with_access_key_id(s3.access_key_id.trim())
            .with_secret_access_key(s3.secret_access_key.trim())
            .with_allow_http(s3.allow_http)
            .with_virtual_hosted_style_request(!s3.force_path_style);
        // 这里只创建对象存储客户端，不做 bucket 管理；bucket 生命周期属于部署控制面。
        let store = builder.build().map_err(storage_config_error)?;
        Ok(Self {
            store: Arc::new(store),
            prefix: s3.normalized_prefix(),
        })
    }
}

#[async_trait]
impl MediaStorage for S3MediaStorage {
    async fn put(&self, input: StoreMediaObjectInput) -> AppResult<StoredMediaObject> {
        let extension = safe_extension(&input.file_name)
            .or_else(|| extension_from_mime(&input.mime_type))
            .unwrap_or("bin");
        let object_key =
            join_object_key(&self.prefix, &format!("{}.{}", Uuid::new_v4(), extension));
        let size_bytes = input.bytes.len() as i64;
        let location = ObjectPath::from(object_key.as_str());
        self.store
            .put(&location, PutPayload::from(input.bytes))
            .await
            .map_err(|err| storage_runtime_error("对象存储写入失败", err))?;
        Ok(StoredMediaObject {
            storage_key: format!("s3/{object_key}"),
            size_bytes,
        })
    }

    async fn delete(&self, storage_key: &str) -> AppResult<()> {
        let Some(object_key) = storage_key.strip_prefix("s3/") else {
            return Err(AppError::Validation(
                "当前 S3 storage driver 只能删除 s3/ 对象".into(),
            ));
        };
        let location = ObjectPath::from(object_key);
        match self.store.delete(&location).await {
            Ok(()) => Ok(()),
            Err(object_store::Error::NotFound { .. }) => Ok(()),
            Err(err) => Err(storage_runtime_error("对象存储删除失败", err)),
        }
    }

    async fn list_objects(&self, query: StorageObjectQuery) -> AppResult<Vec<StorageObjectEntry>> {
        let limit = query.limit.unwrap_or(100);
        let prefix = s3_list_prefix(&self.prefix, query.prefix.as_deref())?;
        let location = if prefix.is_empty() {
            None
        } else {
            Some(ObjectPath::from(prefix.as_str()))
        };
        let mut stream = self.store.list(location.as_ref());
        let mut objects = Vec::new();
        while let Some(item) = stream.next().await {
            let meta = item.map_err(|err| storage_runtime_error("对象存储列表读取失败", err))?;
            objects.push(StorageObjectEntry {
                storage_key: format!("s3/{}", meta.location),
                size_bytes: i64::try_from(meta.size).unwrap_or(i64::MAX),
                updated_at: Some(meta.last_modified.to_rfc3339()),
                e_tag: meta.e_tag,
            });
            if objects.len() >= limit {
                break;
            }
        }
        Ok(objects)
    }
}

fn list_local_objects(
    root: PathBuf,
    query: StorageObjectQuery,
) -> AppResult<Vec<StorageObjectEntry>> {
    let limit = query.limit.unwrap_or(100);
    let prefix = query.prefix.unwrap_or_default().replace('\\', "/");
    let mut stack = vec![root.clone()];
    let mut objects = Vec::new();

    while let Some(dir) = stack.pop() {
        let entries = match std::fs::read_dir(&dir) {
            Ok(entries) => entries,
            Err(err) if err.kind() == std::io::ErrorKind::NotFound => continue,
            Err(err) => return Err(err.into()),
        };
        for entry in entries {
            let entry = entry?;
            let path = entry.path();
            let file_type = entry.file_type()?;
            if file_type.is_dir() {
                stack.push(path);
                continue;
            }
            if !file_type.is_file() {
                continue;
            }
            let storage_key = path
                .strip_prefix(&root)
                .map_err(|err| AppError::Internal(format!("本地对象路径越界：{err}")))?
                .to_string_lossy()
                .replace('\\', "/");
            if !prefix.is_empty() && !storage_key.starts_with(&prefix) {
                continue;
            }
            let metadata = entry.metadata()?;
            let updated_at = metadata
                .modified()
                .ok()
                .map(|time| DateTime::<Utc>::from(time).to_rfc3339());
            objects.push(StorageObjectEntry {
                storage_key,
                size_bytes: i64::try_from(metadata.len()).unwrap_or(i64::MAX),
                updated_at,
                e_tag: None,
            });
        }
    }

    objects.sort_by(|left, right| left.storage_key.cmp(&right.storage_key));
    objects.truncate(limit);
    Ok(objects)
}

fn safe_extension(file_name: &str) -> Option<&str> {
    Path::new(file_name)
        .extension()
        .and_then(|value| value.to_str())
        .map(str::trim)
        .filter(|value| {
            !value.is_empty()
                && value.len() <= 12
                && value
                    .chars()
                    .all(|ch| ch.is_ascii_alphanumeric() || ch == '-')
        })
}

fn extension_from_mime(mime_type: &str) -> Option<&'static str> {
    match mime_type {
        "image/png" => Some("png"),
        "image/jpeg" => Some("jpg"),
        "image/webp" => Some("webp"),
        "image/svg+xml" => Some("svg"),
        "text/plain" => Some("txt"),
        "application/json" => Some("json"),
        _ => None,
    }
}

fn safe_local_path(root: &Path, storage_key: &str) -> AppResult<PathBuf> {
    if storage_key.contains("..") || storage_key.starts_with('/') || storage_key.starts_with('\\') {
        return Err(AppError::Validation(
            "本地媒体 storage key 不能包含路径穿越或绝对路径".into(),
        ));
    }
    Ok(root.join(storage_key.replace('\\', "/")))
}

fn join_object_key(prefix: &str, file_name: &str) -> String {
    if prefix.is_empty() {
        file_name.into()
    } else {
        format!("{prefix}/{file_name}")
    }
}

fn s3_list_prefix(configured_prefix: &str, requested_prefix: Option<&str>) -> AppResult<String> {
    let requested = requested_prefix
        .unwrap_or_default()
        .trim()
        .trim_start_matches("s3/")
        .trim_matches('/');
    if requested.contains("..") || requested.starts_with('/') || requested.starts_with('\\') {
        return Err(AppError::Validation(
            "对象列表 prefix 不能包含路径穿越或绝对路径".into(),
        ));
    }
    if configured_prefix.is_empty() || requested.is_empty() {
        return Ok(if requested.is_empty() {
            configured_prefix.into()
        } else {
            requested.into()
        });
    }
    if requested == configured_prefix || requested.starts_with(&format!("{configured_prefix}/")) {
        Ok(requested.into())
    } else {
        Ok(join_object_key(configured_prefix, requested))
    }
}

fn storage_config_error(err: object_store::Error) -> AppError {
    AppError::Internal(format!("对象存储配置无效：{err}"))
}

fn storage_runtime_error(label: &str, err: object_store::Error) -> AppError {
    AppError::Internal(format!("{label}：{err}"))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn s3_object_key_preserves_optional_prefix() {
        assert_eq!(
            join_object_key("tenant-media", "asset.bin"),
            "tenant-media/asset.bin"
        );
        assert_eq!(join_object_key("", "asset.bin"), "asset.bin");
    }

    #[test]
    fn generated_s3_key_is_valid_service_storage_key() {
        let key = format!("s3/{}", join_object_key("media", "asset.bin"));
        assert!(!key.contains(".."));
        assert!(!key.starts_with('/'));
        assert!(!key.starts_with('\\'));
    }
}
