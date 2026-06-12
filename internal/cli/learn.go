package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/luhmann"
	"github.com/toejough/engram/internal/transcript"
)

// EpisodeRange captures one existing episode's basename and transcript-range
// bounds (RFC3339), used to compute preceding-episode links for a new episode.
type EpisodeRange struct {
	Basename string
	Start    string
	End      string
}

// LearnArgs holds the parsed flags for the learn subcommand.
type LearnArgs struct {
	Type     string
	Slug     string
	Vault    string
	Target   string
	Position string
	Source   string
	Project  string
	Issue    string
	Tier     string

	// feedback / fact / episode all support related-note bullets
	Relations []string

	// feedback / fact / episode share these
	Situation string
	// feedback only
	Behavior string
	Impact   string
	Action   string
	// fact only
	Subject   string
	Predicate string
	Object    string
	// episode only
	Summary           string
	BoundaryRationale string
	TranscriptText    string
	TranscriptFiles   []string
	Sessions          []string
	TranscriptRange   string
	// Preceding holds the computed preceding-episode links for an episode
	// write. Populated by writeLearnUnderLock after the vault scan; nil for
	// fact/feedback and when no ListEpisodes dep is wired.
	Preceding []episodeLink
}

// LearnDeps holds injected dependencies for runLearn. All fields are
// required except Embedder / WriteSidecar / LogWarning, which together
// drive the auto-embed step. A nil Embedder skips auto-embed entirely
// (used by tests that don't exercise the embedding pipeline).
type LearnDeps struct {
	Now           func() time.Time
	Getenv        func(string) string
	StatDir       func(string) error
	InitVault     func(string) error
	ListIDs       func(vault string) ([]string, error)
	ListBasenames func(vault string) ([]string, error)
	ListEpisodes  func(vault string) ([]EpisodeRange, error)
	Lock          func(vault string) (release func(), err error)
	WriteNew      func(path string, data []byte) error
	Embedder      embed.Embedder
	WriteSidecar  func(path string, data []byte) error
	LogWarning    func(format string, args ...any)
}

// unexported constants.
const (
	backtickRune = '`'
	dateFormat   = "2006-01-02"
	envVaultPath = "ENGRAM_VAULT_PATH"
	// episodeFenceMin is the shortest backtick fence the ## Transcript block
	// may use. The actual fence is max(episodeFenceMin, longestBacktickRun+1)
	// so a transcript containing a ``` run still round-trips (the inner run
	// cannot close the outer fence) and ParseWikilinks suppresses any [[...]]
	// inside the fenced transcript (G5).
	episodeFenceMin       = 3
	episodeRelatedHeading = "## Related"
	// episodeSummaryHeading / episodeTranscriptHeading / episodeRelatedHeading
	// are the D6 episode-body section headings.
	episodeSummaryHeading    = "## Summary"
	episodeTranscriptHeading = "## Transcript"
	opencodeSessionPrefix    = "opencode://"
	// precedingActiveRationale / precedingPriorRationale are the rationale
	// strings rendered after a computed preceding-episode link.
	precedingActiveRationale = "preceding episode (active at this episode's start)"
	precedingPriorRationale  = "immediately preceding episode"
	tierL1                   = "L1"
	tierL2                   = "L2"
	tierL3                   = "L3"
	typeEpisode              = "episode"
	typeFact                 = "fact"
	typeFeedback             = "feedback"
)

// unexported variables.
var (
	errEpisodeBodySourceBoth = errors.New(
		"episode: --from-transcript-range and --transcript-text are mutually exclusive",
	)
	errEpisodeBodySourceRequired = errors.New(
		"episode: exactly one of --from-transcript-range or --transcript-text is required",
	)
	errEpisodeBoundaryRequired = errors.New("episode: --boundary-rationale is required")
	errEpisodeFromRangeFmt     = errors.New(
		"episode: --from-transcript-range must be <session-id>:<RFC3339-start>..<RFC3339-end>",
	)
	errEpisodeFromRangeOrder = errors.New(
		"episode: --from-transcript-range start must be before end",
	)
	errEpisodeSessionEmpty       = errors.New("episode: --session must not be empty")
	errEpisodeSessionRequired    = errors.New("episode: at least one --session is required")
	errEpisodeSituationRequired  = errors.New("episode: --situation is required")
	errEpisodeSummaryRequired    = errors.New("episode: --summary is required")
	errEpisodeTranscriptRangeFmt = errors.New(
		"episode: --transcript-range must be <RFC3339>..<RFC3339>",
	)
	errEpisodeTranscriptRangeOrder = errors.New(
		"episode: --transcript-range start must be before end",
	)
	errEpisodeTranscriptRangeReq = errors.New("episode: --transcript-range is required")
	errFactSituationRequired     = errors.New("fact: --situation is required")
	errFeedbackSituationRequired = errors.New("feedback: --situation is required")
	errIssueIDInvalid            = errors.New("issue must be non-empty with no whitespace")
	errLearnBadTier              = errors.New("tier must be L1, L2, or L3")
	errLearnUnknownType          = errors.New("learn: type must be feedback, fact, or episode")
	errProjectSlugInvalid        = errors.New("project slug must match [a-z0-9-]+")
	errSlugEmpty                 = errors.New("slug is required")
	errSlugInvalid               = errors.New("slug must match [a-z0-9-]+")
	slugPattern                  = regexp.MustCompile(`^[a-z0-9-]+$`)
)

type episodeFields struct {
	Situation         string
	Summary           string
	BoundaryRationale string
	TranscriptText    string
	Sessions          []string
	TranscriptFiles   []string
	TranscriptStart   string
	TranscriptEnd     string
	Luhmann           string
	Source            string
	Project           string
	Issue             string
	Tier              string
	// Relations are the authored "<target>|<rationale>" relation bullets,
	// already resolved to full basenames; they render under ## Related after
	// the computed Preceding links.
	Relations []string
	// Preceding are the computed preceding-episode links (active-at-start
	// then immediate-prior), rendered first under ## Related.
	Preceding []episodeLink
}

