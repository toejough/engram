//go:build targ && integration

package dev

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"unicode/utf8"

	. "github.com/onsi/gomega"
	"go.yaml.in/yaml/v3"
)

// Test792b32c1a2_AgentInstructionBundle_EveryShippedSkillHasADescription reads the SKILL.md of every skill the
// repository ships: its frontmatter parses and the description is not empty.
func Test792b32c1a2_AgentInstructionBundle_EveryShippedSkillHasADescription(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	skills := aibShippedSkills(t)
	g.Expect(skills).To(ContainElements(aibSkillsWithFreeTextExamples))

	for _, skill := range skills {
		t.Run(skill, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			description := aibSkillDescription(t, skill)

			g.Expect(strings.TrimSpace(description)).NotTo(BeEmpty())
		})
	}
}

// Test792b32c1a3_AgentInstructionBundle_ADescriptionStaysUnderTheLimit reads the SKILL.md of every skill the repository
// ships: the parsed description is fewer than 1024 characters, counted as characters and not bytes.
func Test792b32c1a3_AgentInstructionBundle_ADescriptionStaysUnderTheLimit(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	skills := aibShippedSkills(t)
	g.Expect(skills).To(ContainElements(aibSkillsWithFreeTextExamples))

	for _, skill := range skills {
		t.Run(skill, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			description := aibSkillDescription(t, skill)

			g.Expect(utf8.RuneCountInString(description)).To(BeNumerically("<", aibMaxDescriptionLength))
		})
	}
}

// Test792b32c2a1_AgentInstructionBundle_ASkillsExampleCommandUsesTheAttachedForm scans every SKILL.md, every
// references markdown file of a skill and every guidance document the repository ships: no free-text flag is written
// as --flag "value", the space form.
func Test792b32c2a1_AgentInstructionBundle_ASkillsExampleCommandUsesTheAttachedForm(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	paths := aibInstructionPaths(t)
	g.Expect(paths).To(ContainElements(
		"agent-instructions/skills/recall/SKILL.md",
		"agent-instructions/skills/write-memory/SKILL.md",
		"agent-instructions/guidance/shim.md",
	))

	offenders := make(map[string][]string)

	for _, path := range paths {
		hits := aibSpaceFormHits(shippedText(t, path))
		if len(hits) > 0 {
			offenders[path] = hits
		}
	}

	g.Expect(offenders).To(BeEmpty())
}

// Test792b32c3a1_AgentInstructionBundle_ASkillStatesTheRuleAndItsReason reads every SKILL.md the repository ships
// that shows an engram command with a free-text flag: it states that the value must be written --flag="value" and
// names a value beginning with a dash as the reason, in one sentence.
func Test792b32c3a1_AgentInstructionBundle_ASkillStatesTheRuleAndItsReason(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	skills := aibSkillsShowingFreeTextFlags(t)
	g.Expect(skills).To(ContainElements(aibSkillsWithFreeTextExamples))

	for _, skill := range skills {
		text := shippedText(t, aibSkillPath(skill))

		g.Expect(text).To(ContainSubstring(aibAttachedFormSentence), skill)
	}
}

// Test792b32c3a2_AgentInstructionBundle_AnImprovisedCommandUsesTheAttachedForm reads the skills the repository ships
// that state the rule: the rule is stated for any free-text flag value and not for one example, no command the skill
// shows contradicts it, and the guidance read on every turn repeats it tersely.
func Test792b32c3a2_AgentInstructionBundle_AnImprovisedCommandUsesTheAttachedForm(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	for _, skill := range aibSkillsWithFreeTextExamples {
		text := shippedText(t, aibSkillPath(skill))

		g.Expect(text).To(ContainSubstring("A free-text flag value (for example"), skill)
		g.Expect(text).To(ContainSubstring("must be written as `--flag=\"value\"`"), skill)
		g.Expect(aibSpaceFormHits(text)).To(BeEmpty(), skill)
	}

	guidance := shippedText(t, "agent-instructions/guidance/shim.md")

	g.Expect(guidance).To(ContainSubstring("(free-text values: `--flag=\"value\"`, not `--flag \"value\"` — " +
		"a leading \"-\" would otherwise be misread as a flag)"))
}

