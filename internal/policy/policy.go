// Package policy provides types and persistence for adaptive policy directives.
package policy

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

// Exported constants.
const (
	// DimensionExtraction targets the memory extraction subsystem.
	DimensionExtraction Dimension = "extraction"
	// DimensionMaintenance targets the memory maintenance subsystem.
	DimensionMaintenance Dimension = "maintenance"
	// DimensionSurfacing targets the memory surfacing subsystem.
	DimensionSurfacing Dimension = "surfacing"
	// StatusActive marks a policy as currently in effect.
	StatusActive Status = "active"
	// StatusApproved marks a policy as approved but not yet active.
	StatusApproved Status = "approved"
	// StatusProposed marks a policy as pending review.
	StatusProposed Status = "proposed"
	// StatusRejected marks a policy as declined.
	StatusRejected Status = "rejected"
	// StatusRetired marks a policy as no longer in effect.
	StatusRetired Status = "retired"
)

// Exported variables.
var (
	ErrInvalidStatus  = errors.New("policy is not in proposed status")
	ErrPolicyNotFound = errors.New("policy not found")
)

// AdaptationConfig holds configurable thresholds for the adaptation analysis engine.
// Zero values mean "use default".
type AdaptationConfig struct {
	MinClusterSize         int     `toml:"min_cluster_size,omitempty"`
	MinFeedbackEvents      int     `toml:"min_feedback_events,omitempty"`
	MeasurementWindow      int     `toml:"measurement_window,omitempty"`
	MaintenanceMinOutcomes int     `toml:"maintenance_min_outcomes,omitempty"`
	MaintenanceMinSuccess  float64 `toml:"maintenance_min_success,omitempty"`
	MinNewFeedback         int     `toml:"min_new_feedback,omitempty"`
}

// ApprovalStreak tracks consecutive approvals per dimension.
type ApprovalStreak struct {
	Extraction  int `toml:"extraction"`
	Surfacing   int `toml:"surfacing"`
	Maintenance int `toml:"maintenance"`
}

// Dimension identifies which subsystem a policy targets.
type Dimension string

// Effectiveness tracks before/after corpus-wide metrics for a policy.
// Uses flat fields (not nested struct) for TOML simplicity.
type Effectiveness struct {
	BeforeFollowRate        float64 `toml:"before_follow_rate,omitempty"`
	BeforeIrrelevanceRatio  float64 `toml:"before_irrelevance_ratio,omitempty"`
	BeforeMeanEffectiveness float64 `toml:"before_mean_effectiveness,omitempty"`
	AfterFollowRate         float64 `toml:"after_follow_rate,omitempty"`
	AfterIrrelevanceRatio   float64 `toml:"after_irrelevance_ratio,omitempty"`
	AfterMeanEffectiveness  float64 `toml:"after_mean_effectiveness,omitempty"`
	MeasuredSessions        int     `toml:"measured_sessions"`
	Validated               bool    `toml:"validated,omitempty"`
}

// Evidence holds the statistical basis for a policy proposal.
type Evidence struct {
	IrrelevantRate   float64 `toml:"irrelevant_rate,omitempty"`
	FollowRate       float64 `toml:"follow_rate,omitempty"`
	Correlation      float64 `toml:"correlation,omitempty"`
	SampleSize       int     `toml:"sample_size"`
	SessionsObserved int     `toml:"sessions_observed,omitempty"`
}

// File represents the policy.toml file.
type File struct {
	Policies       []Policy         `toml:"policies"`
	ApprovalStreak ApprovalStreak   `toml:"approval_streak"`
	Adaptation     AdaptationConfig `toml:"adaptation"`
}

// Active returns all active policies for the given dimension.
func (pf *File) Active(dim Dimension) []Policy {
	result := make([]Policy, 0)

	for _, pol := range pf.Policies {
		if pol.Dimension == dim && pol.Status == StatusActive {
			result = append(result, pol)
		}
	}

	return result
}

// Approve transitions a proposed policy to active, sets the approved timestamp,
// and increments the approval streak for the policy's dimension.
// Returns ErrPolicyNotFound if the ID does not exist, ErrInvalidStatus if not proposed.
func (pf *File) Approve(id, timestamp string) error {
	index := pf.findIndex(id)
	if index < 0 {
		return ErrPolicyNotFound
	}

	if pf.Policies[index].Status != StatusProposed {
		return ErrInvalidStatus
	}

	pf.Policies[index].Status = StatusActive
	pf.Policies[index].ApprovedAt = timestamp
	pf.incrementStreak(pf.Policies[index].Dimension)

	return nil
}

// DeduplicateProposed removes duplicate proposed policies, keeping the first
// occurrence of each unique Directive+Dimension combination. Non-proposed
// policies are always retained.
func (pf *File) DeduplicateProposed() int {
	seen := make(map[string]bool)
	kept := make([]Policy, 0, len(pf.Policies))
	removed := 0

	for _, pol := range pf.Policies {
		if pol.Status != StatusProposed {
			kept = append(kept, pol)

			continue
		}

		key := string(pol.Dimension) + "\x00" + pol.Directive
		if seen[key] {
			removed++

			continue
		}

		seen[key] = true

		kept = append(kept, pol)
	}

	pf.Policies = kept

	return removed
}

