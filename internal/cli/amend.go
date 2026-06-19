package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/vaultgraph"
)

// AmendArgs holds parsed flags for `engram amend`. ChunksDir configures where
// chunk indexes live (like IngestArgs.ChunksDir — path config belongs on Args,
// not Deps).
type AmendArgs struct {
	Vault        string   `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"` //nolint:lll // single unbreakable struct-tag string
	Target       string   `targ:"flag,name=target,required,desc=Luhmann id or basename of the note to amend (required)"`
	Relations    []string `targ:"flag,name=relation,desc=relation <target>|<rationale> for Related to: (repeatable)"`
	ChunkSources []string `targ:"flag,name=chunk-source,desc=chunk id (source#anchor) merged into sources: (repeatable)"`
	ChunksDir    string   `targ:"flag,name=chunks-dir,desc=chunk index dir (default $XDG_DATA_HOME/engram/chunks)"`
	// Content flags — only supplied fields are overwritten.
	Situation string `targ:"flag,name=situation,desc=replace situation (optional)"`
	Subject   string `targ:"flag,name=subject,desc=replace subject (fact; optional)"`
	Predicate string `targ:"flag,name=predicate,desc=replace predicate (fact; optional)"`
	Object    string `targ:"flag,name=object,desc=replace object (fact; optional)"`
	Behavior  string `targ:"flag,name=behavior,desc=replace behavior (feedback; optional)"`
	Impact    string `targ:"flag,name=impact,desc=replace impact (feedback; optional)"`
	Action    string `targ:"flag,name=action,desc=replace action (feedback; optional)"`
	Activate  bool   `targ:"flag,name=activate,desc=bump LastUsed on the sidecar (optional)"`
}

// AmendDeps holds injected I/O dependencies for RunAmend. Path configuration
// (ChunksDir) lives on AmendArgs, not here.
//
// LoadChunkIDs is DI-compliant: it takes injected listIndexes and readFile
// functions (matching buildChunkIDSet from Component 2) and returns a
// map[string]bool keyed by "source#anchor". The production wiring in
// newOsAmendDeps supplies os.ReadDir/os.ReadFile via closures.
type AmendDeps struct {
	Scan          func(vault string) ([]vaultgraph.Note, error)
	Read          func(path string) ([]byte, error)
	Write         func(path string, data []byte) error
	Embedder      embed.Embedder
	Now           func() time.Time
	ListBasenames func(vault string) ([]string, error)
	LoadChunkIDs  func(
		chunksDir string,
		listIndexes func(dir string) ([]string, error),
		readFile func(path string) ([]byte, error),
	) (map[string]bool, error)
	ListIndexes func(dir string) ([]string, error)
	LogWarning  func(string, ...any)
}

// RunAmend modifies a note in place. It merges --relation links into the
// "Related to:" body section (idempotent), merges --chunk-source ids into the
// frontmatter "sources:" list (idempotent), and overwrites only the supplied
// content fields. Re-embeds only when content changed. --activate bumps
// LastUsed in the same write.
func RunAmend(ctx context.Context, args AmendArgs, deps AmendDeps, stdout io.Writer) error {
	notes, scanErr := deps.Scan(args.Vault)
	if scanErr != nil {
		return fmt.Errorf("amend: scan: %w", scanErr)
	}

	relPath, findErr := findNote(notes, args.Target)
	if findErr != nil {
		return fmt.Errorf("%w: %q", errAmendNoteNotFound, args.Target)
	}

	full := filepath.Join(args.Vault, relPath)

	raw, readErr := deps.Read(full)
	if readErr != nil {
		return fmt.Errorf("amend: read %s: %w", relPath, readErr)
	}

	chunkErr := validateChunkSources(args, deps)
	if chunkErr != nil {
		return chunkErr
	}

	resolvedRelations, relErr := resolveAmendRelations(args, deps)
	if relErr != nil {
		return relErr
	}

	amended, contentChanged, amendErr := amendContent(raw, args, resolvedRelations)
	if amendErr != nil {
		return amendErr
	}

	writeErr := deps.Write(full, []byte(amended))
	if writeErr != nil {
		return fmt.Errorf("amend: write %s: %w", relPath, writeErr)
	}

	reEmbedAndActivate(ctx, args, deps, full, relPath, amended, contentChanged)

	_, _ = fmt.Fprintln(stdout, full)

	return nil
}

