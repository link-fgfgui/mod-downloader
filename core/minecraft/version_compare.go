// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package minecraft

import (
	"regexp"
	"strconv"
	"strings"
)

// versionToken finds the first version-looking token in a free-form mod
// version/display name, so a leading "sodium-fabric-" cannot dominate ordering.
var versionToken = regexp.MustCompile(`(?i)v?[0-9]+(?:[._-][0-9]+)*`)

type versionSegment struct {
	number   int64
	text     string
	isNumber bool
}

func versionBody(value string) string {
	text := strings.ToLower(strings.TrimSpace(value))
	loc := versionToken.FindStringIndex(text)
	if loc == nil {
		return text
	}
	body := text[loc[0]:]
	if len(body) > 1 && body[0] == 'v' && body[1] >= '0' && body[1] <= '9' {
		body = body[1:]
	}
	return body
}

func versionSegments(value string) []versionSegment {
	parts := strings.FieldsFunc(versionBody(value), func(r rune) bool {
		return !(r >= '0' && r <= '9') && !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z')
	})
	out := make([]versionSegment, 0, len(parts))
	for _, part := range parts {
		if n, err := strconv.ParseInt(part, 10, 64); err == nil {
			out = append(out, versionSegment{number: n, isNumber: true})
			continue
		}
		out = append(out, versionSegment{text: part})
	}
	return out
}

// CompareVersions returns -1, 0, or 1 for a < b, a == b, a > b. Segments of
// different kinds (text vs number) are treated as incomparable and return 0
// instead of inventing a lexicographic order.
func CompareVersions(a, b string) int {
	left := versionSegments(a)
	right := versionSegments(b)
	length := len(left)
	if len(right) > length {
		length = len(right)
	}
	for i := 0; i < length; i++ {
		if i >= len(left) {
			return -1
		}
		if i >= len(right) {
			return 1
		}
		l, r := left[i], right[i]
		if l.isNumber != r.isNumber {
			return 0
		}
		if l.isNumber {
			if l.number != r.number {
				if l.number < r.number {
					return -1
				}
				return 1
			}
			continue
		}
		if l.text != r.text {
			if l.text < r.text {
				return -1
			}
			return 1
		}
	}
	return 0
}

// IsVersionDowngrade reports whether candidate is strictly older than
// installed. Empty inputs are never downgrades.
func IsVersionDowngrade(candidate, installed string) bool {
	if strings.TrimSpace(candidate) == "" || strings.TrimSpace(installed) == "" {
		return false
	}
	return CompareVersions(candidate, installed) < 0
}

// VersionMajor returns the first numeric segment, used as the major version.
func VersionMajor(value string) (int64, bool) {
	for _, segment := range versionSegments(value) {
		if segment.isNumber {
			return segment.number, true
		}
	}
	return 0, false
}

// HasMajorVersionChange reports whether both versions expose a numeric major
// and they differ.
func HasMajorVersionChange(candidate, installed string) bool {
	left, leftOK := VersionMajor(candidate)
	right, rightOK := VersionMajor(installed)
	if !leftOK || !rightOK {
		return false
	}
	return left != right
}
