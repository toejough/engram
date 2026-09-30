package main

import (
	"errors"
	"fmt"
)

// ErrNoPackSize is returned for a SKU that has no pack size configured.
var ErrNoPackSize = errors.New("sku has no pack size set")

// PackSizes says how many pieces are in a box and how many boxes are on a pallet, for one SKU.
type PackSizes struct {
	PiecesPerBox   int
	BoxesPerPallet int
}

var piecesPer = map[string]func(PackSizes) int{
	"pcs":     func(PackSizes) int { return 1 },
	"boxes":   func(p PackSizes) int { return p.PiecesPerBox },
	"pallets": func(p PackSizes) int { return p.PiecesPerBox * p.BoxesPerPallet },
}

// Convert converts qty of a SKU between pcs, boxes and pallets using its pack sizes.
func Convert(sku string, sizes map[string]PackSizes, qty int, from, to string) (int, error) {
	pack, ok := sizes[sku]
	if !ok || pack.PiecesPerBox == 0 || pack.BoxesPerPallet == 0 {
		return 0, fmt.Errorf("convert %s: %w", sku, ErrNoPackSize)
	}

	fromN, okFrom := piecesPer[from]
	toN, okTo := piecesPer[to]
	if !okFrom || !okTo {
		return 0, fmt.Errorf("convert %s: unknown unit %q or %q", sku, from, to)
	}

	pieces := qty * fromN(pack)
	per := toN(pack)
	if pieces%per != 0 {
		return 0, fmt.Errorf("convert %s: %d %s is not a whole number of %s", sku, qty, from, to)
	}

	return pieces / per, nil
}
