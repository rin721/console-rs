import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { ColumnDef } from "@tanstack/react-table";
import {
  EyeOff,
  Layers,
  ListTree,
  RefreshCw,
  Save,
  Settings,
  ShieldAlert,
  SlidersHorizontal,
  X,
} from "lucide-react";
import { useCallback, useMemo, useState, type FormEvent, type ReactNode } from "react";
import { useTranslation } from "react-i18next";

import { DataTable } from "~/components/aoi/patterns/DataTable";
import { Drawer } from "~/components/aoi/patterns/Drawer";
import { FormField } from "~/components/aoi/patterns/FormField";
import { PanelSkeleton, StatGridSkeleton } from "~/components/aoi/patterns/LoadingSkeletons";
import { Popover } from "~/components/aoi/patterns/Popover";
import { SelectField, type SelectOption } from "~/components/aoi/patterns/SelectField";
import { StateBlock } from "~/components/aoi/patterns/StateBlock";
import { Badge } from "~/components/aoi/primitives/Badge";
import { Button } from "~/components/aoi/primitives/Button";
import { ApiError } from "~/lib/api/client";
import type { TranslationKey } from "~/i18n/keys";
import { queryKeys } from "~/lib/api/query-keys";
import { systemApi } from "~/lib/api/system";
import type {
  SystemConfigGroup,
  SystemConfigItem,
  SystemConfigSection,
  SystemConfigVisibilityCondition,
} from "~/lib/api/types";

type Translate = (key: TranslationKey, options?: Record<string, unknown>) => string;
type ConfigDraftValue = string | boolean;

type ConfigNotice = {
  description: string;
  intent?: "danger" | "info";
  title: string;
};

const valueTypeLabels: Record<string, TranslationKey> = {
  array: "admin.system.valueTypes.array",
  boolean: "admin.system.valueTypes.boolean",
  number: "admin.system.valueTypes.number",
  object: "admin.system.valueTypes.object",
  string: "admin.system.valueTypes.string",
  unknown: "admin.system.valueTypes.unknown",
};

const riskLabels: Record<string, TranslationKey> = {
  high: "admin.system.risk.high",
  low: "admin.system.risk.low",
  medium: "admin.system.risk.medium",
};

