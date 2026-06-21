import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { ColumnDef } from "@tanstack/react-table";
import {
  BookOpenText,
  Database,
  Hash,
  ListChecks,
  Pencil,
  Plus,
  RefreshCw,
  RotateCcw,
  Save,
  Search,
  Tags,
  Trash2,
  X,
} from "lucide-react";
import {
  useCallback,
  useMemo,
  useState,
  type FormEvent,
  type ReactNode,
} from "react";
import { useTranslation } from "react-i18next";
import { z } from "zod";

import { DataTable } from "~/components/aoi/patterns/DataTable";
import { FormField } from "~/components/aoi/patterns/FormField";
import { SelectField, type SelectOption } from "~/components/aoi/patterns/SelectField";
import { StateBlock } from "~/components/aoi/patterns/StateBlock";
import { Badge } from "~/components/aoi/primitives/Badge";
import { Button } from "~/components/aoi/primitives/Button";
import { ApiError } from "~/lib/api/client";
import { queryKeys } from "~/lib/api/query-keys";
import {
  systemApi,
  type SystemDictionaryInput,
  type SystemDictionaryItemInput,
  type SystemDictionaryItemUpdateInput,
  type SystemDictionaryUpdateInput,
} from "~/lib/api/system";
import type { SystemDictionary, SystemDictionaryItem } from "~/lib/api/types";

type DictionaryFilters = {
  keyword: string;
  status: string;
};

type DictionaryDraft = {
  code: string;
  description: string;
  name: string;
  status: "active" | "disabled";
};

type DictionaryItemDraft = {
  extra: string;
  label: string;
  sort: string;
  status: "active" | "disabled";
  value: string;
};

type DictionaryFormState =
  | {
      mode: "create";
    }
  | {
      dictionary: SystemDictionary;
      mode: "edit";
    };

type DictionaryItemFormState =
  | {
      dictionary: SystemDictionary;
      mode: "create";
    }
  | {
      dictionary: SystemDictionary;
      item: SystemDictionaryItem;
      mode: "edit";
    };

type DictionaryNotice = {
  description: string;
  intent?: "danger" | "info";
  title: string;
};

type PendingDelete =
  | {
      dictionary: SystemDictionary;
      mode: "dictionary";
    }
  | {
      dictionary: SystemDictionary;
      item: SystemDictionaryItem;
      mode: "item";
    };

const dictionaryCodePattern = /^[a-z0-9][a-z0-9._-]*$/;
const dictionaryCreateSchema = z.object({
  code: z.string().trim().regex(dictionaryCodePattern),
  description: z.string().trim(),
  name: z.string().trim().min(1),
  status: z.enum(["active", "disabled"]),
});
const dictionaryUpdateSchema = dictionaryCreateSchema.omit({ code: true });
const dictionaryItemSchema = z.object({
  extra: z.string().trim(),
  label: z.string().trim().min(1),
  sort: z.number().int(),
  status: z.enum(["active", "disabled"]),
  value: z.string().trim().min(1),
});

const emptyFilters: DictionaryFilters = {
  keyword: "",
  status: "",
};

const emptyDictionaryDraft: DictionaryDraft = {
  code: "",
  description: "",
  name: "",
  status: "active",
};

const emptyDictionaryItemDraft: DictionaryItemDraft = {
  extra: "",
  label: "",
  sort: "0",
  status: "active",
  value: "",
};

