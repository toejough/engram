//go:build targ && integration

package dev

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"testing"

	. "github.com/onsi/gomega"
)

func Test794b4c1a_InheritedEnvironmentPassThrough(t *testing.T) {
	for _, supplied := range []map[string]string{
		{},
		{"GOMEMLIMIT": "4GiB"},
		{"GOMEMLIMIT": "off", "GOGC": "off", "GOMAXPROCS": "8"},
		{"GOMEMLIMIT": "", "GOGC": "", "GOMAXPROCS": ""},
	} {
		t.Run(formatSettings(supplied), func(t *testing.T) {
			g := NewWithT(t)
			capture := standInNilaway(t, 0)
			setResourceVariables(t, supplied)

			err := executeNilaway(context.Background(), []string{"./..."})

			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(readStandInMarker(t, capture)).To(Equal(standInMarker))
			g.Expect(os.Getenv("ENGRAM_NILAWAY_MARKER")).To(Equal(standInMarker), "the parent's marker changed")
			g.Expect(readStandInResources(t, capture)).To(Equal(supplied))
			g.Expect(currentResourceVariables()).To(Equal(supplied))
		})
	}
}

func Test794b4c1b_NilAwayReceivesTheRequestsArguments(t *testing.T) {
	g := NewWithT(t)
	capture := standInNilaway(t, 0)
	args := []string{"-include-pkgs=github.com/toejough/engram", "./..."}

	err := executeNilaway(context.Background(), args)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(readStandInArgs(t, capture)).To(Equal(args))
}

func Test794b4c1c_SuccessfulRunReturnsSuccess(t *testing.T) {
	g := NewWithT(t)
	standInNilaway(t, 0)

	g.Expect(executeNilaway(context.Background(), []string{"./..."})).To(Succeed())
}

func Test794b4c1d_FailingRunReturnsNilAwaysFailure(t *testing.T) {
	g := NewWithT(t)
	standInNilaway(t, exitFailure)

	err := executeNilaway(context.Background(), []string{"./..."})

	g.Expect(err).To(HaveOccurred())
	var exitError *exec.ExitError
	g.Expect(errors.As(err, &exitError)).To(BeTrue(), "error %v does not carry NilAway's exit status", err)
	if exitError != nil {
		g.Expect(exitError.ExitCode()).To(Equal(exitFailure))
	}
}

// unexported constants.
const (
	exitFailure = 3
)
