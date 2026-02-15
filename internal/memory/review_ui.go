package memory

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
)

// formatProposal formats a maintenance proposal for display with color/emphasis.
// Returns a formatted string ready for terminal output.
func formatProposal(p MaintenanceProposal) string {
	var sb strings.Builder

	// Format tier name with capitalization
	tierName := formatTierName(p.Tier)
	sb.WriteString(fmt.Sprintf("[%s] ", tierName))

	// Add reason
	if p.Reason != "" {
		sb.WriteString(p.Reason)
		sb.WriteString(":\n")
	}

	// Add preview with indentation
	if p.Preview != "" {
		// Indent preview lines
		previewLines := strings.Split(p.Preview, "\n")
		for _, line := range previewLines {
			if line != "" {
				sb.WriteString("  ")
				sb.WriteString(line)
				sb.WriteString("\n")
			}
		}
	}

	// Add action prompt
	actionVerb := formatActionVerb(p.Action, p.Tier)
	sb.WriteString(fmt.Sprintf("  → %s [y/n/s]", actionVerb))

	return sb.String()
}

// formatTierName formats the tier name for display.
func formatTierName(tier string) string {
	switch tier {
	case "embeddings":
		return "Embeddings"
	case "skills":
		return "Skills"
	case "claude-md":
		return "CLAUDE.md"
	default:
		return tier
	}
}

// formatActionVerb converts an action into a question verb.
// For skills tier, "prune" actions use "Archive?" instead of "Prune?".
func formatActionVerb(action, tier string) string {
	// Special case: skills tier uses "Archive?" for prune actions
	if action == "prune" && tier == "skills" {
		return "Archive?"
	}

	switch action {
	case "prune":
		return "Prune?"
	case "decay":
		return "Decay?"
	case "consolidate":
		return "Apply?"
	case "split":
		return "Split?"
	case "promote":
		return "Promote?"
	case "demote":
		return "Demote?"
	case "rewrite":
		return "Rewrite?"
	case "add-rationale":
		return "Add rationale?"
	case "extract-examples":
		return "Extract?"
	default:
		return action + "?"
	}
}

// readResult holds the result of a non-blocking line read.
type readResult struct {
	line string
	err  error
}

// reviewProposal presents a maintenance proposal interactively and prompts for user decision.
// Returns true if approved (y), false if rejected (n) or skipped (s).
// Respects context cancellation (e.g. Ctrl-C) even while blocked on stdin.
func reviewProposal(p MaintenanceProposal, input io.Reader, output io.Writer) (bool, error) {
	return reviewProposalCtx(context.Background(), p, input, output)
}

// reviewProposalCtx is the context-aware implementation of reviewProposal.
func reviewProposalCtx(ctx context.Context, p MaintenanceProposal, input io.Reader, output io.Writer) (bool, error) {
	// Display formatted proposal
	formatted := formatProposal(p)
	fmt.Fprintln(output, formatted)

	// Create buffered reader for input
	reader := bufio.NewReader(input)

	for {
		// Read in a goroutine so we can select on context cancellation
		ch := make(chan readResult, 1)
		go func() {
			line, err := reader.ReadString('\n')
			ch <- readResult{line, err}
		}()

		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case res := <-ch:
			if res.err != nil {
				if res.err == io.EOF {
					return false, io.EOF
				}
				return false, res.err
			}

			response := strings.TrimSpace(strings.ToLower(res.line))

			switch response {
			case "y", "yes":
				return true, nil
			case "n", "no":
				return false, nil
			case "s", "skip":
				return false, nil
			case "":
				continue
			default:
				fmt.Fprintln(output, "Invalid input. Please enter y (yes), n (no), or s (skip).")
				continue
			}
		}
	}
}
