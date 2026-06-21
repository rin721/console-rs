import { Link } from "react-router";
import { useTranslation } from "react-i18next";

import { Button } from "~/components/aoi/primitives/Button";
import { StateBlock } from "~/components/aoi/patterns/StateBlock";

export default function AdminNotFoundRoute() {
  const { t } = useTranslation();

  return (
    <StateBlock
      title={t("admin.notFound.title")}
      description={t("admin.notFound.description")}
      action={
        <Button asChild>
          <Link to="/admin">{t("admin.nav.dashboard")}</Link>
        </Button>
      }
    />
  );
}
