package cli

// Scenario tests of the runbook-trigger-matching component (792b22): the trigger index, the hits and
// the placing of the hits. White-box: matchTriggers, applyTriggerHits and loadAllVaultNotesMeta are unexported.

import (
	"errors"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/engram/internal/vaultgraph"
)

func Test792b22c1a1_TriggerIndex_RunbookTriggersIndexedInOrder(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		runbookBasename = "1.2026-01-01.rtm1-runbook"
		pleaseTrigger   = "/please"
		endToEnd        = "take this end-to-end"
	)

	hits := []compatibleSidecar{{note: vaultgraph.Note{Basename: runbookBasename}}}
	read := rtm1ReadFrom(map[string]string{
		"/vault/" + runbookBasename + ".md": rtm1Runbook("project: widgets\n", pleaseTrigger, endToEnd),
	})

	meta := loadAllVaultNotesMeta(hits, "/vault", read)

	g.Expect(meta.TriggerIndex).To(HaveLen(1))
	g.Expect(meta.TriggerIndex).To(HaveKeyWithValue(runbookBasename, []string{pleaseTrigger, endToEnd}))
}

func Test792b22c3a1_MatchTriggers_HitsListedOnceInAscendingOrder(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		first  = "1.2026-01-01.rtm1-first"
		second = "2.2026-01-01.rtm1-second"
		third  = "3.2026-01-01.rtm1-third"
		absent = "4.2026-01-01.rtm1-absent"
	)

	// Keys are deliberately not in ascending order of how they are written; the third runbook is a hit by
	// two triggers, and a fourth runbook has a trigger that does not occur.
	index := map[string][]string{
		third:  {"rotate the keys", "curate"},
		first:  {"/please"},
		absent: {"unrelated cue"},
		second: {"take this end-to-end"},
	}

	hits := matchTriggers("/please curate the offers, rotate the keys and take this end-to-end", index)

	g.Expect(hits).To(Equal([]string{first, second, third}))
}

func Test792b22c3a2_MatchTriggers_EmptyMessageHasNoHits(t *testing.T) {
	t.Parallel()

	index := map[string][]string{"1.2026-01-01.rtm1-curate": {"curate"}}

	t.Run("the empty message", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		hits := matchTriggers("", index)

		g.Expect(hits).To(BeEmpty())
	})

	t.Run("a message of only whitespace is empty after normalization", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		hits := matchTriggers(" \t\n ", index)

		g.Expect(hits).To(BeEmpty())
	})
}

func Test792b22c4a2_ApplyTriggerHits_RankedHitKeepsItsScore(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		runbookBasename = "1.2026-01-01.rtm1-runbook"
		runbookScore    = float32(0.42)
		strongerScore   = float32(0.9)
	)

	runbookContent := rtm1Runbook("", "/please")
	stronger := resolvedItem{
		notePath: "2.2026-01-01.rtm1-feedback.md", content: "feedback", score: strongerScore,
		provenances: []string{provenanceDirect},
	}
	ranked := resolvedItem{
		notePath: pathOf(runbookBasename), content: runbookContent, score: runbookScore,
		provenances: []string{provenanceDirect},
	}
	meta := AllVaultNotesMeta{ContentByBasename: map[string]string{runbookBasename: "stale meta content"}}

	placed := applyTriggerHits([]resolvedItem{stronger, ranked}, []string{runbookBasename}, meta, "")

	g.Expect(placed).To(HaveLen(2))

	if len(placed) != 2 {
		return
	}

	g.Expect(placed[0].notePath).To(Equal(pathOf(runbookBasename)))
	g.Expect(placed[0].provenances).To(Equal([]string{provenanceDirect, provenanceTrigger}))
	g.Expect(placed[0].score).To(Equal(runbookScore))
	g.Expect(placed[0].content).To(Equal(runbookContent))
	g.Expect(placed[1].notePath).To(Equal(stronger.notePath))
	g.Expect(placed[1].provenances).To(Equal([]string{provenanceDirect}))
}

