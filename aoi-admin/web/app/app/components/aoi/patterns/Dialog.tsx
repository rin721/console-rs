import * as DialogPrimitive from "@radix-ui/react-dialog";
import { X } from "lucide-react";
import type { ReactNode } from "react";

import { Button } from "~/components/aoi/primitives/Button";
import { cn } from "~/lib/cn";
import { useDialogFocusReturn } from "./useDialogFocusReturn";

type DialogProps = {
  children?: ReactNode;
  className?: string;
  closeLabel: string;
  description?: ReactNode;
  footer?: ReactNode;
  open: boolean;
  title: ReactNode;
  onOpenChange: (open: boolean) => void;
};

export function Dialog({
  children,
  className,
  closeLabel,
  description,
  footer,
  open,
  title,
  onOpenChange,
}: DialogProps) {
  const focusReturn = useDialogFocusReturn();

  return (
    <DialogPrimitive.Root open={open} onOpenChange={onOpenChange}>
      <DialogPrimitive.Portal>
        <DialogPrimitive.Overlay className="aoi-dialog-overlay" />
        <DialogPrimitive.Content
          className={cn("aoi-dialog-content", className)}
          onCloseAutoFocus={focusReturn.onCloseAutoFocus}
          onOpenAutoFocus={focusReturn.onOpenAutoFocus}
        >
          <div className="aoi-dialog-header">
            <DialogPrimitive.Title className="aoi-dialog-title">{title}</DialogPrimitive.Title>
            {description ? (
              <DialogPrimitive.Description className="aoi-dialog-description">
                {description}
              </DialogPrimitive.Description>
            ) : null}
          </div>
          {children ? <div className="aoi-dialog-body">{children}</div> : null}
          {footer ? <div className="aoi-dialog-footer">{footer}</div> : null}
          <DialogPrimitive.Close asChild>
            <Button
              appearance="ghost"
              aria-label={closeLabel}
              className="aoi-dialog-close aoi-icon-button"
              icon={<X size={17} />}
            >
              <span className="aoi-sr-only">{closeLabel}</span>
            </Button>
          </DialogPrimitive.Close>
        </DialogPrimitive.Content>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  );
}
