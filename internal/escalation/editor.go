package escalation

// CommandExecutor runs external commands.
type CommandExecutor interface {
	Run(name string, args ...string) error
}

// EnvFunc retrieves environment variables.
type EnvFunc func(key string) string

// SelectEditor returns the editor to use, checking $EDITOR env var first.
// Falls back to vim if $EDITOR is not set.
func SelectEditor(env EnvFunc) string {
	if editor := env("EDITOR"); editor != "" {
		return editor
	}
	return "vim"
}

// OpenInEditor opens a file in the specified editor and waits for it to close.
func OpenInEditor(path string, editor string, exec CommandExecutor) error {
	return exec.Run(editor, path)
}

// ReviewEscalations writes escalations to file, opens editor, and parses results.
func ReviewEscalations(escalations []Escalation, path string, env EnvFunc, exec CommandExecutor, fs EscalationFS) ([]Escalation, error) {
	// Write escalations to file
	if err := WriteEscalationFile(path, escalations, fs); err != nil {
		return nil, err
	}

	// Select and open editor
	editor := SelectEditor(env)
	if err := OpenInEditor(path, editor, exec); err != nil {
		return nil, err
	}

	// Parse edited file
	return ParseEscalationFile(path, fs)
}
