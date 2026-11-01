// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package searchexpr

import (
	"reflect"
	"testing"
)

func TestParseAndMatch(t *testing.T) {
	if got := Parse("sodium | iris & fabric"); !reflect.DeepEqual(got, [][]string{{"sodium"}, {"iris", "fabric"}}) {
		t.Fatalf("Parse() = %#v", got)
	}
	if !Matches("sodium fabric api", "iris | sodium & fabric") {
		t.Fatal("AND should bind tighter than OR")
	}
	if Matches("sodium fabric api", "iris | sodium & forge") {
		t.Fatal("missing AND term should not match")
	}
	if !Matches("Sodium Extra", "  SODIUM  ") {
		t.Fatal("matching should be case-insensitive and trimmed")
	}
	if !Matches("Anything", "") || !Matches("Anything", "|&") {
		t.Fatal("empty/operator-only queries must match everything")
	}
}
