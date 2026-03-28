package policy_test

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/policy"
)

// TestActivePolicies_FiltersByStatus verifies Active() returns only active policies for the given dimension.
func TestActivePolicies_FiltersByStatus(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	polFile := &policy.File{
		Policies: []policy.Policy{
			{ID: "pol-001", Dimension: policy.DimensionExtraction, Status: policy.StatusActive},
			{ID: "pol-002", Dimension: policy.DimensionExtraction, Status: policy.StatusProposed},
			{ID: "pol-003", Dimension: policy.DimensionSurfacing, Status: policy.StatusActive},
			{ID: "pol-004", Dimension: policy.DimensionExtraction, Status: policy.StatusRetired},
			{ID: "pol-005", Dimension: policy.DimensionExtraction, Status: policy.StatusActive},
		},
	}

	active := polFile.Active(policy.DimensionExtraction)
	g.Expect(active).To(HaveLen(2))
	g.Expect(active[0].ID).To(Equal("pol-001"))
	g.Expect(active[1].ID).To(Equal("pol-005"))
}

// TestApprove_AllDimensions_IncrementStreak verifies approve increments the streak for each dimension.
func TestApprove_AllDimensions_IncrementStreak(t *testing.T) {
	t.Parallel()

	t.Run("extraction", func(t *testing.T) {
		t.Parallel()
		g := NewGomegaWithT(t)

		polFile := &policy.File{
			Policies: []policy.Policy{
				{ID: "pol-001", Dimension: policy.DimensionExtraction, Status: policy.StatusProposed},
			},
		}

		err := polFile.Approve("pol-001", "2026-03-27T10:00:00Z")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(polFile.ApprovalStreak.Extraction).To(Equal(1))
	})

	t.Run("maintenance", func(t *testing.T) {
		t.Parallel()
		g := NewGomegaWithT(t)

		polFile := &policy.File{
			Policies: []policy.Policy{
				{ID: "pol-001", Dimension: policy.DimensionMaintenance, Status: policy.StatusProposed},
			},
		}

		err := polFile.Approve("pol-001", "2026-03-27T10:00:00Z")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(polFile.ApprovalStreak.Maintenance).To(Equal(1))
	})
}

// TestApprove_NonProposed_ReturnsError verifies approve returns ErrInvalidStatus for non-proposed policies.
func TestApprove_NonProposed_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	polFile := &policy.File{
		Policies: []policy.Policy{
			{ID: "pol-001", Dimension: policy.DimensionExtraction, Status: policy.StatusActive},
		},
	}

	err := polFile.Approve("pol-001", "2026-03-27T10:00:00Z")
	g.Expect(err).To(MatchError(policy.ErrInvalidStatus))
}

// TestApprove_NotFound_ReturnsError verifies approve returns ErrPolicyNotFound for unknown IDs.
func TestApprove_NotFound_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	polFile := &policy.File{}

	err := polFile.Approve("pol-999", "2026-03-27T10:00:00Z")
	g.Expect(err).To(MatchError(policy.ErrPolicyNotFound))
}

// TestApprove_TransitionsAndUpdatesStreak verifies approve transitions a proposed policy to active
// and increments the streak for its dimension.
func TestApprove_TransitionsAndUpdatesStreak(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	polFile := &policy.File{
		Policies: []policy.Policy{
			{ID: "pol-001", Dimension: policy.DimensionSurfacing, Status: policy.StatusProposed},
		},
		ApprovalStreak: policy.ApprovalStreak{Surfacing: 1},
	}

	err := polFile.Approve("pol-001", "2026-03-27T10:00:00Z")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(polFile.Policies[0].Status).To(Equal(policy.StatusActive))
	g.Expect(polFile.Policies[0].ApprovedAt).To(Equal("2026-03-27T10:00:00Z"))
	g.Expect(polFile.ApprovalStreak.Surfacing).To(Equal(2))
}

