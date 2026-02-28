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

	// Given any query, any hook type, any budget
	_ = NewGomegaWithT(t)
	ctx := context.Background()

	mockStore, storeExp := MockStore(t)
	mockFormatter, _ := MockFormatter(t)
	mockAudit, _ := MockAuditLog(t)

	// When SurfaceRun is called
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

	// Then store.Surface called; Given empty results, nil error
	storeExp.Surface.ArgsShould(match.BeAny, Equal("some query"), Equal(5)).
		Return([]store.ScoredMemory(nil), nil)

	// Then SurfaceRun returns "", nil error (never calls formatter, audit, or IncrementSurfacing)
	call.ReturnsShould(Equal(""), Not(HaveOccurred()))
}

func TestT54_SurfacingPipelineEndToEnd(t *testing.T) {
	t.Parallel()

	// Given any query, any hookType, any budget
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

	// When SurfaceRun is called
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

	// Then store.Surface called; Given [sm1], nil error
	storeExp.Surface.ArgsShould(match.BeAny, Equal("test query"), Equal(3)).
		Return([]store.ScoredMemory{scored}, nil)

	// Then formatter.FormatSurfacing called; Given formatted string
	fmtExp.FormatSurfacing.ArgsShould(Equal([]store.ScoredMemory{scored}), Equal("user-prompt")).
		Return(`<system-reminder source="engram">formatted</system-reminder>`)

	// Then store.IncrementSurfacing called with memory IDs; Given nil error
	storeExp.IncrementSurfacing.ArgsShould(match.BeAny, Equal([]string{"m_test001"})).
		Return(nil)

	// Then audit.Log called with surface/returned entry; Given nil error
	auditExp.Log.ArgsShould(match.BeAny).
		Return(nil)

	// Then SurfaceRun returns formatted string, nil error
	call.ReturnsShould(
		Equal(`<system-reminder source="engram">formatted</system-reminder>`),
		Not(HaveOccurred()),
	)

	// Suppress unused variable warnings
	_ = g
}