export default function AdminSystemRoute() {
  const { i18n, t } = useTranslation();
  const queryClient = useQueryClient();
  const tx = useCallback<Translate>(
    (key, options) => (options ? String(t(key, options)) : String(t(key))),
    [t],
  );
  const [selectedCode, setSelectedCode] = useState("");
  const [editingGroup, setEditingGroup] = useState<SystemConfigGroup | null>(null);
  const [draftValues, setDraftValues] = useState<Record<string, ConfigDraftValue>>({});
  const [configNotice, setConfigNotice] = useState<ConfigNotice | null>(null);

  const configQuery = useQuery({
    queryFn: ({ signal }) => systemApi.getConfig({ signal }),
    queryKey: queryKeys.system.config(i18n.language),
  });

  const sections = useMemo(
    () => sortSections(configQuery.data?.sections ?? []),
    [configQuery.data],
  );
  const selectedSection = sections.find((section) => section.code === selectedCode) ?? sections[0];
  const valueMap = useMemo(() => buildValueMap(sections), [sections]);
  const selectedGroups = useMemo(
    () =>
      groupsForSection(selectedSection).map((group) => ({
        ...group,
        items: group.items.filter((item) => isVisible(item.visibleWhen, valueMap)),
      })),
    [selectedSection, valueMap],
  );
  const summary = useMemo(() => summarizeConfig(sections, valueMap), [sections, valueMap]);
  const editingItems = useMemo(
    () => editingGroup?.items.filter((item) => item.editable) ?? [],
    [editingGroup],
  );
  const changedDraftItems = useMemo(
    () => editingItems.filter((item) => isDraftChanged(item, draftValues)),
    [draftValues, editingItems],
  );
  const canSaveConfig =
    changedDraftItems.length > 0 && editingItems.every((item) => isDraftValid(item, draftValues));

  const updateConfigMutation = useMutation({
    mutationFn: (items: { key: string; value: unknown }[]) =>
      systemApi.updateConfig({ items, persist: true }),
    onError: (error) => {
      const normalized = toError(error);
      setConfigNotice({
        description: errorDescription(normalized, tx),
        intent: "danger",
        title: errorTitle(normalized, tx),
      });
    },
    onMutate: () => {
      setConfigNotice(null);
    },
    onSuccess: (snapshot) => {
      queryClient.setQueryData(queryKeys.system.config(i18n.language), snapshot);
      setConfigNotice({
        description: tx("admin.system.messages.updateSuccessDescription", {
          count: changedDraftItems.length,
        }),
        title: tx("admin.system.messages.updateSuccessTitle"),
      });
      closeConfigEditor();
    },
  });

  const columns = useMemo<ColumnDef<SystemConfigItem>[]>(
    () => [
      {
        accessorKey: "label",
        cell: ({ row }) => (
          <div className="aoi-config-setting">
            <strong>{row.original.label || row.original.key}</strong>
            <span>{row.original.key}</span>
            {row.original.description ? <p>{row.original.description}</p> : null}
          </div>
        ),
        header: t("admin.system.columns.setting"),
      },
      {
        accessorKey: "value",
        cell: ({ row }) => (
          <code className="aoi-config-value">
            {formatConfigValue(row.original, i18n.language, tx)}
          </code>
        ),
        header: t("admin.system.columns.value"),
      },
      {
        accessorKey: "source",
        cell: ({ getValue }) => (
          <span className="aoi-config-source">{formatPlainValue(getValue(), tx)}</span>
        ),
        header: t("admin.system.columns.source"),
      },
      {
        accessorKey: "valueType",
        cell: ({ row }) => (
          <div className="aoi-config-flags">
            <span data-flag={row.original.editable ? "editable" : "readonly"}>
              {row.original.editable
                ? t("admin.system.badges.editable")
                : t("admin.system.badges.readonly")}
            </span>
            {row.original.secret ? (
              <span data-flag="secret">{t("admin.system.badges.secret")}</span>
            ) : null}
            <span>{valueTypeLabel(row.original.valueType, tx)}</span>
          </div>
        ),
        header: t("admin.system.columns.flags"),
      },
    ],
    [i18n.language, t, tx],
  );

  function openConfigEditor(group: SystemConfigGroup) {
    if (!group.items.some((item) => item.editable)) {
      return;
    }
    setConfigNotice(null);
    setEditingGroup(group);
    setDraftValues(seedDraftValues(group.items));
  }

  function closeConfigEditor() {
    setEditingGroup(null);
    setDraftValues({});
  }

  function selectSection(code: string) {
    setSelectedCode(code);
    closeConfigEditor();
  }

  function updateDraftValue(key: string, value: ConfigDraftValue) {
    setDraftValues((current) => ({ ...current, [key]: value }));
  }

  function submitConfigEditor(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!canSaveConfig || updateConfigMutation.isPending) {
      return;
    }
    const payload = changedDraftItems
      .map((item) => ({ key: item.key, value: draftPayload(item, draftValues) }))
      .filter((item) => item.value !== undefined);

    if (payload.length === 0) {
      closeConfigEditor();
      return;
    }

    updateConfigMutation.mutate(payload);
  }

  return (
    <section className="aoi-admin-dashboard" aria-labelledby="admin-system-title">
      <div className="aoi-admin-page-header">
        <div>
          <Badge>{t("admin.system.badge")}</Badge>
          <h1 id="admin-system-title">{t("admin.system.title")}</h1>
          <p>{t("admin.system.description")}</p>
        </div>
        <Button
          appearance="secondary"
          icon={<RefreshCw size={17} />}
          loading={configQuery.isFetching}
          onClick={() => void configQuery.refetch()}
        >
          {t("admin.system.actions.refresh")}
        </Button>
      </div>

      {configQuery.error ? (
        <StateBlock
          intent="danger"
          title={errorTitle(configQuery.error, tx)}
          description={errorDescription(configQuery.error, tx)}
        />
      ) : null}
      {configNotice ? (
        <StateBlock
          description={configNotice.description}
          intent={configNotice.intent}
          title={configNotice.title}
        />
      ) : null}
      <StateBlock
        title={t("admin.system.persistWarningTitle")}
        description={t("admin.system.persistWarningDescription")}
      />

      {configQuery.isLoading ? (
        <StatGridSkeleton />
      ) : (
        <div className="aoi-admin-stat-grid" aria-label={t("admin.system.summaryLabel")}>
          <ConfigStatCard
            icon={<Layers size={19} />}
            label={t("admin.system.metrics.sections")}
            value={formatNumber(summary.sections, i18n.language)}
          />
          <ConfigStatCard
            icon={<ListTree size={19} />}
            label={t("admin.system.metrics.groups")}
            value={formatNumber(summary.visibleGroups, i18n.language)}
          />
          <ConfigStatCard
            icon={<Settings size={19} />}
            label={t("admin.system.metrics.items")}
            value={formatNumber(summary.items, i18n.language)}
          />
          <ConfigStatCard
            icon={<SlidersHorizontal size={19} />}
            label={t("admin.system.metrics.editable")}
            value={formatNumber(summary.editable, i18n.language)}
          />
          <ConfigStatCard
            icon={<ShieldAlert size={19} />}
            label={t("admin.system.metrics.secret")}
            value={formatNumber(summary.secret, i18n.language)}
          />
        </div>
      )}

      <Drawer
        closeLabel={t("admin.system.actions.cancelEdit")}
        description={t("admin.system.editor.description")}
        open={Boolean(editingGroup)}
        title={
          editingGroup
            ? t("admin.system.editor.title", { label: editingGroup.label })
            : t("admin.system.editor.title", { label: "" })
        }
        onOpenChange={(open) => {
          if (!open && !updateConfigMutation.isPending) {
            closeConfigEditor();
          }
        }}
      >
        {editingGroup ? (
          <ConfigEditorPanel
            canSave={canSaveConfig}
            changedItems={changedDraftItems}
            draftValues={draftValues}
            group={editingGroup}
            loading={updateConfigMutation.isPending}
            showHeader={false}
            t={tx}
            onCancel={closeConfigEditor}
            onChange={updateDraftValue}
            onSubmit={submitConfigEditor}
          />
        ) : null}
      </Drawer>

      {configQuery.isLoading ? (
        <section className="aoi-admin-panel">
          <PanelSkeleton rows={5} />
        </section>
      ) : sections.length === 0 ? (
        <StateBlock
          title={t("admin.system.states.emptyTitle")}
          description={t("admin.system.states.emptyDescription")}
        />
      ) : selectedSection ? (
        <section className="aoi-admin-panel aoi-config-workbench">
          <div className="aoi-config-layout">
            <nav className="aoi-config-rail" aria-label={t("admin.system.sectionsLabel")}>
              {sections.map((section) => {
                const groups = groupsForSection(section);
                const active = selectedSection.code === section.code;
                return (
                  <button
                    aria-current={active ? "page" : undefined}
                    className="aoi-config-rail__item"
                    key={section.code}
                    onClick={() => selectSection(section.code)}
                    type="button"
                  >
                    <Settings aria-hidden="true" size={18} />
                    <span>
                      <strong>{section.label || section.code}</strong>
                      <small>{section.description || section.code}</small>
                    </span>
                    <em>{formatNumber(groups.length, i18n.language)}</em>
                  </button>
                );
              })}
            </nav>

            <div className="aoi-config-stage">
              <header className="aoi-config-stage__header">
                <div>
                  <span className="aoi-config-code">{selectedSection.code}</span>
                  <h2>{selectedSection.label || selectedSection.code}</h2>
                  <p>{selectedSection.description}</p>
                </div>
                <span className="aoi-config-count">
                  {t("admin.system.visibleGroupCount", {
                    count: selectedGroups.filter((group) => isVisible(group.visibleWhen, valueMap))
                      .length,
                  })}
                </span>
              </header>

              <div className="aoi-config-group-grid">
                {selectedGroups.map((group) => {
                  const groupVisible = isVisible(group.visibleWhen, valueMap);
                  const editable = groupVisible && group.items.some((item) => item.editable);
                  return (
                    <article
                      className="aoi-config-group-card"
                      data-visible={String(groupVisible)}
                      key={group.key}
                    >
                      <header className="aoi-config-group-card__header">
                        <div>
                          <h3>{group.label || group.key}</h3>
                          <p>{group.description || groupSummary(group, tx)}</p>
                        </div>
                        <div className="aoi-config-group-card__tools">
                          <span
                            className="aoi-config-status"
                            data-status={groupStatus(group, groupVisible)}
                          >
                            {groupStatusLabel(group, groupVisible, tx)}
                          </span>
                          <Popover
                            ariaLabel={t("admin.system.columns.flags")}
                            closeLabel={t("admin.system.actions.cancelEdit")}
                            title={groupStatusLabel(group, groupVisible, tx)}
                          >
                            <p>{group.description || groupSummary(group, tx)}</p>
                          </Popover>
                          <Button
                            appearance="secondary"
                            disabled={!editable || updateConfigMutation.isPending}
                            icon={<SlidersHorizontal size={16} />}
                            onClick={() => openConfigEditor(group)}
                          >
                            {t("admin.system.actions.editGroup")}
                          </Button>
                        </div>
                      </header>

                      {groupVisible ? (
                        group.items.length > 0 ? (
                          <div className="aoi-config-table">
                            <DataTable
                              columns={columns}
                              data={group.items}
                              emptyLabel={t("admin.system.empty")}
                            />
                          </div>
                        ) : (
                          <StateBlock
                            title={t("admin.system.states.groupEmptyTitle")}
                            description={t("admin.system.states.groupEmptyDescription")}
                          />
                        )
                      ) : (
                        <div className="aoi-config-inactive">
                          <EyeOff aria-hidden="true" size={20} />
                          <p>{t("admin.system.group.inactiveDescription")}</p>
                        </div>
                      )}
                    </article>
                  );
                })}
              </div>
            </div>
          </div>
        </section>
      ) : null}
    </section>
  );
}

