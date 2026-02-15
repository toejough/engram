package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/toejough/projctl/internal/config"
)

// realConfigFS implements config.ConfigFS using the real file system.
type realConfigFS struct{}

func (r *realConfigFS) ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (r *realConfigFS) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

type configInitArgs struct {
	Dir string `targ:"flag,short=d,desc=Project directory (default: current)"`
}

func configInit(args configInitArgs) error {
	dir := args.Dir
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	configDir := filepath.Join(dir, ".claude")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create .claude directory: %w", err)
	}

	configPath := filepath.Join(configDir, "project-config.toml")
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("config already exists: %s", configPath)
	}

	cfg := config.Default()

	f, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer func() { _ = f.Close() }()

	if err := toml.NewEncoder(f).Encode(cfg); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Printf("Created %s\n", configPath)
	return nil
}

type configGetArgs struct {
	Dir string `targ:"flag,short=d,desc=Project directory (default: current)"`
	Key string `targ:"flag,short=k,desc=Config key to get (optional; returns all if not specified)"`
}

func configGet(args configGetArgs) error {
	dir := args.Dir
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	fs := &realConfigFS{}
	cfg, err := config.Load(dir, homeDir, fs)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if args.Key == "" {
		// Return all config as JSON
		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to encode config: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	// Return specific key
	switch args.Key {
	case "paths.docs_dir":
		fmt.Println(cfg.Paths.DocsDir)
	case "paths.readme":
		fmt.Println(cfg.Paths.Readme)
	case "paths.requirements":
		fmt.Println(cfg.Paths.Requirements)
	case "paths.design":
		fmt.Println(cfg.Paths.Design)
	case "paths.architecture":
		fmt.Println(cfg.Paths.Architecture)
	case "paths.tasks":
		fmt.Println(cfg.Paths.Tasks)
	case "paths.issues":
		fmt.Println(cfg.Paths.Issues)
	case "paths.glossary":
		fmt.Println(cfg.Paths.Glossary)
	case "paths.traceability":
		fmt.Println(cfg.Paths.Traceability)
	case "paths.projects_dir":
		fmt.Println(cfg.Paths.ProjectsDir)
	case "heuristics.preserve_threshold":
		fmt.Println(cfg.Heuristics.PreserveThreshold)
	case "heuristics.migrate_threshold":
		fmt.Println(cfg.Heuristics.MigrateThreshold)
	case "heuristics.analysis_depth":
		fmt.Println(cfg.Heuristics.AnalysisDepth)
	default:
		return fmt.Errorf("unknown config key: %s", args.Key)
	}

	return nil
}

type configPathArgs struct {
	Dir      string `targ:"flag,short=d,desc=Project directory (default: current)"`
	Artifact string `targ:"flag,short=a,required,desc=Artifact name (readme / requirements / design / architecture / tasks / issues / glossary / traceability / projects)"`
}

func configPath(args configPathArgs) error {
	dir := args.Dir
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	fs := &realConfigFS{}
	cfg, err := config.Load(dir, homeDir, fs)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	resolved := cfg.ResolvePath(args.Artifact)
	if resolved == "" {
		return fmt.Errorf("unknown artifact: %s", args.Artifact)
	}

	fmt.Println(resolved)
	return nil
}
