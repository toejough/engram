package frontmatter

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrNoDate is returned when a page's front matter has no date field.
var ErrNoDate = errors.New("front matter has no date")

// dateLayouts are the accepted forms of the `date:` field, most specific first.
var dateLayouts = []string{
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02",
}

// parseFrontMatterDate turns the raw `date:` value of a page's front matter into a timestamp.
// file is the page's path, used only in the error message.
func parseFrontMatterDate(file, raw string) (time.Time, error) {
	value := strings.Trim(strings.TrimSpace(raw), `"'`)
	if value == "" {
		return time.Time{}, fmt.Errorf("%s: %w", file, ErrNoDate)
	}

	for _, layout := range dateLayouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed.UTC(), nil
		}
	}

	return time.Time{}, fmt.Errorf("invalid date in %s: %q", file, value)
}

// PageDate reads the date field from parsed front matter.
func PageDate(file string, fields map[string]string) (time.Time, error) {
	raw, ok := fields["date"]
	if !ok {
		return time.Time{}, fmt.Errorf("%s: %w", file, ErrNoDate)
	}

	return parseFrontMatterDate(file, raw)
}
