import type { HTMLAttributes } from "react";

import { cn } from "~/lib/cn";

type SkeletonProps = HTMLAttributes<HTMLDivElement> & {
  inline?: boolean;
};

export function Skeleton({ className, inline = false, ...props }: SkeletonProps) {
  const Element = inline ? "span" : "div";

  return <Element className={cn("aoi-skeleton", className)} aria-hidden="true" {...props} />;
}

export function SkeletonText({ className, lines = 1 }: { className?: string; lines?: number }) {
  return (
    <div className={cn("aoi-skeleton-text", className)} aria-hidden="true">
      {Array.from({ length: lines }).map((_, index) => (
        <Skeleton className="aoi-skeleton-text__line" key={index} />
      ))}
    </div>
  );
}