// TestDeduplicateProposed_RemovesDuplicates verifies DeduplicateProposed removes duplicate proposed
// policies (same Directive+Dimension) while keeping the first occurrence and leaving non-proposed untouched.
func TestDeduplicateProposed_RemovesDuplicates(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	polFile := &policy.File{
		Policies: []policy.Policy{
			{
				ID:        "pol-001",
				Dimension: policy.DimensionExtraction,
				Directive: "Extract only actionable insights",
				Status:    policy.StatusProposed,
			},
			{
				ID:        "pol-002",
				Dimension: policy.DimensionExtraction,
				Directive: "Extract only actionable insights",
				Status:    policy.StatusProposed,
			},
			{
				ID:        "pol-003",
				Dimension: policy.DimensionSurfacing,
				Directive: "Surface recent memories first",
				Status:    policy.StatusProposed,
			},
			{
				ID:        "pol-004",
				Dimension: policy.DimensionExtraction,
				Directive: "Extract only actionable insights",
				Status:    policy.StatusActive,
			},
		},
	}

	removed := polFile.DeduplicateProposed()
	g.Expect(removed).To(Equal(1))
	g.Expect(polFile.Policies).To(HaveLen(3))
	g.Expect(polFile.Policies[0].ID).To(Equal("pol-001"))
	g.Expect(polFile.Policies[1].ID).To(Equal("pol-003"))
	g.Expect(polFile.Policies[2].ID).To(Equal("pol-004"))
}

// TestDeduplicateProposed_SameStemDifferentStats verifies DeduplicateProposed treats
// directives with the same stem but different stats suffixes as duplicates.
func TestDeduplicateProposed_SameStemDifferentStats(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	polFile := &policy.File{
		Policies: []policy.Policy{
			{
				ID:        "pol-001",
				Dimension: policy.DimensionExtraction,
				Directive: "retire pol-001: follow rate 37% (was 38%), mean effectiveness 21.2 (was 21.9)",
				Status:    policy.StatusProposed,
			},
			{
				ID:        "pol-002",
				Dimension: policy.DimensionExtraction,
				Directive: "retire pol-001: follow rate 38% (was 38%), mean effectiveness 21.5 (was 21.9)",
				Status:    policy.StatusProposed,
			},
		},
	}

	removed := polFile.DeduplicateProposed()
	g.Expect(removed).To(Equal(1))
	g.Expect(polFile.Policies).To(HaveLen(1))
	g.Expect(polFile.Policies[0].ID).To(Equal("pol-001"))
}

// TestDirectiveStem extracts the semantic action from a directive, ignoring trailing stats after the colon.
func TestDirectiveStem(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		directive string
		want      string
	}{
		{
			name:      "retire with stats",
			directive: `retire pol-021: follow rate 37% (was 38%), mean effectiveness 21.2 (was 21.9)`,
			want:      "retire pol-021",
		},
		{
			name:      "de-prioritize with stats",
			directive: `de-prioritize keyword "spec": 88% irrelevance rate across 11 memories`,
			want:      `de-prioritize keyword "spec"`,
		},
		{
			name:      "no colon returns full string",
			directive: "Extract only actionable insights",
			want:      "Extract only actionable insights",
		},
		{
			name:      "empty string",
			directive: "",
			want:      "",
		},
		{
			name:      "colon at start",
			directive: ": some stats",
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			g.Expect(policy.DirectiveStem(tt.directive)).To(Equal(tt.want))
		})
	}
}

// TestLoad_MissingFile_ReturnsEmpty verifies loading a nonexistent path returns an empty File with no error.
func TestLoad_MissingFile_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	path := filepath.Join(t.TempDir(), "does-not-exist.toml")

	loaded, err := policy.Load(path)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(loaded).NotTo(BeNil())
	g.Expect(loaded.Policies).To(BeEmpty())
}

// TestNextID_Empty verifies NextID returns pol-001 when there are no existing policies.
func TestNextID_Empty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	polFile := &policy.File{}

	nextID := polFile.NextID()
	g.Expect(nextID).To(Equal("pol-001"))
}

// TestNextID_IgnoresNonStandardIDs verifies NextID ignores policies with non-standard ID formats.
func TestNextID_IgnoresNonStandardIDs(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	polFile := &policy.File{
		Policies: []policy.Policy{
			{ID: "pol-001"},
			{ID: "invalid"},
			{ID: "pol-abc"},
		},
	}

	nextID := polFile.NextID()
	g.Expect(nextID).To(Equal("pol-002"))
}

// TestNextID_Sequential verifies NextID returns the next sequential ID after the highest existing one.
func TestNextID_Sequential(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	polFile := &policy.File{
		Policies: []policy.Policy{
			{ID: "pol-001"},
			{ID: "pol-003"},
		},
	}

	nextID := polFile.NextID()
	g.Expect(nextID).To(Equal("pol-004"))
}

