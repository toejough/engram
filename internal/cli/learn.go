package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// LearnArgs holds the parsed flags for the learn subcommand.
type LearnArgs struct {
	Type     string
	Slug     string
	Vault    string
	Target   string
	Relation string
	Source   string

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
	// moc only
	Topic string
}

// LearnDeps holds injected dependencies for runLearn. All fields required.
type LearnDeps struct {
	Now      func() time.Time
	Stdin    io.Reader
	Getenv   func(string) string
	StatDir  func(string) error
	ListIDs  func(vault string) ([]string, error)
	Lock     func(vault string) (release func(), err error)
	WriteNew func(path string, data []byte) error
}

// unexported constants.
const (
	mocSubdir       = "MOCs"
	permanentSubdir = "Permanent"
	typeMOC         = "moc"
)

// unexported variables.
var (
	errLearnUnknownType    = errors.New("learn: type must be feedback, fact, or moc")
	luhmannFilenamePattern = regexp.MustCompile(
		`^([0-9][0-9a-z]*)\.\d{4}-\d{2}-\d{2}\..+\.md$`,
	)
)

type factFields struct {
	Situation string
	Subject   string
	Predicate string
	Object    string
	Luhmann   string
	Source    string
}

type feedbackFields struct {
	Situation string
	Behavior  string
	Impact    string
	Action    string
	Luhmann   string
	Source    string
}

type mocFields struct {
	Topic   string
	Luhmann string
	Source  string
}

func assembleLearnContent(args LearnArgs, luhmann string, when time.Time, body string) (string, error) {
	switch args.Type {
	case typeFeedback:
		f := feedbackFields{
			Situation: args.Situation, Behavior: args.Behavior, Impact: args.Impact,
			Action: args.Action, Luhmann: luhmann, Source: args.Source,
		}

		return renderFeedbackFrontmatter(f, when) + renderFeedbackBody(f, body), nil
	case typeFact:
		f := factFields{
			Situation: args.Situation, Subject: args.Subject, Predicate: args.Predicate,
			Object: args.Object, Luhmann: luhmann, Source: args.Source,
		}

		return renderFactFrontmatter(f, when) + renderFactBody(f, body), nil
	case typeMOC:
		f := mocFields{Topic: args.Topic, Luhmann: luhmann, Source: args.Source}

		return renderMOCFrontmatter(f, when) + renderMOCBody(body), nil
	default:
		return "", fmt.Errorf("%w: got %q", errLearnUnknownType, args.Type)
	}
}

func extractLuhmannFromFilename(name string) (string, bool) {
	m := luhmannFilenamePattern.FindStringSubmatch(name)
	if m == nil {
		return "", false
	}

	return m[1], true
}

func learnPath(vault, memType, luhmann, slug string, when time.Time) string {
	subdir := permanentSubdir
	if memType == typeMOC {
		subdir = mocSubdir
	}

	filename := fmt.Sprintf("%s.%s.%s.md", luhmann, when.Format(dateFormat), slug)

	return filepath.Join(vault, subdir, filename)
}

func newOsLearnDeps() LearnDeps {
	fs := &osLearnFS{}

	return LearnDeps{
		Now:      time.Now,
		Stdin:    os.Stdin,
		Getenv:   os.Getenv,
		StatDir:  fs.StatDir,
		ListIDs:  fs.ListIDs,
		Lock:     fs.Lock,
		WriteNew: fs.WriteNew,
	}
}

func renderFactBody(f factFields, relatedSection string) string {
	formula := fmt.Sprintf(
		"Information learned: when in %s, %s %s %s.\n",
		f.Situation, f.Subject, f.Predicate, f.Object,
	)

	return formula + "\n" + relatedSection
}

func renderFactFrontmatter(f factFields, when time.Time) string {
	return strings.Join([]string{
		"---",
		"type: fact",
		"situation: " + f.Situation,
		"subject: " + f.Subject,
		"predicate: " + f.Predicate,
		"object: " + f.Object,
		fmt.Sprintf("luhmann: %q", f.Luhmann),
		"created: " + when.Format(dateFormat),
		"source: " + f.Source,
		"---",
		"",
	}, "\n")
}

func renderFeedbackBody(f feedbackFields, relatedSection string) string {
	formula := fmt.Sprintf("Lesson learned: when %s, %s.\n", f.Situation, f.Action)

	return formula + "\n" + relatedSection
}

