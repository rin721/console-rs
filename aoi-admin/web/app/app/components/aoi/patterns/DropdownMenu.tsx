import * as DropdownMenuPrimitive from "@radix-ui/react-dropdown-menu";
import { Check } from "lucide-react";
import { forwardRef, type ComponentPropsWithoutRef, type ElementRef, type ReactNode } from "react";

import { cn } from "~/lib/cn";

type DropdownMenuProps = ComponentPropsWithoutRef<typeof DropdownMenuPrimitive.Root>;

export function DropdownMenu({ modal = false, ...props }: DropdownMenuProps) {
  return <DropdownMenuPrimitive.Root modal={modal} {...props} />;
}

export const DropdownMenuGroup = DropdownMenuPrimitive.Group;
export const DropdownMenuRadioGroup = DropdownMenuPrimitive.RadioGroup;
export const DropdownMenuTrigger = DropdownMenuPrimitive.Trigger;

export const DropdownMenuContent = forwardRef<
  ElementRef<typeof DropdownMenuPrimitive.Content>,
  ComponentPropsWithoutRef<typeof DropdownMenuPrimitive.Content>
>(function DropdownMenuContent({ align = "end", className, sideOffset = 8, ...props }, ref) {
  return (
    <DropdownMenuPrimitive.Portal>
      <DropdownMenuPrimitive.Content
        ref={ref}
        align={align}
        className={cn("aoi-dropdown-menu-content", className)}
        sideOffset={sideOffset}
        {...props}
      />
    </DropdownMenuPrimitive.Portal>
  );
});

export const DropdownMenuLabel = forwardRef<
  ElementRef<typeof DropdownMenuPrimitive.Label>,
  ComponentPropsWithoutRef<typeof DropdownMenuPrimitive.Label>
>(function DropdownMenuLabel({ className, ...props }, ref) {
  return (
    <DropdownMenuPrimitive.Label
      ref={ref}
      className={cn("aoi-dropdown-menu-label", className)}
      {...props}
    />
  );
});

export const DropdownMenuSeparator = forwardRef<
  ElementRef<typeof DropdownMenuPrimitive.Separator>,
  ComponentPropsWithoutRef<typeof DropdownMenuPrimitive.Separator>
>(function DropdownMenuSeparator({ className, ...props }, ref) {
  return (
    <DropdownMenuPrimitive.Separator
      ref={ref}
      className={cn("aoi-dropdown-menu-separator", className)}
      {...props}
    />
  );
});

type DropdownMenuItemProps = ComponentPropsWithoutRef<typeof DropdownMenuPrimitive.Item> & {
  icon?: ReactNode;
};

export const DropdownMenuItem = forwardRef<
  ElementRef<typeof DropdownMenuPrimitive.Item>,
  DropdownMenuItemProps
>(function DropdownMenuItem({ asChild, children, className, icon, ...props }, ref) {
  if (asChild) {
    return (
      <DropdownMenuPrimitive.Item
        ref={ref}
        asChild
        className={cn("aoi-dropdown-menu-item", className)}
        {...props}
      >
        {children}
      </DropdownMenuPrimitive.Item>
    );
  }

  return (
    <DropdownMenuPrimitive.Item
      ref={ref}
      className={cn("aoi-dropdown-menu-item", className)}
      {...props}
    >
      {icon ? <span className="aoi-dropdown-menu-item__icon">{icon}</span> : null}
      <span>{children}</span>
    </DropdownMenuPrimitive.Item>
  );
});

type DropdownMenuRadioItemProps = ComponentPropsWithoutRef<
  typeof DropdownMenuPrimitive.RadioItem
> & {
  icon?: ReactNode;
};

export const DropdownMenuRadioItem = forwardRef<
  ElementRef<typeof DropdownMenuPrimitive.RadioItem>,
  DropdownMenuRadioItemProps
>(function DropdownMenuRadioItem({ children, className, icon, ...props }, ref) {
  return (
    <DropdownMenuPrimitive.RadioItem
      ref={ref}
      className={cn("aoi-dropdown-menu-item", "aoi-dropdown-menu-radio-item", className)}
      {...props}
    >
      <span className="aoi-dropdown-menu-item__indicator">
        <DropdownMenuPrimitive.ItemIndicator>
          <Check aria-hidden="true" size={14} />
        </DropdownMenuPrimitive.ItemIndicator>
      </span>
      {icon ? <span className="aoi-dropdown-menu-item__icon">{icon}</span> : null}
      <span>{children}</span>
    </DropdownMenuPrimitive.RadioItem>
  );
});
