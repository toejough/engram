package cli_test

// Scenario tests of the agent-instruction-bundle technical component (792b32 Agent Instruction Bundle), the two
// scenarios of requirement 792b32c2 (Free-text flag values use the attached form) that the binary itself
// answers: a template followed literally with a value that begins with "-" succeeds (792b32c2a2), and the
// instruction the binary emits to an agent writes its free-text values attached (792b32c2a3). The shipped
// SKILL.md and guidance text of 792b32c2a1 is not Go-testable here and is not covered by this file.
//
// The commands of 792b32c2a2 run through the real command wiring (cli.Targets and its flag parsing) over the
// in-memory vault of note-revision part 1 (nr1Rig, a complete cli.EdgeFS): no temporary directory, no real home,
// no git, no network.

import (
	"bytes"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/targ"

	"github.com/toejough/engram/internal/cli"
	"github.com/toejough/engram/internal/update"
)

// Test792b32c2a2_AgentInstructionBundle_AValueBeginningWithADashSucceeds verifies 792b32c2a2: a command that is
// a shipped template followed literally, with every free-text flag attached (--flag=value) to a value that
// begins with "-", exits 0 and writes the value, for each command and flag the rule names. The control case
// shows the Given is a real hazard: the space-separated form of the same value is refused by the parser.
func Test792b32c2a2_AgentInstructionBundle_AValueBeginningWithADashSucceeds(t *testing.T) {
	t.Parallel()

	for name, scenario := range aib1DashCases() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			rig := nr1NewRig(t, nr1Alice)
			rig.plantChunkIndex("a#1")

			for _, given := range scenario.given {
				stdout, stderr := aib1Run(rig, given...)

				g.Expect(stderr).To(BeEmpty(), "the Given note is made by %v", given)
				g.Expect(stdout).NotTo(BeEmpty(), "the Given note is made by %v", given)
			}

			stdout, stderr := aib1Run(rig, scenario.command...)

			g.Expect(stderr).To(BeEmpty(), "the command exits 0")
			g.Expect(stdout).NotTo(BeEmpty())

			written := stdout

			switch scenario.note {
			case "":
			case aib1AllNotes:
				written = aib1AllText(rig)
			default:
				written = rig.note(scenario.note)
			}

			for _, value := range scenario.values {
				g.Expect(written).To(ContainSubstring(value), "the value reached its flag whole")
			}
		})
	}

	t.Run("the space-separated form is refused", func(t *testing.T) {
		t.Parallel()
		g := NewWithT(t)

		rig := nr1NewRig(t, nr1Alice)

		stdout, stderr := aib1Run(rig, "learn", "fact", "--slug", "a", "--position", "top", "--source", "test",
			"--situation", aib1Value("situation"))

		g.Expect(stderr).To(ContainSubstring("needs an argument"))
		g.Expect(stdout).To(BeEmpty())
		g.Expect(rig.fs.files).To(BeEmpty(), "no note was written")
	})
}

// Test792b32c2a3_AgentInstructionBundle_ABinaryEmittedInstructionUsesTheAttachedForm verifies 792b32c2a3: every
// string the binary emits that tells an agent to run an engram command with a free-text value (the curate
// instruction, carried by the pending_offers_hint of a query payload, by the pending-offer notice of
// `engram update` and by the write-path nudge) writes each value attached, --flag="value", with the quote
// touching the "=", and none in the space-separated form.
func Test792b32c2a3_AgentInstructionBundle_ABinaryEmittedInstructionUsesTheAttachedForm(t *testing.T) {
	t.Parallel()

	const (
		wantText   = `--text="curate pending offers"`
		wantPhrase = `--phrase="reviewing pending offers in a vault and judging each against existing notes"`
	)

	for name, emit := range aib1Emitters() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)

			emitted := emit(t)

			g.Expect(emitted).To(ContainSubstring("run `engram query "),
				"the Given: the string tells the agent to run an engram command")
			g.Expect(emitted).To(ContainSubstring(wantText), "the --text value is attached")
			g.Expect(emitted).To(ContainSubstring(wantPhrase), "the --phrase value is attached")

			for _, flag := range aib1FreeTextFlags() {
				g.Expect(emitted).NotTo(MatchRegexp(aib1SpaceForm(flag)), "--%s is not written space-separated", flag)
				g.Expect(emitted).NotTo(MatchRegexp(aib1BareForm(flag)), "--%s is always attached with a quote", flag)
			}
		})
	}
}

