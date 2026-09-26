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
	SkillsDir string   `targ:"flag,name=skills-dir,desc=skills source dir (default: the engram-owned deployed skills dir ~/.claude/engram/skills)"`                 //nolint:lll // unbreakable struct-tag string
	DryRun    bool     `targ:"flag,name=dry-run,desc=list offers without prompting or writing"`
	Accept    []string `targ:"flag,name=accept,desc=accept the matching offers without prompting: a skill key or prefix* pattern or @<scope-id> (repeatable)"`   //nolint:lll // unbreakable struct-tag string
	Decline   []string `targ:"flag,name=decline,desc=decline the matching offers without prompting: a skill key or prefix* pattern or @<scope-id> (repeatable)"` //nolint:lll // unbreakable struct-tag string
	Adopt     []string `targ:"flag,name=adopt,desc=adopt an existing runbook note as <name>'s skill note: <name>=<note-ref> (repeatable)"`                       //nolint:lll // unbreakable struct-tag string
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
	answers, answersErr := ParseSkillAnswers(args.Accept, args.Decline)
	if answersErr != nil {
		return answersErr
	}

	shipped, loadErr := loadShippedSkills(args.SkillsDir, deps.ListSkillsDir, deps.ReadSkillFile)
	if loadErr != nil {
		return loadErr
	}

	shippedByName := skillsByName(shipped)

	if args.DryRun {
		return runSkillRegistrationDryRun(args.Vault, args.SkillsDir, shipped, deps, stdout)
	}

	adoptErr := runSkillAdoptions(ctx, args.Vault, args.Adopt, shippedByName, deps.Adopt, stdout)
	if adoptErr != nil {
		return adoptErr
	}

	return runSkillRegistrationOffers(ctx, args, answers, shipped, shippedByName, deps, stdout)
}

// unexported constants.
const (
	skillMDFilename           = "SKILL.md"
	skillRefreshPromptFormat  = "Skill `%s` changed since its note was last synced. Update the note? [y/N] "
	skillRegisterPromptFormat = "Register skill `%s` as a vault runbook? [y/N] "
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
		return RegisterSkill(ctx, vault, vaultName, shippedByName[offer.Key], deps.Learn, stdout)
	case SkillOfferRefresh:
		return RefreshSkill(ctx, vault, shippedByName[offer.Key], offer.Basename, deps.Accept, stdout)
	case SkillOfferRemove:
		return RemoveSkill(vault, offer.Basename, deps.Accept, stdout)
	default:
		return fmt.Errorf("%w: %s", errUnknownSkillOfferKind, offer.Kind)
	}
}

// computeSkillOffers lists the vault, reads the decline state, and runs
// CompareSkillOffers over the shipped skills of skillsDir — the
// offer-computation shared by the dry-run preview and the answer loop. The
// shipped skills are Claude-user candidates keyed by their bare names, and
// skillsDir is their one read root, so every bare-key note is
// removal-eligible, as before keyed sources existed.
func computeSkillOffers(
	vault, skillsDir string, shipped []ShippedSkill, deps SkillRegistrationDeps,
) (SkillOfferComparison, error) {
	names, listErr := deps.ListMD(vault)
	if listErr != nil {
		return SkillOfferComparison{}, fmt.Errorf("register-skills: listing vault: %w", listErr)
	}

	declined, declinedErr := ReadSkillRegistrations(vault, deps.Accept.Read)
	if declinedErr != nil {
		return SkillOfferComparison{}, fmt.Errorf("register-skills: %w", declinedErr)
	}

	comparison, offersErr := CompareSkillOffers(SkillOfferInput{
		Vault:    vault,
		Names:    names,
		ReadFile: deps.Accept.Read,
		Declined: declined,
		Sources:  ShippedSkillSources(skillsDir, shipped),
	})
	if offersErr != nil {
		return SkillOfferComparison{}, fmt.Errorf("register-skills: %w", offersErr)
	}

	return comparison, nil
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
	_, _ = fmt.Fprintf(stdout, promptFormatForOfferKind(offer.Kind), offer.Key)

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

// runSkillRegistrationDryRun previews every computed offer under its scope
// header (PreviewSkillOffers) and writes nothing, prompts nothing
// (skill-runbook-registration: "`--dry-run` SHALL list every offer and write
// nothing").
// --adopt entries are not run under --dry-run either — a dry run touches no
// vault state.
func runSkillRegistrationDryRun(
	vault, skillsDir string, shipped []ShippedSkill, deps SkillRegistrationDeps, stdout io.Writer,
) error {
	comparison, offersErr := computeSkillOffers(vault, skillsDir, shipped, deps)
	if offersErr != nil {
		return offersErr
	}

	return PreviewSkillOffers(stdout, comparison)
}

// runSkillRegistrationOffers computes the current offers and answers them
// through AnswerSkillOffers: an accepted offer is carried out by
// acceptSkillOffer, a declined one records its hash (RecordSkillDeclined).
func runSkillRegistrationOffers(
	ctx context.Context,
	args SkillRegistrationArgs,
	answers SkillAnswers,
	shipped []ShippedSkill,
	shippedByName map[string]ShippedSkill,
	deps SkillRegistrationDeps,
	stdout io.Writer,
) error {
	comparison, offersErr := computeSkillOffers(args.Vault, args.SkillsDir, shipped, deps)
	if offersErr != nil {
		return offersErr
	}

	return AnswerSkillOffers(SkillOfferAnswering{
		Comparison:  comparison,
		Answers:     answers,
		Interactive: deps.IsTerminal != nil && deps.IsTerminal(),
		Stdin:       deps.Stdin,
		Accept: func(offer SkillOffer) error {
			return acceptSkillOffer(ctx, args.Vault, args.VaultName, offer, shippedByName, deps, stdout)
		},
		Decline: func(offer SkillOffer) error {
			return RecordSkillDeclined(args.Vault, offer.Key, offer.Hash, deps.Accept.Read, deps.Accept.Write)
		},
	}, stdout)
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
