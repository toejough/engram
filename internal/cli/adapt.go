package cli

import (
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"engram/internal/policy"
)

// RunAdapt implements the adapt subcommand.
func RunAdapt(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("adapt", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")
	status := fs.Bool("status", false, "show all policies")
	approve := fs.String("approve", "", "approve a policy by ID")
	reject := fs.String("reject", "", "reject a policy by ID")
	retire := fs.String("retire", "", "retire a policy by ID")

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
		return adaptApprove(policyFile, policyPath, *approve, stdout)
	case *reject != "":
		return adaptReject(policyFile, policyPath, *reject, stdout)
	case *retire != "":
		return adaptRetire(policyFile, policyPath, *retire, stdout)
	default:
		*status = true
	}

	if *status {
		return adaptStatus(policyFile, stdout)
	}

	return nil
}

func adaptApprove(policyFile *policy.File, path, id string, stdout io.Writer) error {
	timestamp := time.Now().UTC().Format(time.RFC3339)

	err := policyFile.Approve(id, timestamp)
	if err != nil {
		return fmt.Errorf("adapt: %w", err)
	}

	err = policy.Save(path, policyFile)
	if err != nil {
		return fmt.Errorf("adapt: %w", err)
	}

	_, _ = fmt.Fprintf(stdout, "[engram] Approved policy %s\n", id)

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
