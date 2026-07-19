package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/toejough/engram/internal/embed"
)

// LearnQAArgs holds parsed flags for the engram learn qa subcommand.
type LearnQAArgs struct {
	Vault        string   `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"` //nolint:lll
	Slug         string   `targ:"flag,name=slug,required,desc=kebab-case tag shared by the Q and A filenames (required)"`
	Question     string   `targ:"flag,name=question,desc=verbatim question text"`
	Answer       string   `targ:"flag,name=answer,desc=inline answer body (mutually exclusive with --answer-file)"`
	AnswerFile   string   `targ:"flag,name=answer-file,desc=path to a file whose content is the answer body"`
	Contributors []string `targ:"flag,name=contributors,desc=full note basenames (no .md) that contributed (repeatable; validated against vault)"` //nolint:lll
	Certainty    string   `targ:"flag,name=certainty,desc=high|medium|low (default medium)"`
	Source       string   `targ:"flag,name=source,required,desc=provenance string for the source field (required)"`
}

// LearnQADeps is the dependency set for RunLearnQA.
type LearnQADeps struct {
	Now        func() time.Time
	Getenv     func(string) string
	StatDir    func(string) error
	InitVault  func(string) error
	ListMD     func(string) ([]string, error)
	Lock       func(vault string) (release func(), err error)
	WriteNew   func(path string, data []byte) error
	RemoveFile func(path string) error
	ReadFile   func(path string) ([]byte, error)
	// Embed-on-write pipeline (optional; nil skips silently).
	Embedder     embed.Embedder
	WriteSidecar func(path string, data []byte) error
	LogWarning   func(format string, args ...any)
	// Vocab assignment pipeline (optional; all three must be non-nil to activate).
	LoadTermVectors func(vault string) ([]TermWithVector, error)
	ReadSidecar     func(path string) ([]byte, error)
	WriteNote       func(path string, data []byte) error
}

// RunLearnQA implements the engram learn qa subcommand.
// Writes Q then A atomically-ish: on A-write failure, removes Q (best-effort)
// and returns a descriptive error. The function calls the existing embed and
// vocab assignment pipeline for both notes, in order.
func RunLearnQA(ctx context.Context, args LearnQAArgs, deps LearnQADeps, stdout io.Writer) error {
	validateErr := validateLearnQAArgs(args)
	if validateErr != nil {
		return validateErr
	}

	certainty := args.Certainty
	if certainty == "" {
		certainty = certMedium
	}

	vault := args.Vault

	ensureErr := ensureQAVault(deps, vault)
	if ensureErr != nil {
		return ensureErr
	}

	// Resolve answer body.
	answerBody := args.Answer
	if args.AnswerFile != "" {
		raw, readErr := deps.ReadFile(args.AnswerFile)
		if readErr != nil {
			return fmt.Errorf("learn qa: reading --answer-file %q: %w", args.AnswerFile, readErr)
		}

		answerBody = string(raw)
	}

	// Validate contributors before acquiring the lock.
	mdNames, listErr := deps.ListMD(vault)
	if listErr != nil {
		return fmt.Errorf("learn qa: listing vault: %w", listErr)
	}

	validateContributorsErr := validateContributors(args.Contributors, mdNames)
	if validateContributorsErr != nil {
		return validateContributorsErr
	}

	when := deps.Now()
	qPath := qaQuestionPath(vault, args.Slug, when)
	aPath := qaAnswerPath(vault, args.Slug, when)

	qContent := renderQAQuestionNote(args.Question, args.Slug, args.Source, when)
	aContent := renderQAAnswerNote(answerBody, args.Slug, args.Source, certainty,
		args.Contributors, when)

	// Write both notes under the lock, then RELEASE before embed/vocab — the
	// learn.go writeLearnUnderLock pattern (note 50: sidecar writes are atomic
	// per-file and need no vault lock; holding it through embedding would block
	// concurrent learns).
	writeErr := writeQANotesUnderLock(deps, vault, qPath, aPath, qContent, aContent)
	if writeErr != nil {
		return writeErr
	}

	// Embed-on-write and vocab assignment for Q note (embed only; no vocab on Q).
	autoEmbedNote(ctx, asLearnDepsForEmbed(deps), qPath, qContent)
	// No vocab assignment for Q notes (D5′: Q notes carry no vocab).

	// Embed-on-write and vocab assignment for A note.
	autoEmbedNote(ctx, asLearnDepsForEmbed(deps), aPath, aContent)
	applyVocabAssignmentAfterLearn(asLearnDepsForVocab(deps), vault, aPath, aContent)

	_, _ = fmt.Fprintln(stdout, qPath)
	_, _ = fmt.Fprintln(stdout, aPath)

	return nil
}

