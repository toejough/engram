package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	agentpkg "engram/internal/agent"
)

// unexported constants.
const (
	stateFileLockDelay   = 25 * time.Millisecond
	stateFileLockRetries = 200
)

// unexported variables.
var (
	errNotImplemented       = errors.New("not implemented")
	errStateFileLockTimeout = errors.New("state file lock timeout after 5s")
)

// deriveStateFilePath mirrors deriveChatFilePath but uses the "state" subdirectory.
func deriveStateFilePath(
	override string,
	homeDir func() (string, error),
	getwd func() (string, error),
) (string, error) {
	if override != "" {
		return override, nil
	}

	home, err := homeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}

	cwd, cwdErr := getwd()
	if cwdErr != nil {
		return "", fmt.Errorf("resolving working directory: %w", cwdErr)
	}

	return filepath.Join(DataDirFromHome(home, os.Getenv), "state", ProjectSlugFromPath(cwd)+".toml"), nil
}

// osStateFileLock acquires a lockfile with a 5s timeout for the wider R-M-W critical section.
func osStateFileLock(name string) (func() error, error) {
	for range stateFileLockRetries {
		f, err := os.OpenFile(name, os.O_CREATE|os.O_EXCL, chatFileMode) //nolint:gosec
		if err == nil {
			return func() error {
				_ = f.Close()

				return os.Remove(name)
			}, nil
		}

		if !os.IsExist(err) {
			return nil, fmt.Errorf("creating state lock: %w", err)
		}

		time.Sleep(stateFileLockDelay)
	}

	return nil, errStateFileLockTimeout
}

// readModifyWriteStateFile performs a locked read-modify-write on the state file.
// Creates the file and its parent directory if they do not exist.
func readModifyWriteStateFile(stateFilePath string, modify func(agentpkg.StateFile) agentpkg.StateFile) error {
	lockPath := stateFilePath + ".lock"

	unlock, lockErr := osStateFileLock(lockPath)
	if lockErr != nil {
		return fmt.Errorf("acquiring state file lock: %w", lockErr)
	}

	defer func() { _ = unlock() }()

	data, readErr := os.ReadFile(stateFilePath) //nolint:gosec
	if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
		return fmt.Errorf("reading state file: %w", readErr)
	}

	currentState, parseErr := agentpkg.ParseStateFile(data)
	if parseErr != nil {
		return fmt.Errorf("parsing state file: %w", parseErr)
	}

	currentState = modify(currentState)

	newData, marshalErr := agentpkg.MarshalStateFile(currentState)
	if marshalErr != nil {
		return fmt.Errorf("marshaling state file: %w", marshalErr)
	}

	dir := filepath.Dir(stateFilePath)

	mkdirErr := os.MkdirAll(dir, chatDirMode)
	if mkdirErr != nil {
		return fmt.Errorf("creating state directory: %w", mkdirErr)
	}

	writeErr := os.WriteFile(stateFilePath, newData, chatFileMode)
	if writeErr != nil {
		return fmt.Errorf("writing state file: %w", writeErr)
	}

	return nil
}

// resolveStateFile derives the state file path, wrapping errors with the subcommand name.
func resolveStateFile(
	override, cmd string,
	homeDir func() (string, error),
	getwd func() (string, error),
) (string, error) {
	path, err := deriveStateFilePath(override, homeDir, getwd)
	if err != nil {
		return "", fmt.Errorf("%s: %w", cmd, err)
	}

	return path, nil
}

// runAgentDispatch routes agent subcommands (spawn|kill|list|wait-ready).
func runAgentDispatch(subArgs []string, stdout io.Writer) error {
	if len(subArgs) < 1 {
		return fmt.Errorf("%w: agent requires a subcommand (spawn|kill|list|wait-ready)", errUsage)
	}

	switch subArgs[0] {
	case "spawn":
		return runAgentSpawn(subArgs[1:], stdout)
	case "kill":
		return runAgentKill(subArgs[1:], stdout)
	case "list":
		return runAgentList(subArgs[1:], stdout)
	case "wait-ready":
		return runAgentWaitReady(subArgs[1:], stdout)
	default:
		return fmt.Errorf("%w: agent %s", errUnknownCommand, subArgs[0])
	}
}

func runAgentKill(_ []string, _ io.Writer) error { return errNotImplemented }

func runAgentList(_ []string, _ io.Writer) error { return errNotImplemented }

// Stub runners — replaced in Tasks 6–9.
func runAgentSpawn(_ []string, _ io.Writer) error { return errNotImplemented }

func runAgentWaitReady(_ []string, _ io.Writer) error { return errNotImplemented }
