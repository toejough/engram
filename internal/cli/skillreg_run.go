package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/toejough/engram/internal/vaultgraph"
)

// RegisterSkillsArgs holds the parsed flags for `engram register-skills`
// (skill-runbook-registration). Adopt entries are `<name>=<note-ref>`
// strings, parsed by parseAdoptFlags into SkillRegistrationArgs' map — the
// CLI-facing repeatable-flag shape differs from the internal args' shape,
// same split as LearnFactArgs/LearnFeedbackArgs/LearnRunbookArgs vs. the
// shared internal LearnArgs.
type RegisterSkillsArgs struct {
	Vault     string   `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"`                                         //nolint:lll // unbreakable env+desc struct-tag string
	VaultName string   `targ:"flag,name=vault-name,env=ENGRAM_VAULT_NAME,desc=vault name stamped on a newly-registered note's vault: field (default \"personal\")"` //nolint:lll // unbreakable env+desc struct-tag string
	SkillsDir string   `targ:"flag,name=skills-dir,desc=skills source dir (default: the deployed Claude Code skills dir under home)"`                               //nolint:lll // unbreakable struct-tag string
	DryRun    bool     `targ:"flag,name=dry-run,desc=list offers without prompting or writing"`
	Accept    []string `targ:"flag,name=accept,desc=accept the named skill's current offer without prompting (repeatable)"`
	Decline   []string `targ:"flag,name=decline,desc=decline the named skill's current offer without prompting (repeatable)"`              //nolint:lll // unbreakable struct-tag string
	Adopt     []string `targ:"flag,name=adopt,desc=adopt an existing runbook note as <name>'s skill note: <name>=<note-ref> (repeatable)"` //nolint:lll // unbreakable struct-tag string
}

// SkillRegistrationArgs holds RunSkillRegistration's inputs: the resolved
// vault/vault-name, the skills source dir to scan, the dry-run flag, and any
// explicit --accept/--decline/--adopt answers (skill-runbook-registration).
type SkillRegistrationArgs struct {
	Vault     string
	VaultName string
	SkillsDir string
	DryRun    bool
	Accept    []string
	Decline   []string
	// Adopt maps a skill name to the note-ref its existing runbook note is
	// adopted from (design D6). Parsed from RegisterSkillsArgs.Adopt's
	// "<name>=<note-ref>" strings by the CLI-wiring layer.
	Adopt map[string]string
}

