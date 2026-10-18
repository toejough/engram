package auto

// PlaceInput is the reply of the lessons judgement and its carry, as the memory mod hands them over (855d4h).
type PlaceInput struct {
	Session string `json:"session"`
	Turn    string `json:"turn"`
	Reply   string `json:"reply"`
	Carry   Carry  `json:"carry"`
}

// Place returns the request that places the lessons against the notes the session has seen (855d4h), and false when
// there is none to place or none seen.
func Place(_ []Lesson, _ []Note) (Step, bool) {
	return Step{}, false
}
