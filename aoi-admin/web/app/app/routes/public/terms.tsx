import { useTranslation } from "react-i18next";

import { Badge } from "~/components/aoi/primitives/Badge";
import { useDocumentMeta } from "~/hooks/useDocumentMeta";

const termSectionKeys = ["scope", "accounts", "content", "changes"] as const;

export default function TermsRoute() {
  const { t } = useTranslation();
  useDocumentMeta("seo.terms.title", "seo.terms.description", {
    canonicalPath: "/terms",
    ogDescriptionKey: "seo.terms.ogDescription",
    ogTitleKey: "seo.terms.ogTitle",
  });

  return (
    <main className="aoi-page aoi-page--narrow">
      <section className="aoi-section aoi-legal-page" aria-labelledby="terms-title">
        <Badge>{t("site.legal.eyebrow")}</Badge>
        <h1 id="terms-title">{t("site.legal.termsTitle")}</h1>
        <p className="aoi-section__description">{t("site.legal.termsDescription")}</p>
        <div className="aoi-stacked-list">
          {termSectionKeys.map((key) => (
            <article className="aoi-stacked-list__item" key={key}>
              <h2>{t(`site.legal.terms.${key}.title`)}</h2>
              <p>{t(`site.legal.terms.${key}.description`)}</p>
            </article>
          ))}
        </div>
      </section>
    </main>
  );
}
