import type { TFunction } from "i18next";
import { useTranslation } from "react-i18next";

export const clientTypeCodes = ["pc_web", "mobile_web", "mobile_app"] as const;

export function clientTypeOptions(t: TFunction) {
  return [
    { label: t("admin.platform.filters.all"), value: "" },
    ...clientTypeCodes.map((value) => ({
      label: clientTypeLabel(value, t),
      value,
    })),
  ];
}

export function clientTypeLabel(value: string | null | undefined, t: TFunction) {
  const clientType = String(value ?? "").trim();
  if (!clientType) {
    return t("admin.platform.empty");
  }
  if (clientTypeCodes.includes(clientType as (typeof clientTypeCodes)[number])) {
    return t(`admin.platform.clientTypes.${clientType}`);
  }
  return t("admin.platform.unknown", { value: clientType });
}

type PlatformTagProps = {
  clientType?: string | null;
  productCode?: string | null;
  showProductCode?: boolean;
};

export function PlatformTag({ clientType, productCode, showProductCode = true }: PlatformTagProps) {
  const { t } = useTranslation();
  const label = clientTypeLabel(clientType, t);
  const product = String(productCode ?? "").trim();

  return (
    <span
      className="aoi-platform-tag"
      aria-label={t("admin.platform.ariaLabel", {
        platform: label,
        product: product || t("common.labels.none"),
      })}
    >
      <span className="aoi-platform-tag__type">{label}</span>
      {showProductCode && product ? (
        <span className="aoi-platform-tag__product">
          {t("admin.platform.productCode", { value: product })}
        </span>
      ) : null}
    </span>
  );
}
