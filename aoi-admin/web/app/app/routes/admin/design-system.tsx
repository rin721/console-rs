import {
  Blocks,
  Code2,
  Download,
  FileJson,
  FileUp,
  LayoutTemplate,
  Palette,
  RotateCcw,
  Save,
  Upload,
} from "lucide-react";
import { useMemo, useState, type CSSProperties, type ChangeEvent } from "react";
import { useTranslation } from "react-i18next";

import { FormField } from "~/components/aoi/patterns/FormField";
import { SelectField, type SelectOption } from "~/components/aoi/patterns/SelectField";
import { StateBlock } from "~/components/aoi/patterns/StateBlock";
import { Badge } from "~/components/aoi/primitives/Badge";
import { Button } from "~/components/aoi/primitives/Button";
import {
  calculateContrastRatio,
  contrastStatus,
  defaultThemeDraft,
  normalizeThemeDraft,
  parseThemeImport,
  serializeThemeDraft,
  type MotionIntensity,
  type ShadowLevel,
  type ThemeDraft,
  type ThemeMode,
} from "~/features/theme-settings/theme-settings";
import type { TranslationKey } from "~/i18n/keys";
import { activeThemeId, activeThemePackage } from "~/theme/generated/theme-metadata";

type Notice = {
  description: string;
  intent?: "danger" | "info";
  title: string;
};

type ColorTokenKey = "accentColor" | "primaryColor" | "surfaceColor" | "textColor";
type NumericTokenKey = "radiusScale" | "spacingScale" | "typographyScale";
type SelectTokenKey = "mode" | "motionIntensity" | "shadowLevel";
type ThemeTokenKey = ColorTokenKey | NumericTokenKey | SelectTokenKey;
type Translate = (key: TranslationKey, options?: Record<string, unknown>) => string;

const storageKey = "aoi-admin-theme-draft";
const themeTokenKeys: ThemeTokenKey[] = [
  "mode",
  "primaryColor",
  "accentColor",
  "surfaceColor",
  "textColor",
  "typographyScale",
  "spacingScale",
  "radiusScale",
  "shadowLevel",
  "motionIntensity",
];

