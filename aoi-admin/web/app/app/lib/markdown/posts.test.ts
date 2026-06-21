import { describe, expect, it } from "vitest";

import { getBlogPost, getBlogPosts, parsePosts } from "./posts";

describe("blog posts", () => {
  it("parses local markdown front matter", () => {
    const posts = parsePosts();
    expect(posts.length).toBeGreaterThanOrEqual(2);
    expect(posts.every((post) => post.slug && post.locale && post.content)).toBe(true);
  });

  it("filters published posts by locale", () => {
    expect(getBlogPosts("zh-CN").every((post) => post.locale === "zh-CN")).toBe(true);
    expect(getBlogPosts("en").every((post) => post.locale === "en")).toBe(true);
  });

  it("finds a localized post by slug", () => {
    expect(getBlogPost("en", "react-frontend-migration")?.title).toContain("React");
    expect(getBlogPost("zh-CN", "react-frontend-migration")?.title).toContain("React");
  });
});
