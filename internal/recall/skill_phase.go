package recall

import (
	"context"
	"fmt"
	"strings"

	"engram/internal/externalsources"
)

// ExtractFromSkills runs the skills phase of the recall pipeline. It builds a
// name+description index from each skill's frontmatter (one Haiku rank call),
// then iterates ranked-winning skills in order, Haiku-extracting from each
// body into the buffer until cap is hit.
func ExtractFromSkills(
	ctx context.Context,
	files []externalsources.ExternalFile,
	query string,
	cache *externalsources.FileCache,
	summarizer SummarizerI,
	buffer *strings.Builder,
	bytesUsed, bytesCap int,
) int {
	if summarizer == nil || cache == nil || bytesUsed >= bytesCap {
		return 0
	}

	pathByName, index := loadSkillIndex(files, cache)
	if index == "" {
		return 0
	}

	rankPrompt := "Rank skills by relevance to the query, one skill name per line. Query: " + query

	rankResponse, rankErr := summarizer.ExtractRelevant(ctx, index, rankPrompt)
	if rankErr != nil {
		return 0
	}

	added := 0

	for _, name := range parseRankedSkillNames(rankResponse) {
		if ctx.Err() != nil || bytesUsed+added >= bytesCap {
			break
		}

		snippet := extractOneSkill(ctx, name, query, pathByName, cache, summarizer)
		if snippet == "" {
			continue
		}

		buffer.WriteString(snippet)

		added += len(snippet)
	}

	return added
}

func extractOneSkill(
	ctx context.Context,
	name, query string,
	pathByName map[string]string,
	cache *externalsources.FileCache,
	summarizer SummarizerI,
) string {
	path, ok := pathByName[name]
	if !ok {
		return ""
	}

	body, readErr := cache.Read(path)
	if readErr != nil {
		return ""
	}

	_, rest := externalsources.ParseFrontmatter(body)

	snippet, extractErr := summarizer.ExtractRelevant(ctx, string(rest), query)
	if extractErr != nil {
		return ""
	}

	return snippet
}

// loadSkillIndex builds the skill index used for the Haiku rank call and
// returns a name → path map in a single pass over the discovered files.
// Each skill file is read and its frontmatter parsed exactly once.
func loadSkillIndex(
	files []externalsources.ExternalFile,
	cache *externalsources.FileCache,
) (pathByName map[string]string, index string) {
	pathByName = make(map[string]string)

	var builder strings.Builder

	for _, file := range files {
		if file.Kind != externalsources.KindSkill {
			continue
		}

		body, err := cache.Read(file.Path)
		if err != nil {
			continue
		}

		matter, _ := externalsources.ParseFrontmatter(body)
		if matter.Name == "" {
			continue
		}

		pathByName[matter.Name] = file.Path
		fmt.Fprintf(&builder, "%s | %s\n", matter.Name, matter.Description)
	}

	return pathByName, builder.String()
}

// parseRankedSkillNames extracts skill names from a Haiku rank response (one per line).
func parseRankedSkillNames(response string) []string {
	lines := strings.Split(strings.TrimSpace(response), "\n")
	names := make([]string, 0, len(lines))

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			names = append(names, trimmed)
		}
	}

	return names
}