export default function AdminDesignSystemRoute() {
  const { i18n, t } = useTranslation();
  const tx = (key: TranslationKey, options?: Record<string, unknown>) =>
    options ? String(t(key, options)) : String(t(key));
  const [initialDraft] = useState(readStoredThemeDraft);
  const [draft, setDraft] = useState<ThemeDraft>(initialDraft);
  const [savedDraft, setSavedDraft] = useState<ThemeDraft>(initialDraft);
  const [notice, setNotice] = useState<Notice | null>(null);
  const [importPayload, setImportPayload] = useState("");
  const recipeEntries = useMemo(() => Object.entries(activeThemePackage.recipes), []);
  const templateEntries = useMemo(() => Object.entries(activeThemePackage.templates), []);
  const assetCount =
    activeThemePackage.assets.fonts.length +
    activeThemePackage.assets.icons.length +
    activeThemePackage.assets.images.length;

  const contrastChecks = useMemo(
    () => [
      {
        background: draft.surfaceColor,
        foreground: draft.textColor,
        key: "textOnSurface",
      },
      {
        background: draft.surfaceColor,
        foreground: draft.primaryColor,
        key: "primaryOnSurface",
      },
      {
        background: draft.surfaceColor,
        foreground: draft.accentColor,
        key: "accentOnSurface",
      },
    ],
    [draft],
  );

  const changedTokens = useMemo(
    () => themeTokenKeys.filter((key) => !sameTokenValue(draft[key], savedDraft[key])),
    [draft, savedDraft],
  );
  const hasLocalChanges = changedTokens.length > 0;
  const previewStyle = {
    "--theme-preview-accent": draft.accentColor,
    "--theme-preview-motion":
      draft.motionIntensity === "none"
        ? "0ms"
        : draft.motionIntensity === "reduced"
          ? "90ms"
          : "180ms",
    "--theme-preview-primary": draft.primaryColor,
    "--theme-preview-radius": `${draft.radiusScale}px`,
    "--theme-preview-shadow":
      draft.shadowLevel === "none"
        ? "none"
        : draft.shadowLevel === "soft"
          ? "var(--aoi-shadow-card)"
          : "var(--aoi-shadow-overlay)",
    "--theme-preview-spacing": `${draft.spacingScale}rem`,
    "--theme-preview-surface": draft.surfaceColor,
    "--theme-preview-text": draft.textColor,
    "--theme-preview-type": `${draft.typographyScale}rem`,
  } as CSSProperties;

  function updateColorToken(key: ColorTokenKey, value: string) {
    updateDraftValue(key, value);
  }

  function updateNumericToken(key: NumericTokenKey, value: string) {
    updateDraftValue(key, Number(value));
  }

  function updateSelectToken(key: "mode", value: string) {
    updateDraftValue(key, value as ThemeMode);
  }

  function updateMotionToken(value: string) {
    updateDraftValue("motionIntensity", value as MotionIntensity);
  }

  function updateShadowToken(value: string) {
    updateDraftValue("shadowLevel", value as ShadowLevel);
  }

  function updateDraftValue<Key extends keyof ThemeDraft>(key: Key, value: ThemeDraft[Key]) {
    setDraft((current) => normalizeThemeDraft({ ...current, [key]: value }));
  }

  function saveLocalDraft() {
    const normalizedDraft = normalizeThemeDraft(draft);
    window.localStorage.setItem(storageKey, serializeThemeDraft(normalizedDraft));
    setDraft(normalizedDraft);
    setSavedDraft(normalizedDraft);
    setNotice({
      description: tx("admin.designSystem.messages.savedDescription"),
      title: tx("admin.designSystem.messages.savedTitle"),
    });
  }

  function restoreDefaults() {
    setDraft(defaultThemeDraft);
    setNotice({
      description: tx("admin.designSystem.messages.defaultsDescription"),
      title: tx("admin.designSystem.messages.defaultsTitle"),
    });
  }

  function exportDraft() {
    const blob = new Blob([serializeThemeDraft(draft)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = "aoi-theme-draft.json";
    link.click();
    URL.revokeObjectURL(url);
    setNotice({
      description: tx("admin.designSystem.messages.exportedDescription"),
      title: tx("admin.designSystem.messages.exportedTitle"),
    });
  }

  function importDraft(rawValue = importPayload) {
    const result = parseThemeImport(rawValue);
    if (!result.ok) {
      setNotice({
        description:
          result.message === "invalid-json"
            ? tx("admin.designSystem.import.errors.invalidJson")
            : tx("admin.designSystem.import.errors.invalidSchema"),
        intent: "danger",
        title: tx("admin.designSystem.import.errors.title"),
      });
      return;
    }
    setDraft(result.draft);
    setNotice({
      description: tx("admin.designSystem.messages.importedDescription"),
      title: tx("admin.designSystem.messages.importedTitle"),
    });
  }

  function importDraftFile(event: ChangeEvent<HTMLInputElement>) {
    const input = event.currentTarget;
    const file = input.files?.[0];
    input.value = "";
    if (!file) {
      return;
    }
    void file
      .text()
      .then((content) => {
        setImportPayload(content);
        importDraft(content);
      })
      .catch(() => {
        setNotice({
          description: tx("admin.designSystem.import.errors.fileRead"),
          intent: "danger",
          title: tx("admin.designSystem.import.errors.title"),
        });
      });
  }

  const modeOptions: SelectOption[] = [
    { label: tx("admin.designSystem.options.mode.light"), value: "light" },
    { label: tx("admin.designSystem.options.mode.dark"), value: "dark" },
  ];
  const shadowOptions: SelectOption[] = [
    { label: tx("admin.designSystem.options.shadow.none"), value: "none" },
    { label: tx("admin.designSystem.options.shadow.soft"), value: "soft" },
    { label: tx("admin.designSystem.options.shadow.standard"), value: "standard" },
  ];
  const motionOptions: SelectOption[] = [
    { label: tx("admin.designSystem.options.motion.none"), value: "none" },
    { label: tx("admin.designSystem.options.motion.reduced"), value: "reduced" },
    { label: tx("admin.designSystem.options.motion.standard"), value: "standard" },
  ];

  return (
    <section className="aoi-admin-dashboard" aria-labelledby="admin-design-system-title">
      <div className="aoi-admin-page-header">
        <div>
          <Badge>{t("admin.designSystem.badge")}</Badge>
          <h1 id="admin-design-system-title">{t("admin.designSystem.title")}</h1>
          <p>{t("admin.designSystem.description")}</p>
        </div>
        <div className="aoi-theme-page-actions">
          <Button appearance="secondary" icon={<RotateCcw size={17} />} onClick={restoreDefaults}>
            {t("admin.designSystem.actions.restoreDefaults")}
          </Button>
          <Button disabled={!hasLocalChanges} icon={<Save size={17} />} onClick={saveLocalDraft}>
            {t("admin.designSystem.actions.saveDraft")}
          </Button>
        </div>
      </div>

      {notice ? (
        <StateBlock description={notice.description} intent={notice.intent} title={notice.title} />
      ) : null}
      <StateBlock
        title={t("admin.designSystem.contract.title")}
        description={t("admin.designSystem.contract.description")}
      />

      <div className="aoi-admin-stat-grid" aria-label={t("admin.designSystem.summaryLabel")}>
        <ThemeStatCard label={t("admin.designSystem.metrics.activeTheme")} value={activeThemeId} />
        <ThemeStatCard
          label={t("admin.designSystem.metrics.coverage")}
          value={new Intl.NumberFormat(i18n.language).format(
            activeThemePackage.tokens.coverage.length,
          )}
        />
        <ThemeStatCard
          label={t("admin.designSystem.metrics.recipes")}
          value={new Intl.NumberFormat(i18n.language).format(recipeEntries.length)}
        />
        <ThemeStatCard
          label={t("admin.designSystem.metrics.templates")}
          value={new Intl.NumberFormat(i18n.language).format(templateEntries.length)}
        />
        <ThemeStatCard
          label={t("admin.designSystem.metrics.assets")}
          value={new Intl.NumberFormat(i18n.language).format(assetCount)}
        />
      </div>

      <div className="aoi-theme-layout">
        <section className="aoi-admin-panel aoi-theme-package-panel">
          <header>
            <h2>{t("admin.designSystem.package.title")}</h2>
            <p>{t("admin.designSystem.package.description")}</p>
          </header>
          <div className="aoi-theme-package-card">
            <span aria-hidden="true">
              <Blocks size={20} />
            </span>
            <div>
              <strong>{activeThemePackage.manifest.name}</strong>
              <p>{activeThemePackage.manifest.description}</p>
              <dl>
                <div>
                  <dt>{t("admin.designSystem.package.fields.id")}</dt>
                  <dd>{activeThemePackage.manifest.id}</dd>
                </div>
                <div>
                  <dt>{t("admin.designSystem.package.fields.version")}</dt>
                  <dd>{activeThemePackage.manifest.version}</dd>
                </div>
                <div>
                  <dt>{t("admin.designSystem.package.fields.source")}</dt>
                  <dd>{activeThemePackage.manifest.source}</dd>
                </div>
              </dl>
            </div>
          </div>
        </section>

        <section className="aoi-admin-panel aoi-theme-coverage-panel">
          <header>
            <h2>{t("admin.designSystem.coverage.title")}</h2>
            <p>{t("admin.designSystem.coverage.description")}</p>
          </header>
          <div className="aoi-theme-chip-list">
            {activeThemePackage.tokens.coverage.map((item) => (
              <span key={item}>{item}</span>
            ))}
          </div>
        </section>

        <section className="aoi-admin-panel aoi-theme-recipe-panel">
          <header>
            <h2>{t("admin.designSystem.recipes.title")}</h2>
            <p>{t("admin.designSystem.recipes.description")}</p>
          </header>
          <div className="aoi-theme-card-grid">
            {recipeEntries.map(([key, recipe]) => (
              <article className="aoi-theme-source-card" key={key}>
                <Code2 aria-hidden="true" size={18} />
                <strong>{recipe.label}</strong>
                <p>{recipe.description}</p>
                <code>{recipe.tokens.join(", ")}</code>
              </article>
            ))}
          </div>
        </section>

        <section className="aoi-admin-panel aoi-theme-template-panel">
          <header>
            <h2>{t("admin.designSystem.templates.title")}</h2>
            <p>{t("admin.designSystem.templates.description")}</p>
          </header>
          <div className="aoi-theme-card-grid">
            {templateEntries.map(([key, template]) => (
              <article className="aoi-theme-source-card" key={key}>
                <LayoutTemplate aria-hidden="true" size={18} />
                <strong>{template.label}</strong>
                <p>{template.description}</p>
                <code>{key}</code>
              </article>
            ))}
          </div>
        </section>

        <section className="aoi-admin-panel aoi-theme-controls">
          <header>
            <h2>{t("admin.designSystem.controls.title")}</h2>
            <p>{t("admin.designSystem.controls.description")}</p>
          </header>

          <div className="aoi-theme-control-grid">
            <SelectField
              help={t("admin.designSystem.fields.mode.help")}
              label={t("admin.designSystem.fields.mode.label")}
              options={modeOptions}
              value={draft.mode}
              onChange={(event) => updateSelectToken("mode", event.currentTarget.value)}
            />
            <SelectField
              help={t("admin.designSystem.fields.shadowLevel.help")}
              label={t("admin.designSystem.fields.shadowLevel.label")}
              options={shadowOptions}
              value={draft.shadowLevel}
              onChange={(event) => updateShadowToken(event.currentTarget.value)}
            />
            <SelectField
              help={t("admin.designSystem.fields.motionIntensity.help")}
              label={t("admin.designSystem.fields.motionIntensity.label")}
              options={motionOptions}
              value={draft.motionIntensity}
              onChange={(event) => updateMotionToken(event.currentTarget.value)}
            />
          </div>

          <div className="aoi-theme-color-grid">
            {(["primaryColor", "accentColor", "surfaceColor", "textColor"] as ColorTokenKey[]).map(
              (key) => (
                <div className="aoi-theme-color-field" key={key}>
                  <FormField
                    help={t(`admin.designSystem.fields.${key}.help`)}
                    label={t(`admin.designSystem.fields.${key}.label`)}
                    type="color"
                    value={draft[key]}
                    onChange={(event) => updateColorToken(key, event.currentTarget.value)}
                  />
                  <code>{draft[key]}</code>
                </div>
              ),
            )}
          </div>

          <div className="aoi-theme-control-grid">
            <ThemeRangeField
              help={t("admin.designSystem.fields.typographyScale.help")}
              label={t("admin.designSystem.fields.typographyScale.label")}
              max={1.12}
              min={0.92}
              step={0.01}
              value={draft.typographyScale}
              valueLabel={formatPercent(draft.typographyScale, i18n.language)}
              onChange={(value) => updateNumericToken("typographyScale", value)}
            />
            <ThemeRangeField
              help={t("admin.designSystem.fields.spacingScale.help")}
              label={t("admin.designSystem.fields.spacingScale.label")}
              max={1.25}
              min={0.85}
              step={0.01}
              value={draft.spacingScale}
              valueLabel={formatPercent(draft.spacingScale, i18n.language)}
              onChange={(value) => updateNumericToken("spacingScale", value)}
            />
            <ThemeRangeField
              help={t("admin.designSystem.fields.radiusScale.help")}
              label={t("admin.designSystem.fields.radiusScale.label")}
              max={24}
              min={0}
              step={1}
              value={draft.radiusScale}
              valueLabel={t("admin.designSystem.values.radius", { value: draft.radiusScale })}
              onChange={(value) => updateNumericToken("radiusScale", value)}
            />
          </div>
        </section>

        <section className="aoi-admin-panel aoi-theme-preview-panel">
          <header>
            <h2>{t("admin.designSystem.preview.title")}</h2>
            <p>{t("admin.designSystem.preview.description")}</p>
          </header>
          <div className="aoi-theme-preview" data-mode={draft.mode} style={previewStyle}>
            <div className="aoi-theme-preview__surface">
              <span>{t("admin.designSystem.preview.badge")}</span>
              <h3>{t("admin.designSystem.preview.heading")}</h3>
              <p>{t("admin.designSystem.preview.copy")}</p>
              <div className="aoi-theme-preview__actions">
                <button type="button">{t("admin.designSystem.preview.primaryAction")}</button>
                <button type="button">{t("admin.designSystem.preview.secondaryAction")}</button>
              </div>
            </div>
            <div className="aoi-theme-preview__module">
              <strong>{t("admin.designSystem.preview.moduleTitle")}</strong>
              <span>{t("admin.designSystem.preview.moduleMeta")}</span>
            </div>
          </div>
        </section>

        <section className="aoi-admin-panel aoi-theme-contrast-panel">
          <header>
            <h2>{t("admin.designSystem.contrast.title")}</h2>
            <p>{t("admin.designSystem.contrast.description")}</p>
          </header>
          <div className="aoi-theme-contrast-grid">
            {contrastChecks.map((check) => {
              const ratio = calculateContrastRatio(check.foreground, check.background);
              const status = contrastStatus(ratio);
              return (
                <article className="aoi-theme-contrast-card" data-status={status} key={check.key}>
                  <div
                    aria-hidden="true"
                    style={{
                      backgroundColor: check.background,
                      color: check.foreground,
                    }}
                  >
                    Aa
                  </div>
                  <span>{t(`admin.designSystem.contrast.checks.${check.key}`)}</span>
                  <strong>{formatRatio(ratio, i18n.language)}</strong>
                  <em>{t(`admin.designSystem.contrast.status.${status}`)}</em>
                </article>
              );
            })}
          </div>
        </section>

        <section className="aoi-admin-panel aoi-theme-import-panel">
          <header>
            <h2>{t("admin.designSystem.import.title")}</h2>
            <p>{t("admin.designSystem.import.description")}</p>
          </header>
          <div className="aoi-theme-import-actions">
            <Button appearance="secondary" icon={<Download size={17} />} onClick={exportDraft}>
              {t("admin.designSystem.actions.exportJson")}
            </Button>
            <label className="aoi-theme-file-button">
              <FileUp aria-hidden="true" size={17} />
              <span>{t("admin.designSystem.actions.importFile")}</span>
              <input
                accept="application/json,.json"
                aria-label={t("admin.designSystem.import.fileLabel")}
                type="file"
                onChange={importDraftFile}
              />
            </label>
          </div>
          <div className="aoi-form-field">
            <label htmlFor="theme-import-json">{t("admin.designSystem.import.textarea")}</label>
            <textarea
              id="theme-import-json"
              placeholder={t("admin.designSystem.import.placeholder")}
              rows={8}
              value={importPayload}
              onChange={(event) => setImportPayload(event.currentTarget.value)}
            />
            <span className="aoi-form-field__help">{t("admin.designSystem.import.help")}</span>
          </div>
          <Button appearance="secondary" icon={<Upload size={17} />} onClick={() => importDraft()}>
            {t("admin.designSystem.actions.applyImport")}
          </Button>
        </section>

        <section className="aoi-admin-panel aoi-theme-release-panel">
          <header>
            <h2>{t("admin.designSystem.release.title")}</h2>
            <p>{t("admin.designSystem.release.description")}</p>
          </header>
          <div className="aoi-theme-diff" aria-label={t("admin.designSystem.diff.label")}>
            {changedTokens.length === 0 ? (
              <p>{t("admin.designSystem.release.sourceOnly")}</p>
            ) : (
              <dl>
                {changedTokens.map((key) => (
                  <div key={key}>
                    <dt>{t(`admin.designSystem.tokenLabels.${key}`)}</dt>
                    <dd>
                      {t("admin.designSystem.diff.changed", {
                        from: formatTokenValue(key, savedDraft[key], tx, i18n.language),
                        to: formatTokenValue(key, draft[key], tx, i18n.language),
                      })}
                    </dd>
                  </div>
                ))}
              </dl>
            )}
          </div>
          <div className="aoi-theme-release-actions">
            <Button disabled icon={<FileJson size={17} />}>
              {t("admin.designSystem.actions.sourcePackage")}
            </Button>
            <Button appearance="secondary" disabled icon={<Palette size={17} />}>
              {t("admin.designSystem.actions.backendDisabled")}
            </Button>
          </div>
        </section>
      </div>
    </section>
  );
}

type ThemeRangeFieldProps = {
  help: string;
  label: string;
  max: number;
  min: number;
  step: number;
  value: number;
  valueLabel: string;
  onChange: (value: string) => void;
};

function ThemeRangeField({
  help,
  label,
  max,
  min,
  onChange,
  step,
  value,
  valueLabel,
}: ThemeRangeFieldProps) {
  return (
    <div className="aoi-theme-range-field">
      <FormField
        help={help}
        label={label}
        max={max}
        min={min}
        step={step}
        type="range"
        value={value}
        onChange={(event) => onChange(event.currentTarget.value)}
      />
      <output>{valueLabel}</output>
    </div>
  );
}

type ThemeStatCardProps = {
  label: string;
  value: string;
};

function ThemeStatCard({ label, value }: ThemeStatCardProps) {
  return (
    <article className="aoi-admin-stat-card">
      <span aria-hidden="true">
        <Palette size={18} />
      </span>
      <div>
        <p>{label}</p>
        <strong>{value}</strong>
      </div>
    </article>
  );
}

function sameTokenValue(left: ThemeDraft[ThemeTokenKey], right: ThemeDraft[ThemeTokenKey]) {
  return JSON.stringify(left) === JSON.stringify(right);
}

function contrastSummary(
  checks: Array<{ background: string; foreground: string; key: string }>,
  t: Translate,
) {
  return checks.every(
    (check) =>
      contrastStatus(calculateContrastRatio(check.foreground, check.background)) === "pass",
  )
    ? t("admin.designSystem.metrics.contrastPass")
    : t("admin.designSystem.metrics.contrastFail");
}

function formatTokenValue(
  key: ThemeTokenKey,
  value: ThemeDraft[ThemeTokenKey],
  t: Translate,
  locale: string,
) {
  if (key === "mode") {
    return t(`admin.designSystem.options.mode.${value}`);
  }
  if (key === "motionIntensity") {
    return t(`admin.designSystem.options.motion.${value}`);
  }
  if (key === "shadowLevel") {
    return t(`admin.designSystem.options.shadow.${value}`);
  }
  if (key === "typographyScale" || key === "spacingScale") {
    return formatPercent(Number(value), locale);
  }
  if (key === "radiusScale") {
    return t("admin.designSystem.values.radius", { value });
  }
  return String(value);
}

function formatPercent(value: number, locale: string) {
  return new Intl.NumberFormat(locale, {
    maximumFractionDigits: 0,
    style: "percent",
  }).format(value);
}

function formatRatio(value: number, locale: string) {
  return new Intl.NumberFormat(locale, {
    maximumFractionDigits: 2,
    minimumFractionDigits: 2,
  }).format(value);
}

function readStoredThemeDraft() {
  if (typeof window === "undefined") {
    return defaultThemeDraft;
  }
  const stored = window.localStorage.getItem(storageKey);
  if (!stored) {
    return defaultThemeDraft;
  }
  const result = parseThemeImport(stored);
  return result.ok ? result.draft : defaultThemeDraft;
}
