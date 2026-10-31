// Lightweight search expression parser for local mod filtering.
//
// Grammar:
//   expression := branch ( '|' branch )*
//   branch     := term ( '&' term )*
//
// `&` (AND) binds tighter than `|` (OR): the query is split into OR branches on
// `|`, and every `&`-separated term inside a branch must be present. Operators
// tolerate surrounding whitespace and repeated symbols; empty terms are dropped.

const OR_SEPARATOR = "|";
const AND_SEPARATOR = "&";

export type SearchExpression = {
    /** Each branch is an AND-group of required terms (already lower-cased). */
    branches: string[][];
};

/** Collapse redundant whitespace around a term. */
const normalizeTerm = (term: string) => term.trim().toLocaleLowerCase();

/**
 * Parse a raw query into OR branches of AND terms. A plain query with no
 * operators yields a single branch (so behavior matches substring search for
 * the common case, while still treating spaces as part of the term).
 */
export function parseSearchExpression(query: string): SearchExpression {
    const raw = (query || "").trim();
    if (!raw) {
        return { branches: [] };
    }
    const branches: string[][] = [];
    for (const branchText of raw.split(OR_SEPARATOR)) {
        const terms = branchText
            .split(AND_SEPARATOR)
            .map(normalizeTerm)
            .filter(Boolean);
        if (terms.length > 0) {
            branches.push(terms);
        }
    }
    return { branches };
}

/**
 * Return true when `text` satisfies the query. Empty queries match everything.
 * A query of only operators (e.g. `|&`) also matches everything rather than
 * silently hiding all rows.
 */
export function matchesSearchExpression(text: string, query: string): boolean {
    const normalizedText = (text || "").toLocaleLowerCase();
    const { branches } = parseSearchExpression(query);
    if (branches.length === 0) {
        return true;
    }
    return branches.some((terms) => terms.every((term) => normalizedText.includes(term)));
}
