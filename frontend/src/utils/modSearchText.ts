// GUI-only pinyin initials for local mod search.
//
// Core owns the canonical search text (`LocalModGroup.SearchText` /
// `modgraph.Node.SearchText`). The GUI adds pinyin initials on top so CJK names
// stay searchable by their initials ("nlz" -> "钠离子").

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
