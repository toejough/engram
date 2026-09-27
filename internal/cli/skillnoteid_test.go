package cli_test

import (
	"fmt"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/toejough/engram/internal/cli"
)

func TestFindSkillNote_BySkillKey(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const note = "2000.2026-09-26.skill-superpowers-brainstorming.md"

	vault := newSkillregFixtureVault()
	vault.put(note, keyedSkillNote("abc", "superpowers:brainstorming"))

	basename, hash, found, err := cli.FindSkillNote("/vault", "superpowers:brainstorming", []string{note}, vault.readFile)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeTrue())
	g.Expect(basename).To(Equal(strings.TrimSuffix(note, ".md")))
	g.Expect(hash).To(Equal("abc"))

	// A note with a skill_key is never also matched by its slug remainder.
	_, _, found, err = cli.FindSkillNote("/vault", "superpowers-brainstorming", []string{note}, vault.readFile)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeFalse())
}

// TestFindSkillNote_IdentityProperty is model-based: over a generated vault
// of keyed, unkeyed and hash-less notes, a key's lookup finds exactly the
// keyed notes the generator made for it with a skill_hash, reporting a
// duplicate when there are several; an unkeyed note, whatever its slug, is
// no key's note (design D3).
func TestFindSkillNote_IdentityProperty(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		keys := []string{
			"claude:route", "claude:curate", "superpowers:brainstorming", "claude:cmd:opsx:apply", "pi:route",
		}
		vault := newSkillregFixtureVault()
		count := rapid.IntRange(0, 8).Draw(rt, "count")
		names := make([]string, 0, count)
		expected := map[string][]string{}

		for index := range count {
			key := rapid.SampledFrom(keys).Draw(rt, fmt.Sprintf("key%d", index))
			basename := fmt.Sprintf("%d.2026-09-26.%s", index+1, cli.SkillKeySlug(key))
			hash := fmt.Sprintf("h%d", index)

			switch rapid.IntRange(0, 2).Draw(rt, fmt.Sprintf("form%d", index)) {
			case 0:
				vault.put(basename+".md", keyedSkillNote(hash, key))
				expected[key] = append(expected[key], basename)
			case 1:
				vault.put(basename+".md", runbookNote(hash))
			default:
				vault.put(basename+".md", "---\ntype: runbook\nsituation: s\ndone_when: d\n---\n\nb\n")
			}

			names = append(names, basename+".md")
		}

		slugRemainders := []string{"claude-route", "route", "curate", "superpowers-brainstorming", "claude-cmd-opsx-apply"}
		for _, key := range append(keys, slugRemainders...) {
			basename, _, found, err := cli.FindSkillNote("/vault", key, names, vault.readFile)
			assertNoteLookup(rt, key, expected[key], basename, found, err)
		}
	})
}

// TestFindSkillNote_KeyedNoteBesideAnUnkeyedOneIsNoDuplicate: an unkeyed
// legacy note is no skill note, so it never duplicates the keyed note of
// the same skill.
func TestFindSkillNote_KeyedNoteBesideAnUnkeyedOneIsNoDuplicate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const (
		legacy = "1049.2026-09-21.skill-curate.md"
		keyed  = "2001.2026-09-26.skill-claude-curate.md"
	)

	vault := newSkillregFixtureVault()
	vault.put(legacy, legacySkillNote("1049", "curate", "abc"))
	vault.put(keyed, keyedSkillNote("def", "claude:curate"))

	basename, hash, found, err := cli.FindSkillNote("/vault", "claude:curate", []string{legacy, keyed}, vault.readFile)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeTrue())
	g.Expect(basename).To(Equal(strings.TrimSuffix(keyed, ".md")))
	g.Expect(hash).To(Equal("def"))
}

func TestFindSkillNote_SkillKeyWithoutHashIsNotASkillNote(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	const note = "2000.2026-09-26.skill-superpowers-brainstorming.md"

	vault := newSkillregFixtureVault()
	vault.put(note, "---\ntype: runbook\nsituation: s\ndone_when: d\nskill_key: superpowers:brainstorming\n---\n\nb\n")

	_, _, found, err := cli.FindSkillNote("/vault", "superpowers:brainstorming", []string{note}, vault.readFile)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(BeFalse())
}

// unexported constants.
const (
	note820Key = "edits-validated-by-baseline-pressure-tests"
)

// unexported variables.
var (
	legacyNotes = map[string]string{
		"1036": "route", "1045": "please", "1049": "curate", "1053": "write-memory", "1067": "learn", "1068": "recall",
	}
)

// assertDuplicateNamed checks that err is a duplicate error naming every
// one of want.
func assertDuplicateNamed(rt *rapid.T, key string, want []string, err error) {
	if err == nil {
		rt.Fatalf("key %q: want a duplicate error for %v", key, want)

		return
	}

	for _, dup := range want {
		if !strings.Contains(err.Error(), dup) {
			rt.Fatalf("key %q: error %v does not name %s", key, err, dup)
		}
	}
}

// assertNoteLookup checks one FindSkillNote result against the notes the
// generator made for key: none, exactly one, or a duplicate error naming
// every one.
func assertNoteLookup(rt *rapid.T, key string, want []string, basename string, found bool, err error) {
	if len(want) > 1 {
		assertDuplicateNamed(rt, key, want, err)

		return
	}

	wantBasename := ""
	if len(want) == 1 {
		wantBasename = want[0]
	}

	if err != nil || found != (len(want) == 1) || basename != wantBasename {
		rt.Fatalf("key %q: got %q found=%v err=%v, want %v", key, basename, found, err, want)
	}
}

// keyedSkillNote renders a runbook note carrying skill_hash and skill_key.
func keyedSkillNote(hash, key string) string {
	return "---\ntype: runbook\nsituation: s\ndone_when: d\nskill_hash: \"" + hash + "\"\nskill_key: \"" + key +
		"\"\nskill_source: ~/x/SKILL.md\n---\n\nbody\n"
}

// legacyVault holds the six legacy skill notes (hash of the skill's name as
// content) plus a note-820-shaped runbook without skill_hash.
func legacyVault() (*skillregFixtureVault, []string) {
	vault := newSkillregFixtureVault()
	names := make([]string, 0, len(legacyNotes)+1)

	for luhmann, key := range legacyNotes {
		name := luhmann + ".2026-09-18.skill-" + key + ".md"
		vault.put(name, legacySkillNote(luhmann, key, cli.SkillContentHash([]byte(key))))
		names = append(names, name)
	}

	note820 := "820.2026-08-29.skill-" + note820Key + ".md"
	vault.put(note820, "---\ntype: runbook\ntier: L2\nsituation: editing a SKILL.md\ndone_when: d\n"+
		"luhmann: \"820\"\ncreated: \"2026-08-29\"\nsource: migrated\n---\n\n1. Run a baseline.\n")
	names = append(names, note820)

	return vault, names
}
