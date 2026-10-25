import assert from "node:assert/strict";
import { test } from "node:test";

import { matchesSearchExpression, parseSearchExpression } from "../src/utils/searchExpression.ts";

test("empty query matches everything", () => {
    assert.equal(matchesSearchExpression("sodium", ""), true);
    assert.equal(matchesSearchExpression("sodium", "   "), true);
});

test("plain substring query is case-insensitive", () => {
    assert.equal(matchesSearchExpression("Sodium Extra", "sodium"), true);
    assert.equal(matchesSearchExpression("Sodium Extra", "iris"), false);
});

test("OR branches match when any branch is satisfied", () => {
    assert.equal(matchesSearchExpression("sodium", "sodium | iris"), true);
    assert.equal(matchesSearchExpression("iris", "sodium | iris"), true);
    assert.equal(matchesSearchExpression("lithium", "sodium | iris"), false);
});

test("AND requires every term inside a branch", () => {
    assert.equal(matchesSearchExpression("fabric api", "fabric & api"), true);
    assert.equal(matchesSearchExpression("fabric loader", "fabric & api"), false);
});

test("AND binds tighter than OR", () => {
    // cloth & config | yacl  ==  (cloth AND config) OR yacl
    assert.equal(matchesSearchExpression("cloth config", "cloth & config | yacl"), true);
    assert.equal(matchesSearchExpression("yacl", "cloth & config | yacl"), true);
    assert.equal(matchesSearchExpression("cloth", "cloth & config | yacl"), false);
    assert.equal(matchesSearchExpression("config", "cloth & config | yacl"), false);
});

test("operators tolerate surrounding whitespace and empty segments", () => {
    assert.equal(matchesSearchExpression("sodium", "| sodium |"), true);
    assert.equal(matchesSearchExpression("sodium extra", " sodium  &  extra "), true);
    assert.equal(matchesSearchExpression("sodium", "||"), true);
    assert.equal(matchesSearchExpression("sodium", "&|&"), true);
});

test("parseSearchExpression splits branches and terms", () => {
    assert.deepEqual(parseSearchExpression("fabric & api | sodium"), {
        branches: [["fabric", "api"], ["sodium"]],
    });
    assert.deepEqual(parseSearchExpression(""), { branches: [] });
});
