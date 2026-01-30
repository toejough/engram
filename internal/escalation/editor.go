package escalation

// CommandExecutor runs external commands.
type CommandExecutor interface {
	Run(name string, args ...string) error
}

// EnvFunc retrieves environment variables.
type EnvFunc func(key string) string

// SelectEditor returns the editor to use, checking $EDITOR env var first.
func SelectEditor(env EnvFunc) string {
	// TODO: implement
	return ""
}

// OpenInEditor opens a file in the specified editor.
func OpenInEditor(path string, editor string, exec CommandExecutor) error {
	// TODO: implement
	return nil
}

// ReviewEscalations writes escalations to file, opens editor, and parses results.
func ReviewEscalations(escalations []Escalation, path string, env EnvFunc, exec CommandExecutor, fs EscalationFS) ([]Escalation, error) {
	// TODO: implement
	return nil, nil
}
