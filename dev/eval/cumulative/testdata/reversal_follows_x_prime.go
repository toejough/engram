package main

import (
	"errors"
	"os"
)

var ErrNotFound = errors.New("not found")
var ErrInvalid = errors.New("invalid input")

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