// episodeFrontmatterDoc is the YAML shape of an episode note's frontmatter.
// Field order here determines key order in the rendered document. Nested
// provenance.sessions + provenance.transcript_range live in named structs so
// yaml.v3 emits stable nested key order.
type episodeFrontmatterDoc struct {
	Type              string               `yaml:"type"`
	Tier              string               `yaml:"tier,omitempty"`
	Situation         string               `yaml:"situation"`
	BoundaryRationale string               `yaml:"boundary_rationale"`
	Provenance        episodeProvenanceDoc `yaml:"provenance"`
	Luhmann           quotedString         `yaml:"luhmann"`
	Created           string               `yaml:"created"`
	Source            string               `yaml:"source"`
	Project           string               `yaml:"project,omitempty"`
	Issue             quotedString         `yaml:"issue,omitempty"`
}

// episodeLink is a resolved preceding-episode (or authored-relation) link: the
// full note basename plus the rationale rendered after the em-dash.
type episodeLink struct {
	Basename  string
	Rationale string
}

// episodeProvenanceDoc holds the nested provenance fields for an episode.
type episodeProvenanceDoc struct {
	Sessions        []string             `yaml:"sessions"`
	TranscriptFiles []string             `yaml:"transcript_files,omitempty"`
	TranscriptRange episodeTranscriptDoc `yaml:"transcript_range"`
}

// episodeTranscriptDoc holds the start/end RFC3339 bounds for an episode's
// transcript range.
type episodeTranscriptDoc struct {
	Start string `yaml:"start"`
	End   string `yaml:"end"`
}

type factFields struct {
	Situation string
	Subject   string
	Predicate string
	Object    string
	Luhmann   string
	Source    string
	Project   string
	Issue     string
	Tier      string
}

// factFrontmatterDoc is the YAML shape of a fact's frontmatter. Field order
// here determines key order in the rendered document.
type factFrontmatterDoc struct {
	Type      string       `yaml:"type"`
	Tier      string       `yaml:"tier,omitempty"`
	Situation string       `yaml:"situation"`
	Subject   string       `yaml:"subject"`
	Predicate string       `yaml:"predicate"`
	Object    string       `yaml:"object"`
	Luhmann   quotedString `yaml:"luhmann"`
	Created   string       `yaml:"created"`
	Source    string       `yaml:"source"`
	Project   string       `yaml:"project,omitempty"`
	Issue     quotedString `yaml:"issue,omitempty"`
}

type feedbackFields struct {
	Situation string
	Behavior  string
	Impact    string
	Action    string
	Luhmann   string
	Source    string
	Project   string
	Issue     string
	Tier      string
}

// feedbackFrontmatterDoc is the YAML shape of a feedback note's frontmatter.
type feedbackFrontmatterDoc struct {
	Type      string       `yaml:"type"`
	Tier      string       `yaml:"tier,omitempty"`
	Situation string       `yaml:"situation"`
	Behavior  string       `yaml:"behavior"`
	Impact    string       `yaml:"impact"`
	Action    string       `yaml:"action"`
	Luhmann   quotedString `yaml:"luhmann"`
	Created   string       `yaml:"created"`
	Source    string       `yaml:"source"`
	Project   string       `yaml:"project,omitempty"`
	Issue     quotedString `yaml:"issue,omitempty"`
}

// parsedEpisodeRange is an EpisodeRange with its bounds parsed to time.Time.
type parsedEpisodeRange struct {
	basename string
	start    time.Time
	end      time.Time
}

// quotedString is a YAML scalar that always renders double-quoted. Used for
// the Luhmann ID field so the rendered frontmatter matches the vault
// convention (luhmann: "9aa") regardless of whether yaml.v3 would otherwise
// quote the value. Without this, IDs like "9aa" emit unquoted, and IDs like
// "1e1" would mis-parse as the float 10.0 on read-back.
type quotedString string

// IsZero lets yaml.v3's `omitempty` skip empty quotedString values; without
// this, custom scalar types always render even when their underlying value
// is "" because the marshaler is invoked unconditionally.
func (q quotedString) IsZero() bool { return string(q) == "" }

// MarshalYAML emits the value as a double-quoted scalar node.
func (q quotedString) MarshalYAML() (any, error) {
	return &yaml.Node{
		Kind:  yaml.ScalarNode,
		Style: yaml.DoubleQuotedStyle,
		Value: string(q),
	}, nil
}

// activeAtStart returns the episodes whose range spans start
// (f.start <= start <= f.end), sorted by f.start ascending (basename
// tie-breaks for determinism).
func activeAtStart(parsed []parsedEpisodeRange, start time.Time) []parsedEpisodeRange {
	active := make([]parsedEpisodeRange, 0, len(parsed))

	for _, f := range parsed {
		if !f.start.After(start) && !f.end.Before(start) {
			active = append(active, f)
		}
	}

	sort.Slice(active, func(i, j int) bool {
		if active[i].start.Equal(active[j].start) {
			return active[i].basename < active[j].basename
		}

		return active[i].start.Before(active[j].start)
	})

	return active
}

func assembleLearnContent(args LearnArgs, luhmann string, when time.Time) (string, error) {
	tierErr := validateTier(args.Tier)
	if tierErr != nil {
		return "", tierErr
	}

	related := renderRelatedSection(args.Relations)

	switch args.Type {
	case typeFeedback:
		if strings.TrimSpace(args.Situation) == "" {
			return "", errFeedbackSituationRequired
		}

		f := feedbackFields{
			Situation: args.Situation, Behavior: args.Behavior, Impact: args.Impact,
			Action: args.Action, Luhmann: luhmann, Source: args.Source,
			Project: args.Project, Issue: args.Issue, Tier: tierOrDefault(args.Tier),
		}

		return renderFeedbackFrontmatter(f, when) + renderFeedbackBody(f, related), nil
	case typeFact:
		if strings.TrimSpace(args.Situation) == "" {
			return "", errFactSituationRequired
		}

		f := factFields{
			Situation: args.Situation, Subject: args.Subject, Predicate: args.Predicate,
			Object: args.Object, Luhmann: luhmann, Source: args.Source,
			Project: args.Project, Issue: args.Issue, Tier: tierOrDefault(args.Tier),
		}

		return renderFactFrontmatter(f, when) + renderFactBody(f, related), nil
	case typeEpisode:
		f, parseErr := buildEpisodeFields(args, luhmann)
		if parseErr != nil {
			return "", parseErr
		}

		f.Tier = tierL1

		// Episodes build their own ## Related section from preceding links +
		// authored relations; they do not use the shared "Related to:" block.
		return renderEpisodeFrontmatter(f, when) + renderEpisodeBody(f), nil
	default:
		return "", fmt.Errorf("%w: got %q", errLearnUnknownType, args.Type)
	}
}

