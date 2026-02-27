package trace

// DetectCycle checks if the graph contains a cycle using DFS.
// Returns (hasCycle, cyclePath) where cyclePath contains IDs in the cycle.
func DetectCycle(graph *Graph) (bool, []string) {
	if graph == nil {
		return false, nil
	}

	visited := make(map[string]bool)
	inStack := make(map[string]bool)

	var cyclePath []string

	for nodeID := range graph.Nodes {
		if visited[nodeID] {
			continue
		}

		if hasCycle, path := dfs(graph, nodeID, visited, inStack, []string{}); hasCycle {
			return true, path
		}
	}

	return false, cyclePath
}

func dfs(graph *Graph, nodeID string, visited, inStack map[string]bool, path []string) (bool, []string) {
	visited[nodeID] = true
	inStack[nodeID] = true
	path = append(path, nodeID)

	for _, edge := range graph.Edges[nodeID] {
		if inStack[edge.To] {
			// Found a cycle - extract the cycle portion
			cycleStart := -1

			for i, id := range path {
				if id == edge.To {
					cycleStart = i
					break
				}
			}

			if cycleStart >= 0 {
				return true, path[cycleStart:]
			}
			// Self-loop case
			return true, []string{edge.To}
		}

		if !visited[edge.To] {
			if hasCycle, cyclePath := dfs(graph, edge.To, visited, inStack, path); hasCycle {
				return true, cyclePath
			}
		}
	}

	inStack[nodeID] = false

	return false, nil
}