type ConfigEditorPanelProps = {
  canSave: boolean;
  changedItems: SystemConfigItem[];
  draftValues: Record<string, ConfigDraftValue>;
  group: SystemConfigGroup;
  loading: boolean;
  showHeader?: boolean;
  t: Translate;
  onCancel: () => void;
  onChange: (key: string, value: ConfigDraftValue) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
};

function ConfigEditorPanel({
  canSave,
  changedItems,
  draftValues,
  group,
  loading,
  showHeader = true,
  t,
  onCancel,
  onChange,
  onSubmit,
}: ConfigEditorPanelProps) {
  const editableItems = group.items.filter((item) => item.editable);

  return (
    <section
      className="aoi-config-editor"
      aria-label={!showHeader ? t("admin.system.editor.title", { label: group.label }) : undefined}
      aria-labelledby={showHeader ? "aoi-config-editor-title" : undefined}
    >
      {showHeader ? (
        <header className="aoi-config-editor__header">
          <div>
            <span className="aoi-config-code">{group.key}</span>
            <h3 id="aoi-config-editor-title">
              {t("admin.system.editor.title", { label: group.label })}
            </h3>
            <p>{t("admin.system.editor.description")}</p>
          </div>
          <span className="aoi-config-count">
            {t("admin.system.editor.pendingCount", { count: changedItems.length })}
          </span>
        </header>
      ) : (
        <span className="aoi-config-count">
          {t("admin.system.editor.pendingCount", { count: changedItems.length })}
        </span>
      )}

      <form className="aoi-config-editor__form" onSubmit={onSubmit}>
        <div className="aoi-config-editor__fields">
          {editableItems.map((item) => (
            <ConfigEditorField
              draftValue={draftValues[item.key]}
              item={item}
              key={item.key}
              t={t}
              onChange={(value) => onChange(item.key, value)}
            />
          ))}
        </div>

        <section className="aoi-config-diff" aria-labelledby="aoi-config-diff-title">
          <h4 id="aoi-config-diff-title">{t("admin.system.editor.pendingChanges")}</h4>
          {changedItems.length === 0 ? (
            <p>{t("admin.system.editor.noChanges")}</p>
          ) : (
            <dl>
              {changedItems.map((item) => (
                <div key={item.key}>
                  <dt>{item.label || item.key}</dt>
                  <dd>{formatDraftPreview(item, draftValues, t)}</dd>
                </div>
              ))}
            </dl>
          )}
        </section>

        <div className="aoi-config-editor__actions">
          <Button disabled={!canSave} icon={<Save size={17} />} loading={loading} type="submit">
            {t("admin.system.actions.saveChanges")}
          </Button>
          <Button
            appearance="secondary"
            disabled={loading}
            icon={<X size={17} />}
            onClick={onCancel}
          >
            {t("admin.system.actions.cancelEdit")}
          </Button>
        </div>
      </form>
    </section>
  );
}

