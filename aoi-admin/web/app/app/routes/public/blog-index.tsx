import { Link } from "react-router";
import { useTranslation } from "react-i18next";

import { Badge } from "~/components/aoi/primitives/Badge";
import { Button } from "~/components/aoi/primitives/Button";
import { StateBlock } from "~/components/aoi/patterns/StateBlock";
import { useDocumentMeta } from "~/hooks/useDocumentMeta";
import { getBlogPosts } from "~/lib/markdown/posts";
import type { AppLocale } from "~/i18n/resources";

export default function BlogIndexRoute() {
  const { i18n, t } = useTranslation();
  useDocumentMeta("seo.blog.title", "seo.blog.description", {
    canonicalPath: "/blog",
    ogDescriptionKey: "seo.blog.ogDescription",
    ogTitleKey: "seo.blog.ogTitle",
  });
  const posts = getBlogPosts(i18n.language as AppLocale);

  return (
    <main className="aoi-page">
      <section className="aoi-section" aria-labelledby="blog-title">
        <div className="aoi-section__header">
          <Badge>{t("site.blog.eyebrow")}</Badge>
          <h1 id="blog-title">{t("site.blog.title")}</h1>
          <p className="aoi-section__description">{t("site.blog.description")}</p>
        </div>
        {posts.length ? (
          <div className="aoi-blog-grid">
            {posts.map((post) => (
              <article className="aoi-card aoi-blog-card" key={post.slug}>
                <img className="aoi-blog-card__cover" src={post.cover} alt={post.title} />
                <div className="aoi-blog-card__body">
                  <div className="aoi-blog-card__meta">
                    <Badge>{post.tags[0]}</Badge>
                    <span>
                      {t("markdown.blog.dateLabel")}: {post.date}
                    </span>
                  </div>
                  <h2>{post.title}</h2>
                  <p>{post.description}</p>
                  <Button appearance="ghost" asChild>
                    <Link to={`/blog/${post.slug}`}>{t("common.actions.readArticle")}</Link>
                  </Button>
                </div>
              </article>
            ))}
          </div>
        ) : (
          <StateBlock
            title={t("site.blog.emptyTitle")}
            description={t("site.blog.emptyDescription")}
          />
        )}
      </section>
    </main>
  );
}
