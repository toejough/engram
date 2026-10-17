//go:build targ && integration

package dev

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

// Test793d2a1g_Mod_ASessionThatDoesNotGiveItsIDInTimeDoesNotHoldThePrompt runs the mod under node against a stand-in
// host: a live session cannot be made to stall in the call for its ID. The stand-in's timer fires at once and records
// the delay it was asked for, and its call for the ID never answers.
func Test793d2a1g_Mod_ASessionThatDoesNotGiveItsIDInTimeDoesNotHoldThePrompt(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	node, findErr := findNode()
	g.Expect(findErr).NotTo(HaveOccurred())

	if findErr != nil {
		return
	}

	hooks := filepath.Join(repositoryRoot, "agent-instructions", "mods", "engram-memory", "hooks")
	source, readErr := os.ReadFile(filepath.Join(hooks, "register.js"))
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	// a module under a name that node reads as one, whatever release of node it is
	module := filepath.Join(t.TempDir(), "register.mjs")
	g.Expect(os.WriteFile(module, source, 0o600)).To(Succeed())

	ctx, cancel := context.WithTimeout(t.Context(), driverTimeout)
	defer cancel()

	command := exec.CommandContext(ctx, node, "--input-type=module", "--eval", hostDriver)
	command.Env = append(os.Environ(), "MOD="+module)

	out, runErr := command.Output()
	if runErr != nil {
		t.Fatalf("the mod did not let the prompt through (%v): %s", runErr, out)
	}

	var outcome struct {
		Passed struct {
			Text    string   `json:"text"`
			Context []string `json:"context"`
		} `json:"passed"`
		Delays []int `json:"delays"`
		Ran    int   `json:"ran"`
	}

	g.Expect(json.Unmarshal(out, &outcome)).To(Succeed(), "the driver printed %q", out)
	g.Expect(outcome.Passed.Text).To(Equal("hello"), "the prompt goes on as it was")
	g.Expect(outcome.Passed.Context).To(BeEmpty(), "nothing is added")
	g.Expect(outcome.Delays).To(Equal([]int{7000}), "the one timer the mod asked for, in milliseconds")
	g.Expect(outcome.Ran).To(BeZero(), "the command was run")
}

// Test793d2b1a_Mod_JudgesATurnAndGivesTheReplyToTheWriting runs the mod under node against a stand-in host: a live
// session cannot show what a stand-in records. The stand-in's judgement does not answer until the driver lets it, so a
// hook that waited for the job would not give the event's result back.
func Test793d2b1a_Mod_JudgesATurnAndGivesTheReplyToTheWriting(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	outcome := runCaptureDriver(t, captureDriver)

	g.Expect(outcome.Returned.Text).To(Equal("Understood."), "the event's result is given back")
	g.Expect(outcome.Held).To(BeFalse(), "the event's result was not given back while the judgement was running")
	g.Expect(outcome.Calls).To(HaveLen(3), "the judgement, the model and the writing, in that order")

	if len(outcome.Calls) != 3 {
		return
	}

	g.Expect(outcome.Calls[0].Step).To(Equal("engram auto judge"))
	g.Expect(outcome.Calls[0].Input).To(Equal(map[string]any{"session": "a-session", "answer": "Understood."}))
	g.Expect(outcome.Calls[1].Step).To(Equal("model"))
	g.Expect(outcome.Calls[1].Asked).To(Equal(map[string]any{
		"model": "haiku", "effort": "low", "prompt": "THE PROMPT", "maxTokens": float64(1500), "timeoutMs": float64(60000),
	}), "the request, exactly as the judgement printed it")
	g.Expect(outcome.Calls[2].Step).To(Equal("engram auto write"))
	g.Expect(outcome.Calls[2].Input).To(Equal(map[string]any{"session": "a-session", "reply": "THE REPLY"}))
}

// TestMods_EveryJavaScriptModuleParses: the real mods parse under the real node, and the real node rejects a broken
// module, which the stand-in node of the unit tests cannot show (793b55).
func TestMods_EveryJavaScriptModuleParses(t *testing.T) {
	t.Parallel()

	t.Run("the mods of the source", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)
		modules := modulesOf(t, filepath.Join(repositoryRoot, "agent-instructions", "mods"))

		g.Expect(modules).NotTo(BeEmpty(), "the JavaScript of the mods")
		g.Expect(checkModuleSyntax(t.Context(), modules, findNode, parseModule)).To(Succeed())
	})

	t.Run("a broken module", func(t *testing.T) {
		t.Parallel()

		g := NewWithT(t)
		err := checkModuleSyntax(t.Context(),
			map[string]string{"hooks/broken.js": "export function register( {\n"}, findNode, parseModule)

		g.Expect(err).To(MatchError(ContainSubstring("hooks/broken.js")))
		g.Expect(err).To(MatchError(ContainSubstring("SyntaxError")))
	})
}

