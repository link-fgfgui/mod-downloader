import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";

import {
  MANAGE_VIEW_MODE_STORAGE_KEY,
  useManageViewMode,
} from "@/hooks/useManageViewMode";

describe("useManageViewMode", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("defaults to the list view", () => {
    const { result } = renderHook(() => useManageViewMode());
    expect(result.current[0]).toBe("list");
  });

  it("persists the selected view across remounts", () => {
    const first = renderHook(() => useManageViewMode());
    act(() => first.result.current[1]("tree"));
    expect(window.localStorage.getItem(MANAGE_VIEW_MODE_STORAGE_KEY)).toBe("tree");
    first.unmount();

    const second = renderHook(() => useManageViewMode());
    expect(second.result.current[0]).toBe("tree");
  });

  it("ignores unknown stored values", () => {
    window.localStorage.setItem(MANAGE_VIEW_MODE_STORAGE_KEY, "grid");
    const { result } = renderHook(() => useManageViewMode());
    expect(result.current[0]).toBe("list");
  });
});