// autoEmbedNote writes a sidecar for the newly-created note. Failure is
// warned-and-ignored: the Luhmann write is atomic, so a missing sidecar
// is recoverable via `engram embed apply --missing` later.
func autoEmbedNote(ctx context.Context, deps LearnDeps, notePath, content string) {
	if deps.Embedder == nil || deps.WriteSidecar == nil {
		return
	}

	sidecar, embErr := embed.BuildSidecar(ctx, deps.Embedder, []byte(content))
	if embErr != nil {
		if deps.LogWarning != nil {
			deps.LogWarning("learn: embed failed for %s: %v", notePath, embErr)
		}

		return
	}

	writeErr := deps.WriteSidecar(embed.SidecarPath(notePath), embed.MarshalSidecar(sidecar))
	if writeErr != nil && deps.LogWarning != nil {
		deps.LogWarning("learn: sidecar write failed for %s: %v", notePath, writeErr)
	}
}

// buildEpisodeFields validates and parses LearnArgs into the episodeFields
// projection used for rendering. Validation here covers required-field
// presence beyond what targ's `required` tag enforces (empty values for
// required flags reject), the --transcript-range format / ordering, and
// the --boundary-rationale + body-source contract. TranscriptText is
// expected to have been pre-resolved by the caller (either from
// --transcript-text verbatim, or from reading --from-transcript-range
// slices); buildEpisodeFields itself does no I/O.
func buildEpisodeFields(args LearnArgs, luhmann string) (episodeFields, error) {
	situationErr := validateEpisodeSituation(args.Situation)
	if situationErr != nil {
		return episodeFields{}, situationErr
	}

	summaryErr := validateEpisodeSummary(args.Summary)
	if summaryErr != nil {
		return episodeFields{}, summaryErr
	}

	rationaleErr := validateEpisodeBoundaryRationale(args.BoundaryRationale)
	if rationaleErr != nil {
		return episodeFields{}, rationaleErr
	}

	sessions, sessionErr := validateEpisodeSessions(args.Sessions)
	if sessionErr != nil {
		return episodeFields{}, sessionErr
	}

	start, end, rangeErr := parseTranscriptRange(args.TranscriptRange)
	if rangeErr != nil {
		return episodeFields{}, rangeErr
	}

	return episodeFields{
		Situation:         args.Situation,
		Summary:           args.Summary,
		BoundaryRationale: args.BoundaryRationale,
		TranscriptText:    args.TranscriptText,
		Sessions:          sessions,
		TranscriptFiles:   args.TranscriptFiles,
		TranscriptStart:   start,
		TranscriptEnd:     end,
		Luhmann:           luhmann,
		Source:            args.Source,
		Project:           args.Project,
		Issue:             args.Issue,
		Relations:         args.Relations,
		Preceding:         args.Preceding,
	}, nil
}

// computePrecedingLinks derives the preceding-episode links for a new episode
// whose transcript range starts at newStart (RFC3339), given the existing
// episodes' ranges. The rule:
//
//   - Active-at-start: every existing F whose range spans newStart
//     (F.start <= newStart <= F.end), rationale precedingActiveRationale,
//     emitted in F.start-ascending order.
//   - Immediate-prior: the single F with the greatest F.end among those with
//     F.end <= newStart, rationale precedingPriorRationale, appended after the
//     active set — unless that F is already in the active set (the
//     F.end == newStart boundary), in which case the single active bullet is
//     kept.
//
// Existing episodes with an unparseable/empty range, or with the same basename
// as a basename already linked, are skipped. Returns nil when nothing links.
func computePrecedingLinks(existing []EpisodeRange, newStart string) []episodeLink {
	start, startErr := time.Parse(time.RFC3339, newStart)
	if startErr != nil {
		return nil
	}

	parsed := make([]parsedEpisodeRange, 0, len(existing))

	for _, candidate := range existing {
		fStart, fStartErr := time.Parse(time.RFC3339, candidate.Start)
		if fStartErr != nil {
			continue
		}

		fEnd, fEndErr := time.Parse(time.RFC3339, candidate.End)
		if fEndErr != nil {
			continue
		}

		parsed = append(parsed, parsedEpisodeRange{basename: candidate.Basename, start: fStart, end: fEnd})
	}

	active := activeAtStart(parsed, start)
	links := make([]episodeLink, 0, len(active)+1)
	linked := make(map[string]struct{}, len(active)+1)

	for _, f := range active {
		if _, dup := linked[f.basename]; dup {
			continue
		}

		linked[f.basename] = struct{}{}

		links = append(links, episodeLink{Basename: f.basename, Rationale: precedingActiveRationale})
	}

	prior, hasPrior := immediatePrior(parsed, start)
	if hasPrior {
		if _, alreadyActive := linked[prior.basename]; !alreadyActive {
			links = append(links, episodeLink{Basename: prior.basename, Rationale: precedingPriorRationale})
		}
	}

	if len(links) == 0 {
		return nil
	}

	return links
}

// cutSessionRef splits "<session-ref>:<start>" into the session reference and
// the start timestamp. An opencode://<id> URI carries a colon in its scheme,
// so the split happens on the colon AFTER the scheme; a bare Claude session ID
// (a UUID, no colon) splits on the first colon.
func cutSessionRef(sessionAndStart string) (string, string, bool) {
	if rest, found := strings.CutPrefix(sessionAndStart, opencodeSessionPrefix); found {
		id, startRaw, ok := strings.Cut(rest, ":")

		return opencodeSessionPrefix + id, startRaw, ok
	}

	return strings.Cut(sessionAndStart, ":")
}

