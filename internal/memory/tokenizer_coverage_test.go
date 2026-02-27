package memory

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
)

// TestDownloadVocab_BadDirectory verifies downloadVocab returns error when dir can't be created.
func TestDownloadVocab_BadDirectory(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// /dev/null/subdir cannot be created
	err := downloadVocab("/dev/null/cannot/create/here/vocab.txt", http.DefaultClient)

	g.Expect(err).To(HaveOccurred())
}

// TestDownloadVocab_ClientGetError verifies downloadVocab returns error when the HTTP GET fails.
// Covers the client.Get error return path (lines 128-130).
func TestDownloadVocab_ClientGetError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	client := &http.Client{Transport: &errorRoundTripper{}}
	vocabPath := filepath.Join(t.TempDir(), "vocab.txt")

	err := downloadVocab(vocabPath, client)

	g.Expect(err).To(MatchError(ContainSubstring("failed to download vocab")))
}

// TestDownloadVocab_CreateFileError verifies downloadVocab returns error when os.Create fails
// on the temp file (lines 146-148). Pre-creates a directory at tempPath to force the failure.
func TestDownloadVocab_CreateFileError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	tmpDir := t.TempDir()
	vocabPath := filepath.Join(tmpDir, "vocab.txt")

	// Pre-create a directory at the temp file path so os.Create(vocabPath+".tmp") fails.
	err := os.Mkdir(vocabPath+".tmp", 0755)
	g.Expect(err).ToNot(HaveOccurred())

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintln(w, "token1")
	}))

	defer ts.Close()

	err = downloadVocab(vocabPath, testClient(ts.URL))

	g.Expect(err).To(MatchError(ContainSubstring("failed to create vocab file")))
}

// TestDownloadVocab_ExistingFile verifies downloadVocab returns nil when file already exists.
func TestDownloadVocab_ExistingFile(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	tmpDir := t.TempDir()
	vocabPath := filepath.Join(tmpDir, "vocab.txt")

	// Create file so it already exists
	err := os.WriteFile(vocabPath, []byte("token1\ntoken2\n"), 0644)
	g.Expect(err).ToNot(HaveOccurred())

	// downloadVocab should return nil when file already exists (fast path)
	err = downloadVocab(vocabPath, http.DefaultClient)

	g.Expect(err).ToNot(HaveOccurred())
}

// TestDownloadVocab_ScannerError verifies downloadVocab returns error when the response body
// returns a read error mid-scan (lines 163-166). Uses pipeBodyRoundTripper for reliable triggering.
func TestDownloadVocab_ScannerError(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	client := &http.Client{Transport: &pipeBodyRoundTripper{}}
	vocabPath := filepath.Join(t.TempDir(), "vocab.txt")

	err := downloadVocab(vocabPath, client)

	g.Expect(err).To(MatchError(ContainSubstring("failed to read vocab")))
}

// TestLoadVocab_CachedPath verifies loadVocab returns cached vocab without re-downloading.
// NOTE: Not parallel - mutates vocabCache global state. Must restore cache after test.
func TestLoadVocab_CachedPath(t *testing.T) {
	g := NewWithT(t)

	// Pre-populate the cache so loadVocab hits the "already cached" fast path
	fakeVocab := map[string]int64{"hello": 1, "world": 2}

	vocabCacheMu.Lock()

	savedCache := vocabCache
	vocabCache = fakeVocab

	vocabCacheMu.Unlock()

	defer func() {
		vocabCacheMu.Lock()

		vocabCache = savedCache

		vocabCacheMu.Unlock()
	}()

	result, err := loadVocab()

	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result).To(Equal(fakeVocab))
}

// TestNewTokenizer_CachedVocab verifies NewTokenizer returns tokenizer with cached vocab.
// NOTE: Not parallel - mutates vocabCache global state.
func TestNewTokenizer_CachedVocab(t *testing.T) {
	g := NewWithT(t)

	fakeVocab := map[string]int64{"the": 100, "##ing": 101}

	vocabCacheMu.Lock()

	savedCache := vocabCache
	vocabCache = fakeVocab

	vocabCacheMu.Unlock()

	defer func() {
		vocabCacheMu.Lock()

		vocabCache = savedCache

		vocabCacheMu.Unlock()
	}()

	tok := NewTokenizer()

	g.Expect(tok).ToNot(BeNil())
	// With the cached vocab, the tokenizer should have the vocab set
	tokens := tok.Tokenize("the")
	// [CLS] the [SEP]
	g.Expect(tokens).To(HaveLen(3))
	g.Expect(tokens[1]).To(Equal(int64(100)))
}

// TestNewTokenizer_ReturnsNonNil verifies NewTokenizer always returns a non-nil tokenizer.
func TestNewTokenizer_ReturnsNonNil(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// NewTokenizer may fail to load vocab (no network/file), but still returns a tokenizer
	tok := NewTokenizer()

	g.Expect(tok).ToNot(BeNil())
}

// TestNewTokenizer_TokenizesWithEmptyVocab verifies Tokenize works with empty vocab.
func TestNewTokenizer_TokenizesWithEmptyVocab(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Create tokenizer with empty vocab (all words become [UNK])
	tok := &Tokenizer{vocab: make(map[string]int64)}
	tokens := tok.Tokenize("hello world")

	// Should have at least [CLS] and [SEP] + possibly [UNK] tokens
	g.Expect(tokens).ToNot(BeEmpty())
	g.Expect(tokens[0]).To(Equal(int64(clsTokenID)))
	g.Expect(tokens[len(tokens)-1]).To(Equal(int64(sepTokenID)))
}

// TestNewTokenizer_TokenizesWithVocab verifies Tokenize with a populated vocab.
func TestNewTokenizer_TokenizesWithVocab(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vocab := map[string]int64{
		"hello": 500,
		"world": 600,
	}
	tok := &Tokenizer{vocab: vocab}
	tokens := tok.Tokenize("hello world")

	// [CLS] hello world [SEP]
	g.Expect(tokens).To(HaveLen(4))
	g.Expect(tokens[0]).To(Equal(int64(clsTokenID)))
	g.Expect(tokens[1]).To(Equal(int64(500)))
	g.Expect(tokens[2]).To(Equal(int64(600)))
	g.Expect(tokens[3]).To(Equal(int64(sepTokenID)))
}

// errorRoundTripper always returns an error from RoundTrip, simulating a connection failure.
type errorRoundTripper struct{}

func (e *errorRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("simulated connection error")
}

// pipeBodyRoundTripper returns a 200 response whose body writes one line then errors.
// This triggers the scanner error path (bufio.Scanner.Err() != nil).
type pipeBodyRoundTripper struct{}

func (p *pipeBodyRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	pr, pw := io.Pipe()

	go func() {
		_, _ = fmt.Fprintln(pw, "hello")
		_ = pw.CloseWithError(errors.New("simulated read error"))
	}()

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       pr,
		Header:     make(http.Header),
	}, nil
}
