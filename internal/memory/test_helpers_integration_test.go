//go:build integration

package memory_test

import (
	"context"
	"fmt"

	"github.com/toejough/projctl/internal/memory"
)

// mockSkillCompiler provides test implementation for skill compilation.
type mockSkillCompiler struct {
	compileFunc    func(ctx context.Context, theme string, memories []string) (string, error)
	synthesizeFunc func(ctx context.Context, memories []string) (string, error)
}

func (m *mockSkillCompiler) CompileSkill(ctx context.Context, theme string, memories []string) (string, error) {
	if m.compileFunc != nil {
		return m.compileFunc(ctx, theme, memories)
	}
	return "", fmt.Errorf("LLM unavailable")
}

func (m *mockSkillCompiler) Decide(newMessage string, existing []memory.ExistingMemory) (*memory.IngestDecision, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockSkillCompiler) Extract(message string) (*memory.Observation, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockSkillCompiler) Synthesize(ctx context.Context, memories []string) (string, error) {
	if m.synthesizeFunc != nil {
		return m.synthesizeFunc(ctx, memories)
	}
	return "", fmt.Errorf("not implemented")
}
