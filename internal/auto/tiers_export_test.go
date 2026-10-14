package auto

func (t Tiers) CaptureWith(window string, near []Note) (Request, error) {
	return Try(func() Request { return t.captureRequest(window, near) })
}

// The request builders of a tier table given, for the tier tests (793d25b1).
func (t Tiers) ClassifyWith(recent string) (Request, error) {
	return Try(func() Request { return t.classifyRequest(recent) })
}

func (t Tiers) CurationWith(offer Note, near []Note) (Request, error) {
	return Try(func() Request { return t.curationRequest(offer, near) })
}

func (t Tiers) RelevanceWith(moment, text string, notes []Note) (Request, error) {
	return Try(func() Request { return t.relevanceRequest(moment, text, notes) })
}