// unexported constants.
const (
	answeredByBodyMarker   = embed.AnsweredByBodyMarker
	answersBodyMarker      = embed.AnswersBodyMarker
	certMedium             = "medium"
	contributorsBodyMarker = embed.ContributorsBodyMarker
	qaAnswerSuffix         = ".a.md"
	qaNotePrefix           = "qa."
	qaQuestionSuffix       = ".q.md"
	// qaRound2MinPairs is the QA-pair count at which the round-2 gate reads READY.
	qaRound2MinPairs = 20
	typeQAAnswer     = "qa-answer"
	typeQAQuestion   = "qa-question"
)

// unexported variables.
var (
	errQAAnswerSourceRequired = errors.New("learn qa: exactly one of --answer or --answer-file is required")
	errQACertaintyInvalid     = errors.New("learn qa: --certainty must be high, medium, or low")
	errQAContributorNotFound  = errors.New("learn qa: contributor not found in vault")
	errQAQuestionRequired     = errors.New("learn qa: --question is required")
	errQASourceRequired       = errors.New("learn qa: --source is required")
)

// qaAnswerFrontmatterDoc is the YAML shape of a QA answer note's frontmatter.
type qaAnswerFrontmatterDoc struct {
	Type         string   `yaml:"type"`
	Date         string   `yaml:"date"`
	Answers      string   `yaml:"answers"`
	Certainty    string   `yaml:"certainty"`
	Contributors []string `yaml:"contributors,omitempty"`
	Source       string   `yaml:"source"`
}

// qaQuestionFrontmatterDoc is the YAML shape of a QA question note's frontmatter.
type qaQuestionFrontmatterDoc struct {
	Type       string `yaml:"type"`
	Date       string `yaml:"date"`
	AnsweredBy string `yaml:"answered_by"`
	Source     string `yaml:"source"`
}

// asLearnDepsForEmbed builds the minimal LearnDeps needed by autoEmbedNote.
func asLearnDepsForEmbed(d LearnQADeps) LearnDeps {
	return LearnDeps{
		Embedder:     d.Embedder,
		WriteSidecar: d.WriteSidecar,
		LogWarning:   d.LogWarning,
	}
}

// asLearnDepsForVocab builds the minimal LearnDeps needed by applyVocabAssignmentAfterLearn.
func asLearnDepsForVocab(d LearnQADeps) LearnDeps {
	return LearnDeps{
		Now:             d.Now,
		LoadTermVectors: d.LoadTermVectors,
		ReadSidecar:     d.ReadSidecar,
		WriteNote:       d.WriteNote,
		LogWarning:      d.LogWarning,
		ListMD:          d.ListMD,
	}
}

// countQAPairs counts vault files where both the .q.md and matching .a.md exist.
// Pure read-time scan; no new state.
func countQAPairs(names []string) int {
	nameSet := make(map[string]struct{}, len(names))
	for _, name := range names {
		nameSet[name] = struct{}{}
	}

	count := 0

	for _, name := range names {
		if !isQAQuestionFilename(name) {
			continue
		}
		// Derive expected A filename: replace .q.md suffix with .a.md.
		base := strings.TrimSuffix(name, qaQuestionSuffix)
		aName := base + qaAnswerSuffix

		if _, ok := nameSet[aName]; ok {
			count++
		}
	}

	return count
}

// ensureQAVault checks that vault exists, creating it if missing.
// Returns an error for non-ErrNotExist stat failures or init failures.
func ensureQAVault(deps LearnQADeps, vault string) error {
	return ensureVaultDir(deps.StatDir, deps.InitVault, vault, "learn qa")
}

