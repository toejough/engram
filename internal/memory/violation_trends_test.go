package memory_test

import (
	"testing"
	"time"

	"github.com/toejough/projctl/internal/memory"
)

func TestComputeViolationTrends_DecliningViolations(t *testing.T) {
	now := time.Now()
	periodDays := 7

	// Create violations: 10 in old period, 3 in recent period (declining)
	violations := []memory.ChangelogEntry{
		// Old period (14-7 days ago): 10 violations
		{Timestamp: now.Add(-14 * 24 * time.Hour), Action: "hook_violation", Metadata: map[string]string{"rule": "no-amend-pushed", "hook": "PreToolUse"}},
		{Timestamp: now.Add(-13 * 24 * time.Hour), Action: "hook_violation", Metadata: map[string]string{"rule": "no-amend-pushed", "hook": "PreToolUse"}},
		{Timestamp: now.Add(-12 * 24 * time.Hour), Action: "hook_violation", Metadata: map[string]string{"rule": "no-amend-pushed", "hook": "PreToolUse"}},
		{Timestamp: now.Add(-11 * 24 * time.Hour), Action: "hook_violation", Metadata: map[string]string{"rule": "no-amend-pushed", "hook": "PreToolUse"}},
		{Timestamp: now.Add(-10 * 24 * time.Hour), Action: "hook_violation", Metadata: map[string]string{"rule": "no-amend-pushed", "hook": "PreToolUse"}},
		{Timestamp: now.Add(-9 * 24 * time.Hour), Action: "hook_violation", Metadata: map[string]string{"rule": "no-amend-pushed", "hook": "PreToolUse"}},
		{Timestamp: now.Add(-8 * 24 * time.Hour), Action: "hook_violation", Metadata: map[string]string{"rule": "no-amend-pushed", "hook": "PreToolUse"}},
		{Timestamp: now.Add(-8 * 24 * time.Hour), Action: "hook_violation", Metadata: map[string]string{"rule": "no-amend-pushed", "hook": "PreToolUse"}},
		{Timestamp: now.Add(-7 * 24 * time.Hour), Action: "hook_violation", Metadata: map[string]string{"rule": "no-amend-pushed", "hook": "PreToolUse"}},
		{Timestamp: now.Add(-7 * 24 * time.Hour), Action: "hook_violation", Metadata: map[string]string{"rule": "no-amend-pushed", "hook": "PreToolUse"}},
		// Recent period (last 7 days): 3 violations
		{Timestamp: now.Add(-6 * 24 * time.Hour), Action: "hook_violation", Metadata: map[string]string{"rule": "no-amend-pushed", "hook": "PreToolUse"}},
		{Timestamp: now.Add(-3 * 24 * time.Hour), Action: "hook_violation", Metadata: map[string]string{"rule": "no-amend-pushed", "hook": "PreToolUse"}},
		{Timestamp: now.Add(-1 * 24 * time.Hour), Action: "hook_violation", Metadata: map[string]string{"rule": "no-amend-pushed", "hook": "PreToolUse"}},
	}

	trends := memory.ComputeViolationTrends(violations, periodDays)

	if len(trends) != 1 {
		t.Fatalf("expected 1 trend, got %d", len(trends))
	}

	trend, ok := trends["no-amend-pushed"]
	if !ok {
		t.Fatal("expected trend for 'no-amend-pushed'")
	}

	if trend.Trending != "improving" {
		t.Errorf("expected trending='improving', got %q", trend.Trending)
	}

	if trend.TotalViolations != 13 {
		t.Errorf("expected total_violations=13, got %d", trend.TotalViolations)
	}
}

func TestComputeViolationTrends_StableViolations(t *testing.T) {
	now := time.Now()
	periodDays := 7

	// Create violations: 5 in old period, 5 in recent period (stable)
	violations := []memory.ChangelogEntry{
		// Old period (14-7 days ago): 5 violations
		{Timestamp: now.Add(-14 * 24 * time.Hour), Action: "hook_violation", Metadata: map[string]string{"rule": "use-targ", "hook": "PostToolUse"}},
		{Timestamp: now.Add(-13 * 24 * time.Hour), Action: "hook_violation", Metadata: map[string]string{"rule": "use-targ", "hook": "PostToolUse"}},
		{Timestamp: now.Add(-11 * 24 * time.Hour), Action: "hook_violation", Metadata: map[string]string{"rule": "use-targ", "hook": "PostToolUse"}},
		{Timestamp: now.Add(-9 * 24 * time.Hour), Action: "hook_violation", Metadata: map[string]string{"rule": "use-targ", "hook": "PostToolUse"}},
		{Timestamp: now.Add(-8 * 24 * time.Hour), Action: "hook_violation", Metadata: map[string]string{"rule": "use-targ", "hook": "PostToolUse"}},
		// Recent period (last 7 days): 5 violations
		{Timestamp: now.Add(-6 * 24 * time.Hour), Action: "hook_violation", Metadata: map[string]string{"rule": "use-targ", "hook": "PostToolUse"}},
		{Timestamp: now.Add(-5 * 24 * time.Hour), Action: "hook_violation", Metadata: map[string]string{"rule": "use-targ", "hook": "PostToolUse"}},
		{Timestamp: now.Add(-4 * 24 * time.Hour), Action: "hook_violation", Metadata: map[string]string{"rule": "use-targ", "hook": "PostToolUse"}},
		{Timestamp: now.Add(-2 * 24 * time.Hour), Action: "hook_violation", Metadata: map[string]string{"rule": "use-targ", "hook": "PostToolUse"}},
		{Timestamp: now.Add(-1 * 24 * time.Hour), Action: "hook_violation", Metadata: map[string]string{"rule": "use-targ", "hook": "PostToolUse"}},
	}

	trends := memory.ComputeViolationTrends(violations, periodDays)

	if len(trends) != 1 {
		t.Fatalf("expected 1 trend, got %d", len(trends))
	}

	trend, ok := trends["use-targ"]
	if !ok {
		t.Fatal("expected trend for 'use-targ'")
	}

	if trend.Trending != "stable" {
		t.Errorf("expected trending='stable', got %q", trend.Trending)
	}

	if trend.TotalViolations != 10 {
		t.Errorf("expected total_violations=10, got %d", trend.TotalViolations)
	}
}

