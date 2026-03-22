// Package cli implements the engram command-line interface (ARCH-6).
package cli

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"engram/internal/classify"
	sessionctx "engram/internal/context"
	"engram/internal/correct"
	"engram/internal/creationlog"
	"engram/internal/dedup"
	"engram/internal/effectiveness"
	"engram/internal/extract"
	"engram/internal/instruct"
	"engram/internal/learn"
	"engram/internal/maintain"
	"engram/internal/memory"
	"engram/internal/recall"
	"engram/internal/render"
	"engram/internal/retrieve"
	reviewpkg "engram/internal/review"
	"engram/internal/signal"
	"engram/internal/surface"
	"engram/internal/surfacinglog"
	"engram/internal/tfidf"
	"engram/internal/tokenresolver"
	"engram/internal/tomlwriter"
	"engram/internal/toolgate"
	"engram/internal/track"
	"engram/internal/transcript"
)

// Exported variables.
var (
	AnthropicAPIURL = "https://api.anthropic.com/v1/messages" //nolint:gochecknoglobals // test-overridable endpoint
)

// ReadFileFunc reads a file by path, injected for testability.
type ReadFileFunc func(path string) ([]byte, error)

// RenderLearnResult writes DES-10 feedback for a learn result to w.
func RenderLearnResult(w io.Writer, result *learn.Result) {
	if len(result.CreatedPaths) == 0 {
		_, _ = fmt.Fprintln(w, "[engram] No new learnings extracted.")
		return
	}

	tierBreakdown := formatTierBreakdown(result.TierCounts)

	_, _ = fmt.Fprintf(
		w,
		"[engram] Extracted %d learnings from session. %s\n",
		len(result.CreatedPaths),
		tierBreakdown,
	)

	for _, path := range result.CreatedPaths {
		base := filepath.Base(path)
		_, _ = fmt.Fprintf(w, "  - %q (%s)\n", base, base)
	}

	if result.SkippedCount > 0 {
		_, _ = fmt.Fprintf(
			w,
			"[engram] Skipped %d duplicates.\n",
			result.SkippedCount,
		)
	}
}

// Run dispatches to the appropriate subcommand based on args.
// Output is written to stdout. Errors are returned (caller logs to stderr, exit 0).
//
//nolint:cyclop // CLI dispatch switch grows with each new subcommand
func Run(
	args []string,
	stdout, stderr io.Writer,
	stdin io.Reader,
) error {
	if len(args) < minArgs {
		return errUsage
	}

	cmd := args[1]
	subArgs := args[minArgs:]

	switch cmd {
	case "correct":
		return runCorrect(subArgs, stdout)
	case "flush":
		return runFlush(subArgs, stdout, stderr, stdin)
	case "review":
		return RunReview(subArgs, stdout)
	case "maintain":
		return runMaintain(subArgs, stdout)
	case "surface":
		return runSurface(subArgs, stdout)
	case "learn":
		return runLearn(subArgs, stderr, stdin)
	case "instruct":
		return runInstructAudit(subArgs, stdout)
	case "feedback":
		return runFeedback(subArgs, stdout)
	case "show":
		return runShow(subArgs, stdout)
	case "apply-proposal":
		return runApplyProposal(subArgs, stdout)
	case "recall":
		return runRecall(subArgs, stdout)
	default:
		return fmt.Errorf("%w: %s", errUnknownCommand, cmd)
	}
}

// RunLearn implements the learn subcommand with an injectable HTTP doer.
// httpClient is passed to the extractor; nil uses a default http.Client.
// token is the Anthropic API token; empty string triggers the no-token error path.
//
//nolint:funlen // orchestration function: wires extractor, learner, and DI dependencies
func RunLearn(
	args []string,
	token string,
	stderr io.Writer,
	stdin io.Reader,
	httpClient extract.HTTPDoer,
) error {
	fs := flag.NewFlagSet("learn", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")
	transcriptPath := fs.String(
		"transcript-path", "", "path to session transcript",
	)
	sessionID := fs.String("session-id", "", "session identifier")
	projectSlug := fs.String("project-slug", "", "originating project slug")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("learn: %w", parseErr)
	}

	if *dataDir == "" {
		return errLearnMissingFlags
	}

	if token == "" {
		_, _ = fmt.Fprintln(
			stderr,
			"[engram] Error: session learning skipped"+
				" — no API token configured",
		)

		return nil
	}

	if httpClient == nil {
		httpClient = &http.Client{}
	}

	extractor := extract.New(token, httpClient)
	retriever := retrieve.New()
	deduplicator := dedup.New()
	writer := tomlwriter.New()

	learner := learn.New(
		extractor, retriever, deduplicator, writer, *dataDir,
	)

	if *projectSlug != "" {
		learner.SetProjectSlug(*projectSlug)
	}

	learner.SetCreationLogger(creationlog.NewLogWriter())

	learner.SetRegisterMemory(func(filePath, _, content string, _ time.Time) error {
		h := sha256.Sum256([]byte(content))
		hash := hex.EncodeToString(h[:])

		return memory.ReadModifyWrite(filePath, func(r *memory.MemoryRecord) {
			r.SourceType = "memory"
			r.SourcePath = filePath
			r.ContentHash = hash
		})
	})

	ctx := context.Background()

	// Incremental mode: read delta from transcript file.
	if *transcriptPath != "" && *sessionID != "" {
		return runIncrementalLearn(
			ctx, learner, *transcriptPath, *sessionID, *dataDir, stderr,
		)
	}

	// Stdin mode: read full transcript from stdin (backward compatible).
	transcriptBytes, err := io.ReadAll(stdin)
	if err != nil {
		return fmt.Errorf("learn: reading stdin: %w", err)
	}

	result, err := learner.Run(ctx, string(transcriptBytes))
	if err != nil {
		return fmt.Errorf("learn: %w", err)
	}

	RenderLearnResult(stderr, result)

	return nil
}