// unexported variables.
var (
	errAmendNoFrontmatter   = errors.New("amend: note has no parseable frontmatter")
	errAmendNoteNotFound    = errors.New("amend: note not found")
	errAmendUnknownType     = errors.New("amend: unknown note type")
	errAmendUnresolvedChunk = errors.New("amend: unresolved chunk-source id")
)

// fieldOverride pairs a mutable note field with the incoming replacement value.
type fieldOverride struct {
	current  *string
	incoming string
}

// typedAmend captures the type-specific behavior of an amend so the generic
// driver applyTypedAmend can share the parse → override → merge → render flow
// across fact and feedback notes without duplicating it.
//
//   - kind:     human label used in the parse-error message
//   - created:  reads the decoded doc's created: date (the generic T cannot
//     reach doc.Created directly, so the concrete accessor is supplied here —
//     avoiding a second unmarshal of the frontmatter just to recover the date)
//   - override: applies args' field overrides to the decoded doc, returning
//     the per-field changed flags; it also merges chunk sources into the doc
//   - render:   produces (frontmatter+body) for the (possibly) updated doc,
//     re-rendering the body only when contentChanged is true
type typedAmend[T any] struct {
	kind     string
	created  func(doc T) string
	override func(doc *T, args AmendArgs) bool
	render   func(doc T, when time.Time, body string, contentChanged bool) string
}

// amendContent applies all amendments to raw note bytes. Returns the
// updated content, whether the semantic content changed (triggers re-embed),
// and any error. Link/provenance-only changes do NOT set contentChanged.
func amendContent(raw []byte, args AmendArgs, resolvedRelations []string) (string, bool, error) {
	frontmatter, ok := splitFrontmatter(raw)
	if !ok {
		return "", false, errAmendNoFrontmatter
	}

	noteType := peekNoteType(frontmatter)
	body := embed.ExtractBody(raw)

	// merge relations into body
	bodyStr := string(body)
	if len(resolvedRelations) > 0 {
		bodyStr = mergeRelatedSection(bodyStr, resolvedRelations)
	}

	// merge chunk sources into frontmatter + apply field overrides
	updated, contentChanged, fieldErr := applyFieldReplacement(raw, args, bodyStr, noteType)
	if fieldErr != nil {
		return "", false, fieldErr
	}

	return updated, contentChanged, nil
}

// applyFactAmend overrides supplied fact fields, merges chunk-source provenance,
// and re-renders the note (Issue round-trips via quotedString — CA-11).
func applyFactAmend(frontmatter []byte, args AmendArgs, body string) (string, bool, error) {
	return applyTypedAmend(frontmatter, args, body, typedAmend[factFrontmatterDoc]{
		kind:     "fact",
		created:  func(doc factFrontmatterDoc) string { return doc.Created },
		override: overrideFactFields,
		render:   renderAmendedFact,
	})
}

// applyFeedbackAmend mirrors applyFactAmend for feedback notes (behavior/impact/
// action fields; Issue round-trips via quotedString — CA-11).
func applyFeedbackAmend(frontmatter []byte, args AmendArgs, body string) (string, bool, error) {
	return applyTypedAmend(frontmatter, args, body, typedAmend[feedbackFrontmatterDoc]{
		kind:     "feedback",
		created:  func(doc feedbackFrontmatterDoc) string { return doc.Created },
		override: overrideFeedbackFields,
		render:   renderAmendedFeedback,
	})
}

// applyFieldOverrides applies each override in place and reports whether any
// field actually changed (a supplied value differing from the current one).
func applyFieldOverrides(overrides []fieldOverride) bool {
	changed := false

	for _, o := range overrides {
		newValue, didChange := overrideField(*o.current, o.incoming)
		*o.current = newValue
		changed = changed || didChange
	}

	return changed
}

