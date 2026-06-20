package cmd

import "strings"

// compareVersions compares two Debian-style version strings, returning -1, 0,
// or 1. It implements dpkg's algorithm: alternating non-digit and digit runs,
// where digit runs compare numerically and non-digit runs compare by a special
// ordering in which '~' sorts before everything (even end-of-string) and
// letters sort before other punctuation.
//
// It is used for --latest (newest version per package) and version sorting, so
// 0.0.9 correctly precedes 0.0.10.
func compareVersions(a, b string) int {
	for len(a) > 0 || len(b) > 0 {
		// Non-digit run.
		ai, bi := 0, 0
		for ai < len(a) && !isDigit(a[ai]) {
			ai++
		}
		for bi < len(b) && !isDigit(b[bi]) {
			bi++
		}
		if c := compareNonDigit(a[:ai], b[:bi]); c != 0 {
			return c
		}
		a, b = a[ai:], b[bi:]

		// Digit run, compared numerically (ignoring leading zeros).
		ai, bi = 0, 0
		for ai < len(a) && isDigit(a[ai]) {
			ai++
		}
		for bi < len(b) && isDigit(b[bi]) {
			bi++
		}
		if c := compareNumeric(a[:ai], b[:bi]); c != 0 {
			return c
		}
		a, b = a[ai:], b[bi:]
	}
	return 0
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }
func isAlpha(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }

// order returns the dpkg sort weight of a character.
func order(c byte) int {
	switch {
	case c == '~':
		return -1
	case isAlpha(c):
		return int(c)
	default:
		return int(c) + 256
	}
}

func compareNonDigit(a, b string) int {
	for i := 0; i < len(a) || i < len(b); i++ {
		var oa, ob int
		if i < len(a) {
			oa = order(a[i])
		}
		if i < len(b) {
			ob = order(b[i])
		}
		if oa != ob {
			if oa < ob {
				return -1
			}
			return 1
		}
	}
	return 0
}

func compareNumeric(a, b string) int {
	a = strings.TrimLeft(a, "0")
	b = strings.TrimLeft(b, "0")
	if len(a) != len(b) {
		if len(a) < len(b) {
			return -1
		}
		return 1
	}
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
