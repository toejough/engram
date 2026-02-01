package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/toejough/projctl/internal/refactor"
)

type refactorRenameArgs struct {
	Dir    string `targ:"flag,short=d,required,desc=Directory containing Go code"`
	Symbol string `targ:"flag,short=s,required,desc=Symbol to rename"`
	To     string `targ:"flag,short=t,required,desc=New name for the symbol"`
}

func refactorRename(args refactorRenameArgs) error {
	opts := refactor.RenameOpts{
		Dir:    args.Dir,
		Symbol: args.Symbol,
		To:     args.To,
	}

	result, err := refactor.Rename(opts)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "symbol not found") {
			fmt.Fprintf(os.Stderr, "Error: symbol not found: %s\n", args.Symbol)
			os.Exit(1)
		}
		if strings.Contains(errStr, "conflict") {
			fmt.Fprintf(os.Stderr, "Error: conflict: symbol %q already exists\n", args.To)
			os.Exit(1)
		}

		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Renamed %s in %d files\n", args.Symbol, result.FilesChanged)
	return nil
}
