package territory

import (
	"fmt"
	"os"
	"time"
)

// MapArgs holds arguments for the territory map command.
type MapArgs struct {
	Dir    string `targ:"flag,short=d,desc=Project directory (default: current)"`
	Output string `targ:"flag,short=o,desc=Output file path (default: stdout)"`
	Cached bool   `targ:"flag,short=c,desc=Use cached territory if available"`
	Force  bool   `targ:"flag,short=f,desc=Force regeneration (ignore cache)"`
}

// ShowArgs holds arguments for the territory show command.
type ShowArgs struct {
	Dir string `targ:"flag,short=d,desc=Project directory (default: current)"`
}

// RunMap generates and outputs a territory map.
func RunMap(args MapArgs) error {
	dir := args.Dir
	if dir == "" {
		var err error

		dir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	var (
		m        Map
		err      error
		cacheHit bool
	)

	if args.Cached && !args.Force {
		m, cacheHit, err = LoadCached(dir, time.Now)
		if err != nil {
			return fmt.Errorf("failed to load territory map: %w", err)
		}

		if cacheHit {
			fmt.Fprintln(os.Stderr, "Using cached territory map")
		}
	} else {
		m, err = Generate(dir)
		if err != nil {
			return fmt.Errorf("failed to generate territory map: %w", err)
		}
	}

	data, err := Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal territory map: %w", err)
	}

	if args.Output != "" {
		err := os.WriteFile(args.Output, data, 0o644)
		if err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}

		fmt.Printf("Territory map written to: %s\n", args.Output)

		return nil
	}

	fmt.Print(string(data))

	return nil
}

// RunShow displays the current cached territory map.
func RunShow(args ShowArgs) error {
	dir := args.Dir
	if dir == "" {
		var err error

		dir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	cached, err := Show(dir)
	if err != nil {
		return err
	}

	data, err := MarshalCached(cached)
	if err != nil {
		return fmt.Errorf("failed to marshal territory map: %w", err)
	}

	fmt.Print(string(data))

	return nil
}
