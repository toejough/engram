package surface_test

import (
	"context"
	"errors"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/toejough/imptest/match"

	"engram/internal/store"
	"engram/internal/surface"
)

func TestRun_AuditLogError(t *testing.T) {
	t.Parallel()

	_ = NewGomegaWithT(t)
	ctx := context.Background()
	scored := store.ScoredMemory{Memory: store.Memory{ID: "m_1"}, Score: 0.5}
	mockStore, storeExp := MockStore(t)
	mockFormatter, fmtExp := MockFormatter(t)
	mockAudit, auditExp := MockAuditLog(t)
	call := StartRun(t, surface.Run, ctx, mockStore, mockFormatter, mockAudit, "hook", "q", 5)

	storeExp.Surface.ArgsShould(match.BeAny, match.BeAny, match.BeAny).
		Return([]store.ScoredMemory{scored}, nil)
	fmtExp.FormatSurfacing.ArgsShould(match.BeAny, match.BeAny).
		Return("formatted")
	storeExp.IncrementSurfacing.ArgsShould(match.BeAny, match.BeAny).
		Return(nil)
	storeExp.RecordSurfacing.ArgsShould(match.BeAny, match.BeAny).
		Return(nil)
	auditExp.Log.ArgsShould(match.BeAny).
		Return(errors.New("audit error"))
	call.ReturnsShould(match.BeAny, HaveOccurred())
}

func TestRun_IncrementError(t *testing.T) {
	t.Parallel()

	_ = NewGomegaWithT(t)
	ctx := context.Background()
	scored := store.ScoredMemory{Memory: store.Memory{ID: "m_1"}, Score: 0.5}
	mockStore, storeExp := MockStore(t)
	mockFormatter, fmtExp := MockFormatter(t)
	mockAudit, _ := MockAuditLog(t)
	call := StartRun(t, surface.Run, ctx, mockStore, mockFormatter, mockAudit, "hook", "q", 5)

	storeExp.Surface.ArgsShould(match.BeAny, match.BeAny, match.BeAny).
		Return([]store.ScoredMemory{scored}, nil)
	fmtExp.FormatSurfacing.ArgsShould(match.BeAny, match.BeAny).
		Return("formatted")
	storeExp.IncrementSurfacing.ArgsShould(match.BeAny, match.BeAny).
		Return(errors.New("increment error"))
	call.ReturnsShould(match.BeAny, HaveOccurred())
}

func TestRun_SurfaceQueryError(t *testing.T) {
	t.Parallel()

	_ = NewGomegaWithT(t)
	ctx := context.Background()
	mockStore, storeExp := MockStore(t)
	mockFormatter, _ := MockFormatter(t)
	mockAudit, _ := MockAuditLog(t)
	call := StartRun(t, surface.Run, ctx, mockStore, mockFormatter, mockAudit, "hook", "q", 5)

	storeExp.Surface.ArgsShould(match.BeAny, match.BeAny, match.BeAny).
		Return(nil, errors.New("db error"))
	call.ReturnsShould(match.BeAny, HaveOccurred())
}

func TestT53_EmptyResultReturnsEmptyString(t *testing.T) {
	t.Parallel()

	_ = NewGomegaWithT(t)
	ctx := context.Background()

	mockStore, storeExp := MockStore(t)
	mockFormatter, _ := MockFormatter(t)
	mockAudit, _ := MockAuditLog(t)

	call := StartRun(
		t,
		surface.Run,
		ctx,
		mockStore,
		mockFormatter,
		mockAudit,
		"session-start",
		"some query",
		5,
	)

	// ClearSessionSurfacings called first (session-start hook)
	storeExp.ClearSessionSurfacings.ArgsShould(match.BeAny).
		Return(nil)

	// store.Surface returns empty
	storeExp.Surface.ArgsShould(match.BeAny, Equal("some query"), Equal(5)).
		Return([]store.ScoredMemory(nil), nil)

	// SurfaceRun returns "", nil error
	call.ReturnsShould(Equal(""), Not(HaveOccurred()))
}

