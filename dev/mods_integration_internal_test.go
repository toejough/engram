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