// applyFieldReplacement parses the note frontmatter, applies field overrides and
// provenance merge, rebuilds the frontmatter, and reassembles with the (already
// relation-merged) body. contentChanged is true only when a semantic field
// (situation/subject/predicate/object/behavior/impact/action) changed.
func applyFieldReplacement(raw []byte, args AmendArgs, body, noteType string) (string, bool, error) {
	frontmatter, _ := splitFrontmatter(raw) // already validated upstream

	switch noteType {
	case typeFact:
		return applyFactAmend(frontmatter, args, body)
	case typeFeedback:
		return applyFeedbackAmend(frontmatter, args, body)
	default:
		return "", false, fmt.Errorf("%w: %q", errAmendUnknownType, noteType)
	}
}

// applyTypedAmend is the shared fact/feedback amend driver. It unmarshals the
// frontmatter into T, preserves the created date, applies the type-specific
// overrides + provenance merge, and renders. contentChanged is true only when a
// semantic field actually changed value, so relation-only or provenance-only
// amends do not trigger a re-embed (D3).
func applyTypedAmend[T any](frontmatter []byte, args AmendArgs, body string, spec typedAmend[T]) (string, bool, error) {
	var doc T

	err := yaml.Unmarshal(frontmatter, &doc)
	if err != nil {
		return "", false, fmt.Errorf("amend: parsing %s frontmatter: %w", spec.kind, err)
	}

	when, createdErr := parseCreated(spec.created(doc))
	if createdErr != nil {
		return "", false, createdErr
	}

	contentChanged := spec.override(&doc, args)

	return spec.render(doc, when, body, contentChanged), contentChanged, nil
}

