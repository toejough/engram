package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"

	"engram/internal/signal"
)

// IssueCreator creates GitHub issues (injected interface for DI).
type IssueCreator interface {
	Create(title, body string) (issueURL string, err error)
}

// unexported constants.
const (
	percentageFactor = 100.0
)

// unexported variables.
var (
	errGraduateAcceptMissingFlags  = errors.New("graduate accept: --data-dir and --id required")
	errGraduateAcceptNotFound      = errors.New("graduate accept: entry not found")
	errGraduateDismissMissingFlags = errors.New("graduate dismiss: --data-dir and --id required")
	errGraduateListMissingFlags    = errors.New("graduate list: --data-dir required")
	errGraduateSubcmdRequired      = errors.New("graduate: subcommand required (accept, dismiss, list)")
	errGraduateSurfaceMissingFlags = errors.New("graduate-surface: --data-dir required")
	errGraduateUnknownSubcmd       = errors.New("graduate: unknown subcommand")
)

// ghIssueCreator creates GitHub issues by shelling out to the gh CLI.
type ghIssueCreator struct{}

func (g *ghIssueCreator) Create(title, body string) (string, error) {
	//nolint:gosec // gh CLI arguments are user-provided issue content, not shell injection
	out, err := exec.CommandContext(
		context.Background(), "gh", "issue", "create", "--title", title, "--body", body,
	).Output()
	if err != nil {
		return "", fmt.Errorf("gh issue create: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}

// calcQuality returns the quality metric string.
func calcQuality(accepted, dismissed int) string {
	if accepted+dismissed == 0 {
		return "n/a"
	}

	percentage := float64(accepted) * percentageFactor / float64(accepted+dismissed)

	return fmt.Sprintf("%.1f%%", percentage)
}

// classifyGraduationEntries partitions entries by status.
func classifyGraduationEntries(
	entries []signal.GraduationEntry,
) (pending []signal.GraduationEntry, accepted, dismissed int) {
	pending = make([]signal.GraduationEntry, 0)

	for _, entry := range entries {
		switch entry.Status {
		case "pending":
			pending = append(pending, entry)
		case "accepted":
			accepted++
		case "dismissed":
			dismissed++
		}
	}

	return pending, accepted, dismissed
}

// findGraduationEntry looks up an entry by ID in the store.
func findGraduationEntry(store *signal.GraduationStore, queuePath, id string) (*signal.GraduationEntry, error) {
	entries, err := store.List(queuePath)
	if err != nil {
		return nil, fmt.Errorf("reading queue: %w", err)
	}

	for i := range entries {
		if entries[i].ID == id {
			return &entries[i], nil
		}
	}

	return nil, nil //nolint:nilnil // nil,nil signals "not found" per caller contract
}

func newGHIssueCreator() IssueCreator {
	return &ghIssueCreator{}
}

// printGraduateSurfaceJSON writes the JSON surface output for SessionStart hooks.
func printGraduateSurfaceJSON(stdout io.Writer, pending []signal.GraduationEntry) {
	_, _ = fmt.Fprintf(stdout, `{"summary":"[engram] %d pending graduation signal(s)","context":"`, len(pending))
	_, _ = fmt.Fprintf(stdout, "[engram] %d pending graduation signal(s):\\n\\n", len(pending))

	for _, entry := range pending {
		_, _ = fmt.Fprintf(stdout, "  - ID: %s | Memory: %s | Recommendation: %s | Detected: %s\\n",
			entry.ID, entry.MemoryPath, entry.Recommendation,
			entry.DetectedAt.Format("2006-01-02T15:04:05Z07:00"))
	}

	_, _ = fmt.Fprintf(stdout,
		"\\nAsk the user if they would like to create GitHub issues for each graduated memory."+
			"\\nCommands: engram graduate accept --data-dir <data-dir> --id <id> to accept,"+
			"\\n          engram graduate dismiss --data-dir <data-dir> --id <id> to dismiss.\"}")
	_, _ = fmt.Fprintf(stdout, "\n")
}

// printGraduationEntries prints one line per entry to stdout.
func printGraduationEntries(stdout io.Writer, entries []signal.GraduationEntry) {
	for _, entry := range entries {
		_, _ = fmt.Fprintf(stdout, "  ID:             %s\n", entry.ID)
		_, _ = fmt.Fprintf(stdout, "  Memory:         %s\n", entry.MemoryPath)
		_, _ = fmt.Fprintf(stdout, "  Recommendation: %s\n", entry.Recommendation)
		_, _ = fmt.Fprintf(stdout, "  Detected:       %s\n\n",
			entry.DetectedAt.Format("2006-01-02T15:04:05Z07:00"))
	}
}

// runGraduateAccept creates a GitHub issue and marks entry accepted.
func runGraduateAccept(args []string, stdout io.Writer, creator IssueCreator) error {
	fs := flag.NewFlagSet("graduate accept", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")
	id := fs.String("id", "", "graduation entry ID")

	err := fs.Parse(args)
	if err != nil {
		return fmt.Errorf("graduate accept: parsing flags: %w", err)
	}

	if *dataDir == "" || *id == "" {
		return errGraduateAcceptMissingFlags
	}

	queuePath := filepath.Join(*dataDir, "graduation-queue.jsonl")
	store := signal.NewGraduationStore()

	entry, err := findGraduationEntry(store, queuePath, *id)
	if err != nil {
		return fmt.Errorf("graduate accept: %w", err)
	}

	if entry == nil {
		return errGraduateAcceptNotFound
	}

	issueURL := ""

	if creator != nil {
		url, cerr := creator.Create(entry.MemoryPath, entry.Recommendation)
		if cerr != nil {
			return fmt.Errorf("graduate accept: creating issue: %w", cerr)
		}

		issueURL = url
	}

	now := signal.TimestampNow()

	err = store.SetStatus(queuePath, *id, "accepted", now, issueURL)
	if err != nil {
		return fmt.Errorf("graduate accept: updating status: %w", err)
	}

	_, _ = fmt.Fprintf(stdout, "Accepted: %s\nIssue: %s\n", *id, issueURL)

	return nil
}

// runGraduateCommand dispatches graduate subcommands (accept, dismiss, list).
func runGraduateCommand(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return errGraduateSubcmdRequired
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "accept":
		return runGraduateAccept(subArgs, stdout, newGHIssueCreator())
	case "dismiss":
		return runGraduateDismiss(subArgs, stdout)
	case "list":
		return runGraduateList(subArgs, stdout)
	default:
		return fmt.Errorf("%w: %s", errGraduateUnknownSubcmd, sub)
	}
}

// runGraduateDismiss marks entry dismissed.
func runGraduateDismiss(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("graduate dismiss", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")
	id := fs.String("id", "", "graduation entry ID")

	err := fs.Parse(args)
	if err != nil {
		return fmt.Errorf("graduate dismiss: parsing flags: %w", err)
	}

	if *dataDir == "" || *id == "" {
		return errGraduateDismissMissingFlags
	}

	queuePath := filepath.Join(*dataDir, "graduation-queue.jsonl")
	store := signal.NewGraduationStore()

	now := signal.TimestampNow()

	err = store.SetStatus(queuePath, *id, "dismissed", now, "")
	if err != nil {
		return fmt.Errorf("graduate dismiss: updating status: %w", err)
	}

	_, _ = fmt.Fprintf(stdout, "Dismissed: %s\n", *id)

	return nil
}

// runGraduateList lists pending graduation signals.
func runGraduateList(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("graduate list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")

	err := fs.Parse(args)
	if err != nil {
		return fmt.Errorf("graduate list: parsing flags: %w", err)
	}

	if *dataDir == "" {
		return errGraduateListMissingFlags
	}

	queuePath := filepath.Join(*dataDir, "graduation-queue.jsonl")
	store := signal.NewGraduationStore()

	entries, err := store.List(queuePath)
	if err != nil {
		return fmt.Errorf("graduate list: reading queue: %w", err)
	}

	pending, accepted, dismissed := classifyGraduationEntries(entries)

	if len(pending) == 0 && accepted == 0 && dismissed == 0 {
		_, _ = fmt.Fprintf(stdout, "No pending graduation signals.\n")
		return nil
	}

	_, _ = fmt.Fprintf(stdout, "Graduation Signals (%d pending)\n\n", len(pending))
	printGraduationEntries(stdout, pending)
	_, _ = fmt.Fprintf(stdout, "Quality: %s accepted (%d accepted, %d dismissed)\n",
		calcQuality(accepted, dismissed), accepted, dismissed)

	return nil
}

// runGraduateSurface reads pending entries and formats for SessionStart.
func runGraduateSurface(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("graduate-surface", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	dataDir := fs.String("data-dir", "", "path to data directory")
	format := fs.String("format", "text", "output format: text or json")

	err := fs.Parse(args)
	if err != nil {
		return fmt.Errorf("graduate-surface: parsing flags: %w", err)
	}

	if *dataDir == "" {
		return errGraduateSurfaceMissingFlags
	}

	queuePath := filepath.Join(*dataDir, "graduation-queue.jsonl")
	store := signal.NewGraduationStore()

	entries, err := store.List(queuePath)
	if err != nil {
		return fmt.Errorf("graduate-surface: reading queue: %w", err)
	}

	pending := make([]signal.GraduationEntry, 0)

	for _, entry := range entries {
		if entry.Status == "pending" {
			pending = append(pending, entry)
		}
	}

	if len(pending) == 0 {
		return nil
	}

	if *format == formatJSON {
		printGraduateSurfaceJSON(stdout, pending)
	} else {
		_, _ = fmt.Fprintf(stdout, "[engram] %d pending graduation signal(s):\n\n", len(pending))
		printGraduationEntries(stdout, pending)
	}

	return nil
}
