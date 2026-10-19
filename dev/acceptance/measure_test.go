//go:build network

package acceptance_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"
)

// unexported constants.
const (
	// measureBound is the 2 minutes of every wait of the flow of a prompt (855d2a).
	measureBound = 2 * time.Minute
	// the kinds of the lines of the meter that only the tagged rows read
	meterTool = "tool"
	meterTurn = "turn"
)

// chains links the machine's step runs and model calls.
func (m *machine) chains() []chain { return chainsOf(m.stepRuns(), m.meterRecords()) }

// hasMeter installs the meter mod beside the mod of automatic memory.
func (m *machine) hasMeter() {
	m.t.Helper()

	root, err := moduleRoot()
	if err != nil {
		m.t.Fatalf("finding the module: %v", err)
	}

	m.installMod("meter", filepath.Join(root, "dev", "acceptance", "testdata", "meter"))
}

// meterRecords reads what the meter mod wrote, oldest first.
func (m *machine) meterRecords() []meterRecord {
	data, _ := os.ReadFile(filepath.Join(m.root, "meter.jsonl"))

	var records []meterRecord

	for line := range strings.SplitSeq(string(data), "\n") {
		var record meterRecord

		if json.Unmarshal([]byte(line), &record) == nil && record.Kind != "" {
			records = append(records, record)
		}
	}

	return records
}

// stepRuns reads the runs of the steps that stepsAreRecorded kept and that have ended.
func (m *machine) stepRuns() []stepRun {
	inputs, _ := filepath.Glob(filepath.Join(m.root, "steps-seen", "*-*.in"))

	var runs []stepRun

	for _, name := range inputs {
		base := strings.TrimSuffix(filepath.Base(name), ".in")
		step, _, _ := strings.Cut(base, "-")
		data, readErr := os.ReadFile(name)
		started, startErr := os.Stat(name)
		ended, endErr := os.Stat(strings.TrimSuffix(name, ".in") + ".done")
		output, _ := os.ReadFile(strings.TrimSuffix(name, ".in") + ".out")

		var input map[string]any

		if readErr != nil || startErr != nil || endErr != nil || json.Unmarshal(data, &input) != nil {
			continue
		}

		runs = append(runs, stepRun{
			step: step, input: input, output: string(output), start: started.ModTime(), end: ended.ModTime(),
		})
	}

	return runs
}

// table writes rows as columns under a header, for the log of a measurement.
func table(header []string, rows [][]string) string {
	var out strings.Builder

	writer := tabwriter.NewWriter(&out, 0, 0, 2, ' ', 0)

	_, _ = fmt.Fprintln(writer, strings.Join(header, "\t"))

	for _, row := range rows {
		_, _ = fmt.Fprintln(writer, strings.Join(row, "\t"))
	}

	_ = writer.Flush()

	return out.String()
}
