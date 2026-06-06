// Command notes is a NAIVE scorer fixture: it builds and runs basic add/list,
// but deliberately violates the architecture conventions (global mutable state,
// in-place writes with bare permissions, no sentinel errors, no DI, no JSON
// mode, no NO_COLOR handling). It exists so the scorer's good-vs-naive
// separation can be validated without spending on an LLM.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Note is a stored note.
type Note struct {
	ID   int
	Text string
	Tags []string
}

// notes is package-level mutable state (an anti-pattern the scorer flags).
var notes []Note

func dataPath() string {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		base = filepath.Join(os.Getenv("HOME"), ".local", "share")
	}

	return filepath.Join(base, "naive-notes.json")
}

func load() {
	data, err := os.ReadFile(dataPath())
	if err != nil {
		return
	}

	_ = json.Unmarshal(data, &notes)
}

func save() error {
	data, _ := json.Marshal(notes)
	_ = os.MkdirAll(filepath.Dir(dataPath()), 0o755)

	// In-place write with a bare permission literal — not crash-safe.
	if err := os.WriteFile(dataPath(), data, 0o644); err != nil {
		return fmt.Errorf("could not save: %v", err)
	}

	return nil
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		fmt.Println("usage: notes <add|list>")

		return
	}

	load()

	switch args[0] {
	case "add":
		var text, tags []string

		for i := 1; i < len(args); i++ {
			if args[i] == "--tag" && i+1 < len(args) {
				i++
				tags = append(tags, args[i])
			} else {
				text = append(text, args[i])
			}
		}

		notes = append(notes, Note{ID: len(notes) + 1, Text: strings.Join(text, " "), Tags: tags})
		if err := save(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		fmt.Println("added")
	case "list":
		for _, n := range notes {
			fmt.Printf("%d: %s\n", n.ID, n.Text)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", args[0])
		os.Exit(1)
	}
}
