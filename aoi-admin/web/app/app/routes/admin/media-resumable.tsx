import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ArrowLeft,
  CheckCircle2,
  FileArchive,
  FolderTree,
  HardDrive,
  RefreshCw,
  RotateCcw,
  UploadCloud,
  XCircle,
} from "lucide-react";
import { useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router";

import { SelectField, type SelectOption } from "~/components/aoi/patterns/SelectField";
import { StateBlock } from "~/components/aoi/patterns/StateBlock";
import { Badge } from "~/components/aoi/primitives/Badge";
import { Button } from "~/components/aoi/primitives/Button";
import { ApiError } from "~/lib/api/client";
import { queryKeys } from "~/lib/api/query-keys";
import { systemApi } from "~/lib/api/system";
import type {
  SystemMediaAsset,
  SystemMediaCategory,
  SystemMediaResumableCheckResult,
} from "~/lib/api/types";

const chunkSizeBytes = 1024 * 1024;
const defaultUploadMaxBytes = 20 * 1024 * 1024;

type UploadStatus =
  | "aborted"
  | "checking"
  | "completed"
  | "error"
  | "hashing"
  | "idle"
  | "ready"
  | "uploading";

type Notice = {
  description: string;
  intent: "danger" | "info";
  title: string;
};

type FlatCategory = {
  category: SystemMediaCategory;
  depth: number;
};

export default function AdminMediaResumableRoute() {
  const { i18n, t } = useTranslation();
  const queryClient = useQueryClient();
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const checkAbortRef = useRef<AbortController | null>(null);
  const uploadAbortRef = useRef<AbortController | null>(null);

  const [categoryId, setCategoryId] = useState("");
  const [checkResult, setCheckResult] = useState<SystemMediaResumableCheckResult | null>(null);
  const [fileHash, setFileHash] = useState("");
  const [lastAsset, setLastAsset] = useState<SystemMediaAsset | null>(null);
  const [notice, setNotice] = useState<Notice | null>(null);
  const [progress, setProgress] = useState(0);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [status, setStatus] = useState<UploadStatus>("idle");

  const categoryQuery = useQuery({
    queryFn: ({ signal }) => systemApi.listMediaCategories({ signal }),
    queryKey: queryKeys.system.mediaCategories(i18n.language),
  });

  const uploadStatusQuery = useQuery({
    queryFn: ({ signal }) => systemApi.listMediaAssets({ page: 1, pageSize: 1 }, { signal }),
    queryKey: queryKeys.system.mediaAssets(i18n.language, 1, 1, {}),
  });

  useEffect(
    () => () => {
      checkAbortRef.current?.abort();
      uploadAbortRef.current?.abort();
    },
    [],
  );

  const flatCategories = useMemo(
    () => flattenCategories(categoryQuery.data?.items ?? []),
    [categoryQuery.data?.items],
  );

  const categoryOptions = useMemo<SelectOption[]>(
    () => [
      { label: t("admin.mediaResumable.fields.categoryRoot"), value: "" },
      ...flatCategories.map(({ category, depth }) => ({
        label: `${depth > 0 ? `${"  ".repeat(depth)}` : ""}${
          category.name || t("admin.mediaResumable.fields.unknownCategory")
        }`,
        value: String(category.id),
      })),
    ],
    [flatCategories, t],
  );

  const uploadMaxBytes =
    checkResult?.uploadMaxBytes ?? uploadStatusQuery.data?.uploadMaxBytes ?? defaultUploadMaxBytes;
  const uploadMaxMb =
    checkResult?.uploadMaxMb ?? uploadStatusQuery.data?.uploadMaxMb ?? uploadMaxBytes / 1024 / 1024;
  const uploadUnavailable =
    checkResult?.uploadUnavailable ?? uploadStatusQuery.data?.uploadUnavailable ?? false;
  const storageStatus =
    checkResult?.storageStatus ??
    uploadStatusQuery.data?.storageStatus ??
    categoryQuery.data?.storageStatus;
  const chunkTotal =
    checkResult?.session.chunkTotal ??
    (selectedFile ? Math.max(1, Math.ceil(selectedFile.size / chunkSizeBytes)) : 0);
  const boundedProgress = normalizeProgress(progress || checkResult?.progress || 0);
  const busy = status === "hashing" || status === "checking" || status === "uploading";
  const canUpload = Boolean(selectedFile && checkResult && !uploadUnavailable && !busy);
  const canAbort = Boolean(
    checkResult &&
    status !== "completed" &&
    status !== "uploading" &&
    status !== "hashing" &&
    status !== "checking",
  );

  const refreshStatus = () => {
    void categoryQuery.refetch();
    void uploadStatusQuery.refetch();
  };

  const openFilePicker = () => {
    fileInputRef.current?.click();
  };

  const resetUpload = () => {
    checkAbortRef.current?.abort();
    uploadAbortRef.current?.abort();
    if (fileInputRef.current) {
      fileInputRef.current.value = "";
    }
    setCheckResult(null);
    setFileHash("");
    setLastAsset(null);
    setNotice(null);
    setProgress(0);
    setSelectedFile(null);
    setStatus("idle");
  };

  const prepareFile = async (file: File) => {
    checkAbortRef.current?.abort();
    const controller = new AbortController();
    checkAbortRef.current = controller;
    setCheckResult(null);
    setFileHash("");
    setLastAsset(null);
    setNotice(null);
    setProgress(0);
    setSelectedFile(file);

    if (file.size <= 0) {
      setStatus("error");
      setNotice({
        description: t("admin.mediaResumable.messages.emptyFileDescription"),
        intent: "danger",
        title: t("admin.mediaResumable.messages.emptyFileTitle"),
      });
      return;
    }

    if (uploadMaxBytes > 0 && file.size > uploadMaxBytes) {
      setStatus("error");
      setNotice({
        description: t("admin.mediaResumable.messages.fileTooLargeDescription", {
          limit: formatUploadLimit(uploadMaxMb, i18n.language, t),
          size: formatBytes(file.size, i18n.language, t),
        }),
        intent: "danger",
        title: t("admin.mediaResumable.messages.fileTooLargeTitle"),
      });
      return;
    }

    if (!hasWebCrypto()) {
      setStatus("error");
      setNotice({
        description: t("admin.mediaResumable.messages.hashUnsupportedDescription"),
        intent: "danger",
        title: t("admin.mediaResumable.messages.hashUnsupportedTitle"),
      });
      return;
    }

    try {
      setStatus("hashing");
      const hash = await sha256Blob(file);
      const total = Math.max(1, Math.ceil(file.size / chunkSizeBytes));
      setFileHash(hash);
      setStatus("checking");
      const result = await systemApi.checkMediaResumableUpload(
        {
          categoryId: categoryId || undefined,
          chunkSize: chunkSizeBytes,
          chunkTotal: total,
          fileHash: hash,
          fileName: file.name,
          sizeBytes: file.size,
        },
        { signal: controller.signal },
      );
      setCheckResult(result);
      setProgress(result.progress);

      if (result.session.status === "completed" && result.asset) {
        setLastAsset(result.asset);
        setStatus("completed");
        setNotice({
          description: t("admin.mediaResumable.messages.duplicateDescription"),
          intent: "info",
          title: t("admin.mediaResumable.messages.duplicateTitle"),
        });
        return;
      }

      setStatus("ready");
      setNotice({
        description:
          result.uploadedChunks.length > 0
            ? t("admin.mediaResumable.messages.resumeDescription", {
                count: result.uploadedChunks.length,
              })
            : t("admin.mediaResumable.messages.readyDescription", {
                count: result.missingChunks.length,
              }),
        intent: "info",
        title:
          result.uploadedChunks.length > 0
            ? t("admin.mediaResumable.messages.resumeTitle")
            : t("admin.mediaResumable.messages.readyTitle"),
      });
    } catch (error) {
      if (controller.signal.aborted) {
        return;
      }
      setStatus("error");
      setNotice({
        description: apiErrorDescription(error, t),
        intent: "danger",
        title: t("admin.mediaResumable.messages.checkFailedTitle"),
      });
    } finally {
      if (checkAbortRef.current === controller) {
        checkAbortRef.current = null;
      }
    }
  };

  const uploadFile = async () => {
    if (!(selectedFile && checkResult)) {
      return;
    }
    if (uploadUnavailable) {
      setNotice({
        description: t("admin.mediaResumable.messages.storageUnavailableDescription"),
        intent: "danger",
        title: t("admin.mediaResumable.messages.storageUnavailableTitle"),
      });
      return;
    }

    uploadAbortRef.current?.abort();
    const controller = new AbortController();
    uploadAbortRef.current = controller;

    try {
      setStatus("uploading");
      setNotice(null);

      let currentCheck = checkResult;
      const missingChunks = [...currentCheck.missingChunks].sort((left, right) => left - right);
      for (const chunkIndex of missingChunks) {
        const start = chunkIndex * currentCheck.chunkSize;
        const end = Math.min(start + currentCheck.chunkSize, selectedFile.size);
        const chunk = selectedFile.slice(start, end);
        const chunkHash = await sha256Blob(chunk);
        const chunkResult = await systemApi.uploadMediaChunk(
          chunk,
          {
            chunkHash,
            chunkIndex,
            chunkTotal: currentCheck.session.chunkTotal,
            fileHash: currentCheck.session.fileHash,
            fileName: currentCheck.session.fileName,
            sessionId: currentCheck.session.id,
          },
          { signal: controller.signal },
        );

        currentCheck = {
          ...currentCheck,
          missingChunks: chunkResult.missingChunks,
          progress: chunkResult.progress,
          storageStatus: chunkResult.storageStatus,
          uploadedChunks: chunkResult.uploadedChunks,
        };
        setCheckResult(currentCheck);
        setProgress(chunkResult.progress);
      }

      const complete = await systemApi.completeMediaResumableUpload(
        {
          fileHash: currentCheck.session.fileHash,
          sessionId: currentCheck.session.id,
        },
        { signal: controller.signal },
      );
      setLastAsset(complete.asset);
      setCheckResult({
        ...currentCheck,
        asset: complete.asset,
        missingChunks: [],
        progress: 100,
        session: {
          ...currentCheck.session,
          finalAssetId: complete.asset.id,
          status: "completed",
        },
        storageStatus: complete.storageStatus,
      });
      setProgress(100);
      setStatus("completed");
      setNotice({
        description: t("admin.mediaResumable.messages.uploadCompletedDescription", {
          name: complete.asset.displayName || complete.asset.originalName,
        }),
        intent: "info",
        title: t("admin.mediaResumable.messages.uploadCompletedTitle"),
      });
      void queryClient.invalidateQueries({ queryKey: ["system", "media", "assets"] });
    } catch (error) {
      if (controller.signal.aborted) {
        return;
      }
      setStatus("error");
      setNotice({
        description: apiErrorDescription(error, t),
        intent: "danger",
        title: t("admin.mediaResumable.messages.uploadFailedTitle"),
      });
    } finally {
      if (uploadAbortRef.current === controller) {
        uploadAbortRef.current = null;
      }
    }
  };

  const abortUpload = async () => {
    if (!checkResult) {
      return;
    }
    const controller = new AbortController();

    try {
      setStatus("checking");
      const abortResult = await systemApi.abortMediaResumableUpload(
        {
          fileHash: checkResult.session.fileHash,
          sessionId: checkResult.session.id,
        },
        { signal: controller.signal },
      );
      setCheckResult({
        ...checkResult,
        missingChunks: [],
        progress: 0,
        session: {
          ...checkResult.session,
          status: "aborted",
        },
        uploadedChunks: [],
      });
      setProgress(0);
      setStatus("aborted");
      setNotice({
        description: t("admin.mediaResumable.messages.abortCompletedDescription", {
          sessionId: abortResult.sessionId,
        }),
        intent: "info",
        title: t("admin.mediaResumable.messages.abortCompletedTitle"),
      });
    } catch (error) {
      setStatus("error");
      setNotice({
        description: apiErrorDescription(error, t),
        intent: "danger",
        title: t("admin.mediaResumable.messages.abortFailedTitle"),
      });
    }
  };

  return (
    <section className="aoi-admin-dashboard" aria-labelledby="admin-media-resumable-title">
      <div className="aoi-admin-page-header">
        <div>
          <Badge>{t("admin.mediaResumable.badge")}</Badge>
          <h1 id="admin-media-resumable-title">{t("admin.mediaResumable.title")}</h1>
          <p>{t("admin.mediaResumable.description")}</p>
        </div>
        <div className="aoi-admin-action-row">
          <Button appearance="secondary" asChild>
            <Link to="/admin/media">
              <ArrowLeft aria-hidden="true" size={17} />
              <span>{t("admin.mediaResumable.actions.mediaLibrary")}</span>
            </Link>
          </Button>
          <Button
            appearance="secondary"
            icon={<RefreshCw size={17} />}
            loading={categoryQuery.isFetching || uploadStatusQuery.isFetching}
            onClick={refreshStatus}
          >
            {t("admin.mediaResumable.actions.refreshStatus")}
          </Button>
        </div>
      </div>

      {categoryQuery.error || uploadStatusQuery.error ? (
        <StateBlock
          intent="danger"
          title={t("admin.mediaResumable.messages.statusLoadFailedTitle")}
          description={apiErrorDescription(categoryQuery.error ?? uploadStatusQuery.error, t)}
        />
      ) : null}

      {notice ? (
        <StateBlock description={notice.description} intent={notice.intent} title={notice.title} />
      ) : null}

      {uploadUnavailable ? (
        <StateBlock
          intent="danger"
          title={t("admin.mediaResumable.messages.storageUnavailableTitle")}
          description={t("admin.mediaResumable.messages.storageUnavailableDescription")}
        />
      ) : null}

      <div className="aoi-admin-stat-grid" aria-label={t("admin.mediaResumable.summaryLabel")}>
        <MediaUploadStatCard
          icon={<UploadCloud size={19} />}
          label={t("admin.mediaResumable.metrics.status")}
          value={uploadStatusLabel(status, t)}
        />
        <MediaUploadStatCard
          icon={<FileArchive size={19} />}
          label={t("admin.mediaResumable.metrics.uploadLimit")}
          value={formatUploadLimit(uploadMaxMb, i18n.language, t)}
        />
        <MediaUploadStatCard
          icon={<HardDrive size={19} />}
          label={t("admin.mediaResumable.metrics.storage")}
          value={storageStatus ? storageStatusLabel(storageStatus, t) : t("common.labels.none")}
        />
        <MediaUploadStatCard
          icon={<FolderTree size={19} />}
          label={t("admin.mediaResumable.metrics.chunkSize")}
          value={formatBytes(chunkSizeBytes, i18n.language, t)}
        />
      </div>

      <section className="aoi-admin-panel aoi-media-resumable-panel">
        <header>
          <h2>{t("admin.mediaResumable.upload.title")}</h2>
          <p>{t("admin.mediaResumable.upload.description")}</p>
        </header>

        <div className="aoi-media-resumable-workbench">
          <div className="aoi-media-upload-dropzone" data-state={status}>
            <input
              ref={fileInputRef}
              aria-label={t("admin.mediaResumable.a11y.fileInput")}
              className="aoi-sr-only"
              type="file"
              onChange={(event) => {
                const file = event.currentTarget.files?.[0];
                if (file) {
                  void prepareFile(file);
                }
              }}
            />
            <span className="aoi-media-upload-dropzone__icon" aria-hidden="true">
              {status === "completed" ? <CheckCircle2 size={28} /> : <UploadCloud size={28} />}
            </span>
            <div>
              <strong>
                {selectedFile?.name ?? t("admin.mediaResumable.upload.emptyFileTitle")}
              </strong>
              <p>
                {selectedFile
                  ? t("admin.mediaResumable.upload.fileDescription", {
                      chunks: chunkTotal,
                      size: formatBytes(selectedFile.size, i18n.language, t),
                    })
                  : t("admin.mediaResumable.upload.emptyFileDescription")}
              </p>
            </div>
            <div className="aoi-media-upload-actions">
              <Button
                appearance="secondary"
                disabled={busy}
                icon={<UploadCloud size={17} />}
                onClick={openFilePicker}
              >
                {t("admin.mediaResumable.actions.chooseFile")}
              </Button>
              <Button
                disabled={!canUpload}
                icon={<UploadCloud size={17} />}
                loading={status === "uploading"}
                onClick={() => {
                  void uploadFile();
                }}
              >
                {t("admin.mediaResumable.actions.upload")}
              </Button>
              <Button
                appearance="secondary"
                disabled={!canAbort}
                icon={<XCircle size={17} />}
                onClick={() => {
                  void abortUpload();
                }}
              >
                {t("admin.mediaResumable.actions.abort")}
              </Button>
              <Button
                appearance="ghost"
                disabled={busy}
                icon={<RotateCcw size={17} />}
                onClick={resetUpload}
              >
                {t("admin.mediaResumable.actions.reset")}
              </Button>
            </div>
          </div>

          <div className="aoi-media-upload-side">
            <SelectField
              disabled={Boolean(selectedFile) || busy}
              help={t("admin.mediaResumable.fields.categoryHelp")}
              label={t("admin.mediaResumable.fields.category")}
              options={categoryOptions}
              value={categoryId}
              onChange={(event) => setCategoryId(event.currentTarget.value)}
            />

            <ProgressMeter
              label={t("admin.mediaResumable.progress.label")}
              progress={boundedProgress}
              progressLabel={formatProgress(boundedProgress, i18n.language)}
            />

            <dl className="aoi-admin-key-values" data-columns="2">
              <KeyValue
                label={t("admin.mediaResumable.meta.session")}
                value={checkResult?.session.id ?? t("common.labels.none")}
              />
              <KeyValue
                label={t("admin.mediaResumable.meta.sessionStatus")}
                value={
                  checkResult?.session.status
                    ? backendSessionStatusLabel(checkResult.session.status, t)
                    : t("common.labels.none")
                }
              />
              <KeyValue
                label={t("admin.mediaResumable.meta.uploadedChunks")}
                value={checkResult?.uploadedChunks.length ?? 0}
              />
              <KeyValue
                label={t("admin.mediaResumable.meta.missingChunks")}
                value={checkResult?.missingChunks.length ?? 0}
              />
              <KeyValue
                label={t("admin.mediaResumable.meta.expiresAt")}
                value={
                  checkResult?.session.expiresAt
                    ? formatDate(checkResult.session.expiresAt, i18n.language)
                    : t("common.labels.none")
                }
              />
              <KeyValue
                label={t("admin.mediaResumable.meta.fileHash")}
                value={fileHash || t("common.labels.none")}
              />
            </dl>
          </div>
        </div>
      </section>

      {lastAsset ? (
        <section className="aoi-admin-panel">
          <header className="aoi-admin-panel-header-row">
            <div>
              <h2>{t("admin.mediaResumable.result.title")}</h2>
              <p>
                {t("admin.mediaResumable.result.description", {
                  name: lastAsset.displayName || lastAsset.originalName,
                })}
              </p>
            </div>
            <Button appearance="secondary" asChild>
              <Link to="/admin/media">
                <ArrowLeft aria-hidden="true" size={17} />
                <span>{t("admin.mediaResumable.actions.viewMediaLibrary")}</span>
              </Link>
            </Button>
          </header>
          <dl className="aoi-admin-key-values" data-columns="3">
            <KeyValue
              label={t("admin.mediaResumable.result.asset")}
              value={lastAsset.displayName || lastAsset.originalName}
            />
            <KeyValue
              label={t("admin.mediaResumable.result.mimeType")}
              value={lastAsset.mimeType || t("common.labels.none")}
            />
            <KeyValue
              label={t("admin.mediaResumable.result.size")}
              value={formatBytes(lastAsset.sizeBytes, i18n.language, t)}
            />
          </dl>
        </section>
      ) : null}
    </section>
  );
}

function MediaUploadStatCard({
  icon,
  label,
  value,
}: {
  icon: ReactNode;
  label: string;
  value: string;
}) {
  return (
    <article className="aoi-admin-stat-card">
      <span aria-hidden="true">{icon}</span>
      <div>
        <p>{label}</p>
        <strong>{value}</strong>
      </div>
    </article>
  );
}

function ProgressMeter({
  label,
  progress,
  progressLabel,
}: {
  label: string;
  progress: number;
  progressLabel: string;
}) {
  return (
    <div className="aoi-progress-meter">
      <div>
        <span>{label}</span>
        <strong>{progressLabel}</strong>
      </div>
      <progress aria-label={label} max={100} value={progress} />
    </div>
  );
}

function KeyValue({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div>
      <dt>{label}</dt>
      <dd>{value}</dd>
    </div>
  );
}

function flattenCategories(items: SystemMediaCategory[], depth = 0): FlatCategory[] {
  return items.flatMap((category) => [
    { category, depth },
    ...flattenCategories(category.children ?? [], depth + 1),
  ]);
}

function normalizeProgress(value: number) {
  if (!Number.isFinite(value)) {
    return 0;
  }
  return Math.min(100, Math.max(0, value));
}

function hasWebCrypto() {
  return typeof globalThis.crypto?.subtle?.digest === "function";
}

async function sha256Blob(blob: Blob) {
  const buffer = await blob.arrayBuffer();
  const digest = await globalThis.crypto.subtle.digest("SHA-256", buffer);
  return Array.from(new Uint8Array(digest), (byte) => byte.toString(16).padStart(2, "0")).join("");
}

function uploadStatusLabel(status: UploadStatus, t: ReturnType<typeof useTranslation>["t"]) {
  return t(`admin.mediaResumable.status.${status}`);
}

function backendSessionStatusLabel(status: string, t: ReturnType<typeof useTranslation>["t"]) {
  return t(`admin.mediaResumable.sessionStatus.${status}`, { defaultValue: status });
}

function storageStatusLabel(status: string, t: ReturnType<typeof useTranslation>["t"]) {
  return t(`admin.mediaResumable.storageStatus.${status}`, { defaultValue: status });
}

function apiErrorDescription(error: unknown, t: ReturnType<typeof useTranslation>["t"]) {
  if (error instanceof ApiError) {
    return error.message || t("errors.api.requestFailed");
  }
  if (error instanceof Error) {
    return error.message;
  }
  return t("errors.api.requestFailed");
}

function formatDate(value: string, locale: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return new Intl.DateTimeFormat(locale, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}

function formatBytes(bytes: number, locale: string, t: ReturnType<typeof useTranslation>["t"]) {
  if (!Number.isFinite(bytes) || bytes <= 0) {
    return t("common.labels.none");
  }
  const units = ["B", "KB", "MB", "GB", "TB"];
  let value = bytes;
  let unitIndex = 0;
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex += 1;
  }
  return `${new Intl.NumberFormat(locale, {
    maximumFractionDigits: value >= 10 ? 0 : 1,
  }).format(value)} ${units[unitIndex]}`;
}

function formatProgress(progress: number, locale: string) {
  return `${new Intl.NumberFormat(locale, { maximumFractionDigits: 0 }).format(progress)}%`;
}

function formatUploadLimit(
  limitMb: number,
  locale: string,
  t: ReturnType<typeof useTranslation>["t"],
) {
  if (!Number.isFinite(limitMb) || limitMb <= 0) {
    return t("common.labels.none");
  }
  return t("admin.mediaResumable.values.uploadLimit", {
    value: new Intl.NumberFormat(locale, { maximumFractionDigits: 0 }).format(limitMb),
  });
}
