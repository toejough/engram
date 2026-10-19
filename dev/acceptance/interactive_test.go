//go:build network

package acceptance_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The replay rows drive the installed binary with the record of an interactive session and no model: the rows of an
// interactive record are stamped human, which a headless session's record is not (855d4g1e, 855d4g1g).

func Test855d4g1e_AnInteractiveSessionsTypedTurnIsJudged(t *testing.T) {
	t.Parallel()

	const typed = "No - build with make zorblax-build, never make all."

	machine := replayMachine(t)
	input := `{"origin":"composer","session":"interactive","turn":"t2","text":"` + typed +
		`","answer":"Built with make zorblax-build."`

	moment := machine.runWithInput(machine.work, input+`,"cue":"turn-end"}`, nil,
		filepath.Join(machine.gobin, engramBinaryName), "auto", "moment")
	if moment.exit != 0 || !strings.Contains(moment.stdout, typed) ||
		strings.Contains(moment.stdout, "skip the review") {
		t.Fatalf("auto moment (exit %d) holds no typed text: %q, %q", moment.exit, moment.stdout, moment.stderr)
	}

	judged := machine.runWithInput(machine.work, input+`,"reply":"{\"learn\":true}"}`, nil,
		filepath.Join(machine.gobin, engramBinaryName), "auto", "judge")
	if judged.exit != 0 {
		t.Fatalf("auto judge exited with %d: %q", judged.exit, judged.stderr)
	}

	var step struct {
		Request struct {
			Prompt string `json:"prompt"`
		} `json:"request"`
	}

	if err := json.Unmarshal([]byte(judged.stdout), &step); err != nil {
		t.Fatalf("auto judge printed no request: %q (%v)", judged.stdout, err)
	}

	prompt := step.Request.Prompt

	if !strings.Contains(prompt, "User: "+typed) || strings.Contains(prompt, "Background task") ||
		strings.Contains(prompt, "skip the review") {
		t.Fatalf("the window of the judgement is not the typed turn: %q", prompt)
	}
}

func Test855d4g1g_ATurnFromAnOlderModIsSkippedAndIsNoFailure(t *testing.T) {
	t.Parallel()

	machine := replayMachine(t)

	// the old mod hands over no reply
	judged := machine.runWithInput(machine.work,
		`{"origin":"composer","session":"interactive","turn":"t2","answer":"Built."}`, nil,
		filepath.Join(machine.gobin, engramBinaryName), "auto", "judge")
	if judged.exit != 0 || judged.stdout != "" {
		t.Fatalf("auto judge (exit %d) printed %q, %q", judged.exit, judged.stdout, judged.stderr)
	}

	skipped, err := filepath.Glob(filepath.Join(machine.home, ".local", "share", "engram", "auto", "interactive",
		"*-skipped.json"))
	if err != nil || len(skipped) != 1 {
		t.Fatalf("the records of the turn: %v (%v)", skipped, err)
	}
}

// replayMachine is a machine whose vault exists and whose transcripts hold the record of an interactive session.
func replayMachine(t *testing.T) *machine {
	t.Helper()

	machine := newMachine(t)
	machine.hasEngram()
	mkdirAll(t, machine.vault)

	record, err := os.ReadFile(filepath.Join("..", "..", "internal", "cli", "testdata", "interactive_session.jsonl"))
	if err != nil {
		t.Fatalf("reading the record of the interactive session: %v", err)
	}

	project := filepath.Join(machine.transcripts, "-home-joe-zorblax")
	mkdirAll(t, project)
	writeFile(t, filepath.Join(project, "interactive.jsonl"), string(record))

	return machine
}