// Test792b32c4a1_AgentInstructionBundle_ANewNoteIsWrittenByLearn reads the skills and the guidance the repository
// ships: a lesson is written by handing it to write-memory, which runs engram learn, and never as a plain note file.
func Test792b32c4a1_AgentInstructionBundle_ANewNoteIsWrittenByLearn(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	learn := shippedText(t, "agent-instructions/skills/learn/SKILL.md")

	g.Expect(learn).To(ContainSubstring("**REQUIRED SUB-SKILL:** invoke the **write-memory** skill with this handoff"))
	g.Expect(learn).To(ContainSubstring("write-memory composes, executes, and reports the note path."))

	writeMemory := shippedText(t, "agent-instructions/skills/write-memory/SKILL.md")

	g.Expect(writeMemory).To(ContainSubstring("composes the engram command from the provided fields, runs it"))
	g.Expect(writeMemory).To(ContainSubstring("engram learn feedback --slug"))
	g.Expect(writeMemory).To(ContainSubstring("Run the command. On success the CLI prints the written note path(s)."))

	recall := shippedText(t, "agent-instructions/skills/recall/SKILL.md")

	g.Expect(recall).To(ContainSubstring("Write sites hand off to write-memory — parents judge, the worker writes"))

	guidance := shippedText(t, "agent-instructions/guidance/learn.md")

	g.Expect(guidance).To(ContainSubstring("which writes a vault note (via `engram learn`). NOT a plain note file"))

	// The negative half: no shipped instruction has the agent create a note file itself.
	creators := make(map[string][]string)

	for _, path := range aibInstructionPaths(t) {
		hits := aibDirectNoteHits(aibNoteCreatePattern, shippedText(t, path))
		if len(hits) > 0 {
			creators[path] = hits
		}
	}

	g.Expect(creators).To(BeEmpty())
}

// Test792b32c4a2_AgentInstructionBundle_AnExistingNoteIsEditedByAmendOrResituate reads the skills the repository
// ships that change a note: curate, recall and route change it by running engram amend, and curate says that every
// curation action is an amend call.
func Test792b32c4a2_AgentInstructionBundle_AnExistingNoteIsEditedByAmendOrResituate(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	curate := shippedText(t, "agent-instructions/skills/curate/SKILL.md")

	g.Expect(curate).To(ContainSubstring(
		"Every curation action above is an `engram amend` call, composed and executed directly."))
	g.Expect(curate).To(ContainSubstring("Amend the note: `engram amend --target <basename> --situation="))

	recall := shippedText(t, "agent-instructions/skills/recall/SKILL.md")

	g.Expect(recall).To(ContainSubstring("Updates use `engram amend`; `engram learn` is create-only"))

	route := shippedText(t, "agent-instructions/skills/route/SKILL.md")

	g.Expect(route).To(ContainSubstring("composed and executed directly by you, the route-executing agent, " +
		"not handed to write-memory"))
	g.Expect(route).To(ContainSubstring("engram amend --target <matched basename, no .md>"))

	// The negative half: no shipped instruction has the agent edit a note file itself.
	editors := make(map[string][]string)

	for _, path := range aibInstructionPaths(t) {
		hits := aibDirectNoteHits(aibNoteEditPattern, shippedText(t, path))
		if len(hits) > 0 {
			editors[path] = hits
		}
	}

	g.Expect(editors).To(BeEmpty())
}

// TestAibDirectNoteHits shows the scanner of Test792b32c4a1 and Test792b32c4a2 finds an instruction to create or edit
// a note file directly, and passes the sentences that name the note file only to forbid it or to describe it.
func TestAibDirectNoteHits(t *testing.T) {
	t.Parallel()

	t.Run("an instruction to create a note file is found", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		for _, text := range []string{
			"then write the note file yourself with this content",
			"Create the note file in the vault yourself with the Write tool",
			"create the note directly",
		} {
			g.Expect(aibDirectNoteHits(aibNoteCreatePattern, text)).NotTo(BeEmpty(), text)
		}
	})

	t.Run("an instruction to edit a note file is found", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		for _, text := range []string{
			"edit the note file directly with the Edit tool instead of running engram amend",
			"Edit the note .md file directly",
			"modify the note yourself",
		} {
			g.Expect(aibDirectNoteHits(aibNoteEditPattern, text)).NotTo(BeEmpty(), text)
		}
	})

	t.Run("the shipped wording that forbids or describes a note file is clean", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		text := "NOT a plain note file, NOT Claude Code's native project memory. " +
			"a real note file, marked `pending: true`. Every curation action above is an `engram amend` call."

		g.Expect(aibDirectNoteHits(aibNoteCreatePattern, text)).To(BeEmpty())
		g.Expect(aibDirectNoteHits(aibNoteEditPattern, text)).To(BeEmpty())
	})
}

