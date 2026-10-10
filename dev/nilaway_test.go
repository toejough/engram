//go:build targ

package dev

import (
	"context"
	"reflect"
	"testing"

	"github.com/toejough/targ"
	targdev "github.com/toejough/targ/dev"
)

func Test794b3a1a_ModeArguments(t *testing.T) {
	for _, fix := range []bool{false, true} {
		want := []string{"-include-pkgs=github.com/toejough/engram"}
		if fix {
			want = append(want, "-fix")
		}
		want = append(want, "./...")
		if got := nilawayArgs(fix); !reflect.DeepEqual(got, want) {
			t.Fatalf("fix=%v: got %v, want %v", fix, got, want)
		}
	}
}

func Test794b3b1a_EnvironmentPrecedence(t *testing.T) {
	for _, supplied := range []map[string]string{
		{},
		{"GOMEMLIMIT": "4GiB"},
		{"GOMEMLIMIT": "off", "GOGC": "off", "GOMAXPROCS": "8"},
		{"GOMEMLIMIT": "", "GOGC": "", "GOMAXPROCS": ""},
	} {
		got := nilawayEnvironment(func(key string) (string, bool) {
			value, present := supplied[key]
			return value, present
		})
		want := map[string]string{"GOMEMLIMIT": "2GiB", "GOGC": "50", "GOMAXPROCS": "2"}
		for key, value := range supplied {
			want[key] = value
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("supplied=%v: got %v, want %v", supplied, got, want)
		}
	}
}

func Test794b4a1a_TargetWiring(t *testing.T) {
	for target, expected := range map[*targ.Target]func(context.Context) error{
		targdev.CheckNilsFix:     checkNilsFix,
		targdev.CheckNilsForFail: checkNilsForFail,
	} {
		if reflect.ValueOf(target.Fn()).Pointer() != reflect.ValueOf(expected).Pointer() {
			t.Fatalf("%s does not use Engram's NilAway invocation", target.GetName())
		}
	}
	if targdev.CheckNilsFix.GetName() != "check-nils-fix" || targdev.CheckNilsForFail.GetName() != "check-nils-for-fail" {
		t.Fatal("direct NilAway target names changed")
	}
	for target, dependency := range map[*targ.Target]*targ.Target{
		targdev.Check:     targdev.CheckNils,
		targdev.CheckNils: targdev.CheckNilsFix,
	} {
		found := false
		for _, candidate := range target.GetDeps() {
			if candidate == dependency {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s lost dependency %s", target.GetName(), dependency.GetName())
		}
	}
	for _, target := range []*targ.Target{targdev.CheckForFail, targdev.CheckFull, targdev.CheckNils} {
		found := false
		for _, dependency := range target.GetDeps() {
			if dependency == targdev.CheckNilsForFail {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s lost its configured NilAway dependency", target.GetName())
		}
	}
}
