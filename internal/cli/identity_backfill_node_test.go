package cli_test

// Identity backfill as a YAML-node edit (#789 defect 2, design D2; spec
// vault-note-identity "Backfill for pre-existing notes missing identity
// fields"): every key backfill does not set survives with its value, an
// anchored key it would set refuses that note untouched, and the run
// stamps every other note first.

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/embed"
)

// TestBackfillIdentity_ConvertsCRLFNote: backfill reads a CRLF note as LF
// (#789 design D2): a flagged note in any CRLF layout is stamped and
// written as LF — byte-identical to the backfill of its LF form — with its
// sidecar rebuilt; a note backfill does not write stays CRLF; a dry run
// counts it and writes nothing.
func TestBackfillIdentity_ConvertsCRLFNote(t *testing.T) {
	t.Parallel()

	lfNote := backfillNodeNote("fact", "source: agent\nluhmann_old: \"12\"\n")

	for _, layout := range []string{"all", "frontmatter", "body"} {
		t.Run(layout, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			crlf := newBackfillVault(map[string]string{"1.2026-01-01.a.md": crlfLayout(lfNote, layout)})
			lfVault := newBackfillVault(map[string]string{"1.2026-01-01.a.md": lfNote})

			stamped, err := crlf.backfill(t.Context(), false)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(stamped).To(Equal(1), "a CRLF note missing identity is stamped, not skipped")

			_, lfErr := lfVault.backfill(t.Context(), false)
			g.Expect(lfErr).NotTo(HaveOccurred())

			written := crlf.files["/vault/1.2026-01-01.a.md"]
			g.Expect(written).NotTo(ContainSubstring("\r\n"))
			g.Expect(written).To(Equal(lfVault.files["/vault/1.2026-01-01.a.md"]))

			sidecar, sidecarErr := embed.UnmarshalSidecar([]byte(crlf.files["/vault/1.2026-01-01.a.vec.json"]))
			g.Expect(sidecarErr).NotTo(HaveOccurred())

			if sidecarErr != nil {
				return
			}

			g.Expect(sidecar.ContentHash).To(Equal(embed.ContentHash([]byte(written))),
				"a converted note's sidecar is rebuilt")
		})
	}

	t.Run("already stamped stays CRLF", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		stampedNote := toCRLF(backfillNodeNote("fact", "source: agent\nuser: alice@example.com\nvault: personal\n"))
		vault := newBackfillVault(map[string]string{"1.2026-01-01.a.md": stampedNote})

		stamped, err := vault.backfill(t.Context(), false)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(stamped).To(BeZero())
		g.Expect(vault.writes).To(BeZero())
		g.Expect(vault.files["/vault/1.2026-01-01.a.md"]).To(Equal(stampedNote))
	})

	t.Run("dry run", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := newBackfillVault(map[string]string{"1.2026-01-01.a.md": toCRLF(lfNote)})

		stamped, err := vault.backfill(t.Context(), true)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(stamped).To(Equal(1))
		g.Expect(vault.writes).To(BeZero())
	})
}

// TestBackfillIdentity_KeepsAnchorOnUneditedKey: an anchor on a key
// backfill does not edit, and the key aliasing it, both survive.
func TestBackfillIdentity_KeepsAnchorOnUneditedKey(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	vault := newBackfillVault(map[string]string{
		"1.2026-01-01.a.md": backfillNodeNote("fact", "source: &src agent\nx_src: *src\n"),
	})

	stamped, err := vault.backfill(t.Context(), false)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stamped).To(Equal(1))

	written := vault.files["/vault/1.2026-01-01.a.md"]
	g.Expect(written).To(ContainSubstring("source: &src agent\n"))
	g.Expect(written).To(ContainSubstring("x_src: *src\n"))

	decoded := decodeBackfillFrontmatter(t, written)
	g.Expect(decoded).To(HaveKeyWithValue("source", "agent"))
	g.Expect(decoded).To(HaveKeyWithValue("x_src", "agent"))
	g.Expect(decoded).To(HaveKeyWithValue("user", "bob@example.com"))
}

// TestBackfillIdentity_KeepsUnmodeledKeys: a scalar and a nested map the
// typed note model does not define survive a backfill of fact and
// feedback notes.
func TestBackfillIdentity_KeepsUnmodeledKeys(t *testing.T) {
	t.Parallel()

	for _, noteType := range []string{"fact", "feedback"} {
		t.Run(noteType, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			vault := newBackfillVault(map[string]string{
				"1.2026-01-01.a.md": backfillNodeNote(noteType,
					"source: agent\nluhmann_old: \"12\"\nprovenance:\n    origin: import\n    steps:\n        - a\n        - b\n"),
			})

			stamped, err := vault.backfill(t.Context(), false)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(stamped).To(Equal(1))

			decoded := decodeBackfillFrontmatter(t, vault.files["/vault/1.2026-01-01.a.md"])
			g.Expect(decoded).To(HaveKeyWithValue("luhmann_old", "12"))
			g.Expect(decoded).To(HaveKeyWithValue("provenance", map[string]any{
				"origin": "import", "steps": []any{"a", "b"},
			}))
			g.Expect(decoded).To(HaveKeyWithValue("repo", "git@github.com:example/vault.git"))
			g.Expect(decoded).To(HaveKeyWithValue("user", "bob@example.com"))
			g.Expect(decoded).To(HaveKeyWithValue("vault", "personal"))
		})
	}
}

