import assert from "node:assert/strict";
import { test } from "node:test";

import { buildModSearchText, pinyinInitialsOf } from "../src/utils/modSearchText.ts";

test("indexes whitelisted fields and categories", () => {
    const text = buildModSearchText([{
        id: "sodium",
        name: "Sodium",
        onlineName: "Sodium Extra",
        version: "0.5.8",
        fileName: "sodium-fabric-0.5.8.jar",
        description: "Rendering optimizer",
        categories: ["Optimization", "Client"],
    }]);
    for (const expected of ["sodium", "sodium extra", "0.5.8", "sodium-fabric-0.5.8.jar", "optimization", "client"]) {
        assert.equal(text.includes(expected), true, expected);
    }
});

test("never indexes paths or dependency declarations", () => {
    const mod: any = {
        id: "example",
        fileName: "example.jar",
        path: "/instances/1.20.1/mods/example.jar",
        dependencies: [{ modId: "minecraft", version: "1.20.1", type: "required" }],
    };
    const text = buildModSearchText([mod]);
    assert.equal(text.includes("1.20.1"), false);
    assert.equal(text.includes("/instances"), false);
    assert.equal(text.includes("minecraft"), false);
});

test("adds pinyin initials for Chinese names and categories", () => {
    assert.equal(pinyinInitialsOf("钠离子"), "nlz");
    assert.equal(pinyinInitialsOf("Sodium"), "");
    const text = buildModSearchText([{
        name: "钠离子",
        categories: ["优化"],
        jijMods: [{ id: "lib", name: "前置库" }],
    }]);
    for (const expected of ["钠离子", "nlz", "优化", "yh", "qzk"]) {
        assert.equal(text.includes(expected), true, expected);
    }
});
