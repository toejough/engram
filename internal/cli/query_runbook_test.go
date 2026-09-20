package cli_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/cli"
)

func TestRunQuery_NoPhrasesAndNoTextIsRejected(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memFS, vault := plantTriggerVault(t)

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(), cli.QueryArgs{VaultPath: vault}, newQueryDeps(memFS), &out)
	g.Expect(err).To(HaveOccurred())
}

// TestRunQuery_NoTextPayloadUnchanged: without --text (or with text hitting
// nothing) trigger matching never runs and no item carries "trigger".
func TestRunQuery_NoTextPayloadUnchanged(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memFS, vault := plantTriggerVault(t)

	base := cli.QueryArgs{Phrases: triggerPhrases, Limit: 20}

	_, noText := runTriggerQuery(t, memFS, vault, base)

	miss := base
	miss.Text = "nothing relevant here"

	_, missText := runTriggerQuery(t, memFS, vault, miss)

	g.Expect(missText).To(Equal(noText))
	g.Expect(noText).NotTo(ContainSubstring("- trigger\n"), "no item carries provenance role trigger")
}

func TestRunQuery_NonRunbookTriggersNeverHit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	plantNoteWithSidecar(t, memFS, vault, "1.2026-09-20.fact-with-triggers.md",
		"---\ntype: fact\nsituation: a fact\ntriggers:\n    - /please\n---\n\nInformation learned: a fact.\n")

	parsed, _ := runTriggerQuery(t, memFS, vault, cli.QueryArgs{
		Phrases: []string{"a fact"}, Text: "/please do it", Limit: 20,
	})

	for _, item := range parsed.Items {
		g.Expect(item.Provenances).NotTo(ContainElement("trigger"))
	}
}

// TestRunQuery_OversizedRedFlagsListKeepsNewestEntry reproduces engram#763:
// a runbook whose red_flags list is long enough to exceed the preview
// budget must still deliver its most-recently-appended entry through the
// query payload, not silently drop it.
func TestRunQuery_OversizedRedFlagsListKeepsNewestEntry(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	var body strings.Builder

	body.WriteString("---\ntype: runbook\nsituation: dispatching a subagent for a scoped unit of work\n" +
		"done_when: the dispatch is recorded\nred_flags:\n")

	for range 20 {
		body.WriteString("    - " + strings.Repeat("a pre-existing red flag entry with enough filler text ", 3) + "\n")
	}

	body.WriteString("    - the newly added red flag entry for this specific defect\n---\n\n1. Dispatch\n2. Record\n")

	plantNoteWithSidecar(t, memFS, vault, "1.2026-09-18.oversized-red-flags.md", body.String())

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"dispatching a subagent for a scoped unit of work"}, VaultPath: vault, Limit: 20},
		newQueryDeps(memFS), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed queryParsed

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())

	var runbookContent string

	for _, item := range parsed.Items {
		if item.Kind == "runbook" {
			runbookContent = item.Content
		}
	}

	g.Expect(runbookContent).To(ContainSubstring("the newly added red flag entry for this specific defect"))
	g.Expect(runbookContent).To(ContainSubstring("EARLIER RED_FLAGS OMITTED"))
}

