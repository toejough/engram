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
	// ErrPlacementReply is the failure of a writing given a placement reply that is no list (793d4h).
	ErrPlacementReply = errors.New("the reply of the placement judgement is not a list")
)

// Placed is where a lesson is written: its position in the order of the vault, the seen note it is a continuation or a
// sibling of, and the seen notes it supersedes (793d4h).
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

// Ties is the supersessions a lesson of a deep delivery names: the file name of the note, as the instructions show,
// whose type and claim the lesson holds, or an object or a list of objects that hold their own.
type Ties []Supersession

// UnmarshalJSON reads a note's file name, a list of file names, a supersession object or a list of them. A form it does
// not know is no supersession and no failure: the lesson that holds it is written without one.
func (t *Ties) UnmarshalJSON(data []byte) error {
	var name string

	var names []string

	var list []Supersession

	var one Supersession

	switch {
	case json.Unmarshal(data, &name) == nil:
		*t = Ties{{Note: name}}
	case json.Unmarshal(data, &names) == nil:
		*t = make(Ties, 0, len(names))
		for _, each := range names {
			*t = append(*t, Supersession{Note: each})
		}
	case json.Unmarshal(data, &list) == nil:
		*t = list
	case json.Unmarshal(data, &one) == nil:
		*t = Ties{one}
	default:
		*t = nil
	}

	return nil
}

// ParsePlacements reads the placement judgement's reply: a JSON list, with or without a code fence around it (793d4h).
func ParsePlacements(reply string) ([]Placement, error) {
	var placements []Placement

	readErr := json.Unmarshal([]byte(replyValue(reply)), &placements)
	if readErr != nil {
		return nil, ErrPlacementReply
	}

	return placements, nil
}

// PlacedAt returns where the lesson with a slug is written (793d4h): as a continuation or a sibling of the target the
// placement gave it only when the target is a seen note, with the supersessions of seen notes only; any other placement
// or supersession is ignored, and the lesson is written at the top.
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