func renderFeedbackFrontmatter(f feedbackFields, when time.Time) string {
	return strings.Join([]string{
		"---",
		"type: feedback",
		"situation: " + f.Situation,
		"behavior: " + f.Behavior,
		"impact: " + f.Impact,
		"action: " + f.Action,
		fmt.Sprintf("luhmann: %q", f.Luhmann),
		"created: " + when.Format(dateFormat),
		"source: " + f.Source,
		"---",
		"",
	}, "\n")
}

func renderMOCBody(framing string) string {
	return framing
}

func renderMOCFrontmatter(f mocFields, when time.Time) string {
	return strings.Join([]string{
		"---",
		"type: moc",
		"topic: " + f.Topic,
		fmt.Sprintf("luhmann: %q", f.Luhmann),
		"created: " + when.Format(dateFormat),
		"source: " + f.Source,
		"---",
		"",
	}, "\n")
}

// runLearn orchestrates the learn subcommand: validates inputs, acquires the lock,
// computes the next Luhmann ID, and writes the file.
func runLearn(_ context.Context, args LearnArgs, deps LearnDeps, stdout io.Writer) error {
	slugErr := validateSlug(args.Slug)
	if slugErr != nil {
		return fmt.Errorf("learn: %w", slugErr)
	}

	vault, err := resolveVault(args.Vault, deps.Getenv)
	if err != nil {
		return fmt.Errorf("learn: %w", err)
	}

	dirErr := deps.StatDir(vault)
	if dirErr != nil {
		return fmt.Errorf("learn: vault %s: %w", vault, dirErr)
	}

	body, bodyErr := io.ReadAll(deps.Stdin)
	if bodyErr != nil {
		return fmt.Errorf("learn: reading stdin: %w", bodyErr)
	}

	path, writeErr := writeLearnUnderLock(args, deps, vault, string(body))
	if writeErr != nil {
		return writeErr
	}

	_, _ = fmt.Fprintln(stdout, path)

	return nil
}

func runLearnFromFactArgs(ctx context.Context, a LearnFactArgs, stdout io.Writer) error {
	deps := newOsLearnDeps()

	return runLearn(ctx, LearnArgs{
		Type:      typeFact,
		Slug:      a.Slug,
		Vault:     a.Vault,
		Target:    a.Target,
		Relation:  a.Relation,
		Source:    a.Source,
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
		Relation:  a.Relation,
		Source:    a.Source,
		Situation: a.Situation,
		Behavior:  a.Behavior,
		Impact:    a.Impact,
		Action:    a.Action,
	}, deps, stdout)
}

func runLearnFromMOCArgs(ctx context.Context, a LearnMOCArgs, stdout io.Writer) error {
	deps := newOsLearnDeps()

	return runLearn(ctx, LearnArgs{
		Type:     typeMOC,
		Slug:     a.Slug,
		Vault:    a.Vault,
		Target:   a.Target,
		Relation: a.Relation,
		Source:   a.Source,
		Topic:    a.Topic,
	}, deps, stdout)
}

// writeLearnUnderLock acquires the vault lock, computes the next Luhmann ID,
// assembles file content, and writes it. The lock spans listing existing IDs
// through writing the new file to prevent ID collisions.
func writeLearnUnderLock(args LearnArgs, deps LearnDeps, vault, body string) (string, error) {
	release, lockErr := deps.Lock(vault)
	if lockErr != nil {
		return "", fmt.Errorf("learn: acquiring lock: %w", lockErr)
	}
	defer release()

	existing, listErr := deps.ListIDs(vault)
	if listErr != nil {
		return "", fmt.Errorf("learn: listing existing IDs: %w", listErr)
	}

	luhmann, idErr := nextLuhmannID(existing, args.Target, args.Relation)
	if idErr != nil {
		return "", fmt.Errorf("learn: %w", idErr)
	}

	when := deps.Now()
	path := learnPath(vault, args.Type, luhmann, args.Slug, when)

	content, contentErr := assembleLearnContent(args, luhmann, when, body)
	if contentErr != nil {
		return "", fmt.Errorf("learn: %w", contentErr)
	}

	writeErr := deps.WriteNew(path, []byte(content))
	if writeErr != nil {
		return "", fmt.Errorf("learn: writing %s: %w", path, writeErr)
	}

	return path, nil
}
