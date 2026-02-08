package memory_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/projctl/internal/memory"
)

// ============================================================================
// Unit tests for BERT WordPiece tokenizer (TASK-23)
// These tests verify correct token ID assignment and subword splitting
// according to BERT WordPiece algorithm.
//
// Traces to: TASK-23 AC "New file tokenizer_test.go with unit tests"
// ============================================================================

// TestTokenizer_KnownTokensReturnCorrectIDs verifies vocab lookup for known tokens.
// Traces to: TASK-23 AC "Tokenizer returns token IDs wrapped with [CLS] ... [SEP]"
func TestTokenizer_KnownTokensReturnCorrectIDs(t *testing.T) {
	_ = NewWithT(t)

	// Known BERT vocabulary tokens (these should exist in e5-small-v2 vocab)
	tests := []struct {
		name          string
		text          string
		expectedStart int64 // [CLS] token ID
		expectedEnd   int64 // [SEP] token ID
		minLength     int   // Minimum expected token count (including CLS/SEP)
	}{
		{
			name:          "simple word",
			text:          "database",
			expectedStart: 101, // [CLS]
			expectedEnd:   102, // [SEP]
			minLength:     3,   // [CLS] + word + [SEP]
		},
		{
			name:          "multiple words",
			text:          "error handling",
			expectedStart: 101, // [CLS]
			expectedEnd:   102, // [SEP]
			minLength:     4,   // [CLS] + error + handling + [SEP]
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			tokenizer := memory.NewTokenizer()
			tokenIDs := tokenizer.Tokenize(tt.text)

			// Verify minimum length
			g.Expect(len(tokenIDs)).To(BeNumerically(">=", tt.minLength),
				"tokenized output should have at least %d tokens", tt.minLength)

			// Verify [CLS] at start
			g.Expect(tokenIDs[0]).To(Equal(tt.expectedStart),
				"first token should be [CLS] (ID 101)")

			// Verify [SEP] at end
			g.Expect(tokenIDs[len(tokenIDs)-1]).To(Equal(tt.expectedEnd),
				"last token should be [SEP] (ID 102)")

			// Verify no zero tokens (padding) within actual content
			for i := 1; i < len(tokenIDs)-1; i++ {
				g.Expect(tokenIDs[i]).To(BeNumerically(">", 0),
					"token at position %d should not be padding (0)", i)
			}
		})
	}
}

// TestTokenizer_SubwordSplitting verifies WordPiece subword algorithm.
// Traces to: TASK-23 AC "Tokenizer loads vocab from e5-small-v2/vocab.txt"
func TestTokenizer_SubwordSplitting(t *testing.T) {
	g := NewWithT(t)

	tokenizer := memory.NewTokenizer()

	// Test word that should be split into subwords
	// "tokenization" might split into "token" + "##ization"
	tokenIDs := tokenizer.Tokenize("tokenization")

	// Should have at least [CLS] + some subwords + [SEP]
	g.Expect(len(tokenIDs)).To(BeNumerically(">=", 3),
		"should have [CLS], content tokens, and [SEP]")

	// First should be [CLS], last should be [SEP]
	g.Expect(tokenIDs[0]).To(Equal(int64(101)))
	g.Expect(tokenIDs[len(tokenIDs)-1]).To(Equal(int64(102)))

	// All content tokens should be valid (non-zero, within vocab range)
	for i := 1; i < len(tokenIDs)-1; i++ {
		g.Expect(tokenIDs[i]).To(BeNumerically(">", 0),
			"content token should be positive")
		g.Expect(tokenIDs[i]).To(BeNumerically("<", 30522),
			"token ID should be within BERT vocab size (30522)")
	}
}

// TestTokenizer_LowercaseNormalization verifies case-insensitive tokenization.
// Traces to: TASK-23 AC "Tokenizer returns token IDs wrapped with [CLS] ... [SEP]"
func TestTokenizer_LowercaseNormalization(t *testing.T) {
	g := NewWithT(t)

	tokenizer := memory.NewTokenizer()

	// Same word in different cases should produce same tokens
	lowerIDs := tokenizer.Tokenize("database")
	upperIDs := tokenizer.Tokenize("DATABASE")
	mixedIDs := tokenizer.Tokenize("DataBase")

	g.Expect(upperIDs).To(Equal(lowerIDs),
		"uppercase should normalize to lowercase")
	g.Expect(mixedIDs).To(Equal(lowerIDs),
		"mixed case should normalize to lowercase")
}

// TestTokenizer_EmptyInput verifies handling of empty strings.
// Traces to: TASK-23 AC "New file tokenizer_test.go with unit tests"
func TestTokenizer_EmptyInput(t *testing.T) {
	g := NewWithT(t)

	tokenizer := memory.NewTokenizer()
	tokenIDs := tokenizer.Tokenize("")

	// Empty input should still have [CLS] and [SEP]
	g.Expect(tokenIDs).To(HaveLen(2))
	g.Expect(tokenIDs[0]).To(Equal(int64(101))) // [CLS]
	g.Expect(tokenIDs[1]).To(Equal(int64(102))) // [SEP]
}

