import type { ReactNode } from "react";
import ReactMarkdown from "react-markdown";
import rehypeSanitize from "rehype-sanitize";
import remarkGfm from "remark-gfm";

import { codeBlockId } from "~/lib/markdown/highlight";

type MarkdownProseProps = {
  content: string;
  highlightedCode?: Record<string, string>;
};

export function MarkdownProse({ content, highlightedCode = {} }: MarkdownProseProps) {
  return (
    <article className="aoi-prose">
      <ReactMarkdown
        components={{
          code({ children, className, ...props }) {
            const language = /language-([A-Za-z0-9_+-]+)/.exec(className ?? "")?.[1] ?? "text";
            const code = codeNodeText(children).replace(/\n$/, "");
            const html = highlightedCode[codeBlockId(language, code)];
            if (html) {
              return <div className="aoi-code-block" dangerouslySetInnerHTML={{ __html: html }} />;
            }
            return (
              <code className={className} {...props}>
                {children}
              </code>
            );
          },
          pre({ children }) {
            return <>{children}</>;
          },
        }}
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[rehypeSanitize]}
      >
        {content}
      </ReactMarkdown>
    </article>
  );
}

function codeNodeText(children: ReactNode): string {
  if (typeof children === "string" || typeof children === "number") {
    return String(children);
  }
  if (Array.isArray(children)) {
    return children.map(codeNodeText).join("");
  }
  return "";
}