// RunMaintain implements the maintain subcommand: generates maintenance
// proposals as a JSON array on stdout. token controls LLM availability
// for leech/hidden-gem proposals; empty string skips them.
//
//nolint:cyclop,funlen // CLI wiring
func RunMaintain(
	args []string,
	token string,
	stdout io.Writer,
	opts ...maintain.Option,
) error {
	fs := flag.NewFlagSet("maintain", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")
	applyMode := fs.Bool("apply", false, "apply proposals instead of generating")
	proposalsPath := fs.String("proposals", "", "path to proposals JSON file")
	autoYes := fs.Bool("yes", false, "auto-approve all proposals (no confirmation)")
	dryRun := fs.Bool("dry-run", false, "print merge plan without writing files")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("maintain: %w", parseErr)
	}

	if *dataDir == "" {
		return errMaintainMissingFlags
	}

	if *applyMode {
		return runMaintainApply(
			*proposalsPath, *autoYes, token, stdout,
		)
	}

	ctx := context.Background()
	retriever := retrieve.New()

	if *dryRun {
		return runMaintainDryRun(ctx, retriever, *dataDir, stdout)
	}

	memories, err := retriever.ListMemories(ctx, *dataDir)
	if err != nil {
		return fmt.Errorf("maintain: listing memories: %w", err)
	}

	if len(memories) == 0 {
		_, _ = fmt.Fprint(stdout, "[]\n")

		return nil
	}

	stats := effectiveness.FromMemories(memories)

	// Consolidate duplicates before classification (UC-34).
	backupDir := filepath.Join(*dataDir, "memories", ".backup")

	consolidator := signal.NewConsolidator(
		signal.WithLister(&memoryListerAdapter{
			retriever: retriever,
			dataDir:   *dataDir,
		}),
		signal.WithMerger(&fileMergeExecutor{}),
		signal.WithFileWriter(newStoredMemoryWriter()),
		signal.WithFileDeleter(&osFileDeleter{}),
		signal.WithBackupWriter(&osBackupWriter{now: time.Now}, backupDir),
		signal.WithEntryRemover(os.Remove),
		signal.WithEffectiveness(&effectivenessReaderAdapter{stats: stats}),
		signal.WithStderr(os.Stderr),
		signal.WithPrincipleSynthesizer(newPrincipleSynthesizer(token)),
		signal.WithLinkRecomputer(newGraphLinkRecomputer(*dataDir)),
		signal.WithTextSimilarityScorer(tfidf.NewScorer()),
	)

	_, consolidateErr := consolidator.Consolidate(ctx)
	if consolidateErr != nil {
		return fmt.Errorf("maintain: consolidating: %w", consolidateErr)
	}

	tracking := buildTrackingFromMemories(memories)
	classified := reviewpkg.Classify(stats, tracking)

	memoryMap := buildMemoryMapFromSlice(memories)

	allOpts := make([]maintain.Option, 0, len(opts)+1)
	if token != "" {
		allOpts = append(allOpts,
			maintain.WithLLMCaller(makeAnthropicCaller(token)))
	}

	allOpts = append(allOpts, opts...)

	generator := maintain.New(allOpts...)
	proposals := generator.Generate(ctx, classified, memoryMap)

	// UC-21: Run escalation engine on leech memories (ARCH-50).
	leeches := buildEscalationMemories(classified, memoryMap)
	if len(leeches) > 0 {
		engine := maintain.NewEscalationEngine(maintain.EffData{}, nil)

		escalations, escErr := engine.Analyze(leeches)
		if escErr == nil {
			for idx := range escalations {
				escJSON := maintain.MarshalProposal(escalations[idx])
				proposals = append(proposals, maintain.Proposal{
					MemoryPath: escalations[idx].MemoryPath,
					Quadrant:   string(reviewpkg.Leech),
					Diagnosis:  escalations[idx].Rationale,
					Action:     "escalation_" + escalations[idx].ProposalType,
					Details:    escJSON,
				})
			}
		}
	}

	//nolint:wrapcheck // thin JSON encoding at CLI boundary
	return json.NewEncoder(stdout).Encode(proposals)
}

