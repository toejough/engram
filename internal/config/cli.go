package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// GetArgs holds arguments for the config get command.
type GetArgs struct {
	Dir string `targ:"flag,short=d,desc=Project directory (default: current)"`
	Key string `targ:"flag,short=k,desc=Config key to get (optional; returns all if not specified)"`
}

// InitArgs holds arguments for the config init command.
type InitArgs struct {
	Dir string `targ:"flag,short=d,desc=Project directory (default: current)"`
}

// PathArgs holds arguments for the config path command.
type PathArgs struct {
	Dir      string `targ:"flag,short=d,desc=Project directory (default: current)"`
	Artifact string `targ:"flag,short=a,required,desc=Artifact name (readme / requirements / design / architecture / tasks / issues / glossary / traceability / projects)"`
}

// RealFS implements ConfigFS using the real file system.
type RealFS struct{}

func (r RealFS) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (r RealFS) ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// RunGet retrieves a configuration value.
func RunGet(args GetArgs) error {
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

	cfg, err := Load(dir, homeDir, RealFS{})
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if args.Key == "" {
		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to encode config: %w", err)
		}

		fmt.Println(string(data))

		return nil
	}

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

// RunInit initializes a project configuration file.
func RunInit(args InitArgs) error {
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

	cfg := Default()

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

// RunPath shows the resolved path for a configuration artifact.
func RunPath(args PathArgs) error {
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

	cfg, err := Load(dir, homeDir, RealFS{})
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