// TestRunQuery_RunbookCompetesInMainMatchedSet proves runbook notes receive no
// exclusion treatment (unlike qa-question) and rank purely by situation
// similarity, identical to fact/feedback — design.md Decision 2/3,
// recall-runbook-surfacing spec.
func TestRunQuery_RunbookCompetesInMainMatchedSet(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	plantNoteWithSidecar(t, memFS, vault, "1.2026-08-24.release-runbook.md",
		"---\ntype: runbook\nsituation: releasing a Go module version\ndone_when: the tag is pushed\n---\n\n"+
			"1. Tag the release\n2. Push the tag\n")
	plantNoteWithSidecar(t, memFS, vault, "2.2026-08-24.release-fact.md",
		"---\ntype: fact\nsituation: releasing a Go module version\n---\n\n"+
			"Information learned: when releasing a Go module version, tags matter.\n")

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"releasing a Go module version"}, VaultPath: vault, Limit: 20},
		newQueryDeps(memFS), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed queryParsed

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())

	var runbookFound, factFound bool

	var runbookScore, factScore float32

	for _, item := range parsed.Items {
		switch item.Kind {
		case "runbook":
			runbookFound = true
			runbookScore = item.Score
		case "fact":
			factFound = true
			factScore = item.Score
		}
	}

	g.Expect(runbookFound).To(BeTrue(), "runbook note must appear in the matched set, not excluded")
	g.Expect(factFound).To(BeTrue())

	// Identical situation text on both notes → identical score. No
	// kind-specific boost or penalty differentiates a runbook from a fact.
	g.Expect(runbookScore).To(BeNumerically("~", factScore, 0.0001))

	foundInCandidates := false

	for _, c := range parsed.Clusters {
		for _, cand := range c.CandidateL2s {
			if cand.Path == "1.2026-08-24.release-runbook.md" {
				foundInCandidates = true
			}
		}
	}

	g.Expect(foundInCandidates).To(BeTrue(), "runbook note must be nominated in candidate_l2s, like fact/feedback")
}

// TestRunQuery_RunbookItemContentIncludesRedFlags proves a runbook note's
// red_flags entries reach the query payload's item content, so an agent
// never has to fall back to `engram show` just to see them
// (recall-runbook-surfacing spec, "Red flags visible in the query payload").
func TestRunQuery_RunbookItemContentIncludesRedFlags(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	plantNoteWithSidecar(t, memFS, vault, "1.2026-08-24.history-rewrite.md",
		"---\ntype: runbook\nsituation: rewriting git history\n"+
			"done_when: the backup branch still has every original commit\n"+
			"red_flags:\n    - filter-branch on all refs sweeps the backup branch\n"+
			"    - force-push without --force-with-lease\n---\n\n"+
			"1. Create a backup branch\n2. Rewrite history\n")

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(),
		cli.QueryArgs{Phrases: []string{"rewriting git history"}, VaultPath: vault, Limit: 20},
		newQueryDeps(memFS), &out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var parsed queryParsed

	g.Expect(yaml.Unmarshal(out.Bytes(), &parsed)).NotTo(HaveOccurred())

	var runbookContent string

	for _, item := range parsed.Items {
		if item.Kind == "runbook" {
			runbookContent = item.Content
		}
	}

	g.Expect(runbookContent).To(ContainSubstring("red_flags:"))
	g.Expect(runbookContent).To(ContainSubstring("filter-branch on all refs sweeps the backup branch"))
	g.Expect(runbookContent).To(ContainSubstring("force-push without --force-with-lease"))
}

func TestRunQuery_TextWithoutPhrasesReturnsJustHit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memFS, vault := plantTriggerVault(t)

	parsed, _ := runTriggerQuery(t, memFS, vault, cli.QueryArgs{Text: "/please anything", Limit: 20})

	g.Expect(parsed.Items).To(HaveLen(1))

	if len(parsed.Items) == 1 {
		g.Expect(parsed.Items[0].Path).To(Equal(triggerRunbookName))
		g.Expect(parsed.Items[0].Provenances).To(Equal([]string{"trigger"}))
	}
}

// TestRunQuery_TriggerHitDoesNotConsumeLimit: --limit 1 returns the hit plus
// one similarity item.
func TestRunQuery_TriggerHitDoesNotConsumeLimit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := t.TempDir()
	memFS := newInMemoryFS()

	plantNoteWithSidecar(t, memFS, vault, triggerRunbookName, triggerRunbookNote)
	plantNoteWithSidecar(t, memFS, vault, triggerFeedbackName, triggerFeedbackNote)
	plantNoteWithSidecar(t, memFS, vault, "3.2026-09-20.other-feedback.md",
		"---\ntype: feedback\nsituation: fixing a flaky login test quickly\n---\n\nBehavior: x. Impact: y. Action: z.\n")

	parsed, _ := runTriggerQuery(t, memFS, vault, cli.QueryArgs{
		Phrases: triggerPhrases, Text: "/please fix the flaky login test", Limit: 1,
	})

	g.Expect(parsed.Items).To(HaveLen(2))

	if len(parsed.Items) == 2 {
		g.Expect(parsed.Items[0].Path).To(Equal(triggerRunbookName))
		g.Expect(parsed.Items[1].Path).NotTo(Equal(triggerRunbookName))
	}
}

