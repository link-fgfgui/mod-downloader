import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/services/app", () => ({
  api: {
    ListFavoriteLists: vi.fn().mockResolvedValue([]),
    ListFavoriteContentsForScope: vi.fn().mockResolvedValue([]),
    GetSettings: vi.fn(),
  },
}));

import { TooltipProvider } from "@/components/ui/tooltip";
import { FavoritesPage } from "@/pages/Favorites";
import { useFavoritesStore } from "@/stores/favorites";
import { useMinecraftStore } from "@/stores/minecraft";
import type { FavoriteList, FavoriteModEntry } from "@/types";

const list = {
  id: "list-1",
  name: "Favorites",
  minecraftVersion: "1.20.1",
  modLoader: "fabric",
  createdAt: 0,
  updatedAt: 0,
  sortOrder: 0,
} as FavoriteList;

const plainItem = {
  id: "entry-1",
  listId: "list-1",
  platform: "modrinth",
  modId: "sodium",
  title: "Sodium",
  createdAt: 0,
  updatedAt: 0,
} as FavoriteModEntry;

const referencedItem = {
  id: "entry-2",
  listId: "list-1",
  platform: "modrinth",
  modId: "iris",
  title: "Iris",
  referenced: true,
  sourceListId: "list-2",
  sourceListName: "Shared",
  createdAt: 0,
  updatedAt: 0,
} as FavoriteModEntry;

describe("FavoritesPage selection guard", () => {
  beforeEach(() => {
    useFavoritesStore.setState({
      lists: [list],
      selectedListId: "list-1",
      items: [plainItem, referencedItem],
      overview: { pinned: [], ungrouped: [list] },
      contents: { refs: [] } as never,
      displayMinecraftVersion: "1.20.1",
      displayModLoader: "fabric",
      loadingLists: false,
      loadingItems: false,
      exportingPackwiz: false,
      error: "",
      loadLists: vi.fn().mockResolvedValue(undefined),
      selectList: vi.fn().mockResolvedValue(undefined),
      setDisplayScope: vi.fn(),
    });
    useMinecraftStore.setState({ minecraftVersion: "1.20.1", modLoader: "fabric" });
  });

  it("selects plain rows but never reference rows", async () => {
    const user = userEvent.setup();
    render(
      <MemoryRouter>
        <TooltipProvider>
          <FavoritesPage />
        </TooltipProvider>
      </MemoryRouter>,
    );

    await user.click(await screen.findByText("Sodium"));
    expect(screen.getByLabelText("1 selected")).toBeInTheDocument();

    // Referenced entries are read-only projections of another list: clicking
    // them must not extend the batch selection.
    await user.click(screen.getByText("Iris"));
    expect(screen.getByLabelText("1 selected")).toBeInTheDocument();
    expect(screen.getByText("Iris").closest(".md-row-interactive")).toHaveClass("border-l-cyan-500");
  });
});