type ConfigEditorFieldProps = {
  draftValue: ConfigDraftValue | undefined;
  item: SystemConfigItem;
  t: Translate;
  onChange: (value: ConfigDraftValue) => void;
};

function ConfigEditorField({ draftValue, item, onChange, t }: ConfigEditorFieldProps) {
  const fieldId = `config-${safeFieldId(item.key)}`;
  const helpId = `${fieldId}-help`;
  const errorId = `${fieldId}-error`;
  const error = isDraftValid(item, { [item.key]: draftValue ?? "" })
    ? ""
    : t("admin.system.editor.invalidNumber");
  const label = item.secret ? t("admin.system.editor.newSecretValue") : item.label || item.key;
  const help = fieldHelp(item, t);

  if (item.valueType === "boolean" || item.editor === "switch") {
    return (
      <div className="aoi-config-switch">
        <input
          aria-describedby={helpId}
          checked={Boolean(draftValue)}
          id={fieldId}
          type="checkbox"
          onChange={(event) => onChange(event.currentTarget.checked)}
        />
        <label htmlFor={fieldId}>
          <strong>{item.label || item.key}</strong>
          <small id={helpId}>{help}</small>
          <code>{item.key}</code>
        </label>
      </div>
    );
  }

  if (item.editor === "select" || (item.options?.length ?? 0) > 0) {
    return (
      <div className="aoi-config-editor-field">
        <SelectField
          error={error || undefined}
          help={help}
          label={label}
          options={optionItems(item)}
          value={String(draftValue ?? "")}
          onChange={(event) => onChange(event.currentTarget.value)}
        />
        <small>{item.key}</small>
      </div>
    );
  }

  if (item.valueType === "array" || item.editor === "textarea") {
    return (
      <div className="aoi-config-editor-field">
        <div className="aoi-form-field">
          <label htmlFor={fieldId}>{label}</label>
          <textarea
            aria-describedby={`${helpId}${error ? ` ${errorId}` : ""}`}
            aria-invalid={Boolean(error)}
            id={fieldId}
            rows={5}
            value={String(draftValue ?? "")}
            onChange={(event) => onChange(event.currentTarget.value)}
          />
          <span id={helpId} className="aoi-form-field__help">
            {help}
          </span>
          {error ? (
            <span id={errorId} className="aoi-form-field__error">
              {error}
            </span>
          ) : null}
        </div>
        <small>{item.key}</small>
      </div>
    );
  }

  return (
    <div className="aoi-config-editor-field">
      <FormField
        error={error || undefined}
        help={help}
        label={label}
        step={item.valueType === "number" ? 1 : undefined}
        type={
          item.editor === "password" || item.secret
            ? "password"
            : item.valueType === "number"
              ? "number"
              : "text"
        }
        value={String(draftValue ?? "")}
        onChange={(event) => onChange(event.currentTarget.value)}
      />
      <small>{item.key}</small>
    </div>
  );
}

