import { describe, expect, it } from "vitest";

import { matchesSearchExpression, parseSearchExpression } from "@/utils/searchExpression";

describe("searchExpression", () => {
  it("matches everything for an empty query", () => {
    expect(matchesSearchExpression("Anything", "")).toBe(true);
    expect(matchesSearchExpression("Anything", "|&")).toBe(true);
  });

  it("parses OR branches of AND terms", () => {
    expect(parseSearchExpression("sodium | iris & fabric")).toEqual({
      branches: [["sodium"], ["iris", "fabric"]],
    });
  });

  it("binds AND tighter than OR", () => {
    const text = "sodium fabric api";
    expect(matchesSearchExpression(text, "iris | sodium & fabric")).toBe(true);
    expect(matchesSearchExpression(text, "iris | sodium & forge")).toBe(false);
  });

  it("is case-insensitive and trims terms", () => {
    expect(matchesSearchExpression("Sodium Extra", "  SODIUM  ")).toBe(true);
  });
});
