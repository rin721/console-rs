import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";

import { SkeletonText } from "~/components/aoi/primitives/Skeleton";
import { Collapse } from "./Collapse";
import { Dialog } from "./Dialog";
import { Drawer } from "./Drawer";
import { Popover } from "./Popover";

describe("Aoi interaction patterns", () => {
  it("renders skeleton text placeholders without exposing loading copy", () => {
    const { container } = render(<SkeletonText lines={3} />);

    expect(container.querySelectorAll(".aoi-skeleton-text__line")).toHaveLength(3);
  });

  it("toggles collapse content with aria-expanded", () => {
    render(
      <Collapse defaultOpen={false} title="Filters">
        <p>Filter form</p>
      </Collapse>,
    );

    const trigger = screen.getByRole("button", { name: /filters/i });
    expect(trigger).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByText("Filter form")).not.toBeInTheDocument();

    fireEvent.click(trigger);

    expect(trigger).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByText("Filter form")).toBeInTheDocument();
  });

  it("renders dialog content and calls close handlers", () => {
    const onOpenChange = vi.fn();

    render(
      <Dialog
        closeLabel="Close"
        description="This cannot be undone."
        footer={<button type="button">Confirm</button>}
        open
        title="Confirm delete"
        onOpenChange={onOpenChange}
      />,
    );

    expect(screen.getByRole("dialog", { name: "Confirm delete" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Close" }));
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it("renders drawer content as a dialog surface", () => {
    render(
      <Drawer closeLabel="Close" open title="Edit item" onOpenChange={vi.fn()}>
        <p>Drawer form</p>
      </Drawer>,
    );

    expect(screen.getByRole("dialog", { name: "Edit item" })).toBeInTheDocument();
    expect(screen.getByText("Drawer form")).toBeInTheDocument();
  });

  it("returns focus to an external dialog trigger after close", async () => {
    function DialogHarness() {
      const [open, setOpen] = useState(false);

      return (
        <>
          <button type="button" onClick={() => setOpen(true)}>
            Open dialog
          </button>
          <Dialog closeLabel="Close" open={open} title="External dialog" onOpenChange={setOpen}>
            <p>Dialog body</p>
          </Dialog>
        </>
      );
    }

    render(<DialogHarness />);

    const trigger = screen.getByRole("button", { name: "Open dialog" });
    trigger.focus();
    fireEvent.click(trigger);
    fireEvent.click(screen.getByRole("button", { name: "Close" }));

    await waitFor(() => expect(trigger).toHaveFocus());
  });

  it("opens popover content from its trigger", () => {
    render(
      <Popover ariaLabel="Show details" title="Details">
        <p>Additional context</p>
      </Popover>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Show details" }));

    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByText("Additional context")).toBeInTheDocument();
  });
});
