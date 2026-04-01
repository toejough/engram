package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"engram/internal/maintain"
	"engram/internal/memory"
	"engram/internal/policy"
	"engram/internal/tomlwriter"
)

// unexported variables.
var (
	errApplyProposalMissingID  = errors.New("apply-proposal: --id required")
	errProposalNotFound        = errors.New("proposal not found")
	errRejectProposalMissingID = errors.New("reject-proposal: --id required")
)

// applyFieldUpdate sets a SBIA field on a memory record by name.
func applyFieldUpdate(record *memory.MemoryRecord, field, value string) {
	switch field {
	case "situation":
		record.Situation = value
	case "behavior":
		record.Behavior = value
	case "impact":
		record.Impact = value
	case "action":
		record.Action = value
	}

	record.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
}

// executeProposal executes a single proposal action against the filesystem.
func executeProposal(proposal maintain.Proposal) error {
	switch proposal.Action {
	case maintain.ActionDelete:
		removeErr := os.Remove(proposal.Target)
		if removeErr != nil {
			return fmt.Errorf("deleting memory %s: %w", proposal.Target, removeErr)
		}
	case maintain.ActionUpdate:
		if filepath.Base(proposal.Target) == "policy.toml" {
			return nil // informational — no-op
		}

		modifier := memory.NewModifier(
			memory.WithModifierWriter(tomlwriter.New()),
		)

		modErr := modifier.ReadModifyWrite(proposal.Target, func(record *memory.MemoryRecord) {
			applyFieldUpdate(record, proposal.Field, proposal.Value)
		})
		if modErr != nil {
			return fmt.Errorf("updating memory %s: %w", proposal.Target, modErr)
		}
	case maintain.ActionMerge:
		return nil // handled via skill — no-op
	case maintain.ActionRecommend:
		return nil // informational — no-op
	}

	return nil
}

// findAndRemoveProposal finds a proposal by ID and returns it along with the remaining list.
// Returns nil proposal if not found.
func findAndRemoveProposal(
	proposals []maintain.Proposal,
	id string,
) (*maintain.Proposal, []maintain.Proposal) {
	for idx, proposal := range proposals {
		if proposal.ID == id {
			found := proposal
			remaining := make([]maintain.Proposal, 0, len(proposals)-1)
			remaining = append(remaining, proposals[:idx]...)
			remaining = append(remaining, proposals[idx+1:]...)

			return &found, remaining
		}
	}

	return nil, proposals
}

// runApplyProposal applies a single proposal by ID and removes it from the pending file.
//
//nolint:funlen // CLI wiring: flag parsing + proposal lookup + execution + history
func runApplyProposal(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("apply-proposal", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")
	proposalID := fs.String("id", "", "proposal ID to apply")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("apply-proposal: %w", parseErr)
	}

	if *proposalID == "" {
		return errApplyProposalMissingID
	}

	defaultErr := applyDataDirDefault(dataDir)
	if defaultErr != nil {
		return fmt.Errorf("apply-proposal: %w", defaultErr)
	}

	proposalPath := filepath.Join(*dataDir, "pending-proposals.json")

	proposals, readErr := maintain.ReadProposals(proposalPath, os.ReadFile)
	if readErr != nil {
		return fmt.Errorf("apply-proposal: %w", readErr)
	}

	found, remaining := findAndRemoveProposal(proposals, *proposalID)
	if found == nil {
		return fmt.Errorf("apply-proposal: %w: %s", errProposalNotFound, *proposalID)
	}

	execErr := executeProposal(*found)
	if execErr != nil {
		return fmt.Errorf("apply-proposal: %w", execErr)
	}

	policyPath := filepath.Join(*dataDir, "policy.toml")

	historyEntry := policy.ChangeEntry{
		Action:    found.Action,
		Target:    found.Target,
		Field:     found.Field,
		NewValue:  found.Value,
		Status:    "approved",
		Rationale: found.Rationale,
		ChangedAt: time.Now().UTC().Format(time.RFC3339),
	}

	historyErr := policy.AppendChangeHistory(
		policyPath, historyEntry, os.ReadFile, writeFileAdapter,
	)
	if historyErr != nil {
		return fmt.Errorf("apply-proposal: recording history: %w", historyErr)
	}

	writeErr := maintain.WriteProposals(proposalPath, remaining, os.WriteFile)
	if writeErr != nil {
		return fmt.Errorf("apply-proposal: %w", writeErr)
	}

	_, _ = fmt.Fprintf(stdout, "applied proposal %s (%s on %s)\n",
		found.ID, found.Action, filepath.Base(found.Target))

	return nil
}

