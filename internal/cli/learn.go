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
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/luhmann"
)

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

	// Supersedes carries `--supersedes "<basename>|<type>|<claim>"` flags. Each
	// entry is validated and written to both the frontmatter supersedes: list and
	// the body Supersedes: wikilink lines.
	Supersedes []string

	// ChunkSources carries chunk-index ids (source#anchor) to record as frontmatter
	// provenance. Written to `sources:` when non-empty. Passed via --chunk-source.
	// learn records these unvalidated by design — at create time the just-written
	// chunks may not yet be in the index. (engram amend, by contrast, validates
	// --chunk-source ids against the index because it runs after ingestion.)
	ChunkSources []string

	// feedback / fact share these
	Situation string
	// feedback only
	Behavior string
	Impact   string
	Action   string
	// fact only
	Subject   string
	Predicate string
	Object    string
}

// LearnDeps holds injected dependencies for runLearn. All fields are
// required except Embedder / WriteSidecar / LogWarning, which together
// drive the auto-embed step. A nil Embedder skips auto-embed entirely
// (used by tests that don't exercise the embedding pipeline).
// The vocab assignment fields (LoadTermVectors / ReadSidecar / WriteNote)
// are optional: all three must be non-nil for assignment to run. Nil deps
// → no-op (backward compatible with pre-bootstrap vaults).
type LearnDeps struct {
	Now           func() time.Time
	Getenv        func(string) string
	StatDir       func(string) error
	InitVault     func(string) error
	ListIDs       func(vault string) ([]string, error)
	ListBasenames func(vault string) ([]string, error)
	Lock          func(vault string) (release func(), err error)
	WriteNew      func(path string, data []byte) error
	Embedder      embed.Embedder
	WriteSidecar  func(path string, data []byte) error
	LogWarning    func(format string, args ...any)
	// Vocab assignment — optional; all three must be non-nil to activate.
	// LoadTermVectors returns vocab term→vector pairs from the vault.
	// Returns nil/empty when no term notes exist (no-op, backward compat).
	LoadTermVectors func(vault string) ([]TermWithVector, error)
	// ReadSidecar reads a .vec.json sidecar file. Used to load the
	// newly-written note's body vector for vocab assignment.
	ReadSidecar func(path string) ([]byte, error)
	// WriteNote atomically rewrites an existing note file. Used only when
	// vocab assignment produces tags to write both channels.
	WriteNote func(path string, data []byte) error
	// ListMD lists full .md filenames in the vault for the vocab trigger scan.
	// Optional: nil skips the trigger check (backward compat).
	// Must use full filenames (not stripped basenames) to avoid false-firing the
	// untagged-rate trigger on every learn.
	ListMD func(vault string) ([]string, error)
}

// unexported constants.
const (
	dateFormat   = "2006-01-02"
	envVaultPath = "ENGRAM_VAULT_PATH"
	tierL2       = "L2"
	typeFact     = "fact"
	typeFeedback = "feedback"
)

// unexported variables.
var (
	errFactSituationRequired     = errors.New("fact: --situation is required")
	errFeedbackSituationRequired = errors.New("feedback: --situation is required")
	errIssueIDInvalid            = errors.New("issue must be non-empty with no whitespace")
	errLearnBadTier              = errors.New("tier must be L2 (active) or L1/L3 (legacy backward-compat)")
	errLearnUnknownType          = errors.New("learn: type must be feedback or fact")
	errProjectSlugInvalid        = errors.New("project slug must match [a-z0-9-]+")
	errSlugEmpty                 = errors.New("slug is required")
	errSlugInvalid               = errors.New("slug must match [a-z0-9-]+")
	slugPattern                  = regexp.MustCompile(`^[a-z0-9-]+$`)
)

type factFields struct {
	Situation    string
	Subject      string
	Predicate    string
	Object       string
	Luhmann      string
	Source       string
	Project      string
	Issue        string
	Tier         string
	ChunkSources []string
	Supersedes   []supersedesEntry
}

// factFrontmatterDoc is the YAML shape of a fact's frontmatter. Field order
// here determines key order in the rendered document.
type factFrontmatterDoc struct {
	Type       string            `yaml:"type"`
	Tier       string            `yaml:"tier,omitempty"`
	Situation  string            `yaml:"situation"`
	Subject    string            `yaml:"subject"`
	Predicate  string            `yaml:"predicate"`
	Object     string            `yaml:"object"`
	Luhmann    quotedString      `yaml:"luhmann"`
	Created    string            `yaml:"created"`
	Source     string            `yaml:"source"`
	Project    string            `yaml:"project,omitempty"`
	Issue      quotedString      `yaml:"issue,omitempty"`
	Sources    []string          `yaml:"sources,omitempty"`
	Vocab      []string          `yaml:"vocab,omitempty"`
	Supersedes []supersedesEntry `yaml:"supersedes,omitempty"`
}

