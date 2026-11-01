// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

// Package searchexpr parses the local-mod search grammar shared by the GUI and
// TUI.
//
//	expression := branch ( '|' branch )*
//	branch     := term ( '&' term )*
//
// `&` (AND) binds tighter than `|` (OR). Operators tolerate surrounding
// whitespace and repeated symbols; empty terms are dropped. A query with no
// terms matches everything, so a query of only operators never hides rows.
package searchexpr

import "strings"

// Parse splits a raw query into OR branches of required AND terms. Terms are
// lower-cased and trimmed.
func Parse(query string) [][]string {
	raw := strings.TrimSpace(query)
	if raw == "" {
		return nil
	}
	branches := make([][]string, 0, 2)
	for _, branchText := range strings.Split(raw, "|") {
		terms := make([]string, 0, 2)
		for _, term := range strings.Split(branchText, "&") {
			term = strings.ToLower(strings.TrimSpace(term))
			if term != "" {
				terms = append(terms, term)
			}
		}
		if len(terms) > 0 {
			branches = append(branches, terms)
		}
	}
	return branches
}

// Matches reports whether text satisfies query. Empty queries match
// everything.
func Matches(text, query string) bool {
	branches := Parse(query)
	if len(branches) == 0 {
		return true
	}
	normalized := strings.ToLower(text)
	for _, terms := range branches {
		all := true
		for _, term := range terms {
			if !strings.Contains(normalized, term) {
				all = false
				break
			}
		}
		if all {
			return true
		}
	}
	return false
}
