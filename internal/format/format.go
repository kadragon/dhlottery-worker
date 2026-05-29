// Package format provides number and currency formatting helpers that match
// the ko-KR locale output used by the original implementation
// (e.g. 20000 -> "20,000", 5000 -> "5,000원").
package format

import "strconv"

// KoreanNumber formats an integer with comma thousands separators, matching
// JavaScript's Number.prototype.toLocaleString("ko-KR") for integer values.
func KoreanNumber(n int) string {
	neg := n < 0
	s := strconv.Itoa(n)
	if neg {
		s = s[1:]
	}

	// Insert a comma every three digits from the right.
	groups := (len(s) - 1) / 3
	if groups == 0 {
		if neg {
			return "-" + s
		}
		return s
	}

	out := make([]byte, 0, len(s)+groups)
	lead := len(s) % 3
	if lead == 0 {
		lead = 3
	}
	out = append(out, s[:lead]...)
	for i := lead; i < len(s); i += 3 {
		out = append(out, ',')
		out = append(out, s[i:i+3]...)
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}

// Currency formats an amount as "{KoreanNumber}원".
func Currency(n int) string {
	return KoreanNumber(n) + "원"
}