// TestRunQuery_TriggerHitSurfacesFirst: a runbook whose trigger appears in
// --text is items[0] with provenance "trigger", ahead of a strongly
// similarity-matched feedback note (runbook-lexical-triggers spec).
func TestRunQuery_TriggerHitSurfacesFirst(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memFS, vault := plantTriggerVault(t)

	parsed, _ := runTriggerQuery(t, memFS, vault, cli.QueryArgs{
		Phrases: triggerPhrases, Text: "/please fix the flaky login test", Limit: 20,
	})

	g.Expect(parsed.Items).NotTo(BeEmpty())

	if len(parsed.Items) == 0 {
		return
	}

	g.Expect(parsed.Items[0].Path).To(Equal(triggerRunbookName))
	g.Expect(parsed.Items[0].Kind).To(Equal("runbook"))
	g.Expect(parsed.Items[0].Provenances).To(ContainElement("trigger"))
	g.Expect(parsed.Items[0].Content).To(ContainSubstring("Do it"))

	paths := make([]string, 0, len(parsed.Items))
	for _, item := range parsed.Items {
		paths = append(paths, item.Path)
	}

	g.Expect(paths).To(ContainElement(triggerFeedbackName))
}

func TestRunQuery_TriggerMatchIsCaseAndWhitespaceInsensitive(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	memFS, vault := plantTriggerVault(t)

	parsed, _ := runTriggerQuery(t, memFS, vault, cli.QueryArgs{
		Phrases: triggerPhrases, Text: "PLEASE   TAKE   this\tend-to-end: rename X", Limit: 20,
	})

	g.Expect(parsed.Items).NotTo(BeEmpty())

	if len(parsed.Items) == 0 {
		return
	}

	g.Expect(parsed.Items[0].Path).To(Equal(triggerRunbookName))
	g.Expect(parsed.Items[0].Provenances).To(ContainElement("trigger"))
}

// unexported constants.
const (
	triggerFeedbackName = "2.2026-09-20.flaky-feedback.md"
	triggerFeedbackNote = "---\ntype: feedback\nsituation: fixing a flaky login test\n---\n\n" +
		"Behavior: x. Impact: y. Action: z.\n"
	triggerRunbookName = "1.2026-09-20.drive-ask.md"
	triggerRunbookNote = "---\ntype: runbook\nsituation: zebra migration planning at the savanna\ndone_when: done\ntriggers:\n    - /please\n    - take this end-to-end\n---\n\n1. Do it\n" //nolint:lll // fixture literal
)

// unexported variables.
var (
	triggerPhrases = []string{"fixing a flaky login test", "diagnosing intermittent test failures"}
)

func plantTriggerVault(t *testing.T) (*inMemoryFS, string) {
	t.Helper()

	vault := t.TempDir()
	memFS := newInMemoryFS()

	plantNoteWithSidecar(t, memFS, vault, triggerRunbookName, triggerRunbookNote)
	plantNoteWithSidecar(t, memFS, vault, triggerFeedbackName, triggerFeedbackNote)

	return memFS, vault
}

func runTriggerQuery(t *testing.T, memFS *inMemoryFS, vault string, args cli.QueryArgs) (queryParsed, string) {
	t.Helper()

	args.VaultPath = vault

	var out bytes.Buffer

	err := cli.RunQuery(context.Background(), args, newQueryDeps(memFS), &out)
	if err != nil {
		t.Fatalf("RunQuery: %v", err)
	}

	var parsed queryParsed

	err = yaml.Unmarshal(out.Bytes(), &parsed)
	if err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	return parsed, out.String()
}
