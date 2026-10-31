// Builds the searchable text for local mods.
//
// Only human-facing fields are indexed. Paths and dependency declarations are
// deliberately excluded: they embed the Minecraft/loader version and used to
// make every mod match a version query. Chinese names additionally contribute
// pinyin initials so "nlz" can find "钠离子".

import { pinyin } from "pinyin-pro";

const CJK_PATTERN = /[\u3400-\u4dbf\u4e00-\u9fff\uf900-\ufaff]/;

export function pinyinInitialsOf(value: string): string {
    const text = (value || "").trim();
    if (!text || !CJK_PATTERN.test(text)) return "";
    return pinyin(text, {
        pattern: "first",
        toneType: "none",
        type: "array",
        nonZh: "removed",
    }).join("");
}

type LocalModLike = {
    id?: string;
    name?: string;
    onlineName?: string;
    version?: string;
    fileName?: string;
    description?: string;
    categories?: string[] | null;
    jijMods?: Array<{ id?: string; name?: string }> | null;
};

export function buildModSearchText(mods: LocalModLike[]): string {
    const values: string[] = [];
    const push = (value: unknown, withPinyin = false) => {
        if (value === null || value === undefined) return;
        if (typeof value !== "string" && typeof value !== "number") return;
        const text = String(value).trim();
        if (!text) return;
        values.push(text);
        if (withPinyin) {
            const initials = pinyinInitialsOf(text);
            if (initials) values.push(initials);
        }
    };

    for (const mod of mods || []) {
        push(mod?.id);
        push(mod?.name, true);
        push(mod?.onlineName, true);
        push(mod?.version);
        push(mod?.fileName);
        push(mod?.description);
        for (const category of mod?.categories || []) {
            push(category, true);
        }
        for (const jij of mod?.jijMods || []) {
            push(jij?.id);
            push(jij?.name, true);
        }
    }
    return values.join("\n").toLocaleLowerCase();
}