export default function AdminDictionariesRoute() {
  const { i18n, t } = useTranslation();
  const queryClient = useQueryClient();
  const [filters, setFilters] = useState<DictionaryFilters>(emptyFilters);
  const [dictionaryForm, setDictionaryForm] = useState<DictionaryFormState | null>(null);
  const [dictionaryDraft, setDictionaryDraft] = useState<DictionaryDraft>(emptyDictionaryDraft);
  const [itemForm, setItemForm] = useState<DictionaryItemFormState | null>(null);
  const [itemDraft, setItemDraft] = useState<DictionaryItemDraft>(emptyDictionaryItemDraft);
  const [pendingDelete, setPendingDelete] = useState<PendingDelete | null>(null);
  const [notice, setNotice] = useState<DictionaryNotice | null>(null);

  const dictionariesQuery = useQuery({
    queryFn: ({ signal }) => systemApi.listDictionaries({ signal }),
    queryKey: queryKeys.system.dictionaries(i18n.language),
  });

  const invalidateDictionaries = () =>
    queryClient.invalidateQueries({ queryKey: ["system", "dictionaries"] });

  const closeDictionaryForm = useCallback(() => {
    setDictionaryForm(null);
    setDictionaryDraft(emptyDictionaryDraft);
  }, []);

  const closeItemForm = useCallback(() => {
    setItemForm(null);
    setItemDraft(emptyDictionaryItemDraft);
  }, []);

  const createDictionaryMutation = useMutation({
    mutationFn: (input: SystemDictionaryInput) => systemApi.createDictionary(input),
    onError: (error) => {
      setNotice({
        description: mutationErrorDescription(toError(error), "dictionary:create", t),
        intent: "danger",
        title: t("admin.dictionaries.messages.saveFailedTitle"),
      });
    },
    onSuccess: (dictionary) => {
      closeDictionaryForm();
      setNotice({
        description: t("admin.dictionaries.messages.createdDescription", {
          name: dictionary.name,
        }),
        title: t("admin.dictionaries.messages.createdTitle"),
      });
      void invalidateDictionaries();
    },
  });

  const updateDictionaryMutation = useMutation({
    mutationFn: (input: { id: number | string; value: SystemDictionaryUpdateInput }) =>
      systemApi.updateDictionary(input.id, input.value),
    onError: (error) => {
      setNotice({
        description: mutationErrorDescription(toError(error), "dictionary:update", t),
        intent: "danger",
        title: t("admin.dictionaries.messages.saveFailedTitle"),
      });
    },
    onSuccess: (dictionary) => {
      closeDictionaryForm();
      setNotice({
        description: t("admin.dictionaries.messages.updatedDescription", {
          name: dictionary.name,
        }),
        title: t("admin.dictionaries.messages.updatedTitle"),
      });
      void invalidateDictionaries();
    },
  });

  const deleteDictionaryMutation = useMutation({
    mutationFn: (dictionary: SystemDictionary) => systemApi.deleteDictionary(dictionary.id),
    onError: (error) => {
      setNotice({
        description: mutationErrorDescription(toError(error), "dictionary:delete", t),
        intent: "danger",
        title: t("admin.dictionaries.messages.deleteFailedTitle"),
      });
    },
    onSuccess: (_result, dictionary) => {
      const id = dictionaryIdValue(dictionary);
      if (dictionaryForm?.mode === "edit" && dictionaryIdValue(dictionaryForm.dictionary) === id) {
        closeDictionaryForm();
      }
      if (itemForm && dictionaryIdValue(itemForm.dictionary) === id) {
        closeItemForm();
      }
      setPendingDelete(null);
      setNotice({
        description: t("admin.dictionaries.messages.deletedDescription", {
          name: dictionary.name,
        }),
        title: t("admin.dictionaries.messages.deletedTitle"),
      });
      void invalidateDictionaries();
    },
  });

  const createDictionaryItemMutation = useMutation({
    mutationFn: (input: {
      dictionary: SystemDictionary;
      value: SystemDictionaryItemInput;
    }) => systemApi.createDictionaryItem(input.dictionary.id, input.value),
    onError: (error) => {
      setNotice({
        description: mutationErrorDescription(toError(error), "dictionary:update", t),
        intent: "danger",
        title: t("admin.dictionaries.messages.saveFailedTitle"),
      });
    },
    onSuccess: (item, input) => {
      closeItemForm();
      setNotice({
        description: t("admin.dictionaries.messages.itemCreatedDescription", {
          dictionary: input.dictionary.name,
          label: item.label,
        }),
        title: t("admin.dictionaries.messages.itemCreatedTitle"),
      });
      void invalidateDictionaries();
    },
  });

  const updateDictionaryItemMutation = useMutation({
    mutationFn: (input: { item: SystemDictionaryItem; value: SystemDictionaryItemUpdateInput }) =>
      systemApi.updateDictionaryItem(input.item.id, input.value),
    onError: (error) => {
      setNotice({
        description: mutationErrorDescription(toError(error), "dictionary:update", t),
        intent: "danger",
        title: t("admin.dictionaries.messages.saveFailedTitle"),
      });
    },
    onSuccess: (item) => {
      closeItemForm();
      setNotice({
        description: t("admin.dictionaries.messages.itemUpdatedDescription", {
          label: item.label,
        }),
        title: t("admin.dictionaries.messages.itemUpdatedTitle"),
      });
      void invalidateDictionaries();
    },
  });

  const deleteDictionaryItemMutation = useMutation({
    mutationFn: (item: SystemDictionaryItem) => systemApi.deleteDictionaryItem(item.id),
    onError: (error) => {
      setNotice({
        description: mutationErrorDescription(toError(error), "dictionary:delete", t),
        intent: "danger",
        title: t("admin.dictionaries.messages.deleteFailedTitle"),
      });
    },
    onSuccess: (_result, item) => {
      const id = itemIdValue(item);
      if (itemForm?.mode === "edit" && itemIdValue(itemForm.item) === id) {
        closeItemForm();
      }
      setPendingDelete(null);
      setNotice({
        description: t("admin.dictionaries.messages.itemDeletedDescription", {
          label: item.label,
        }),
        title: t("admin.dictionaries.messages.itemDeletedTitle"),
      });
      void invalidateDictionaries();
    },
  });

  const dictionaries = useMemo(() => dictionariesQuery.data?.items ?? [], [dictionariesQuery.data]);
  const summary = useMemo(() => summarizeDictionaries(dictionaries), [dictionaries]);
  const filteredDictionaries = useMemo(
    () => filterDictionaries(dictionaries, filters, t),
    [dictionaries, filters, t],
  );
  const filteredItemCount = useMemo(
    () => filteredDictionaries.reduce((count, dictionary) => count + dictionary.items.length, 0),
    [filteredDictionaries],
  );
  const storagePersisted = dictionariesQuery.data?.storageStatus === "persisted";
  const writePending =
    createDictionaryMutation.isPending ||
    updateDictionaryMutation.isPending ||
    deleteDictionaryMutation.isPending ||
    createDictionaryItemMutation.isPending ||
    updateDictionaryItemMutation.isPending ||
    deleteDictionaryItemMutation.isPending;
  const dictionaryDraftValid =
    dictionaryForm?.mode === "edit"
      ? Boolean(dictionaryDraft.name.trim())
      : Boolean(dictionaryDraft.code.trim() && dictionaryDraft.name.trim());
  const itemDraftValid = Boolean(itemDraft.label.trim() && itemDraft.value.trim());

  const statusOptions = useMemo<SelectOption[]>(
    () => [
      { label: t("admin.dictionaries.filters.allStatuses"), value: "" },
      { label: t("admin.dictionaries.status.active"), value: "active" },
      { label: t("admin.dictionaries.status.disabled"), value: "disabled" },
    ],
    [t],
  );

  const writeStatusOptions = useMemo<SelectOption[]>(
    () => [
      { label: t("admin.dictionaries.status.active"), value: "active" },
      { label: t("admin.dictionaries.status.disabled"), value: "disabled" },
    ],
    [t],
  );

  const startCreateDictionary = () => {
    setDictionaryForm({ mode: "create" });
    setDictionaryDraft(emptyDictionaryDraft);
    closeItemForm();
    setPendingDelete(null);
    setNotice(null);
  };

  const startEditDictionary = useCallback(
    (dictionary: SystemDictionary) => {
      setDictionaryForm({ dictionary, mode: "edit" });
      setDictionaryDraft({
        code: dictionary.code,
        description: dictionary.description ?? "",
        name: dictionary.name,
        status: normalizeStatus(dictionary.status),
      });
      closeItemForm();
      setPendingDelete(null);
      setNotice(null);
    },
    [closeItemForm],
  );

  const startCreateItem = useCallback(
    (dictionary: SystemDictionary) => {
      setItemForm({ dictionary, mode: "create" });
      setItemDraft(emptyDictionaryItemDraft);
      closeDictionaryForm();
      setPendingDelete(null);
      setNotice(null);
    },
    [closeDictionaryForm],
  );

  const startEditItem = useCallback(
    (dictionary: SystemDictionary, item: SystemDictionaryItem) => {
      setItemForm({ dictionary, item, mode: "edit" });
      setItemDraft({
        extra: item.extra ?? "",
        label: item.label,
        sort: String(item.sort ?? 0),
        status: normalizeStatus(item.status),
        value: item.value,
      });
      closeDictionaryForm();
      setPendingDelete(null);
      setNotice(null);
    },
    [closeDictionaryForm],
  );

  const createItemColumns = useCallback(
    (dictionary: SystemDictionary): ColumnDef<SystemDictionaryItem>[] => [
      {
        accessorKey: "label",
        cell: ({ row }) => (
          <div className="aoi-dictionary-item-name">
            <strong>{row.original.label}</strong>
            <span>{row.original.id}</span>
          </div>
        ),
        header: t("admin.dictionaries.columns.label"),
      },
      {
        accessorKey: "value",
        cell: ({ getValue }) => <code className="aoi-dictionary-code">{String(getValue())}</code>,
        header: t("admin.dictionaries.columns.value"),
      },
      {
        accessorKey: "sort",
        cell: ({ getValue }) => formatNumber(Number(getValue()), i18n.language),
        header: t("admin.dictionaries.columns.sort"),
      },
      {
        accessorKey: "status",
        cell: ({ getValue }) => (
          <StatusPill status={String(getValue())} label={statusLabel(String(getValue()), t)} />
        ),
        header: t("admin.dictionaries.columns.status"),
      },
      {
        accessorKey: "extra",
        cell: ({ row }) => {
          const value = row.original.extra;
          return value ? (
            <span className="aoi-dictionary-extra">{value}</span>
          ) : (
            t("common.labels.none")
          );
        },
        header: t("admin.dictionaries.columns.extra"),
      },
      {
        accessorKey: "updatedAt",
        cell: ({ getValue }) => formatDate(String(getValue()), i18n.language),
        header: t("admin.dictionaries.columns.updatedAt"),
      },
      {
        id: "actions",
        cell: ({ row }) => (
          <div className="aoi-dictionary-item-actions">
            <Button
              appearance="secondary"
              aria-label={t("admin.dictionaries.actions.editItemFor", {
                label: row.original.label,
              })}
              disabled={!storagePersisted || writePending}
              icon={<Pencil size={15} />}
              onClick={() => startEditItem(dictionary, row.original)}
            >
              {t("admin.dictionaries.actions.edit")}
            </Button>
            <Button
              appearance="ghost"
              aria-label={t("admin.dictionaries.actions.deleteItemFor", {
                label: row.original.label,
              })}
              disabled={!storagePersisted || writePending}
              icon={<Trash2 size={15} />}
              onClick={() => setPendingDelete({ dictionary, item: row.original, mode: "item" })}
            >
              {t("admin.dictionaries.actions.delete")}
            </Button>
          </div>
        ),
        header: t("admin.dictionaries.columns.actions"),
      },
    ],
    [i18n.language, startEditItem, storagePersisted, t, writePending],
  );

  const updateFilter = (key: keyof DictionaryFilters, value: string) => {
    setFilters((current) => ({ ...current, [key]: value }));
  };

  const resetFilters = () => {
    setFilters(emptyFilters);
  };

  const updateDictionaryDraft = <K extends keyof DictionaryDraft>(
    key: K,
    value: DictionaryDraft[K],
  ) => {
    setDictionaryDraft((current) => ({ ...current, [key]: value }));
  };

  const updateItemDraft = <K extends keyof DictionaryItemDraft>(
    key: K,
    value: DictionaryItemDraft[K],
  ) => {
    setItemDraft((current) => ({ ...current, [key]: value }));
  };

  const submitDictionary = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!dictionaryForm) {
      return;
    }
    if (!storagePersisted) {
      setNotice({
        description: t("admin.dictionaries.states.storageUnavailableDescription"),
        intent: "danger",
        title: t("admin.dictionaries.states.storageUnavailableTitle"),
      });
      return;
    }

    if (dictionaryForm.mode === "edit") {
      const normalized = normalizeDictionaryUpdateDraft(dictionaryDraft);
      if (!normalized.ok) {
        setNotice({
          description: normalized.description,
          intent: "danger",
          title: t("admin.dictionaries.validation.dictionaryTitle"),
        });
        return;
      }
      setNotice(null);
      updateDictionaryMutation.mutate({
        id: dictionaryForm.dictionary.id,
        value: normalized.value,
      });
      return;
    }
    const normalized = normalizeDictionaryCreateDraft(dictionaryDraft);
    if (!normalized.ok) {
      setNotice({
        description: normalized.description,
        intent: "danger",
        title: t("admin.dictionaries.validation.dictionaryTitle"),
      });
      return;
    }
    setNotice(null);
    createDictionaryMutation.mutate(normalized.value);
  };

  const submitItem = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!itemForm) {
      return;
    }
    if (!storagePersisted) {
      setNotice({
        description: t("admin.dictionaries.states.storageUnavailableDescription"),
        intent: "danger",
        title: t("admin.dictionaries.states.storageUnavailableTitle"),
      });
      return;
    }

    const normalized = normalizeDictionaryItemDraft(itemDraft);
    if (!normalized.ok) {
      setNotice({
        description: normalized.description,
        intent: "danger",
        title: t("admin.dictionaries.validation.itemTitle"),
      });
      return;
    }

    setNotice(null);
    if (itemForm.mode === "edit") {
      updateDictionaryItemMutation.mutate({
        item: itemForm.item,
        value: normalized.value,
      });
      return;
    }
    createDictionaryItemMutation.mutate({
      dictionary: itemForm.dictionary,
      value: normalized.value,
    });
  };

  const confirmPendingDelete = () => {
    if (!pendingDelete || !storagePersisted) {
      return;
    }
    setNotice(null);
    if (pendingDelete.mode === "dictionary") {
      deleteDictionaryMutation.mutate(pendingDelete.dictionary);
      return;
    }
    deleteDictionaryItemMutation.mutate(pendingDelete.item);
  };

  const normalizeDictionaryCreateDraft = (
    draft: DictionaryDraft,
  ): { ok: true; value: SystemDictionaryInput } | { description: string; ok: false } => {
    const payload = {
      code: draft.code,
      description: draft.description,
      name: draft.name,
      status: draft.status,
    };
    const result = dictionaryCreateSchema.safeParse(payload);

    if (!result.success) {
      return {
        description: dictionaryValidationDescription(result.error, t),
        ok: false,
      };
    }
    return { ok: true, value: result.data };
  };

  const normalizeDictionaryUpdateDraft = (
    draft: DictionaryDraft,
  ): { ok: true; value: SystemDictionaryUpdateInput } | { description: string; ok: false } => {
    const result = dictionaryUpdateSchema.safeParse({
      description: draft.description,
      name: draft.name,
      status: draft.status,
    });

    if (!result.success) {
      return {
        description: dictionaryValidationDescription(result.error, t),
        ok: false,
      };
    }
    return { ok: true, value: result.data };
  };

  const normalizeDictionaryItemDraft = (
    draft: DictionaryItemDraft,
  ): { ok: true; value: SystemDictionaryItemInput } | { description: string; ok: false } => {
    const sort = draft.sort.trim() ? Number(draft.sort) : 0;
    const result = dictionaryItemSchema.safeParse({
      extra: draft.extra,
      label: draft.label,
      sort,
      status: draft.status,
      value: draft.value,
    });

    if (!result.success) {
      return {
        description: dictionaryItemValidationDescription(result.error, t),
        ok: false,
      };
    }
    return { ok: true, value: result.data };
  };

  return (
    <section className="aoi-admin-dashboard" aria-labelledby="admin-dictionaries-title">
      <div className="aoi-admin-page-header">
        <div>
          <Badge>{t("admin.dictionaries.badge")}</Badge>
          <h1 id="admin-dictionaries-title">{t("admin.dictionaries.title")}</h1>
          <p>{t("admin.dictionaries.description")}</p>
        </div>
        <div className="aoi-dictionary-page-actions">
          <Button
            disabled={!storagePersisted || writePending}
            icon={<Plus size={17} />}
            onClick={startCreateDictionary}
          >
            {t("admin.dictionaries.actions.create")}
          </Button>
          <Button
            appearance="secondary"
            icon={<RefreshCw size={17} />}
            loading={dictionariesQuery.isFetching}
            onClick={() => void dictionariesQuery.refetch()}
          >
            {t("admin.dictionaries.actions.refresh")}
          </Button>
        </div>
      </div>

      {dictionariesQuery.error ? (
        <StateBlock
          intent="danger"
          title={errorTitle(dictionariesQuery.error, t)}
          description={errorDescription(dictionariesQuery.error, t)}
        />
      ) : null}

      {notice ? (
        <StateBlock description={notice.description} intent={notice.intent} title={notice.title} />
      ) : null}

      {pendingDelete ? (
        <StateBlock
          action={
            <div className="aoi-dictionary-confirm-actions">
              <Button loading={writePending} onClick={confirmPendingDelete}>
                {t("admin.dictionaries.actions.confirmDelete")}
              </Button>
              <Button
                appearance="secondary"
                disabled={writePending}
                onClick={() => setPendingDelete(null)}
              >
                {t("admin.dictionaries.actions.cancel")}
              </Button>
            </div>
          }
          description={
            pendingDelete.mode === "dictionary"
              ? t("admin.dictionaries.delete.dictionaryDescription", {
                  count: pendingDelete.dictionary.items.length,
                  name: pendingDelete.dictionary.name,
                })
              : t("admin.dictionaries.delete.itemDescription", {
                  label: pendingDelete.item.label,
                  name: pendingDelete.dictionary.name,
                })
          }
          title={
            pendingDelete.mode === "dictionary"
              ? t("admin.dictionaries.delete.dictionaryTitle")
              : t("admin.dictionaries.delete.itemTitle")
          }
        />
      ) : null}

      {dictionaryForm ? (
        <section className="aoi-admin-panel aoi-dictionary-form-panel">
          <header className="aoi-admin-panel-header-row">
            <div>
              <h2>
                {dictionaryForm.mode === "edit"
                  ? t("admin.dictionaries.form.editDictionaryTitle")
                  : t("admin.dictionaries.form.createDictionaryTitle")}
              </h2>
              <p>{t("admin.dictionaries.form.dictionaryDescription")}</p>
            </div>
            {dictionaryForm.mode === "edit" ? <Badge>{dictionaryForm.dictionary.id}</Badge> : null}
          </header>
          <form className="aoi-dictionary-form-grid" onSubmit={submitDictionary}>
            <FormField
              disabled={dictionaryForm.mode === "edit" || writePending}
              help={t("admin.dictionaries.form.codeHelp")}
              label={t("admin.dictionaries.form.code")}
              placeholder={t("admin.dictionaries.form.placeholders.code")}
              value={dictionaryDraft.code}
              onChange={(event) => updateDictionaryDraft("code", event.currentTarget.value)}
            />
            <FormField
              disabled={writePending}
              label={t("admin.dictionaries.form.name")}
              placeholder={t("admin.dictionaries.form.placeholders.name")}
              value={dictionaryDraft.name}
              onChange={(event) => updateDictionaryDraft("name", event.currentTarget.value)}
            />
            <SelectField
              disabled={writePending}
              label={t("admin.dictionaries.form.status")}
              options={writeStatusOptions}
              value={dictionaryDraft.status}
              onChange={(event) =>
                updateDictionaryDraft("status", normalizeStatus(event.currentTarget.value))
              }
            />
            <label className="aoi-form-field aoi-dictionary-form-field--span">
              <span>{t("admin.dictionaries.form.descriptionField")}</span>
              <textarea
                disabled={writePending}
                placeholder={t("admin.dictionaries.form.placeholders.description")}
                rows={3}
                value={dictionaryDraft.description}
                onChange={(event) =>
                  updateDictionaryDraft("description", event.currentTarget.value)
                }
              />
            </label>
            <div className="aoi-dictionary-form-actions">
              <Button
                disabled={!dictionaryDraftValid || !storagePersisted}
                icon={<Save size={17} />}
                loading={createDictionaryMutation.isPending || updateDictionaryMutation.isPending}
                type="submit"
              >
                {dictionaryForm.mode === "edit"
                  ? t("admin.dictionaries.actions.save")
                  : t("admin.dictionaries.actions.submitCreate")}
              </Button>
              <Button
                appearance="secondary"
                disabled={writePending}
                icon={<X size={17} />}
                onClick={closeDictionaryForm}
              >
                {t("admin.dictionaries.actions.cancel")}
              </Button>
            </div>
          </form>
        </section>
      ) : null}

      {itemForm ? (
        <section className="aoi-admin-panel aoi-dictionary-form-panel">
          <header className="aoi-admin-panel-header-row">
            <div>
              <h2>
                {itemForm.mode === "edit"
                  ? t("admin.dictionaries.form.editItemTitle")
                  : t("admin.dictionaries.form.createItemTitle")}
              </h2>
              <p>
                {t("admin.dictionaries.form.itemDescription", {
                  name: itemForm.dictionary.name,
                })}
              </p>
            </div>
            <Badge>{itemForm.dictionary.code}</Badge>
          </header>
          <form className="aoi-dictionary-form-grid" onSubmit={submitItem}>
            <FormField
              disabled={writePending}
              label={t("admin.dictionaries.form.itemLabel")}
              placeholder={t("admin.dictionaries.form.placeholders.itemLabel")}
              value={itemDraft.label}
              onChange={(event) => updateItemDraft("label", event.currentTarget.value)}
            />
            <FormField
              disabled={writePending}
              label={t("admin.dictionaries.form.itemValue")}
              placeholder={t("admin.dictionaries.form.placeholders.itemValue")}
              value={itemDraft.value}
              onChange={(event) => updateItemDraft("value", event.currentTarget.value)}
            />
            <FormField
              disabled={writePending}
              label={t("admin.dictionaries.form.itemSort")}
              step={1}
              type="number"
              value={itemDraft.sort}
              onChange={(event) => updateItemDraft("sort", event.currentTarget.value)}
            />
            <SelectField
              disabled={writePending}
              label={t("admin.dictionaries.form.itemStatus")}
              options={writeStatusOptions}
              value={itemDraft.status}
              onChange={(event) =>
                updateItemDraft("status", normalizeStatus(event.currentTarget.value))
              }
            />
            <label className="aoi-form-field aoi-dictionary-form-field--span">
              <span>{t("admin.dictionaries.form.itemExtra")}</span>
              <textarea
                disabled={writePending}
                placeholder={t("admin.dictionaries.form.placeholders.itemExtra")}
                rows={3}
                value={itemDraft.extra}
                onChange={(event) => updateItemDraft("extra", event.currentTarget.value)}
              />
            </label>
            <div className="aoi-dictionary-form-actions">
              <Button
                disabled={!itemDraftValid || !storagePersisted}
                icon={<Save size={17} />}
                loading={
                  createDictionaryItemMutation.isPending || updateDictionaryItemMutation.isPending
                }
                type="submit"
              >
                {itemForm.mode === "edit"
                  ? t("admin.dictionaries.actions.saveItem")
                  : t("admin.dictionaries.actions.submitCreateItem")}
              </Button>
              <Button
                appearance="secondary"
                disabled={writePending}
                icon={<X size={17} />}
                onClick={closeItemForm}
              >
                {t("admin.dictionaries.actions.cancel")}
              </Button>
            </div>
          </form>
        </section>
      ) : null}

      <div className="aoi-admin-stat-grid" aria-label={t("admin.dictionaries.summaryLabel")}>
        <DictionaryStatCard
          icon={<BookOpenText size={19} />}
          label={t("admin.dictionaries.metrics.dictionaries")}
          value={
            dictionariesQuery.data
              ? formatNumber(dictionariesQuery.data.total, i18n.language)
              : fallbackValue(dictionariesQuery.isLoading, t)
          }
        />
        <DictionaryStatCard
          icon={<Tags size={19} />}
          label={t("admin.dictionaries.metrics.items")}
          value={formatNumber(summary.items, i18n.language)}
        />
        <DictionaryStatCard
          icon={<ListChecks size={19} />}
          label={t("admin.dictionaries.metrics.activeDictionaries")}
          value={formatNumber(summary.activeDictionaries, i18n.language)}
        />
        <DictionaryStatCard
          icon={<Hash size={19} />}
          label={t("admin.dictionaries.metrics.activeItems")}
          value={formatNumber(summary.activeItems, i18n.language)}
        />
        <DictionaryStatCard
          icon={<Database size={19} />}
          label={t("admin.dictionaries.metrics.storage")}
          value={
            dictionariesQuery.data
              ? storageStatusLabel(dictionariesQuery.data.storageStatus, t)
              : fallbackValue(dictionariesQuery.isLoading, t)
          }
        />
      </div>

      <section className="aoi-admin-panel">
        <header>
          <h2>{t("admin.dictionaries.filters.title")}</h2>
          <p>{t("admin.dictionaries.filters.description")}</p>
        </header>
        <form
          className="aoi-admin-filter-form aoi-admin-filter-form--compact"
          onSubmit={(event) => event.preventDefault()}
        >
          <FormField
            label={t("admin.dictionaries.filters.keyword")}
            value={filters.keyword}
            onChange={(event) => updateFilter("keyword", event.currentTarget.value)}
          />
          <SelectField
            label={t("admin.dictionaries.filters.status")}
            options={statusOptions}
            value={filters.status}
            onChange={(event) => updateFilter("status", event.currentTarget.value)}
          />
          <div className="aoi-admin-filter-actions">
            <Button appearance="secondary" icon={<RotateCcw size={17} />} onClick={resetFilters}>
              {t("admin.dictionaries.actions.reset")}
            </Button>
          </div>
        </form>
      </section>

      <section className="aoi-admin-panel">
        <header className="aoi-admin-panel-header-row">
          <div>
            <h2>{t("admin.dictionaries.list.title")}</h2>
            <p>
              {t("admin.dictionaries.list.description", {
                count: filteredDictionaries.length,
                items: filteredItemCount,
                total: dictionaries.length,
              })}
            </p>
          </div>
          <span className="aoi-api-count">
            <Search aria-hidden="true" size={16} />
            {formatNumber(filteredDictionaries.length, i18n.language)}
          </span>
        </header>

        {dictionariesQuery.isLoading ? (
          <StateBlock
            title={t("admin.dictionaries.states.loadingTitle")}
            description={t("admin.dictionaries.states.loadingDescription")}
          />
        ) : dictionariesQuery.data ? (
          <>
            {storagePersisted ? null : (
              <StateBlock
                title={t("admin.dictionaries.states.storageUnavailableTitle")}
                description={t("admin.dictionaries.states.storageUnavailableDescription")}
              />
            )}
            {filteredDictionaries.length > 0 ? (
              <div className="aoi-dictionary-groups">
                {filteredDictionaries.map((dictionary) => (
                  <section className="aoi-dictionary-group" key={dictionary.id}>
                    <header>
                      <div className="aoi-dictionary-title">
                        <BookOpenText aria-hidden="true" size={18} />
                        <div>
                          <h3>{dictionary.name}</h3>
                          <p>
                            {t("admin.dictionaries.dictionaryMeta", {
                              code: dictionary.code,
                              count: dictionary.items.length,
                            })}
                          </p>
                        </div>
                      </div>
                      <div className="aoi-dictionary-actions">
                        <StatusPill
                          status={dictionary.status}
                          label={statusLabel(dictionary.status, t)}
                        />
                        <Button
                          appearance="secondary"
                          aria-label={t("admin.dictionaries.actions.editDictionaryFor", {
                            name: dictionary.name,
                          })}
                          disabled={!storagePersisted || writePending}
                          icon={<Pencil size={15} />}
                          onClick={() => startEditDictionary(dictionary)}
                        >
                          {t("admin.dictionaries.actions.edit")}
                        </Button>
                        <Button
                          appearance="secondary"
                          aria-label={t("admin.dictionaries.actions.addItemFor", {
                            name: dictionary.name,
                          })}
                          disabled={!storagePersisted || writePending}
                          icon={<Plus size={15} />}
                          onClick={() => startCreateItem(dictionary)}
                        >
                          {t("admin.dictionaries.actions.addItem")}
                        </Button>
                        <Button
                          appearance="ghost"
                          aria-label={t("admin.dictionaries.actions.deleteDictionaryFor", {
                            name: dictionary.name,
                          })}
                          disabled={!storagePersisted || writePending}
                          icon={<Trash2 size={15} />}
                          onClick={() => setPendingDelete({ dictionary, mode: "dictionary" })}
                        >
                          {t("admin.dictionaries.actions.delete")}
                        </Button>
                      </div>
                    </header>
                    {dictionary.description ? (
                      <p className="aoi-dictionary-description">{dictionary.description}</p>
                    ) : null}
                    <div className="aoi-dictionary-table">
                      <DataTable
                        columns={createItemColumns(dictionary)}
                        data={dictionary.items}
                        emptyLabel={t("admin.dictionaries.emptyItems")}
                      />
                    </div>
                  </section>
                ))}
              </div>
            ) : (
              <StateBlock
                title={t("admin.dictionaries.states.noMatchesTitle")}
                description={t("admin.dictionaries.empty")}
              />
            )}
          </>
        ) : (
          <StateBlock
            title={t("admin.dictionaries.states.emptyTitle")}
            description={t("admin.dictionaries.states.emptyDescription")}
          />
        )}
      </section>
    </section>
  );
}

