package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/toejough/engram/internal/vaultgraph"
)

// RegisterSkillsArgs holds the parsed flags for `engram register-skills`
// (skill-runbook-registration). Adopt entries are `<key>=<note-ref>`
// strings, parsed by parseAdoptFlags into SkillRegistrationArgs' map — the
// CLI-facing repeatable-flag shape differs from the internal args' shape,
// same split as LearnFactArgs/LearnFeedbackArgs/LearnRunbookArgs vs. the
// shared internal LearnArgs.
type RegisterSkillsArgs struct {
	Vault     string   `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"`                                                                     //nolint:lll // unbreakable env+desc struct-tag string
	VaultName string   `targ:"flag,name=vault-name,env=ENGRAM_VAULT_NAME,desc=vault name stamped on a newly-registered note's vault: field (default \"personal\")"`                             //nolint:lll // unbreakable env+desc struct-tag string
	SkillsDir []string `targ:"flag,name=skills-dir,desc=preview a skills dir instead of the default folders (repeatable; read-only: implies --dry-run and refuses --accept/--decline/--adopt)"` //nolint:lll // unbreakable struct-tag string
	DryRun    bool     `targ:"flag,name=dry-run,desc=list offers without prompting or writing"`
	Accept    []string `targ:"flag,name=accept,desc=accept the matching offers without prompting: a skill key or prefix* pattern or @<scope-id> (repeatable)"`   //nolint:lll // unbreakable struct-tag string
	Decline   []string `targ:"flag,name=decline,desc=decline the matching offers without prompting: a skill key or prefix* pattern or @<scope-id> (repeatable)"` //nolint:lll // unbreakable struct-tag string
	Adopt     []string `targ:"flag,name=adopt,desc=adopt an existing runbook note as a skill key's note: <key>=<note-ref> (repeatable)"`                         //nolint:lll // unbreakable struct-tag string
}

// SkillRegistrationArgs holds RunSkillRegistration's inputs: the resolved
// vault/vault-name, the home the default source set is resolved from, any
// `--skills-dir` preview dirs, the dry-run flag, and any explicit
// --accept/--decline/--adopt answers (skill-runbook-registration).
type SkillRegistrationArgs struct {
	Vault     string
	VaultName string
	// Home is the user's home dir: the default source set is resolved from
	// it (ResolveSkillSources, design D1), a note's skill_source is written
	// `~`-relative to it, and removal eligibility expands it (design D5, D8).
	Home string
	// PreviewDirs are `--skills-dir`'s dirs: they replace the default set,
	// each scanned with the Claude-user rules (keys `claude:<n>`), and the
	// run is a read-only preview (design D9).
	PreviewDirs []string
	DryRun      bool
	Accept      []string
	Decline     []string
	// Adopt maps a skill key to the note-ref its existing runbook note is
	// adopted from (design D6). Parsed from RegisterSkillsArgs.Adopt's
	// "<key>=<note-ref>" strings by the CLI-wiring layer.
	Adopt map[string]string
}

// SkillRegistrationDeps holds RunSkillRegistration's injected capabilities:
// resolving the default source set (Sources, Getwd), listing/reading vault
// notes for the offer comparison, the interactive-prompt gate, and the three
// accept-path deps structs (Learn for Register, Accept for Refresh/Remove,
// Adopt for --adopt) that skillreg_accept.go's actions already define.
type SkillRegistrationDeps struct {
	// Sources backs ResolveSkillSources — the read-only filesystem (also the
	// one `--skills-dir`'s dirs are scanned with) and the git command runner
	// for the project probe (production: EdgeFS and the CLI Commander).
	Sources SkillSourceDeps
	// Getwd supplies the working directory the default source set is
	// resolved from (production: os.Getwd). The update hook's re-exec child
	// inherits the parent's working directory, so both commands resolve
	// from the directory the user ran them in.
	Getwd func() (string, error)
	// ListMD lists the vault's full .md filenames, for the offer comparison
	// (CompareSkillOffers/FindSkillNote) — same shape as LearnDeps.ListMD.
	ListMD func(vault string) ([]string, error)
	// IsTerminal reports whether stdin is an interactive terminal — the
	// interactive-prompt gate (skill-runbook-registration: "Registration
	// SHALL never prompt or write without a terminal"). nil is treated as
	// non-interactive.
	IsTerminal func() bool
	// Stdin supplies interactive prompt answers, read one line per offer via
	// an internally-owned bufio.Scanner. Unused when IsTerminal is false (or
	// nil) or every offer has an explicit --accept/--decline answer.
	Stdin io.Reader

	// Accept backs RefreshSkill/RemoveSkill for accepted refresh/removal
	// offers. Its Read/Write also back the offer comparison's vault-note
	// reads and skill-registrations.json's read/write (RecordSkillDeclined,
	// ReadSkillRegistrations) — the same "read/atomically-write a
	// vault-joined path" capability both need.
	Accept SkillAcceptDeps
	// Adopt backs AdoptSkillNote for --adopt entries.
	Adopt SkillAdoptDeps
	// Learn backs RegisterSkill for accepted registration offers.
	Learn LearnDeps
}

