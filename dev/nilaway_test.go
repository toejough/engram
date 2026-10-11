//go:build targ

package dev

import (
	"context"
	"reflect"
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
		targdev.CheckNilsFix:     checkNilsFix,
		targdev.CheckNilsForFail: checkNilsForFail,
	} {
		g.Expect(sameFunc(target.Fn(), expected)).To(BeTrue(), "%s does not use Engram's NilAway invocation", target.GetName())
	}
	g.Expect(targdev.CheckNilsFix.GetName()).To(Equal("check-nils-fix"))
	g.Expect(targdev.CheckNilsForFail.GetName()).To(Equal("check-nils-for-fail"))
	g.Expect(targdev.CheckNils.GetDeps()).To(ContainElements(targdev.CheckNilsFix, targdev.CheckNilsForFail))
	g.Expect(targdev.Check.GetDeps()).To(ContainElement(targdev.CheckNils))
	for _, target := range []*targ.Target{targdev.CheckForFail, targdev.CheckFull} {
		g.Expect(target.GetDeps()).To(ContainElement(targdev.CheckNilsForFail), target.GetName())
	}
}

func Test794b4a1b_RegistrationMetadataIsPreserved(t *testing.T) {
	t.Parallel()
	t.Skip("blocked: targ has no supported way to replace a target's action while keeping its source package; " +
		"see the 794 report")

	g := NewWithT(t)
	for _, target := range []*targ.Target{targdev.CheckNilsFix, targdev.CheckNilsForFail} {
		g.Expect(target.GetSource()).To(Equal(targSourcePackage), "%s lost its registration source", target.GetName())
		g.Expect(target.GetDescription()).NotTo(BeEmpty(), target.GetName())
	}
}

// unexported constants.
const (
	targSourcePackage = "github.com/toejough/targ/dev"
)

func sameFunc(actual any, expected func(context.Context) error) bool {
	return reflect.ValueOf(actual).Pointer() == reflect.ValueOf(expected).Pointer()
}