// unexported constants.
const (
	aib1AllNotes = "*"
	aib1Note     = "1.2026-10-09.a"
)

// aib1DashCase is one command of 792b32c2a2: the commands that make its Given, the command under test, the note
// it writes ("" when it prints its result), and the values that must arrive whole.
type aib1DashCase struct {
	given   [][]string
	command []string
	note    string
	values  []string
}

// aib1AllText returns the text of every file in the rig's vault, in the order of their names.
func aib1AllText(rig *nr1Rig) string {
	rig.t.Helper()

	names := slices.Sorted(maps.Keys(rig.fs.files))
	texts := make([]string, 0, len(names))

	for _, name := range names {
		texts = append(texts, string(rig.fs.files[name]))
	}

	return strings.Join(texts, "\n")
}

// aib1Attached returns the arguments --flag=value for each flag, every value beginning with "-".
func aib1Attached(flags ...string) []string {
	args := make([]string, 0, len(flags))

	for _, flag := range flags {
		args = append(args, "--"+flag+"="+aib1Value(flag))
	}

	return args
}

// aib1BareForm matches --flag="value" in a string with the quote not touching the "=", or the "=" absent.
func aib1BareForm(flag string) string {
	return `--` + regexp.QuoteMeta(flag) + `(?:=[^"]|[^=a-z-])`
}

// aib1Command returns a command line: the words, then each flag attached.
func aib1Command(words []string, flags ...string) []string {
	return append(append([]string{}, words...), aib1Attached(flags...)...)
}

// aib1DashCases returns the commands of 792b32c2a2, one per command and flag group the rule names. The Given
// notes carry plain values (no leading dash) that differ from the dash-led values of the command, so a value
// found afterwards was written by the command under test.
func aib1DashCases() map[string]aib1DashCase {
	learnFact := []string{"learn", "fact", "--slug", "a", "--position", "top"}
	learnFeedback := []string{"learn", "feedback", "--slug", "a", "--position", "top"}
	learnRunbook := []string{"learn", "runbook", "--slug", "a", "--position", "top"}
	amend := []string{"amend", "--target", aib1Note, "--chunks-dir", nr1Chunks}

	giveFact := aib1PlainCommand(learnFact, "source", "situation")
	giveFeedback := aib1PlainCommand(learnFeedback, "source", "situation")
	giveRunbook := aib1PlainCommand(learnRunbook, "source", "situation", "done-when")

	return map[string]aib1DashCase{
		"learn fact": {
			command: aib1Command(learnFact, "source", "situation", "subject", "predicate", "object"),
			note:    aib1Note,
			values:  aib1Values("source", "situation", "subject", "predicate", "object"),
		},
		"learn feedback": {
			command: aib1Command(learnFeedback, "source", "situation", "behavior", "impact", "action"),
			note:    aib1Note,
			values:  aib1Values("source", "situation", "behavior", "impact", "action"),
		},
		"learn runbook": {
			command: aib1Command(learnRunbook, "source", "situation", "done-when", "body", "trigger", "red-flag"),
			note:    aib1Note,
			values:  aib1Values("source", "situation", "done-when", "body", "trigger", "red-flag"),
		},
		"learn qa": {
			command: aib1Command([]string{"learn", "qa", "--slug", "a"}, "source", "question", "answer"),
			note:    aib1AllNotes,
			values:  aib1Values("source", "question", "answer"),
		},
		"learn with a claim": {
			given: [][]string{giveFact},
			command: append(aib1Command([]string{"learn", "fact", "--slug", "b", "--position", "top"},
				"source", "situation"), "--supersedes="+aib1Note+"|updates|"+aib1Value("claim")),
			note:   "2.2026-10-09.b",
			values: []string{aib1Value("claim")},
		},
		"amend fact": {
			given:   [][]string{giveFact},
			command: aib1Command(amend, "situation", "subject", "predicate", "object"),
			note:    aib1Note,
			values:  aib1Values("situation", "subject", "predicate", "object"),
		},
		"amend feedback": {
			given:   [][]string{giveFeedback},
			command: aib1Command(amend, "behavior", "impact", "action"),
			note:    aib1Note,
			values:  aib1Values("behavior", "impact", "action"),
		},
		"amend runbook": {
			given:   [][]string{giveRunbook},
			command: aib1Command(amend, "done-when", "body", "trigger", "red-flag"),
			note:    aib1Note,
			values:  aib1Values("done-when", "body", "trigger", "red-flag"),
		},
		"resituate": {
			given:   [][]string{giveFact},
			command: aib1Command([]string{"resituate", "--note", aib1Note}, "situation"),
			note:    aib1Note,
			values:  aib1Values("situation"),
		},
		"query": {
			given:   [][]string{giveFact},
			command: aib1Command([]string{"query", "--chunks-dir", nr1Chunks}, "text", "phrase"),
		},
	}
}