// RunSkillRegistration orchestrates `engram register-skills` and `engram
// update`'s post-deploy registration hook (skill-runbook-registration): it
// resolves the default source set (ResolveSkillSources, design D1 — the one
// definition both commands share) from args.Home and the working directory,
// runs any --adopt entries first, computes the
// register/refresh/remove offers against the vault, and then — for each
// offer, in order — acts on an explicit --accept/--decline answer, prompts
// interactively when stdin is a terminal and this isn't a dry run, or else
// leaves it unanswered. A dry run only previews offers and touches nothing.
//
// With PreviewDirs (`--skills-dir`, design D9) the run is a read-only
// preview of those dirs alone: any --accept, --decline or --adopt is refused
// with errSkillsDirReadOnly before anything is scanned, and no removal is
// offered.
func RunSkillRegistration(
	ctx context.Context, args SkillRegistrationArgs, deps SkillRegistrationDeps, stdout io.Writer,
) error {
	if len(args.PreviewDirs) > 0 {
		return previewSkillsDirs(args, deps, stdout)
	}

	answers, answersErr := ParseSkillAnswers(args.Accept, args.Decline)
	if answersErr != nil {
		return answersErr
	}

	sources, resolveErr := resolveDefaultSkillSources(ctx, args.Home, deps)
	if resolveErr != nil {
		return resolveErr
	}

	if args.DryRun {
		return previewSkillSources(args.Vault, args.Home, sources, false, deps, stdout)
	}

	return answerSkillSources(ctx, args, answers, sources, deps, stdout)
}

// unexported constants.
const (
	skillMDFilename = "SKILL.md"
	// skillRefreshPromptFormat, skillRegisterPromptFormat and
	// skillRemovePromptFormat are the per-offer prompts (design D4's table):
	// each takes the offer's key and offerPromptSource's " (<source>)", which
	// is empty only for a removal with no recorded skill_source.
	skillRefreshPromptFormat  = "Skill `%s`%s changed since its note was last synced. Update the note? [y/N] "
	skillRegisterPromptFormat = "Register skill `%s`%s as a vault runbook? [y/N] "
	// skillRegistrationAwaitingAnswerFormat is the one-line, non-interactive
	// summary of every offer left unanswered this run (skill-runbook-
	// registration: "Registration SHALL never prompt or write without a
	// terminal ... It SHALL print one line giving the total number of
	// outstanding offers, each scope's selector with its count, and the
	// `engram register-skills` command that answers them"; design D6). It
	// fills in the total, "offer"/"offers", and the comma-joined
	// `@<scope-id> <count>` labels; the `<key|prefix*|@scope>` placeholders
	// are literal command syntax.
	skillRegistrationAwaitingAnswerFormat = "engram: %d skill runbook %s awaiting an answer: %s — run " +
		"`engram register-skills` in a terminal, or `engram register-skills --accept <key|prefix*|@scope>` / " +
		"`--decline <key|prefix*|@scope>`\n"
	// skillRegistrationDryRunOfferFormat previews one offer under --dry-run,
	// indented under its scope header (design D6: "`--dry-run` prints `would
	// offer: <kind> <key> (<source>)` under scope headers").
	skillRegistrationDryRunOfferFormat = "  would offer: %s %s (%s)\n"
	// skillRegistrationNoOfferFormat reports a --accept/--decline token (key,
	// pattern or selector) matching no current offer: not an error, just a
	// one-line note, and registration continues (skill-runbook-registration:
	// "Registration SHALL be invocable standalone with explicit answers").
	skillRegistrationNoOfferFormat = "engram: no pending offer for skill %q — ignoring --%s\n"
	skillRemovePromptFormat        = "Skill `%s`%s is no longer found in its source. Remove its runbook note? [y/N] "
)

