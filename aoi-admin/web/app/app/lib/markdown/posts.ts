import { z } from "zod";

import type { AppLocale } from "~/i18n/resources";
import { generatedBlogPosts } from "./generated-posts";

const frontMatterDate = z.preprocess((value) => {
  if (value instanceof Date) {
    return value.toISOString().slice(0, 10);
  }
  return value;
}, z.string().date());

export const blogFrontMatterSchema = z.object({
  author: z.string().min(1),
  cover: z.string().min(1),
  date: frontMatterDate,
  description: z.string().min(1),
  draft: z.boolean(),
  locale: z.enum(["zh-CN", "en"]),
  slug: z.string().min(1),
  tags: z.array(z.string().min(1)),
  title: z.string().min(1),
  updatedAt: frontMatterDate,
});

export type BlogPostMeta = z.infer<typeof blogFrontMatterSchema>;

export type BlogPost = BlogPostMeta & {
  content: string;
  highlightedCode: Record<string, string>;
  path: string;
};

export function getBlogPosts(locale: AppLocale): BlogPost[] {
  return parsePosts()
    .filter((post) => post.locale === locale && !post.draft)
    .sort((a, b) => b.date.localeCompare(a.date));
}

export function getBlogPost(locale: AppLocale, slug: string): BlogPost | null {
  return getBlogPosts(locale).find((post) => post.slug === slug) ?? null;
}

export function parsePosts(): BlogPost[] {
  return generatedBlogPosts.map((post) => blogPostSchema.parse(post));
}

const blogPostSchema = blogFrontMatterSchema.extend({
  content: z.string().min(1),
  highlightedCode: z.record(z.string(), z.string()),
  path: z.string().min(1),
});