// defaultSessionPathResolver maps a Claude Code session ID to its JSONL
// path inside the per-project transcript directory. Resolution follows
// the same pattern as `engram transcript`: cwd → ProjectSlugFromPath →
// $HOME/.claude/projects/<slug>/<session-id>.jsonl. Composed via
// resolveSessionPath so error branches are unit-testable via injection.
func defaultSessionPathResolver(sessionID string) (string, error) {
	return resolveSessionPath(sessionID, os.Getwd, os.UserHomeDir)
}

// episodePrecedingLinks scans the vault for existing episodes (via
// deps.ListEpisodes) and computes the new episode's preceding links from its
// transcript-range start. A nil ListEpisodes (tests that don't wire the scan)
// safely yields no links. A malformed --transcript-range yields no links here;
// buildEpisodeFields raises the canonical validation error downstream.
func episodePrecedingLinks(args LearnArgs, deps LearnDeps, vault string) ([]episodeLink, error) {
	if deps.ListEpisodes == nil {
		return nil, nil
	}

	start, _, rangeErr := parseTranscriptRange(args.TranscriptRange)
	if rangeErr != nil {
		// Swallow: a malformed range yields no preceding links here.
		// buildEpisodeFields raises the canonical validation error downstream,
		// so surfacing rangeErr now would only duplicate (and worsen) it.
		return nil, nil //nolint:nilerr // canonical range error is raised in buildEpisodeFields
	}

	existing, listErr := deps.ListEpisodes(vault)
	if listErr != nil {
		return nil, fmt.Errorf("learn: listing episodes: %w", listErr)
	}

	return computePrecedingLinks(existing, start), nil
}

// episodeRangeFromNote extracts the (basename, start, end) EpisodeRange from a
// raw note's frontmatter, reporting whether the note is an episode. Non-episode
// notes and notes without parseable frontmatter return ok=false.
func episodeRangeFromNote(basename string, raw []byte) (EpisodeRange, bool) {
	frontmatter, ok := splitFrontmatter(raw)
	if !ok {
		return EpisodeRange{}, false
	}

	if peekNoteType(frontmatter) != typeEpisode {
		return EpisodeRange{}, false
	}

	var doc episodeFrontmatterDoc

	unmarshalErr := yaml.Unmarshal(frontmatter, &doc)
	if unmarshalErr != nil {
		return EpisodeRange{}, false
	}

	return EpisodeRange{
		Basename: basename,
		Start:    doc.Provenance.TranscriptRange.Start,
		End:      doc.Provenance.TranscriptRange.End,
	}, true
}

// episodeRelatedBullets renders the ## Related bullets: the computed preceding
// links first (in their given order), then the authored relations. Both are
// "- [[basename]] — rationale". Duplicates by basename are dropped, keeping the
// first occurrence (so a preceding link wins over an authored relation to the
// same note). Returns nil when there is nothing to link.
func episodeRelatedBullets(preceding []episodeLink, relations []string) []string {
	bullets := make([]string, 0, len(preceding)+len(relations))
	seen := make(map[string]struct{}, len(preceding)+len(relations))

	add := func(basename, rationale string) {
		if basename == "" {
			return
		}

		if _, dup := seen[basename]; dup {
			return
		}

		seen[basename] = struct{}{}

		bullets = append(bullets, fmt.Sprintf("- [[%s]] — %s", basename, rationale))
	}

	for _, link := range preceding {
		add(link.Basename, link.Rationale)
	}

	for _, relation := range relations {
		target, rationale, _ := strings.Cut(relation, "|")
		add(strings.TrimSpace(target), strings.TrimSpace(rationale))
	}

	if len(bullets) == 0 {
		return nil
	}

	return bullets
}

// extractLuhmannFromFilename strips the `.md` extension and delegates to
// luhmann.FromBasename — the canonical extractor (see #626). Returns
// ("", false) for any non-`.md` filename or one without a valid leading ID.
func extractLuhmannFromFilename(name string) (string, bool) {
	const mdExt = ".md"

	if !strings.HasSuffix(name, mdExt) {
		return "", false
	}

	return luhmann.FromBasename(strings.TrimSuffix(name, mdExt))
}

// immediatePrior returns the episode with the greatest end among those ending
// at or before start, and whether any such episode exists. Basename
// tie-breaks when two share the greatest end, for determinism.
func immediatePrior(parsed []parsedEpisodeRange, start time.Time) (parsedEpisodeRange, bool) {
	var (
		best  parsedEpisodeRange
		found bool
	)

	for _, f := range parsed {
		if f.end.After(start) {
			continue
		}

		switch {
		case !found:
			best, found = f, true
		case f.end.After(best.end):
			best = f
		case f.end.Equal(best.end) && f.basename < best.basename:
			best = f
		}
	}

	return best, found
}

func learnPath(vault, luhmann, slug string, when time.Time) string {
	filename := fmt.Sprintf("%s.%s.%s.md", luhmann, when.Format(dateFormat), slug)

	return filepath.Join(vault, filename)
}

// logWarningToStderrf is the production LogWarning hook. Method-named so
// coverage attributes its execution to one identifier rather than to an
// anonymous closure inside newOsLearnDeps.
func logWarningToStderrf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, "warning: "+format+"\n", args...)
}

// longestBacktickRun returns the length of the longest run of consecutive
// backtick characters anywhere in text.
func longestBacktickRun(text string) int {
	longest, current := 0, 0

	for _, r := range text {
		if r == backtickRune {
			current++
			if current > longest {
				longest = current
			}

			continue
		}

		current = 0
	}

	return longest
}

// marshalFrontmatter encodes v as YAML and wraps the result with the "---"
// delimiters and trailing blank line used by Permanent/MOC notes. All callers
// pass structs of typed string fields and do not implement MarshalYAML, so
// yaml.Marshal cannot fail on them — yaml.v3 returns errors only via custom
// marshalers, and panics on truly unencodable types (programmer error).
func marshalFrontmatter(v any) string {
	body, _ := yaml.Marshal(v)

	return "---\n" + string(body) + "---\n\n"
}