// unexported variables.
var (
	// errAdoptKeyConflict reports an --adopt <key>=<ref> entry whose key's
	// copies differ (design D4): there is no one file to mirror, and nothing
	// is written for a conflicted key.
	errAdoptKeyConflict = errors.New("register-skills: adopt: key conflict")
	// errAdoptSkillNotShipped reports an --adopt <key>=<ref> entry whose key
	// names no scanned, enabled skill — AdoptSkillNote needs the source
	// file's current bytes to render the note's body.
	errAdoptSkillNotShipped = errors.New("register-skills: adopt: key names no scanned skill")
	// errDuplicateAdoptKey reports a key named by more than one --adopt
	// entry.
	errDuplicateAdoptKey = errors.New("register-skills: adopt: key named by more than one --adopt entry")
	// errMalformedAdoptFlag reports a --adopt flag that isn't shaped
	// "<key>=<note-ref>" (both sides non-empty).
	errMalformedAdoptFlag = errors.New("register-skills: --adopt must be <key>=<note-ref>")
	// errSkillOfferSourceMissing reports an accepted Register or Refresh
	// offer whose key and source path name no candidate's note source.
	errSkillOfferSourceMissing = errors.New("register-skills: offer names no scanned source")
	// errSkillSourcesNeedWorkingDir reports a default-set run with no working
	// directory to resolve the project sources from.
	errSkillSourcesNeedWorkingDir = errors.New("register-skills: resolving the working directory")
	// errSkillsDirNeedsWorkingDir reports a relative --skills-dir with no
	// working directory to resolve it against.
	errSkillsDirNeedsWorkingDir = errors.New("register-skills: --skills-dir: resolving a relative dir")
	// errSkillsDirReadOnly refuses --accept, --decline and --adopt on a
	// --skills-dir run, before anything is scanned (design D9).
	errSkillsDirReadOnly = errors.New(
		"register-skills: --skills-dir is a read-only preview; it cannot be combined with " +
			"--accept, --decline or --adopt")
	// errUnknownSkillOfferKind guards acceptSkillOffer's switch — unreachable
	// in production since CompareSkillOffers only ever emits the three known
	// SkillOfferKind values.
	errUnknownSkillOfferKind = errors.New("register-skills: unknown offer kind")
)

// absolutePreviewDirs resolves each relative `--skills-dir` against the
// working directory, which getwd supplies (called only when needed).
func absolutePreviewDirs(dirs []string, getwd func() (string, error)) ([]string, error) {
	absolute := make([]string, 0, len(dirs))

	for _, dir := range dirs {
		if filepath.IsAbs(dir) {
			absolute = append(absolute, filepath.Clean(dir))

			continue
		}

		if getwd == nil {
			return nil, fmt.Errorf("%w %q: no working directory", errSkillsDirNeedsWorkingDir, dir)
		}

		cwd, cwdErr := getwd()
		if cwdErr != nil {
			return nil, fmt.Errorf("%w %q: %w", errSkillsDirNeedsWorkingDir, dir, cwdErr)
		}

		absolute = append(absolute, filepath.Join(cwd, dir))
	}

	return absolute, nil
}

// acceptSkillOffer dispatches an accepted offer to RegisterSkill, RefreshSkill,
// or RemoveSkill, by kind. A Register or Refresh offer mirrors the candidate
// it came from, looked up in noteSources; a lookup that misses is
// errSkillOfferSourceMissing, never a note with an empty key or slug.
func acceptSkillOffer(
	ctx context.Context,
	vault, vaultName string,
	offer SkillOffer,
	noteSources map[string]SkillNoteSource,
	deps SkillRegistrationDeps,
	stdout io.Writer,
) error {
	switch offer.Kind {
	case SkillOfferRegister, SkillOfferRefresh:
		return mirrorSkillOffer(ctx, vault, vaultName, offer, noteSources, deps, stdout)
	case SkillOfferRemove:
		return RemoveSkill(vault, offer.Basename, deps.Accept, stdout)
	default:
		return fmt.Errorf("%w: %s", errUnknownSkillOfferKind, offer.Kind)
	}
}

