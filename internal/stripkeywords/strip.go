// Package stripkeywords removes legacy Keywords suffixes from memory situation fields.
package stripkeywords

import "strings"

// StripKeywordsSuffix removes the first occurrence of "\nKeywords: ..." from s,
// where "..." extends to the end of the string. If no such suffix exists, s is
// returned unchanged. The operation is idempotent.
func StripKeywordsSuffix(s string) string {
	idx := strings.Index(s, "\nKeywords:")
	if idx == -1 {
		return s
	}

	return s[:idx]
}
