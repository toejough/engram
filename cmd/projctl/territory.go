package main

import (
	"fmt"
	"os"
	"time"

	"github.com/toejough/projctl/internal/territory"
)

type territoryMapArgs struct {
	Dir    string `targ:"flag,short=d,desc=Project directory (default: current)"`
	Output string `targ:"flag,short=o,desc=Output file path (default: stdout)"`
	Cached bool   `targ:"flag,short=c,desc=Use cached territory if available"`
	Force  bool   `targ:"flag,short=f,desc=Force regeneration (ignore cache)"`
}

func territoryMap(args territoryMapArgs) error {
	dir := args.Dir
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	var m territory.Map
	var err error
	var cacheHit bool

	if args.Cached && !args.Force {
		m, cacheHit, err = territory.LoadCached(dir, time.Now)
		if err != nil {
			return fmt.Errorf("failed to load territory map: %w", err)
		}
		if cacheHit {
			fmt.Fprintln(os.Stderr, "Using cached territory map")
		}
	} else {
		m, err = territory.Generate(dir)
		if err != nil {
			return fmt.Errorf("failed to generate territory map: %w", err)
		}
	}

	data, err := territory.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal territory map: %w", err)
	}

	if args.Output != "" {
		if err := os.WriteFile(args.Output, data, 0o644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Printf("Territory map written to: %s\n", args.Output)
		return nil
	}

	fmt.Print(string(data))
	return nil
}

type territoryShowArgs struct {
	Dir string `targ:"flag,short=d,desc=Project directory (default: current)"`
}

func territoryShow(args territoryShowArgs) error {
	dir := args.Dir
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	cached, err := territory.Show(dir)
	if err != nil {
		return err
	}

	data, err := territory.MarshalCached(cached)
	if err != nil {
		return fmt.Errorf("failed to marshal territory map: %w", err)
	}

	fmt.Print(string(data))
	return nil
}