// adoptSourceFor returns the note source an --adopt entry for key mirrors:
// the precedence winner among key's enabled candidates (design D4). A key
// with no enabled candidate is errAdoptSkillNotShipped; a key whose copies
// conflict is errAdoptKeyConflict.
func adoptSourceFor(key string, sources ResolvedSkillSources, home string) (SkillNoteSource, error) {
	matching := make([]SkillCandidate, 0, 1)

	for _, candidate := range sources.Candidates {
		if candidate.Key == key {
			matching = append(matching, candidate)
		}
	}

	groups, comparison := dedupeSkillCandidates(matching)

	switch {
	case len(groups) == 0:
		return SkillNoteSource{}, fmt.Errorf("%w: %q", errAdoptSkillNotShipped, key)
	case groups[0].conflicted:
		return SkillNoteSource{}, fmt.Errorf("%w: %s", errAdoptKeyConflict, strings.Join(comparison.Conflicts, "; "))
	default:
		return NewSkillNoteSource(groups[0].members[0], home, sources.ResolvedHome), nil
	}
}

// answerSkillSources runs the --adopt entries over sources, then computes
// the offers and answers them through AnswerSkillOffers: an accepted offer
// is carried out by acceptSkillOffer, a declined one records its hash
// (RecordSkillDeclined).
//
// Every read that can fail the run — the decline state (a non-v2 file must
// write nothing, design D7), the vault listing and the offer comparison — is
// done before the first --adopt writes, so a failing read leaves the vault
// untouched. The decline state is read once and reused; the comparison is
// recomputed after the adopts, since they rename and re-key notes.
func answerSkillSources(
	ctx context.Context,
	args SkillRegistrationArgs,
	answers SkillAnswers,
	sources ResolvedSkillSources,
	deps SkillRegistrationDeps,
	stdout io.Writer,
) error {
	declined, declinedErr := readSkillDeclines(args.Vault, deps)
	if declinedErr != nil {
		return declinedErr
	}

	if len(args.Adopt) > 0 {
		_, preflightErr := computeSkillOffers(args.Vault, args.Home, sources, false, declined, deps)
		if preflightErr != nil {
			return preflightErr
		}
	}

	adoptErr := runSkillAdoptions(ctx, args, sources, deps.Adopt, stdout)
	if adoptErr != nil {
		return adoptErr
	}

	comparison, offersErr := computeSkillOffers(args.Vault, args.Home, sources, false, declined, deps)
	if offersErr != nil {
		return offersErr
	}

	// The scan and key warnings are printed on every path, not only the
	// preview (skill-runbook-registration: skipped entries are skipped
	// "with a warning").
	comparison.Warnings = append(slices.Clone(sources.Warnings), comparison.Warnings...)
	noteSources := skillNoteSourcesByRef(sources, args.Home)

	return AnswerSkillOffers(SkillOfferAnswering{
		Comparison:  comparison,
		Answers:     answers,
		Interactive: deps.IsTerminal != nil && deps.IsTerminal(),
		Stdin:       deps.Stdin,
		Accept: func(offer SkillOffer) error {
			return acceptSkillOffer(ctx, args.Vault, args.VaultName, offer, noteSources, deps, stdout)
		},
		Decline: func(offer SkillOffer) error {
			return RecordSkillDeclined(args.Vault, offer.Key, offer.Hash, deps.Accept.Read, deps.Accept.Write)
		},
	}, stdout)
}

// computeSkillOffers lists the vault and runs CompareSkillOffers over
// sources against the declined state (readSkillDeclines) — the
// offer-computation shared by the dry-run preview and the answer loop. home
// expands a note's `~`-relative skill_source; noRemovals suppresses removal
// offers (`--skills-dir`).
func computeSkillOffers(
	vault, home string,
	sources ResolvedSkillSources,
	noRemovals bool,
	declined map[string]string,
	deps SkillRegistrationDeps,
) (SkillOfferComparison, error) {
	names, listErr := deps.ListMD(vault)
	if listErr != nil {
		return SkillOfferComparison{}, fmt.Errorf("register-skills: listing vault: %w", listErr)
	}

	comparison, offersErr := CompareSkillOffers(SkillOfferInput{
		Vault:      vault,
		Names:      names,
		ReadFile:   deps.Accept.Read,
		Declined:   declined,
		Home:       home,
		Sources:    sources,
		NoRemovals: noRemovals,
	})
	if offersErr != nil {
		return SkillOfferComparison{}, fmt.Errorf("register-skills: %w", offersErr)
	}

	return comparison, nil
}

