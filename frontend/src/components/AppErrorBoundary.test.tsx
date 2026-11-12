import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { AppErrorBoundary } from "@/components/AppErrorBoundary";

function ThrowingChild(): null {
  throw new Error("render exploded");
}

// React reports caught render errors as uncaught window errors in jsdom; the
// boundary handles them, so silence the reporter for this file only.
const silenceWindowError = (event: ErrorEvent) => event.preventDefault();

describe("AppErrorBoundary", () => {
  beforeEach(() => {
    vi.spyOn(console, "error").mockImplementation(() => undefined);
    window.addEventListener("error", silenceWindowError);
  });

  afterEach(() => {
    window.removeEventListener("error", silenceWindowError);
    vi.restoreAllMocks();
  });

  it("shows a recovery screen instead of unmounting the tree", () => {
    render(
      <AppErrorBoundary>
        <ThrowingChild />
      </AppErrorBoundary>,
    );

    expect(screen.getByText("Something went wrong")).toBeInTheDocument();
    expect(screen.getByText("render exploded")).toBeInTheDocument();
  });

  it("reloads the application from the fallback", async () => {
    const reload = vi.fn();
    const user = userEvent.setup();
    render(
      <AppErrorBoundary onReload={reload}>
        <ThrowingChild />
      </AppErrorBoundary>,
    );

    await user.click(screen.getByRole("button", { name: "Reload application" }));
    expect(reload).toHaveBeenCalledTimes(1);
  });

  it("renders children when no error is thrown", () => {
    render(
      <AppErrorBoundary>
        <div>All good</div>
      </AppErrorBoundary>,
    );

    expect(screen.getByText("All good")).toBeInTheDocument();
  });
});
