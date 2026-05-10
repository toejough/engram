package cli

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// unexported constants.
const (
	mocSubdir       = "MOCs"
	permanentSubdir = "Permanent"
	typeMOC         = "moc"
)

// unexported variables.
var (
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

func extractLuhmannFromFilename(name string) (string, bool) {
	m := luhmannFilenamePattern.FindStringSubmatch(name)
	if m == nil {
		return "", false
	}

	return m[1], true
}

func promotePath(vault, memType, luhmann, slug string, when time.Time) string {
	subdir := permanentSubdir
	if memType == typeMOC {
		subdir = mocSubdir
	}

	filename := fmt.Sprintf("%s.%s.%s.md", luhmann, when.Format(dateFormat), slug)

	return filepath.Join(vault, subdir, filename)
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
