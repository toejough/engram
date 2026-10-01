package cli

import (
	"testing"

	. "github.com/onsi/gomega"
)

// TestIdentityStamp_Stamp_NilReceiverIsNoop pins stamp's documented
// contract directly: a nil *identityStamp (a bookkeeping amend) leaves
// repo/user/vault untouched. The only production callers now reach stamp
// through stampPreservingUser, which already guards nil before calling
// stamp — so stamp's own nil branch needs this direct unit test to stay
// covered.
func TestIdentityStamp_Stamp_NilReceiverIsNoop(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stamp *identityStamp

	repo, user, vault := "r", "u", "v"
	stamp.stamp(&repo, &user, &vault)

	g.Expect(repo).To(Equal("r"))
	g.Expect(user).To(Equal("u"))
	g.Expect(vault).To(Equal("v"))
}

// TestIdentityStamp_Stamp_OverwritesAllThree pins stamp's non-nil path: it
// overwrites repo, user and vault unconditionally, including blanking a
// non-empty user to "" — the behavior stampPreservingUser exists to avoid
// on the amend path, kept here as stamp's own documented contract.
func TestIdentityStamp_Stamp_OverwritesAllThree(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	stamp := &identityStamp{Repo: "new-repo", User: "", Vault: "new-vault"}

	repo, user, vault := "old-repo", "old-user", "old-vault"
	stamp.stamp(&repo, &user, &vault)

	g.Expect(repo).To(Equal("new-repo"))
	g.Expect(user).To(Equal(""))
	g.Expect(vault).To(Equal("new-vault"))
}