// ensureVaultDir stats the vault dir and initializes it when absent — the
// shared init path for every note-writing subcommand (learn, learn qa, ...).
func ensureVaultDir(statDir, initVault func(string) error, vault, prefix string) error {
	dirErr := statDir(vault)
	if dirErr == nil {
		return nil
	}

	if !errors.Is(dirErr, fs.ErrNotExist) {
		return fmt.Errorf("%s: vault %s: %w", prefix, vault, dirErr)
	}

	initErr := initVault(vault)
	if initErr != nil {
		return fmt.Errorf("%s: %w", prefix, initErr)
	}

	return nil
}

// isQAQuestionFilename reports whether a filename is a QA question note
// (prefix "qa." AND suffix ".q.md").
func isQAQuestionFilename(name string) bool {
	return strings.HasPrefix(name, qaNotePrefix) && strings.HasSuffix(name, qaQuestionSuffix)
}

// isQAQuestionKind reports whether the note's frontmatter type is qa-question.
func isQAQuestionKind(content string) bool {
	return kindFromContent(content) == typeQAQuestion
}

// isQueryExcludedKind reports whether a note should be excluded from the
// query pipeline's main matched set. Excludes qa-question only — vocab
// definition and member notes are ordinary recallable notes (#678 Task 6
// retired the vocab type-based exclusion). qa-answer COMPETES in the main
// set (D5′ — A notes are synthesis notes).
func isQueryExcludedKind(content string) bool {
	return isQAQuestionKind(content)
}

// newQaDeps composes LearnQADeps purely from the injected edge capabilities.
func newQaDeps(d Deps) LearnQADeps {
	return LearnQADeps{
		Now:          d.Now,
		Getenv:       d.Getenv,
		StatDir:      statDirFromFS(d.FS),
		InitVault:    initVaultFromFS(d.FS),
		ListMD:       listMDFromFS(d.FS),
		Lock:         vaultLockFromLocker(d.Lock),
		WriteNew:     writeNewFromFS(d.FS),
		RemoveFile:   d.FS.Remove,
		ReadFile:     d.FS.ReadFile,
		Embedder:     d.Embed,
		WriteSidecar: writeAtomicFromFS(d.FS, sidecarPerm, "write sidecar"),
		LogWarning:   logWarningTo(d.Stderr),
		LoadTermVectors: func(vault string) ([]TermWithVector, error) {
			return loadAssignmentTermVectors(vault, listMDFromFS(d.FS), d.FS.ReadFile)
		},
		ReadSidecar: d.FS.ReadFile,
		WriteNote:   writeAtomicFromFS(d.FS, vocabNotePerm, "write note"),
	}
}

// qaAnswerPath returns the full filesystem path for a QA answer note.
func qaAnswerPath(vault, slug string, when time.Time) string {
	return filepath.Join(vault, qaSlug(slug, when)+qaAnswerSuffix)
}

// qaQuestionPath returns the full filesystem path for a QA question note.
func qaQuestionPath(vault, slug string, when time.Time) string {
	return filepath.Join(vault, qaSlug(slug, when)+qaQuestionSuffix)
}

// qaSlug returns the shared date+slug prefix: "qa.<YYYY-MM-DD>.<slug>".
func qaSlug(slug string, when time.Time) string {
	return qaNotePrefix + when.Format(dateFormat) + "." + slug
}

// renderQAAnswerNote assembles the full content of a QA answer note.
// The machine `Answers:` and `Contributors:` lines are appended.
func renderQAAnswerNote(answerBody, slug, source, certainty string,
	contributors []string, when time.Time,
) string {
	sharedSlug := qaSlug(slug, when)
	// Full basename of the paired question note (no .md).
	qBasename := sharedSlug + ".q"

	frontmatter := marshalFrontmatter(qaAnswerFrontmatterDoc{
		Type:         typeQAAnswer,
		Date:         when.Format(dateFormat),
		Answers:      qBasename,
		Certainty:    certainty,
		Contributors: contributors,
		Source:       source,
	})

	body := strings.TrimRight(answerBody, "\n") + "\n"
	body += "\n" + answersBodyMarker + " [[" + qBasename + "]]\n"

	if len(contributors) > 0 {
		parts := make([]string, len(contributors))
		for i, contributor := range contributors {
			parts[i] = "[[" + contributor + "]]"
		}

		body += contributorsBodyMarker + " " + strings.Join(parts, ", ") + "\n"
	}

	return frontmatter + "\n" + body
}

