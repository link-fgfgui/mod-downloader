import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";

import { Combobox } from "@/components/ui/combobox";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";

const fruitOptions = [
  { value: "apple", label: "Apple" },
  { value: "banana", label: "Banana" },
  { value: "cherry", label: "Cherry" },
];

describe("Combobox", () => {
  it("filters options through the search field", async () => {
    const user = userEvent.setup();
    render(<Combobox value="apple" onValueChange={() => undefined} options={fruitOptions} searchable aria-label="fruit" />);

    await user.click(screen.getByRole("combobox", { name: "fruit" }));
    expect(await screen.findByRole("option", { name: "Banana" })).toBeInTheDocument();

    await user.type(screen.getByRole("textbox"), "ban");

    await waitFor(() => {
      expect(screen.queryByRole("option", { name: "Apple" })).not.toBeInTheDocument();
    });
    expect(screen.getByRole("option", { name: "Banana" })).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: "Cherry" })).not.toBeInTheDocument();
  });

  it("commits typed text as the value in allowCustom mode", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<Combobox value="" onValueChange={onChange} options={["sodium"]} allowCustom aria-label="mod name" />);

    await user.type(screen.getByRole("combobox", { name: "mod name" }), "my-mod");

    expect(onChange).toHaveBeenLastCalledWith("my-mod");
    expect(await screen.findByRole("option", { name: "my-mod" })).toBeInTheDocument();
  });

  it("moves the highlight with arrow keys and commits with Enter", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<Combobox value="apple" onValueChange={onChange} options={fruitOptions} aria-label="fruit" />);

    await user.click(screen.getByRole("combobox", { name: "fruit" }));
    // The selected option starts active; one ArrowDown lands on Banana.
    await user.keyboard("{ArrowDown}");
    await user.keyboard("{Enter}");

    expect(onChange).toHaveBeenCalledWith("banana");
    expect(screen.getByRole("combobox", { name: "fruit" })).toHaveAttribute("aria-expanded", "false");
  });

  it("closes only the panel on Escape while a dialog stays open", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    function Harness() {
      const [value, setValue] = useState("apple");
      return (
        <Dialog open onOpenChange={onOpenChange}>
          <DialogContent>
            <DialogTitle>Pick one</DialogTitle>
            <Combobox value={value} onValueChange={setValue} options={fruitOptions} aria-label="fruit" />
          </DialogContent>
        </Dialog>
      );
    }
    render(<Harness />);

    await user.click(await screen.findByRole("combobox", { name: "fruit" }));
    expect(await screen.findByRole("option", { name: "Banana" })).toBeInTheDocument();

    // First Escape belongs to the open panel: the dialog must not react.
    await user.keyboard("{Escape}");
    await waitFor(() => {
      expect(screen.getByRole("combobox", { name: "fruit" })).toHaveAttribute("aria-expanded", "false");
    });
    expect(onOpenChange).not.toHaveBeenCalled();

    // The panel is closed now, so the next Escape reaches the dialog.
    await user.keyboard("{Escape}");
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});
