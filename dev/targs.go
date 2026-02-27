//go:build targ

package dev

import (
	"github.com/toejough/targ"
	_ "github.com/toejough/targ/dev"
)

func init() {
	targ.Register()
	// _ = targ.DeregisterFrom("github.com/toejough/targ/dev")
}
