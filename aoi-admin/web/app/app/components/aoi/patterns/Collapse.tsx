import { ChevronDown } from "lucide-react";
import { useId, useState, type ReactNode } from "react";

import { cn } from "~/lib/cn";

type CollapseProps = {
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
  defaultOpen?: boolean;
  description?: ReactNode;
  title: ReactNode;
};

export function Collapse({
  actions,
  children,
  className,
  defaultOpen = true,
  description,
  title,
}: CollapseProps) {
  const [open, setOpen] = useState(defaultOpen);
  const contentId = useId();

  return (
    <section className={cn("aoi-collapse", className)} data-state={open ? "open" : "closed"}>
      <header className="aoi-collapse__header">
        <button
          aria-controls={contentId}
          aria-expanded={open}
          className="aoi-collapse__trigger"
          type="button"
          onClick={() => setOpen((current) => !current)}
        >
          <span>
            <strong>{title}</strong>
            {description ? <small>{description}</small> : null}
          </span>
          <ChevronDown aria-hidden="true" size={18} />
        </button>
        {actions ? <div className="aoi-collapse__actions">{actions}</div> : null}
      </header>
      {open ? (
        <div className="aoi-collapse__content" id={contentId}>
          {children}
        </div>
      ) : null}
    </section>
  );
}
