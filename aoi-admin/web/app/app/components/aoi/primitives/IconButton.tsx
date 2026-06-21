import { forwardRef, type ButtonHTMLAttributes, type ReactNode } from "react";

import { Tooltip } from "~/components/aoi/primitives/Tooltip";
import { cn } from "~/lib/cn";

type IconButtonAppearance = "default" | "ghost" | "primary";

export type IconButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  appearance?: IconButtonAppearance;
  icon: ReactNode;
  label: string;
  tooltip?: ReactNode;
};

export const IconButton = forwardRef<HTMLButtonElement, IconButtonProps>(function IconButton(
  { appearance = "default", className, icon, label, tooltip, type = "button", ...props },
  ref,
) {
  const button = (
    <button
      ref={ref}
      aria-label={label}
      className={cn("aoi-icon-button", `aoi-icon-button--${appearance}`, className)}
      type={type}
      {...props}
    >
      {icon}
    </button>
  );

  return tooltip ? (
    <Tooltip content={tooltip} side="bottom">
      {button}
    </Tooltip>
  ) : (
    button
  );
});
