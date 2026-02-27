package memory

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// RunScoreSession scores a session with a timeout.
func RunScoreSession(args ScoreSessionArgs, homeDir string, stdin io.Reader) error {
	timeout := args.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	type funcResult struct {
		err error
	}

	done := make(chan funcResult, 1)

	go func() {
		done <- funcResult{doScoreSession(args, homeDir, stdin)}
	}()

	select {
	case r := <-done:
		return r.err
	case <-time.After(timeout):
		fmt.Fprintln(os.Stderr, "Warning: score-session timed out after 60s, skipping")
		return nil
	}
}

func doScoreSession(args ScoreSessionArgs, homeDir string, stdin io.Reader) error {
	start := time.Now()
	logPath := filepath.Join(homeDir, ".claude", "memory", "score-session.log")
	logFile, _ := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	dbg := func(msg string) {
		line := fmt.Sprintf("%s [score-session] %s (+%dms)\n",
			time.Now().Format("15:04:05"), msg, time.Since(start).Milliseconds())
		if logFile != nil {
			_, _ = logFile.WriteString(line)
		}
	}

	defer func() {
		if logFile != nil {
			_ = logFile.Close()
		}
	}()

	dbg("starting")

	hookInput, _ := ParseHookInput(stdin)

	dbg("stdin parsed")

	var sessionID string
	if hookInput != nil {
		sessionID = hookInput.SessionID
	}

	memoryRoot := args.MemoryRoot
	if memoryRoot == "" {
		memoryRoot = filepath.Join(homeDir, ".claude", "memory")
	}

	db, err := InitDBForTest(memoryRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: score-session: failed to open database: %v\n", err)
		return nil
	}

	defer func() { _ = db.Close() }()

	dbg("db opened")

	if hookInput != nil && hookInput.HookEventName == "SessionStart" {
		unscoredSession, findErr := FindLatestUnscoredSession(db)
		if findErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: score-session: %v\n", findErr)
			return nil
		}

		if unscoredSession == "" {
			dbg("no unscored sessions found")
			return nil
		}

		sessionID = unscoredSession
		dbg("found unscored session: " + sessionID)
	}

	if sessionID == "" {
		dbg("no session_id available, skipping")
		return nil
	}

	ext := NewLLMExtractor()
	if ext == nil {
		fmt.Fprintln(os.Stderr, "Warning: score-session: LLM extractor unavailable, skipping")
		return nil
	}

	dbg("LLM extractor ready")

	if err := ScoreSession(db, sessionID, ext); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: score-session: %v\n", err)
		return nil
	}

	dbg("scoring complete")

	return nil
}
