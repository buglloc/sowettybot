package renderer

import "strings"

func EscapeTgMd(in string) string {
	var out strings.Builder
	for _, c := range in {
		// https://core.telegram.org/bots/api#markdownv2-style
		switch c {
		case '_', '*', '[', ']', '(', ')', '~', '>', '#', '+', '-', '=', '|', '{', '}', '.', '!':
			_ = out.WriteByte('\\')
		}
		_, _ = out.WriteRune(c)
	}

	return out.String()
}