type feedbackFields struct {
	Situation    string
	Behavior     string
	Impact       string
	Action       string
	Luhmann      string
	Source       string
	Project      string
	Issue        string
	Tier         string
	ChunkSources []string
	Supersedes   []supersedesEntry
}

// feedbackFrontmatterDoc is the YAML shape of a feedback note's frontmatter.
type feedbackFrontmatterDoc struct {
	Type       string            `yaml:"type"`
	Tier       string            `yaml:"tier,omitempty"`
	Situation  string            `yaml:"situation"`
	Behavior   string            `yaml:"behavior"`
	Impact     string            `yaml:"impact"`
	Action     string            `yaml:"action"`
	Luhmann    quotedString      `yaml:"luhmann"`
	Created    string            `yaml:"created"`
	Source     string            `yaml:"source"`
	Project    string            `yaml:"project,omitempty"`
	Issue      quotedString      `yaml:"issue,omitempty"`
	Sources    []string          `yaml:"sources,omitempty"`
	Vocab      []string          `yaml:"vocab,omitempty"`
	Supersedes []supersedesEntry `yaml:"supersedes,omitempty"`
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

// applyLearnVocabAssignment performs only the term-assignment part of
// applyVocabAssignmentAfterLearn, keeping the trigger check outside this
// early-return chain.
func applyLearnVocabAssignment(deps LearnDeps, vault, notePath, content string) {
	applyVocabAssignmentCore(
		deps.LoadTermVectors, deps.ReadSidecar, deps.WriteNote, deps.LogWarning,
		vault, notePath, content, "learn")
}

// applyVocabAssignmentAfterLearn assigns vocab terms to a newly written note,
// then evaluates the vocab refit trigger. Vocab assignment requires all three
// of LoadTermVectors, ReadSidecar, and WriteNote to be non-nil; a nil dep or
// empty term set silently skips assignment (backward compat). The trigger
// check runs unconditionally — it is gated on deps.ListMD inside the callee.
func applyVocabAssignmentAfterLearn(deps LearnDeps, vault, notePath, content string) {
	applyLearnVocabAssignment(deps, vault, notePath, content)

	// Trigger check: evaluate vocab refit thresholds after every note write.
	// Uses existing deps; all must be non-nil (gated inside the callee).
	checkAndPersistVocabRefitTrigger(
		vault,
		deps.ListMD,
		deps.ReadSidecar,
		deps.WriteNote,
		deps.LogWarning,
		deps.Now(),
	)
}

func assembleLearnContent(args LearnArgs, luhmann string, when time.Time) (string, error) {
	tierErr := validateTier(args.Tier)
	if tierErr != nil {
		return "", tierErr
	}

	parsedSupersedes, supErr := parseAllSupersedes(args.Supersedes)
	if supErr != nil {
		return "", fmt.Errorf("learn: %w", supErr)
	}

	switch args.Type {
	case typeFeedback:
		if strings.TrimSpace(args.Situation) == "" {
			return "", errFeedbackSituationRequired
		}

		f := feedbackFields{
			Situation: args.Situation, Behavior: args.Behavior, Impact: args.Impact,
			Action: args.Action, Luhmann: luhmann, Source: args.Source,
			Project: args.Project, Issue: args.Issue, Tier: tierOrDefault(args.Tier),
			ChunkSources: args.ChunkSources, Supersedes: parsedSupersedes,
		}

		return renderFeedbackFrontmatter(f, when) + renderFeedbackBody(f), nil
	case typeFact:
		if strings.TrimSpace(args.Situation) == "" {
			return "", errFactSituationRequired
		}

		f := factFields{
			Situation: args.Situation, Subject: args.Subject, Predicate: args.Predicate,
			Object: args.Object, Luhmann: luhmann, Source: args.Source,
			Project: args.Project, Issue: args.Issue, Tier: tierOrDefault(args.Tier),
			ChunkSources: args.ChunkSources, Supersedes: parsedSupersedes,
		}

		return renderFactFrontmatter(f, when) + renderFactBody(f), nil
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
	osVault := &osVaultFS{}

	return LearnDeps{
		Now:           time.Now,
		Getenv:        os.Getenv,
		StatDir:       vaultFS.StatDir,
		InitVault:     func(path string) error { return initializeVault(vaultFS, path) },
		ListIDs:       vaultFS.ListIDs,
		ListBasenames: vaultFS.ListBasenames,
		Lock:          vaultFS.Lock,
		WriteNew:      vaultFS.WriteNew,
		Embedder:      sharedEmbedder,
		WriteSidecar:  vaultFS.WriteSidecar,
		LogWarning:    logWarningToStderrf,
		// Vocab assignment wiring: no-op when the vault has no term notes.
		// Uses stored member centroids (vocab.centroids.json) when present,
		// falling back to description embeddings per term.
		LoadTermVectors: func(vault string) ([]TermWithVector, error) {
			return loadAssignmentTermVectors(vault, osVault.ListMD, osVault.ReadFile)
		},
		ReadSidecar: osVault.ReadFile,
		WriteNote: func(path string, data []byte) error {
			return atomicWriteFile(path, data, vocabNotePerm)
		},
		// ListMD provides full .md filenames for the vocab trigger scan.
		// Must use ListMD (not ListBasenames) — ListBasenames strips .md and
		// filters to Luhmann IDs, causing 100% false-fire on the untagged trigger.
		ListMD: osVault.ListMD,
	}
}

func renderFactBody(f factFields) string {
	formula := fmt.Sprintf(
		"Information learned: when in %s, %s %s %s.\n",
		stripLeadingWhen(f.Situation), f.Subject, f.Predicate, f.Object,
	)

	return formula + "\n" + renderSupersedes(f.Supersedes)
}

func renderFactFrontmatter(f factFields, when time.Time) string {
	return marshalFrontmatter(factFrontmatterDoc{
		Type:       typeFact,
		Tier:       f.Tier,
		Situation:  f.Situation,
		Subject:    f.Subject,
		Predicate:  f.Predicate,
		Object:     f.Object,
		Luhmann:    quotedString(f.Luhmann),
		Created:    when.Format(dateFormat),
		Source:     f.Source,
		Project:    f.Project,
		Issue:      quotedString(f.Issue),
		Sources:    f.ChunkSources,
		Supersedes: f.Supersedes,
	})
}

func renderFeedbackBody(f feedbackFields) string {
	formula := fmt.Sprintf(
		"Lesson learned: when %s, %s.\n",
		stripLeadingWhen(f.Situation), f.Action,
	)

	return formula + "\n" + renderSupersedes(f.Supersedes)
}

func renderFeedbackFrontmatter(f feedbackFields, when time.Time) string {
	return marshalFrontmatter(feedbackFrontmatterDoc{
		Type:       typeFeedback,
		Tier:       f.Tier,
		Situation:  f.Situation,
		Behavior:   f.Behavior,
		Impact:     f.Impact,
		Action:     f.Action,
		Luhmann:    quotedString(f.Luhmann),
		Created:    when.Format(dateFormat),
		Source:     f.Source,
		Project:    f.Project,
		Issue:      quotedString(f.Issue),
		Sources:    f.ChunkSources,
		Supersedes: f.Supersedes,
	})
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

func runLearnFromFactArgs(ctx context.Context, a LearnFactArgs, stdout io.Writer) error {
	deps := newOsLearnDeps()

	return runLearn(ctx, LearnArgs{
		Type:         typeFact,
		Slug:         a.Slug,
		Vault:        a.Vault,
		Target:       a.Target,
		Position:     a.Position,
		Source:       a.Source,
		Project:      a.Project,
		Issue:        a.Issue,
		Tier:         a.Tier,
		Supersedes:   a.Supersedes,
		ChunkSources: a.ChunkSources,
		Situation:    a.Situation,
		Subject:      a.Subject,
		Predicate:    a.Predicate,
		Object:       a.Object,
	}, deps, stdout)
}

func runLearnFromFeedbackArgs(ctx context.Context, a LearnFeedbackArgs, stdout io.Writer) error {
	deps := newOsLearnDeps()

	return runLearn(ctx, LearnArgs{
		Type:         typeFeedback,
		Slug:         a.Slug,
		Vault:        a.Vault,
		Target:       a.Target,
		Position:     a.Position,
		Source:       a.Source,
		Project:      a.Project,
		Issue:        a.Issue,
		Tier:         a.Tier,
		Supersedes:   a.Supersedes,
		ChunkSources: a.ChunkSources,
		Situation:    a.Situation,
		Behavior:     a.Behavior,
		Impact:       a.Impact,
		Action:       a.Action,
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
// L2 — the active write tier for notes. Extracted so both the fact and
// feedback branches of assembleLearnContent share one branch point.
func tierOrDefault(tier string) string {
	if tier == "" {
		return tierL2
	}

	return tier
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
// of the recognised values. L2 is the active write tier; L1 and L3 are
// accepted as legacy values for backward-compat with manually-edited notes
// (DECISION-4: keep acceptance, remove write defaults).
func validateTier(tier string) error {
	if tier == "" {
		return nil
	}

	switch tier {
	case "L1", tierL2, "L3":
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

	content, contentErr := assembleLearnContent(args, luhmann, when)
	if contentErr != nil {
		return "", fmt.Errorf("learn: %w", contentErr)
	}

	writeErr := deps.WriteNew(path, []byte(content))
	if writeErr != nil {
		return "", fmt.Errorf("learn: writing %s: %w", path, writeErr)
	}

	autoEmbedNote(ctx, deps, path, content)
	applyVocabAssignmentAfterLearn(deps, vault, path, content)

	return path, nil
}