// RunReview implements the review subcommand: reads the TOML memory directory,
// classifies entries by quadrant, and renders grouped output (ARCH-59, DES-27).
func RunReview(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("review", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")
	format := fs.String("format", "table", "output format: json, table")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("review: %w", parseErr)
	}

	if *dataDir == "" {
		return errReviewMissingFlags
	}

	memoriesDir := filepath.Join(*dataDir, "memories")

	records, err := memory.ListAll(memoriesDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			_, _ = fmt.Fprintln(stdout, "[engram] No registry entries found.")

			return nil
		}

		return fmt.Errorf("review: listing memories: %w", err)
	}

	if len(records) == 0 {
		_, _ = fmt.Fprintln(stdout, "[engram] No registry entries found.")

		return nil
	}

	classifications := classifyStoredRecords(records, *dataDir)

	switch *format {
	case formatJSON:
		//nolint:wrapcheck // thin JSON encoding at CLI boundary
		return json.NewEncoder(stdout).Encode(classifications)
	default:
		renderReviewTable(stdout, classifications)

		return nil
	}
}

// unexported constants.
const (
	anthropicMaxTokens           = 1024
	anthropicVersion             = "2023-06-01"
	filePermOwnerRW              = 0o600
	formatJSON                   = "json"
	haikuModel                   = "claude-haiku-4-5-20251001"
	maintainModel                = "claude-haiku-4-5-20251001"
	maxTitleLength               = 38
	maxTranscriptTok             = 2000
	minArgs                      = 2
	reviewEffectivenessThreshold = 50.0
	reviewSurfacingThreshold     = 3
)

// unexported variables.
var (
	errCorrectMissingFlags = errors.New(
		"correct: --message and --data-dir required",
	)
	errInstructMissingFlags = errors.New(
		"instruct audit: --data-dir required",
	)
	errLearnMissingFlags             = errors.New("learn: --data-dir required")
	errMaintainApplyMissingProposals = errors.New(
		"maintain --apply: --proposals required",
	)
	errMaintainMissingFlags = errors.New("maintain: --data-dir required")
	errNilAPIResponse       = errors.New("calling Anthropic API: nil response")
	errNoContentBlocks      = errors.New("API response contained no content blocks")
	errRecallMissingFlags   = errors.New(
		"recall: --data-dir and --project-slug required",
	)
	errReviewMissingFlags  = errors.New("review: --data-dir required")
	errSkillExists         = errors.New("skill already exists")
	errSurfaceMissingFlags = errors.New(
		"surface: --mode and --data-dir required",
	)
	errUnknownCommand = errors.New("unknown command")
	errUsage          = errors.New(
		"usage: engram <correct|surface|learn|recall" +
			"|review|maintain|instruct|show|feedback> [flags]",
	)
)

// anthropicContentBlock is a content block in an Anthropic API response.
type anthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// anthropicMessage is a single message in the Anthropic messages API.
type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// anthropicRequest is the request body for the Anthropic messages API.
//
//nolint:tagliatelle // Anthropic API requires snake_case JSON field names.
type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system"`
	Messages  []anthropicMessage `json:"messages"`
}

// anthropicResponse is the response body from the Anthropic messages API.
type anthropicResponse struct {
	Content []anthropicContentBlock `json:"content"`
}

// cliConfirmer prompts the user for confirmation via stdin/stdout.
type cliConfirmer struct {
	stdout      io.Writer
	stdin       io.Reader
	autoConfirm bool
}

func (c *cliConfirmer) Confirm(preview string) (bool, error) {
	_, _ = fmt.Fprintf(c.stdout, "\n--- Skill Preview ---\n%s\n--- End Preview ---\n", preview)

	if c.autoConfirm {
		_, _ = fmt.Fprintln(c.stdout, "Auto-confirmed (--yes).")

		return true, nil
	}

	_, _ = fmt.Fprint(c.stdout, "Apply this change? [y/N] ")

	var response string

	_, err := fmt.Fscan(c.stdin, &response)
	if err != nil {
		return false, fmt.Errorf("reading confirmation: %w", err)
	}

	return strings.EqualFold(response, "y") || strings.EqualFold(response, "yes"), nil
}

// cliLLMCaller implements maintain.LLMCaller via the Anthropic API.
type cliLLMCaller struct {
	token string
}

func (c *cliLLMCaller) Call(ctx context.Context, prompt string) (string, error) {
	caller := makeAnthropicCaller(c.token)

	return caller(ctx, maintainModel, "You are a memory maintenance assistant.", prompt)
}

// effectivenessAdapter bridges a pre-built stats map to surface.EffectivenessComputer.
type effectivenessAdapter struct {
	stats map[string]effectiveness.Stat
}

func (a *effectivenessAdapter) Aggregate() (map[string]surface.EffectivenessStat, error) {
	result := make(map[string]surface.EffectivenessStat, len(a.stats))

	for memPath, stat := range a.stats {
		total := stat.FollowedCount + stat.ContradictedCount + stat.IgnoredCount
		result[memPath] = surface.EffectivenessStat{
			SurfacedCount:      total,
			EffectivenessScore: stat.EffectivenessScore,
		}
	}

	return result, nil
}

// fileCounterStore implements toolgate.CounterStore with atomic file I/O.
type fileCounterStore struct {
	path string
}