func TestT54_SurfacingPipelineEndToEnd(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)
	ctx := context.Background()

	scored := store.ScoredMemory{
		Memory: store.Memory{
			ID:         "m_test001",
			Title:      "Test memory",
			Content:    "Some guidance",
			Confidence: "A",
		},
		Score: 0.8,
	}

	mockStore, storeExp := MockStore(t)
	mockFormatter, fmtExp := MockFormatter(t)
	mockAudit, auditExp := MockAuditLog(t)

	call := StartRun(
		t,
		surface.Run,
		ctx,
		mockStore,
		mockFormatter,
		mockAudit,
		"user-prompt",
		"test query",
		3,
	)

	storeExp.Surface.ArgsShould(match.BeAny, Equal("test query"), Equal(3)).
		Return([]store.ScoredMemory{scored}, nil)

	fmtExp.FormatSurfacing.ArgsShould(Equal([]store.ScoredMemory{scored}), Equal("user-prompt")).
		Return(`<system-reminder source="engram">formatted</system-reminder>`)

	storeExp.IncrementSurfacing.ArgsShould(match.BeAny, Equal([]string{"m_test001"})).
		Return(nil)

	storeExp.RecordSurfacing.ArgsShould(match.BeAny, Equal([]string{"m_test001"})).
		Return(nil)

	auditExp.Log.ArgsShould(match.BeAny).
		Return(nil)

	call.ReturnsShould(
		Equal(`<system-reminder source="engram">formatted</system-reminder>`),
		Not(HaveOccurred()),
	)

	_ = g
}

// T-71: Surface pipeline records surfaced memory IDs
func TestT71_SurfacePipelineRecordsSurfacedMemoryIDs(t *testing.T) {
	t.Parallel()

	_ = NewGomegaWithT(t)
	ctx := context.Background()

	scored1 := store.ScoredMemory{Memory: store.Memory{ID: "m_aaa"}, Score: 0.9}
	scored2 := store.ScoredMemory{Memory: store.Memory{ID: "m_bbb"}, Score: 0.7}

	mockStore, storeExp := MockStore(t)
	mockFormatter, fmtExp := MockFormatter(t)
	mockAudit, auditExp := MockAuditLog(t)

	// user-prompt hook — no ClearSessionSurfacings
	call := StartRun(
		t,
		surface.Run,
		ctx,
		mockStore,
		mockFormatter,
		mockAudit,
		"user-prompt",
		"query",
		3,
	)

	storeExp.Surface.ArgsShould(match.BeAny, match.BeAny, match.BeAny).
		Return([]store.ScoredMemory{scored1, scored2}, nil)

	fmtExp.FormatSurfacing.ArgsShould(match.BeAny, match.BeAny).
		Return("formatted output")

	storeExp.IncrementSurfacing.ArgsShould(match.BeAny, Equal([]string{"m_aaa", "m_bbb"})).
		Return(nil)

	// RecordSurfacing called with the 2 surfaced memory IDs
	storeExp.RecordSurfacing.ArgsShould(match.BeAny, Equal([]string{"m_aaa", "m_bbb"})).
		Return(nil)

	auditExp.Log.ArgsShould(match.BeAny).
		Return(nil)

	call.ReturnsShould(Equal("formatted output"), Not(HaveOccurred()))
}

// T-72: Session-start surface clears surfacing log before surfacing
func TestT72_SessionStartClearsSurfacingLogBeforeSurfacing(t *testing.T) {
	t.Parallel()

	_ = NewGomegaWithT(t)
	ctx := context.Background()

	scored := store.ScoredMemory{Memory: store.Memory{ID: "m_ccc"}, Score: 0.8}

	mockStore, storeExp := MockStore(t)
	mockFormatter, fmtExp := MockFormatter(t)
	mockAudit, auditExp := MockAuditLog(t)

	call := StartRun(
		t,
		surface.Run,
		ctx,
		mockStore,
		mockFormatter,
		mockAudit,
		"session-start",
		"query",
		5,
	)

	// ClearSessionSurfacings called FIRST (before Surface)
	storeExp.ClearSessionSurfacings.ArgsShould(match.BeAny).
		Return(nil)

	storeExp.Surface.ArgsShould(match.BeAny, match.BeAny, match.BeAny).
		Return([]store.ScoredMemory{scored}, nil)

	fmtExp.FormatSurfacing.ArgsShould(match.BeAny, match.BeAny).
		Return("session output")

	storeExp.IncrementSurfacing.ArgsShould(match.BeAny, Equal([]string{"m_ccc"})).
		Return(nil)

	// RecordSurfacing called with the surfaced memory ID
	storeExp.RecordSurfacing.ArgsShould(match.BeAny, Equal([]string{"m_ccc"})).
		Return(nil)

	auditExp.Log.ArgsShould(match.BeAny).
		Return(nil)

	call.ReturnsShould(Equal("session output"), Not(HaveOccurred()))
}
