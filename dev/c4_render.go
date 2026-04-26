//go:build targ

package dev

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/toejough/targ"
)

func init() {
	targ.Register(targ.Targ(c4Render).Name("c4-render").
		Description("Render architecture/c4/svg/*.mmd to .svg via mmdc with ELK layout. " +
			"Skips files whose .svg is up-to-date. Requires npx (mermaid-cli runs via npx)."))
}

// C4RenderArgs configures the c4-render target.
type C4RenderArgs struct {
	Dir   string `targ:"flag,name=dir,desc=Directory containing .mmd sources (default architecture/c4/svg)"`
	Force bool   `targ:"flag,name=force,desc=Re-render even when .svg is newer than .mmd"`
}

// unexported constants.
const (
	defaultC4SVGDir = "architecture/c4/svg"
)

func c4Render(ctx context.Context, args C4RenderArgs) error {
	dir := args.Dir
	if dir == "" {
		dir = defaultC4SVGDir
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading %s: %w", dir, err)
	}

	mmdFiles := make([]string, 0, len(entries))

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".mmd") {
			continue
		}

		mmdFiles = append(mmdFiles, e.Name())
	}

	sort.Strings(mmdFiles)

	if len(mmdFiles) == 0 {
		targ.Print(ctx, fmt.Sprintf("No .mmd files found under %s.\n", dir))

		return nil
	}

	rendered := 0
	skipped := 0

	for _, mmd := range mmdFiles {
		mmdPath := filepath.Join(dir, mmd)
		svgPath := filepath.Join(dir, strings.TrimSuffix(mmd, ".mmd")+".svg")

		if !args.Force && svgIsFresh(mmdPath, svgPath) {
			skipped++

			continue
		}

		targ.Print(ctx, fmt.Sprintf("rendering %s -> %s\n", mmdPath, svgPath))

		if err := targ.RunContext(ctx, "npx", "--yes", "@mermaid-js/mermaid-cli@latest",
			"-i", mmdPath, "-o", svgPath); err != nil {
			return fmt.Errorf("rendering %s: %w", mmdPath, err)
		}

		rendered++
	}

	targ.Print(ctx, fmt.Sprintf("c4-render: %d rendered, %d up-to-date.\n", rendered, skipped))

	return nil
}

// svgIsFresh reports whether svgPath exists and is at least as new as mmdPath.
func svgIsFresh(mmdPath, svgPath string) bool {
	mmdStat, err := os.Stat(mmdPath)
	if err != nil {
		return false
	}

	svgStat, err := os.Stat(svgPath)
	if err != nil {
		return false
	}

	return !svgStat.ModTime().Before(mmdStat.ModTime())
}
