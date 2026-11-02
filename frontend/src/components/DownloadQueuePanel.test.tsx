import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { DownloadQueuePanel } from "@/components/DownloadQueuePanel";
import { TooltipProvider } from "@/components/ui/tooltip";
import { useDownloadQueueStore } from "@/stores/downloadQueue";
import type { DownloadQueueItem, DownloadQueueState, OptionalDependencyReminder } from "@/types";

const pendingItem: DownloadQueueItem = {
  id: "item-pending",
  projectId: "sodium",
  status: "pending",
  title: "Sodium",
  fileName: "sodium.jar",
  versionId: "v1",
  platform: "modrinth",
  minecraftVersion: "1.20.1",
  modLoader: "fabric",
  cancelable: true,
  retryable: false,
  bytesComplete: 10,
  totalBytes: 100,
};

const failedItem: DownloadQueueItem = {
  ...pendingItem,
  id: "item-failed",
  title: "Iris",
  fileName: "iris.jar",
  status: "failed",
  cancelable: false,
  retryable: true,
};

const reminder: OptionalDependencyReminder = {
  id: "reminder-1",
  mainProjectKey: "modrinth:iris",
  mainTitle: "Iris",
  mainVersionId: "v2",
  minecraftVersion: "1.20.1",
  modLoader: "fabric",
  dependencies: [],
};

function setQueue(queue: Partial<DownloadQueueState>) {
  useDownloadQueueStore.setState({
    queue: {
      active: true,
      pending: 0,
      running: 0,
      messageCount: 0,
      bytesComplete: 0,
      totalBytes: 0,
      bytesPerSecond: 0,
      items: [],
      optionalReminders: [],
      ...queue,
    },
  });
}

function renderPanel() {
  return render(
    <TooltipProvider>
      <DownloadQueuePanel />
    </TooltipProvider>,
  );
}

describe("DownloadQueuePanel", () => {
  beforeEach(() => {
    useDownloadQueueStore.setState({ queue: { active: false, pending: 0, running: 0, messageCount: 0, bytesComplete: 0, totalBytes: 0, bytesPerSecond: 0, items: [], optionalReminders: [] } });
  });

  it("renders nothing when the queue is empty", () => {
    const { container } = renderPanel();
    expect(container).toBeEmptyDOMElement();
  });

  it("opens the panel and switches between download and optional tabs", async () => {
    const user = userEvent.setup();
    setQueue({ pending: 1, items: [pendingItem], messageCount: 1, optionalReminders: [reminder] });
    renderPanel();

    await user.click(screen.getByRole("button", { name: "Download Queue" }));
    expect(screen.getByRole("tab", { name: "Downloads" })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByText("Sodium")).toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: /Optional/ }));
    expect(screen.getByRole("tab", { name: /Optional/ })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByText("Iris")).toBeInTheDocument();
    expect(screen.queryByText("Sodium")).not.toBeInTheDocument();
  });

  it("auto-opens the optional tab when only reminders remain", async () => {
    const user = userEvent.setup();
    setQueue({ messageCount: 1, optionalReminders: [reminder] });
    renderPanel();

    await user.click(screen.getByRole("button", { name: "Download Queue" }));
    expect(screen.getByRole("tab", { name: /Optional/ })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByText("Iris")).toBeInTheDocument();
  });

  it("plays the row exit before removing a failed task", async () => {
    const user = userEvent.setup();
    const remove = vi.fn().mockResolvedValue(undefined);
    useDownloadQueueStore.setState({ remove });
    setQueue({ pending: 1, items: [failedItem] });
    renderPanel();

    await user.click(screen.getByRole("button", { name: "Download Queue" }));
    const row = document.querySelector('[data-queue-item="item-failed"]');
    expect(row).not.toBeNull();

    await user.click(screen.getByRole("button", { name: "Remove queue task" }));
    // The row starts collapsing immediately; the store call waits for it.
    expect(row).toHaveClass("opacity-0");
    expect(remove).not.toHaveBeenCalled();

    await waitFor(() => expect(remove).toHaveBeenCalledWith("item-failed"));
  });
});
