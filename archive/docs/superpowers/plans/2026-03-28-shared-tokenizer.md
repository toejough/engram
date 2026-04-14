# Shared Tokenizer — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract the duplicated tokenization logic from `bm25` and `tfidf` into a shared `internal/tokenize` package.

**Architecture:** Create `internal/tokenize` with a `Tokenize(text string) []string` function. `bm25.tokenize` becomes a direct call. `tfidf.termFrequencies` calls `Tokenize` then counts frequencies.

**Tech Stack:** Go, unicode, strings

---

### Task 1: Create shared tokenize package

**Files:**
- Create: `internal/tokenize/tokenize.go`
- Create: `internal/tokenize/tokenize_test.go`

- [ ] **Step 1: Write failing tests**

```go
package tokenize_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/tokenize"
)

func TestTokenize_BasicSplit(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	tokens := tokenize.Tokenize("Hello World")
	g.Expect(tokens).To(Equal([]string{"hello", "world"}))
}

func TestTokenize_PunctuationStripped(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	tokens := tokenize.Tokenize("configuration-management, testing!")
	g.Expect(tokens).To(Equal([]string{"configuration", "management", "testing"}))
}

func TestTokenize_DigitsIncluded(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	tokens := tokenize.Tokenize("http2 h264codec")
	g.Expect(tokens).To(Equal([]string{"http2", "h264codec"}))
}

func TestTokenize_EmptyString(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	tokens := tokenize.Tokenize("")
	g.Expect(tokens).To(BeEmpty())
}

func TestTokenize_OnlyPunctuation(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	tokens := tokenize.Tokenize("---!!!")
	g.Expect(tokens).To(BeEmpty())
}

func TestFrequencies_CountsDuplicates(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	freqs := tokenize.Frequencies("the cat sat on the mat")
	g.Expect(freqs).To(Equal(map[string]int{
		"the": 2, "cat": 1, "sat": 1, "on": 1, "mat": 1,
	}))
}

func TestFrequencies_EmptyString(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	freqs := tokenize.Frequencies("")
	g.Expect(freqs).To(BeEmpty())
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `targ test -- -run 'TestTokenize_|TestFrequencies_' ./internal/tokenize/...`
Expected: FAIL — package does not exist

- [ ] **Step 3: Write implementation**

```go
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test -- -run 'TestTokenize_|TestFrequencies_' ./internal/tokenize/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tokenize/
git commit -m "refactor: add shared internal/tokenize package

Extracts duplicated tokenization logic from bm25 and tfidf.
Provides Tokenize ([]string) and Frequencies (map[string]int).

Refs #411

AI-Used: [claude]"
```

### Task 2: Migrate bm25 to use shared tokenizer

**Files:**
- Modify: `internal/bm25/bm25.go` — remove `tokenize`, import `tokenize.Tokenize`

- [ ] **Step 1: Run existing bm25 tests to confirm green baseline**

Run: `targ test -- ./internal/bm25/...`
Expected: PASS

- [ ] **Step 2: Refactor bm25.go**

Remove the `tokenize` function (lines 157-177). Replace all calls:
- Line 44: `queryTerms := tokenize(query)` → `queryTerms := tokenize.Tokenize(query)` (add import `"engram/internal/tokenize"`)
- Line 54: `docTokens := tokenize(doc.Text)` → `docTokens := tokenize.Tokenize(doc.Text)`

- [ ] **Step 3: Run bm25 tests to verify they pass**

Run: `targ test -- ./internal/bm25/...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/bm25/
git commit -m "refactor(bm25): use shared tokenizer

Refs #411

AI-Used: [claude]"
```

### Task 3: Migrate tfidf to use shared tokenizer

**Files:**
- Modify: `internal/tfidf/tfidf.go` — remove `termFrequencies`, import `tokenize.Frequencies`

- [ ] **Step 1: Run existing tfidf tests to confirm green baseline**

Run: `targ test -- ./internal/tfidf/...`
Expected: PASS

- [ ] **Step 2: Refactor tfidf.go**

Remove the `termFrequencies` function (lines 101-120). Replace the call at line 31:
- `termFreq := termFrequencies(text)` → `termFreq := tokenize.Frequencies(text)` (add import `"engram/internal/tokenize"`)

- [ ] **Step 3: Run tfidf tests to verify they pass**

Run: `targ test -- ./internal/tfidf/...`
Expected: PASS

- [ ] **Step 4: Run full test suite and lint**

Run: `targ check-full`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/tfidf/
git commit -m "refactor(tfidf): use shared tokenizer

Closes #411

AI-Used: [claude]"
```