// TestBackfillIdentity_PreservesUnknownKeysAndAnchorsProperty: for any
// fact or feedback note missing identity, with unknown keys, optionally an
// anchor on an unedited key, and any CRLF layout, backfill stamps it, writes
// it as LF, and every other key decodes to its input value.
func TestBackfillIdentity_PreservesUnknownKeysAndAnchorsProperty(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		noteType := rapid.SampledFrom([]string{"fact", "feedback"}).Draw(rt, "type")
		extra := drawBackfillExtraKeys(rt)

		source := "source: agent\n"
		if rapid.Bool().Draw(rt, "anchor") {
			source = "source: &src agent\nx_src: *src\n"
		}

		lfInput := backfillNodeNote(noteType, source+extra)
		before := decodeBackfillFrontmatter(rt, lfInput)

		layout := rapid.SampledFrom([]string{"none", "all", "frontmatter", "body"}).Draw(rt, "crlf")

		input := lfInput
		if layout != "none" {
			input = crlfLayout(lfInput, layout)
		}

		vault := newBackfillVault(map[string]string{"1.2026-01-01.a.md": input})

		stamped, err := vault.backfill(context.Background(), false)
		if err != nil || stamped != 1 {
			rt.Fatalf("backfill: stamped %d, err %v", stamped, err)
		}

		written := vault.files["/vault/1.2026-01-01.a.md"]
		if strings.Contains(written, "\r\n") {
			rt.Fatalf("CRLF (%s) left in the written note: %q", layout, written)
		}

		assertBackfillKeptKeys(rt, before, written)
	})
}

// TestBackfillIdentity_RefusesAnchoredIdentityKey: a flagged note whose
// user: carries an anchor another key aliases is left untouched, the other
// flagged note is still stamped, and the run fails naming the refused
// note and key — on a dry run too, which writes nothing.
func TestBackfillIdentity_RefusesAnchoredIdentityKey(t *testing.T) {
	t.Parallel()

	anchored := backfillNodeNote("fact", "source: agent\nuser: &u \"\"\nx_user: *u\n")
	plain := backfillNodeNote("feedback", "source: agent\n")

	t.Run("apply", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := newBackfillVault(map[string]string{"1.2026-01-01.anchored.md": anchored, "2.2026-01-01.plain.md": plain})

		stamped, err := vault.backfill(t.Context(), false)
		g.Expect(err).To(MatchError(ContainSubstring("1.2026-01-01.anchored")))
		g.Expect(err).To(MatchError(ContainSubstring("YAML anchor: user")))
		g.Expect(stamped).To(Equal(1))
		g.Expect(vault.files["/vault/1.2026-01-01.anchored.md"]).To(Equal(anchored), "the refused note is untouched")
		g.Expect(vault.files["/vault/2.2026-01-01.plain.md"]).To(ContainSubstring("user: bob@example.com"))
	})

	t.Run("dry run", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		vault := newBackfillVault(map[string]string{"1.2026-01-01.anchored.md": anchored, "2.2026-01-01.plain.md": plain})

		stamped, err := vault.backfill(t.Context(), true)
		g.Expect(err).To(MatchError(ContainSubstring("YAML anchor: user")))
		g.Expect(stamped).To(Equal(1), "the dry run counts the note it would stamp")
		g.Expect(vault.writes).To(BeZero())
	})
}

// TestBackfillIdentity_WriteAndRebuildFailuresSurface: a failed note write,
// a failed re-embed of a converted note, and a failed sidecar write each
// fail the run, naming the note.
func TestBackfillIdentity_WriteAndRebuildFailuresSurface(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		embedder embed.Embedder
		failPath string
		want     string
	}{
		{"note write", stubEmbedder{modelID: "stub@4", dims: 4}, ".md", "write"},
		{"re-embed", failingEmbedder{}, "", "re-embedding"},
		{"sidecar write", stubEmbedder{modelID: "stub@4", dims: 4}, ".vec.json", "writing sidecar"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			deps := cli.IdentityDeps{
				Lock:     func(string) (func(), error) { return func() {}, nil },
				ListMD:   func(string) ([]string, error) { return []string{"1.2026-01-01.a.md"}, nil },
				ReadFile: func(string) ([]byte, error) { return []byte(toCRLF(backfillNodeNote("fact", "source: agent\n"))), nil },
				WriteFile: func(path string, _ []byte) error {
					if tc.failPath != "" && strings.HasSuffix(path, tc.failPath) {
						return errBackfillDiskFull
					}

					return nil
				},
				DetectRepo: func(context.Context) string { return "" },
				DetectUser: func(context.Context) string { return "bob@example.com" },
				Getenv:     func(string) string { return "" },
				Embedder:   tc.embedder,
			}

			_, err := cli.ExportBackfillIdentity(t.Context(), "/vault", deps, false)
			g.Expect(err).To(MatchError(ContainSubstring(tc.want)))
			g.Expect(err).To(MatchError(ContainSubstring("1.2026-01-01.a")))
		})
	}
}

