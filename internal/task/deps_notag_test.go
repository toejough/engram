package task_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/task"
)

func TestDependencyGraph_CyclePath_NilReceiver(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var graph *task.DependencyGraph

	g.Expect(graph.CyclePath()).To(BeNil())
}

func TestDependencyGraph_HasCycle_NilReceiver(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var graph *task.DependencyGraph

	g.Expect(graph.HasCycle()).To(BeFalse())
}

func TestDependencyGraph_Roots_NilReceiver(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	var graph *task.DependencyGraph

	g.Expect(graph.Roots()).To(BeNil())
}
