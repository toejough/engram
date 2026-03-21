package keyword

import "strings"

// Normalize canonicalizes a keyword: lowercase, hyphens replaced with underscores.
// This ensures "prefixed-ids", "prefixed_IDs", and "prefixed_ids" all map to "prefixed_ids".
func Normalize(kw string) string {
	return strings.ReplaceAll(strings.ToLower(kw), "-", "_")
}

// NormalizeAll normalizes a slice of keywords, returning a new slice.
// Returns nil if input is nil.
func NormalizeAll(kws []string) []string {
	if kws == nil {
		return nil
	}

	result := make([]string, len(kws))

	for i, kw := range kws {
		result[i] = Normalize(kw)
	}

	return result
}
