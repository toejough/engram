//go:build targ

package dev

import (
	"context"
	"errors"

	"github.com/toejough/targ"
)

var errNotImplemented = errors.New("not implemented")

func init() {
	targ.Register(targ.Targ(c4Audit).Name("c4-audit").
		Description("Structurally audit a C4 L1 markdown file (rule 6 + front-matter + cross-links). Exits 1 on any finding."))
	targ.Register(targ.Targ(c4L1Build).Name("c4-l1-build").
		Description("Build canonical C4 L1 markdown from a JSON spec next to the input file."))
	targ.Register(targ.Targ(c4L1Externals).Name("c4-l1-externals").
		Description("Walk the repo with Go AST analysis and emit external-system candidates as JSON."))
	targ.Register(targ.Targ(c4History).Name("c4-history").
		Description("Wrap git log and emit structured JSON of commit metadata + bodies."))
}

func c4Audit(_ context.Context) error       { return errNotImplemented }
func c4L1Build(_ context.Context) error     { return errNotImplemented }
func c4L1Externals(_ context.Context) error { return errNotImplemented }
func c4History(_ context.Context) error     { return errNotImplemented }