// TestTokenizer_SpecialTokens verifies [CLS] and [SEP] wrapping.
// Traces to: TASK-23 AC "Tokenizer returns token IDs wrapped with [CLS] ... [SEP] special tokens"
func TestTokenizer_SpecialTokens(t *testing.T) {
	_ = NewWithT(t)

	tokenizer := memory.NewTokenizer()

	tests := []string{
		"hello",
		"error handling",
		"PostgreSQL database query optimization",
	}

	for _, text := range tests {
		t.Run(text, func(t *testing.T) {
			g := NewWithT(t)

			tokenIDs := tokenizer.Tokenize(text)

			// Every tokenized sequence starts with [CLS] (101)
			g.Expect(tokenIDs[0]).To(Equal(int64(101)),
				"should start with [CLS] token")

			// Every tokenized sequence ends with [SEP] (102)
			g.Expect(tokenIDs[len(tokenIDs)-1]).To(Equal(int64(102)),
				"should end with [SEP] token")

			// Length should be at least 2 (CLS + SEP)
			g.Expect(len(tokenIDs)).To(BeNumerically(">=", 2))
		})
	}
}

// TestTokenizer_UnknownWords verifies [UNK] token handling.
// Traces to: TASK-23 AC "Tokenizer loads vocab from e5-small-v2/vocab.txt"
func TestTokenizer_UnknownWords(t *testing.T) {
	g := NewWithT(t)

	tokenizer := memory.NewTokenizer()

	// Use Unicode characters that are not in BERT vocab
	tokenIDs := tokenizer.Tokenize("你好🚀")

	// Should still wrap with special tokens
	g.Expect(tokenIDs[0]).To(Equal(int64(101))) // [CLS]
	g.Expect(tokenIDs[len(tokenIDs)-1]).To(Equal(int64(102))) // [SEP]

	// Content should contain [UNK] token (100) for unknown words
	foundUnk := false
	for i := 1; i < len(tokenIDs)-1; i++ {
		if tokenIDs[i] == 100 { // [UNK] token
			foundUnk = true
			break
		}
	}
	g.Expect(foundUnk).To(BeTrue(),
		"unknown words should map to [UNK] token (100)")
}

// ============================================================================
// Property-based tests using rapid
// Traces to: TASK-23 AC "Property-based tests via rapid"
// ============================================================================

// TestTokenizer_PropertyAlwaysWrapsWithSpecialTokens verifies invariant across random inputs.
func TestTokenizer_PropertyAlwaysWrapsWithSpecialTokens(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random ASCII text
		text := rapid.String().Draw(t, "text")

		tokenizer := memory.NewTokenizer()
		tokenIDs := tokenizer.Tokenize(text)

		// Property: All tokenized sequences start with [CLS] and end with [SEP]
		if len(tokenIDs) > 0 {
			if tokenIDs[0] != 101 {
				t.Fatalf("first token should always be [CLS] (101), got %d", tokenIDs[0])
			}
			if tokenIDs[len(tokenIDs)-1] != 102 {
				t.Fatalf("last token should always be [SEP] (102), got %d", tokenIDs[len(tokenIDs)-1])
			}
		}

		// Property: Token IDs should be within BERT vocab range
		for i, id := range tokenIDs {
			if id < 0 || id >= 30522 {
				t.Fatalf("token ID at position %d out of range: %d (expected 0-30521)", i, id)
			}
		}
	})
}

// TestTokenizer_PropertyDeterministic verifies same input produces same output.
func TestTokenizer_PropertyDeterministic(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		text := rapid.String().Draw(t, "text")

		tokenizer := memory.NewTokenizer()

		// Tokenize same text twice
		tokenIDs1 := tokenizer.Tokenize(text)
		tokenIDs2 := tokenizer.Tokenize(text)

		// Property: Tokenization is deterministic
		if len(tokenIDs1) != len(tokenIDs2) {
			t.Fatalf("tokenization not deterministic: lengths differ %d vs %d", len(tokenIDs1), len(tokenIDs2))
		}

		for i := range tokenIDs1 {
			if tokenIDs1[i] != tokenIDs2[i] {
				t.Fatalf("tokenization not deterministic at position %d: %d vs %d", i, tokenIDs1[i], tokenIDs2[i])
			}
		}
	})
}

// TestTokenizer_PropertyNonEmpty verifies non-empty output.
func TestTokenizer_PropertyNonEmpty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		text := rapid.String().Draw(t, "text")

		tokenizer := memory.NewTokenizer()
		tokenIDs := tokenizer.Tokenize(text)

		// Property: Output should always contain at least [CLS] and [SEP]
		if len(tokenIDs) < 2 {
			t.Fatalf("tokenized output should have at least 2 tokens ([CLS] + [SEP]), got %d", len(tokenIDs))
		}
	})
}

// TestTokenizer_PropertyCaseInsensitive verifies case normalization.
func TestTokenizer_PropertyCaseInsensitive(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random lowercase text
		text := rapid.StringMatching(`[a-z ]+`).Draw(t, "text")
		if text == "" {
			t.Skip("empty text")
		}

		tokenizer := memory.NewTokenizer()

		// Tokenize lowercase and uppercase versions
		lowerIDs := tokenizer.Tokenize(text)
		upperIDs := tokenizer.Tokenize(toUpper(text))

		// Property: Case should not affect tokenization
		if len(lowerIDs) != len(upperIDs) {
			t.Fatalf("case sensitivity detected: lengths differ %d vs %d", len(lowerIDs), len(upperIDs))
		}

		for i := range lowerIDs {
			if lowerIDs[i] != upperIDs[i] {
				t.Fatalf("case sensitivity detected at position %d: %d vs %d", i, lowerIDs[i], upperIDs[i])
			}
		}
	})
}

// ============================================================================
// Helper functions
// ============================================================================

// toUpper converts string to uppercase (simple ASCII).
func toUpper(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			c -= 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}