// TestPendingPolicies_FiltersByProposed verifies Pending() returns only proposed policies across all dimensions.
func TestPendingPolicies_FiltersByProposed(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	polFile := &policy.File{
		Policies: []policy.Policy{
			{ID: "pol-001", Dimension: policy.DimensionExtraction, Status: policy.StatusActive},
			{ID: "pol-002", Dimension: policy.DimensionSurfacing, Status: policy.StatusProposed},
			{ID: "pol-003", Dimension: policy.DimensionMaintenance, Status: policy.StatusProposed},
			{ID: "pol-004", Dimension: policy.DimensionExtraction, Status: policy.StatusRejected},
		},
	}

	pending := polFile.Pending()
	g.Expect(pending).To(HaveLen(2))
	g.Expect(pending[0].ID).To(Equal("pol-002"))
	g.Expect(pending[1].ID).To(Equal("pol-003"))
}

// TestReject_AllDimensions_ResetsStreak verifies reject resets the streak for each dimension.
func TestReject_AllDimensions_ResetsStreak(t *testing.T) {
	t.Parallel()

	t.Run("extraction", func(t *testing.T) {
		t.Parallel()
		g := NewGomegaWithT(t)

		polFile := &policy.File{
			Policies: []policy.Policy{
				{ID: "pol-001", Dimension: policy.DimensionExtraction, Status: policy.StatusProposed},
			},
			ApprovalStreak: policy.ApprovalStreak{Extraction: 5},
		}

		err := polFile.Reject("pol-001")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(polFile.ApprovalStreak.Extraction).To(Equal(0))
	})

	t.Run("surfacing", func(t *testing.T) {
		t.Parallel()
		g := NewGomegaWithT(t)

		polFile := &policy.File{
			Policies: []policy.Policy{
				{ID: "pol-001", Dimension: policy.DimensionSurfacing, Status: policy.StatusProposed},
			},
			ApprovalStreak: policy.ApprovalStreak{Surfacing: 2},
		}

		err := polFile.Reject("pol-001")
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(polFile.ApprovalStreak.Surfacing).To(Equal(0))
	})
}

// TestReject_NotFound_ReturnsError verifies reject returns ErrPolicyNotFound for unknown IDs.
func TestReject_NotFound_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	polFile := &policy.File{}

	err := polFile.Reject("pol-999")
	g.Expect(err).To(MatchError(policy.ErrPolicyNotFound))
}

// TestReject_TransitionsAndResetsStreak verifies reject transitions a proposed policy to rejected
// and resets the streak for its dimension to 0.
func TestReject_TransitionsAndResetsStreak(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	polFile := &policy.File{
		Policies: []policy.Policy{
			{ID: "pol-001", Dimension: policy.DimensionMaintenance, Status: policy.StatusProposed},
		},
		ApprovalStreak: policy.ApprovalStreak{Maintenance: 3},
	}

	err := polFile.Reject("pol-001")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(polFile.Policies[0].Status).To(Equal(policy.StatusRejected))
	g.Expect(polFile.ApprovalStreak.Maintenance).To(Equal(0))
}

// TestRetire_NotFound_ReturnsError verifies retire returns ErrPolicyNotFound for unknown IDs.
func TestRetire_NotFound_ReturnsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	polFile := &policy.File{}

	err := polFile.Retire("pol-999")
	g.Expect(err).To(MatchError(policy.ErrPolicyNotFound))
}

// TestRetire_Transitions verifies retire transitions any policy to retired status.
func TestRetire_Transitions(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	polFile := &policy.File{
		Policies: []policy.Policy{
			{ID: "pol-001", Dimension: policy.DimensionExtraction, Status: policy.StatusActive},
		},
	}

	err := polFile.Retire("pol-001")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(polFile.Policies[0].Status).To(Equal(policy.StatusRetired))
}

// TestRoundTrip_AdaptationConfig_AllFields verifies AdaptationConfig fields survive a TOML round-trip.
func TestRoundTrip_AdaptationConfig_AllFields(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	polFile := &policy.File{
		Adaptation: policy.AdaptationConfig{
			MinClusterSize:             3,
			MinFeedbackEvents:          10,
			MeasurementWindow:          30,
			MaintenanceMinOutcomes:     5,
			MaintenanceMinSuccess:      0.85,
			MinNewFeedback:             7,
			ConsolidationMinConfidence: 0.75,
		},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "adaptation.toml")

	saveErr := policy.Save(path, polFile)
	g.Expect(saveErr).NotTo(HaveOccurred())

	if saveErr != nil {
		return
	}

	loaded, loadErr := policy.Load(path)
	g.Expect(loadErr).NotTo(HaveOccurred())

	if loadErr != nil {
		return
	}

	got := loaded.Adaptation
	g.Expect(got.MinClusterSize).To(Equal(3))
	g.Expect(got.MinFeedbackEvents).To(Equal(10))
	g.Expect(got.MeasurementWindow).To(Equal(30))
	g.Expect(got.MaintenanceMinOutcomes).To(Equal(5))
	g.Expect(got.MaintenanceMinSuccess).To(BeNumerically("~", 0.85, 0.001))
	g.Expect(got.MinNewFeedback).To(Equal(7))
	g.Expect(got.ConsolidationMinConfidence).To(BeNumerically("~", 0.75, 0.001))
}

