package cli

import (
	"errors"
	"fmt"
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

// unexported constants.
const (
	answeredByBodyMarker   = embed.AnsweredByBodyMarker
	answersBodyMarker      = embed.AnswersBodyMarker
	contributorsBodyMarker = embed.ContributorsBodyMarker
	qaAnswerSuffix         = ".a.md"
	qaNotePrefix           = "qa."
	qaQuestionSuffix       = ".q.md"
	typeQAAnswer           = "qa-answer"
	typeQAQuestion         = "qa-question"
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
// query pipeline's main matched set. Excludes vocab kinds AND qa-question.
// qa-answer COMPETES in the main set (D5′ — A notes are synthesis notes).
func isQueryExcludedKind(content string) bool {
	return isVocabKind(content) || isQAQuestionKind(content)
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
// vaultMDNames is the list of full .md filenames (from ListMDFilenames).
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
		certainty = "medium"
	}

	switch certainty {
	case "high", "medium", "low":
		return nil
	default:
		return fmt.Errorf("%w: got %q", errQACertaintyInvalid, certainty)
	}
}
