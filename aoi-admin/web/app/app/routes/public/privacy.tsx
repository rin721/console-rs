import { useTranslation } from "react-i18next";

import { Badge } from "~/components/aoi/primitives/Badge";
import { useDocumentMeta } from "~/hooks/useDocumentMeta";

const privacySectionKeys = ["collection", "storage", "integrations", "launch"] as const;

export default function PrivacyRoute() {
  const { t } = useTranslation();
  useDocumentMeta("seo.privacy.title", "seo.privacy.description", {
    canonicalPath: "/privacy",
    ogDescriptionKey: "seo.privacy.ogDescription",
    ogTitleKey: "seo.privacy.ogTitle",
  });

  return (
    <main className="aoi-page aoi-page--narrow">
      <section className="aoi-section aoi-legal-page" aria-labelledby="privacy-title">
        <Badge>{t("site.legal.eyebrow")}</Badge>
        <h1 id="privacy-title">{t("site.legal.privacyTitle")}</h1>
        <p className="aoi-section__description">{t("site.legal.privacyDescription")}</p>
        <div className="aoi-stacked-list">
          {privacySectionKeys.map((key) => (
            <article className="aoi-stacked-list__item" key={key}>
              <h2>{t(`site.legal.privacy.${key}.title`)}</h2>
              <p>{t(`site.legal.privacy.${key}.description`)}</p>
            </article>
          ))}
        </div>
      </section>
    </main>
  );
}
