import * as DialogPrimitive from "@radix-ui/react-dialog";
import { X } from "lucide-react";
import type { ReactNode } from "react";

import { Button } from "~/components/aoi/primitives/Button";
import { cn } from "~/lib/cn";
import { useDialogFocusReturn } from "./useDialogFocusReturn";

type DrawerSide = "bottom" | "left" | "right" | "top";

type DrawerProps = {
  children: ReactNode;
  className?: string;
  closeLabel: string;
  description?: ReactNode;
  footer?: ReactNode;
  open: boolean;
  side?: DrawerSide;
  title: ReactNode;
  onOpenChange: (open: boolean) => void;
};

export function Drawer({
  children,
  className,
  closeLabel,
  description,
  footer,
  open,
  side = "right",
  title,
  onOpenChange,
}: DrawerProps) {
  const focusReturn = useDialogFocusReturn();

  return (
    <DialogPrimitive.Root open={open} onOpenChange={onOpenChange}>
      <DialogPrimitive.Portal>
        <DialogPrimitive.Overlay className="aoi-dialog-overlay" />
        <DialogPrimitive.Content
          className={cn("aoi-drawer-content", `aoi-drawer-content--${side}`, className)}
          onCloseAutoFocus={focusReturn.onCloseAutoFocus}
          onOpenAutoFocus={focusReturn.onOpenAutoFocus}
        >
          <header className="aoi-drawer-header">
            <div>
              <DialogPrimitive.Title className="aoi-drawer-title">{title}</DialogPrimitive.Title>
              {description ? (
                <DialogPrimitive.Description className="aoi-drawer-description">
                  {description}
                </DialogPrimitive.Description>
              ) : null}
            </div>
            <DialogPrimitive.Close asChild>
              <Button
                appearance="ghost"
                aria-label={closeLabel}
                className="aoi-icon-button"
                icon={<X size={17} />}
              >
                <span className="aoi-sr-only">{closeLabel}</span>
              </Button>
            </DialogPrimitive.Close>
          </header>
          <div className="aoi-drawer-body">{children}</div>
          {footer ? <footer className="aoi-drawer-footer">{footer}</footer> : null}
        </DialogPrimitive.Content>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  );
}