// TestAibSpaceFormHits shows the scanner of Test792b32c2a1 finds the space form of a free-text flag, in either quote,
// and passes the attached form, a flag outside the free-text list and a longer flag name.
func TestAibSpaceFormHits(t *testing.T) {
	t.Parallel()

	t.Run("the attached form is clean", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		text := `engram learn fact --situation="- the agent assumed X" --source="session notes"`

		g.Expect(aibSpaceFormHits(text)).To(BeEmpty())
	})

	t.Run("the space form is found in either quote", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		hits := aibSpaceFormHits(`engram query --phrase "a phrase" and --source 'a source'`)

		g.Expect(hits).To(HaveLen(2))
		g.Expect(hits[0]).To(HavePrefix("--phrase"))
		g.Expect(hits[1]).To(HavePrefix("--source"))
	})

	t.Run("an identifier flag and a longer flag name are clean", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)

		text := `engram learn fact --slug "my-slug" --chunk-source "1.note#top" and --sourced-from "x"`

		g.Expect(aibSpaceFormHits(text)).To(BeEmpty())
	})
}

// unexported constants.
const (
	aibAttachedFormReason = "because a value beginning with `-` is otherwise misread as a new flag and the command fails."
	aibAttachedFormRule   = "A free-text flag value (for example `--situation`, `--source`, `--subject`, `--behavior`) " +
		"must be written as `--flag=\"value\"` — the quote touching the `=`, no space —"
	// aibAttachedFormSentence is the rule and its reason as one sentence, the D18 sentence of the pytest guard.
	aibAttachedFormSentence = aibAttachedFormRule + " " + aibAttachedFormReason
	aibFrontmatterParts     = 3
	aibGuidanceDirectory    = "agent-instructions/guidance"
	aibMaxDescriptionLength = 1024
	aibSkillFile            = "SKILL.md"
	aibSkillsDirectory      = "agent-instructions/skills"
)

// unexported variables.
var (
	aibAttachedFormPattern = regexp.MustCompile(`--(?:` + strings.Join(aibFreeTextFlags, "|") + `)=`)
	// aibFreeTextFlags is the free-text flags of query, learn, amend and resituate (792b32c2), with --contributors of
	// the qa form, as the pytest guard dev/eval/test_no_space_form_free_text_flags.py lists them.
	aibFreeTextFlags = []string{
		"text", "phrase", "situation", "subject", "predicate", "object", "behavior", "impact", "action", "source",
		"body", "done-when", "red-flag", "trigger", "question", "answer", "supersedes", "contributors",
	}
	aibNoteCreatePattern = regexp.MustCompile(`(?i)\b(?:write|create|author|save)\s+(?:the|a|that|each|every)\s+` +
		`(?:new\s+)?note(?:\s+\.md)?\s+(?:file\b|yourself|directly)|\bwith\s+the\s+Write\s+tool\b`)
	aibNoteEditPattern = regexp.MustCompile(`(?i)\b(?:edit|modify|change|update|rewrite|hand-edit)\s+` +
		`(?:the|a|that|each|every)\s+note(?:\s+\.md)?\s+(?:file\b|yourself|directly)|\bwith\s+the\s+Edit\s+tool\b`)
	// aibSkillsWithFreeTextExamples is the skills whose body shows an engram command with a free-text flag.
	aibSkillsWithFreeTextExamples = []string{"curate", "recall", "route", "write-memory"}
	aibSpaceFormPattern           = regexp.MustCompile(`--(?:` + strings.Join(aibFreeTextFlags, "|") + `)\s+["']`)
)

type aibSkillMetadata struct {
	Description string `yaml:"description"`
}

