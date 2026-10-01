package cli

import (
	"errors"
	"testing"

	. "github.com/onsi/gomega"
)

// TestPreflightRenames_EmptyMapListsNothing: an empty rename map needs no
// pre-flight, so the vault is not even listed.
func TestPreflightRenames_EmptyMapListsNothing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := RenameRewriteDeps{ListMD: func(string) ([]string, error) {
		t.Error("ListMD called for an empty rename map")

		return nil, nil
	}}

	g.Expect(preflightRenames(deps, "/vault", nil)).To(Succeed())
}

// TestPreflightRenames_ListErrorPropagates: a listing failure is wrapped and
// returned.
func TestPreflightRenames_ListErrorPropagates(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	errList := errors.New("list failed")
	deps := RenameRewriteDeps{ListMD: func(string) ([]string, error) { return nil, errList }}

	g.Expect(preflightRenames(deps, "/vault", map[string]string{"1.2026-01-01.a": "2.2026-01-01.a"})).
		To(MatchError(errList))
}
