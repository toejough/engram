// Package tokenize provides shared text tokenization for retrieval scoring.
package tokenize

import (
	"strings"
	"unicode"
)

// Tokenize splits text into lowercase alphanumeric tokens.
// Non-letter, non-digit characters are treated as delimiters.
func Tokenize(text string) []string {
	var tokens []string

	var current strings.Builder

	for _, ch := range strings.ToLower(text) {
		if unicode.IsLetter(ch) || unicode.IsDigit(ch) {
			current.WriteRune(ch)
		} else if current.Len() > 0 {
			tokens = append(tokens, current.String())
			current.Reset()
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// Frequencies tokenizes text and returns a term-frequency map.
func Frequencies(text string) map[string]int {
	tokens := Tokenize(text)
	if len(tokens) == 0 {
		return nil
	}

	freqs := make(map[string]int, len(tokens))
	for _, tok := range tokens {
		freqs[tok]++
	}

	return freqs
}