func (s *fileCounterStore) Load() (map[string]toolgate.CounterEntry, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]toolgate.CounterEntry), nil
		}

		return nil, fmt.Errorf("toolgate load: %w", err)
	}

	var counters map[string]toolgate.CounterEntry

	jsonErr := json.Unmarshal(data, &counters)
	if jsonErr != nil {
		return make(map[string]toolgate.CounterEntry), nil //nolint:nilerr // corrupt file: start fresh
	}

	return counters, nil
}

func (s *fileCounterStore) Save(counters map[string]toolgate.CounterEntry) error {
	data, err := json.Marshal(counters)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	tmp := s.path + ".tmp"

	writeErr := os.WriteFile(tmp, data, filePermOwnerRW)
	if writeErr != nil {
		return fmt.Errorf("write tmp: %w", writeErr)
	}

	renameErr := os.Rename(tmp, s.path)
	if renameErr != nil {
		return fmt.Errorf("rename: %w", renameErr)
	}

	return nil
}

// haikuCallerAdapter adapts makeAnthropicCaller to the recall.HaikuCaller interface.
type haikuCallerAdapter struct {
	caller func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error)
}

func (a *haikuCallerAdapter) Call(
	ctx context.Context,
	systemPrompt, userPrompt string,
) (string, error) {
	return a.caller(ctx, haikuModel, systemPrompt, userPrompt)
}

// osClaudeMDStore reads and writes CLAUDE.md files on disk.
type osClaudeMDStore struct {
	path string
}

func (s *osClaudeMDStore) Read() (string, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return "", nil
	}

	if err != nil {
		return "", fmt.Errorf("reading CLAUDE.md: %w", err)
	}

	return string(data), nil
}

func (s *osClaudeMDStore) Write(content string) error {
	const filePerms = 0o644

	writeErr := os.WriteFile(s.path, []byte(content), filePerms)
	if writeErr != nil {
		return fmt.Errorf("writing CLAUDE.md: %w", writeErr)
	}

	return nil
}

// osDirLister lists .jsonl files in a directory using os.ReadDir.
type osDirLister struct{}

func (l *osDirLister) ListJSONL(dir string) ([]recall.FileEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("listing directory: %w", err)
	}

	results := make([]recall.FileEntry, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".jsonl") {
			continue
		}

		info, infoErr := entry.Info()
		if infoErr != nil {
			continue
		}

		results = append(results, recall.FileEntry{
			Path:  filepath.Join(dir, name),
			Mtime: info.ModTime(),
		})
	}

	return results, nil
}

// I/O adapters for context package DI interfaces.

type osFileReader struct{}

func (r *osFileReader) Read(path string) ([]byte, error) {
	return os.ReadFile(path) //nolint:gosec,wrapcheck // thin I/O adapter
}

// osMemoryRemover deletes a memory TOML file from disk.
type osMemoryRemover struct{}

func (r *osMemoryRemover) Remove(path string) error {
	rmErr := os.Remove(path)
	if rmErr != nil {
		return fmt.Errorf("removing memory: %w", rmErr)
	}

	return nil
}

// osOffsetStore implements learn.OffsetStore using the filesystem.
type osOffsetStore struct{}

func (s *osOffsetStore) Read(path string) (learn.Offset, error) {
	data, err := os.ReadFile(path) //nolint:gosec // thin I/O adapter
	if err != nil {
		return learn.Offset{}, fmt.Errorf("reading offset: %w", err)
	}

	var offset learn.Offset

	unmarshalErr := json.Unmarshal(data, &offset)
	if unmarshalErr != nil {
		return learn.Offset{}, fmt.Errorf("parsing offset: %w", unmarshalErr)
	}

	return offset, nil
}

func (s *osOffsetStore) Write(path string, offset learn.Offset) error {
	data, err := json.Marshal(offset)
	if err != nil {
		return fmt.Errorf("marshaling offset: %w", err)
	}

	const filePerms = 0o644

	return os.WriteFile(path, data, filePerms) //nolint:wrapcheck // thin I/O adapter
}

// osSkillWriter writes skill files to a directory on disk.
type osSkillWriter struct {
	dir string
}

func (w *osSkillWriter) Write(name, content string) (string, error) {
	const dirPerms = 0o755

	mkErr := os.MkdirAll(w.dir, dirPerms)
	if mkErr != nil {
		return "", fmt.Errorf("creating skills dir: %w", mkErr)
	}

	path := filepath.Join(w.dir, name+".md")

	_, err := os.Stat(path)
	if err == nil {
		return "", fmt.Errorf("%w at %s: %q", errSkillExists, path, name)
	}

	const filePerms = 0o644

	writeErr := os.WriteFile(path, []byte(content), filePerms)
	if writeErr != nil {
		return "", fmt.Errorf("writing skill: %w", writeErr)
	}

	return path, nil
}

// reviewClassification holds the quadrant classification for a single entry.
//
//nolint:tagliatelle // spec requires snake_case JSON field names.
type reviewClassification struct {
	ID            string   `json:"id"`
	SourceType    string   `json:"source_type"`
	Title         string   `json:"title"`
	Quadrant      string   `json:"quadrant"`
	Effectiveness *float64 `json:"effectiveness,omitempty"`
	SurfacedCount int      `json:"surfaced_count"`
}