// mirrorSkillOffer carries out an accepted Register or Refresh offer: it
// looks up the note source of the candidate the offer came from, and a miss
// is errSkillOfferSourceMissing — nothing is written.
func mirrorSkillOffer(
	ctx context.Context,
	vault, vaultName string,
	offer SkillOffer,
	noteSources map[string]SkillNoteSource,
	deps SkillRegistrationDeps,
	stdout io.Writer,
) error {
	source, found := noteSources[skillCandidateRef(offer.Key, offer.SourcePath)]
	if !found {
		return fmt.Errorf("%w: %s %s (%s)", errSkillOfferSourceMissing, offer.Kind, offer.Key, offer.SourcePath)
	}

	if offer.Kind == SkillOfferRegister {
		return RegisterSkill(ctx, vault, vaultName, source, deps.Learn, stdout)
	}

	return RefreshSkill(ctx, vault, source, offer.Basename, deps.Accept, stdout)
}

// newSkillRegistrationDeps composes SkillRegistrationDeps from the injected
// edge Deps (pure composition — no direct I/O; #700).
func newSkillRegistrationDeps(d Deps) SkillRegistrationDeps {
	vfs := newVaultFS(d.FS)

	return SkillRegistrationDeps{
		Sources:    SkillSourceDeps{FS: d.FS, Commander: d.Commander},
		Getwd:      d.Getwd,
		ListMD:     vfs.ListMD,
		IsTerminal: d.IsTerminal,
		Stdin:      d.Stdin,
		Accept: SkillAcceptDeps{
			Lock:     vaultLockFromLocker(d.Lock),
			Read:     vfs.ReadFile,
			Write:    writeAtomicWithPathFromFS(d.FS),
			Remove:   d.FS.Remove,
			Embedder: d.Embed,
		},
		Adopt: SkillAdoptDeps{
			Lock: vaultLockFromLocker(d.Lock),
			Scan: func(vault string) ([]vaultgraph.Note, error) {
				return vaultgraph.ScanVault(vfs, vault)
			},
			Rename:   newRenameRewriteDeps(d),
			Embedder: d.Embed,
		},
		Learn: newLearnDeps(d),
	}
}

// parseAdoptFlags parses RegisterSkillsArgs.Adopt's repeatable
// "<key>=<note-ref>" strings into SkillRegistrationArgs.Adopt's map. The key
// ends at the first `=`. A key named by more than one entry is refused
// (errDuplicateAdoptKey) rather than letting a later flag silently replace
// an earlier one.
func parseAdoptFlags(raw []string) (map[string]string, error) {
	adopt := make(map[string]string, len(raw))

	for _, entry := range raw {
		key, ref, found := strings.Cut(entry, "=")
		if !found || key == "" || ref == "" {
			return nil, fmt.Errorf("%w: %q", errMalformedAdoptFlag, entry)
		}

		if prior, seen := adopt[key]; seen {
			return nil, fmt.Errorf("%w: %q names both %q and %q", errDuplicateAdoptKey, key, prior, ref)
		}

		adopt[key] = ref
	}

	return adopt, nil
}

// previewSkillSources previews every computed offer under its scope header
// (PreviewSkillOffers), after sources' scan warnings, and writes nothing,
// prompts nothing (skill-runbook-registration: "`--dry-run` SHALL list
// every offer and write nothing"). --adopt entries are not run under a
// preview either — a dry run touches no vault state.
func previewSkillSources(
	vault, home string, sources ResolvedSkillSources, noRemovals bool, deps SkillRegistrationDeps, stdout io.Writer,
) error {
	declined, declinedErr := readSkillDeclines(vault, deps)
	if declinedErr != nil {
		return declinedErr
	}

	comparison, offersErr := computeSkillOffers(vault, home, sources, noRemovals, declined, deps)
	if offersErr != nil {
		return offersErr
	}

	comparison.Warnings = append(slices.Clone(sources.Warnings), comparison.Warnings...)

	return PreviewSkillOffers(stdout, comparison)
}

