// Package territory generates compressed codebase territory maps.
package territory

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Map represents a compressed territory map of a codebase.
type Map struct {
	Structure   Structure   `toml:"structure"`
	EntryPoints EntryPoints `toml:"entry_points"`
	Packages    Packages    `toml:"packages"`
	Tests       Tests       `toml:"tests"`
	Docs        Docs        `toml:"docs"`
}

// Structure describes the project structure.
type Structure struct {
	Root          string   `toml:"root"`
	Languages     []string `toml:"languages"`
	BuildTool     string   `toml:"build_tool"`
	TestFramework string   `toml:"test_framework"`
}

// EntryPoints identifies where execution begins.
type EntryPoints struct {
	CLI       string `toml:"cli"`
	PublicAPI string `toml:"public_api"`
}

// Packages summarizes the package structure.
type Packages struct {
	Count    int      `toml:"count"`
	Internal []string `toml:"internal"`
}

// Tests describes the test structure.
type Tests struct {
	Pattern string `toml:"pattern"`
	Count   int    `toml:"count"`
}

// Docs describes documentation files.
type Docs struct {
	Readme    string   `toml:"readme"`
	Artifacts []string `toml:"artifacts"`
}

// Generate creates a territory map for the given directory.
func Generate(dir string) (Map, error) {
	m := Map{
		Structure: Structure{
			Root: dir,
		},
		Packages: Packages{
			Internal: []string{},
		},
		Docs: Docs{
			Artifacts: []string{},
		},
	}

	// Detect languages and build tools
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
		m.Structure.Languages = append(m.Structure.Languages, "go")
		m.Structure.BuildTool = "go"
		m.Structure.TestFramework = "go test"
		m.Tests.Pattern = "*_test.go"
	}

	if _, err := os.Stat(filepath.Join(dir, "package.json")); err == nil {
		m.Structure.Languages = append(m.Structure.Languages, "javascript")
		m.Structure.BuildTool = "npm"
	}

	// Find CLI entry points
	cmdDir := filepath.Join(dir, "cmd")
	if entries, err := os.ReadDir(cmdDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				m.EntryPoints.CLI = filepath.Join("cmd", e.Name())
				break
			}
		}
	}

	// Find public API (root-level .go files)
	if entries, err := os.ReadDir(dir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") && !strings.HasSuffix(e.Name(), "_test.go") {
				m.EntryPoints.PublicAPI = e.Name()
				break
			}
		}
	}

	// Count internal packages
	internalDir := filepath.Join(dir, "internal")
	if entries, err := os.ReadDir(internalDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				m.Packages.Internal = append(m.Packages.Internal, e.Name())
				m.Packages.Count++
			}
		}
	}

	// Count test files
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") {
			m.Tests.Count++
		}
		return nil
	})

	// Find docs
	if _, err := os.Stat(filepath.Join(dir, "README.md")); err == nil {
		m.Docs.Readme = "README.md"
	}

	docsDir := filepath.Join(dir, "docs")
	if entries, err := os.ReadDir(docsDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				m.Docs.Artifacts = append(m.Docs.Artifacts, filepath.Join("docs", e.Name()))
			}
		}
	}

	return m, nil
}

// Marshal converts a Map to TOML bytes.
func Marshal(m Map) ([]byte, error) {
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(m); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