// stdinConfirmer implements maintain.Confirmer with stdin/stdout interaction.
type stdinConfirmer struct {
	stdout io.Writer
	stdin  io.Reader
}

func (sc *stdinConfirmer) Confirm(preview string) (bool, error) {
	_, _ = fmt.Fprintf(sc.stdout, "\n%s\n\nApply? [a]pply / [s]kip / [q]uit: ", preview)

	const confirmBufSize = 16

	buf := make([]byte, confirmBufSize)

	n, err := sc.stdin.Read(buf)
	if err != nil {
		return false, fmt.Errorf("reading confirmation: %w", err)
	}

	input := string(buf[:n])
	if len(input) > 0 {
		switch input[0] {
		case 'a', 'A', 'y', 'Y':
			return true, nil
		case 'q', 'Q':
			return false, maintain.ErrUserQuit
		}
	}

	return false, nil
}

// buildEscalationMemories extracts leech memories for the escalation engine (UC-21, ARCH-50).
func buildEscalationMemories(
	classified []reviewpkg.ClassifiedMemory,
	memoryMap map[string]*memory.Stored,
) []maintain.EscalationMemory {
	leeches := make([]maintain.EscalationMemory, 0)

	for _, classifiedMem := range classified {
		if classifiedMem.Quadrant != reviewpkg.Leech {
			continue
		}

		stored := memoryMap[classifiedMem.Name]
		content := ""

		if stored != nil {
			content = stored.Content
		}

		leeches = append(leeches, maintain.EscalationMemory{
			Path:          classifiedMem.Name,
			Content:       content,
			Effectiveness: classifiedMem.EffectivenessScore,
		})
	}

	return leeches
}

// buildMemoryMapFromSlice builds a path→Stored map from a pre-loaded slice.
func buildMemoryMapFromSlice(memories []*memory.Stored) map[string]*memory.Stored {
	memMap := make(map[string]*memory.Stored, len(memories))
	for _, mem := range memories {
		memMap[mem.FilePath] = mem
	}

	return memMap
}

// buildTrackingFromMemories builds a path→TrackingData map from a pre-loaded slice.
func buildTrackingFromMemories(memories []*memory.Stored) map[string]reviewpkg.TrackingData {
	tracking := make(map[string]reviewpkg.TrackingData, len(memories))

	for _, mem := range memories {
		tracking[mem.FilePath] = reviewpkg.TrackingData{
			SurfacedCount: mem.SurfacedCount,
		}
	}

	return tracking
}

// callAnthropicAPI makes a single call to the Anthropic messages API and returns the text response.
func callAnthropicAPI(
	ctx context.Context,
	client *http.Client,
	token, model, systemPrompt, userPrompt string,
) (string, error) {
	reqBody := anthropicRequest{
		Model:     model,
		MaxTokens: anthropicMaxTokens,
		System:    systemPrompt,
		Messages:  []anthropicMessage{{Role: "user", Content: userPrompt}},
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		AnthropicAPIURL,
		bytes.NewReader(reqBytes),
	)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Anthropic-Version", anthropicVersion)
	req.Header.Set("Anthropic-Beta", "oauth-2025-04-20")
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("calling Anthropic API: %w", err)
	}

	if resp == nil {
		return "", errNilAPIResponse
	}

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	var apiResp anthropicResponse

	jsonErr := json.Unmarshal(body, &apiResp)
	if jsonErr != nil {
		return "", fmt.Errorf("parsing API response: %w", jsonErr)
	}

	if len(apiResp.Content) == 0 {
		return "", errNoContentBlocks
	}

	return apiResp.Content[0].Text, nil
}

// classifyStoredRecords builds review classifications from memory.StoredRecord values.
// The ID for each entry is the relative path within dataDir (e.g. "memories/foo.toml").
func classifyStoredRecords(records []memory.StoredRecord, dataDir string) []reviewClassification {
	classifications := make([]reviewClassification, 0, len(records))

	for i := range records {
		rec := &records[i]

		relPath, relErr := filepath.Rel(dataDir, rec.Path)
		if relErr != nil {
			relPath = rec.Path
		}

		total := rec.Record.FollowedCount + rec.Record.ContradictedCount + rec.Record.IgnoredCount

		var eff *float64

		const (
			minEvals          = 3
			percentMultiplier = 100.0
		)

		if total >= minEvals {
			score := float64(rec.Record.FollowedCount) / float64(total) * percentMultiplier
			eff = &score
		}

		quadrant := reviewQuadrant(rec.Record.SurfacedCount, eff)

		classifications = append(classifications, reviewClassification{
			ID:            relPath,
			SourceType:    rec.Record.SourceType,
			Title:         rec.Record.Title,
			Quadrant:      quadrant,
			Effectiveness: eff,
			SurfacedCount: rec.Record.SurfacedCount,
		})
	}

	return classifications
}

