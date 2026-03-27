// Package cli implements the engram command-line interface (ARCH-6).
package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
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

	"engram/internal/adapt"
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
	"engram/internal/policy"
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
	case "migrate-scores":
		return runMigrateScores(subArgs, stdout, stderr)
	case "migrate-slugs":
		return runMigrateSlugs(subArgs, stdout)
	case "adapt":
		return RunAdapt(subArgs, stdout)
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

	defaultErr := applyDataDirDefault(dataDir)
	if defaultErr != nil {
		return fmt.Errorf("learn: %w", defaultErr)
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

	policyPath := filepath.Join(*dataDir, "policy.toml")
	guidance := loadExtractionGuidance(policyPath)

	extractor := extract.New(token, httpClient, extract.WithGuidance(guidance))
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
	learner.SetRegisterMemory(registerMemory)

	ctx := context.Background()

	// Incremental mode: read delta from transcript file.
	var learnErr error
	if *transcriptPath != "" && *sessionID != "" {
		learnErr = runIncrementalLearn(
			ctx, learner, *transcriptPath, *sessionID, *dataDir, stderr,
		)
	} else {
		learnErr = runStdinLearn(ctx, learner, stdin, stderr)
	}

	if learnErr != nil {
		return learnErr
	}

	// Run feedback analysis to generate adaptation proposals (fire-and-forget, ARCH-6).
	runAdaptationAnalysis(ctx, *dataDir, policyPath)

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
	purgeTierC := fs.Bool("purge-tier-c", false, "delete all tier C memory files")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("maintain: %w", parseErr)
	}

	defaultErr := applyDataDirDefault(dataDir)
	if defaultErr != nil {
		return fmt.Errorf("maintain: %w", defaultErr)
	}

	if *purgeTierC {
		return runMaintainPurgeTierC(*dataDir, stdout)
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

	// Detect duplicate clusters for consolidation proposals (UC-34).
	consolidator := signal.NewConsolidator(
		signal.WithLister(&memoryListerAdapter{
			retriever: retriever,
			dataDir:   *dataDir,
		}),
		signal.WithEffectiveness(&effectivenessReaderAdapter{stats: stats}),
		signal.WithTextSimilarityScorer(tfidf.NewScorer()),
	)

	plans, planErr := consolidator.Plan(ctx)
	if planErr != nil {
		fmt.Fprintf(os.Stderr, "[engram] consolidation plan: %v\n", planErr)
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

	// Convert consolidation plans to proposals (#373).
	for idx := range plans {
		memberCount := len(plans[idx].Absorbed) + 1
		members := make([]consolidateMember, 0, memberCount)
		members = append(members, consolidateMember{
			Path:  plans[idx].Survivor,
			Title: titleOrPath(memoryMap, plans[idx].Survivor),
		})

		for _, absorbed := range plans[idx].Absorbed {
			members = append(members, consolidateMember{
				Path:  absorbed,
				Title: titleOrPath(memoryMap, absorbed),
			})
		}

		sharedKW := sharedKeywords(memoryMap, plans[idx].Survivor, plans[idx].Absorbed)

		//nolint:errchkjson // consolidateDetails has only string/float fields; cannot fail.
		details, _ := json.Marshal(consolidateDetails{
			Members:        members,
			SharedKeywords: sharedKW,
			Confidence:     plans[idx].Confidence,
		})

		proposals = append(proposals, maintain.Proposal{
			MemoryPath: plans[idx].Survivor,
			Quadrant:   "",
			Diagnosis: fmt.Sprintf(
				"Cluster of %d memories with overlapping keywords (confidence %.2f)",
				len(members), plans[idx].Confidence,
			),
			Action:  maintain.ActionConsolidate,
			Details: details,
		})
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

	defaultErr := applyDataDirDefault(dataDir)
	if defaultErr != nil {
		return fmt.Errorf("review: %w", defaultErr)
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
		"correct: --message required",
	)
	errMaintainApplyMissingProposals = errors.New(
		"maintain --apply: --proposals required",
	)
	errNilAPIResponse          = errors.New("calling Anthropic API: nil response")
	errNoContentBlocks         = errors.New("API response contained no content blocks")
	errSkillExists             = errors.New("skill already exists")
	errSurfaceMissingFlags     = errors.New("surface: --mode required")
	errSurfaceStopNoTranscript = errors.New("surface: --transcript-path required for stop mode")
	errUnknownCommand          = errors.New("unknown command")
	errUsage                   = errors.New(
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

//nolint:tagliatelle // DES-23 specifies snake_case JSON field names.
type consolidateDetails struct {
	Members        []consolidateMember `json:"members"`
	SharedKeywords []string            `json:"shared_keywords"`
	Confidence     float64             `json:"confidence"`
}

// consolidateMember holds the file path and display title of one member in a consolidation cluster.
type consolidateMember struct {
	Path  string `json:"path"`
	Title string `json:"title"`
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

// applyDataDirDefault sets *dataDir to the standard engram data path when empty.
func applyDataDirDefault(dataDir *string) error {
	if *dataDir != "" {
		return nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolving home directory: %w", err)
	}

	*dataDir = DataDirFromHome(home)

	return nil
}

// applyProjectSlugDefault sets *slug to the PWD-derived slug when empty.
func applyProjectSlugDefault(slug *string) error {
	if *slug != "" {
		return nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolving working directory: %w", err)
	}

	*slug = ProjectSlugFromPath(cwd)

	return nil
}

// buildMemoryMapFromSlice builds a path→Stored map from a pre-loaded slice.
func buildMemoryMapFromSlice(memories []*memory.Stored) map[string]*memory.Stored {
	memMap := make(map[string]*memory.Stored, len(memories))
	for _, mem := range memories {
		memMap[mem.FilePath] = mem
	}

	return memMap
}

// buildRecallSurfacer creates a memory surfacer for the recall pipeline.
// Returns nil surfacer (not an error) when the memories directory does not exist.
func buildRecallSurfacer(ctx context.Context, dataDir string) (recall.MemorySurfacer, error) {
	retriever := retrieve.New()

	allMemories, memErr := retriever.ListMemories(ctx, dataDir)
	if memErr != nil {
		if errors.Is(memErr, os.ErrNotExist) {
			return nil, nil //nolint:nilnil // nil surfacer is valid when no memories exist
		}

		return nil, fmt.Errorf("listing memories: %w", memErr)
	}

	effAdapter := &effectivenessAdapter{stats: effectiveness.FromMemories(allMemories)}
	surfacerOpts := []surface.SurfacerOption{
		surface.WithEffectiveness(effAdapter),
		surface.WithSurfacingRecorder(recordSurfacing),
	}

	realSurfacer := surface.New(retriever, surfacerOpts...)

	return NewRecallSurfacer(
		&surfaceRunnerAdapter{surfacer: realSurfacer},
		dataDir,
	), nil
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

// extractAssistantDelta reads new transcript lines since the last stop offset,
// strips JSONL to text, and returns only assistant content joined by newlines.
func extractAssistantDelta(dataDir, transcriptPath, sessionID string) (string, error) {
	offsetPath := filepath.Join(dataDir, "stop-surface-offset.json")
	store := &osOffsetStore{}

	stored, readErr := store.Read(offsetPath)
	if readErr != nil {
		stored = learn.Offset{}
	}

	offset := stored.Offset
	if sessionID != stored.SessionID {
		offset = 0
	}

	reader := &osFileReader{}
	delta := sessionctx.NewDeltaReader(reader)

	lines, newOffset, deltaErr := delta.Read(transcriptPath, offset)
	if deltaErr != nil {
		return "", fmt.Errorf("reading transcript delta: %w", deltaErr)
	}

	// Always update offset so next call resumes from here.
	_ = store.Write(offsetPath, learn.Offset{Offset: newOffset, SessionID: sessionID})

	if len(lines) == 0 {
		return "", nil
	}

	stripped := sessionctx.Strip(lines)

	var assistantLines []string

	for _, line := range stripped {
		if text, ok := strings.CutPrefix(line, "ASSISTANT: "); ok {
			assistantLines = append(assistantLines, text)
		}
	}

	return strings.Join(assistantLines, "\n"), nil
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

// loadExtractionGuidance loads active extraction policies from policyPath and
// returns them as ExtractionGuidance slices. Returns nil if the file is missing or unreadable.
func loadExtractionGuidance(policyPath string) []extract.ExtractionGuidance {
	pf, policyErr := policy.Load(policyPath)
	if policyErr != nil {
		return nil
	}

	active := pf.Active(policy.DimensionExtraction)

	var guidance []extract.ExtractionGuidance

	for _, p := range active {
		guidance = append(guidance, extract.ExtractionGuidance{
			Directive: p.Directive,
			Rationale: p.Rationale,
		})
	}

	return guidance
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

func newTokenResolver() *tokenresolver.Resolver {
	return tokenresolver.New(
		os.Getenv,
		func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, name, args...).Output() //nolint:gosec // args are hardcoded in caller
		},
		runtime.GOOS,
	)
}

// recordSurfacing increments the surfaced count and timestamp for a memory file.
func recordSurfacing(path string) error {
	return memory.ReadModifyWrite(path, func(record *memory.MemoryRecord) {
		record.SurfacedCount++
		record.LastSurfacedAt = time.Now().UTC().Format(time.RFC3339)
	})
}

// runIncrementalLearn creates an IncrementalLearner and runs it.
// registerMemory hashes content and writes metadata to the memory TOML file (UC-23).
func registerMemory(filePath, _, content string, _ time.Time) error {
	h := sha256.Sum256([]byte(content))
	hash := hex.EncodeToString(h[:])

	return memory.ReadModifyWrite(filePath, func(r *memory.MemoryRecord) {
		r.SourceType = "memory"
		r.SourcePath = filePath
		r.ContentHash = hash
	})
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

// runAdaptationAnalysis analyses feedback patterns and appends new proposals to policy.toml.
// Errors are silently ignored (fire-and-forget, ARCH-6).
func runAdaptationAnalysis(ctx context.Context, dataDir, policyPath string) {
	const (
		minClusterSize    = 5
		minFeedbackEvents = 3
	)

	analysisConfig := adapt.Config{
		MinClusterSize:    minClusterSize,
		MinFeedbackEvents: minFeedbackEvents,
	}

	allMemories, listErr := retrieve.New().ListMemories(ctx, dataDir)
	if listErr != nil || len(allMemories) == 0 {
		return
	}

	adaptPF, loadErr := policy.Load(policyPath)
	if loadErr != nil {
		return
	}

	newProposals := adapt.AnalyzeAll(allMemories, analysisConfig)
	if len(newProposals) == 0 {
		return
	}

	for i := range newProposals {
		newProposals[i].ID = adaptPF.NextID()
		newProposals[i].CreatedAt = time.Now().UTC().Format(time.RFC3339)
		adaptPF.Policies = append(adaptPF.Policies, newProposals[i])
	}

	_ = policy.Save(policyPath, adaptPF)
}

//nolint:funlen // orchestration function: wires classifier, corrector, transcript, and DI dependencies
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

	defaultErr := applyDataDirDefault(dataDir)
	if defaultErr != nil {
		return fmt.Errorf("correct: %w", defaultErr)
	}

	slugErr := applyProjectSlugDefault(projectSlug)
	if slugErr != nil {
		return fmt.Errorf("correct: %w", slugErr)
	}

	if *message == "" {
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

	corrector.SetProjectSlug(*projectSlug)

	output, err := corrector.Run(ctx, *message, transcriptCtx)
	if err != nil {
		return fmt.Errorf("correct: %w", err)
	}

	if output != "" {
		_, _ = fmt.Fprint(stdout, output)
	}

	return nil
}

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

	defaultErr := applyDataDirDefault(dataDir)
	if defaultErr != nil {
		return fmt.Errorf("instruct: %w", defaultErr)
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

// runMaintainPurgeTierC deletes all tier C memory files and reports the count.
func runMaintainPurgeTierC(dataDir string, stdout io.Writer) error {
	memoriesDir := filepath.Join(dataDir, "memories")

	deleted, err := maintain.PurgeTierC(memoriesDir, os.Remove)
	if err != nil {
		return fmt.Errorf("maintain --purge-tier-c: %w", err)
	}

	_, _ = fmt.Fprintf(stdout, "[engram] purged %d tier C memories\n", deleted)

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

	defaultErr := applyDataDirDefault(dataDir)
	if defaultErr != nil {
		return fmt.Errorf("recall: %w", defaultErr)
	}

	slugErr := applyProjectSlugDefault(projectSlug)
	if slugErr != nil {
		return fmt.Errorf("recall: %w", slugErr)
	}

	home, homeErr := os.UserHomeDir()
	if homeErr != nil {
		return fmt.Errorf("recall: %w", homeErr)
	}

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

	memorySurfacer, surfErr := buildRecallSurfacer(ctx, *dataDir)
	if surfErr != nil {
		return fmt.Errorf("recall: %w", surfErr)
	}

	orch := recall.NewOrchestrator(finder, reader, summarizer, memorySurfacer)

	result, err := orch.Recall(ctx, projectDir, *query)
	if err != nil {
		return fmt.Errorf("recall: %w", err)
	}

	return recall.FormatResult(stdout, result)
}

// runStdinLearn reads a full transcript from stdin and runs the learner.
func runStdinLearn(
	ctx context.Context,
	learner *learn.Learner,
	stdin io.Reader,
	stderr io.Writer,
) error {
	transcriptBytes, err := io.ReadAll(stdin)
	if err != nil {
		return fmt.Errorf("learn: reading stdin: %w", err)
	}

	result, runErr := learner.Run(ctx, string(transcriptBytes))
	if runErr != nil {
		return fmt.Errorf("learn: %w", runErr)
	}

	RenderLearnResult(stderr, result)

	return nil
}

//nolint:funlen,cyclop // wired with flags, cross-ref checker, transcript window, and stop-mode delegation
func runSurface(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("surface", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	mode := fs.String("mode", "", "surface mode: prompt, stop")
	dataDir := fs.String("data-dir", "", "path to data directory")
	message := fs.String("message", "", "user message (prompt mode)")
	format := fs.String("format", "", "output format: json")
	transcriptWindow := fs.String("transcript-window", "", "recent transcript text for suppression (REQ-P4f-3)")
	claudeDir := fs.String("claude-dir", "", "path to .claude directory for cross-source suppression (REQ-P4f-2)")
	transcriptPath := fs.String("transcript-path", "", "transcript JSONL path (stop mode)")
	sessionID := fs.String("session-id", "", "session ID (stop mode)")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("surface: %w", parseErr)
	}

	defaultErr := applyDataDirDefault(dataDir)
	if defaultErr != nil {
		return fmt.Errorf("surface: %w", defaultErr)
	}

	if *mode == "" {
		return errSurfaceMissingFlags
	}

	if *mode == "stop" {
		if *transcriptPath == "" {
			return errSurfaceStopNoTranscript
		}

		assistantText, deltaErr := extractAssistantDelta(*dataDir, *transcriptPath, *sessionID)
		if deltaErr != nil {
			return fmt.Errorf("surface: %w", deltaErr)
		}

		if assistantText == "" {
			return nil
		}

		*mode = surface.ModePrompt
		*message = assistantText
	}

	currentProjectSlug := filepath.Base(*dataDir)

	opts := surface.Options{
		Mode:               *mode,
		DataDir:            *dataDir,
		Message:            *message,
		Format:             *format,
		TranscriptWindow:   *transcriptWindow,
		CurrentProjectSlug: currentProjectSlug,
	}

	retriever := retrieve.New()
	recorder := track.NewRecorder()
	surfLogger := surfacinglog.New(*dataDir)

	ctx := context.Background()

	memories, memErr := retriever.ListMemories(ctx, *dataDir)
	if memErr != nil {
		return fmt.Errorf("surface: listing memories: %w", memErr)
	}

	effAdapter := &effectivenessAdapter{stats: effectiveness.FromMemories(memories)}

	surfacerOpts := []surface.SurfacerOption{
		surface.WithTracker(recorder),
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

	surfacerOpts = append(surfacerOpts, surface.WithPolicyPath(filepath.Join(*dataDir, "policy.toml")))

	surfacer := surface.New(retriever, surfacerOpts...)

	return surfacer.Run(ctx, stdout, opts)
}

// sharedKeywords extracts keywords present in both the survivor and at least one absorbed memory.
func sharedKeywords(
	memMap map[string]*memory.Stored,
	survivorPath string,
	absorbedPaths []string,
) []string {
	survivor := memMap[survivorPath]
	if survivor == nil {
		return nil
	}

	survKW := make(map[string]struct{}, len(survivor.Keywords))
	for _, kw := range survivor.Keywords {
		survKW[strings.ToLower(kw)] = struct{}{}
	}

	shared := make(map[string]struct{})

	for _, absPath := range absorbedPaths {
		absorbed := memMap[absPath]
		if absorbed == nil {
			continue
		}

		for _, kw := range absorbed.Keywords {
			lower := strings.ToLower(kw)
			if _, ok := survKW[lower]; ok {
				shared[kw] = struct{}{}
			}
		}
	}

	result := make([]string, 0, len(shared))
	for kw := range shared {
		result = append(result, kw)
	}

	sort.Strings(result)

	return result
}

// titleOrPath returns the memory title if available, otherwise the path.
func titleOrPath(memMap map[string]*memory.Stored, path string) string {
	if mem := memMap[path]; mem != nil && mem.Title != "" {
		return mem.Title
	}

	return path
}

func truncateTitle(title string) string {
	if len(title) <= maxTitleLength {
		return title
	}

	return title[:maxTitleLength-1] + "…"
}
