import { Link, useParams } from "react-router";
import { useTranslation } from "react-i18next";

import { Badge } from "~/components/aoi/primitives/Badge";
import { Button } from "~/components/aoi/primitives/Button";
import { MarkdownProse } from "~/components/aoi/patterns/MarkdownProse";
import { StateBlock } from "~/components/aoi/patterns/StateBlock";
import { useDocumentMeta } from "~/hooks/useDocumentMeta";
import { useJsonLd } from "~/hooks/useJsonLd";
import type { AppLocale } from "~/i18n/resources";
import { getBlogPost } from "~/lib/markdown/posts";

export default function BlogDetailRoute() {
  const { i18n, t } = useTranslation();
  const params = useParams();
  const post = getBlogPost(i18n.language as AppLocale, params.slug ?? "");
  useDocumentMeta("seo.blog.title", "seo.blog.description", {
    article: post
      ? {
          author: post.author,
          modifiedTime: post.updatedAt,
          publishedTime: post.date,
        }
      : undefined,
    canonicalPath: post ? `/blog/${post.slug}` : "/blog",
    description: post?.description,
    image: post?.cover,
    title: post?.title,
    type: post ? "article" : "website",
  });
  useJsonLd(
    "blog-article",
    post
      ? {
          "@context": "https://schema.org",
          "@type": "Article",
          author: {
            "@type": "Organization",
            name: post.author,
          },
          dateModified: post.updatedAt,
          datePublished: post.date,
          description: post.description,
          headline: post.title,
          image: post.cover,
          mainEntityOfPage: {
            "@type": "WebPage",
            "@id": `/blog/${post.slug}`,
          },
        }
      : null,
  );

  if (!post) {
    return (
      <main className="aoi-page aoi-page--narrow">
        <StateBlock
          title={t("empty.notFound.title")}
          description={t("empty.notFound.description")}
          action={
            <Button asChild>
              <Link to="/blog">{t("common.actions.viewBlog")}</Link>
            </Button>
          }
        />
      </main>
    );
  }

  return (
    <main className="aoi-page aoi-page--narrow">
      <header className="aoi-article-header">
        <div className="aoi-blog-card__meta">
          <Badge>{post.tags[0]}</Badge>
          <span>
            {t("markdown.blog.authorLabel")}: {post.author}
          </span>
        </div>
        <h1>{post.title}</h1>
        <p>{post.description}</p>
        <dl className="aoi-article-header__meta">
          <div>
            <dt>{t("markdown.blog.dateLabel")}</dt>
            <dd>{post.date}</dd>
          </div>
          <div>
            <dt>{t("markdown.blog.updatedLabel")}</dt>
            <dd>{post.updatedAt}</dd>
          </div>
        </dl>
      </header>
      <img className="aoi-article-cover" src={post.cover} alt={post.title} />
      <MarkdownProse content={post.content} highlightedCode={post.highlightedCode} />
    </main>
  );
}