// cryptoRandFloat returns a cryptographically random float64 in [0, 1).
func cryptoRandFloat() float64 {
	const (
		maxMantissa  = 1 << 53
		randBytes    = 8
		mantissaBits = 11
	)

	b := make([]byte, randBytes)
	_, _ = rand.Read(b)

	val := binary.BigEndian.Uint64(b) >> mantissaBits

	return float64(val) / float64(maxMantissa)
}

// formatTierBreakdown returns a string like "(A: 2, B: 1, C: 3)" from tier counts.
func formatTierBreakdown(counts map[string]int) string {
	if len(counts) == 0 {
		return ""
	}

	parts := make([]string, 0, len(counts))

	for _, tier := range []string{"A", "B", "C"} {
		if count, ok := counts[tier]; ok && count > 0 {
			parts = append(
				parts,
				fmt.Sprintf("%s: %d", tier, count),
			)
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return "(" + strings.Join(parts, ", ") + ")"
}

// makeAnthropicCaller returns an LLM caller function backed by the Anthropic API.
func makeAnthropicCaller(
	token string,
) func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
	client := &http.Client{}

	return func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
		return callAnthropicAPI(ctx, client, token, model, systemPrompt, userPrompt)
	}
}

func newFileCounterStore(dataDir string) *fileCounterStore {
	return &fileCounterStore{path: filepath.Join(dataDir, "tool-frecency.json")}
}

func newTokenResolver() *tokenresolver.Resolver {
	return tokenresolver.New(
		os.Getenv,
		func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, name, args...).Output() //nolint:gosec // args are hardcoded in caller
		},
		runtime.GOOS,
	)
}

func renderReviewTable(
	writer io.Writer,
	classifications []reviewClassification,
) {
	_, _ = fmt.Fprintf(writer,
		"[engram] Instruction Review (%d entries)\n\n",
		len(classifications))

	// Group by source_type.
	groups := make(map[string][]reviewClassification)

	for _, classification := range classifications {
		groups[classification.SourceType] = append(
			groups[classification.SourceType], classification)
	}

	sourceTypes := make([]string, 0, len(groups))
	for sourceType := range groups {
		sourceTypes = append(sourceTypes, sourceType)
	}

	sort.Strings(sourceTypes)

	for _, sourceType := range sourceTypes {
		_, _ = fmt.Fprintf(writer, "Source: %s\n", sourceType)

		group := groups[sourceType]

		sort.Slice(group, func(i, j int) bool {
			return group[i].Quadrant < group[j].Quadrant
		})

		for _, entry := range group {
			effStr := "N/A"
			if entry.Effectiveness != nil {
				effStr = fmt.Sprintf("%.1f%%", *entry.Effectiveness)
			}

			_, _ = fmt.Fprintf(writer,
				"  %-14s %-40s %8s  surfaced=%d\n",
				entry.Quadrant, truncateTitle(entry.Title),
				effStr, entry.SurfacedCount)
		}

		_, _ = fmt.Fprintln(writer)
	}
}

// resolveSkillsDir returns the plugin's skills directory if available.
func resolveSkillsDir() string {
	pluginRoot := os.Getenv("CLAUDE_PLUGIN_ROOT")
	if pluginRoot == "" {
		return ""
	}

	return filepath.Join(pluginRoot, "skills")
}

// resolveToken returns the API token from the environment or macOS Keychain.
// Resolver errors are swallowed (keychain unavailability is non-fatal), so
// the error return is always nil and callers receive an empty string on failure.
func resolveToken(ctx context.Context) string {
	token, _ := newTokenResolver().Resolve(ctx) // error is always nil; keychain failures are swallowed

	return token
}

// reviewQuadrant classifies a memory by surfacing frequency and effectiveness.
func reviewQuadrant(surfacedCount int, eff *float64) string {
	if eff == nil {
		return "Insufficient"
	}

	highEff := *eff >= reviewEffectivenessThreshold
	oftenSurfaced := surfacedCount >= reviewSurfacingThreshold

	switch {
	case oftenSurfaced && highEff:
		return "Working"
	case oftenSurfaced && !highEff:
		return "Leech"
	case !oftenSurfaced && highEff:
		return "Hidden Gem"
	default:
		return "Noise"
	}
}

func runCorrect(
	args []string,
	stdout io.Writer,
) error {
	fs := flag.NewFlagSet("correct", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	message := fs.String("message", "", "user message text")
	dataDir := fs.String("data-dir", "", "path to data directory")
	transcriptPath := fs.String(
		"transcript-path", "", "path to session transcript",
	)
	projectSlug := fs.String("project-slug", "", "originating project slug")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("correct: %w", parseErr)
	}

	if *message == "" || *dataDir == "" {
		return errCorrectMissingFlags
	}

	// Read transcript context if available (os.ReadFile wired at the edge)
	reader := transcript.New(os.ReadFile)
	reader.SetStrip(sessionctx.Strip)

	transcriptCtx, _ := reader.ReadRecent(
		*transcriptPath, maxTranscriptTok,
	)

	ctx := context.Background()

	token := resolveToken(ctx)

	classifier := classify.New(token, &http.Client{})
	writer := tomlwriter.New()
	renderer := render.New()

	corrector := correct.New(classifier, writer, renderer, *dataDir)

	if *projectSlug != "" {
		corrector.SetProjectSlug(*projectSlug)
	}

	output, err := corrector.Run(ctx, *message, transcriptCtx)
	if err != nil {
		return fmt.Errorf("correct: %w", err)
	}

	if output != "" {
		_, _ = fmt.Fprint(stdout, output)
	}

	return nil
}

