//go:build targ

package dev

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/toejough/imptest/match"
)

//go:generate impgen executeNilaway --dependency --name MockExecuteNilaway

func Test794b4b1a_ConfiguredInvocation(t *testing.T) {
	for _, fix := range []bool{false, true} {
		t.Run(map[bool]string{false: "check", true: "fix"}[fix], func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			execute, expectation := MockExecuteNilaway(t)
			supplied := map[string]string{"GOMEMLIMIT": "4GiB"}
			lookup := func(key string) (string, bool) {
				value, present := supplied[key]
				return value, present
			}
			done := make(chan error, 1)
			go func() { done <- runConfiguredNilaway(ctx, fix, lookup, execute) }()
			call := expectation.ArgsShould(match.BeAny, match.BeAny, match.BeAny)
			got := call.GetArgs()
			wantArgs := []string{"-include-pkgs=github.com/toejough/engram"}
			if fix {
				wantArgs = append(wantArgs, "-fix")
			}
			wantArgs = append(wantArgs, "./...")
			wantSettings := map[string]string{"GOMEMLIMIT": "4GiB", "GOGC": "50", "GOMAXPROCS": "2"}
			if got.Ctx != ctx || !reflect.DeepEqual(got.Args, wantArgs) || !reflect.DeepEqual(got.Settings, wantSettings) {
				t.Errorf("command request = %#v, want supplied context, %v, %v", got, wantArgs, wantSettings)
			}
			call.Return(nil)
			if err := <-done; err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(supplied, map[string]string{"GOMEMLIMIT": "4GiB"}) {
				t.Fatal("caller settings were modified")
			}
		})
	}
}

func Test794b4b1b_AnalyzerFailure(t *testing.T) {
	ctx := context.Background()
	execute, expectation := MockExecuteNilaway(t)
	want := errors.New("analyzer failed")
	done := make(chan error, 1)
	go func() {
		done <- runConfiguredNilaway(ctx, false, func(string) (string, bool) { return "", false }, execute)
	}()
	call := expectation.ArgsShould(match.BeAny, match.BeAny, match.BeAny)
	call.Return(want)
	if got := <-done; got != want {
		t.Fatalf("got %v, want original analyzer failure %v", got, want)
	}
}