func TestComputeViolationTrends_IncreasingViolations(t *testing.T) {
	now := time.Now()
	periodDays := 7

	// Create violations: 2 in old period, 8 in recent period (degrading)
	violations := []memory.ChangelogEntry{
		// Old period (14-7 days ago): 2 violations
		{Timestamp: now.Add(-12 * 24 * time.Hour), Action: "hook_violation", Metadata: map[string]string{"rule": "check-claudemd-lines", "hook": "Stop"}},
		{Timestamp: now.Add(-10 * 24 * time.Hour), Action: "hook_violation", Metadata: map[string]string{"rule": "check-claudemd-lines", "hook": "Stop"}},
		// Recent period (last 7 days): 8 violations
		{Timestamp: now.Add(-6 * 24 * time.Hour), Action: "hook_violation", Metadata: map[string]string{"rule": "check-claudemd-lines", "hook": "Stop"}},
		{Timestamp: now.Add(-6 * 24 * time.Hour), Action: "hook_violation", Metadata: map[string]string{"rule": "check-claudemd-lines", "hook": "Stop"}},
		{Timestamp: now.Add(-5 * 24 * time.Hour), Action: "hook_violation", Metadata: map[string]string{"rule": "check-claudemd-lines", "hook": "Stop"}},
		{Timestamp: now.Add(-4 * 24 * time.Hour), Action: "hook_violation", Metadata: map[string]string{"rule": "check-claudemd-lines", "hook": "Stop"}},
		{Timestamp: now.Add(-3 * 24 * time.Hour), Action: "hook_violation", Metadata: map[string]string{"rule": "check-claudemd-lines", "hook": "Stop"}},
		{Timestamp: now.Add(-2 * 24 * time.Hour), Action: "hook_violation", Metadata: map[string]string{"rule": "check-claudemd-lines", "hook": "Stop"}},
		{Timestamp: now.Add(-1 * 24 * time.Hour), Action: "hook_violation", Metadata: map[string]string{"rule": "check-claudemd-lines", "hook": "Stop"}},
		{Timestamp: now.Add(-1 * 24 * time.Hour), Action: "hook_violation", Metadata: map[string]string{"rule": "check-claudemd-lines", "hook": "Stop"}},
	}

	trends := memory.ComputeViolationTrends(violations, periodDays)

	if len(trends) != 1 {
		t.Fatalf("expected 1 trend, got %d", len(trends))
	}

	trend, ok := trends["check-claudemd-lines"]
	if !ok {
		t.Fatal("expected trend for 'check-claudemd-lines'")
	}

	if trend.Trending != "degrading" {
		t.Errorf("expected trending='degrading', got %q", trend.Trending)
	}

	if trend.TotalViolations != 10 {
		t.Errorf("expected total_violations=10, got %d", trend.TotalViolations)
	}
}

func TestComputeViolationTrends_NoViolations(t *testing.T) {
	violations := []memory.ChangelogEntry{}
	periodDays := 7

	trends := memory.ComputeViolationTrends(violations, periodDays)

	if len(trends) != 0 {
		t.Errorf("expected empty map, got %d trends", len(trends))
	}
}

func TestComputeViolationTrends_SingleViolation(t *testing.T) {
	now := time.Now()
	violations := []memory.ChangelogEntry{
		{Timestamp: now.Add(-2 * 24 * time.Hour), Action: "hook_violation", Metadata: map[string]string{"rule": "single-rule", "hook": "PreToolUse"}},
	}
	periodDays := 7

	trends := memory.ComputeViolationTrends(violations, periodDays)

	if len(trends) != 1 {
		t.Fatalf("expected 1 trend, got %d", len(trends))
	}

	trend, ok := trends["single-rule"]
	if !ok {
		t.Fatal("expected trend for 'single-rule'")
	}

	if trend.Trending != "stable" {
		t.Errorf("expected trending='stable' for single violation, got %q", trend.Trending)
	}

	if trend.TotalViolations != 1 {
		t.Errorf("expected total_violations=1, got %d", trend.TotalViolations)
	}
}
