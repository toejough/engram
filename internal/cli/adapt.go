package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"engram/internal/adapt"
	"engram/internal/memory"
	"engram/internal/policy"
	"engram/internal/retrieve"
)

// AdaptApproveWithSnapshot approves a policy and stores the corpus snapshot
// as the before-state for later effectiveness measurement.
func AdaptApproveWithSnapshot(
	policyPath, id string,
	snapshot adapt.CorpusSnapshot,
	stdout io.Writer,
) error {
	policyFile, err := policy.Load(policyPath)
	if err != nil {
		return fmt.Errorf("adapt: %w", err)
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)

	approveErr := policyFile.Approve(id, timestamp)
	if approveErr != nil {
		return fmt.Errorf("adapt: %w", approveErr)
	}

	// Find the approved policy and set before-snapshot
	for idx := range policyFile.Policies {
		if policyFile.Policies[idx].ID == id {
			policyFile.Policies[idx].Effectiveness.BeforeFollowRate = snapshot.FollowRate
			policyFile.Policies[idx].Effectiveness.BeforeIrrelevanceRatio = snapshot.IrrelevanceRatio
			policyFile.Policies[idx].Effectiveness.BeforeMeanEffectiveness = snapshot.MeanEffectiveness

			break
		}
	}

	saveErr := policy.Save(policyPath, policyFile)
	if saveErr != nil {
		return fmt.Errorf("adapt: %w", saveErr)
	}

	const percentMultiplier = 100.0

	_, _ = fmt.Fprintf(stdout, "[engram] Approved policy %s (snapshot: follow=%.0f%%, eff=%.1f)\n",
		id, snapshot.FollowRate*percentMultiplier, snapshot.MeanEffectiveness)

	return nil
}

// IncrementPolicySessions increments MeasuredSessions on all active policies.
// Called once per session. Errors silently ignored (fire-and-forget, ARCH-6).
func IncrementPolicySessions(policyPath string) {
	policyFile, err := policy.Load(policyPath)
	if err != nil {
		return
	}

	changed := false

	for idx := range policyFile.Policies {
		if policyFile.Policies[idx].Status == policy.StatusActive {
			policyFile.Policies[idx].Effectiveness.MeasuredSessions++
			changed = true
		}
	}

	if changed {
		_ = policy.Save(policyPath, policyFile)
	}
}

// RunAdapt implements the adapt subcommand.
func RunAdapt(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("adapt", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")
	status := fs.Bool("status", false, "show all policies")
	approve := fs.String("approve", "", "approve a policy by ID")
	reject := fs.String("reject", "", "reject a policy by ID")
	retire := fs.String("retire", "", "retire a policy by ID")
	dedup := fs.Bool("dedup", false, "remove duplicate proposed policies")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("adapt: %w", parseErr)
	}

	defaultErr := applyDataDirDefault(dataDir)
	if defaultErr != nil {
		return fmt.Errorf("adapt: %w", defaultErr)
	}

	policyPath := filepath.Join(*dataDir, "policy.toml")

	policyFile, err := policy.Load(policyPath)
	if err != nil {
		return fmt.Errorf("adapt: %w", err)
	}

	switch {
	case *approve != "":
		return adaptApproveWithCorpusSnapshot(policyPath, *approve, *dataDir, stdout)
	case *reject != "":
		return adaptReject(policyFile, policyPath, *reject, stdout)
	case *retire != "":
		return adaptRetire(policyFile, policyPath, *retire, stdout)
	case *dedup:
		return adaptDedup(policyFile, policyPath, stdout)
	default:
		*status = true
	}

	if *status {
		return adaptStatus(policyFile, stdout)
	}

	return nil
}

func adaptApproveWithCorpusSnapshot(policyPath, id, dataDir string, stdout io.Writer) error {
	allMemories, listErr := retrieve.New().ListMemories(context.Background(), dataDir)
	if listErr != nil && !errors.Is(listErr, os.ErrNotExist) {
		return fmt.Errorf("adapt: listing memories: %w", listErr)
	}

	if allMemories == nil {
		allMemories = make([]*memory.Stored, 0)
	}

	snapshot := adapt.ComputeCorpusSnapshot(allMemories)

	return AdaptApproveWithSnapshot(policyPath, id, snapshot, stdout)
}

func adaptDedup(policyFile *policy.File, path string, stdout io.Writer) error {
	removed := policyFile.DeduplicateProposed()
	if removed == 0 {
		_, _ = fmt.Fprintln(stdout, "[engram] No duplicate proposals found.")
		return nil
	}

	err := policy.Save(path, policyFile)
	if err != nil {
		return fmt.Errorf("adapt: saving after dedup: %w", err)
	}

	_, _ = fmt.Fprintf(stdout, "[engram] Removed %d duplicate proposals.\n", removed)

	return nil
}

func adaptReject(policyFile *policy.File, path, id string, stdout io.Writer) error {
	err := policyFile.Reject(id)
	if err != nil {
		return fmt.Errorf("adapt: %w", err)
	}

	err = policy.Save(path, policyFile)
	if err != nil {
		return fmt.Errorf("adapt: %w", err)
	}

	_, _ = fmt.Fprintf(stdout, "[engram] Rejected policy %s\n", id)

	return nil
}

func adaptRetire(policyFile *policy.File, path, id string, stdout io.Writer) error {
	err := policyFile.Retire(id)
	if err != nil {
		return fmt.Errorf("adapt: %w", err)
	}

	err = policy.Save(path, policyFile)
	if err != nil {
		return fmt.Errorf("adapt: %w", err)
	}

	_, _ = fmt.Fprintf(stdout, "[engram] Retired policy %s\n", id)

	return nil
}

func adaptStatus(policyFile *policy.File, stdout io.Writer) error {
	if len(policyFile.Policies) == 0 {
		_, _ = fmt.Fprintln(stdout, "[engram] No policies.")
		return nil
	}

	for _, pol := range policyFile.Policies {
		_, _ = fmt.Fprintf(stdout, "  %s [%s] %s — %s\n",
			pol.ID, pol.Dimension, string(pol.Status), pol.Directive)
	}

	return nil
}