// previewSkillsDirs is a `--skills-dir` run (design D9): it refuses any
// --accept, --decline or --adopt before scanning, then scans each preview
// dir with the Claude-user rules (keys `claude:<n>`) in place of the default
// set, and previews the offers with no removal offer.
func previewSkillsDirs(args SkillRegistrationArgs, deps SkillRegistrationDeps, stdout io.Writer) error {
	if len(args.Accept) > 0 || len(args.Decline) > 0 || len(args.Adopt) > 0 {
		return errSkillsDirReadOnly
	}

	var sources ResolvedSkillSources

	for _, dir := range args.PreviewDirs {
		mergeSkillScanResult(&sources.SkillScanResult, ScanClaudeUserSkills(deps.Sources.FS, dir))
	}

	keyed, keyWarnings := AssignSkillKeys(sources.Candidates)
	sources.Candidates = keyed
	sources.Warnings = append(sources.Warnings, keyWarnings...)

	return previewSkillSources(args.Vault, args.Home, sources, true, deps, stdout)
}

// promptForOffer writes offer's prompt to stdout and reads one line from
// scanner: y/yes (case-insensitive) accepts, and any other line declines.
// End of input (scanner.Scan returning false) is no answer at all
// (SkillAnswerNone): the offer is neither accepted nor declined, like the
// grouped prompt's skip (ruling R29).
func promptForOffer(offer SkillOffer, scanner *bufio.Scanner, stdout io.Writer) SkillAnswer {
	_, _ = fmt.Fprintf(stdout, promptFormatForOfferKind(offer.Kind), offer.Key, offerPromptSource(offer))

	if !scanner.Scan() {
		return SkillAnswerNone
	}

	answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
	if answer == "y" || answer == "yes" {
		return SkillAnswerAccept
	}

	return SkillAnswerDecline
}

// promptFormatForOfferKind returns the exact prompt text (design D4's table)
// for kind.
func promptFormatForOfferKind(kind SkillOfferKind) string {
	switch kind {
	case SkillOfferRegister:
		return skillRegisterPromptFormat
	case SkillOfferRefresh:
		return skillRefreshPromptFormat
	case SkillOfferRemove:
		return skillRemovePromptFormat
	default:
		return ""
	}
}

// readSkillDeclines reads the vault's decline state
// (ReadSkillRegistrations): a missing file is an empty state, and any other
// read or schema failure is an error (design D7).
func readSkillDeclines(vault string, deps SkillRegistrationDeps) (map[string]string, error) {
	declined, declinedErr := ReadSkillRegistrations(vault, deps.Accept.Read)
	if declinedErr != nil {
		return nil, fmt.Errorf("register-skills: %w", declinedErr)
	}

	return declined, nil
}

// resolveDefaultSkillSources resolves the default source set
// (ResolveSkillSources) from home and deps.Getwd's working directory. With
// no working directory it fails rather than silently resolving without the
// project sources.
func resolveDefaultSkillSources(
	ctx context.Context, home string, deps SkillRegistrationDeps,
) (ResolvedSkillSources, error) {
	if deps.Getwd == nil {
		return ResolvedSkillSources{}, fmt.Errorf("%w: no working directory", errSkillSourcesNeedWorkingDir)
	}

	cwd, cwdErr := deps.Getwd()
	if cwdErr != nil {
		return ResolvedSkillSources{}, fmt.Errorf("%w: %w", errSkillSourcesNeedWorkingDir, cwdErr)
	}

	sources, resolveErr := ResolveSkillSources(ctx, home, cwd, deps.Sources)
	if resolveErr != nil {
		return ResolvedSkillSources{}, fmt.Errorf("register-skills: resolving skill sources: %w", resolveErr)
	}

	return sources, nil
}