// renderQAQuestionNote assembles the full content of a QA question note.
// The machine `Answered by:` line is appended after the question body.
func renderQAQuestionNote(questionText, slug, source string, when time.Time) string {
	sharedSlug := qaSlug(slug, when)
	// Full basename of the paired answer note (no .md).
	aBasename := sharedSlug + ".a"

	frontmatter := marshalFrontmatter(qaQuestionFrontmatterDoc{
		Type:       typeQAQuestion,
		Date:       when.Format(dateFormat),
		AnsweredBy: aBasename,
		Source:     source,
	})

	body := strings.TrimRight(questionText, "\n") + "\n"
	body += "\n" + answeredByBodyMarker + " [[" + aBasename + "]]\n"

	return frontmatter + "\n" + body
}

// validateContributors checks that each full basename exists in the vault.
// vaultMDNames is the list of full .md filenames (from ListMD).
func validateContributors(contributors []string, vaultMDNames []string) error {
	nameSet := make(map[string]struct{}, len(vaultMDNames))
	for _, name := range vaultMDNames {
		nameSet[strings.TrimSuffix(name, ".md")] = struct{}{}
	}

	for _, contributor := range contributors {
		if _, ok := nameSet[contributor]; !ok {
			return fmt.Errorf("%w: %q", errQAContributorNotFound, contributor)
		}
	}

	return nil
}

// validateLearnQAArgs validates all LearnQAArgs fields before any I/O.
func validateLearnQAArgs(args LearnQAArgs) error {
	slugErr := validateSlug(args.Slug)
	if slugErr != nil {
		return fmt.Errorf("learn qa: %w", slugErr)
	}

	if strings.TrimSpace(args.Question) == "" {
		return errQAQuestionRequired
	}

	answerSourceErr := validateQAAnswerSource(args.Answer, args.AnswerFile)
	if answerSourceErr != nil {
		return answerSourceErr
	}

	if args.Source == "" {
		return errQASourceRequired
	}

	return validateQACertainty(args.Certainty)
}

// validateQAAnswerSource validates that exactly one of answer or answerFile is set.
func validateQAAnswerSource(answer, answerFile string) error {
	bothEmpty := answer == "" && answerFile == ""
	bothSet := answer != "" && answerFile != ""

	if bothEmpty || bothSet {
		return errQAAnswerSourceRequired
	}

	return nil
}

// validateQACertainty validates the certainty field (defaulting to "medium").
func validateQACertainty(certainty string) error {
	if certainty == "" {
		certainty = certMedium
	}

	switch certainty {
	case "high", certMedium, "low":
		return nil
	default:
		return fmt.Errorf("%w: got %q", errQACertaintyInvalid, certainty)
	}
}

// writeQANotesUnderLock writes the Q then A note under the vault lock and
// releases it on return. The lock prevents partial-write races with concurrent
// learn operations; QA notes use date+slug names, not Luhmann IDs.
// On A-write failure the Q note is removed (best-effort) so no orphan remains.
func writeQANotesUnderLock(deps LearnQADeps, vault, qPath, aPath, qContent, aContent string) error {
	release, lockErr := deps.Lock(vault)
	if lockErr != nil {
		return fmt.Errorf("learn qa: acquiring lock: %w", lockErr)
	}

	defer release()

	qWriteErr := deps.WriteNew(qPath, []byte(qContent))
	if qWriteErr != nil {
		return fmt.Errorf("learn qa: writing Q note: %w", qWriteErr)
	}

	aWriteErr := deps.WriteNew(aPath, []byte(aContent))
	if aWriteErr != nil {
		removeErr := deps.RemoveFile(qPath)
		if removeErr != nil {
			return fmt.Errorf("learn qa: writing A note: %w (also failed to remove orphan Q note %q: %s)",
				aWriteErr, qPath, removeErr.Error())
		}

		return fmt.Errorf("learn qa: writing A note (Q note removed): %w", aWriteErr)
	}

	return nil
}
