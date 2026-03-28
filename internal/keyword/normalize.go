package keyword

import "strings"

// IntersectionSize returns the number of keys present in both sets.
func IntersectionSize(a, b map[string]struct{}) int {
	count := 0

	for key := range a {
		if _, ok := b[key]; ok {
			count++
		}
	}

	return count
}

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

// Set builds a normalized keyword set from a slice of keywords.
func Set(keywords []string) map[string]struct{} {
	set := make(map[string]struct{}, len(keywords))
	for _, kw := range keywords {
		set[Normalize(kw)] = struct{}{}
	}

	return set
}
