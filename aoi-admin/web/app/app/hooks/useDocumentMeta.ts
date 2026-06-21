import { useEffect } from "react";
import { useTranslation } from "react-i18next";

import type { TranslationKey } from "~/i18n/keys";

type DocumentMetaOptions = {
  article?: {
    author?: string;
    modifiedTime?: string;
    publishedTime?: string;
  };
  canonicalPath?: string;
  description?: string;
  image?: string;
  ogDescription?: string;
  ogDescriptionKey?: TranslationKey;
  ogTitle?: string;
  ogTitleKey?: TranslationKey;
  title?: string;
  type?: "article" | "website";
};

export function useDocumentMeta(
  titleKey: TranslationKey,
  descriptionKey: TranslationKey,
  options: DocumentMetaOptions = {},
) {
  const { i18n, t } = useTranslation();

  useEffect(() => {
    const title = options.title ?? t(titleKey);
    const description = options.description ?? t(descriptionKey);
    const ogTitle = options.ogTitle ?? (options.ogTitleKey ? t(options.ogTitleKey) : title);
    const ogDescription =
      options.ogDescription ??
      (options.ogDescriptionKey ? t(options.ogDescriptionKey) : description);
    const canonicalUrl = options.canonicalPath
      ? new URL(options.canonicalPath, window.location.origin).toString()
      : new URL(window.location.pathname, window.location.origin).toString();

    document.title = title;
    document.documentElement.lang = i18n.language;

    setNamedMeta("description", description);
    setNamedMeta("twitter:card", options.image ? "summary_large_image" : "summary");
    setNamedMeta("twitter:title", ogTitle);
    setNamedMeta("twitter:description", ogDescription);
    setPropertyMeta("og:title", ogTitle);
    setPropertyMeta("og:description", ogDescription);
    setPropertyMeta("og:type", options.type ?? "website");
    setPropertyMeta("og:url", canonicalUrl);
    setPropertyMeta("og:locale", toOpenGraphLocale(i18n.language));
    setOptionalPropertyMeta("og:image", options.image ? absoluteUrl(options.image) : undefined);
    setOptionalNamedMeta("twitter:image", options.image ? absoluteUrl(options.image) : undefined);
    setOptionalPropertyMeta("article:published_time", options.article?.publishedTime);
    setOptionalPropertyMeta("article:modified_time", options.article?.modifiedTime);
    setOptionalPropertyMeta("article:author", options.article?.author);
    setCanonicalLink(canonicalUrl);
  }, [
    descriptionKey,
    i18n.language,
    options.article?.author,
    options.article?.modifiedTime,
    options.article?.publishedTime,
    options.canonicalPath,
    options.description,
    options.image,
    options.ogDescription,
    options.ogDescriptionKey,
    options.ogTitle,
    options.ogTitleKey,
    options.title,
    options.type,
    t,
    titleKey,
  ]);
}

function setNamedMeta(name: string, content: string) {
  setMeta("name", name, content);
}

function setPropertyMeta(property: string, content: string) {
  setMeta("property", property, content);
}

function setOptionalNamedMeta(name: string, content: string | undefined) {
  setOptionalMeta("name", name, content);
}

function setOptionalPropertyMeta(property: string, content: string | undefined) {
  setOptionalMeta("property", property, content);
}

function setOptionalMeta(attribute: "name" | "property", value: string, content: string | undefined) {
  if (!content) {
    document.querySelector<HTMLMetaElement>(`meta[${attribute}="${value}"]`)?.remove();
    return;
  }

  setMeta(attribute, value, content);
}

function setMeta(attribute: "name" | "property", value: string, content: string) {
  let meta = document.querySelector<HTMLMetaElement>(`meta[${attribute}="${value}"]`);
  if (!meta) {
    meta = document.createElement("meta");
    meta.setAttribute(attribute, value);
    document.head.append(meta);
  }
  meta.content = content;
}

function setCanonicalLink(href: string) {
  let link = document.querySelector<HTMLLinkElement>('link[rel="canonical"]');
  if (!link) {
    link = document.createElement("link");
    link.rel = "canonical";
    document.head.append(link);
  }
  link.href = href;
}

function absoluteUrl(pathOrUrl: string) {
  return new URL(pathOrUrl, window.location.origin).toString();
}

function toOpenGraphLocale(locale: string) {
  return locale === "en" ? "en_US" : "zh_CN";
}
