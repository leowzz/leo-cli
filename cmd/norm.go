package cmd

import "strings"

func normalizeCode(text string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= '\uff01' && r <= '\uff5e':
			return r - 0xfee0
		case r == '\u3000':
			return ' '
		}

		switch r {
		case '\u3001':
			return ','
		case '\u3002':
			return '.'
		case '\u3008', '\u300a':
			return '<'
		case '\u3009', '\u300b':
			return '>'
		case '\u3010', '\u3014':
			return '['
		case '\u3011', '\u3015':
			return ']'
		case '\u2018', '\u2019':
			return '\''
		case '\u201c', '\u201d':
			return '"'
		default:
			return r
		}
	}, text)
}
