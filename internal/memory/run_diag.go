package memory

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"
)

// RunDiag runs memory system diagnostics.
func RunDiag(args DiagArgs, homeDir string) error {
	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		memoryRoot = filepath.Join(homeDir, ".claude", "memory")
	}

	fmt.Println("Memory system diagnostics")
	fmt.Println("\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500")

	t := time.Now()
	extractor := NewLLMExtractor()

	d := time.Since(t)
	if extractor == nil {
		fmt.Printf("  keychain auth:  FAIL (%dms)\n", d.Milliseconds())
		return errors.New("keychain auth failed \u2014 cannot test API")
	}

	fmt.Printf("  keychain auth:  ok (%dms)\n", d.Milliseconds())

	t = time.Now()
	obs, err := extractor.Extract(context.Background(), "Always use targ instead of mage for building")

	d = time.Since(t)
	if err != nil {
		fmt.Printf("  API Extract:    FAIL (%dms) %v\n", d.Milliseconds(), err)
	} else if obs == nil {
		fmt.Printf("  API Extract:    FAIL (%dms) nil observation\n", d.Milliseconds())
	} else {
		fmt.Printf("  API Extract:    ok (%dms) type=%s\n", d.Milliseconds(), obs.Type)
	}

	t = time.Now()
	decision, err := extractor.Decide(context.Background(), "Use targ for builds", []ExistingMemory{
		{ID: 1, Content: "Always use targ instead of mage", Similarity: 0.85},
	})

	d = time.Since(t)
	if err != nil {
		fmt.Printf("  API Decide:     FAIL (%dms) %v\n", d.Milliseconds(), err)
	} else if decision == nil {
		fmt.Printf("  API Decide:     FAIL (%dms) nil decision\n", d.Milliseconds())
	} else {
		fmt.Printf("  API Decide:     ok (%dms) action=%s\n", d.Milliseconds(), decision.Action)
	}

	t = time.Now()
	db, err := InitEmbeddingsDB(memoryRoot)

	d = time.Since(t)
	if err != nil {
		fmt.Printf("  DB open:        FAIL (%dms) %v\n", d.Milliseconds(), err)
	} else {
		fmt.Printf("  DB open:        ok (%dms)\n", d.Milliseconds())

		_ = db.Close()
	}

	t = time.Now()
	err = Learn(LearnOpts{
		Message:    "__diag_timing_test__",
		MemoryRoot: memoryRoot,
		Extractor:  extractor,
	})

	d = time.Since(t)
	if err != nil {
		fmt.Printf("  Learn (full):   FAIL (%dms) %v\n", d.Milliseconds(), err)
	} else {
		fmt.Printf("  Learn (full):   ok (%dms)\n", d.Milliseconds())
	}

	return nil
}
