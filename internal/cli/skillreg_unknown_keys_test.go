package cli_test

// Adopt and refresh keep frontmatter keys the typed runbook note model does
// not define (fix-show-amend-reparent-frontmatter design D6;
// skill-runbook-registration "Refresh keeps an unmodeled key", "Adopt keeps
// an unmodeled key"; task 5.1).

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/vaultgraph"
)

// TestAdoptAndRefresh_KeepUnmodeledKeys is the two spec scenarios: a
// refreshed and an adopted note both still carry `luhmann_old: "12"` and a
// nested unknown map with their values.
func TestAdoptAndRefresh_KeepUnmodeledKeys(t *testing.T) {
	t.Parallel()

	const unknown = "luhmann_old: \"12\"\nprovenance:\n    origin: import\n    steps: [a, b]\n"

	for _, mode := range []string{"refresh", "adopt"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			before := strings.Replace(curatePromotedNoteFixture(), "vault: personal\n", "vault: personal\n"+unknown, 1)

			after, err := runSkillAcceptMode(t.Context(), mode, before)
			g.Expect(err).NotTo(HaveOccurred())

			fields := frontmatterOf(after)
			g.Expect(fields["luhmann_old"]).To(Equal("12"))
			g.Expect(fields["provenance"]).To(Equal(map[string]any{"origin": "import", "steps": []any{"a", "b"}}))
		})
	}
}

// TestAdoptAndRefresh_PendingParity: refresh writes `pending: true`; adopt
// writes no pending key at all, even over a note that carried one.
func TestAdoptAndRefresh_PendingParity(t *testing.T) {
	t.Parallel()

	for mode, want := range map[string]any{"refresh": true, "adopt": nil} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			after, err := runSkillAcceptMode(t.Context(), mode, curatePromotedNoteFixturePending())
			g.Expect(err).NotTo(HaveOccurred())

			fields := frontmatterOf(after)
			if want == nil {
				g.Expect(fields).NotTo(HaveKey("pending"))
				g.Expect(after).NotTo(ContainSubstring("pending:"))

				return
			}

			g.Expect(fields["pending"]).To(Equal(want))
		})
	}
}

// TestAdoptAndRefresh_UnknownKeysProperty is P2 (design D7): for LF,
// anchor-free runbook notes carrying 0-3 unknown keys (scalar, list or
// nested map), adopt and refresh write frontmatter that decodes, sets the
// skill fields, and keeps every other key — unknown ones included — at its
// decoded value.
func TestAdoptAndRefresh_UnknownKeysProperty(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		mode := rapid.SampledFrom([]string{"refresh", "adopt"}).Draw(rt, "mode")
		unknown := skillUnknownKeysGen().Draw(rt, "unknown")
		// created: quoted, as learn writes it, so a decoded date compares as
		// the same string on both sides.
		before := strings.Replace(curatePromotedNoteFixture(), "vault: personal\n", "vault: personal\n"+unknown, 1)
		before = strings.Replace(before, "created: 2026-09-21\n", "created: \"2026-09-21\"\n", 1)

		after, err := runSkillAcceptMode(context.Background(), mode, before)
		if err != nil {
			rt.Fatalf("%s: %v", mode, err)
		}

		fields := frontmatterOf(after)
		if fields["skill_key"] != "claude:curate" || fields["skill_hash"] == nil || fields["skill_source"] == nil {
			rt.Fatalf("%s did not set the skill fields: %v", mode, fields)
		}

		beforeFields := frontmatterOf(before)

		for _, key := range []string{"skill_key", "skill_hash", "skill_source", "pending", "luhmann", "aliases"} {
			delete(beforeFields, key)
			delete(fields, key)
		}

		if !reflect.DeepEqual(beforeFields, fields) {
			rt.Fatalf("%s changed a key it does not set:\nbefore %v\nafter  %v", mode, beforeFields, fields)
		}
	})
}

// runSkillAcceptMode refreshes or adopts the curate note content (basename
// 1049.2026-09-21.curate-review-pending-offers) and returns what it wrote.
func runSkillAcceptMode(ctx context.Context, mode, content string) (string, error) {
	const (
		oldBasename = "1049.2026-09-21.curate-review-pending-offers"
		newBasename = "1049.2026-09-21.skill-claude-curate"
	)

	vault := newSkillAcceptFixtureVault()
	vault.put(oldBasename+".md", content)

	skill := installedEngramSkill("curate", []byte("# Curate\n\n1. New text.\n"))

	var (
		stdout bytes.Buffer
		err    error
	)

	written := oldBasename

	if mode == "refresh" {
		err = cli.RefreshSkill(ctx, "/vault", skill, oldBasename, skillAcceptDeps(vault), &stdout)
	} else {
		written = newBasename
		err = cli.AdoptSkillNote(ctx, "/vault", skill, "1049", cli.SkillAdoptDeps{
			Lock:     noLock,
			Scan:     func(v string) ([]vaultgraph.Note, error) { return vaultgraph.ScanVault(vault, v) },
			Rename:   skillAcceptRenameDeps(vault),
			Embedder: skillAcceptFakeEmbedder{},
		}, &stdout)
	}

	after, _ := vault.get(written + ".md")

	return after, err
}

// skillUnknownKeysGen draws 0-3 frontmatter keys no note model defines, each
// a quoted scalar, a list, or a nested map.
func skillUnknownKeysGen() *rapid.Generator[string] {
	return rapid.Custom(func(rt *rapid.T) string {
		value := rapid.StringMatching(`[a-z][a-z0-9 ]{0,10}[a-z0-9]`)
		count := rapid.IntRange(0, 3).Draw(rt, "count")

		var block strings.Builder

		for index := range count {
			key := fmt.Sprintf("x_unknown_%d", index)

			switch rapid.IntRange(0, 2).Draw(rt, key+"Kind") {
			case 0:
				fmt.Fprintf(&block, "%s: %q\n", key, value.Draw(rt, key))
			case 1:
				fmt.Fprintf(&block, "%s:\n", key)

				for _, item := range rapid.SliceOfN(value, 1, 3).Draw(rt, key+"Items") {
					fmt.Fprintf(&block, "    - %q\n", item)
				}
			default:
				fmt.Fprintf(&block, "%s:\n    inner: %q\n    list: [%q]\n", key, value.Draw(rt, key+"Inner"),
					value.Draw(rt, key+"List"))
			}
		}

		return block.String()
	})
}
