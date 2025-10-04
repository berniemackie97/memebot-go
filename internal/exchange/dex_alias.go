package exchange

import (
	"strings"
)

func sanitizeDexAlias(alias string) string {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(alias))
	for _, r := range alias {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			if r >= 'a' && r <= 'z' {
				r -= 32
			}
			b.WriteRune(r)
		}
	}
	return b.String()
}

func composeDexAlias(base, address string) string {
	base = sanitizeDexAlias(base)
	suffix := sanitizeDexAlias(address)
	if len(suffix) > 6 {
		suffix = suffix[len(suffix)-6:]
	}
	if base == "" {
		if suffix == "" {
			return "PAIR"
		}
		return "PAIR_" + suffix
	}
	if suffix == "" {
		return base
	}
	return base + "_" + suffix
}