func Test792b22c4a3_ApplyTriggerHits_ChunkIsNotAHit(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		runbookBasename = "1.2026-01-01.rtm1-runbook"
		chunkScore      = float32(0.8)
	)

	runbookContent := rtm1Runbook("", "/please")
	chunk := resolvedItem{
		notePath: pathOf(runbookBasename), content: "a passage of the runbook", score: chunkScore,
		kind: chunkItemKind, provenances: []string{provenanceDirect},
	}
	meta := AllVaultNotesMeta{ContentByBasename: map[string]string{runbookBasename: runbookContent}}

	placed := applyTriggerHits([]resolvedItem{chunk}, []string{runbookBasename}, meta, "")

	g.Expect(placed).To(HaveLen(2))

	if len(placed) != 2 {
		return
	}

	// The hit is the note itself, added from its content; the chunk follows it untouched.
	g.Expect(placed[0].kind).NotTo(Equal(chunkItemKind))
	g.Expect(placed[0].provenances).To(Equal([]string{provenanceTrigger}))
	g.Expect(placed[0].content).To(Equal(runbookContent))
	g.Expect(placed[1]).To(Equal(chunk))
	g.Expect(placed[1].provenances).NotTo(ContainElement(provenanceTrigger))
}

func Test792b22c4a4_ApplyTriggerHits_HitOutsideRequestedProjectIsNotAdded(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		otherBasename = "1.2026-01-01.rtm1-other"
		sameBasename  = "2.2026-01-01.rtm1-same"
		project       = "widgets"
	)

	meta := AllVaultNotesMeta{ContentByBasename: map[string]string{
		otherBasename: rtm1Runbook("project: gadgets\n", "/please"),
		sameBasename:  rtm1Runbook("project: widgets\n", "/please"),
	}}

	placed := applyTriggerHits(nil, []string{otherBasename, sameBasename}, meta, project)

	// The hit in the requested project is the control: only the other project's note is dropped.
	g.Expect(placed).To(HaveLen(1))

	if len(placed) != 1 {
		return
	}

	g.Expect(placed[0].notePath).To(Equal(pathOf(sameBasename)))
	g.Expect(placed[0].provenances).To(Equal([]string{provenanceTrigger}))
}

func Test792b22c4a5_ApplyTriggerHits_NoHitsChangesNothing(t *testing.T) {
	t.Parallel()

	const (
		firstScore  = float32(0.7)
		secondScore = float32(0.3)
	)

	items := func() []resolvedItem {
		return []resolvedItem{
			{notePath: "1.2026-01-01.rtm1-a.md", content: "a", score: firstScore, provenances: []string{provenanceDirect}},
			{notePath: "2.2026-01-01.rtm1-b.md", content: "b", score: secondScore, kind: chunkItemKind},
		}
	}
	meta := AllVaultNotesMeta{ContentByBasename: map[string]string{"3.2026-01-01.rtm1-c": rtm1Runbook("", "x")}}

	t.Run("nil hits", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		g.Expect(applyTriggerHits(items(), nil, meta, "")).To(Equal(items()))
	})

	t.Run("empty hits with a project requested", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		g.Expect(applyTriggerHits(items(), []string{}, meta, "widgets")).To(Equal(items()))
	})
}

// rtm1ReadFrom returns a read function serving the given path-to-content map, failing for any other path.
func rtm1ReadFrom(files map[string]string) func(string) ([]byte, error) {
	return func(path string) ([]byte, error) {
		content, ok := files[path]
		if !ok {
			return nil, errors.New("rtm1ReadFrom: no such file: " + path)
		}

		return []byte(content), nil
	}
}

// rtm1Runbook builds a runbook note with the given extra frontmatter lines (each ending in a newline) and
// triggers.
func rtm1Runbook(extra string, triggers ...string) string {
	var note strings.Builder

	note.WriteString("---\ntype: runbook\nsituation: s\ndone_when: d\n")

	if len(triggers) > 0 {
		note.WriteString("triggers:\n")

		for _, trigger := range triggers {
			note.WriteString("    - " + trigger + "\n")
		}
	}

	note.WriteString(extra + "luhmann: \"1\"\n---\n\n1. step\n")

	return note.String()
}