// aibDirectNoteHits returns a snippet for every instruction in the text that the pattern finds.
func aibDirectNoteHits(pattern *regexp.Regexp, text string) []string {
	return pattern.FindAllString(text, -1)
}

// aibInstructionPaths returns the repository relative paths of every SKILL.md, every references markdown file of a
// skill and every guidance document, sorted.
func aibInstructionPaths(t *testing.T) []string {
	t.Helper()

	paths := make([]string, 0)

	walkErr := filepath.WalkDir(filepath.Join(repositoryRoot, aibSkillsDirectory),
		func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			relative, relErr := filepath.Rel(repositoryRoot, path)
			if relErr != nil {
				return relErr
			}

			if aibIsSkillInstruction(filepath.ToSlash(relative), entry) {
				paths = append(paths, filepath.ToSlash(relative))
			}

			return nil
		})
	if walkErr != nil {
		t.Fatalf("walking %s: %v", aibSkillsDirectory, walkErr)
	}

	guidance, readErr := os.ReadDir(filepath.Join(repositoryRoot, aibGuidanceDirectory))
	if readErr != nil {
		t.Fatalf("reading %s: %v", aibGuidanceDirectory, readErr)
	}

	for _, entry := range guidance {
		if strings.HasSuffix(entry.Name(), ".md") {
			paths = append(paths, aibGuidanceDirectory+"/"+entry.Name())
		}
	}

	sort.Strings(paths)

	return paths
}

// aibIsSkillInstruction reports whether a file under the skills directory is a SKILL.md or a markdown file of a
// references directory.
func aibIsSkillInstruction(relative string, entry fs.DirEntry) bool {
	if entry.IsDir() {
		return false
	}

	if entry.Name() == aibSkillFile {
		return true
	}

	return strings.Contains(relative, "/references/") && strings.HasSuffix(entry.Name(), ".md")
}

// aibShippedSkills returns the names of the skill directories the repository ships, sorted.
func aibShippedSkills(t *testing.T) []string {
	t.Helper()

	entries, err := os.ReadDir(filepath.Join(repositoryRoot, aibSkillsDirectory))
	if err != nil {
		t.Fatalf("reading %s: %v", aibSkillsDirectory, err)
	}

	skills := make([]string, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() {
			skills = append(skills, entry.Name())
		}
	}

	return skills
}

// aibSkillDescription returns the description of a skill, parsed from the YAML frontmatter of its SKILL.md.
func aibSkillDescription(t *testing.T, skill string) string {
	t.Helper()

	content, err := os.ReadFile(filepath.Join(repositoryRoot, aibSkillPath(skill)))
	if err != nil {
		t.Fatalf("reading the SKILL.md of %s: %v", skill, err)
	}

	parts := strings.SplitN(string(content), "---", aibFrontmatterParts)
	if len(parts) != aibFrontmatterParts {
		t.Fatalf("the SKILL.md of %s has no frontmatter block", skill)
	}

	var metadata aibSkillMetadata

	err = yaml.Unmarshal([]byte(parts[1]), &metadata)
	if err != nil {
		t.Fatalf("parsing the frontmatter of %s: %v", skill, err)
	}

	return metadata.Description
}

// aibSkillPath returns the repository relative path of the SKILL.md of a skill.
func aibSkillPath(skill string) string {
	return aibSkillsDirectory + "/" + skill + "/" + aibSkillFile
}

// aibSkillsShowingFreeTextFlags returns the skills whose SKILL.md shows an engram command with a free-text flag.
func aibSkillsShowingFreeTextFlags(t *testing.T) []string {
	t.Helper()

	showing := make([]string, 0)

	for _, skill := range aibShippedSkills(t) {
		if aibAttachedFormPattern.MatchString(shippedText(t, aibSkillPath(skill))) {
			showing = append(showing, skill)
		}
	}

	return showing
}

// aibSpaceFormHits returns a snippet for every free-text flag written as --flag "value" in the text.
func aibSpaceFormHits(text string) []string {
	matches := aibSpaceFormPattern.FindAllStringIndex(text, -1)
	hits := make([]string, 0, len(matches))

	for _, match := range matches {
		hits = append(hits, text[match[0]:match[1]])
	}

	return hits
}
