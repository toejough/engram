package cli

import (
	"fmt"
	"strings"
	"time"
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