func newOsLearnDeps() LearnDeps {
	vaultFS := &osLearnFS{}

	return LearnDeps{
		Now:           time.Now,
		Getenv:        os.Getenv,
		StatDir:       vaultFS.StatDir,
		InitVault:     func(path string) error { return initializeVault(vaultFS, path) },
		ListIDs:       vaultFS.ListIDs,
		ListBasenames: vaultFS.ListBasenames,
		ListEpisodes:  vaultFS.ListEpisodes,
		Lock:          vaultFS.Lock,
		WriteNew:      vaultFS.WriteNew,
		Embedder:      sharedEmbedder,
		WriteSidecar:  vaultFS.WriteSidecar,
		LogWarning:    logWarningToStderrf,
	}
}

// parseFromTranscriptRange splits a "<session-id>:<RFC3339-start>..<RFC3339-end>"
// literal into its three components. RFC3339 timestamps contain colons
// (HH:MM:SS), so the parser first cuts on ".." to peel the end, then
// splits the start half on the FIRST colon to isolate the session ID.
// Claude Code session IDs are UUIDs (no colons), so first-colon is safe.
func parseFromTranscriptRange(raw string) (string, time.Time, time.Time, error) {
	startEndRaw, endRaw, ok := strings.Cut(raw, "..")
	if !ok || startEndRaw == "" || endRaw == "" {
		return "", time.Time{}, time.Time{},
			fmt.Errorf("%w: got %q", errEpisodeFromRangeFmt, raw)
	}

	sessionRef, startRaw, ok := cutSessionRef(startEndRaw)
	if !ok || sessionRef == "" || startRaw == "" {
		return "", time.Time{}, time.Time{},
			fmt.Errorf("%w: got %q", errEpisodeFromRangeFmt, raw)
	}

	start, startErr := time.Parse(time.RFC3339, startRaw)
	if startErr != nil {
		return "", time.Time{}, time.Time{},
			fmt.Errorf("%w: start %q: %w", errEpisodeFromRangeFmt, startRaw, startErr)
	}

	end, endErr := time.Parse(time.RFC3339, endRaw)
	if endErr != nil {
		return "", time.Time{}, time.Time{},
			fmt.Errorf("%w: end %q: %w", errEpisodeFromRangeFmt, endRaw, endErr)
	}

	if !start.Before(end) {
		return "", time.Time{}, time.Time{},
			fmt.Errorf("%w: %s..%s", errEpisodeFromRangeOrder, startRaw, endRaw)
	}

	return sessionRef, start, end, nil
}

// parseTranscriptRange parses a "<RFC3339-start>..<RFC3339-end>" string into
// its two RFC3339 components. Returns an error if the literal is malformed,
// either side fails to parse as RFC3339, or start is not strictly before end.
func parseTranscriptRange(raw string) (string, string, error) {
	if raw == "" {
		return "", "", errEpisodeTranscriptRangeReq
	}

	const sep = ".."

	startRaw, endRaw, ok := strings.Cut(raw, sep)
	if !ok || startRaw == "" || endRaw == "" {
		return "", "", fmt.Errorf("%w: got %q", errEpisodeTranscriptRangeFmt, raw)
	}

	start, startErr := time.Parse(time.RFC3339, startRaw)
	if startErr != nil {
		return "", "", fmt.Errorf(
			"%w: start %q: %w",
			errEpisodeTranscriptRangeFmt,
			startRaw,
			startErr,
		)
	}

	end, endErr := time.Parse(time.RFC3339, endRaw)
	if endErr != nil {
		return "", "", fmt.Errorf("%w: end %q: %w", errEpisodeTranscriptRangeFmt, endRaw, endErr)
	}

	if !start.Before(end) {
		return "", "", fmt.Errorf("%w: %s..%s", errEpisodeTranscriptRangeOrder, startRaw, endRaw)
	}

	return startRaw, endRaw, nil
}

// renderEpisodeBody assembles the D6 L1 episode body as exactly three
// sections: ## Summary (the authored "what happened" prose), ## Transcript
// (the verbatim chunk inside a backtick fence long enough to round-trip any
// inner fence and to suppress wikilinks per ParseWikilinks/G5), and ##
// Related (computed preceding-episode links then authored relations). The
// ## Related section is omitted entirely when there are zero of both.
func renderEpisodeBody(f episodeFields) string {
	var builder strings.Builder

	builder.WriteString(episodeSummaryHeading)
	builder.WriteString("\n")
	builder.WriteString(strings.TrimRight(f.Summary, "\n"))
	builder.WriteString("\n\n")

	builder.WriteString(episodeTranscriptHeading)
	builder.WriteString("\n")
	builder.WriteString(renderFencedTranscript(f.TranscriptText))

	bullets := episodeRelatedBullets(f.Preceding, f.Relations)
	if len(bullets) > 0 {
		builder.WriteString("\n")
		builder.WriteString(episodeRelatedHeading)
		builder.WriteString("\n")
		builder.WriteString(strings.Join(bullets, "\n"))
		builder.WriteString("\n")
	}

	return builder.String()
}

// renderEpisodeFrontmatter encodes an episode's metadata as YAML wrapped in
// "---" delimiters. Key order is fixed by the field declaration order on
// episodeFrontmatterDoc / episodeProvenanceDoc.
func renderEpisodeFrontmatter(f episodeFields, when time.Time) string {
	return marshalFrontmatter(episodeFrontmatterDoc{
		Type:              typeEpisode,
		Tier:              f.Tier,
		Situation:         f.Situation,
		BoundaryRationale: f.BoundaryRationale,
		Provenance: episodeProvenanceDoc{
			Sessions:        f.Sessions,
			TranscriptFiles: f.TranscriptFiles,
			TranscriptRange: episodeTranscriptDoc{
				Start: f.TranscriptStart,
				End:   f.TranscriptEnd,
			},
		},
		Luhmann: quotedString(f.Luhmann),
		Created: when.Format(dateFormat),
		Source:  f.Source,
		Project: f.Project,
		Issue:   quotedString(f.Issue),
	})
}

