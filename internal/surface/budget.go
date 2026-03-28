package surface

// Exported constants.
const (
	DefaultSessionStartBudget     = 600
	DefaultStopBudget             = 500
	DefaultUserPromptSubmitBudget = 250
)

// BudgetConfig holds per-hook token budget caps (ARCH-40).
type BudgetConfig struct {
	SessionStart     int
	UserPromptSubmit int
	Stop             int
}

// ForMode returns the token budget for a given surface mode.
func (c BudgetConfig) ForMode(mode string) int {
	switch mode {
	case ModePrompt:
		return c.UserPromptSubmit
	default:
		return 0
	}
}

// BudgetConfigReader loads budget configuration from persistent storage.
type BudgetConfigReader interface {
	ReadBudgetConfig() (BudgetConfig, error)
}

// DefaultBudgetConfig returns the default budget configuration.
func DefaultBudgetConfig() BudgetConfig {
	return BudgetConfig{
		SessionStart:     DefaultSessionStartBudget,
		UserPromptSubmit: DefaultUserPromptSubmitBudget,
		Stop:             DefaultStopBudget,
	}
}

// EstimateTokens returns the estimated token count for text using len/4 truncation.
func EstimateTokens(text string) int {
	return len(text) / estimateTokensDivisor
}

// WithBudgetConfig sets the budget configuration for a Surfacer.
func WithBudgetConfig(config BudgetConfig) SurfacerOption {
	return func(s *Surfacer) { s.budgetConfig = &config }
}

// unexported constants.
const (
	estimateTokensDivisor = 4
)

// applyPromptBudget returns the prefix of matches that fits within the token budget.
// Budget of 0 means unlimited.
func applyPromptBudget(matches []promptMatch, budget int) []promptMatch {
	if budget <= 0 {
		return matches
	}

	accumulated := 0
	result := make([]promptMatch, 0, len(matches))

	for _, match := range matches {
		tokens := EstimateTokens(match.searchText)
		if accumulated+tokens > budget {
			break
		}

		accumulated += tokens

		result = append(result, match)
	}

	return result
}
