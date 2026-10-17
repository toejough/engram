//go:build targ && integration

package dev

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func Test793d4n1e_AutoSweep_ASweepThatIsEndedLeavesTheChunkIndexUsable(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	machine := newSweepMachine(t)
	record := machine.plantRecord(t, sweepSession, longRecordTurns)

	ctx, cancel := context.WithTimeout(t.Context(), sweepTimeout)
	defer cancel()

	first := machine.sweep(ctx, sweepSession)
	startErr := first.Start()

	g.Expect(startErr).NotTo(HaveOccurred())

	if startErr != nil {
		return
	}

	exited := make(chan error, 1)

	go func() { exited <- first.Wait() }()

	// the sweep holds the lock of the manifest while it embeds: it is ended there
	g.Expect(machine.awaitLock(t, exited)).To(Succeed())
	time.Sleep(embedLead)

	select {
	case <-exited:
		t.Fatal("the sweep ended before it was ended: the record is too short for the test")
	default:
	}

	g.Expect(first.Process.Kill()).To(Succeed())
	<-exited

	g.Expect(machine.readableIndex(t)).To(Succeed(), "the chunk index after the sweep was ended")

	secondOut, secondErr := machine.sweepOutput(ctx, sweepSession)

	g.Expect(secondErr).NotTo(HaveOccurred(), "the second sweep: %s", secondOut)
	g.Expect(machine.manifest(t)).To(HaveKey(record), "the manifest records the record")
	g.Expect(machine.chunksOf(t, record)).To(BeNumerically(">=", longRecordTurns), "the chunks of the record")
}

func Test793d4n1f_AutoSweep_TwoSweepsAtOnceIndexBothRecords(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	machine := newSweepMachine(t)
	records := map[string]string{
		sweepSession:      machine.plantRecord(t, sweepSession, shortRecordTurns),
		otherSweepSession: machine.plantRecord(t, otherSweepSession, shortRecordTurns),
	}

	ctx, cancel := context.WithTimeout(t.Context(), sweepTimeout)
	defer cancel()

	var (
		group    sync.WaitGroup
		mutex    sync.Mutex
		failures = map[string]string{}
	)

	for session := range records {
		group.Go(func() {
			out, err := machine.sweepOutput(ctx, session)
			if err != nil {
				mutex.Lock()
				failures[session] = fmt.Sprintf("%v: %s", err, out)
				mutex.Unlock()
			}
		})
	}

	group.Wait()

	g.Expect(failures).To(BeEmpty(), "the sweeps that failed")

	manifest := machine.manifest(t)

	for _, record := range records {
		g.Expect(manifest).To(HaveKey(record), "the manifest records %s", record)
		g.Expect(machine.chunksOf(t, record)).To(BeNumerically(">=", shortRecordTurns), "the chunks of %s", record)
	}
}

// unexported constants.
const (
	// embedLead is how long a sweep that holds the lock of the manifest is left to go on before it is ended: long
	// enough for it to be embedding, short enough that a record of longRecordTurns turns is not yet done.
	embedLead = 1500 * time.Millisecond
	// longRecordTurns and shortRecordTurns are the turns of a record: each is two messages of more than 600 characters,
	// each of which is a chunk of its own.
	longRecordTurns   = 12
	otherSweepSession = "9e2b7d41-31a8-4f0c-8c55-6a1d0e47b3f2"
	shortRecordTurns  = 4
	sweepSession      = "4f1c0a9e-6f1d-4f5e-9d58-0f3d9a3c8e11"
	// sweepTimeout is how long the sweeps of a test may take, of 24 chunks at the most.
	sweepTimeout = 3 * time.Minute
)

// unexported variables.
var (
	errPartialLine = errors.New("a partial line")
)

// sweepMachine is a computer on which the real engram sweeps: its own home, binary, records of sessions and chunk
// index.
type sweepMachine struct {
	chunks   string
	sessions string
	env      []string
	binary   string
}

// awaitLock waits until a sweep holds the lock of the manifest, which it takes once it has found the record, and fails
// when the sweep ends first.
func (m *sweepMachine) awaitLock(t *testing.T, exited chan error) error {
	t.Helper()

	deadline := time.After(sweepTimeout)

	for {
		_, err := os.Stat(filepath.Join(m.chunks, ".manifest.lock"))
		if err == nil {
			return nil
		}

		select {
		case runErr := <-exited:
			return fmt.Errorf("the sweep ended before it held the lock: %w", runErr)
		case <-deadline:
			return fmt.Errorf("the sweep never held the lock: %w", err)
		case <-time.After(10 * time.Millisecond):
		}
	}
}

