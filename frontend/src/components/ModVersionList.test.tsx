import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { ModVersionList } from "@/components/ModVersionList";
import { TooltipProvider } from "@/components/ui/tooltip";
import type { ModVersion } from "@/types";

const versions: ModVersion[] = [
  {
    id: "v1",
    versionId: "v1",
    platform: "Modrinth",
    projectId: "sodium",
    name: "Sodium 0.6.0",
    version: "0.6.0",
    releaseType: "release",
    fileName: "sodium-0.6.0.jar",
    fileSize: 100,
    downloadUrl: "https://example.invalid/sodium.jar",
    sha1: "",
    publishedAt: 0,
    downloads: 0,
    gameVersions: [],
    loaders: [],
  },
];

describe("ModVersionList", () => {
  it("renders versions and reports selection", async () => {
    const user = userEvent.setup();
    const onSelect = vi.fn();
    render(
      <TooltipProvider>
        <ModVersionList versions={versions} installedVersionId="v1" onSelect={onSelect} />
      </TooltipProvider>,
    );

    expect(screen.getByText("sodium-0.6.0.jar")).toBeInTheDocument();
    expect(screen.getByText("Installed")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Replace with this version" }));
    expect(onSelect).toHaveBeenCalledWith(versions[0]);
  });

  it("shows the empty message", () => {
    render(<ModVersionList versions={[]} onSelect={() => undefined} />);
    expect(screen.getByText("No versions match the current Minecraft version and Mod Loader.")).toBeInTheDocument();
  });
});
