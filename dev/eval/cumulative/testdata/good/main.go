// Command notes is a GOOD scorer fixture for the cumulative-accumulation eval.
// It satisfies every name-agnostic ARCH detector while deliberately using
// non-vault vocabulary ("Repository", not "Store") so the scorer's
// name-agnosticism is actually exercised, not just its structure detection.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func main() {
	if err := run(os.Args[1:], fileRepository{path: dataPath()}, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Exported variables.
var (
	ErrNotFound = errors.New("note not found")
)

// Note is a single stored note.
type Note struct {
	ID   int      `json:"id"`
	Text string   `json:"text"`
	Tags []string `json:"tags"`
}

// Repository is the injected persistence boundary. The name is intentionally
// not "Store": DI is any injected persistence interface regardless of name.
type Repository interface {
	Load() ([]Note, error)
	Save(notes []Note) error
}

// unexported constants.
const (
	dirPerm  os.FileMode = 0o750
	filePerm os.FileMode = 0o600
)

type fileRepository struct{ path string }

func (r fileRepository) Load() ([]Note, error) {
	data, err := os.ReadFile(r.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("loading notes: %w", err)
	}

	var notes []Note
	if err := json.Unmarshal(data, &notes); err != nil {
		return nil, fmt.Errorf("decoding notes: %w", err)
	}

	return notes, nil
}

func (r fileRepository) Save(notes []Note) error {
	if err := os.MkdirAll(filepath.Dir(r.path), dirPerm); err != nil {
		return fmt.Errorf("creating data dir: %w", err)
	}

	data, err := json.Marshal(notes)
	if err != nil {
		return fmt.Errorf("encoding notes: %w", err)
	}

	// Atomic write: temp file + rename, so a crash mid-write can't corrupt data.
	tmp, err := os.CreateTemp(filepath.Dir(r.path), "notes-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}

	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Chmod(tmp.Name(), filePerm); err != nil {
		return fmt.Errorf("setting permissions: %w", err)
	}

	if err := os.Rename(tmp.Name(), r.path); err != nil {
		return fmt.Errorf("renaming temp file: %w", err)
	}

	return nil
}

type parsedArgs struct {
	positional []string
	tags       []string
	search     string
	asJSON     bool
}

type service struct{ repo Repository }

func (s service) add(text string, tags []string, out io.Writer) error {
	notes, err := s.repo.Load()
	if err != nil {
		return err
	}

	next := 1
	for _, n := range notes {
		if n.ID >= next {
			next = n.ID + 1
		}
	}

	notes = append(notes, Note{ID: next, Text: text, Tags: tags})
	if err := s.repo.Save(notes); err != nil {
		return err
	}

	fmt.Fprintf(out, "added %d\n", next)

	return nil
}

func (s service) edit(id int, text string, out io.Writer) error {
	notes, idx, err := s.find(id)
	if err != nil {
		return err
	}

	notes[idx].Text = text
	if err := s.repo.Save(notes); err != nil {
		return err
	}

	fmt.Fprintf(out, "edited %d\n", id)

	return nil
}

func (s service) find(id int) ([]Note, int, error) {
	notes, err := s.repo.Load()
	if err != nil {
		return nil, -1, err
	}

	for i, n := range notes {
		if n.ID == id {
			return notes, i, nil
		}
	}

	return notes, -1, fmt.Errorf("note %d: %w", id, ErrNotFound)
}

func (s service) get(id int, asJSON bool, out io.Writer) error {
	notes, idx, err := s.find(id)
	if err != nil {
		return err
	}

	return render(out, []Note{notes[idx]}, asJSON)
}

func (s service) list(tag, search string, asJSON bool, out io.Writer) error {
	notes, err := s.repo.Load()
	if err != nil {
		return err
	}

	filtered := make([]Note, 0, len(notes))
	for _, n := range notes {
		if matches(n, tag, search) {
			filtered = append(filtered, n)
		}
	}

	sort.Slice(filtered, func(i, j int) bool { return filtered[i].ID < filtered[j].ID })

	return render(out, filtered, asJSON)
}

func (s service) remove(id int, out io.Writer) error {
	notes, idx, err := s.find(id)
	if err != nil {
		return err
	}

	notes = append(notes[:idx], notes[idx+1:]...)
	if err := s.repo.Save(notes); err != nil {
		return err
	}

	fmt.Fprintf(out, "removed %d\n", id)

	return nil
}

func (s service) retag(id int, tag string, add bool, out io.Writer) error {
	notes, idx, err := s.find(id)
	if err != nil {
		return err
	}

	kept := notes[idx].Tags[:0:0]
	for _, t := range notes[idx].Tags {
		if !strings.EqualFold(t, tag) {
			kept = append(kept, t)
		}
	}

	if add {
		kept = append(kept, tag)
	}

	notes[idx].Tags = kept
	if err := s.repo.Save(notes); err != nil {
		return err
	}

	fmt.Fprintf(out, "ok %d\n", id)

	return nil
}

// colorEnabled honors NO_COLOR (any value, even empty) and only colors a TTY.
func colorEnabled() bool {
	if _, noColor := os.LookupEnv("NO_COLOR"); noColor {
		return false
	}

	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeCharDevice != 0
}

func dataPath() string {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".local", "share")
	}

	return filepath.Join(base, "notes", "notes.json")
}