// TestRoundTrip_AdaptationConfig_ZeroValues verifies a zero AdaptationConfig round-trips as all zeros.
func TestRoundTrip_AdaptationConfig_ZeroValues(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	polFile := &policy.File{}

	dir := t.TempDir()
	path := filepath.Join(dir, "adaptation-zero.toml")

	saveErr := policy.Save(path, polFile)
	g.Expect(saveErr).NotTo(HaveOccurred())

	if saveErr != nil {
		return
	}

	loaded, loadErr := policy.Load(path)
	g.Expect(loadErr).NotTo(HaveOccurred())

	if loadErr != nil {
		return
	}

	got := loaded.Adaptation
	g.Expect(got.MinClusterSize).To(Equal(0))
	g.Expect(got.MinFeedbackEvents).To(Equal(0))
	g.Expect(got.MeasurementWindow).To(Equal(0))
	g.Expect(got.MaintenanceMinOutcomes).To(Equal(0))
	g.Expect(got.MaintenanceMinSuccess).To(BeNumerically("~", 0.0, 0.001))
	g.Expect(got.MinNewFeedback).To(Equal(0))
	g.Expect(got.ConsolidationMinConfidence).To(BeNumerically("~", 0.0, 0.001))
}

// TestRoundTrip_Effectiveness_FlatCorpusSnapshot verifies all Effectiveness fields round-trip through TOML.
func TestRoundTrip_Effectiveness_FlatCorpusSnapshot(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	polFile := &policy.File{
		Policies: []policy.Policy{
			{
				ID:        "pol-001",
				Dimension: policy.DimensionSurfacing,
				Status:    policy.StatusActive,
				CreatedAt: "2026-03-27T00:00:00Z",
				Effectiveness: policy.Effectiveness{
					BeforeFollowRate:        0.61,
					BeforeIrrelevanceRatio:  0.25,
					BeforeMeanEffectiveness: 0.48,
					AfterFollowRate:         0.79,
					AfterIrrelevanceRatio:   0.12,
					AfterMeanEffectiveness:  0.71,
					MeasuredSessions:        12,
					Validated:               true,
				},
			},
		},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "effectiveness.toml")

	saveErr := policy.Save(path, polFile)
	g.Expect(saveErr).NotTo(HaveOccurred())

	if saveErr != nil {
		return
	}

	loaded, loadErr := policy.Load(path)
	g.Expect(loadErr).NotTo(HaveOccurred())

	if loadErr != nil {
		return
	}

	g.Expect(loaded.Policies).To(HaveLen(1))
	eff := loaded.Policies[0].Effectiveness
	g.Expect(eff.BeforeFollowRate).To(BeNumerically("~", 0.61, 0.001))
	g.Expect(eff.BeforeIrrelevanceRatio).To(BeNumerically("~", 0.25, 0.001))
	g.Expect(eff.BeforeMeanEffectiveness).To(BeNumerically("~", 0.48, 0.001))
	g.Expect(eff.AfterFollowRate).To(BeNumerically("~", 0.79, 0.001))
	g.Expect(eff.AfterIrrelevanceRatio).To(BeNumerically("~", 0.12, 0.001))
	g.Expect(eff.AfterMeanEffectiveness).To(BeNumerically("~", 0.71, 0.001))
	g.Expect(eff.MeasuredSessions).To(Equal(12))
	g.Expect(eff.Validated).To(BeTrue())
}