func renderFactBody(f factFields, relatedSection string) string {
	formula := fmt.Sprintf(
		"Information learned: when in %s, %s %s %s.\n",
		stripLeadingWhen(f.Situation), f.Subject, f.Predicate, f.Object,
	)

	return formula + "\n" + relatedSection
}

func renderFactFrontmatter(f factFields, when time.Time) string {
	return marshalFrontmatter(factFrontmatterDoc{
		Type:      typeFact,
		Tier:      f.Tier,
		Situation: f.Situation,
		Subject:   f.Subject,
		Predicate: f.Predicate,
		Object:    f.Object,
		Luhmann:   quotedString(f.Luhmann),
		Created:   when.Format(dateFormat),
		Source:    f.Source,
		Project:   f.Project,
		Issue:     quotedString(f.Issue),
	})
}

func renderFeedbackBody(f feedbackFields, relatedSection string) string {
	formula := fmt.Sprintf(
		"Lesson learned: when %s, %s.\n",
		stripLeadingWhen(f.Situation), f.Action,
	)

	return formula + "\n" + relatedSection
}

func renderFeedbackFrontmatter(f feedbackFields, when time.Time) string {
	return marshalFrontmatter(feedbackFrontmatterDoc{
		Type:      typeFeedback,
		Tier:      f.Tier,
		Situation: f.Situation,
		Behavior:  f.Behavior,
		Impact:    f.Impact,
		Action:    f.Action,
		Luhmann:   quotedString(f.Luhmann),
		Created:   when.Format(dateFormat),
		Source:    f.Source,
		Project:   f.Project,
		Issue:     quotedString(f.Issue),
	})
}

// renderFencedTranscript wraps text in a backtick fence whose run length is
// max(episodeFenceMin, longestBacktickRun(text)+1), so an inner ``` cannot
// close the block. The opening and closing fences are on their own lines, and
// the transcript is newline-terminated before the closer.
func renderFencedTranscript(text string) string {
	fence := strings.Repeat(string(backtickRune), transcriptFenceLen(text))

	body := text
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}

	return fence + "\n" + body + fence + "\n"
}

// renderRelatedSection turns a list of "wikilink|rationale" entries into the
// "Related to:\n- [[...]] — rationale.\n" block. Returns "" when empty.
func renderRelatedSection(entries []string) string {
	if len(entries) == 0 {
		return ""
	}

	lines := make([]string, 0, len(entries)+1)
	lines = append(lines, relatedSectionMarker)

	for _, entry := range entries {
		target, rationale, _ := strings.Cut(entry, "|")
		lines = append(
			lines,
			fmt.Sprintf("- [[%s]] — %s.", strings.TrimSpace(target), strings.TrimSpace(rationale)),
		)
	}

	return strings.Join(lines, "\n") + "\n"
}

// resolveEpisodeBody returns the body text for an episode write. Exactly
// one of --from-transcript-range or --transcript-text must be set;
// resolveEpisodeBody enforces that XOR. For --from-transcript-range,
// each entry is parsed into (sessionID, start, end), the session is
// located via sessionPath, and the chunk is read+filtered through the
// injected RangeReader. Multiple --from-transcript-range entries are
// concatenated in input order with blank-line separators between
// sessions.
func resolveEpisodeBody(
	a LearnEpisodeArgs,
	reader transcript.RangeReader,
	sessionPath func(sessionID string) (string, error),
	opencodeDBPath string,
) (string, []string, error) {
	hasRange := len(a.FromTranscriptRange) > 0
	hasText := a.TranscriptText != ""

	switch {
	case hasRange && hasText:
		return "", nil, errEpisodeBodySourceBoth
	case !hasRange && !hasText:
		return "", nil, errEpisodeBodySourceRequired
	case hasText:
		return a.TranscriptText, nil, nil
	}

	chunks := make([]string, 0, len(a.FromTranscriptRange))
	files := make([]string, 0, len(a.FromTranscriptRange))
	seen := make(map[string]bool, len(a.FromTranscriptRange))

	for _, raw := range a.FromTranscriptRange {
		chunk, path, spanErr := resolveTranscriptSpan(raw, reader, sessionPath, opencodeDBPath)
		if spanErr != nil {
			return "", nil, spanErr
		}

		chunks = append(chunks, chunk)

		if !seen[path] {
			seen[path] = true

			files = append(files, path)
		}
	}

	return strings.Join(chunks, "\n"), files, nil
}

// resolveSessionPath is the injectable computation behind
// defaultSessionPathResolver. getwd / homeDir are factored as
// dependencies so coverage can drive both error branches via fakes.
func resolveSessionPath(
	sessionID string,
	getwd func() (string, error),
	homeDir func() (string, error),
) (string, error) {
	cwd, err := getwd()
	if err != nil {
		return "", fmt.Errorf("getwd: %w", err)
	}

	home, err := homeDir()
	if err != nil {
		return "", fmt.Errorf("home dir: %w", err)
	}

	slug := ProjectSlugFromPath(cwd)

	return filepath.Join(home, ".claude", "projects", slug, sessionID+".jsonl"), nil
}

// resolveSpanPaths returns the read handle (what the RangeReader reads) and the
// record path (what provenance.transcript_files stores) for a session ref. An
// opencode://<id> session reads via the URI and records the OpenCode DB file
// path; a bare Claude session ID resolves to (and records) its .jsonl path.
func resolveSpanPaths(
	sessionRef string,
	sessionPath func(sessionID string) (string, error),
	opencodeDBPath string,
) (string, string, error) {
	if strings.HasPrefix(sessionRef, opencodeSessionPrefix) {
		return sessionRef, opencodeDBPath, nil
	}

	path, pathErr := sessionPath(sessionRef)
	if pathErr != nil {
		return "", "", fmt.Errorf("episode: resolving session path for %q: %w", sessionRef, pathErr)
	}

	return path, path, nil
}

