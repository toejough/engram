package main

import (
	"errors"
	"os"
)

// Exported variables.
var (
	ErrInvalid  = errors.New("invalid input")
	ErrNotFound = errors.New("not found")
)

func findItem(id string) (*Item, error) {
	item, ok := store[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &item, nil
}

func validate(input string) error {
	if input == "" {
		return ErrInvalid
	}
	return nil
}