// runMaintain is the public entry point for the maintain command.
func runMaintain(args []string, stdout io.Writer) error {
	return runMaintainWith(args, stdout, nil)
}

// runMaintainWith generates maintenance proposals and writes them to pending-proposals.json.
// callerOverride injects a mock LLM caller for testing; pass nil to use the real Anthropic client.
//
//nolint:funlen // CLI wiring: sequential flag parsing + dependency setup
func runMaintainWith(args []string, stdout io.Writer, callerOverride CallerFunc) error {
	fs := flag.NewFlagSet("maintain", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("maintain: %w", parseErr)
	}

	defaultErr := applyDataDirDefault(dataDir)
	if defaultErr != nil {
		return fmt.Errorf("maintain: %w", defaultErr)
	}

	policyPath := filepath.Join(*dataDir, "policy.toml")

	pol, polErr := policy.LoadFromPath(policyPath)
	if polErr != nil {
		pol = policy.Defaults()
	}

	var caller CallerFunc

	if callerOverride != nil {
		caller = callerOverride
	} else {
		ctx := context.Background()
		token := resolveToken(ctx)

		if token != "" {
			caller = makeAnthropicCaller(token)
		}
	}

	changeHistory, _ := policy.ReadChangeHistory(policyPath, os.ReadFile)

	ctx := context.Background()

	proposals, runErr := maintain.Run(ctx, maintain.Config{
		Policy:        pol,
		DataDir:       *dataDir,
		Caller:        caller,
		ChangeHistory: changeHistory,
	})
	// Run may return both proposals and an error (e.g., decision tree succeeded
	// but Sonnet-dependent analyses failed). Write whatever proposals we got,
	// then surface the error.

	proposalPath := filepath.Join(*dataDir, "pending-proposals.json")

	writeErr := maintain.WriteProposals(proposalPath, proposals, os.WriteFile)
	if writeErr != nil {
		return fmt.Errorf("maintain: %w", writeErr)
	}

	encoded, encErr := json.Marshal(proposals)
	if encErr != nil {
		return fmt.Errorf("maintain: encoding proposals: %w", encErr)
	}

	_, _ = fmt.Fprintf(stdout, "%s\n", encoded)

	if runErr != nil {
		return fmt.Errorf("maintain: %w", runErr)
	}

	return nil
}

// runRejectProposal rejects a single proposal by ID and removes it from the pending file.
func runRejectProposal(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("reject-proposal", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")
	proposalID := fs.String("id", "", "proposal ID to reject")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("reject-proposal: %w", parseErr)
	}

	if *proposalID == "" {
		return errRejectProposalMissingID
	}

	defaultErr := applyDataDirDefault(dataDir)
	if defaultErr != nil {
		return fmt.Errorf("reject-proposal: %w", defaultErr)
	}

	proposalPath := filepath.Join(*dataDir, "pending-proposals.json")

	proposals, readErr := maintain.ReadProposals(proposalPath, os.ReadFile)
	if readErr != nil {
		return fmt.Errorf("reject-proposal: %w", readErr)
	}

	found, remaining := findAndRemoveProposal(proposals, *proposalID)
	if found == nil {
		return fmt.Errorf("reject-proposal: %w: %s", errProposalNotFound, *proposalID)
	}

	policyPath := filepath.Join(*dataDir, "policy.toml")

	historyEntry := policy.ChangeEntry{
		Action:    found.Action,
		Target:    found.Target,
		Field:     found.Field,
		NewValue:  found.Value,
		Status:    "rejected",
		Rationale: found.Rationale,
		ChangedAt: time.Now().UTC().Format(time.RFC3339),
	}

	historyErr := policy.AppendChangeHistory(
		policyPath, historyEntry, os.ReadFile, writeFileAdapter,
	)
	if historyErr != nil {
		return fmt.Errorf("reject-proposal: recording history: %w", historyErr)
	}

	writeErr := maintain.WriteProposals(proposalPath, remaining, os.WriteFile)
	if writeErr != nil {
		return fmt.Errorf("reject-proposal: %w", writeErr)
	}

	_, _ = fmt.Fprintf(stdout, "rejected proposal %s (%s on %s)\n",
		found.ID, found.Action, filepath.Base(found.Target))

	return nil
}

// writeFileAdapter bridges os.WriteFile (3-arg) to policy.WriteFileFunc (2-arg).
func writeFileAdapter(path string, data []byte) error {
	return os.WriteFile(path, data, filePerms) //nolint:wrapcheck // thin adapter
}
