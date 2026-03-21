package learn

// ExportUnionKeywords exposes unionKeywords for black-box testing.
func ExportUnionKeywords(l *Learner, a, b []string) []string {
	return l.unionKeywords(a, b)
}
