//go:build targ

package dev

import (
	"errors"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/toejough/imptest/match"
)

// TestModuleSyntax_FailsWhenNodeIsNotFound: a check that cannot run does not pass (793b55).
func TestModuleSyntax_FailsWhenNodeIsNotFound(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	find, findCalls := MockFindNode(t)
	parse, _ := MockParseModule(t)
	done := make(chan error, 1)

	go func() {
		done <- checkModuleSyntax(t.Context(), map[string]string{"hooks/register.js": "export {}"}, find, parse)
	}()

	// the expectation runs apart: a call that does not match one fails the test, but this version of imptest then leaves
	// the expectation waiting, so the test waits for the outcome with a time limit
	go func() { findCalls.Called().Return("", errNodeAbsent) }()

	select {
	case err := <-done:
		g.Expect(err).To(MatchError(errNodeAbsent))
		g.Expect(err).To(MatchError(ContainSubstring("node")))
	case <-time.After(mockTimeout):
		t.Fatal("the check did not finish: it did not look node up, or it parsed a module without node")
	}
}

// TestModuleSyntax_NamesEachFileThatDoesNotParse: node is asked about each module in the order of the paths, with the
// module's source, and the modules it rejects are named with what it said.
func TestModuleSyntax_NamesEachFileThatDoesNotParse(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	find, findCalls := MockFindNode(t)
	parse, parseCalls := MockParseModule(t)
	done := make(chan error, 1)
	modules := map[string]string{"b/two.js": "export function (", "a/one.js": "export {}"}

	go func() { done <- checkModuleSyntax(t.Context(), modules, find, parse) }()

	go func() {
		findCalls.Called().Return("/usr/local/bin/node", nil)
		parseCalls.ArgsShould(match.BeAny, Equal("/usr/local/bin/node"), Equal("export {}")).Return([]byte(nil), nil)
		parseCalls.ArgsShould(match.BeAny, Equal("/usr/local/bin/node"), Equal("export function (")).
			Return([]byte("SyntaxError: Unexpected end of input\n"), errNodeRejected)
	}()

	select {
	case err := <-done:
		g.Expect(err).To(HaveOccurred())

		if err != nil {
			g.Expect(err.Error()).To(ContainSubstring("b/two.js"))
			g.Expect(err.Error()).To(ContainSubstring("SyntaxError: Unexpected end of input"))
			g.Expect(err.Error()).NotTo(ContainSubstring("a/one.js"))
		}
	case <-time.After(mockTimeout):
		t.Fatal("the check did not finish: node was not asked about each module, in order, with its source")
	}
}

// unexported constants.
const (
	// mockTimeout is how long a test waits for the code under it before it fails, so that an expectation that is never
	// met does not hold the run.
	mockTimeout = 5 * time.Second
)

// unexported variables.
var (
	errNodeAbsent   = errors.New("node is not on the path")
	errNodeRejected = errors.New("exit status 1")
)