// aib1Emitters returns the strings the binary emits that carry the curate instruction, by where they are emitted.
func aib1Emitters() map[string]func(t *testing.T) string {
	return map[string]func(t *testing.T) string{
		"the shared instruction": func(*testing.T) string {
			return cli.ExportPendingOfferCurateInstruction
		},
		"the query payload hint": func(t *testing.T) string {
			t.Helper()

			payload, err := cli.ExportRenderQueryPayloadPendingOffers(true)
			if err != nil {
				t.Fatalf("rendering the query payload: %v", err)
			}

			return payload
		},
		"the update notice": func(t *testing.T) string {
			t.Helper()

			var buffer bytes.Buffer

			err := cli.ExportWriteUpdateReport(&buffer, update.Report{VaultHasPendingOffers: true})
			if err != nil {
				t.Fatalf("writing the update report: %v", err)
			}

			return buffer.String()
		},
		"the write-path nudge": func(t *testing.T) string {
			t.Helper()

			var warnings []string

			cli.ExportWarnIfPendingOffers("/vault",
				func(string) ([]string, error) { return []string{"1.2026-01-01.a.md"}, nil },
				func(string) ([]byte, error) { return []byte(pendingFactNote), nil },
				func(format string, args ...any) { warnings = append(warnings, fmt.Sprintf(format, args...)) })

			return strings.Join(warnings, "\n")
		},
	}
}

// aib1FreeTextFlags returns the free-text flags of query, learn, amend and resituate that 792b32c2 names.
func aib1FreeTextFlags() []string {
	return []string{
		"phrase", "text", "situation", "subject", "predicate", "object", "behavior", "impact", "action",
		"done-when", "body", "question", "answer", "source", "supersedes", "trigger", "red-flag",
	}
}

// aib1PlainCommand returns a command line: the words, then each flag attached to a plain value (no leading dash).
func aib1PlainCommand(words []string, flags ...string) []string {
	args := append([]string{}, words...)

	for _, flag := range flags {
		args = append(args, "--"+flag+"=the given "+flag)
	}

	return args
}

// aib1Run runs `engram <args> --vault <vault>` over the rig's configuration as rig.run does, returning stdout and
// everything the command says on failure, including what targ's parser printed, which rig.run does not keep.
func aib1Run(rig *nr1Rig, args ...string) (string, string) {
	rig.t.Helper()

	var stdout, stderr bytes.Buffer

	deps := newTestDeps(&stdout, &stderr)
	rig.configure(&deps)

	argv := append(append([]string{"engram"}, args...), "--vault", nr1Vault)

	result, err := targ.Execute(argv, cli.Targets(deps)...)
	if err != nil {
		return stdout.String(), result.Output + stderr.String() + err.Error()
	}

	return stdout.String(), result.Output + stderr.String()
}

// aib1SpaceForm matches --flag followed by whitespace and a quote.
func aib1SpaceForm(flag string) string {
	return `--` + regexp.QuoteMeta(flag) + `[ \t]+["']`
}

// aib1Value returns the value of a flag in the scenarios: free text that begins with a dash.
func aib1Value(flag string) string {
	return "- the " + flag + " begins with a dash"
}

// aib1Values returns the value of each flag.
func aib1Values(flags ...string) []string {
	values := make([]string, 0, len(flags))

	for _, flag := range flags {
		values = append(values, aib1Value(flag))
	}

	return values
}