func firstOr(xs []string, fallback string) string {
	if len(xs) > 0 {
		return xs[0]
	}

	return fallback
}

func hasTag(n Note, tag string) bool {
	for _, t := range n.Tags {
		if strings.EqualFold(t, tag) {
			return true
		}
	}

	return false
}

func matches(n Note, tag, search string) bool {
	if tag != "" && !hasTag(n, tag) {
		return false
	}

	if search != "" {
		q := strings.ToLower(search)
		if !strings.Contains(strings.ToLower(n.Text), q) && !hasTag(n, search) {
			return false
		}
	}

	return true
}

// parseArgs collects --tag/--search/--json from anywhere in the arg list so
// flags may appear before or after positionals.
func parseArgs(args []string) parsedArgs {
	out := parsedArgs{}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--tag":
			if i+1 < len(args) {
				i++
				out.tags = append(out.tags, args[i])
			}
		case "--search":
			if i+1 < len(args) {
				i++
				out.search = args[i]
			}
		case "--json":
			out.asJSON = true
		default:
			out.positional = append(out.positional, args[i])
		}
	}

	return out
}

func render(out io.Writer, notes []Note, asJSON bool) error {
	if asJSON {
		enc := json.NewEncoder(out)
		if err := enc.Encode(notes); err != nil {
			return fmt.Errorf("encoding output: %w", err)
		}

		return nil
	}

	for _, n := range notes {
		line := fmt.Sprintf("%d: %s", n.ID, n.Text)
		if len(n.Tags) > 0 {
			line += " [" + strings.Join(n.Tags, ",") + "]"
		}

		if colorEnabled() {
			line = "\x1b[36m" + line + "\x1b[0m"
		}

		fmt.Fprintln(out, line)
	}

	return nil
}

func run(args []string, repo Repository, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: notes <add|list|get|edit|tag|untag|rm>")
	}

	svc := service{repo: repo}
	pa := parseArgs(args[1:])

	switch args[0] {
	case "add":
		return svc.add(strings.Join(pa.positional, " "), pa.tags, out)
	case "list":
		return svc.list(firstOr(pa.tags, ""), pa.search, pa.asJSON, out)
	case "get":
		return withID(pa.positional, func(id int) error { return svc.get(id, pa.asJSON, out) })
	case "edit":
		return withID(pa.positional, func(id int) error {
			return svc.edit(id, strings.Join(pa.positional[1:], " "), out)
		})
	case "tag":
		return withID(pa.positional, func(id int) error { return svc.retag(id, pa.positional[1], true, out) })
	case "untag":
		return withID(pa.positional, func(id int) error { return svc.retag(id, pa.positional[1], false, out) })
	case "rm":
		return withID(pa.positional, func(id int) error { return svc.remove(id, out) })
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func withID(pos []string, fn func(int) error) error {
	if len(pos) == 0 {
		return errors.New("missing id")
	}

	id, err := strconv.Atoi(pos[0])
	if err != nil {
		return fmt.Errorf("bad id %q: %w", pos[0], err)
	}

	return fn(id)
}