// runIncrementalLearn creates an IncrementalLearner and runs it.
func runIncrementalLearn(
	ctx context.Context,
	learner *learn.Learner,
	transcriptPath, sessionID, dataDir string,
	stderr io.Writer,
) error {
	reader := &osFileReader{}
	delta := sessionctx.NewDeltaReader(reader)
	offsetStore := &osOffsetStore{}
	offsetPath := filepath.Join(dataDir, "learn-offset.json")

	inc := learn.NewIncrementalLearner(
		learner, delta, sessionctx.Strip, offsetStore, stderr,
	)

	result, err := inc.RunIncremental(
		ctx, transcriptPath, sessionID, offsetPath,
	)
	if err != nil {
		return fmt.Errorf("learn: incremental: %w", err)
	}

	if result != nil {
		RenderLearnResult(stderr, result)
	}

	return nil
}

func runInstructAudit(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("instruct", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")
	projectDir := fs.String("project-dir", "", "path to project directory")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("instruct: %w", parseErr)
	}

	if *dataDir == "" {
		return errInstructMissingFlags
	}

	if *projectDir == "" {
		*projectDir = "."
	}

	scanner := &instruct.Scanner{
		ReadFile:  os.ReadFile,
		GlobFiles: filepath.Glob,
		EffData:   map[string]float64{},
	}

	ctx := context.Background()

	token := resolveToken(ctx)

	var llmCaller func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error)
	if token != "" {
		llmCaller = makeAnthropicCaller(token)
	}

	auditor := &instruct.Auditor{
		Scanner:   scanner,
		LLMCaller: llmCaller,
	}

	report, err := auditor.Run(ctx, *dataDir, *projectDir)
	if err != nil {
		return fmt.Errorf("instruct audit: %w", err)
	}

	//nolint:wrapcheck // thin JSON encoding at CLI boundary
	return json.NewEncoder(stdout).Encode(report)
}

func runLearn(args []string, stderr io.Writer, stdin io.Reader) error {
	return RunLearn(args, resolveToken(context.Background()), stderr, stdin, nil)
}

func runMaintain(args []string, stdout io.Writer) error {
	return RunMaintain(args, resolveToken(context.Background()), stdout)
}

// a JSON file and applies them with user confirmation (T-264, ARCH-66).
func runMaintainApply(
	proposalsPath string, autoYes bool,
	token string, stdout io.Writer,
) error {
	if proposalsPath == "" {
		return errMaintainApplyMissingProposals
	}

	cleanPath := filepath.Clean(proposalsPath)

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return fmt.Errorf("maintain --apply: reading proposals: %w", err)
	}

	proposals, err := maintain.IngestProposals(data)
	if err != nil {
		return fmt.Errorf("maintain --apply: %w", err)
	}

	if len(proposals) == 0 {
		_, _ = fmt.Fprintln(stdout, "[engram] No valid proposals to apply.")

		return nil
	}

	_, _ = fmt.Fprintf(stdout,
		"[engram] %d proposals to apply.\n", len(proposals))

	execOpts := make([]maintain.ExecutorOption, 0, 5) //nolint:mnd

	rewriter := maintain.NewTOMLRewriter()
	execOpts = append(execOpts, maintain.WithRewriter(rewriter))
	execOpts = append(execOpts, maintain.WithRemover(&osMemoryRemover{}))
	execOpts = append(execOpts, maintain.WithFileRemover(os.Remove))

	if token != "" {
		execOpts = append(execOpts, maintain.WithLLMCaller2(
			&cliLLMCaller{token: token},
		))
	}

	if !autoYes {
		execOpts = append(execOpts, maintain.WithConfirmer(
			&stdinConfirmer{stdout: stdout, stdin: os.Stdin},
		))
	}

	executor := maintain.NewExecutor(execOpts...)
	ctx := context.Background()
	report := executor.Apply(ctx, proposals)

	_, _ = fmt.Fprintf(stdout,
		"[engram] Applied %d/%d (%d skipped, %d not reached)\n",
		report.Applied, report.Total, report.Skipped, report.NotReached,
	)

	for _, reason := range report.SkipReasons {
		_, _ = fmt.Fprintf(stdout, "  skipped: %s\n", reason)
	}

	return nil
}

// runMaintainDryRun computes the merge plan and prints it as JSON without
// modifying any files (--dry-run flag, T-362, #335).
func runMaintainDryRun(ctx context.Context, retriever *retrieve.Retriever, dataDir string, stdout io.Writer) error {
	consolidator := signal.NewConsolidator(
		signal.WithLister(&memoryListerAdapter{
			retriever: retriever,
			dataDir:   dataDir,
		}),
		signal.WithTextSimilarityScorer(tfidf.NewScorer()),
	)

	plans, err := consolidator.Plan(ctx)
	if err != nil {
		return fmt.Errorf("maintain: planning: %w", err)
	}

	encErr := json.NewEncoder(stdout).Encode(plans)
	if encErr != nil {
		return fmt.Errorf("maintain: encoding plan: %w", encErr)
	}

	return nil
}

