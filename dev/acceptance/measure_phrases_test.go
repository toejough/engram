//go:build network

package acceptance_test

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

func Test855d7a1g_Hyp_ThePhrasesFindWhatTheRecallSkillFinds(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	machine := newMachine(t)

	step(t, "Given a real Claude Code session, a vault, and 10 prompts for each of which the recall skill run in "+
		"glance mode names the notes it draws on", func() {
		machine.hasMeasuredVault("echo")
	})

	var arrived []promptRecall

	step(t, "When each prompt is sent with automatic memory on", func() {
		for index, prompt := range skillPrompts {
			machine.startSession(fmt.Sprintf("with-%d", index)).send(prompt)
		}

		arrived = machine.promptRecalls(skillPrompts)
	})

	step(t, "Then the share of those notes that are among the candidates is recorded, with the share the skill's own "+
		"phrases reach", func() {
		machine.automaticMemoryIsOff()
		machine.agentMayRunEngramAndSkills()

		rows := make([][]string, 0, len(skillPrompts))
		drawn, amongCandidates, amongSkillsOwn := 0, 0, 0

		for index, prompt := range skillPrompts {
			skilled := machine.startSession(fmt.Sprintf("skill-%d", index)).send(skillAsk(prompt))
			named := namedNotes(skilled.reply, machine.noteNames())
			phrases := phrasesOfTurn(skilled)
			reached := machine.lookupWith(prompt, phrases)

			drawn += len(named)
			amongCandidates += len(overlap(named, arrived[index].candidates))
			amongSkillsOwn += len(overlap(named, reached))

			rows = append(rows, []string{
				fmt.Sprintf("%d", index+1), fmt.Sprintf("%d", len(named)), arrived[index].depth,
				fmt.Sprintf("%d", len(arrived[index].candidates)), fmt.Sprintf("%d", len(overlap(named, arrived[index].candidates))),
				fmt.Sprintf("%d", len(phrases)), fmt.Sprintf("%d", len(reached)), fmt.Sprintf("%d", len(overlap(named, reached))),
			})
		}

		t.Logf("the notes the recall skill drew on, against the candidates of the chain and of the lookup with the skill's "+
			"own phrases:\n%s", table([]string{"prompt", "drawn on", "depth", "candidates", "drawn among them", "skill phrases",
			"skill's lookup", "drawn among them"}, rows))
		t.Logf("the notes drawn on: %d; among the candidates of the chain: %d (%.2f); among the lookup of the skill's own "+
			"phrases: %d (%.2f)", drawn, amongCandidates, ratio(amongCandidates, drawn), amongSkillsOwn,
			ratio(amongSkillsOwn, drawn))

		// refuted when the share among the candidates is lower than the share the skill's phrases reach: it was, and
		// the numbers are in 855b109, so the row keeps a record-only assertion
		t.Logf("the hypothesis is refuted when %.2f < %.2f: %v (855b109)", ratio(amongCandidates, drawn),
			ratio(amongSkillsOwn, drawn), ratio(amongCandidates, drawn) < ratio(amongSkillsOwn, drawn))

		g.Expect(drawn).To(BeNumerically(">", 0), "the notes the skill drew on")
	})
}

// skillAsk asks, in a session without automatic memory, for the recall skill at its glance mode for a prompt, and for
// the notes it drew on by their file names.
func skillAsk(prompt string) string {
	return "Run the recall skill in its glance mode for this prompt: \"" + prompt + "\" " +
		"Then reply with the file names of the notes the skill drew on, one to a line, and nothing else."
}

// unexported variables.
var (
	// skillPrompts are 10 prompts that notes of the vault apply to (855d7a1g).
	skillPrompts = []string{
		"Plan a migration of the Zorblax staging database for tonight, around whatever is scheduled on that database.",
		"Prepare a test run against the Zorblax staging database tonight.",
		"Plan how to restore the Zorblax staging database from last night's backup.",
		"Should the Zorblax staging test run go ahead on a public holiday?",
		"Cut release v2.0.0 for the Zorblax project and tell me when it is done.",
		"Add an index on orders.created_at to the schema.",
		"Why would make ci fail here with a LINT_STRICT message?",
		"I am about to commit trailing.py after editing it. What should I do first?",
		"When is the Zorblax staging database reset?",
		"Plan a backup schedule around the resets of the Zorblax staging database.",
	}
)

// promptRecall is what the chain of a prompt looked up: the depth chosen and the notes among its candidates.
type promptRecall struct {
	depth      string
	candidates []string
}