// runSkillAdoptions runs every args --adopt entry, in sorted-by-key order
// for determinism, via AdoptSkillNote — before offers are computed (design
// D6/tasks.md 1.8: "adopt entries ... run first"). Every entry is validated
// before the first one writes (validateAdoptEntry; AdoptSkillNote keeps its
// own checks as defense in depth), and no two entries may name the same
// target note — adopt never silently re-keys a note. Each entry then writes
// to the basename validation resolved, never re-resolving its raw ref. A
// key named by two entries is refused earlier, by parseAdoptFlags.
func runSkillAdoptions(
	ctx context.Context,
	args SkillRegistrationArgs,
	sources ResolvedSkillSources,
	deps SkillAdoptDeps,
	stdout io.Writer,
) error {
	keys := make([]string, 0, len(args.Adopt))
	for key := range args.Adopt {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	adoptSources := make(map[string]SkillNoteSource, len(keys))
	targets := make(map[string]string, len(keys))
	claimedBy := make(map[string]string, len(keys))

	for _, key := range keys {
		source, basename, validateErr := validateAdoptEntry(key, args, sources, deps)
		if validateErr != nil {
			return validateErr
		}

		if other, claimed := claimedBy[basename]; claimed {
			return fmt.Errorf("%w: %s is named by both --adopt %q and --adopt %q",
				errAdoptTargetKeyed, basename, other, key)
		}

		claimedBy[basename] = key
		adoptSources[key] = source
		targets[key] = basename
	}

	// Each entry writes to the basename validation resolved, never
	// re-resolving the raw ref against a vault earlier entries renamed.
	for _, key := range keys {
		adoptErr := AdoptSkillNote(ctx, args.Vault, adoptSources[key], targets[key], deps, stdout)
		if adoptErr != nil {
			return adoptErr
		}
	}

	return nil
}

// skillCandidateRef identifies the candidate an offer came from by its key
// and resolved source path (a NUL cannot occur in either).
func skillCandidateRef(key, path string) string {
	return key + "\x00" + path
}

// skillNoteSourcesByRef builds the note source of every candidate, indexed
// by its key and resolved source path — the pair an offer carries.
func skillNoteSourcesByRef(sources ResolvedSkillSources, home string) map[string]SkillNoteSource {
	byRef := make(map[string]SkillNoteSource, len(sources.Candidates))

	for _, candidate := range sources.Candidates {
		ref := skillCandidateRef(candidate.Key, candidate.SourcePath)
		if _, seen := byRef[ref]; !seen {
			byRef[ref] = NewSkillNoteSource(candidate, home, sources.ResolvedHome)
		}
	}

	return byRef
}

// toStringSet converts values to a set for O(1) membership checks.
func toStringSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, v := range values {
		set[v] = true
	}

	return set
}

// validateAdoptEntry runs, against the vault as it stands before any entry
// writes, the checks AdoptSkillNote runs before its rename, and returns the
// entry's note source and its target's resolved basename: the key names a
// scanned, unconflicted skill (adoptSourceFor); the ref resolves to exactly
// one note, a runbook (resolveAdoptTarget — an ambiguous bare id is
// refused); that note is keyed to no other skill (checkAdoptTargetKey); the
// key is registered to no other note (checkAdoptConflict), so an entry
// cannot rely on another entry re-keying the key's note away; the basename
// carries a Luhmann id and date (skillNoteBasename); and the note renders
// (the pure applySkillNoteBody, result discarded — its full frontmatter
// decode rejects wrong-typed fields the key probe accepts) from exactly the
// content the rename will leave (adoptRenderInput). It does not
// cover I/O failures of the rename, write or embed that follow.
func validateAdoptEntry(
	key string, args SkillRegistrationArgs, sources ResolvedSkillSources, deps SkillAdoptDeps,
) (SkillNoteSource, string, error) {
	source, sourceErr := adoptSourceFor(key, sources, args.Home)
	if sourceErr != nil {
		return SkillNoteSource{}, "", sourceErr
	}

	basename, raw, targetErr := resolveAdoptTarget(args.Vault, args.Adopt[key], deps)
	if targetErr != nil {
		return SkillNoteSource{}, "", targetErr
	}

	keyErr := checkAdoptTargetKey(basename, raw, key)
	if keyErr != nil {
		return SkillNoteSource{}, "", keyErr
	}

	conflictErr := checkAdoptConflict(args.Vault, key, basename, deps)
	if conflictErr != nil {
		return SkillNoteSource{}, "", conflictErr
	}

	newBasename, basenameErr := skillNoteBasename(basename, key)
	if basenameErr != nil {
		return SkillNoteSource{}, "", basenameErr
	}

	_, renderErr := applySkillNoteBody(adoptRenderInput(raw, basename, newBasename), source, false)
	if renderErr != nil {
		return SkillNoteSource{}, "", renderErr
	}

	return source, basename, nil
}
