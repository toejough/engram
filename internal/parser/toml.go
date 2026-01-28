package parser

import (
	"fmt"
	"time"

	"github.com/BurntSushi/toml"

	"github.com/toejough/projctl/internal/trace"
)

// TOMLResult contains parsed TOML items and deprecation flag.
type TOMLResult struct {
	Items      []*trace.TraceItem
	Deprecated bool // Always true for TOML format
}

// tomlDocument represents the TOML document structure.
type tomlDocument struct {
	// Single item format (flat)
	ID       string    `toml:"id"`
	Type     string    `toml:"type"`
	Project  string    `toml:"project"`
	Title    string    `toml:"title"`
	Status   string    `toml:"status"`
	TracesTo []string  `toml:"traces_to"`
	Tags     []string  `toml:"tags"`
	Created  time.Time `toml:"created"`
	Updated  time.Time `toml:"updated"`

	// Multiple items format (array of tables)
	Item []tomlItem `toml:"item"`
}

// tomlItem represents a single item in the [[item]] array.
type tomlItem struct {
	ID       string    `toml:"id"`
	Type     string    `toml:"type"`
	Project  string    `toml:"project"`
	Title    string    `toml:"title"`
	Status   string    `toml:"status"`
	TracesTo []string  `toml:"traces_to"`
	Tags     []string  `toml:"tags"`
	Created  time.Time `toml:"created"`
	Updated  time.Time `toml:"updated"`
}

// ParseTOML parses TOML-formatted content into TraceItems.
// Returns items with Deprecated=true to signal legacy format.
func ParseTOML(content string) (*TOMLResult, error) {
	var doc tomlDocument
	if _, err := toml.Decode(content, &doc); err != nil {
		return nil, fmt.Errorf("invalid TOML: %w", err)
	}

	var items []*trace.TraceItem

	// Check for multiple items format
	if len(doc.Item) > 0 {
		for _, item := range doc.Item {
			traceItem := tomlItemToTraceItem(item)
			if err := traceItem.Validate(); err != nil {
				return nil, err
			}
			items = append(items, traceItem)
		}
	} else if doc.ID != "" {
		// Single item format
		item := tomlItem{
			ID:       doc.ID,
			Type:     doc.Type,
			Project:  doc.Project,
			Title:    doc.Title,
			Status:   doc.Status,
			TracesTo: doc.TracesTo,
			Tags:     doc.Tags,
			Created:  doc.Created,
			Updated:  doc.Updated,
		}
		traceItem := tomlItemToTraceItem(item)
		if err := traceItem.Validate(); err != nil {
			return nil, err
		}
		items = append(items, traceItem)
	}

	return &TOMLResult{
		Items:      items,
		Deprecated: true,
	}, nil
}

func tomlItemToTraceItem(item tomlItem) *trace.TraceItem {
	return &trace.TraceItem{
		ID:           item.ID,
		Type:         trace.NodeType(item.Type),
		Project:      item.Project,
		Title:        item.Title,
		Status:       item.Status,
		TracesTo:     item.TracesTo,
		Tags:         item.Tags,
		Created:      item.Created,
		Updated:      item.Updated,
		SourceFormat: "toml",
	}
}
