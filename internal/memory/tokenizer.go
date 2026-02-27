package memory

import (
	"bufio"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Tokenizer implements BERT WordPiece tokenization.
type Tokenizer struct {
	vocab map[string]int64
}

// NewTokenizer creates a new WordPiece tokenizer with e5-small-v2 vocabulary.
func NewTokenizer() *Tokenizer {
	vocab, err := loadVocab()
	if err != nil {
		// Return tokenizer with empty vocab - will use [UNK] for all tokens
		return &Tokenizer{vocab: make(map[string]int64)}
	}

	return &Tokenizer{vocab: vocab}
}

// Tokenize converts text to token IDs using WordPiece algorithm.
// Returns token IDs wrapped with [CLS] at start and [SEP] at end.
func (t *Tokenizer) Tokenize(text string) []int64 {
	// Start with [CLS]
	tokens := []int64{clsTokenID}

	// Normalize to lowercase
	text = strings.ToLower(text)

	// Split into words
	words := strings.FieldsSeq(text)

	// Apply WordPiece to each word
	for word := range words {
		tokens = append(tokens, t.wordpiece(word)...)
	}

	// End with [SEP]
	tokens = append(tokens, sepTokenID)

	return tokens
}

// wordpiece applies WordPiece algorithm to split a single word into subwords.
func (t *Tokenizer) wordpiece(word string) []int64 {
	if len(word) == 0 {
		return nil
	}

	tokens := []int64{}
	start := 0

	for start < len(word) {
		end := len(word)

		var subword string

		found := false

		// Greedily match longest subword
		for end > start {
			if start > 0 {
				subword = "##" + word[start:end]
			} else {
				subword = word[start:end]
			}

			if id, exists := t.vocab[subword]; exists {
				tokens = append(tokens, id)
				found = true

				break
			}

			end--
		}

		if !found {
			// No match found - use [UNK] for entire word and stop
			tokens = append(tokens, unkTokenID)
			break
		}

		start = end
	}

	return tokens
}

// unexported constants.
const (
	clsTokenID      = 101 // [CLS]
	e5SmallVocabURL = "https://huggingface.co/intfloat/e5-small-v2/resolve/main/vocab.txt"
	sepTokenID      = 102 // [SEP]
	unkTokenID      = 100 // [UNK]
)

// unexported variables.
var (
	vocabCache   map[string]int64
	vocabCacheMu sync.RWMutex
	vocabOnce    sync.Once
)

// downloadVocab downloads the vocab file from HuggingFace if not present.
func downloadVocab(vocabPath string, client *http.Client) error {
	// Check if already exists
	if _, err := os.Stat(vocabPath); err == nil {
		return nil
	}

	// Create directory
	if err := os.MkdirAll(filepath.Dir(vocabPath), 0755); err != nil {
		return fmt.Errorf("failed to create vocab directory: %w", err)
	}

	// Download vocab
	resp, err := client.Get(e5SmallVocabURL)
	if err != nil {
		return fmt.Errorf("failed to download vocab: %w", err)
	}

	if resp == nil {
		return errors.New("failed to download vocab: nil response")
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download vocab: HTTP %d", resp.StatusCode)
	}

	// Write to temp file
	tempPath := vocabPath + ".tmp"

	out, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create vocab file: %w", err)
	}

	defer func() { _ = out.Close() }()

	// Copy data
	scanner := bufio.NewScanner(resp.Body)

	writer := bufio.NewWriter(out)
	for scanner.Scan() {
		if _, err := writer.WriteString(scanner.Text() + "\n"); err != nil {
			_ = os.Remove(tempPath)
			return fmt.Errorf("failed to write vocab: %w", err)
		}
	}

	if err := scanner.Err(); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to read vocab: %w", err)
	}

	if err := writer.Flush(); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to flush vocab: %w", err)
	}

	_ = out.Close()

	// Rename to final path
	if err := os.Rename(tempPath, vocabPath); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to finalize vocab: %w", err)
	}

	return nil
}

// loadVocab loads the e5-small-v2 vocabulary file.
// Uses caching to avoid repeated downloads.
func loadVocab() (map[string]int64, error) {
	// Try cached vocab first
	vocabCacheMu.RLock()

	if vocabCache != nil {
		defer vocabCacheMu.RUnlock()
		return vocabCache, nil
	}

	vocabCacheMu.RUnlock()

	// Load vocab (once)
	var loadErr error

	vocabOnce.Do(func() {
		modelDir := filepath.Join(os.Getenv("HOME"), ".projctl", "models")
		vocabPath := filepath.Join(modelDir, "vocab.txt")

		// Download if not present
		if err := downloadVocab(vocabPath, http.DefaultClient); err != nil {
			loadErr = err
			return
		}

		// Parse vocab file
		vocab, err := parseVocab(vocabPath)
		if err != nil {
			loadErr = err
			return
		}

		// Cache for future use
		vocabCacheMu.Lock()

		vocabCache = vocab

		vocabCacheMu.Unlock()
	})

	if loadErr != nil {
		return nil, loadErr
	}

	vocabCacheMu.RLock()
	defer vocabCacheMu.RUnlock()

	return vocabCache, nil
}

// parseVocab parses the vocab.txt file into a token -> ID map.
func parseVocab(vocabPath string) (map[string]int64, error) {
	file, err := os.Open(vocabPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open vocab: %w", err)
	}

	defer func() { _ = file.Close() }()

	vocab := make(map[string]int64)
	scanner := bufio.NewScanner(file)
	lineNum := int64(0)

	for scanner.Scan() {
		token := scanner.Text()
		vocab[token] = lineNum
		lineNum++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read vocab: %w", err)
	}

	return vocab, nil
}