// resolveTranscriptSpan parses one --from-transcript-range entry, resolves the
// session's transcript file path, and reads+filters the chunk for that span.
// Returns the chunk text and the resolved file path — the path is recorded in
// the episode's provenance.transcript_files so the L1 note links back to its
// source transcript.
func resolveTranscriptSpan(
	raw string,
	reader transcript.RangeReader,
	sessionPath func(sessionID string) (string, error),
	opencodeDBPath string,
) (string, string, error) {
	sessionRef, start, end, parseErr := parseFromTranscriptRange(raw)
	if parseErr != nil {
		return "", "", parseErr
	}

	readHandle, recordPath, resolveErr := resolveSpanPaths(sessionRef, sessionPath, opencodeDBPath)
	if resolveErr != nil {
		return "", "", resolveErr
	}

	chunk, readErr := reader.ReadRange(readHandle, start, end)
	if readErr != nil {
		return "", "", fmt.Errorf("episode: reading transcript range %q: %w", raw, readErr)
	}

	return chunk, recordPath, nil
}

// resolveVault returns the vault path. Flag wins, then env, then the XDG
// default ($XDG_DATA_HOME/engram/vault, falling back to
// $HOME/.local/share/engram/vault). home and getenv are injected so callers
// control environment access; pass the result of os.UserHomeDir and
// os.Getenv in production. The returned path is never empty — callers that
// need "does this exist?" semantics must stat it separately.
func resolveVault(flagValue, home string, getenv func(string) string) string {
	if flagValue != "" {
		return flagValue
	}

	if env := getenv(envVaultPath); env != "" {
		return env
	}

	return filepath.Join(DataDirFromHome(home, getenv), "vault")
}

// runLearn orchestrates the learn subcommand: validates inputs, ensures the
// vault directory exists (creating it on first use), acquires the lock,
// computes the next Luhmann ID, and writes the file. args.Vault must
// already be resolved by the caller via resolveVault.
func runLearn(ctx context.Context, args LearnArgs, deps LearnDeps, stdout io.Writer) error {
	slugErr := validateSlug(args.Slug)
	if slugErr != nil {
		return fmt.Errorf("learn: %w", slugErr)
	}

	projectErr := validateProjectSlug(args.Project)
	if projectErr != nil {
		return fmt.Errorf("learn: %w", projectErr)
	}

	issueErr := validateIssueID(args.Issue)
	if issueErr != nil {
		return fmt.Errorf("learn: %w", issueErr)
	}

	vault := args.Vault

	dirErr := deps.StatDir(vault)
	if dirErr != nil {
		if !errors.Is(dirErr, fs.ErrNotExist) {
			return fmt.Errorf("learn: vault %s: %w", vault, dirErr)
		}

		initErr := deps.InitVault(vault)
		if initErr != nil {
			return fmt.Errorf("learn: %w", initErr)
		}
	}

	path, writeErr := writeLearnUnderLock(ctx, args, deps, vault)
	if writeErr != nil {
		return writeErr
	}

	_, _ = fmt.Fprintln(stdout, path)

	return nil
}

func runLearnFromEpisodeArgs(ctx context.Context, a LearnEpisodeArgs, stdout io.Writer) error {
	deps := newOsLearnDeps()
	dbPath := transcript.DefaultOpencodeDBPath()
	reader := transcript.NewCompositeRangeReader(
		transcript.NewJSONLRangeReader(&osFileReader{}),
		transcript.NewOpencodeTranscriptReader(dbPath),
	)

	return runLearnFromEpisodeArgsWithReader(
		ctx,
		a,
		reader,
		defaultSessionPathResolver,
		dbPath,
		deps,
		stdout,
	)
}

// runLearnFromEpisodeArgsWithReader is the testable seam for episode writes.
// It resolves --from-transcript-range or --transcript-text into a single
// body string via the injected RangeReader and session-path resolver, then
// delegates to runLearn. Pre-resolving here keeps assembleLearnContent /
// buildEpisodeFields pure (no I/O in the renderer).
func runLearnFromEpisodeArgsWithReader(
	ctx context.Context,
	a LearnEpisodeArgs,
	reader transcript.RangeReader,
	sessionPath func(sessionID string) (string, error),
	opencodeDBPath string,
	deps LearnDeps,
	stdout io.Writer,
) error {
	body, files, bodyErr := resolveEpisodeBody(a, reader, sessionPath, opencodeDBPath)
	if bodyErr != nil {
		return bodyErr
	}

	return runLearn(ctx, LearnArgs{
		Type:              typeEpisode,
		Slug:              a.Slug,
		Vault:             a.Vault,
		Target:            a.Target,
		Position:          a.Position,
		Source:            a.Source,
		Project:           a.Project,
		Issue:             a.Issue,
		Tier:              a.Tier,
		Relations:         a.Relations,
		Situation:         a.Situation,
		Summary:           a.Summary,
		BoundaryRationale: a.BoundaryRationale,
		TranscriptText:    body,
		TranscriptFiles:   files,
		Sessions:          a.Sessions,
		TranscriptRange:   a.TranscriptRange,
	}, deps, stdout)
}

func runLearnFromFactArgs(ctx context.Context, a LearnFactArgs, stdout io.Writer) error {
	deps := newOsLearnDeps()

	return runLearn(ctx, LearnArgs{
		Type:      typeFact,
		Slug:      a.Slug,
		Vault:     a.Vault,
		Target:    a.Target,
		Position:  a.Position,
		Source:    a.Source,
		Project:   a.Project,
		Issue:     a.Issue,
		Tier:      a.Tier,
		Relations: a.Relations,
		Situation: a.Situation,
		Subject:   a.Subject,
		Predicate: a.Predicate,
		Object:    a.Object,
	}, deps, stdout)
}

func runLearnFromFeedbackArgs(ctx context.Context, a LearnFeedbackArgs, stdout io.Writer) error {
	deps := newOsLearnDeps()

	return runLearn(ctx, LearnArgs{
		Type:      typeFeedback,
		Slug:      a.Slug,
		Vault:     a.Vault,
		Target:    a.Target,
		Position:  a.Position,
		Source:    a.Source,
		Project:   a.Project,
		Issue:     a.Issue,
		Tier:      a.Tier,
		Relations: a.Relations,
		Situation: a.Situation,
		Behavior:  a.Behavior,
		Impact:    a.Impact,
		Action:    a.Action,
	}, deps, stdout)
}

