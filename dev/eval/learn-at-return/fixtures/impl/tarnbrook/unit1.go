package importer

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Row is one stock line from an import file.
type Row struct {
	Name     string
	Quantity int
	Unit     string
}

// RowError reports a malformed row by its 1-based line number.
type RowError struct {
	Line int
	Err  error
}

func (e RowError) Error() string { return fmt.Sprintf("line %d: %v", e.Line, e.Err) }

// ReadStock parses `name,quantity,unit` rows. Malformed rows are skipped and returned as errors.
func ReadStock(r io.Reader) ([]Row, []RowError, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1

	var rows []Row
	var bad []RowError

	for line := 1; ; line++ {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			return rows, bad, nil
		}
		if err != nil {
			return nil, nil, fmt.Errorf("reading stock csv: %w", err)
		}
		if len(record) != 3 {
			bad = append(bad, RowError{Line: line, Err: fmt.Errorf("want 3 fields, got %d", len(record))})
			continue
		}
		qty, err := strconv.Atoi(strings.TrimSpace(record[1]))
		if err != nil {
			bad = append(bad, RowError{Line: line, Err: fmt.Errorf("quantity %q: %w", record[1], err)})
			continue
		}
		rows = append(rows, Row{Name: strings.TrimSpace(record[0]), Quantity: qty, Unit: strings.TrimSpace(record[2])})
	}
}
