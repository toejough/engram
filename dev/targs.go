//go:build targ

package dev

import (
	"os"

	"github.com/toejough/targ"
	_ "github.com/toejough/targ/dev"
)

func init() {
	// Engram's spec-traced tests use TestT<N>_ naming (not TestProperty_).
	if os.Getenv("TARG_BASELINE_PATTERN") == "" {
		os.Setenv("TARG_BASELINE_PATTERN", `TestT[0-9]+_`)
	}

	targ.Register()
	// _ = targ.DeregisterFrom("github.com/toejough/targ/dev")
}