// NextID returns the next sequential policy ID in the form "pol-NNN".
// It finds the highest existing numeric suffix and increments it.
// Returns "pol-001" if no policies exist.
func (pf *File) NextID() string {
	maxNum := 0

	for _, pol := range pf.Policies {
		num := parsePolNum(pol.ID)
		if num > maxNum {
			maxNum = num
		}
	}

	return fmt.Sprintf("pol-%03d", maxNum+1)
}

// Pending returns all proposed policies across all dimensions.
func (pf *File) Pending() []Policy {
	result := make([]Policy, 0)

	for _, pol := range pf.Policies {
		if pol.Status == StatusProposed {
			result = append(result, pol)
		}
	}

	return result
}

// Reject transitions a proposed policy to rejected and resets the approval streak
// for the policy's dimension to 0.
// Returns ErrPolicyNotFound if the ID does not exist, ErrInvalidStatus if not proposed.
func (pf *File) Reject(id string) error {
	index := pf.findIndex(id)
	if index < 0 {
		return ErrPolicyNotFound
	}

	if pf.Policies[index].Status != StatusProposed {
		return ErrInvalidStatus
	}

	pf.Policies[index].Status = StatusRejected
	pf.resetStreak(pf.Policies[index].Dimension)

	return nil
}

// Retire transitions any policy to retired status.
// Returns ErrPolicyNotFound if the ID does not exist.
func (pf *File) Retire(id string) error {
	index := pf.findIndex(id)
	if index < 0 {
		return ErrPolicyNotFound
	}

	pf.Policies[index].Status = StatusRetired

	return nil
}

// findIndex returns the slice index of the policy with the given ID, or -1 if not found.
func (pf *File) findIndex(id string) int {
	for index, pol := range pf.Policies {
		if pol.ID == id {
			return index
		}
	}

	return -1
}

// incrementStreak increments the approval streak for the given dimension.
func (pf *File) incrementStreak(dim Dimension) {
	switch dim {
	case DimensionExtraction:
		pf.ApprovalStreak.Extraction++
	case DimensionSurfacing:
		pf.ApprovalStreak.Surfacing++
	case DimensionMaintenance:
		pf.ApprovalStreak.Maintenance++
	}
}

// resetStreak resets the approval streak for the given dimension to 0.
func (pf *File) resetStreak(dim Dimension) {
	switch dim {
	case DimensionExtraction:
		pf.ApprovalStreak.Extraction = 0
	case DimensionSurfacing:
		pf.ApprovalStreak.Surfacing = 0
	case DimensionMaintenance:
		pf.ApprovalStreak.Maintenance = 0
	}
}

// Policy represents a single learned adaptation directive.
type Policy struct {
	ID            string        `toml:"id"`
	Dimension     Dimension     `toml:"dimension"`
	Directive     string        `toml:"directive"`
	Rationale     string        `toml:"rationale"`
	Evidence      Evidence      `toml:"evidence"`
	Status        Status        `toml:"status"`
	CreatedAt     string        `toml:"created_at"`
	ApprovedAt    string        `toml:"approved_at,omitempty"`
	Effectiveness Effectiveness `toml:"effectiveness"`
	Parameter     string        `toml:"parameter,omitempty"`
	Value         float64       `toml:"value,omitempty"`
}

// Status tracks the lifecycle state of a policy.
type Status string

// Load reads a policy file from path. Returns an empty File if the file does not exist.
func Load(path string) (*File, error) {
	data, readErr := os.ReadFile(path) //nolint:gosec // path is caller-controlled, intentional
	if os.IsNotExist(readErr) {
		return &File{}, nil
	}

	if readErr != nil {
		return nil, fmt.Errorf("policy: read file: %w", readErr)
	}

	var policyFile File

	_, decodeErr := toml.Decode(string(data), &policyFile)
	if decodeErr != nil {
		return nil, fmt.Errorf("policy: decode TOML: %w", decodeErr)
	}

	return &policyFile, nil
}

// Save writes the policy file to path as TOML, creating parent directories as needed.
func Save(path string, policyFile *File) error {
	dir := filepath.Dir(path)

	mkdirErr := os.MkdirAll(dir, policyDirPerm)
	if mkdirErr != nil {
		return fmt.Errorf("policy: create dir: %w", mkdirErr)
	}

	outFile, createErr := os.Create(path) //nolint:gosec // path is caller-controlled, intentional
	if createErr != nil {
		return fmt.Errorf("policy: create file: %w", createErr)
	}

	encodeErr := toml.NewEncoder(outFile).Encode(policyFile)
	if encodeErr != nil {
		_ = outFile.Close()
		return fmt.Errorf("policy: encode TOML: %w", encodeErr)
	}

	closeErr := outFile.Close()
	if closeErr != nil {
		return fmt.Errorf("policy: close file: %w", closeErr)
	}

	return nil
}

// unexported constants.
const (
	policyDirPerm = 0o750
)

// parsePolNum parses the numeric suffix from a policy ID like "pol-042".
// Returns 0 if the ID does not match the expected format.
func parsePolNum(id string) int {
	prefix := "pol-"
	if !strings.HasPrefix(id, prefix) {
		return 0
	}

	numStr := strings.TrimPrefix(id, prefix)

	num, err := strconv.Atoi(numStr)
	if err != nil {
		return 0
	}

	return num
}
