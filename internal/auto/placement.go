package auto

import (
	"encoding/json"
	"errors"
	"slices"
)

// Exported constants.
const (
	PositionContinuation = "continuation"
	PositionSibling      = "sibling"
	PositionTop          = "top"
)

// Exported variables.
var (
	// ErrPlacementReply is the failure of a writing given a placement reply that is no list (855d4i).
	ErrPlacementReply = errors.New("the reply of the placement judgement is not a list")
)

// Placed is where a lesson is written: its position in the order of the vault, the seen note it is a continuation or a
// sibling of, and the seen notes it supersedes (855d4i).
type Placed struct {
	Position   string
	Target     string
	Supersedes []Supersession
}

// Placement is the placement judgement's entry for a lesson, by its slug (855d4h).
type Placement struct {
	Slug       string         `json:"slug"`
	Position   string         `json:"position"`
	Target     string         `json:"target"`
	Supersedes []Supersession `json:"supersedes"`
}

// Supersession is a note a lesson corrects, narrows or refutes, with the type of the tie and the claim that changes.
type Supersession struct {
	Note  string `json:"note"`
	Type  string `json:"type"`
	Claim string `json:"claim"`
}

// ParsePlacements reads the placement judgement's reply: a JSON list, with or without a code fence around it (855d4i).
func ParsePlacements(reply string) ([]Placement, error) {
	var placements []Placement

	readErr := json.Unmarshal([]byte(replyValue(reply)), &placements)
	if readErr != nil {
		return nil, ErrPlacementReply
	}

	return placements, nil
}

// PlacedAt returns where the lesson with a slug is written (855d4i): as a continuation or a sibling of the target the
// placement gave it only when the target is a seen note, with the supersessions of seen notes only; any other placement
// or supersession is ignored, and the lesson is written at the top (855b31).
func PlacedAt(placements []Placement, slug string, seen []string) Placed {
	index := slices.IndexFunc(placements, func(placement Placement) bool { return placement.Slug == slug })
	if index < 0 {
		return Placed{Position: PositionTop}
	}

	placement := placements[index]
	placed := Placed{Position: PositionTop}

	if slices.Contains([]string{PositionContinuation, PositionSibling}, placement.Position) &&
		slices.Contains(seen, placement.Target) {
		placed.Position, placed.Target = placement.Position, placement.Target
	}

	for _, tie := range placement.Supersedes {
		if slices.Contains(seen, tie.Note) && isSupersedesType(tie.Type) && !isBlank(tie.Claim) {
			placed.Supersedes = append(placed.Supersedes,
				Supersession{Note: tie.Note, Type: tie.Type, Claim: oneRedacted(tie.Claim)})
		}
	}

	return placed
}

// isSupersedesType reports whether a type is one of the ties a lesson has with the note it supersedes (792a15h).
func isSupersedesType(tie string) bool {
	return tie == "updates" || tie == "narrows" || tie == "refutes"
}
