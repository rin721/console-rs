import { Slot } from "@radix-ui/react-slot";
import type { ButtonHTMLAttributes, ReactNode } from "react";

import { cn } from "~/lib/cn";

type ButtonAppearance = "primary" | "secondary" | "ghost";

export type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  appearance?: ButtonAppearance;
  asChild?: boolean;
  icon?: ReactNode;
  loading?: boolean;
};

export function Button({
  appearance = "primary",
  asChild = false,
  children,
  className,
  disabled,
  icon,
  loading,
  type = "button",
  ...props
}: ButtonProps) {
  if (asChild) {
    return (
      <Slot
        className={cn("aoi-button", `aoi-button--${appearance}`, className)}
        aria-busy={loading || undefined}
        aria-disabled={disabled || loading || undefined}
        {...props}
      >
        {children}
      </Slot>
    );
  }

  return (
    <button
      className={cn("aoi-button", `aoi-button--${appearance}`, className)}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      type={type}
      {...props}
    >
      {icon ? <span aria-hidden="true">{icon}</span> : null}
      <span>{children}</span>
    </button>
  );
}