// SkillRegistrationDeps holds RunSkillRegistration's injected capabilities:
// loading shipped skills off the skills source dir, listing/reading vault
// notes for the offer comparison, the interactive-prompt gate, and the three
// accept-path deps structs (Learn for Register, Accept for Refresh/Remove,
// Adopt for --adopt) that skillreg_accept.go's actions already define.
type SkillRegistrationDeps struct {
	// ListSkillsDir lists a skills source dir's immediate entries (production:
	// EdgeFS.ReadDir). Each subdirectory containing SKILL.md becomes a
	// ShippedSkill; an entry without one is ignored.
	ListSkillsDir func(dir string) ([]fs.DirEntry, error)
	// ReadSkillFile reads one skill's SKILL.md bytes (production:
	// EdgeFS.ReadFile).
	ReadSkillFile func(path string) ([]byte, error)
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
// loads the shipped skills off args.SkillsDir, runs any --adopt entries
// first, computes the register/refresh/remove offers against the vault, and
// then — for each offer, in order — acts on an explicit --accept/--decline
// answer, prompts interactively when stdin is a terminal and this isn't a
// dry run, or else leaves it unanswered. A dry run only previews offers and
// touches nothing.
func RunSkillRegistration(
	ctx context.Context, args SkillRegistrationArgs, deps SkillRegistrationDeps, stdout io.Writer,
) error {
	conflictErr := checkAcceptDeclineConflict(args.Accept, args.Decline)
	if conflictErr != nil {
		return conflictErr
	}

	shipped, loadErr := loadShippedSkills(args.SkillsDir, deps.ListSkillsDir, deps.ReadSkillFile)
	if loadErr != nil {
		return loadErr
	}

	shippedByName := skillsByName(shipped)

	if args.DryRun {
		return runSkillRegistrationDryRun(args.Vault, shipped, deps, stdout)
	}

	adoptErr := runSkillAdoptions(ctx, args.Vault, args.Adopt, shippedByName, deps.Adopt, stdout)
	if adoptErr != nil {
		return adoptErr
	}

	return runSkillRegistrationOffers(ctx, args, shipped, shippedByName, deps, stdout)
}

// unexported constants.
const (
	skillMDFilename           = "SKILL.md"
	skillRefreshPromptFormat  = "Skill `%s` changed since its note was last synced. Update the note? [y/N] "
	skillRegisterPromptFormat = "Register skill `%s` as a vault runbook? [y/N] "
	// skillRegistrationAwaitingAnswerFormat is the one-line, non-interactive
	// summary naming every offer left unanswered this run
	// (skill-runbook-registration: "Registration SHALL never prompt or write
	// without a terminal ... it SHALL print one line naming the skills with
	// outstanding offers and the `engram register-skills` command that
	// answers them"). The trailing "<name>" placeholders are literal command
	// syntax, not further substitutions — only the leading %s (the
	// comma-joined list of waiting skill names) is filled in.
	skillRegistrationAwaitingAnswerFormat = "engram: skill runbook offers awaiting an answer: %s — run " +
		"`engram register-skills` in a terminal, or `engram register-skills --accept <name>` / `--decline <name>`\n"
	// skillRegistrationDryRunOfferFormat previews one offer under --dry-run
	// (skill-runbook-registration: "`--dry-run` SHALL list every offer and
	// write nothing").
	skillRegistrationDryRunOfferFormat = "would offer: %s %s\n"
	// skillRegistrationNoOfferFormat reports a --accept/--decline flag naming
	// a skill with no current offer: not an error, just a one-line note, and
	// registration continues (skill-runbook-registration: "Registration SHALL
	// be invocable standalone with explicit answers").
	skillRegistrationNoOfferFormat = "engram: no pending offer for skill %q — ignoring --%s\n"
	skillRemovePromptFormat        = "Skill `%s` is no longer shipped. Remove its runbook note? [y/N] "
)

// unexported variables.
var (
	// errAdoptSkillNotShipped reports an --adopt <name>=<ref> entry whose
	// name matches no shipped skill — AdoptSkillNote needs the skill's
	// current SKILL.md bytes to render the note's body, which only a shipped
	// skill supplies.
	errAdoptSkillNotShipped = errors.New("register-skills: adopt: skill not currently shipped")
	// errMalformedAdoptFlag reports a --adopt flag that isn't shaped
	// "<name>=<note-ref>" (both sides non-empty).
	errMalformedAdoptFlag = errors.New("register-skills: --adopt must be <name>=<note-ref>")
	// errSkillNamedInBothAcceptAndDecline reports a skill name passed to both
	// --accept and --decline in the same invocation (skill-runbook-
	// registration D4/D9: an ambiguous answer is refused before acting on
	// anything).
	errSkillNamedInBothAcceptAndDecline = errors.New(
		"register-skills: skill named in both --accept and --decline")
	// errUnknownSkillOfferKind guards acceptSkillOffer's switch — unreachable
	// in production since CompareSkillOffers only ever emits the three known
	// SkillOfferKind values.
	errUnknownSkillOfferKind = errors.New("register-skills: unknown offer kind")
)

// acceptSkillOffer dispatches an accepted offer to RegisterSkill, RefreshSkill,
// or RemoveSkill, by kind.
func acceptSkillOffer(
	ctx context.Context,
	vault, vaultName string,
	offer SkillOffer,
	shippedByName map[string]ShippedSkill,
	deps SkillRegistrationDeps,
	stdout io.Writer,
) error {
	switch offer.Kind {
	case SkillOfferRegister:
		return RegisterSkill(ctx, vault, vaultName, shippedByName[offer.Skill], deps.Learn, stdout)
	case SkillOfferRefresh:
		return RefreshSkill(ctx, vault, shippedByName[offer.Skill], offer.Basename, deps.Accept, stdout)
	case SkillOfferRemove:
		return RemoveSkill(vault, offer.Basename, deps.Accept, stdout)
	default:
		return fmt.Errorf("%w: %s", errUnknownSkillOfferKind, offer.Kind)
	}
}

// checkAcceptDeclineConflict errors when a skill name appears in both accept
// and decline — an ambiguous answer refused before anything is acted on.
func checkAcceptDeclineConflict(accept, decline []string) error {
	declineSet := toStringSet(decline)

	for _, name := range accept {
		if declineSet[name] {
			return fmt.Errorf("%w: %s", errSkillNamedInBothAcceptAndDecline, name)
		}
	}

	return nil
}

// computeSkillOffers lists the vault, reads the decline state, and runs
// CompareSkillOffers — the offer-computation shared by the dry-run preview
// and the answer loop.
func computeSkillOffers(vault string, shipped []ShippedSkill, deps SkillRegistrationDeps) ([]SkillOffer, error) {
	names, listErr := deps.ListMD(vault)
	if listErr != nil {
		return nil, fmt.Errorf("register-skills: listing vault: %w", listErr)
	}

	declined, declinedErr := ReadSkillRegistrations(vault, deps.Accept.Read)
	if declinedErr != nil {
		return nil, fmt.Errorf("register-skills: %w", declinedErr)
	}

	offers, offersErr := CompareSkillOffers(vault, shipped, names, deps.Accept.Read, declined)
	if offersErr != nil {
		return nil, fmt.Errorf("register-skills: %w", offersErr)
	}

	return offers, nil
}

// loadShippedSkills lists skillsDir's immediate subdirectories and reads
// each one's SKILL.md into a ShippedSkill, sorted by name for deterministic
// output. A subdirectory without SKILL.md is ignored (skill-runbook-
// registration: registration never requires any file beyond SKILL.md
// itself); any other read failure is propagated.
func loadShippedSkills(
	skillsDir string,
	readDir func(string) ([]fs.DirEntry, error),
	readFile func(string) ([]byte, error),
) ([]ShippedSkill, error) {
	entries, dirErr := readDir(skillsDir)
	if dirErr != nil {
		return nil, fmt.Errorf("register-skills: listing skills dir %s: %w", skillsDir, dirErr)
	}

	skills := make([]ShippedSkill, 0, len(entries))

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillMDPath := filepath.Join(skillsDir, entry.Name(), skillMDFilename)

		content, readErr := readFile(skillMDPath)
		if readErr != nil {
			if errors.Is(readErr, fs.ErrNotExist) {
				continue
			}

			return nil, fmt.Errorf("register-skills: reading %s: %w", skillMDPath, readErr)
		}

		skills = append(skills, ShippedSkill{Name: entry.Name(), Content: content})
	}

	sort.Slice(skills, func(i, j int) bool { return skills[i].Name < skills[j].Name })

	return skills, nil
}

