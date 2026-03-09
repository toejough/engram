// Package cli implements the engram command-line interface (ARCH-6).
package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"engram/internal/audit"
	"engram/internal/automate"
	"engram/internal/classify"
	sessionctx "engram/internal/context"
	"engram/internal/correct"
	"engram/internal/creationlog"
	"engram/internal/dedup"
	"engram/internal/effectiveness"
	"engram/internal/evaluate"
	"engram/internal/extract"
	"engram/internal/instruct"
	"engram/internal/learn"
	"engram/internal/maintain"
	"engram/internal/memory"
	regpkg "engram/internal/registry"
	"engram/internal/remind"
	"engram/internal/render"
	"engram/internal/retrieve"
	reviewpkg "engram/internal/review"
	"engram/internal/surface"
	"engram/internal/surfacinglog"
	"engram/internal/tomlwriter"
	"engram/internal/track"
	"engram/internal/transcript"
)

// Exported variables.
var (
	AnthropicAPIURL = "https://api.anthropic.com/v1/messages" //nolint:gochecknoglobals // test-overridable endpoint
)

// RenderEvaluateResult writes the evaluation summary to w (T-119).
// No output is written when outcomes is empty.
func RenderEvaluateResult(w io.Writer, outcomes []evaluate.Outcome) {
	if len(outcomes) == 0 {
		return
	}

	var followed, contradicted, ignored int

	for _, outcome := range outcomes {
		switch outcome.Outcome {
		case "followed":
			followed++
		case "contradicted":
			contradicted++
		case "ignored":
			ignored++
		}
	}

	_, _ = fmt.Fprintf(w,
		"[engram] Evaluated %d memories: %d followed, %d contradicted, %d ignored.\n",
		len(outcomes), followed, contradicted, ignored)
}

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
	case "audit":
		return runAudit(subArgs, stdout, stderr, stdin)
	case "automate":
		return runAutomate(subArgs, stdout)
	case "correct":
		return runCorrect(subArgs, stdout)
	case "evaluate":
		return runEvaluate(subArgs, stdout, stderr, stdin)
	case "review":
		return runReview(subArgs, stdout)
	case "maintain":
		return runMaintain(subArgs, stdout)
	case "surface":
		return runSurface(subArgs, stdout)
	case "learn":
		return runLearn(subArgs, stderr, stdin)
	case "remind":
		return runRemind(subArgs, stdout)
	case "instruct":
		return runInstructAudit(subArgs, stdout)
	case "context-update":
		return runContextUpdate(subArgs)
	case "registry":
		return runRegistry(subArgs, stdout)
	default:
		return fmt.Errorf("%w: %s", errUnknownCommand, cmd)
	}
}

