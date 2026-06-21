import {
  ArrowRight,
  BookOpen,
  Boxes,
  CheckCircle2,
  FileText,
  Gauge,
  Languages,
  Network,
  ShieldCheck,
  Sparkles,
  Wrench,
} from "lucide-react";
import { Link } from "react-router";
import { useTranslation } from "react-i18next";

import { Badge } from "~/components/aoi/primitives/Badge";
import { Button } from "~/components/aoi/primitives/Button";
import { useDocumentMeta } from "~/hooks/useDocumentMeta";
import { useJsonLd } from "~/hooks/useJsonLd";
import { usePublicSettings } from "~/hooks/usePublicSettings";

const surfaceKeys = ["site", "setup", "admin", "productLines"] as const;
const surfaceIcons = {
  admin: Gauge,
  productLines: Boxes,
  setup: Wrench,
  site: FileText,
};

const railKeys = ["i18n", "api", "markdown", "quality"] as const;
const railIcons = {
  api: Network,
  i18n: Languages,
  markdown: BookOpen,
  quality: ShieldCheck,
};

const capabilityKeys = ["iam", "tenant", "configuration", "audit", "plugins", "media", "versions"] as const;
const previewSurfaceKeys = ["site", "setup", "admin", "productLines"] as const;

export default function HomeRoute() {
  const { t } = useTranslation();
  const { brandName } = usePublicSettings();
  useDocumentMeta("seo.home.title", "seo.home.description", {
    canonicalPath: "/",
    ogDescriptionKey: "seo.home.ogDescription",
    ogTitleKey: "seo.home.ogTitle",
  });
  useJsonLd("home", {
    "@context": "https://schema.org",
    "@type": "SoftwareApplication",
    applicationCategory: "BusinessApplication",
    description: t("seo.home.description"),
    name: brandName,
    operatingSystem: "Web",
  });

  return (
    <main className="aoi-page">
      <section className="aoi-hero aoi-hero--public" aria-labelledby="home-title">
        <div className="aoi-hero__copy">
          <p className="aoi-eyebrow">{t("site.home.eyebrow")}</p>
          <h1 id="home-title">{t("site.home.title")}</h1>
          <p>{t("site.home.description")}</p>
          <div className="aoi-hero__actions">
            <Button asChild>
              <Link to="/admin">{t("site.home.primaryCta")}</Link>
            </Button>
            <Button appearance="secondary" asChild>
              <Link to="/blog">{t("site.home.secondaryCta")}</Link>
            </Button>
          </div>
        </div>
        <ProductPreview />
      </section>

      <section className="aoi-section" aria-labelledby="home-surfaces">
        <div className="aoi-section__header">
          <Badge>{t("site.home.surfaces.eyebrow")}</Badge>
          <h2 id="home-surfaces">{t("site.home.surfaces.title")}</h2>
          <p className="aoi-section__description">{t("site.home.surfaces.description")}</p>
        </div>
        <div className="aoi-grid">
          {surfaceKeys.map((key) => {
            const Icon = surfaceIcons[key];
            return (
              <article className="aoi-card" key={key}>
                <Icon aria-hidden="true" size={24} />
                <h3>{t(`site.home.surfaces.items.${key}.title`)}</h3>
                <p>{t(`site.home.surfaces.items.${key}.description`)}</p>
              </article>
            );
          })}
        </div>
      </section>

      <section className="aoi-section aoi-section--split" aria-labelledby="home-capabilities">
        <div className="aoi-section__header">
          <Badge>{t("site.home.capabilities.eyebrow")}</Badge>
          <h2 id="home-capabilities">{t("site.home.capabilities.title")}</h2>
          <p className="aoi-section__description">{t("site.home.capabilities.description")}</p>
        </div>
        <div className="aoi-capability-list" aria-label={t("site.home.capabilities.listLabel")}>
          {capabilityKeys.map((key) => (
            <div className="aoi-capability-list__item" key={key}>
              <CheckCircle2 aria-hidden="true" size={20} />
              <span>{t(`site.home.capabilities.items.${key}`)}</span>
            </div>
          ))}
        </div>
      </section>

      <section className="aoi-section" aria-labelledby="home-rails">
        <div className="aoi-section__header">
          <Badge>{t("site.home.rails.eyebrow")}</Badge>
          <h2 id="home-rails">{t("site.home.rails.title")}</h2>
          <p className="aoi-section__description">{t("site.home.rails.description")}</p>
        </div>
        <div className="aoi-grid">
          {railKeys.map((key) => {
            const Icon = railIcons[key];
            return (
              <article className="aoi-card" key={key}>
                <Icon aria-hidden="true" size={24} />
                <h3>{t(`site.home.rails.items.${key}.title`)}</h3>
                <p>{t(`site.home.rails.items.${key}.description`)}</p>
              </article>
            );
          })}
        </div>
      </section>

      <section className="aoi-section aoi-cta-band" aria-labelledby="home-cta">
        <div>
          <Badge>{t("site.home.cta.eyebrow")}</Badge>
          <h2 id="home-cta">{t("site.home.cta.title")}</h2>
          <p>{t("site.home.cta.description")}</p>
        </div>
        <div className="aoi-cta-band__actions">
          <Button asChild>
            <Link to="/setup">
              <span>{t("site.home.cta.primary")}</span>
              <ArrowRight aria-hidden="true" size={18} />
            </Link>
          </Button>
          <Button appearance="secondary" asChild>
            <Link to="/about">{t("site.home.cta.secondary")}</Link>
          </Button>
        </div>
      </section>
    </main>
  );
}

function ProductPreview() {
  const { t } = useTranslation();

  return (
    <figure className="aoi-preview aoi-preview--map" aria-label={t("a11y.dashboardPreview")}>
      <div className="aoi-preview__bar" aria-hidden="true">
        <span className="aoi-preview__dot" />
        <span className="aoi-preview__dot" />
        <span className="aoi-preview__dot" />
        <span className="aoi-preview__bar-label">{t("site.home.preview.chromeLabel")}</span>
      </div>
      <div className="aoi-preview__map">
        <div className="aoi-preview__summary">
          <Sparkles aria-hidden="true" size={22} />
          <div>
            <h2>{t("site.home.preview.title")}</h2>
            <p>{t("site.home.preview.description")}</p>
          </div>
        </div>
        <div className="aoi-preview__surfaces">
          {previewSurfaceKeys.map((key) => (
            <section className="aoi-preview__surface" key={key}>
              <span>{t(`site.home.preview.surfaces.${key}.label`)}</span>
              <h3>{t(`site.home.preview.surfaces.${key}.title`)}</h3>
              <p>{t(`site.home.preview.surfaces.${key}.description`)}</p>
            </section>
          ))}
        </div>
      </div>
    </figure>
  );
}
