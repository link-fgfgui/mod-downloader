// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package minecraft

import (
	"fmt"
	"strings"
)

// VersionRangeAllows reports whether a declared dependency version range
// accepts the given version. It supports the common Maven-style and prefix
// operators used by mod metadata, and treats an empty or wildcard range as
// "any". Compound ranges may be separated by commas or whitespace (for example
// Modrinth's ">=1.21- <1.21.2-"); every clause must accept the version.
//
// The second result reports whether the range was recognized; callers use it to
// distinguish "unconstrained" from "unsupported".
func VersionRangeAllows(declared, version string) (bool, bool) {
	declared = strings.TrimSpace(declared)
	if declared == "" || declared == "*" {
		return true, false
	}
	version = strings.TrimSpace(version)
	if version == "" {
		return true, false
	}
	if strings.HasPrefix(declared, "[") || strings.HasPrefix(declared, "(") {
		return mavenRangeAllows(declared, version), true
	}

	parts := splitRangeClauses(declared)
	if len(parts) == 0 {
		return false, false
	}
	allowed := true
	for _, part := range parts {
		ok, recognized := singleRangeAllows(part, version)
		if !recognized {
			return false, false
		}
		if !ok {
			allowed = false
		}
	}
	return allowed, true
}

// splitRangeClauses splits a compound range on commas and whitespace.
func splitRangeClauses(declared string) []string {
	fields := strings.FieldsFunc(declared, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		if field = strings.TrimSpace(field); field != "" {
			parts = append(parts, field)
		}
	}
	return parts
}

func singleRangeAllows(part, version string) (bool, bool) {
	switch {
	case strings.HasPrefix(part, ">="):
		return compareVersions(version, strings.TrimSpace(part[2:])) >= 0, true
	case strings.HasPrefix(part, "<="):
		return compareVersions(version, strings.TrimSpace(part[2:])) <= 0, true
	case strings.HasPrefix(part, ">"):
		return compareVersions(version, strings.TrimSpace(part[1:])) > 0, true
	case strings.HasPrefix(part, "<"):
		return compareVersions(version, strings.TrimSpace(part[1:])) < 0, true
	case strings.HasPrefix(part, "=="):
		return compareVersions(version, strings.TrimSpace(part[2:])) == 0, true
	case strings.HasPrefix(part, "~"):
		return tildeAllows(strings.TrimSpace(part[1:]), version), true
	case strings.HasPrefix(part, "^"):
		return caretAllows(strings.TrimSpace(part[1:]), version), true
	case strings.HasPrefix(part, "["):
		return mavenRangeAllows(part, version), true
	case strings.HasPrefix(part, "("):
		return mavenRangeAllows(part, version), true
	default:
		if idx := strings.Index(part, ".x"); idx >= 0 {
			prefix := strings.TrimSuffix(part[:idx], ".")
			return strings.HasPrefix(version, prefix+".") || version == prefix, true
		}
		return compareVersions(version, part) == 0, true
	}
}

// tildeAllows implements "~1.21.1" => >=1.21.1 <1.22.0 (patch-level flexibility).
func tildeAllows(base, version string) bool {
	baseParts := splitVersion(base)
	versionParts := splitVersion(version)
	if len(baseParts) == 0 || len(versionParts) == 0 {
		return false
	}
	if len(baseParts) == 1 {
		return versionParts[0] == baseParts[0]
	}
	if compareVersions(version, base) < 0 {
		return false
	}
	upper := make([]int, len(baseParts))
	copy(upper, baseParts)
	upper[len(baseParts)-2]++
	for i := len(baseParts) - 1; i > len(baseParts)-2; i-- {
		upper[i] = 0
	}
	return compareVersionParts(versionParts, upper) < 0
}

// caretAllows implements "^1.21.1" => >=1.21.1 <2.0.0 (major-level flexibility).
func caretAllows(base, version string) bool {
	baseParts := splitVersion(base)
	versionParts := splitVersion(version)
	if len(baseParts) == 0 || len(versionParts) == 0 {
		return false
	}
	if compareVersions(version, base) < 0 {
		return false
	}
	upper := make([]int, len(baseParts))
	copy(upper, baseParts)
	upper[0]++
	for i := 1; i < len(upper); i++ {
		upper[i] = 0
	}
	return compareVersionParts(versionParts, upper) < 0
}

func mavenRangeAllows(part, version string) bool {
	part = strings.TrimSpace(part)
	trimmed := strings.Trim(part, "[]()")
	parts := strings.SplitN(trimmed, ",", 2)
	if len(parts) != 2 {
		return false
	}
	low := strings.TrimSpace(parts[0])
	high := strings.TrimSpace(parts[1])
	lowInclusive := strings.HasPrefix(part, "[")
	highInclusive := strings.HasSuffix(part, "]")
	cmpLow := compareVersions(version, low)
	cmpHigh := compareVersions(version, high)
	if low != "" {
		if cmpLow < 0 || (cmpLow == 0 && !lowInclusive) {
			return false
		}
	}
	if high != "" {
		if cmpHigh > 0 || (cmpHigh == 0 && !highInclusive) {
			return false
		}
	}
	return true
}

func compareVersions(left, right string) int {
	return compareVersionParts(splitVersion(left), splitVersion(right))
}

func compareVersionParts(left, right []int) int {
	size := len(left)
	if len(right) > size {
		size = len(right)
	}
	for i := 0; i < size; i++ {
		var l, r int
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		if l != r {
			if l < r {
				return -1
			}
			return 1
		}
	}
	return 0
}

func splitVersion(version string) []int {
	version = strings.TrimSpace(version)
	version = strings.TrimPrefix(version, "v")
	version = strings.TrimPrefix(version, "^")
	version = strings.TrimPrefix(version, "~")
	parts := make([]int, 0, 4)
	for _, raw := range strings.Split(version, ".") {
		digits := strings.Builder{}
		for _, ch := range raw {
			if ch < '0' || ch > '9' {
				break
			}
			digits.WriteRune(ch)
		}
		if digits.Len() == 0 {
			parts = append(parts, 0)
			continue
		}
		value, err := parseIntSafe(digits.String())
		if err != nil {
			parts = append(parts, 0)
			continue
		}
		parts = append(parts, value)
	}
	return parts
}

func parseIntSafe(value string) (int, error) {
	result := 0
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("invalid digit")
		}
		result = result*10 + int(ch-'0')
	}
	return result, nil
}
