//go:build targ

package dev

import (
	"context"
	"reflect"
	"slices"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/targ"
	targdev "github.com/toejough/targ/dev"
)

func Test794b3a1a_ModeArguments(t *testing.T) {
	t.Parallel()

	for _, scenario := range []struct {
		name string
		fix  bool
		want []string
	}{
		{"check", false, []string{"-include-pkgs=github.com/toejough/engram", "./..."}},
		{"fix", true, []string{"-include-pkgs=github.com/toejough/engram", "-fix", "./..."}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()

			g := NewWithT(t)

			g.Expect(nilawayArgs(scenario.fix)).To(Equal(scenario.want))
		})
	}
}

func Test794b4a1a_TargetWiring(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	for target, expected := range map[*targ.Target]func(context.Context) error{
		checkNilsFixTarget:     checkNilsFix,
		checkNilsForFailTarget: checkNilsForFail,
	} {
		g.Expect(sameFunc(target.Fn(), expected)).To(BeTrue(),
			"%s does not use Engram's NilAway invocation", target.GetName())
	}

	g.Expect(checkNilsFixTarget.GetName()).To(Equal("check-nils-fix"))
	g.Expect(checkNilsForFailTarget.GetName()).To(Equal("check-nils-for-fail"))

	for name, aggregate := range map[string]*targ.Target{
		"check-nils":     checkNilsTarget,
		"check":          checkTarget,
		"check-for-fail": checkForFailTarget,
		"check-full":     checkFullTarget,
	} {
		g.Expect(aggregate.GetName()).To(Equal(name))
		g.Expect(reaches(aggregate, checkNilsForFailTarget) || reaches(aggregate, checkNilsFixTarget)).To(BeTrue(), name)

		for _, targs := range []*targ.Target{targdev.CheckNilsFix, targdev.CheckNilsForFail} {
			g.Expect(reaches(aggregate, targs)).To(BeFalse(), "%s reaches Targ's %s", name, targs.GetName())
		}
	}

	g.Expect(reaches(checkTarget, checkNilsFixTarget)).To(BeTrue())
	g.Expect(reaches(checkTarget, checkNilsForFailTarget)).To(BeTrue())

	for _, aggregate := range []*targ.Target{checkForFailTarget, checkFullTarget} {
		g.Expect(reaches(aggregate, checkNilsForFailTarget)).To(BeTrue(), aggregate.GetName())
	}
}

// Test794b4a1c fails whenever Targ's aggregates change shape, so Engram's copies cannot drift unnoticed.
func Test794b4a1c_AggregatesMatchTargsApartFromNilTargets(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)
	nilTargets := map[*targ.Target]*targ.Target{
		targdev.CheckNils:        checkNilsTarget,
		targdev.CheckNilsFix:     checkNilsFixTarget,
		targdev.CheckNilsForFail: checkNilsForFailTarget,
	}
	// Engram's own additions to the aggregates, appended to the final dependency group: the dev package's tests
	// and the lint run that covers the dev package's build-tagged files.
	engramExtras := map[string][]*targ.Target{
		"check":          {lintDevTarget},
		"check-for-fail": {testDevTarget, testDevIntegrationTarget, lintDevTarget},
		"check-full":     {testDevTarget, testDevIntegrationTarget, lintDevTarget},
	}

	for _, pair := range []struct{ engram, targs *targ.Target }{
		{checkNilsTarget, targdev.CheckNils},
		{checkTarget, targdev.Check},
		{checkForFailTarget, targdev.CheckForFail},
		{checkFullTarget, targdev.CheckFull},
	} {
		name := pair.engram.GetName()
		g.Expect(name).To(Equal(pair.targs.GetName()))
		g.Expect(pair.engram.GetDescription()).To(Equal(pair.targs.GetDescription()), name)

		want := pair.targs.GetDepGroups()
		for index := range want {
			mapped := make([]*targ.Target, 0, len(want[index].Targets))
			for _, dep := range want[index].Targets {
				if replacement, isNil := nilTargets[dep]; isNil {
					dep = replacement
				}

				mapped = append(mapped, dep)
			}

			want[index].Targets = mapped
		}

		if extras := engramExtras[name]; len(extras) > 0 {
			last := len(want) - 1
			want[last].Targets = append(want[last].Targets, extras...)
		}

		g.Expect(pair.engram.GetDepGroups()).To(Equal(want), name)
	}
}

func sameFunc(actual any, expected func(context.Context) error) bool {
	return reflect.ValueOf(actual).Pointer() == reflect.ValueOf(expected).Pointer()
}

// reaches reports whether from is, or depends transitively on, wanted.
func reaches(from, wanted *targ.Target) bool {
	if from == wanted {
		return true
	}

	return slices.ContainsFunc(from.GetDeps(), func(dep *targ.Target) bool { return reaches(dep, wanted) })
}
