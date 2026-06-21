package main

import (
	"errors"
	"fmt"
)

func findItem(id string) (*Item, error) {
	item, ok := store[id]
	if !ok {
		return nil, fmt.Errorf("finding item %s: %w", id, errBase)
	}
	return &item, nil
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}
	return parseConfig(data)
}