// unexported variables.
var (
	errBackfillDiskFull = errors.New("disk full")
)

// backfillVault is an in-memory vault for backfill tests: path → content,
// and a count of writes.
type backfillVault struct {
	files  map[string]string
	writes int
}

func (v *backfillVault) backfill(ctx context.Context, dryRun bool) (int, error) {
	deps := cli.IdentityDeps{
		Lock: func(string) (func(), error) { return func() {}, nil },
		ListMD: func(string) ([]string, error) {
			names := make([]string, 0, len(v.files))
			for path := range maps.Keys(v.files) {
				names = append(names, strings.TrimPrefix(path, "/vault/"))
			}

			slices.Sort(names)

			return names, nil
		},
		ReadFile: func(path string) ([]byte, error) { return []byte(v.files[path]), nil },
		WriteFile: func(path string, data []byte) error {
			v.writes++
			v.files[path] = string(data)

			return nil
		},
		DetectRepo: func(context.Context) string { return "git@github.com:example/vault.git" },
		DetectUser: func(context.Context) string { return "bob@example.com" },
		Getenv:     func(string) string { return "" },
		Embedder:   stubEmbedder{modelID: "stub@4", dims: 4},
	}

	return cli.ExportBackfillIdentity(ctx, "/vault", deps, dryRun)
}

// assertBackfillKeptKeys checks a backfilled note: every key but the
// identity keys and created: decodes to its input value, and user:/vault:
// are stamped.
func assertBackfillKeptKeys(rt *rapid.T, before map[string]any, written string) {
	after := decodeBackfillFrontmatter(rt, written)

	for key, value := range before {
		if slices.Contains([]string{"repo", "user", "vault", "created"}, key) {
			continue
		}

		if fmt.Sprint(after[key]) != fmt.Sprint(value) {
			rt.Fatalf("key %s: before %v, after %v\n%s", key, value, after[key], written)
		}
	}

	if after["user"] != "bob@example.com" || after["vault"] != "personal" {
		rt.Fatalf("identity not stamped: %v", after)
	}
}

// backfillNodeNote is a note missing identity with extra frontmatter lines
// (which must include source:) after created:.
func backfillNodeNote(noteType, extra string) string {
	fields := map[string]string{
		"fact":     "situation: s\nsubject: a\npredicate: b\nobject: c\n",
		"feedback": "situation: s\nbehavior: b\nimpact: i\naction: act\n",
	}[noteType]

	return "---\ntype: " + noteType + "\n" + fields + "luhmann: \"1\"\ncreated: \"2026-01-01\"\n" + extra +
		"---\n\nbody text\n"
}

func decodeBackfillFrontmatter(tester failer, content string) map[string]any {
	tester.Helper()

	const frontmatterParts = 3 // before, frontmatter, body

	parts := strings.SplitN(content, "---\n", frontmatterParts)
	if len(parts) < frontmatterParts {
		tester.Fatalf("no frontmatter in %q", content)

		return nil
	}

	decoded := map[string]any{}

	err := yaml.Unmarshal([]byte(parts[1]), &decoded)
	if err != nil {
		tester.Fatalf("decoding frontmatter: %v\n%s", err, content)
	}

	return decoded
}

// drawBackfillExtraKeys draws 0–3 unknown keys with scalar, list or map
// values, as frontmatter lines.
func drawBackfillExtraKeys(rt *rapid.T) string {
	var lines strings.Builder

	const maxUnknownKeys = 3 // design D4: 0–3 unknown keys

	count := rapid.IntRange(0, maxUnknownKeys).Draw(rt, "unknown-count")

	for index := range count {
		key := fmt.Sprintf("x_unknown_%d", index)
		word := rapid.StringMatching(`[a-z]{1,8}`).Draw(rt, key+"-word")

		switch rapid.IntRange(0, 2).Draw(rt, key+"-shape") { // scalar, list, map
		case 0:
			lines.WriteString(key + ": " + word + "\n")
		case 1:
			lines.WriteString(key + ":\n    - " + word + "\n    - " + word + "2\n")
		default:
			lines.WriteString(key + ":\n    inner: " + word + "\n")
		}
	}

	return lines.String()
}

func newBackfillVault(notes map[string]string) *backfillVault {
	files := make(map[string]string, len(notes))
	for name, content := range notes {
		files["/vault/"+name] = content
	}

	return &backfillVault{files: files}
}