// newSkillRegistrationDeps composes SkillRegistrationDeps from the injected
// edge Deps (pure composition — no direct I/O; #700).
func newSkillRegistrationDeps(d Deps) SkillRegistrationDeps {
	vfs := newVaultFS(d.FS)

	return SkillRegistrationDeps{
		ListSkillsDir: d.FS.ReadDir,
		ReadSkillFile: d.FS.ReadFile,
		ListMD:        vfs.ListMD,
		IsTerminal:    d.IsTerminal,
		Stdin:         d.Stdin,
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
// "<name>=<note-ref>" strings into SkillRegistrationArgs.Adopt's map.
func parseAdoptFlags(raw []string) (map[string]string, error) {
	adopt := make(map[string]string, len(raw))

	for _, entry := range raw {
		name, ref, found := strings.Cut(entry, "=")
		if !found || name == "" || ref == "" {
			return nil, fmt.Errorf("%w: %q", errMalformedAdoptFlag, entry)
		}

		adopt[name] = ref
	}

	return adopt, nil
}

// promptForOffer writes offer's prompt to stdout and reads one line from
// scanner: y/yes (case-insensitive) accepts; anything else, including EOF
// (scanner.Scan returning false), declines (skill-runbook-registration).
func promptForOffer(offer SkillOffer, scanner *bufio.Scanner, stdout io.Writer) bool {
	_, _ = fmt.Fprintf(stdout, promptFormatForOfferKind(offer.Kind), offer.Skill)

	if !scanner.Scan() {
		return false
	}

	answer := strings.ToLower(strings.TrimSpace(scanner.Text()))

	return answer == "y" || answer == "yes"
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

// reportUnmatchedExplicitAnswers prints skillRegistrationNoOfferFormat for
// every name in names that isn't among offered — a --accept/--decline flag
// naming a skill with no current offer (not an error).
func reportUnmatchedExplicitAnswers(stdout io.Writer, names []string, flag string, offered map[string]bool) {
	for _, name := range names {
		if offered[name] {
			continue
		}

		_, _ = fmt.Fprintf(stdout, skillRegistrationNoOfferFormat, name, flag)
	}
}

// resolveOneSkillOffer answers a single offer: an explicit --accept/--decline
// wins over interactivity; otherwise it prompts when interactive, else
// appends to unanswered. Split out of runSkillRegistrationOffers's loop to
// keep both within the repo's cyclomatic-complexity budget.
func resolveOneSkillOffer(
	ctx context.Context,
	args SkillRegistrationArgs,
	offer SkillOffer,
	shippedByName map[string]ShippedSkill,
	acceptSet, declineSet map[string]bool,
	interactive bool,
	scanner *bufio.Scanner,
	deps SkillRegistrationDeps,
	stdout io.Writer,
	unanswered *[]string,
) error {
	switch {
	case acceptSet[offer.Skill]:
		return acceptSkillOffer(ctx, args.Vault, args.VaultName, offer, shippedByName, deps, stdout)
	case declineSet[offer.Skill]:
		return RecordSkillDeclined(args.Vault, offer.Skill, offer.Hash, deps.Accept.Read, deps.Accept.Write)
	case interactive:
		if promptForOffer(offer, scanner, stdout) {
			return acceptSkillOffer(ctx, args.Vault, args.VaultName, offer, shippedByName, deps, stdout)
		}

		return RecordSkillDeclined(args.Vault, offer.Skill, offer.Hash, deps.Accept.Read, deps.Accept.Write)
	default:
		*unanswered = append(*unanswered, offer.Skill)

		return nil
	}
}

// runSkillAdoptions runs every args --adopt entry, in sorted-by-name order
// for determinism, via AdoptSkillNote — before offers are computed (design
// D6/tasks.md 1.8: "adopt entries ... run first").
func runSkillAdoptions(
	ctx context.Context,
	vault string,
	adopt map[string]string,
	shippedByName map[string]ShippedSkill,
	deps SkillAdoptDeps,
	stdout io.Writer,
) error {
	names := make([]string, 0, len(adopt))
	for name := range adopt {
		names = append(names, name)
	}

	sort.Strings(names)

	for _, name := range names {
		skill, found := shippedByName[name]
		if !found {
			return fmt.Errorf("%w: %q", errAdoptSkillNotShipped, name)
		}

		adoptErr := AdoptSkillNote(ctx, vault, skill, adopt[name], deps, stdout)
		if adoptErr != nil {
			return adoptErr
		}
	}

	return nil
}

// runSkillRegistrationDryRun previews every computed offer as "would offer:
// <kind> <skill>" and writes nothing, prompts nothing (skill-runbook-
// registration: "`--dry-run` SHALL list every offer and write nothing").
// --adopt entries are not run under --dry-run either — a dry run touches no
// vault state.
func runSkillRegistrationDryRun(
	vault string, shipped []ShippedSkill, deps SkillRegistrationDeps, stdout io.Writer,
) error {
	offers, offersErr := computeSkillOffers(vault, shipped, deps)
	if offersErr != nil {
		return offersErr
	}

	for _, offer := range offers {
		_, _ = fmt.Fprintf(stdout, skillRegistrationDryRunOfferFormat, offer.Kind, offer.Skill)
	}

	return nil
}

// runSkillRegistrationOffers computes the current offers and, for each one:
// acts immediately on an explicit --accept/--decline answer; else prompts
// when interactive; else leaves it unanswered. Explicit answers naming a
// skill with no current offer are reported (not an error), and any offers
// left unanswered are named in one final summary line.
func runSkillRegistrationOffers(
	ctx context.Context,
	args SkillRegistrationArgs,
	shipped []ShippedSkill,
	shippedByName map[string]ShippedSkill,
	deps SkillRegistrationDeps,
	stdout io.Writer,
) error {
	offers, offersErr := computeSkillOffers(args.Vault, shipped, deps)
	if offersErr != nil {
		return offersErr
	}

	acceptSet := toStringSet(args.Accept)
	declineSet := toStringSet(args.Decline)
	interactive := deps.IsTerminal != nil && deps.IsTerminal()

	var scanner *bufio.Scanner
	if interactive {
		scanner = bufio.NewScanner(deps.Stdin)
	}

	offeredNames := make(map[string]bool, len(offers))
	unanswered := make([]string, 0, len(offers))

	for _, offer := range offers {
		offeredNames[offer.Skill] = true

		actionErr := resolveOneSkillOffer(ctx, args, offer, shippedByName, acceptSet, declineSet,
			interactive, scanner, deps, stdout, &unanswered)
		if actionErr != nil {
			return actionErr
		}
	}

	reportUnmatchedExplicitAnswers(stdout, args.Accept, "accept", offeredNames)
	reportUnmatchedExplicitAnswers(stdout, args.Decline, "decline", offeredNames)

	if len(unanswered) > 0 {
		_, _ = fmt.Fprintf(stdout, skillRegistrationAwaitingAnswerFormat, strings.Join(unanswered, ", "))
	}

	return nil
}

// skillsByName indexes skills by name for accept-time lookup.
func skillsByName(skills []ShippedSkill) map[string]ShippedSkill {
	byName := make(map[string]ShippedSkill, len(skills))
	for _, skill := range skills {
		byName[skill.Name] = skill
	}

	return byName
}

// toStringSet converts values to a set for O(1) membership checks.
func toStringSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, v := range values {
		set[v] = true
	}

	return set
}