func runRecall(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("recall", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")
	projectSlug := fs.String("project-slug", "", "project directory slug")
	query := fs.String("query", "", "search query (omit for summary mode)")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("recall: %w", parseErr)
	}

	if *dataDir == "" || *projectSlug == "" {
		return errRecallMissingFlags
	}

	home := os.Getenv("HOME")
	projectDir := filepath.Join(home, ".claude", "projects", *projectSlug)

	ctx := context.Background()

	token := resolveToken(ctx)

	finder := recall.NewSessionFinder(&osDirLister{})
	reader := recall.NewTranscriptReader(&osFileReader{})

	var summarizer recall.SummarizerI
	if token != "" {
		summarizer = recall.NewSummarizer(&haikuCallerAdapter{
			caller: makeAnthropicCaller(token),
		})
	}

	orch := recall.NewOrchestrator(finder, reader, summarizer, nil)

	result, err := orch.Recall(ctx, projectDir, *query)
	if err != nil {
		return fmt.Errorf("recall: %w", err)
	}

	_, writeErr := fmt.Fprint(stdout, result.Summary)
	if writeErr != nil {
		return fmt.Errorf("recall: writing summary: %w", writeErr)
	}

	if result.Memories != "" {
		_, writeErr = fmt.Fprint(stdout, "\n=== MEMORIES ===\n", result.Memories)
		if writeErr != nil {
			return fmt.Errorf("recall: writing memories: %w", writeErr)
		}
	}

	return nil
}

//nolint:funlen // wired with flags, cross-ref checker, and transcript window
func runSurface(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("surface", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	mode := fs.String("mode", "", "surface mode: session-start, prompt, tool, precompact")
	dataDir := fs.String("data-dir", "", "path to data directory")
	message := fs.String("message", "", "user message (prompt mode)")
	toolName := fs.String("tool-name", "", "tool name (tool mode)")
	toolInput := fs.String("tool-input", "", "tool input JSON (tool mode)")
	toolOutput := fs.String("tool-output", "", "tool output or error text (tool mode)")
	toolErrored := fs.Bool("tool-errored", false, "true if tool call failed (tool mode)")
	format := fs.String("format", "", "output format: json")
	budget := fs.Int("budget", 0, "token budget override (precompact mode)")
	transcriptWindow := fs.String("transcript-window", "", "recent transcript text for suppression (REQ-P4f-3)")
	claudeDir := fs.String("claude-dir", "", "path to .claude directory for cross-source suppression (REQ-P4f-2)")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("surface: %w", parseErr)
	}

	if *mode == "" || *dataDir == "" {
		return errSurfaceMissingFlags
	}

	opts := surface.Options{
		Mode:             *mode,
		DataDir:          *dataDir,
		Message:          *message,
		ToolName:         *toolName,
		ToolInput:        *toolInput,
		ToolOutput:       *toolOutput,
		ToolErrored:      *toolErrored,
		Format:           *format,
		Budget:           *budget,
		TranscriptWindow: *transcriptWindow,
	}

	retriever := retrieve.New()
	recorder := track.NewRecorder()
	logReader := creationlog.NewLogReader()
	surfLogger := surfacinglog.New(*dataDir)

	ctx := context.Background()

	memories, memErr := retriever.ListMemories(ctx, *dataDir)
	if memErr != nil {
		return fmt.Errorf("surface: listing memories: %w", memErr)
	}

	effAdapter := &effectivenessAdapter{stats: effectiveness.FromMemories(memories)}

	surfacerOpts := []surface.SurfacerOption{
		surface.WithTracker(recorder),
		surface.WithLogReader(logReader),
		surface.WithSurfacingLogger(surfLogger),
		surface.WithEffectiveness(effAdapter),
		surface.WithSurfacingRecorder(func(path string) error {
			return memory.ReadModifyWrite(path, func(record *memory.MemoryRecord) {
				record.SurfacedCount++
				record.LastSurfacedAt = time.Now().UTC().Format(time.RFC3339)
			})
		}),
	}

	if *claudeDir != "" {
		checker := newSourceCrossRefChecker(*claudeDir, memories)
		if checker != nil {
			surfacerOpts = append(surfacerOpts, surface.WithCrossRefChecker(checker))
		}
	}

	// Tool frecency gate: inject with crypto/rand source and file-backed store.
	store := newFileCounterStore(*dataDir)
	gate := toolgate.NewGate(store, cryptoRandFloat)
	surfacerOpts = append(surfacerOpts, surface.WithToolGate(gate))

	surfacer := surface.New(retriever, surfacerOpts...)

	return surfacer.Run(ctx, stdout, opts)
}

func truncateTitle(title string) string {
	if len(title) <= maxTitleLength {
		return title
	}

	return title[:maxTitleLength-1] + "…"
}
