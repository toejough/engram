package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/toejough/projctl/internal/memory"
)

type memoryDiagArgs struct {
	MemoryRoot string `targ:"flag,name=memory-root,desc=Memory root directory (default: ~/.claude/memory)"`
}

// memoryDiag runs diagnostic timing checks on the memory system components.
func memoryDiag(args memoryDiagArgs) error {
	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		memoryRoot = filepath.Join(home, ".claude", "memory")
	}

	fmt.Println("Memory system diagnostics")
	fmt.Println("─────────────────────────")

	// 1. Keychain auth
	t := time.Now()
	extractor := memory.NewLLMExtractor()
	d := time.Since(t)
	if extractor == nil {
		fmt.Printf("  keychain auth:  FAIL (%dms)\n", d.Milliseconds())
		return fmt.Errorf("keychain auth failed — cannot test API")
	}
	fmt.Printf("  keychain auth:  ok (%dms)\n", d.Milliseconds())

	// 2. API round-trip (Extract)
	t = time.Now()
	obs, err := extractor.Extract(context.Background(), "Always use targ instead of mage for building")
	d = time.Since(t)
	if err != nil {
		fmt.Printf("  API Extract:    FAIL (%dms) %v\n", d.Milliseconds(), err)
	} else {
		fmt.Printf("  API Extract:    ok (%dms) type=%s\n", d.Milliseconds(), obs.Type)
	}

	// 3. API round-trip (Decide)
	t = time.Now()
	decision, err := extractor.Decide(context.Background(), "Use targ for builds", []memory.ExistingMemory{
		{ID: 1, Content: "Always use targ instead of mage", Similarity: 0.85},
	})
	d = time.Since(t)
	if err != nil {
		fmt.Printf("  API Decide:     FAIL (%dms) %v\n", d.Milliseconds(), err)
	} else {
		fmt.Printf("  API Decide:     ok (%dms) action=%s\n", d.Milliseconds(), decision.Action)
	}

	// 4. DB open
	t = time.Now()
	db, err := memory.InitEmbeddingsDB(memoryRoot)
	d = time.Since(t)
	if err != nil {
		fmt.Printf("  DB open:        FAIL (%dms) %v\n", d.Milliseconds(), err)
	} else {
		fmt.Printf("  DB open:        ok (%dms)\n", d.Milliseconds())
		_ = db.Close()
	}

	// 5. ONNX embedding
	t = time.Now()
	err = memory.Learn(memory.LearnOpts{
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
