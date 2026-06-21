import { useQuery } from "@tanstack/react-query";
import { useTranslation } from "react-i18next";

import { queryKeys } from "~/lib/api/query-keys";
import { systemApi } from "~/lib/api/system";

export function usePublicSettings() {
  const { i18n, t } = useTranslation();
  const settingsQuery = useQuery({
    queryFn: ({ signal }) => systemApi.getPublicSettings({ signal }),
    queryKey: queryKeys.system.publicSettings(i18n.language),
    retry: false,
    staleTime: 5 * 60 * 1000,
  });

  return {
    ...settingsQuery,
    brandName: settingsQuery.data?.brand.productName?.trim() || t("common.brand.name"),
  };
}
