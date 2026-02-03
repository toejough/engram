package main

import (
	"fmt"
	"strings"

	"github.com/toejough/projctl/internal/yield"
)

type yieldValidateArgs struct {
	Path string `targ:"flag,short=p,required,desc=Path to yield TOML file"`
}

func yieldValidate(args yieldValidateArgs) error {
	result, err := yield.Validate(args.Path)
	if err != nil {
		return err
	}

	if result.Valid {
		fmt.Println("✓ Valid yield file")
		return nil
	}

	fmt.Println("✗ Invalid yield file:")
	for _, e := range result.Errors {
		fmt.Printf("  - %s\n", e)
	}

	return fmt.Errorf("validation failed: %d errors", len(result.Errors))
}

type yieldTypesArgs struct{}

func yieldTypes(args yieldTypesArgs) error {
	fmt.Println("Producer yield types:")
	for _, t := range yield.ValidProducerTypes {
		fmt.Printf("  - %s\n", t)
	}
	fmt.Println()
	fmt.Println("QA yield types:")
	for _, t := range yield.ValidQATypes {
		fmt.Printf("  - %s\n", t)
	}
	fmt.Println()
	fmt.Println("Resumable types (require [context] section):")
	fmt.Printf("  %s\n", strings.Join(yield.ResumableTypes, ", "))
	return nil
}
