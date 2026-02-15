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

	if p.LLMEval != nil {
		// LLM-evaluated format
		sb.WriteString(fmt.Sprintf("\n━━━ Proposed Change: %s ━━━\n", formatActionExplanation(p.Action, p.Tier)))

		// Why proposed
		sb.WriteString(fmt.Sprintf("  Why proposed: %s\n", p.Reason))

		// What changes
		if p.Preview != "" {
			sb.WriteString("\n  What changes:\n")
			previewLines := strings.Split(p.Preview, "\n")
			for _, line := range previewLines {
				if line != "" {
					sb.WriteString("  │ ")
					sb.WriteString(line)
					sb.WriteString("\n")
				}
			}
		}

		// Haiku validation
		sb.WriteString("\n")
		if p.LLMEval.HaikuValid {
			sb.WriteString(fmt.Sprintf("  Haiku: Valid concern — %s\n", p.LLMEval.HaikuRationale))
		} else {
			sb.WriteString(fmt.Sprintf("  Haiku: Invalid concern — %s\n", p.LLMEval.HaikuRationale))
		}

		// Sonnet recommendation
		recommendAction := "Apply"
		if p.LLMEval.SonnetRecommend == "skip" {
			recommendAction = "Skip"
		}
		sb.WriteString(fmt.Sprintf("  Sonnet recommends: %s this change (%s confidence)\n", recommendAction, p.LLMEval.SonnetConfidence))

		// Scenario results (indented under Sonnet section)
		if len(p.LLMEval.ScenarioResults) > 0 {
			for _, scenario := range p.LLMEval.ScenarioResults {
				if scenario.Preserved {
					sb.WriteString(fmt.Sprintf("    ✓ \"%s\" → guidance preserved\n", scenario.Prompt))
				} else {
					sb.WriteString(fmt.Sprintf("    ✗ \"%s\" → %s\n", scenario.Prompt, scenario.Lost))
				}
			}
		}

		// Summary (labeled, part of Sonnet section)
		if p.LLMEval.SonnetSummary != "" {
			sb.WriteString(fmt.Sprintf("    Summary: %s\n", p.LLMEval.SonnetSummary))
		}

		// Prompt
		sb.WriteString("\n  [a]pply change / [s]kip change")

	} else {
		// Non-LLM-evaluated format
		sb.WriteString(fmt.Sprintf("\n━━━ Proposed Change: %s ━━━\n", formatActionExplanation(p.Action, p.Tier)))

		// Why proposed
		sb.WriteString(fmt.Sprintf("  Why proposed: %s\n", p.Reason))

		// Content
		if p.Preview != "" {
			sb.WriteString("\n  What changes:\n")
			previewLines := strings.Split(p.Preview, "\n")
			for _, line := range previewLines {
				if line != "" {
					sb.WriteString("  │ ")
					sb.WriteString(line)
					sb.WriteString("\n")
				}
			}
		}

		// Show LLM eval status for actions that should have been triaged
		if needsLLMTriage(p.Action) {
			sb.WriteString("\n  Haiku: not evaluated (triage failed or skipped)\n")
			sb.WriteString("  Sonnet: not evaluated\n")
		}

		// Prompt
		sb.WriteString("\n  [a]pply / [s]kip")
	}

	return sb.String()
}

// formatActionExplanation returns a human-readable explanation of what the action does.
func formatActionExplanation(action, tier string) string {
	switch action {
	case "prune":
		if tier == "skills" {
			return "Move this skill to archive (stops retrieval, keeps file for reference)"
		}
		if tier == "claude-md" {
			return "Remove this line from CLAUDE.md"
		}
		return "Delete this memory entry permanently (archived for recovery)"
	case "decay":
		return "Reduce confidence score (makes it less likely to be retrieved)"
	case "consolidate":
		if tier == "claude-md" {
			return "Merge duplicate CLAUDE.md entries into one"
		}
		return "Delete the second entry, keep the first (they appear to be duplicates)"
	case "split":
		return "Break this multi-topic entry into separate focused entries"
	case "promote":
		if tier == "embeddings" {
			return "Create a new skill from this high-value memory"
		}
		if tier == "skills" {
			return "Add this skill's content to CLAUDE.md (always-loaded)"
		}
		return "Move to a higher tier for more visibility"
	case "demote":
		if tier == "claude-md" {
			return "Remove from CLAUDE.md (too narrow for always-loaded tier)"
		}
		return "Move to a lower tier"
	case "rewrite":
		return "Replace content with an LLM-improved version for clarity"
	case "add-rationale":
		return "Append an explanation of WHY this rule matters"
	case "extract-examples":
		return "Pull out concrete examples into separate entries"
	default:
		return ""
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
			case "y", "yes", "a", "apply":
				return true, nil
			case "n", "no":
				return false, nil
			case "s", "skip":
				return false, nil
			case "":
				continue
			default:
				fmt.Fprintln(output, "Invalid input. Please enter a (apply) or s (skip).")
				continue
			}
		}
	}
}