// mergeChunkSources returns a deduped union of existing and incoming chunk ids.
func mergeChunkSources(existing, incoming []string) []string {
	seen := make(map[string]struct{}, len(existing)+len(incoming))
	out := make([]string, 0, len(existing)+len(incoming))

	for _, id := range existing {
		if _, dup := seen[id]; !dup {
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}

	for _, id := range incoming {
		if _, dup := seen[id]; !dup {
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}

	return out
}

// mergeRelatedSection parses the existing "Related to:" block from body,
// deduplicates with incoming relations, and returns the updated body with
// only new relations appended. Incoming relations must already be in
// "basename|rationale" resolved form. Existing bullets "- [[basename]] — ..."
// are parsed for their basename to detect duplicates.
func mergeRelatedSection(body string, incoming []string) string {
	idx := strings.LastIndex(body, relatedSectionMarker)

	var head, existingSection string

	if idx == -1 {
		head = body
	} else {
		head = body[:idx]
		existingSection = body[idx:]
	}

	// collect existing basenames from bullets. Normalize a trailing ".md": a
	// hand-edited bullet may carry "[[foo.md]]" while resolveRelationTargetsStrict
	// yields the ".md"-stripped ListBasenames form, so dedup on the bare basename
	// keeps the merge idempotent regardless of which form the note already holds.
	existing := map[string]struct{}{}

	for line := range strings.SplitSeq(existingSection, "\n") {
		sub := wikilinkRE.FindStringSubmatch(line)
		if sub != nil {
			existing[strings.TrimSuffix(sub[1], ".md")] = struct{}{}
		}
	}

	// build new bullets for relations not already present
	newBullets := make([]string, 0, len(incoming))

	for _, rel := range incoming {
		target, rationale, _ := strings.Cut(rel, "|")
		target = strings.TrimSpace(target)

		if _, dup := existing[strings.TrimSuffix(target, ".md")]; dup {
			continue
		}

		bullet := "- [[" + target + "]]"
		if r := strings.TrimSpace(rationale); r != "" {
			bullet += " — " + r + "."
		}

		newBullets = append(newBullets, bullet)
	}

	if len(newBullets) == 0 {
		return body // no change
	}

	if idx == -1 {
		tail := relatedSectionMarker + "\n" + strings.Join(newBullets, "\n") + "\n"

		return strings.TrimRight(body, "\n") + "\n\n" + tail
	}

	trimmed := strings.TrimRight(existingSection, "\n")

	return head + trimmed + "\n" + strings.Join(newBullets, "\n") + "\n"
}

// newOsAmendDeps wires RunAmend to the real filesystem + bundled embedder.
// ChunksDir flows through AmendArgs, not here.
func newOsAmendDeps() AmendDeps {
	const perm = 0o600

	return AmendDeps{
		Scan: func(vault string) ([]vaultgraph.Note, error) {
			return vaultgraph.ScanVault(&osVaultFS{}, vault)
		},
		Read: (&osVaultFS{}).ReadFile,
		Write: func(path string, data []byte) error {
			err := os.WriteFile(path, data, perm)
			if err != nil {
				return fmt.Errorf("write %s: %w", path, err)
			}

			return nil
		},
		Embedder: sharedEmbedder,
		Now:      time.Now,
		ListBasenames: func(vault string) ([]string, error) {
			return (&osLearnFS{}).ListBasenames(vault)
		},
		LoadChunkIDs: buildChunkIDSet,
		// listJSONLIndexes (query_chunks.go) lists *.jsonl chunk indexes, treats
		// an absent dir as empty (not an error), and never matches manifest.json
		// (it is not a .jsonl file) — exactly the contract needed here, so reuse
		// it rather than hand-roll a closure.
		ListIndexes: listJSONLIndexes,
		LogWarning:  logWarningToStderrf,
	}
}

// overrideFactFields merges provenance into the fact doc and applies any
// supplied situation/subject/predicate/object overrides, reporting whether a
// semantic field changed.
func overrideFactFields(doc *factFrontmatterDoc, args AmendArgs) bool {
	doc.Sources = mergeChunkSources(doc.Sources, args.ChunkSources)

	return applyFieldOverrides([]fieldOverride{
		{&doc.Situation, args.Situation},
		{&doc.Subject, args.Subject},
		{&doc.Predicate, args.Predicate},
		{&doc.Object, args.Object},
	})
}

// overrideFeedbackFields merges provenance into the feedback doc and applies any
// supplied situation/behavior/impact/action overrides, reporting whether a
// semantic field changed.
func overrideFeedbackFields(doc *feedbackFrontmatterDoc, args AmendArgs) bool {
	doc.Sources = mergeChunkSources(doc.Sources, args.ChunkSources)

	return applyFieldOverrides([]fieldOverride{
		{&doc.Situation, args.Situation},
		{&doc.Behavior, args.Behavior},
		{&doc.Impact, args.Impact},
		{&doc.Action, args.Action},
	})
}

// overrideField returns the incoming value (and changed=true) when it is
// non-empty and differs from current; otherwise it returns current unchanged.
// Centralizing the "only overwrite when supplied and different" rule keeps the
// fact/feedback amend paths free of repeated guard blocks.
func overrideField(current, incoming string) (string, bool) {
	if incoming != "" && incoming != current {
		return incoming, true
	}

	return current, false
}

// reEmbedAndActivate re-embeds the note's sidecar when content changed and an
// embedder is wired, then bumps LastUsed when --activate is set. Both steps
// warn-and-continue on failure: the note write already succeeded, so a missing
// sidecar is recoverable later.
func reEmbedAndActivate(
	ctx context.Context, args AmendArgs, deps AmendDeps, full, relPath, amended string, contentChanged bool,
) {
	if contentChanged && deps.Embedder != nil {
		embedErr := writeAmendedSidecar(ctx, deps, full, amended)
		if embedErr != nil && deps.LogWarning != nil {
			deps.LogWarning("amend: embed failed for %s: %v", relPath, embedErr)
		}
	}

	if args.Activate && deps.Now != nil {
		date := deps.Now().Format(noteDateFormat)
		sidecarPath := embed.SidecarPath(full)

		bumpErr := bumpLastUsed(sidecarPath, date, deps.Read, deps.Write)
		if bumpErr != nil && deps.LogWarning != nil {
			deps.LogWarning("amend: activate failed for %s: %v", relPath, bumpErr)
		}
	}
}

// relatedTailFromBody extracts the "Related to:\n..." suffix from an
// already-relation-merged body string. Returns "" when absent.
func relatedTailFromBody(body string) string {
	idx := strings.LastIndex(body, relatedSectionMarker)
	if idx == -1 {
		return ""
	}

	return body[idx:]
}

// renderAmendedFact re-renders a fact note from the (possibly updated) doc,
// rebuilding the body formula only when a semantic field changed.
func renderAmendedFact(doc factFrontmatterDoc, when time.Time, body string, contentChanged bool) string {
	f := factFields{
		Situation: doc.Situation, Subject: doc.Subject, Predicate: doc.Predicate,
		Object: doc.Object, Luhmann: string(doc.Luhmann), Source: doc.Source,
		Project: doc.Project, Issue: string(doc.Issue), Tier: doc.Tier, ChunkSources: doc.Sources,
	}
	if contentChanged {
		body = renderFactBody(f, relatedTailFromBody(body))
	}

	return renderFactFrontmatter(f, when) + body
}

// renderAmendedFeedback re-renders a feedback note from the (possibly updated)
// doc, rebuilding the body formula only when a semantic field changed.
func renderAmendedFeedback(doc feedbackFrontmatterDoc, when time.Time, body string, contentChanged bool) string {
	f := feedbackFields{
		Situation: doc.Situation, Behavior: doc.Behavior, Impact: doc.Impact,
		Action: doc.Action, Luhmann: string(doc.Luhmann), Source: doc.Source,
		Project: doc.Project, Issue: string(doc.Issue), Tier: doc.Tier, ChunkSources: doc.Sources,
	}
	if contentChanged {
		body = renderFeedbackBody(f, relatedTailFromBody(body))
	}

	return renderFeedbackFrontmatter(f, when) + body
}

// resolveAmendRelations resolves --relation targets to full basenames, failing
// loud on unresolved ids. Returns nil when no relations were supplied.
func resolveAmendRelations(args AmendArgs, deps AmendDeps) ([]string, error) {
	if len(args.Relations) == 0 {
		return nil, nil
	}

	basenames, bErr := deps.ListBasenames(args.Vault)
	if bErr != nil {
		return nil, fmt.Errorf("amend: listing basenames: %w", bErr)
	}

	resolved, strictErr := resolveRelationTargetsStrict(args.Relations, basenames)
	if strictErr != nil {
		return nil, fmt.Errorf("amend: %w", strictErr)
	}

	return resolved, nil
}

// validateChunkSources loads the chunk-id set and fails loud when any
// --chunk-source id is unknown. A no-op when no chunk sources were supplied or
// no loader is wired.
func validateChunkSources(args AmendArgs, deps AmendDeps) error {
	if len(args.ChunkSources) == 0 || deps.LoadChunkIDs == nil {
		return nil
	}

	chunkIDs, loadErr := deps.LoadChunkIDs(args.ChunksDir, deps.ListIndexes, deps.Read)
	if loadErr != nil {
		return fmt.Errorf("amend: loading chunk ids: %w", loadErr)
	}

	for _, id := range args.ChunkSources {
		if !chunkIDs[id] {
			return fmt.Errorf("%w: %q", errAmendUnresolvedChunk, id)
		}
	}

	return nil
}

// writeAmendedSidecar re-embeds the amended note and writes its sidecar.
// Modeled on writeResituatedSidecar in resituate.go. Embed and write failures
// are returned to the caller (which may choose to warn-and-continue for amend).
func writeAmendedSidecar(ctx context.Context, deps AmendDeps, notePath, content string) error {
	sidecar, embErr := embed.BuildSidecar(ctx, deps.Embedder, []byte(content))
	if embErr != nil {
		return fmt.Errorf("amend: embedding %s: %w", notePath, embErr)
	}

	writeErr := deps.Write(embed.SidecarPath(notePath), embed.MarshalSidecar(sidecar))
	if writeErr != nil {
		return fmt.Errorf("amend: writing sidecar for %s: %w", notePath, writeErr)
	}

	return nil
}
