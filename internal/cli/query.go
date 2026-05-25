package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/vaultgraph"
)

// QueryArgs holds parsed flags for `engram query`.
type QueryArgs struct {
	Query     string `targ:"positional,name=query,desc=natural-language query string"`
	VaultPath string `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root"`
	Limit     int    `targ:"flag,name=limit,desc=max number of items to return (default 20)"`
}

// QueryDeps holds injected dependencies for the query command.
type QueryDeps struct {
	Scan     func(vault string) ([]vaultgraph.Note, error)
	Read     func(path string) ([]byte, error)
	Embedder embed.Embedder
}

// RunQuery embeds the query string, scores it against every note that
// has a current-model sidecar, ranks by descending cosine, and emits a
// YAML payload conforming to the spike spec.
func RunQuery(ctx context.Context, args QueryArgs, deps QueryDeps, stdout io.Writer) error {
	if args.Query == "" {
		return errQueryEmptyString
	}

	limit := args.Limit
	if limit == 0 {
		limit = defaultQueryLimit
	}

	notes, scanErr := deps.Scan(args.VaultPath)
	if scanErr != nil {
		return fmt.Errorf("query: scan: %w", scanErr)
	}

	withSidecars := countWithSidecars(notes, args.VaultPath, deps.Read)
	if len(notes) > 0 && withSidecars == 0 {
		return errQueryNoEmbeddings
	}

	queryVec, qErr := deps.Embedder.Embed(ctx, args.Query)
	if qErr != nil {
		return fmt.Errorf("query: embed: %w", qErr)
	}

	candidates := rankCandidates(notes, args.VaultPath, deps, queryVec)
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	return renderQueryPayload(stdout, args.Query, candidates, len(notes), withSidecars, limit)
}

// unexported constants.
const (
	defaultQueryLimit = 20
	unknownKind       = "unknown"
)

// unexported sentinel errors.
var (
	errQueryEmptyString  = errors.New("query: empty query string")
	errQueryNoEmbeddings = errors.New(
		"query: vault has notes but no embeddings; run `engram embed apply --all`",
	)
)

// queryBudget reports the totals visible to the caller per the YAML schema.
// Snake-case keys are spec contract — see
// docs/superpowers/research/2026-05-24-engram-query-spike.md §Spike query output.
//
//nolint:tagliatelle // YAML keys are spec contract
type queryBudget struct {
	TotalNotes         int `yaml:"total_notes"`
	WithEmbeddings     int `yaml:"with_embeddings"`
	DirectHitsReturned int `yaml:"direct_hits_returned"`
	Limit              int `yaml:"limit"`
}

// queryItem is the rendered item shape per the spike spec's YAML schema.
type queryItem struct {
	Path        string   `yaml:"path"`
	Kind        string   `yaml:"kind"`
	Score       float32  `yaml:"score"`
	Provenances []string `yaml:"provenances"`
	Content     string   `yaml:"content"`
}

// queryPayload is the top-level YAML document.
type queryPayload struct {
	Version int         `yaml:"version"`
	Query   string      `yaml:"query"`
	Items   []queryItem `yaml:"items"`
	Budget  queryBudget `yaml:"budget"`
}

// scoredCandidate aggregates one note's match against the query vector.
type scoredCandidate struct {
	notePath string
	score    float32
	content  string
}

func countWithSidecars(
	notes []vaultgraph.Note,
	vault string,
	read func(string) ([]byte, error),
) int {
	count := 0

	for _, note := range notes {
		notePath := pathOf(note.Basename, note.IsMOC)
		scFull := filepath.Join(vault, embed.SidecarPath(notePath))

		_, err := read(scFull)
		if err == nil {
			count++
		}
	}

	return count
}

// kindFromContent reads the frontmatter type field to label the item.
// Falls back to "unknown" — engram's other readers (notes, recall)
// already tolerate this case.
func kindFromContent(content string) string {
	const (
		maxScan        = 256
		typeLineMarker = "\ntype: "
		minViableLen   = len("---\ntype: x\n")
	)

	if len(content) < minViableLen {
		return unknownKind
	}

	scan := content
	if len(scan) > maxScan {
		scan = scan[:maxScan]
	}

	_, after, ok := strings.Cut(scan, typeLineMarker)
	if !ok {
		return unknownKind
	}

	kind, _, ok := strings.Cut(after, "\n")
	if !ok {
		return unknownKind
	}

	return kind
}

// newOsQueryDeps wires the production scan + read for the query command.
func newOsQueryDeps() QueryDeps {
	embedDeps := newOsEmbedDeps()

	return QueryDeps{
		Scan:     embedDeps.Scan,
		Read:     embedDeps.Read,
		Embedder: embedDeps.Embedder,
	}
}

func rankCandidates(
	notes []vaultgraph.Note,
	vault string,
	deps QueryDeps,
	queryVec []float32,
) []scoredCandidate {
	candidates := make([]scoredCandidate, 0, len(notes))
	modelID := deps.Embedder.ModelID()

	for _, note := range notes {
		notePath := pathOf(note.Basename, note.IsMOC)
		full := filepath.Join(vault, notePath)
		scFull := filepath.Join(vault, embed.SidecarPath(notePath))

		scBytes, scErr := deps.Read(scFull)
		if scErr != nil {
			continue
		}

		sidecar, parseErr := embed.UnmarshalSidecar(scBytes)
		if parseErr != nil {
			continue
		}

		if sidecar.EmbeddingModelID != modelID {
			continue
		}

		noteBytes, noteErr := deps.Read(full)
		if noteErr != nil {
			continue
		}

		candidates = append(candidates, scoredCandidate{
			notePath: notePath,
			score:    embed.Cosine(queryVec, sidecar.Vector),
			content:  string(noteBytes),
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	return candidates
}

func renderQueryPayload(
	stdout io.Writer,
	query string,
	candidates []scoredCandidate,
	totalNotes, withEmbeddings, limit int,
) error {
	items := make([]queryItem, len(candidates))

	for i, candidate := range candidates {
		items[i] = queryItem{
			Path:        candidate.notePath,
			Kind:        kindFromContent(candidate.content),
			Score:       candidate.score,
			Provenances: []string{"direct"},
			Content:     candidate.content,
		}
	}

	payload := queryPayload{
		Version: 1,
		Query:   query,
		Items:   items,
		Budget: queryBudget{
			TotalNotes:         totalNotes,
			WithEmbeddings:     withEmbeddings,
			DirectHitsReturned: len(items),
			Limit:              limit,
		},
	}

	const yamlIndent = 2

	encoder := yaml.NewEncoder(stdout)
	encoder.SetIndent(yamlIndent)

	err := encoder.Encode(payload)
	if err != nil {
		return fmt.Errorf("query: encode: %w", err)
	}

	closeErr := encoder.Close()
	if closeErr != nil {
		return fmt.Errorf("query: close encoder: %w", closeErr)
	}

	return nil
}
