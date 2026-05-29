//go:build targ

package eval_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/imptest/match"

	"github.com/toejough/engram/dev/eval"
)

func TestRun_Orchestrates_CurrentState(t *testing.T) {
	// imptest interaction tests must not be parallel — shared Imp state per test.
	g := NewWithT(t)

	resultJSON, err := os.ReadFile("testdata/result.json")
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	sessionJSONL, err := os.ReadFile("testdata/session.jsonl")
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	cloner, clonerImp := MockVaultCloner(t)
	builder, builderImp := MockConfigBuilder(t)
	runner, runnerImp := MockAgentRunner(t)
	writer, writerImp := MockResultsWriter(t)

	deps := eval.Deps{
		Cloner:  cloner,
		Config:  builder,
		Runner:  runner,
		Results: writer,
	}

	cfg := eval.RunConfig{
		Trials:   1,
		Model:    "haiku",
		VaultSrc: "/test/vault",
		OutDir:   "/tmp/test-eval",
	}

	done := make(chan struct{})

	go func() {
		defer close(done)

		runErr := eval.Run(context.Background(), "current-state", cfg, deps)
		g.Expect(runErr).NotTo(HaveOccurred())
	}()

	// Config.Build called once with the current-state arm.
	arm, ok := eval.LookupArm("current-state")
	g.Expect(ok).To(BeTrue())

	builderImp.Build.ArgsShould(match.BeAny, arm, match.BeAny).Return("/test/configdir", "/test/prefix", nil)

	// For each of the 3 scenarios × 1 trial: Clone → Runner.Run → Results.Append.
	scenarios := eval.Scenarios()
	for _, scenario := range scenarios {
		capturedScenario := scenario

		// Clone called with the VaultSrc and some dest dir.
		clonerImp.Clone.ArgsShould(match.BeAny, "/test/vault", match.BeAny).Return(nil)

		// Runner.Run called with the scenario's prompt.
		runnerImp.Run.ArgsShould(
			match.BeAny,
			match.Satisfy(func(inv eval.AgentInvocation) error {
				if inv.Prompt != capturedScenario.Prompt {
					return fmt.Errorf("expected prompt %q, got %q", capturedScenario.Prompt, inv.Prompt)
				}
				if inv.ConfigDir != "/test/configdir" {
					return fmt.Errorf("expected ConfigDir %q, got %q", "/test/configdir", inv.ConfigDir)
				}
				if inv.PathPrefix != "/test/prefix" {
					return fmt.Errorf("expected PathPrefix %q, got %q", "/test/prefix", inv.PathPrefix)
				}
				if inv.VaultPath == "" {
					return fmt.Errorf("VaultPath must not be empty")
				}
				return nil
			}),
		).Return(eval.AgentResult{
			ResultJSON:    resultJSON,
			TranscriptRaw: sessionJSONL,
		}, nil)

		// Results.Append called with the run result; check Layer1.Turns==3 and go-test behavior.
		writerImp.Append.ArgsShould(
			match.BeAny,
			match.Satisfy(func(r eval.RunResult) error {
				if r.Layer1.Turns != 3 {
					return fmt.Errorf("expected Layer1.Turns=3, got %d", r.Layer1.Turns)
				}
				for _, b := range r.Behaviors {
					if b.Name == "used-go-test-not-targ" && !b.Occurred {
						return fmt.Errorf("expected used-go-test-not-targ behavior to have Occurred=true")
					}
				}
				return nil
			}),
		).Return(nil)
	}

	<-done
}

func TestRun_UnknownArm_ReturnsErrUnknownArm(t *testing.T) {
	t.Parallel()

	err := eval.Run(context.Background(), "bogus", eval.RunConfig{}, eval.Deps{})
	if !errors.Is(err, eval.ErrUnknownArm) {
		t.Fatalf("got %v, want ErrUnknownArm", err)
	}
}
