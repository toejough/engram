package slug

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// foldAccents strips combining marks after NFD decomposition, so "é" becomes "e".
var foldAccents = transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)

// Slugify builds a page's URL slug from its title: lowercase, ASCII-folded, with every run of
// non-alphanumeric characters collapsed to one hyphen and hyphens trimmed from both ends.
func Slugify(title string) string {
	folded, _, err := transform.String(foldAccents, title)
	if err != nil {
		folded = title
	}

	var out strings.Builder
	pendingHyphen := false

	for _, r := range strings.ToLower(folded) {
		if r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)) {
			if pendingHyphen && out.Len() > 0 {
				out.WriteByte('-')
			}
			out.WriteRune(r)
			pendingHyphen = false
			continue
		}
		pendingHyphen = true
	}

	return out.String()
}