// unexported constants.
const (
	// captureDriver runs the turn.complete hook of the module named by MOD against a host whose judgement prints a
	// request once the driver lets it, whose model answers THE REPLY, and which records the order and the input of
	// what the module asks of it. It prints what the hook returned, whether the hook was still waiting when the
	// driver gave up on it, and the calls.
	captureDriver = `
import { pathToFileURL } from 'node:url'

const { register } = await import(pathToFileURL(process.env.MOD).href)
const handlers = {}
register((name, handler) => { handlers[name] = handler })

const request = { model: 'haiku', effort: 'low', prompt: 'THE PROMPT', maxTokens: 1500, timeoutMs: 60000 }
const calls = []
let letJudgementAnswer
const judgement = new Promise((resolve) => { letJudgementAnswer = resolve })

const host = {
  session: { id: async () => 'a-session' },
  process: {
    run: async (argv, init) => {
      calls.push({ step: argv.join(' '), input: JSON.parse(init.stdin) })
      return argv[2] === 'judge' ? judgement : { exitCode: 0, stdout: '' }
    },
  },
  model: {
    complete: async (asked) => {
      calls.push({ step: 'model', asked })
      return { isAnswered: true, text: 'THE REPLY' }
    },
  },
  ui: { log: () => {} },
}

const waited = Symbol('waited')
const hook = handlers['turn.complete'](host, { answer: 'Understood.', turnId: 't1', reason: 'answer' },
  async (event) => ({ text: event.answer }))
const returned = await Promise.race([hook, new Promise((resolve) => setTimeout(() => resolve(waited), 500))])
const held = returned === waited

letJudgementAnswer({ exitCode: 0, stdout: JSON.stringify(request) + '\n' })

for (let tick = 0; tick < 200 && calls.length < 3; tick++) {
  await new Promise((resolve) => setTimeout(resolve, 10))
}

console.log(JSON.stringify({ returned: held ? {} : returned, held, calls }))
`
	// driverTimeout is how long the mod may take under the stand-in host before the test gives it up.
	driverTimeout = 20 * time.Second
	// hostDriver runs the prompt.submit hook of the module named by MOD against a host whose call for the session's ID
	// never answers and whose timers fire at once, and prints what the hook let through, the delays the timers were
	// asked for and how many commands the mod ran.
	hostDriver = `
import { pathToFileURL } from 'node:url'

const { register } = await import(pathToFileURL(process.env.MOD).href)
const handlers = {}
register((name, handler) => { handlers[name] = handler })

const delays = []
globalThis.setTimeout = (callback, delay) => { delays.push(delay); callback(); return 0 }

const ran = []
const host = {
  session: { id: () => new Promise(() => {}) },
  process: { run: async (...args) => { ran.push(args); return { exitCode: 0, stdout: '' } } },
}

let passed = null
await handlers['prompt.submit'](host, { text: 'hello', context: [] }, async (event) => { passed = event })
console.log(JSON.stringify({ passed, delays, ran: ran.length }))
`
)

// captureOutcome is what captureDriver printed.
type captureOutcome struct {
	Returned struct {
		Text string `json:"text"`
	} `json:"returned"`
	Held  bool `json:"held"`
	Calls []struct {
		Step  string         `json:"step"`
		Input map[string]any `json:"input"`
		Asked map[string]any `json:"asked"`
	} `json:"calls"`
}

// modulesOf reads every JavaScript file below a directory, by its path relative to the directory.
func modulesOf(t *testing.T, dir string) map[string]string {
	t.Helper()

	modules := map[string]string{}

	walkErr := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".js") {
			return err
		}

		source, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}

		relative, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return relErr
		}

		modules[relative] = string(source)

		return nil
	})
	if walkErr != nil {
		t.Fatalf("reading the JavaScript of %s: %v", dir, walkErr)
	}

	return modules
}

// runCaptureDriver runs a driver under node against the mod of the source, and reads what it printed.
func runCaptureDriver(t *testing.T, driver string) captureOutcome {
	t.Helper()

	node, findErr := findNode()
	if findErr != nil {
		t.Fatalf("finding node: %v", findErr)
	}

	source, readErr := os.ReadFile(filepath.Join(repositoryRoot, "agent-instructions", "mods", "engram-memory", "hooks",
		"register.js"))
	if readErr != nil {
		t.Fatalf("reading the mod: %v", readErr)
	}

	// a module under a name that node reads as one, whatever release of node it is
	module := filepath.Join(t.TempDir(), "register.mjs")

	writeErr := os.WriteFile(module, source, 0o600)
	if writeErr != nil {
		t.Fatalf("writing the mod: %v", writeErr)
	}

	ctx, cancel := context.WithTimeout(t.Context(), driverTimeout)
	defer cancel()

	command := exec.CommandContext(ctx, node, "--input-type=module", "--eval", driver)
	command.Env = append(os.Environ(), "MOD="+module)

	out, runErr := command.CombinedOutput()
	if runErr != nil {
		t.Fatalf("the mod did not run under the stand-in host (%v): %s", runErr, out)
	}

	var outcome captureOutcome

	unmarshalErr := json.Unmarshal(out, &outcome)
	if unmarshalErr != nil {
		t.Fatalf("the driver printed %q: %v", out, unmarshalErr)
	}

	return outcome
}
