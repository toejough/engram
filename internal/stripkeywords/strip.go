// Package stripkeywords removes legacy Keywords suffixes from memory situation fields.
package stripkeywords

import "strings"

// Suffix removes the first occurrence of "\nKeywords: ..." from s,
// where "..." extends to the end of the string. If no such suffix exists, s is
// returned unchanged. The operation is idempotent.
func Suffix(s string) string {
	before, _, found := strings.Cut(s, "\nKeywords:")
	if !found {
		return s
	}

	return before
}
