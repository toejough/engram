package cli

import (
	"context"
	"io"
	"io/fs"
)

// AutoPlaceArgs holds parsed flags for `engram auto place`.
type AutoPlaceArgs struct {
	VaultPath   string `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"`
	SessionsDir string `targ:"flag,name=sessions,desc=directory of recorded sessions (default ENGRAM_TRANSCRIPT_DIR then CLAUDE_CONFIG_DIR/projects then ~/.claude/projects)"` //nolint:lll // single unbreakable struct-tag string
}

// AutoPlaceDeps holds injected dependencies for RunAutoPlace.
type AutoPlaceDeps struct {
	// Stdin carries the reply of the lessons judgement and its carry: one JSON object.
	Stdin io.Reader
	// ReadDir and ReadFile read the vault and the session-log directory.
	ReadDir  func(path string) ([]fs.DirEntry, error)
	ReadFile func(path string) ([]byte, error)
}

// RunAutoPlace reads the reply of the lessons judgement from standard input and prints the request for the model that
// places the lessons against the notes the session has seen (855d4h).
func RunAutoPlace(_ context.Context, _ AutoPlaceArgs, _ AutoPlaceDeps, _ io.Writer) error {
	return nil
}