// chunksOf counts the chunks of the index of a source: the lines of its index file.
func (m *sweepMachine) chunksOf(t *testing.T, source string) int {
	t.Helper()

	files, err := filepath.Glob(filepath.Join(m.chunks, "*.jsonl"))
	if err != nil {
		t.Fatalf("listing the index files: %v", err)
	}

	count := 0

	for _, name := range files {
		data, readErr := os.ReadFile(name)
		if readErr != nil {
			t.Fatalf("reading %s: %v", name, readErr)
		}

		if strings.Contains(string(data), `"source":"`+source+`"`) {
			count += bytes.Count(data, []byte("\n"))
		}
	}

	return count
}

// manifest reads the manifest of the chunk index: the sources it records, by path.
func (m *sweepMachine) manifest(t *testing.T) map[string]json.RawMessage {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(m.chunks, "manifest.json"))
	if err != nil {
		t.Fatalf("reading the manifest: %v", err)
	}

	var recorded map[string]json.RawMessage

	unmarshalErr := json.Unmarshal(data, &recorded)
	if unmarshalErr != nil {
		t.Fatalf("the manifest is not valid: %v\n%s", unmarshalErr, data)
	}

	return recorded
}

// plantRecord writes the record of a session of turns turns, and returns its path.
func (m *sweepMachine) plantRecord(t *testing.T, session string, turns int) string {
	t.Helper()

	project := filepath.Join(m.sessions, "-home-joe-zorblax")

	mkdirErr := os.MkdirAll(project, 0o750)
	if mkdirErr != nil {
		t.Fatalf("creating %s: %v", project, mkdirErr)
	}

	var rows strings.Builder

	for turn := range turns {
		filler := strings.Repeat(fmt.Sprintf("session %s turn %d: the Zorblax relay carries a label. ", session, turn), 14)
		fmt.Fprintf(&rows, `{"type":"user","message":{"role":"user","content":"%s"}}`+"\n", filler)
		fmt.Fprintf(&rows, `{"type":"assistant","message":{"role":"assistant","content":`+
			`[{"type":"text","text":"%s"}]}}`+"\n", filler)
	}

	path := filepath.Join(project, session+".jsonl")

	writeErr := os.WriteFile(path, []byte(rows.String()), 0o600)
	if writeErr != nil {
		t.Fatalf("writing the record: %v", writeErr)
	}

	return path
}

// readableIndex reports whether every file of the chunk index that exists can be read: the manifest is valid when it
// exists, and each index file holds a JSON object on each line.
func (m *sweepMachine) readableIndex(t *testing.T) error {
	t.Helper()

	_, statErr := os.Stat(filepath.Join(m.chunks, "manifest.json"))
	if statErr == nil {
		m.manifest(t)
	}

	files, globErr := filepath.Glob(filepath.Join(m.chunks, "*.jsonl"))
	if globErr != nil {
		return fmt.Errorf("listing the index files: %w", globErr)
	}

	for _, name := range files {
		data, readErr := os.ReadFile(name)
		if readErr != nil {
			return fmt.Errorf("reading %s: %w", name, readErr)
		}

		for line := range strings.SplitSeq(strings.TrimSpace(string(data)), "\n") {
			if !json.Valid([]byte(line)) {
				return fmt.Errorf("%s holds a line that is not JSON: %w", name, errPartialLine)
			}
		}
	}

	return nil
}

// sweep is the command that runs engram auto sweep for a session, not started.
func (m *sweepMachine) sweep(ctx context.Context, session string) *exec.Cmd {
	command := exec.CommandContext(ctx, m.binary, "auto", "sweep")
	command.Env = m.env
	command.Stdin = strings.NewReader(`{"origin":"composer","session":"` + session + `"}`)

	return command
}

// sweepOutput runs a sweep to its end, and returns what it printed and how it ended.
func (m *sweepMachine) sweepOutput(ctx context.Context, session string) (string, error) {
	command := m.sweep(ctx, session)

	var out bytes.Buffer

	command.Stdout = &out
	command.Stderr = &out

	runErr := command.Run()

	return out.String(), runErr
}

// newSweepMachine makes a computer for a real sweep, with the engram of the repository built for it.
func newSweepMachine(t *testing.T) *sweepMachine {
	t.Helper()

	root := t.TempDir()

	machine := &sweepMachine{
		chunks:   filepath.Join(root, "chunks"),
		sessions: filepath.Join(root, "sessions"),
		binary:   filepath.Join(root, "engram"),
	}

	build := exec.CommandContext(t.Context(), "go", "build", "-o", machine.binary, "./cmd/engram")
	build.Dir = repositoryRoot

	out, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("building engram: %v\n%s", err, out)
	}

	machine.env = []string{
		"PATH=/usr/bin:/bin",
		"HOME=" + filepath.Join(root, "home"),
		"TMPDIR=" + root,
		"XDG_CACHE_HOME=" + filepath.Join(root, "cache"),
		"ENGRAM_CHUNKS_DIR=" + machine.chunks,
		"ENGRAM_TRANSCRIPT_DIR=" + machine.sessions,
		"ENGRAM_VAULT_PATH=" + filepath.Join(root, "no-such-vault"),
	}

	return machine
}