// stripLeadingWhen removes a case-insensitive leading "When " or "when " from
// the situation field so body templates that prepend "when " don't double up.
// Skill-spec example situations conventionally start with "When ..." but the
// body template prepends "when " — without this strip, the rendered line
// reads "Lesson learned: when When ...".
func stripLeadingWhen(situation string) string {
	const whenPrefixLen = 5

	if len(situation) < whenPrefixLen {
		return situation
	}

	if strings.EqualFold(situation[:whenPrefixLen], "when ") {
		return situation[whenPrefixLen:]
	}

	return situation
}

// tierOrDefault returns the explicit tier override when set, falling back to
// L2 — the default tier for fact and feedback notes. Extracted so both the
// fact and feedback branches of assembleLearnContent share one branch point.
func tierOrDefault(tier string) string {
	if tier == "" {
		return tierL2
	}

	return tier
}

// transcriptFenceLen returns the backtick-run length for an episode transcript
// fence: the larger of episodeFenceMin and one more than the longest backtick
// run already in the text.
func transcriptFenceLen(text string) int {
	longest := longestBacktickRun(text)
	if longest+1 > episodeFenceMin {
		return longest + 1
	}

	return episodeFenceMin
}

// validateEpisodeBoundaryRationale rejects empty/whitespace rationale strings.
func validateEpisodeBoundaryRationale(rationale string) error {
	if strings.TrimSpace(rationale) == "" {
		return errEpisodeBoundaryRequired
	}

	return nil
}

// validateEpisodeSessions returns the sessions slice unchanged on success.
// Rejects when the slice is empty or any entry is empty/whitespace.
func validateEpisodeSessions(sessions []string) ([]string, error) {
	if len(sessions) == 0 {
		return nil, errEpisodeSessionRequired
	}

	for _, session := range sessions {
		if strings.TrimSpace(session) == "" {
			return nil, errEpisodeSessionEmpty
		}
	}

	return sessions, nil
}

// validateEpisodeSituation rejects empty/whitespace situation strings.
func validateEpisodeSituation(situation string) error {
	if strings.TrimSpace(situation) == "" {
		return errEpisodeSituationRequired
	}

	return nil
}

// validateEpisodeSummary rejects empty/whitespace summary strings. The summary
// is the authored "what happened" prose that heads the D6 episode body.
func validateEpisodeSummary(summary string) error {
	if strings.TrimSpace(summary) == "" {
		return errEpisodeSummaryRequired
	}

	return nil
}

// validateIssueID rejects non-empty IDs that contain whitespace. Empty is
// allowed: issue is optional metadata. Whitespace would corrupt the YAML
// scalar and is also vanishingly unlikely to be a real issue ID.
func validateIssueID(id string) error {
	if id == "" {
		return nil
	}

	if strings.ContainsAny(id, " \t\n\r") {
		return fmt.Errorf("%w: got %q", errIssueIDInvalid, id)
	}

	return nil
}

// validateProjectSlug rejects non-empty slugs that don't fit the kebab-case
// shape. Empty is allowed: project is optional metadata, absence is a
// universal-principle marker.
func validateProjectSlug(slug string) error {
	if slug == "" {
		return nil
	}

	if !slugPattern.MatchString(slug) {
		return fmt.Errorf("%w: got %q", errProjectSlugInvalid, slug)
	}

	return nil
}

// validateSlug returns an error if slug is empty or does not match [a-z0-9-]+.
func validateSlug(slug string) error {
	if slug == "" {
		return errSlugEmpty
	}

	if !slugPattern.MatchString(slug) {
		return fmt.Errorf("%w: got %q", errSlugInvalid, slug)
	}

	return nil
}

// validateTier returns an error if the tier override is non-empty and not one
// of the recognised values L1, L2, or L3.
func validateTier(tier string) error {
	if tier == "" {
		return nil
	}

	switch tier {
	case tierL1, tierL2, tierL3:
		return nil
	default:
		return fmt.Errorf("%w: got %q", errLearnBadTier, tier)
	}
}

// writeLearnUnderLock acquires the vault lock, computes the next Luhmann ID,
// assembles file content, and writes it. The lock spans listing existing IDs
// through writing the new file to prevent ID collisions.
func writeLearnUnderLock(
	ctx context.Context,
	args LearnArgs,
	deps LearnDeps,
	vault string,
) (string, error) {
	release, lockErr := deps.Lock(vault)
	if lockErr != nil {
		return "", fmt.Errorf("learn: acquiring lock: %w", lockErr)
	}
	defer release()

	existing, listErr := deps.ListIDs(vault)
	if listErr != nil {
		return "", fmt.Errorf("learn: listing existing IDs: %w", listErr)
	}

	luhmann, idErr := nextLuhmannID(existing, args.Target, args.Position)
	if idErr != nil {
		return "", fmt.Errorf("learn: %w", idErr)
	}

	when := deps.Now()
	path := learnPath(vault, luhmann, args.Slug, when)

	if deps.ListBasenames != nil {
		basenames, bErr := deps.ListBasenames(vault)
		if bErr != nil {
			return "", fmt.Errorf("learn: listing basenames: %w", bErr)
		}

		args.Relations = resolveRelationTargets(args.Relations, basenames)
	}

	if args.Type == typeEpisode {
		preceding, precErr := episodePrecedingLinks(args, deps, vault)
		if precErr != nil {
			return "", precErr
		}

		args.Preceding = preceding
	}

	content, contentErr := assembleLearnContent(args, luhmann, when)
	if contentErr != nil {
		return "", fmt.Errorf("learn: %w", contentErr)
	}

	writeErr := deps.WriteNew(path, []byte(content))
	if writeErr != nil {
		return "", fmt.Errorf("learn: writing %s: %w", path, writeErr)
	}

	autoEmbedNote(ctx, deps, path, content)

	return path, nil
}