// TestRoundTrip_SaveAndLoad verifies a File saved to disk can be loaded back with all fields intact.
func TestRoundTrip_SaveAndLoad(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	polFile := &policy.File{
		Policies: []policy.Policy{
			{
				ID:        "pol-001",
				Dimension: policy.DimensionExtraction,
				Directive: "Extract only actionable insights",
				Rationale: "Reduces noise",
				Evidence: policy.Evidence{
					IrrelevantRate:   0.42,
					FollowRate:       0.81,
					Correlation:      0.65,
					SampleSize:       50,
					SessionsObserved: 10,
				},
				Status:    policy.StatusActive,
				CreatedAt: "2026-03-01T00:00:00Z",
				Effectiveness: policy.Effectiveness{
					BeforeFollowRate:        0.55,
					BeforeIrrelevanceRatio:  0.30,
					BeforeMeanEffectiveness: 0.40,
					AfterFollowRate:         0.72,
					AfterIrrelevanceRatio:   0.18,
					AfterMeanEffectiveness:  0.65,
					MeasuredSessions:        5,
					Validated:               true,
				},
				Parameter: "min_confidence",
				Value:     0.75,
			},
		},
		ApprovalStreak: policy.ApprovalStreak{
			Extraction:  2,
			Surfacing:   1,
			Maintenance: 0,
		},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "policy.toml")

	saveErr := policy.Save(path, polFile)
	g.Expect(saveErr).NotTo(HaveOccurred())

	if saveErr != nil {
		return
	}

	loaded, loadErr := policy.Load(path)
	g.Expect(loadErr).NotTo(HaveOccurred())

	if loadErr != nil {
		return
	}

	g.Expect(loaded.Policies).To(HaveLen(1))
	got := loaded.Policies[0]
	g.Expect(got.ID).To(Equal("pol-001"))
	g.Expect(got.Dimension).To(Equal(policy.DimensionExtraction))
	g.Expect(got.Directive).To(Equal("Extract only actionable insights"))
	g.Expect(got.Rationale).To(Equal("Reduces noise"))
	g.Expect(got.Evidence.IrrelevantRate).To(BeNumerically("~", 0.42, 0.001))
	g.Expect(got.Evidence.FollowRate).To(BeNumerically("~", 0.81, 0.001))
	g.Expect(got.Evidence.SampleSize).To(Equal(50))
	g.Expect(got.Status).To(Equal(policy.StatusActive))
	g.Expect(got.CreatedAt).To(Equal("2026-03-01T00:00:00Z"))
	g.Expect(got.Effectiveness.BeforeFollowRate).To(BeNumerically("~", 0.55, 0.001))
	g.Expect(got.Effectiveness.BeforeIrrelevanceRatio).To(BeNumerically("~", 0.30, 0.001))
	g.Expect(got.Effectiveness.BeforeMeanEffectiveness).To(BeNumerically("~", 0.40, 0.001))
	g.Expect(got.Effectiveness.AfterFollowRate).To(BeNumerically("~", 0.72, 0.001))
	g.Expect(got.Effectiveness.AfterIrrelevanceRatio).To(BeNumerically("~", 0.18, 0.001))
	g.Expect(got.Effectiveness.AfterMeanEffectiveness).To(BeNumerically("~", 0.65, 0.001))
	g.Expect(got.Effectiveness.MeasuredSessions).To(Equal(5))
	g.Expect(got.Effectiveness.Validated).To(BeTrue())
	g.Expect(got.Parameter).To(Equal("min_confidence"))
	g.Expect(got.Value).To(BeNumerically("~", 0.75, 0.001))
	g.Expect(loaded.ApprovalStreak.Extraction).To(Equal(2))
	g.Expect(loaded.ApprovalStreak.Surfacing).To(Equal(1))
	g.Expect(loaded.ApprovalStreak.Maintenance).To(Equal(0))
}

// TestSave_CreatesParentDirectory verifies Save creates the parent directory if it doesn't exist.
func TestSave_CreatesParentDirectory(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := filepath.Join(t.TempDir(), "nested", "dir")
	path := filepath.Join(dir, "policy.toml")

	polFile := &policy.File{}

	err := policy.Save(path, polFile)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	_, statErr := os.Stat(path)
	g.Expect(statErr).NotTo(HaveOccurred())
}

// TestSave_ErrorWhenCannotCreateFile verifies Save returns an error when the path is a directory.
func TestSave_ErrorWhenCannotCreateFile(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Pass a directory as the path — os.Create will fail.
	dirPath := t.TempDir()

	polFile := &policy.File{}

	err := policy.Save(dirPath, polFile)
	g.Expect(err).To(HaveOccurred())
}

// TestSave_ErrorWhenCannotMkdir verifies Save returns an error when the parent directory cannot be created.
func TestSave_ErrorWhenCannotMkdir(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Use a path nested under a file (not a dir) — MkdirAll will fail.
	baseDir := t.TempDir()
	existingFile := filepath.Join(baseDir, "existing-file")

	writeErr := os.WriteFile(existingFile, []byte("content"), 0o600)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	path := filepath.Join(existingFile, "subdir", "policy.toml")
	polFile := &policy.File{}

	err := policy.Save(path, polFile)
	g.Expect(err).To(HaveOccurred())
}
