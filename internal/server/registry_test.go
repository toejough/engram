package server_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/server"
)

func TestAgentRegistry_DeregisterNonexistent_NoOp(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	registry := server.NewAgentRegistry()
	registry.Register("agent-a")
	registry.Deregister("nonexistent")

	g.Expect(registry.Agents()).To(Equal([]string{"agent-a"}))
}

func TestAgentRegistry_DeregisterRemovesAgent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	registry := server.NewAgentRegistry()
	registry.Register("engram-agent")
	registry.Deregister("engram-agent")

	g.Expect(registry.Agents()).To(BeEmpty())
}

func TestAgentRegistry_EmptyByDefault(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	registry := server.NewAgentRegistry()

	g.Expect(registry.Agents()).To(BeEmpty())
}

func TestAgentRegistry_MultipleAgents(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	registry := server.NewAgentRegistry()
	registry.Register("agent-a")
	registry.Register("agent-b")

	agents := registry.Agents()
	g.Expect(agents).To(HaveLen(2))
	g.Expect(agents).To(ContainElement("agent-a"))
	g.Expect(agents).To(ContainElement("agent-b"))
}

func TestAgentRegistry_RegisterAndList(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	registry := server.NewAgentRegistry()
	registry.Register("engram-agent")

	g.Expect(registry.Agents()).To(Equal([]string{"engram-agent"}))
}
