package server

import "sync"

// AgentRegistry tracks running agent loops. Thread-safe.
type AgentRegistry struct {
	mu     sync.Mutex
	agents []string
}

// NewAgentRegistry creates an empty registry.
func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{}
}

// Agents returns a copy of the currently registered agent names.
func (r *AgentRegistry) Agents() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := make([]string, len(r.agents))
	copy(result, r.agents)

	return result
}

// Deregister removes an agent name from the registry.
func (r *AgentRegistry) Deregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, a := range r.agents {
		if a == name {
			r.agents = append(r.agents[:i], r.agents[i+1:]...)

			return
		}
	}
}

// Register adds an agent name to the registry.
func (r *AgentRegistry) Register(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.agents = append(r.agents, name)
}
