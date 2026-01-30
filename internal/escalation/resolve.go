package escalation

// IssueSpec describes an issue to be created from an escalation.
type IssueSpec struct {
	Source      string // Original escalation ID
	Title       string // Issue title
	Description string // Issue description
	Category    string // deferred-escalation or user-issue
}

// ResolutionResult holds the results of applying resolutions.
type ResolutionResult struct {
	Applied []Escalation // Escalations that were resolved
	Issues  []IssueSpec  // Issues to be created
	Pending []Escalation // Escalations still pending
}

// ApplyResolutions processes escalations and returns results.
func ApplyResolutions(escalations []Escalation) *ResolutionResult {
	result := &ResolutionResult{}

	for _, e := range escalations {
		switch e.Status {
		case "resolved":
			result.Applied = append(result.Applied, e)

		case "deferred":
			result.Issues = append(result.Issues, IssueSpec{
				Source:      e.ID,
				Title:       e.Question,
				Description: e.Context,
				Category:    "deferred-escalation",
			})

		case "issue":
			result.Issues = append(result.Issues, IssueSpec{
				Source:      e.ID,
				Title:       e.Question,
				Description: e.Notes,
				Category:    "user-issue",
			})

		case "pending":
			result.Pending = append(result.Pending, e)
		}
	}

	return result
}