type DictionaryStatCardProps = {
  icon: ReactNode;
  label: string;
  value: string;
};

function DictionaryStatCard({ icon, label, value }: DictionaryStatCardProps) {
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

function StatusPill({ label, status }: { label: string; status: string }) {
  return (
    <span className="aoi-dictionary-status" data-status={status || "unknown"}>
      {label}
    </span>
  );
}

function summarizeDictionaries(dictionaries: SystemDictionary[]) {
  return dictionaries.reduce(
    (summary, dictionary) => {
      if (dictionary.status === "active") {
        summary.activeDictionaries += 1;
      }
      summary.items += dictionary.items.length;
      summary.activeItems += dictionary.items.filter((item) => item.status === "active").length;
      return summary;
    },
    { activeDictionaries: 0, activeItems: 0, items: 0 },
  );
}

function filterDictionaries(
  dictionaries: SystemDictionary[],
  filters: DictionaryFilters,
  t: ReturnType<typeof useTranslation>["t"],
) {
  const keyword = filters.keyword.trim().toLowerCase();
  return dictionaries.filter((dictionary) => {
    const statusMatch = !filters.status || dictionary.status === filters.status;
    return statusMatch && matchesDictionary(dictionary, keyword, t);
  });
}

function matchesDictionary(
  dictionary: SystemDictionary,
  keyword: string,
  t: ReturnType<typeof useTranslation>["t"],
) {
  if (!keyword) {
    return true;
  }
  return [
    dictionary.code,
    dictionary.description,
    dictionary.name,
    statusLabel(dictionary.status, t),
    ...dictionary.items.flatMap((item) => [
      item.extra,
      item.label,
      item.value,
      statusLabel(item.status, t),
    ]),
  ].some((value) => value?.toLowerCase().includes(keyword));
}

function statusLabel(status: string, t: ReturnType<typeof useTranslation>["t"]) {
  if (status === "active") {
    return t("admin.dictionaries.status.active");
  }
  if (status === "disabled") {
    return t("admin.dictionaries.status.disabled");
  }
  return status || t("admin.dictionaries.status.unknown");
}

function normalizeStatus(status: string): "active" | "disabled" {
  return status === "disabled" ? "disabled" : "active";
}

function dictionaryIdValue(dictionary: SystemDictionary) {
  return String(dictionary.id);
}

function itemIdValue(item: SystemDictionaryItem) {
  return String(item.id);
}

function dictionaryValidationDescription(
  error: z.ZodError,
  t: ReturnType<typeof useTranslation>["t"],
) {
  if (error.issues.some((issue) => issue.path[0] === "code")) {
    return t("admin.dictionaries.validation.dictionaryCodeDescription");
  }
  return t("admin.dictionaries.validation.dictionaryRequiredDescription");
}

function dictionaryItemValidationDescription(
  error: z.ZodError,
  t: ReturnType<typeof useTranslation>["t"],
) {
  if (error.issues.some((issue) => issue.path[0] === "sort")) {
    return t("admin.dictionaries.validation.itemSortDescription");
  }
  return t("admin.dictionaries.validation.itemRequiredDescription");
}

function fallbackValue(loading: boolean, t: ReturnType<typeof useTranslation>["t"]) {
  return loading ? t("loading.app") : t("common.labels.none");
}

function storageStatusLabel(status: string, t: ReturnType<typeof useTranslation>["t"]) {
  if (status === "persisted") {
    return t("admin.dictionaries.storage.persisted");
  }
  if (status === "unavailable") {
    return t("admin.dictionaries.storage.unavailable");
  }
  return status || t("admin.dictionaries.storage.unknown");
}

function formatNumber(value: number, locale: string) {
  return new Intl.NumberFormat(locale).format(value);
}

function formatDate(value: string, locale: string) {
  const timestamp = Date.parse(value);
  if (Number.isNaN(timestamp)) {
    return value;
  }
  return new Intl.DateTimeFormat(locale, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(timestamp);
}

function errorTitle(error: Error, t: ReturnType<typeof useTranslation>["t"]) {
  if (error instanceof ApiError && error.status === 403) {
    return t("admin.dictionaries.states.permissionTitle");
  }
  if (error instanceof ApiError && error.status === 401) {
    return t("errors.api.unauthorized");
  }
  return t("admin.dictionaries.states.errorTitle");
}

function errorDescription(error: Error, t: ReturnType<typeof useTranslation>["t"]) {
  if (error instanceof ApiError && error.status === 403) {
    return t("admin.dictionaries.states.permissionDescription");
  }
  return error.message || t("errors.api.requestFailed");
}

function mutationErrorDescription(
  error: Error,
  permission: string,
  t: ReturnType<typeof useTranslation>["t"],
) {
  if (error instanceof ApiError && error.status === 403) {
    return t("admin.dictionaries.states.writePermissionDescription", { permission });
  }
  if (error instanceof ApiError && error.status === 401) {
    return t("errors.api.unauthorized");
  }
  return error.message || t("errors.api.requestFailed");
}

function toError(error: unknown) {
  if (error instanceof Error) {
    return error;
  }
  return new Error(String(error));
}