type ConfigStatCardProps = {
  icon: ReactNode;
  label: string;
  value: string;
};

function ConfigStatCard({ icon, label, value }: ConfigStatCardProps) {
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

function sortSections(sections: SystemConfigSection[]) {
  return [...sections].sort(
    (left, right) => left.order - right.order || left.code.localeCompare(right.code),
  );
}

function groupsForSection(section: SystemConfigSection | undefined): SystemConfigGroup[] {
  if (!section) {
    return [];
  }
  if (section.groups.length > 0) {
    return section.groups;
  }
  if (section.items.length === 0) {
    return [];
  }
  return [
    {
      description: section.description,
      items: section.items,
      key: "general",
      label: section.label,
      testable: false,
    },
  ];
}

function summarizeConfig(sections: SystemConfigSection[], valueMap: Map<string, unknown>) {
  const groups = sections.flatMap(groupsForSection);
  const items = uniqueItems(groups.flatMap((group) => group.items));
  return {
    editable: items.filter((item) => item.editable).length,
    items: items.length,
    secret: items.filter((item) => item.secret).length,
    sections: sections.length,
    visibleGroups: groups.filter((group) => isVisible(group.visibleWhen, valueMap)).length,
  };
}

function uniqueItems(items: SystemConfigItem[]) {
  const seen = new Set<string>();
  return items.filter((item) => {
    if (seen.has(item.key)) {
      return false;
    }
    seen.add(item.key);
    return true;
  });
}

function buildValueMap(sections: SystemConfigSection[]) {
  const values = new Map<string, unknown>();
  for (const section of sections) {
    for (const item of section.items) {
      values.set(item.key, item.value);
    }
    for (const group of section.groups) {
      for (const item of group.items) {
        values.set(item.key, item.value);
      }
    }
  }
  return values;
}

function isVisible(
  condition: SystemConfigVisibilityCondition | undefined,
  valueMap: Map<string, unknown>,
) {
  if (!condition || !condition.field || condition.in.length === 0) {
    return true;
  }
  return condition.in.includes(formatComparableValue(valueMap.get(condition.field)));
}

function formatConfigValue(item: SystemConfigItem, locale: string, t: Translate) {
  if (item.secret) {
    return t("admin.system.values.secret");
  }
  if (typeof item.value === "boolean") {
    return item.value ? t("admin.system.values.true") : t("admin.system.values.false");
  }
  if (typeof item.value === "number") {
    return new Intl.NumberFormat(locale).format(item.value);
  }
  if (Array.isArray(item.value)) {
    return item.value.length > 0
      ? item.value.map((value) => formatPlainValue(value, t)).join(", ")
      : t("common.labels.none");
  }
  if (item.value && typeof item.value === "object") {
    return JSON.stringify(item.value);
  }
  return formatPlainValue(item.value, t);
}

function formatPlainValue(value: unknown, t: Translate) {
  if (value === undefined || value === null || value === "") {
    return t("common.labels.none");
  }
  if (typeof value === "string") {
    return value;
  }
  if (typeof value === "number" || typeof value === "boolean" || typeof value === "bigint") {
    return String(value);
  }
  if (typeof value === "symbol") {
    return value.description || t("common.labels.none");
  }
  if (typeof value === "function") {
    return t("common.labels.none");
  }
  return JSON.stringify(value) ?? t("common.labels.none");
}

function formatComparableValue(value: unknown) {
  if (value === undefined || value === null || value === "") {
    return "";
  }
  if (typeof value === "string") {
    return value;
  }
  if (typeof value === "number" || typeof value === "boolean" || typeof value === "bigint") {
    return String(value);
  }
  if (typeof value === "symbol") {
    return value.description ?? "";
  }
  if (typeof value === "function") {
    return "";
  }
  return JSON.stringify(value) ?? "";
}

function formatNumber(value: number, locale: string) {
  return new Intl.NumberFormat(locale).format(value);
}

function valueTypeLabel(valueType: string, t: Translate) {
  const labelKey = valueTypeLabels[valueType];
  return labelKey ? t(labelKey) : valueType;
}

function groupSummary(group: SystemConfigGroup, t: Translate) {
  const editable = group.items.filter((item) => item.editable).length;
  return t("admin.system.group.summary", { count: group.items.length, editable });
}

function groupStatus(group: SystemConfigGroup, visible: boolean) {
  if (!visible) {
    return "inactive";
  }
  if (group.risk === "high") {
    return "danger";
  }
  if (group.risk === "medium" || group.testable) {
    return "warning";
  }
  return "ready";
}

function groupStatusLabel(group: SystemConfigGroup, visible: boolean, t: Translate) {
  if (!visible) {
    return t("admin.system.group.inactive");
  }
  const riskLabel = group.risk ? riskLabels[group.risk] : undefined;
  if (riskLabel) {
    return t(riskLabel);
  }
  if (group.testable) {
    return t("admin.system.group.testable");
  }
  return t("admin.system.group.ready");
}

function seedDraftValues(items: SystemConfigItem[]) {
  return items.reduce<Record<string, ConfigDraftValue>>((draft, item) => {
    if (item.editable) {
      draft[item.key] = draftValueForItem(item);
    }
    return draft;
  }, {});
}

function draftValueForItem(item: SystemConfigItem): ConfigDraftValue {
  if (item.valueType === "boolean") {
    return Boolean(item.value);
  }
  if (item.secret) {
    return "";
  }
  return formatDraftValue(item.value);
}

function draftPayload(item: SystemConfigItem, draftValues: Record<string, ConfigDraftValue>) {
  const draft = draftValues[item.key];
  if (item.secret && typeof draft === "string" && draft.trim() === "") {
    return undefined;
  }
  if (item.valueType === "boolean") {
    return Boolean(draft);
  }
  if (item.valueType === "number") {
    return Number(draft);
  }
  if (item.valueType === "array") {
    return String(draft ?? "")
      .split(/\r?\n/)
      .map((value) => value.trim())
      .filter(Boolean);
  }
  return String(draft ?? "");
}

function isDraftChanged(item: SystemConfigItem, draftValues: Record<string, ConfigDraftValue>) {
  if (item.secret && String(draftValues[item.key] ?? "").trim() === "") {
    return false;
  }
  return (
    JSON.stringify(draftPayload(item, draftValues)) !== JSON.stringify(normalizedCurrentValue(item))
  );
}

function normalizedCurrentValue(item: SystemConfigItem) {
  if (item.valueType === "boolean") {
    return Boolean(item.value);
  }
  if (item.valueType === "number") {
    return Number(item.value);
  }
  if (item.valueType === "array") {
    return Array.isArray(item.value) ? item.value.map((value) => formatDraftValue(value)) : [];
  }
  if (item.secret) {
    return undefined;
  }
  return formatDraftValue(item.value);
}

function isDraftValid(
  item: SystemConfigItem,
  draftValues: Record<string, ConfigDraftValue | undefined>,
) {
  if (item.valueType !== "number") {
    return true;
  }
  const value = Number(draftValues[item.key]);
  return Number.isFinite(value) && Number.isInteger(value);
}

function formatDraftValue(value: unknown) {
  if (value === undefined || value === null) {
    return "";
  }
  if (typeof value === "string") {
    return value;
  }
  if (typeof value === "number" || typeof value === "boolean" || typeof value === "bigint") {
    return String(value);
  }
  if (Array.isArray(value)) {
    return value.join("\n");
  }
  if (typeof value === "object") {
    return JSON.stringify(value) ?? "";
  }
  if (typeof value === "symbol") {
    return value.description ?? "";
  }
  return "";
}

function optionItems(item: SystemConfigItem): SelectOption[] {
  return (item.options ?? []).map((option) => ({
    label: option.label || option.value,
    value: option.value,
  }));
}

function fieldHelp(item: SystemConfigItem, t: Translate) {
  if (item.secret) {
    return t("admin.system.editor.secretHelp");
  }
  if (item.valueType === "number") {
    return t("admin.system.editor.numberHelp");
  }
  if (item.valueType === "array") {
    return t("admin.system.editor.arrayHelp");
  }
  if (item.valueType === "object") {
    return t("admin.system.editor.objectHelp");
  }
  return item.description || t("admin.system.editor.defaultHelp");
}

function formatDraftPreview(
  item: SystemConfigItem,
  draftValues: Record<string, ConfigDraftValue>,
  t: Translate,
) {
  if (item.secret) {
    return t("admin.system.editor.secretChanged");
  }
  const payload = draftPayload(item, draftValues);
  if (payload === undefined) {
    return t("common.labels.none");
  }
  if (Array.isArray(payload)) {
    return payload.length > 0 ? payload.join(", ") : t("common.labels.none");
  }
  if (typeof payload === "boolean") {
    return payload ? t("admin.system.values.true") : t("admin.system.values.false");
  }
  return String(payload);
}

function safeFieldId(value: string) {
  return value.replace(/[^a-zA-Z0-9_-]/g, "-");
}

function toError(error: unknown) {
  return error instanceof Error ? error : new Error(String(error));
}

function errorTitle(error: Error, t: Translate) {
  if (error instanceof ApiError && error.status === 403) {
    return t("admin.system.states.permissionTitle");
  }
  if (error instanceof ApiError && error.status === 401) {
    return t("errors.api.unauthorized");
  }
  return t("admin.system.states.errorTitle");
}

function errorDescription(error: Error, t: Translate) {
  if (error instanceof ApiError && error.status === 403) {
    return t("admin.system.states.permissionDescription");
  }
  return error.message || t("errors.api.requestFailed");
}
