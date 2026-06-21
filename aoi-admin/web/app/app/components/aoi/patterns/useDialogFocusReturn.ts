import { useRef } from "react";

type AutoFocusEvent = Event & {
  preventDefault: () => void;
};

export function useDialogFocusReturn() {
  const triggerRef = useRef<HTMLElement | null>(null);

  return {
    onCloseAutoFocus: (event: AutoFocusEvent) => {
      const trigger = triggerRef.current;
      triggerRef.current = null;

      if (!trigger?.isConnected) {
        return;
      }

      event.preventDefault();
      window.requestAnimationFrame(() => {
        trigger.focus({ preventScroll: true });
      });
    },
    onOpenAutoFocus: () => {
      const activeElement = document.activeElement;

      if (
        activeElement instanceof HTMLElement &&
        activeElement !== document.body &&
        activeElement !== document.documentElement
      ) {
        triggerRef.current = activeElement;
      }
    },
  };
}