// RunEvaluate implements the evaluate subcommand with injectable evaluate options.
// token is the Anthropic API token; empty string triggers the no-token error path.
// Extra opts are appended after the default LLM caller and can override it (for testing).
func RunEvaluate(
	args []string,
	token string,
	stdout, stderr io.Writer,
	stdin io.Reader,
	opts ...evaluate.Option,
) error {
	fs := flag.NewFlagSet("evaluate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("evaluate: %w", parseErr)
	}

	if *dataDir == "" {
		return errEvaluateMissingFlags
	}

	if token == "" {
		_, _ = fmt.Fprintln(stderr,
			"[engram] Error: evaluation skipped — no API token configured")

		return nil
	}

	transcriptBytes, err := io.ReadAll(stdin)
	if err != nil {
		return fmt.Errorf("evaluate: reading stdin: %w", err)
	}

	// Default LLM caller uses real Anthropic API; opts can override for testing.
	allOpts := append(
		[]evaluate.Option{evaluate.WithLLMCaller(makeAnthropicCaller(token))},
		opts...,
	)

	evaluator := evaluate.New(*dataDir, allOpts...)
	ctx := context.Background()

	transcriptLines := strings.Split(string(transcriptBytes), "\n")
	strippedLines := sessionctx.Strip(transcriptLines)
	strippedTranscript := strings.Join(strippedLines, "\n")

	outcomes, err := evaluator.Evaluate(ctx, strippedTranscript)
	if err != nil {
		return fmt.Errorf("evaluate: %w", err)
	}

	RenderEvaluateResult(stdout, outcomes)

	return nil
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
	learner.SetCreationLogger(creationlog.NewLogWriter())

	registry := openRegistry(*dataDir)
	learner.SetRegistryRegistrar(&learnRegistryAdapter{reg: registry})

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
func RunMaintain(
	args []string,
	token string,
	stdout io.Writer,
	opts ...maintain.Option,
) error {
	fs := flag.NewFlagSet("maintain", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("maintain: %w", parseErr)
	}

	if *dataDir == "" {
		return errMaintainMissingFlags
	}

	evalDir := filepath.Join(*dataDir, "evaluations")

	stats, err := effectiveness.New(evalDir).Aggregate()
	if err != nil {
		return fmt.Errorf(
			"maintain: aggregating effectiveness: %w", err,
		)
	}

	if len(stats) == 0 {
		_, _ = fmt.Fprint(stdout, "[]\n")

		return nil
	}

	tracking := buildTrackingMap(*dataDir)
	classified := reviewpkg.Classify(stats, tracking)

	memoryMap, listErr := buildMemoryMap(*dataDir)
	if listErr != nil {
		return fmt.Errorf("maintain: %w", listErr)
	}

	allOpts := make([]maintain.Option, 0, len(opts)+1)
	if token != "" {
		allOpts = append(allOpts,
			maintain.WithLLMCaller(makeAnthropicCaller(token)))
	}

	allOpts = append(allOpts, opts...)

	ctx := context.Background()
	generator := maintain.New(allOpts...)
	proposals := generator.Generate(ctx, classified, memoryMap)

	// UC-21: Run escalation engine on leech memories (ARCH-50).
	leeches := buildEscalationMemories(classified, memoryMap)
	if len(leeches) > 0 {
		engine := maintain.NewEscalationEngine(maintain.EffData{}, nil)

		escalations, escErr := engine.Analyze(leeches)
		if escErr == nil {
			for idx := range escalations {
				escJSON, marshalErr := maintain.MarshalProposal(escalations[idx])
				if marshalErr == nil {
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
	}

	//nolint:wrapcheck // thin JSON encoding at CLI boundary
	return json.NewEncoder(stdout).Encode(proposals)
}

// RunReview implements the review subcommand: aggregates effectiveness stats,
// retrieves memory tracking data, classifies memories, and renders the matrix.
func RunReview(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("review", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("review: %w", parseErr)
	}

	if *dataDir == "" {
		return errReviewMissingFlags
	}

	evalDir := filepath.Join(*dataDir, "evaluations")

	stats, err := effectiveness.New(evalDir).Aggregate()
	if err != nil {
		return fmt.Errorf("review: aggregating effectiveness: %w", err)
	}

	if len(stats) == 0 {
		_, _ = fmt.Fprintln(stdout, "[engram] No evaluation data found.")

		return nil
	}

	tracking := buildTrackingMap(*dataDir)

	classified := reviewpkg.Classify(stats, tracking)
	reviewpkg.Render(classified, stdout)

	return nil
}

// unexported constants.
const (
	anthropicVersion           = "2023-06-01"
	contextSummarizationPrompt = "Update this task-focused working summary. " +
		"Focus on what's being worked on, decisions made, progress, and open questions. " +
		"Not a dissertation — just what's relevant for resuming work. " +
		"Do NOT include discovered constraints or patterns (those are captured as memories)."
	evaluateMaxTokens = 1024
	haikuModel        = "claude-haiku-4-5-20251001"
	maxTranscriptTok  = 2000
	minArgs           = 2
)

// unexported variables.
var (
	errContextUpdateMissingFlags = errors.New(
		"context-update: --transcript-path, --session-id," +
			" and --data-dir required",
	)
	errCorrectMissingFlags = errors.New(
		"correct: --message and --data-dir required",
	)
	errEvaluateMissingFlags = errors.New("evaluate: --data-dir required")
	errLearnMissingFlags    = errors.New("learn: --data-dir required")
	errMaintainMissingFlags = errors.New("maintain: --data-dir required")
	errNilAPIResponse       = errors.New("calling Anthropic API: nil response")
	errNoContentBlocks      = errors.New("API response contained no content blocks")
	errReviewMissingFlags   = errors.New("review: --data-dir required")
	errSurfaceMissingFlags  = errors.New(
		"surface: --mode and --data-dir required",
	)
	errInstructMissingFlags = errors.New(
		"instruct audit: --data-dir required",
	)
	errRemindMissingFlags   = errors.New("remind: --data-dir required")
	errUnknownCommand       = errors.New("unknown command")
	errAuditMissingFlags    = errors.New("audit: --data-dir required")
	errRegistryMissingFlags = errors.New(
		"registry init: --data-dir required",
	)
	errRegistryUnknownSub = errors.New(
		"registry: unknown subcommand (expected: init, register-source)",
	)
	errRegisterSourceMissingFlags = errors.New(
		"registry register-source: --type and --path and --data-dir required",
	)
	errUsage = errors.New(
		"usage: engram <audit|correct|surface|learn|evaluate" +
			"|review|maintain|remind|instruct|automate" +
			"|context-update|registry> [flags]",
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

// effectivenessAdapter bridges effectiveness.Computer to surface.EffectivenessComputer.
type effectivenessAdapter struct {
	computer *effectiveness.Computer
}

func (a *effectivenessAdapter) Aggregate() (map[string]surface.EffectivenessStat, error) {
	stats, err := a.computer.Aggregate()
	if err != nil {
		return nil, err
	}

	result := make(map[string]surface.EffectivenessStat, len(stats))

	for memPath, stat := range stats {
		total := stat.FollowedCount + stat.ContradictedCount + stat.IgnoredCount
		result[memPath] = surface.EffectivenessStat{
			SurfacedCount:      total,
			EffectivenessScore: stat.EffectivenessScore,
		}
	}

	return result, nil
}

// haikuClientAdapter implements sessionctx.HaikuClient using the Anthropic API.
type haikuClientAdapter struct {
	caller func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error)
}

func (h *haikuClientAdapter) Summarize(
	ctx context.Context,
	previousSummary, delta string,
) (string, error) {
	userPrompt := delta
	if previousSummary != "" {
		userPrompt = "Previous summary:\n" + previousSummary +
			"\n\nNew transcript:\n" + delta
	}

	return h.caller(ctx, haikuModel, contextSummarizationPrompt, userPrompt)
}

type osDirCreator struct{}

func (d *osDirCreator) MkdirAll(path string) error {
	const dirPerms = 0o755

	return os.MkdirAll(path, dirPerms) //nolint:wrapcheck // thin I/O adapter
}

// I/O adapters for context package DI interfaces.

type osFileReader struct{}

func (r *osFileReader) Read(path string) ([]byte, error) {
	return os.ReadFile(path) //nolint:gosec,wrapcheck // thin I/O adapter
}

type osFileWriter struct{}

func (w *osFileWriter) Write(path string, content []byte) error {
	const filePerms = 0o644

	return os.WriteFile(path, content, filePerms) //nolint:wrapcheck // thin I/O adapter
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

type osRenamer struct{}

func (r *osRenamer) Rename(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath) //nolint:wrapcheck // thin I/O adapter
}

type realTimestamper struct{}

func (t *realTimestamper) Now() time.Time {
	return time.Now()
}

// buildTrackingMap retrieves memories and builds a path→TrackingData map.
// Returns empty map if memories cannot be read.
func buildMemoryMap(
	dataDir string,
) (map[string]*memory.Stored, error) {
	retriever := retrieve.New()
	ctx := context.Background()

	memories, err := retriever.ListMemories(ctx, dataDir)
	if err != nil {
		return nil, fmt.Errorf("listing memories: %w", err)
	}

	memMap := make(map[string]*memory.Stored, len(memories))
	for _, mem := range memories {
		memMap[mem.FilePath] = mem
	}

	return memMap, nil
}

func buildTrackingMap(dataDir string) map[string]reviewpkg.TrackingData {
	retriever := retrieve.New()
	ctx := context.Background()

	memories, err := retriever.ListMemories(ctx, dataDir)
	if err != nil {
		return map[string]reviewpkg.TrackingData{}
	}

	tracking := make(map[string]reviewpkg.TrackingData, len(memories))

	for _, mem := range memories {
		tracking[mem.FilePath] = reviewpkg.TrackingData{}
	}

	return tracking
}

// buildEscalationMemories extracts leech memories for the escalation engine (UC-21, ARCH-50).
func buildEscalationMemories(
	classified []reviewpkg.ClassifiedMemory,
	memoryMap map[string]*memory.Stored,
) []maintain.EscalationMemory {
	leeches := make([]maintain.EscalationMemory, 0)

	for _, cm := range classified {
		if cm.Quadrant != reviewpkg.Leech {
			continue
		}

		stored := memoryMap[cm.Name]
		content := ""

		if stored != nil {
			content = stored.Content
		}

		leeches = append(leeches, maintain.EscalationMemory{
			Path:          cm.Name,
			Content:       content,
			Effectiveness: cm.EffectivenessScore,
		})
	}

	return leeches
}

// callAnthropicAPI makes a single call to the Anthropic messages API and returns the text response.
func callAnthropicAPI(
	ctx context.Context,
	client *http.Client,
	token, model, systemPrompt, userPrompt string,
) (string, error) {
	reqBody := anthropicRequest{
		Model:     model,
		MaxTokens: evaluateMaxTokens,
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

func runAutomate(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("automate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("automate: %w", parseErr)
	}

	if *dataDir == "" {
		return errors.New("automate: --data-dir is required")
	}

	retriever := retrieve.New()

	automator := &automate.Automator{
		MemoryLoader: func(dir string) ([]automate.Memory, error) {
			stored, err := retriever.ListMemories(context.Background(), dir)
			if err != nil {
				return nil, err
			}

			return automate.MemoriesFromStored(stored), nil
		},
		// LLMCaller is nil — no API token wiring yet (T-237 path).
	}

	proposals, err := automator.Run(context.Background(), *dataDir)
	if err != nil {
		return err
	}

	encodeErr := json.NewEncoder(stdout).Encode(proposals)
	if encodeErr != nil {
		return fmt.Errorf("automate: encoding JSON: %w", encodeErr)
	}

	return nil
}

func runAudit(args []string, stdout, stderr io.Writer, stdin io.Reader) error {
	fs := flag.NewFlagSet("audit", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")
	timestamp := fs.String("timestamp", "", "audit timestamp (ISO 8601)")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("audit: %w", parseErr)
	}

	if *dataDir == "" {
		return errAuditMissingFlags
	}

	token := os.Getenv("ENGRAM_API_TOKEN")
	if token == "" {
		_, _ = fmt.Fprintln(stderr, "audit: API token missing or invalid, skipping audit")

		return errors.New("audit: API token missing")
	}

	transcriptBytes, err := io.ReadAll(stdin)
	if err != nil {
		return fmt.Errorf("audit: reading stdin: %w", err)
	}

	opts := []audit.Option{
		audit.WithLLMCaller(makeAnthropicCaller(token)),
	}

	if *timestamp != "" {
		parsed, parseTimeErr := time.Parse(time.RFC3339, *timestamp)
		if parseTimeErr == nil {
			opts = append(opts, audit.WithNow(func() time.Time { return parsed }))
		}
	}

	auditor := audit.New(*dataDir, opts...)
	ctx := context.Background()

	transcript := strings.TrimSpace(string(transcriptBytes))

	report, err := auditor.Run(ctx, transcript)
	if err != nil {
		return fmt.Errorf("audit: %w", err)
	}

	if report != nil {
		_, _ = fmt.Fprintf(stdout,
			"[engram] Audit: %d instructions, %d compliant, %d non-compliant.\n",
			report.TotalInstructionsAudited, report.Compliant, report.NonCompliant)
	}

	return nil
}

func runContextUpdate(args []string) error {
	fs := flag.NewFlagSet("context-update", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	transcriptPath := fs.String(
		"transcript-path", "", "path to session transcript",
	)
	sessionID := fs.String("session-id", "", "session identifier")
	dataDir := fs.String("data-dir", "", "path to data directory")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("context-update: %w", parseErr)
	}

	if *transcriptPath == "" || *sessionID == "" || *dataDir == "" {
		return errContextUpdateMissingFlags
	}

	contextFilePath := filepath.Join(
		*dataDir, "session-context.md",
	)

	reader := &osFileReader{}
	writer := &osFileWriter{}
	dirCreator := &osDirCreator{}
	renamer := &osRenamer{}
	clock := &realTimestamper{}

	delta := sessionctx.NewDeltaReader(reader)

	token := os.Getenv("ENGRAM_API_TOKEN")

	var haikuClient sessionctx.HaikuClient
	if token != "" {
		haikuClient = &haikuClientAdapter{
			caller: makeAnthropicCaller(token),
		}
	}

	summarizer := sessionctx.NewSummarizer(haikuClient)
	file := sessionctx.NewSessionFile(
		reader, writer, dirCreator, renamer, clock,
	)

	orchestrator := sessionctx.NewOrchestrator(
		delta, summarizer, file,
	)

	return orchestrator.Update(
		context.Background(),
		*transcriptPath,
		*sessionID,
		contextFilePath,
	)
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

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("correct: %w", parseErr)
	}

	if *message == "" || *dataDir == "" {
		return errCorrectMissingFlags
	}

	// Read transcript context if available (os.ReadFile wired at the edge)
	reader := transcript.New(os.ReadFile)

	transcriptCtx, _ := reader.ReadRecent(
		*transcriptPath, maxTranscriptTok,
	)

	token := os.Getenv("ENGRAM_API_TOKEN")
	classifier := classify.New(token, &http.Client{})
	writer := tomlwriter.New()
	renderer := render.New()

	corrector := correct.New(classifier, writer, renderer, *dataDir)
	ctx := context.Background()

	output, err := corrector.Run(ctx, *message, transcriptCtx)
	if err != nil {
		return fmt.Errorf("correct: %w", err)
	}

	if output != "" {
		_, _ = fmt.Fprint(stdout, output)
	}

	return nil
}

func runEvaluate(args []string, stdout, stderr io.Writer, stdin io.Reader) error {
	// Parse data-dir early to wire registry (UC-23).
	fs := flag.NewFlagSet("evaluate-peek", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "")
	_ = fs.Parse(args)

	var opts []evaluate.Option
	if *dataDir != "" {
		registry := openRegistry(*dataDir)
		opts = append(opts, evaluate.WithRegistry(
			&evaluateRegistryAdapter{reg: registry},
		))
	}

	return RunEvaluate(args, os.Getenv("ENGRAM_API_TOKEN"), stdout, stderr, stdin, opts...)
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

func runRemind(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("remind", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")
	filePath := fs.String("file-path", "", "file path from tool call")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("remind: %w", parseErr)
	}

	if *dataDir == "" {
		return errRemindMissingFlags
	}

	configReader := &osRemindConfigReader{dataDir: *dataDir}
	loader := &osMemoryLoader{dataDir: *dataDir}
	transcriptReader := &noopTranscriptReader{}
	surfLogger := surfacinglog.New(*dataDir)

	reminder := remind.New(configReader, loader, transcriptReader,
		remind.WithSurfacingLogger(surfLogger),
	)

	ctx := context.Background()
	input := remind.ToolCallInput{
		ToolName: "",
		FilePath: *filePath,
	}

	result, err := reminder.Run(ctx, input)
	if err != nil {
		return fmt.Errorf("remind: %w", err)
	}

	if result != "" {
		_, _ = fmt.Fprint(stdout, result)
	}

	return nil
}

// osRemindConfigReader reads reminders.toml from the data directory.
type osRemindConfigReader struct {
	dataDir string
}

func (r *osRemindConfigReader) ReadConfig() (map[string][]string, error) {
	path := filepath.Join(r.dataDir, "reminders.toml")

	data, err := os.ReadFile(path) //nolint:gosec // thin I/O adapter
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("reading reminders.toml: %w", err)
	}

	return parseRemindersToml(data)
}

// parseRemindersToml parses a simple TOML config: ["*.go"]\ninstructions = ["id1", "id2"].
func parseRemindersToml(data []byte) (map[string][]string, error) {
	result := make(map[string][]string)

	lines := strings.Split(string(data), "\n")
	var currentPattern string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Section header: ["*.go"]
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			pattern := strings.Trim(line, "[]\"")
			currentPattern = pattern

			continue
		}

		// Key-value: instructions = ["id1", "id2"]
		if strings.HasPrefix(line, "instructions") && currentPattern != "" {
			_, valueStr, found := strings.Cut(line, "=")
			if !found {
				continue
			}

			valueStr = strings.TrimSpace(valueStr)
			valueStr = strings.Trim(valueStr, "[]")

			parts := strings.Split(valueStr, ",")
			ids := make([]string, 0, len(parts))

			for _, part := range parts {
				id := strings.TrimSpace(strings.Trim(strings.TrimSpace(part), "\""))
				if id != "" {
					ids = append(ids, id)
				}
			}

			result[currentPattern] = ids
		}
	}

	return result, nil
}

// osMemoryLoader loads a memory's principle by instruction ID from the data directory.
type osMemoryLoader struct {
	dataDir string
}

func (l *osMemoryLoader) LoadPrinciple(_ context.Context, instructionID string) (string, error) {
	retriever := retrieve.New()
	ctx := context.Background()

	memories, err := retriever.ListMemories(ctx, l.dataDir)
	if err != nil {
		return "", fmt.Errorf("loading memories: %w", err)
	}

	// Match by filename slug (without .toml extension).
	for _, mem := range memories {
		slug := strings.TrimSuffix(filepath.Base(mem.FilePath), ".toml")
		if slug == instructionID && mem.Principle != "" {
			return mem.Principle, nil
		}
	}

	return "", nil
}

// noopTranscriptReader returns empty transcript (hook passes transcript via stdin in future).
type noopTranscriptReader struct{}

func (r *noopTranscriptReader) ReadRecent(_ int) (string, error) {
	return "", nil
}

func runLearn(args []string, stderr io.Writer, stdin io.Reader) error {
	return RunLearn(args, os.Getenv("ENGRAM_API_TOKEN"), stderr, stdin, nil)
}

func runMaintain(args []string, stdout io.Writer) error {
	return RunMaintain(args, os.Getenv("ENGRAM_API_TOKEN"), stdout)
}

func runReview(args []string, stdout io.Writer) error {
	return RunReview(args, stdout)
}

func runSurface(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("surface", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	mode := fs.String(
		"mode", "", "surface mode: session-start, prompt, tool",
	)
	dataDir := fs.String("data-dir", "", "path to data directory")
	message := fs.String("message", "", "user message (prompt mode)")
	toolName := fs.String("tool-name", "", "tool name (tool mode)")
	toolInput := fs.String(
		"tool-input", "", "tool input JSON (tool mode)",
	)
	format := fs.String("format", "", "output format: json")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("surface: %w", parseErr)
	}

	if *mode == "" || *dataDir == "" {
		return errSurfaceMissingFlags
	}

	retriever := retrieve.New()
	recorder := track.NewRecorder()
	logReader := creationlog.NewLogReader()
	surfLogger := surfacinglog.New(*dataDir)
	evalDir := filepath.Join(*dataDir, "evaluations")
	effAdapter := &effectivenessAdapter{computer: effectiveness.New(evalDir)}

	registry := openRegistry(*dataDir)

	surfacer := surface.New(
		retriever,
		surface.WithTracker(recorder),
		surface.WithLogReader(logReader),
		surface.WithSurfacingLogger(surfLogger),
		surface.WithEffectiveness(effAdapter),
		surface.WithRegistry(&surfaceRegistryAdapter{reg: registry}),
	)
	ctx := context.Background()

	return surfacer.Run(ctx, stdout, surface.Options{
		Mode:      *mode,
		DataDir:   *dataDir,
		Message:   *message,
		ToolName:  *toolName,
		ToolInput: *toolInput,
		Format:    *format,
	})
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
		ReadFile:  osReadFileFunc,
		GlobFiles: filepath.Glob,
		EffData:   map[string]float64{},
	}

	token := os.Getenv("ENGRAM_API_TOKEN")

	var llmCaller func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error)
	if token != "" {
		llmCaller = makeAnthropicCaller(token)
	}

	auditor := &instruct.Auditor{
		Scanner:   scanner,
		LLMCaller: llmCaller,
	}

	ctx := context.Background()

	report, err := auditor.Run(ctx, *dataDir, *projectDir)
	if err != nil {
		return fmt.Errorf("instruct audit: %w", err)
	}

	//nolint:wrapcheck // thin JSON encoding at CLI boundary
	return json.NewEncoder(stdout).Encode(report)
}

func runRegistry(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errRegistryUnknownSub
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "init":
		return runRegistryInit(subArgs, stdout)
	case "register-source":
		return runRegistryRegisterSource(subArgs, stdout)
	default:
		return errRegistryUnknownSub
	}
}

// RunRegistryInit implements the registry init subcommand with injectable
// backfill config components. opts override default I/O adapters for testing.
func RunRegistryInit(
	args []string,
	stdout io.Writer,
	opts ...regpkg.JSONLOption,
) error {
	fs := flag.NewFlagSet("registry init", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")
	dryRun := fs.Bool("dry-run", false, "print entries without writing")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("registry init: %w", parseErr)
	}

	if *dataDir == "" {
		return errRegistryMissingFlags
	}

	config := buildBackfillConfig(*dataDir)

	entries, err := regpkg.Backfill(config)
	if err != nil {
		return fmt.Errorf("registry init: %w", err)
	}

	if *dryRun {
		_, _ = fmt.Fprintf(stdout,
			"[engram] Registry init (dry-run): %d entries\n",
			len(entries))

		for _, entry := range entries {
			_, _ = fmt.Fprintf(stdout,
				"  %s (%s) surfaced=%d evals=%d\n",
				entry.ID, entry.SourceType,
				entry.SurfacedCount, entry.Evaluations.Total())
		}

		return nil
	}

	registryPath := filepath.Join(*dataDir, registryFilename)

	allOpts := []regpkg.JSONLOption{
		regpkg.WithReader(osReadFileFunc),
		regpkg.WithWriter(osWriteFileFunc),
	}
	allOpts = append(allOpts, opts...)

	store := regpkg.NewJSONLStore(registryPath, allOpts...)

	bulkErr := store.BulkLoad(entries)
	if bulkErr != nil {
		return fmt.Errorf("registry init: writing: %w", bulkErr)
	}

	_, _ = fmt.Fprintf(stdout,
		"[engram] Registry initialized: %d entries written to %s\n",
		len(entries), registryPath)

	return nil
}

func runRegistryInit(args []string, stdout io.Writer) error {
	return RunRegistryInit(args, stdout)
}

func runRegistryRegisterSource(args []string, stdout io.Writer) error {
	return RunRegistryRegisterSource(args, stdout, osReadFileFunc)
}

// ReadFileFunc reads a file by path, injected for testability.
type ReadFileFunc func(path string) ([]byte, error)

// RunRegistryRegisterSource implements the registry register-source subcommand.
// readFile is injected for testability (DI).
func RunRegistryRegisterSource(
	args []string,
	stdout io.Writer,
	readFile ReadFileFunc,
	opts ...regpkg.JSONLOption,
) error {
	fs := flag.NewFlagSet("registry register-source", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")
	sourceType := fs.String("type", "", "source type (claude-md, memory-md, rule, skill)")
	sourcePath := fs.String("path", "", "path to source file or name")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("registry register-source: %w", parseErr)
	}

	if *dataDir == "" || *sourceType == "" || *sourcePath == "" {
		return errRegisterSourceMissingFlags
	}

	content, err := readFile(*sourcePath)
	if err != nil {
		return fmt.Errorf("registry register-source: reading source: %w", err)
	}

	extractor, err := buildExtractor(*sourceType, *sourcePath, string(content))
	if err != nil {
		return fmt.Errorf("registry register-source: %w", err)
	}

	entries, err := extractor.Extract()
	if err != nil {
		return fmt.Errorf("registry register-source: extracting: %w", err)
	}

	if len(entries) == 0 {
		_, _ = fmt.Fprintln(stdout,
			"[engram] No instructions extracted from source.")

		return nil
	}

	registryPath := filepath.Join(*dataDir, registryFilename)

	allOpts := []regpkg.JSONLOption{
		regpkg.WithReader(osReadFileFunc),
		regpkg.WithWriter(osWriteFileFunc),
	}
	allOpts = append(allOpts, opts...)

	store := regpkg.NewJSONLStore(registryPath, allOpts...)

	var registered int

	for _, entry := range entries {
		regErr := store.Register(entry)
		if regErr != nil {
			if errors.Is(regErr, regpkg.ErrDuplicateID) {
				continue
			}

			return fmt.Errorf("registry register-source: registering %s: %w",
				entry.ID, regErr)
		}

		registered++
	}

	_, _ = fmt.Fprintf(stdout,
		"[engram] Registered %d instructions from %s (%s)\n",
		registered, *sourcePath, *sourceType)

	return nil
}

// buildExtractor creates the appropriate extractor for the given source type.
func buildExtractor(
	sourceType, sourcePath, content string,
) (regpkg.InstructionExtractor, error) {
	switch sourceType {
	case "claude-md":
		return regpkg.ClaudeMDExtractor{
			Content:    content,
			SourcePath: filepath.Base(sourcePath),
		}, nil
	case "memory-md":
		return regpkg.MemoryMDExtractor{
			Content:    content,
			SourcePath: filepath.Base(sourcePath),
		}, nil
	case "rule":
		return regpkg.RuleExtractor{
			Filename: filepath.Base(sourcePath),
			Content:  content,
		}, nil
	case "skill":
		return regpkg.SkillExtractor{
			SkillName: filepath.Base(sourcePath),
			Content:   content,
		}, nil
	default:
		return nil, fmt.Errorf("unknown source type: %s", sourceType)
	}
}

func buildBackfillConfig(dataDir string) regpkg.BackfillConfig {
	return regpkg.BackfillConfig{
		Scanner:      &osMemoryScanner{dataDir: dataDir},
		SurfacingLog: &osSurfacingLogReader{dataDir: dataDir},
		CreationLog:  &osCreationLogReader{dataDir: dataDir},
		Evaluations:  &osEvaluationsReader{dataDir: dataDir},
		Now:          time.Now(),
	}
}

// osMemoryScanner scans memory TOML files from the data directory.
type osMemoryScanner struct {
	dataDir string
}

func (s *osMemoryScanner) ScanMemories() ([]regpkg.ScannedMemory, error) {
	retriever := retrieve.New()
	ctx := context.Background()

	memories, err := retriever.ListMemories(ctx, s.dataDir)
	if err != nil {
		return nil, fmt.Errorf("listing memories: %w", err)
	}

	result := make([]regpkg.ScannedMemory, 0, len(memories))
	for _, mem := range memories {
		result = append(result, regpkg.ScannedMemory{
			FilePath:  mem.FilePath,
			Title:     mem.Title,
			Content:   mem.Content,
			UpdatedAt: mem.UpdatedAt,
		})
	}

	return result, nil
}

// osSurfacingLogReader reads and aggregates the surfacing log.
type osSurfacingLogReader struct {
	dataDir string
}

func (r *osSurfacingLogReader) AggregateSurfacing() (
	map[string]regpkg.SurfacingData, error,
) {
	logger := surfacinglog.New(r.dataDir)

	events, err := logger.ReadAndClear()
	if err != nil {
		return nil, fmt.Errorf("reading surfacing log: %w", err)
	}

	result := make(map[string]regpkg.SurfacingData, len(events))

	for _, event := range events {
		data := result[event.MemoryPath]
		data.Count++

		surfTime := event.SurfacedAt
		if data.LastSurfaced == nil || surfTime.After(*data.LastSurfaced) {
			data.LastSurfaced = &surfTime
		}

		result[event.MemoryPath] = data
	}

	return result, nil
}

// osCreationLogReader reads creation timestamps.
type osCreationLogReader struct {
	dataDir string
}

func (r *osCreationLogReader) CreationTimes() (map[string]time.Time, error) {
	reader := creationlog.NewLogReader()

	entries, err := reader.ReadAndClear(r.dataDir)
	if err != nil {
		return nil, fmt.Errorf("reading creation log: %w", err)
	}

	result := make(map[string]time.Time, len(entries))

	for _, entry := range entries {
		parsed, parseErr := time.Parse(time.RFC3339, entry.Timestamp)
		if parseErr != nil {
			continue
		}

		result[entry.Filename] = parsed
	}

	return result, nil
}

// osEvaluationsReader aggregates evaluations from JSONL files.
type osEvaluationsReader struct {
	dataDir string
}

func (r *osEvaluationsReader) AggregateEvaluations() (
	map[string]regpkg.EvaluationCounters, error,
) {
	evalDir := filepath.Join(r.dataDir, "evaluations")
	computer := effectiveness.New(evalDir)

	stats, err := computer.Aggregate()
	if err != nil {
		return nil, fmt.Errorf("aggregating evaluations: %w", err)
	}

	result := make(map[string]regpkg.EvaluationCounters, len(stats))

	for memPath, stat := range stats {
		result[memPath] = regpkg.EvaluationCounters{
			Followed:     stat.FollowedCount,
			Contradicted: stat.ContradictedCount,
			Ignored:      stat.IgnoredCount,
		}
	}

	return result, nil
}

// osReadFileFunc wraps os.ReadFile for DI injection.
func osReadFileFunc(path string) ([]byte, error) {
	return os.ReadFile(path) //nolint:gosec,wrapcheck // thin I/O adapter
}

// osWriteFileFunc wraps os.WriteFile for DI injection.
func osWriteFileFunc(path string, content []byte) error {
	const filePermsRW = 0o644

	return os.WriteFile(path, content, filePermsRW) //nolint:wrapcheck,gosec // thin I/O adapter
}

// registryFilename is the default name for the instruction registry file.
const registryFilename = "instruction-registry.jsonl"

// surfaceRegistryAdapter bridges surface.RegistryRecorder to registry.Registry.
type surfaceRegistryAdapter struct {
	reg regpkg.Registry
}

func (a *surfaceRegistryAdapter) RecordSurfacing(id string) error {
	return a.reg.RecordSurfacing(id)
}

// evaluateRegistryAdapter bridges evaluate.RegistryRecorder to registry.Registry.
type evaluateRegistryAdapter struct {
	reg regpkg.Registry
}

func (a *evaluateRegistryAdapter) RecordEvaluation(id, outcome string) error {
	return a.reg.RecordEvaluation(id, regpkg.Outcome(outcome))
}

// learnRegistryAdapter bridges learn.RegistryRegistrar to registry.Registry.
type learnRegistryAdapter struct {
	reg regpkg.Registry
	now func() time.Time
}

func (a *learnRegistryAdapter) RegisterMemory(
	filePath, title, content string, now time.Time,
) error {
	entry := regpkg.InstructionEntry{
		ID:           filePath,
		SourceType:   "memory",
		SourcePath:   filePath,
		Title:        title,
		ContentHash:  contentHashForRegistry(content),
		RegisteredAt: now,
		UpdatedAt:    now,
	}

	return a.reg.Register(entry)
}

// contentHashForRegistry produces a SHA-256 hash consistent with registry.Backfill.
func contentHashForRegistry(content string) string {
	hash := sha256.Sum256([]byte(content))

	return fmt.Sprintf("%x", hash)
}

// openRegistry creates a JSONLStore for the given data directory.
// Returns nil if the registry file cannot be loaded (fire-and-forget).
func openRegistry(dataDir string) *regpkg.JSONLStore {
	registryPath := filepath.Join(dataDir, registryFilename)

	return regpkg.NewJSONLStore(
		registryPath,
		regpkg.WithReader(osReadFileFunc),
		regpkg.WithWriter(osWriteFileFunc),
	)
}
