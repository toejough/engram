package recall

import (
	"context"
	"path/filepath"
	"strings"

	"engram/internal/externalsources"
)

// ExtractFromAutoMemory runs the auto-memory phase of the recall pipeline.
//
// It uses MEMORY.md as the index for one Haiku rank call, then iterates the
// returned topic files in rank order, Haiku-extracting each into the buffer
// until cap is reached. Returns total bytes appended this phase.
func ExtractFromAutoMemory(
	ctx context.Context,
	files []externalsources.ExternalFile,
	query string,
	cache *externalsources.FileCache,
	summarizer Extractor,
	buffer *strings.Builder,
	bytesUsed, bytesCap int,
) int {
	if summarizer == nil || cache == nil || bytesUsed >= bytesCap {
		return 0
	}

	indexBody, indexFound := readMemoryIndex(files, cache)
	if !indexFound {
		return 0
	}

	rankPrompt := "Rank topic files by relevance to the query, one filename per line. Query: " + query

	rankResponse, rankErr := summarizer.ExtractRelevant(ctx, string(indexBody), rankPrompt)
	if rankErr != nil {
		return 0
	}

	topicByName := indexAutoMemoryFiles(files)
	added := 0

	for _, name := range parseRankedLines(rankResponse) {
		if ctx.Err() != nil || bytesUsed+added >= bytesCap {
			break
		}

		added += processOneTopic(ctx, name, topicByName, cache, summarizer, query, buffer)
	}

	return added
}

// indexAutoMemoryFiles builds a basename → absolute path map for auto memory files.
func indexAutoMemoryFiles(files []externalsources.ExternalFile) map[string]string {
	index := make(map[string]string, len(files))

	for _, file := range files {
		if file.Kind == externalsources.KindAutoMemory {
			index[filepath.Base(file.Path)] = file.Path
		}
	}

	return index
}

// processOneTopic extracts a snippet from one ranked topic file and writes it
// to the buffer. Returns the number of bytes written (0 on any failure).
func processOneTopic(
	ctx context.Context,
	name string,
	topicByName map[string]string,
	cache *externalsources.FileCache,
	summarizer Extractor,
	query string,
	buffer *strings.Builder,
) int {
	path, ok := topicByName[name]
	if !ok {
		return 0
	}

	body, readErr := cache.Read(path)
	if readErr != nil {
		return 0
	}

	snippet, extractErr := summarizer.ExtractRelevant(ctx, string(body), query)
	if extractErr != nil || snippet == "" {
		return 0
	}

	buffer.WriteString(snippet)

	return len(snippet)
}

// readMemoryIndex finds MEMORY.md in the files list and returns its body.
func readMemoryIndex(files []externalsources.ExternalFile, cache *externalsources.FileCache) ([]byte, bool) {
	for _, file := range files {
		if file.Kind == externalsources.KindAutoMemory && filepath.Base(file.Path) == "MEMORY.md" {
			body, err := cache.Read(file.Path)
			if err != nil {
				return nil, false
			}

			return body, true
		}
	}

	return nil, false
}
