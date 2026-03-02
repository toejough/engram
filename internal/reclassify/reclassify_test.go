package reclassify_test

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/imptest/match"

	"engram/internal/reclassify"
)

func TestT66_ReclassifyDecreasesImpactForAllSurfaced(t *testing.T) {
	t.Parallel()

	_ = NewGomegaWithT(t)
	ctx := context.Background()

	// Given a mock store where GetSessionSurfacings returns ["m_aaa", "m_bbb"]
	mockStore, storeExp := MockStore(t)
	mockAudit, auditExp := MockAuditLog(t)

	call := StartRun(t, reclassify.Run, ctx, mockStore, mockAudit, 0.8)

	storeExp.GetSessionSurfacings.ArgsShould(match.BeAny).
		Return([]string{"m_aaa", "m_bbb"}, nil)

	// DecreaseImpact called for m_aaa
	storeExp.DecreaseImpact.ArgsShould(match.BeAny, Equal("m_aaa"), Equal(0.8)).
		Return(nil)

	// Audit log called for m_aaa
	auditExp.Log.ArgsShould(match.BeAny).
		Return(nil)

	// DecreaseImpact called for m_bbb
	storeExp.DecreaseImpact.ArgsShould(match.BeAny, Equal("m_bbb"), Equal(0.8)).
		Return(nil)

	// Audit log called for m_bbb
	auditExp.Log.ArgsShould(match.BeAny).
		Return(nil)

	// Then returns (2, nil)
	call.ReturnsShould(Equal(2), Not(HaveOccurred()))
}

func TestT67_ReclassifyNoSurfacedIsNoOp(t *testing.T) {
	t.Parallel()

	_ = NewGomegaWithT(t)
	ctx := context.Background()

	// Given a mock store where GetSessionSurfacings returns empty
	mockStore, storeExp := MockStore(t)
	mockAudit, _ := MockAuditLog(t)

	call := StartRun(t, reclassify.Run, ctx, mockStore, mockAudit, 0.8)

	storeExp.GetSessionSurfacings.ArgsShould(match.BeAny).
		Return([]string{}, nil)

	// Then returns (0, nil) — no DecreaseImpact or Log calls
	call.ReturnsShould(Equal(0), Not(HaveOccurred()))
}

func TestT68_ReclassifyPropagatesStoreErrors(t *testing.T) {
	t.Parallel()

	_ = NewGomegaWithT(t)
	ctx := context.Background()

	// Given a mock store where GetSessionSurfacings returns error
	someError := errors.New("db connection lost")
	mockStore, storeExp := MockStore(t)
	mockAudit, _ := MockAuditLog(t)

	call := StartRun(t, reclassify.Run, ctx, mockStore, mockAudit, 0.8)

	storeExp.GetSessionSurfacings.ArgsShould(match.BeAny).
		Return(nil, someError)

	// Then returns (0, error wrapping someError)
	call.ReturnsShould(Equal(0), MatchError(ContainSubstring("db connection lost")))
}
