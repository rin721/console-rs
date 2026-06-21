import { Info, X } from "lucide-react";
import { useEffect, useId, useRef, useState, type ReactNode } from "react";

import { Button } from "~/components/aoi/primitives/Button";
import { cn } from "~/lib/cn";

type PopoverProps = {
  ariaLabel: string;
  children: ReactNode;
  className?: string;
  closeLabel?: string;
  title?: ReactNode;
  trigger?: ReactNode;
};

export function Popover({
  ariaLabel,
  children,
  className,
  closeLabel = ariaLabel,
  title,
  trigger,
}: PopoverProps) {
  const [open, setOpen] = useState(false);
  const contentId = useId();
  const rootRef = useRef<HTMLDivElement | null>(null);
  const triggerRef = useRef<HTMLButtonElement | null>(null);

  useEffect(() => {
    if (!open) {
      return;
    }

    const handlePointerDown = (event: PointerEvent) => {
      if (rootRef.current?.contains(event.target as Node)) {
        return;
      }
      setOpen(false);
    };
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setOpen(false);
        triggerRef.current?.focus();
      }
    };

    document.addEventListener("pointerdown", handlePointerDown);
    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("pointerdown", handlePointerDown);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [open]);

  return (
    <div className={cn("aoi-popover", className)} ref={rootRef}>
      <button
        ref={triggerRef}
        aria-controls={open ? contentId : undefined}
        aria-expanded={open}
        aria-label={ariaLabel}
        aria-haspopup="dialog"
        className="aoi-popover-trigger"
        type="button"
        onClick={() => setOpen((current) => !current)}
      >
        {trigger ?? <Info aria-hidden="true" size={16} />}
      </button>
      {open ? (
        <div className="aoi-popover-content" id={contentId} role="dialog">
          <div className="aoi-popover-content__inner">
            {title ? <strong>{title}</strong> : null}
            <div>{children}</div>
          </div>
          <Button
            appearance="ghost"
            aria-label={closeLabel}
            className="aoi-popover-close aoi-icon-button"
            icon={<X size={15} />}
            onClick={() => {
              setOpen(false);
              triggerRef.current?.focus();
            }}
          >
            <span className="aoi-sr-only">{closeLabel}</span>
          </Button>
        </div>
      ) : null}
    </div>
  );
}