// candidateNotes are the file names of the notes among the candidates that a recall step printed.
func candidateNotes(output string) []string {
	var printed struct {
		Carry struct {
			Candidates []struct {
				ID   string `json:"id"`
				Kind string `json:"kind"`
			} `json:"candidates"`
		} `json:"carry"`
	}

	if json.Unmarshal([]byte(output), &printed) != nil {
		return nil
	}

	var notes []string

	for _, candidate := range printed.Carry.Candidates {
		if candidate.Kind == "note" {
			notes = append(notes, candidate.ID)
		}
	}

	return notes
}

// namedNotes are the notes of the vault, by file name, that a reply names.
func namedNotes(reply string, vault []string) []string {
	var named []string

	for _, name := range vault {
		if strings.Contains(reply, name) {
			named = append(named, name)
		}
	}

	return named
}

// phrasesOfTurn are the phrases of the engram queries that the agent ran in the turn.
func phrasesOfTurn(sent *turn) []string {
	var phrases []string

	for _, row := range sent.rows {
		for _, command := range row.commands() {
			if strings.Contains(command, "engram query") {
				phrases = append(phrases, queryPhrases(command)...)
			}
		}
	}

	return phrases
}

// queryPhrases are the phrases of the engram queries that a shell command runs, each written `--phrase text` or
// `--phrase=text`.
func queryPhrases(command string) []string {
	var phrases []string

	words := shellWords(command)

	for index, word := range words {
		switch {
		case word == "--phrase" && index+1 < len(words):
			phrases = append(phrases, words[index+1])
		case strings.HasPrefix(word, "--phrase="):
			phrases = append(phrases, strings.TrimPrefix(word, "--phrase="))
		}
	}

	return phrases
}

// ratio is a share, or zero when there is nothing to share.
func ratio(part, whole int) float64 {
	if whole == 0 {
		return 0
	}

	return float64(part) / float64(whole)
}

// overlap are the items of the first list that the second holds.
func overlap(drawn, found []string) []string {
	return slices.DeleteFunc(slices.Clone(drawn), func(item string) bool { return !slices.Contains(found, item) })
}

// shellWords splits a command line into words as a shell does for quotes and backslashes: no expansion is done.
func shellWords(text string) []string {
	var (
		words   []string
		word    strings.Builder
		quote   rune
		started bool
		escaped bool
	)

	for _, char := range text {
		switch {
		case escaped:
			word.WriteRune(char)

			escaped = false
		case char == '\\' && quote != '\'':
			escaped, started = true, true
		case quote != 0 && char == quote:
			quote = 0
		case quote != 0:
			word.WriteRune(char)
		case char == '\'' || char == '"':
			quote, started = char, true
		case char == ' ' || char == '\t' || char == '\n':
			if started {
				words = append(words, word.String())
				word.Reset()

				started = false
			}
		default:
			word.WriteRune(char)

			started = true
		}
	}

	if started {
		words = append(words, word.String())
	}

	return words
}

// lookupWith runs the lookup of the recall step for a prompt with the phrases given, as it is given the phrases of the
// phrases step, and returns the notes among its candidates.
func (m *machine) lookupWith(prompt string, phrases []string) []string {
	m.t.Helper()

	reply, err := json.Marshal(map[string][]string{"phrases": phrases})
	if err != nil {
		m.t.Fatalf("encoding the phrases: %v", err)
	}

	input, err := json.Marshal(map[string]any{
		"session": "measure", "origin": "sdk", "cue": "prompt", "held": []string{}, "reply": string(reply),
		"carry": map[string]string{"cue": "prompt", "depth": "glance", "window": "User: " + prompt},
	})
	if err != nil {
		m.t.Fatalf("encoding the input of the recall step: %v", err)
	}

	out := m.runWithInput(m.work, string(input), nil, filepath.Join(m.gobin, engramBinaryName), "auto", "recall")
	m.mustSucceed(out, "engram auto recall")

	return candidateNotes(out.stdout)
}

// promptRecalls are, for each prompt, what its chain of the delivery flow looked up.
func (m *machine) promptRecalls(prompts []string) []promptRecall {
	recalls := make([]promptRecall, len(prompts))
	chains := chainsOfCue(m.chains(), "prompt")

	for index, prompt := range prompts {
		for _, each := range chains {
			if each.steps[0].field("text") != prompt {
				continue
			}

			recalls[index].depth = each.depth()

			for _, run := range each.steps {
				if run.step == stepRecall {
					recalls[index].candidates = candidateNotes(run.output)
				}
			}
		}
	}

	return recalls
}
