package cmd

import (
	"fmt"
	"sort"
)

// none renders an empty value as a visible placeholder.
func none(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}

// toStr renders an arbitrary JSON value as a compact string for display.
func toStr(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", t)
	}
}

// sortPairs orders key/value pairs by key for stable display.
func sortPairs(pairs [][2]string) [][2]string {
	sort.Slice(pairs, func(i, j int) bool { return pairs[i][0] < pairs[j][0] })
	return pairs
}
