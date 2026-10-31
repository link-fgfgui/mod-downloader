import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { StartupFailure } from "@/components/StartupFailure";

vi.mock("@/services/app", () => ({
  api: {
    Quit: vi.fn().mockResolvedValue(undefined),
  },
}));

import { api } from "@/services/app";

describe("StartupFailure", () => {
  it("shows the failed stage and error details", () => {
    render(<StartupFailure status={{ ready: false, stage: "storage", error: "disk unavailable" }} />);
    expect(screen.getByText("Core service failed to start")).toBeInTheDocument();
    expect(screen.getByText("Storage and cache")).toBeInTheDocument();
    expect(screen.getByText("disk unavailable")).toBeInTheDocument();
  });

  it("quits the application", async () => {
    const user = userEvent.setup();
    render(<StartupFailure status={{ ready: false, stage: "config" }} />);
    await user.click(screen.getByRole("button", { name: "Quit application" }));
    expect(api.Quit).toHaveBeenCalledTimes(1);
  });
});
