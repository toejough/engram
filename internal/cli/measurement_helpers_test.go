//nolint:testpackage // whitebox test — exercises unexported measurement pipeline helpers
package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"

	"engram/internal/adapt"
	"engram/internal/memory"
	"engram/internal/policy"
)

func TestApplyMeasureResults(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()
	memPath := filepath.Join(dir, "test.toml")

	record := memory.MemoryRecord{
		Title:   "test",
		Content: "test content",
		MaintenanceHistory: []memory.MaintenanceAction{
			{Action: "rewrite", EffectivenessBefore: 30.0, Measured: false},
		},
	}

	var buf bytes.Buffer

	err := toml.NewEncoder(&buf).Encode(record)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	writeErr := os.WriteFile(memPath, buf.Bytes(), 0o644)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	records := []adapt.MeasurableRecord{
		{Path: memPath, Record: record},
	}

	results := []adapt.MeasuredResult{
		{Path: memPath, ActionIndex: 0, EffectivenessAfter: 55.0, SurfacedCountAfter: 20},
	}

	applyMeasureResults(records, results)

	data, readErr := os.ReadFile(memPath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	var loaded memory.MemoryRecord

	_, decodeErr := toml.Decode(string(data), &loaded)
	g.Expect(decodeErr).NotTo(HaveOccurred())

	if decodeErr != nil {
		return
	}

	g.Expect(loaded.MaintenanceHistory[0].Measured).To(BeTrue())
	g.Expect(loaded.MaintenanceHistory[0].EffectivenessAfter).To(BeNumerically("~", 55.0, 0.001))
	g.Expect(loaded.MaintenanceHistory[0].SurfacedCountAfter).To(Equal(20))
}

func TestApplyMeasureResults_NoMatchingPath(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()
	memPath := filepath.Join(dir, "test.toml")

	record := memory.MemoryRecord{
		Title:   "test",
		Content: "test content",
		MaintenanceHistory: []memory.MaintenanceAction{
			{Action: "rewrite", EffectivenessBefore: 30.0, Measured: false},
		},
	}

	var buf bytes.Buffer

	err := toml.NewEncoder(&buf).Encode(record)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	writeErr := os.WriteFile(memPath, buf.Bytes(), 0o644)
	g.Expect(writeErr).NotTo(HaveOccurred())

	if writeErr != nil {
		return
	}

	records := []adapt.MeasurableRecord{
		{Path: memPath, Record: record},
	}

	// Results reference a different path — nothing should be updated.
	results := []adapt.MeasuredResult{
		{Path: filepath.Join(dir, "other.toml"), ActionIndex: 0, EffectivenessAfter: 55.0},
	}

	applyMeasureResults(records, results)

	data, readErr := os.ReadFile(memPath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	var loaded memory.MemoryRecord

	_, decodeErr := toml.Decode(string(data), &loaded)
	g.Expect(decodeErr).NotTo(HaveOccurred())

	if decodeErr != nil {
		return
	}

	g.Expect(loaded.MaintenanceHistory[0].Measured).To(BeFalse())
}

func TestCollectActivePolicies(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	policies := []policy.Policy{
		{ID: "pol-001", Status: policy.StatusActive},
		{ID: "pol-002", Status: policy.StatusRetired},
		{ID: "pol-003", Status: policy.StatusActive},
	}

	active := collectActivePolicies(policies)

	g.Expect(active).To(HaveLen(2))
	g.Expect(active[0].ID).To(Equal("pol-001"))
	g.Expect(active[1].ID).To(Equal("pol-003"))
}

func TestCollectActivePolicies_Empty(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	active := collectActivePolicies([]policy.Policy{})

	g.Expect(active).To(BeEmpty())
}

func TestCollectActivePolicies_NoneActive(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	policies := []policy.Policy{
		{ID: "pol-001", Status: policy.StatusRetired},
		{ID: "pol-002", Status: policy.StatusProposed},
	}

	active := collectActivePolicies(policies)

	g.Expect(active).To(BeEmpty())
}

func TestLoadMeasurableRecords_EmptyInput(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	records := loadMeasurableRecords([]*memory.Stored{})

	g.Expect(records).To(BeEmpty())
}

func TestLoadMeasurableRecords_FiltersToHistoryOnly(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()

	// Memory with maintenance history.
	withHistoryPath := filepath.Join(dir, "with_history.toml")
	withHistory := memory.MemoryRecord{
		Title:   "has history",
		Content: "some content",
		MaintenanceHistory: []memory.MaintenanceAction{
			{Action: "rewrite", EffectivenessBefore: 40.0},
		},
	}

	var buf1 bytes.Buffer

	err1 := toml.NewEncoder(&buf1).Encode(withHistory)
	g.Expect(err1).NotTo(HaveOccurred())

	if err1 != nil {
		return
	}

	writeErr1 := os.WriteFile(withHistoryPath, buf1.Bytes(), 0o644)
	g.Expect(writeErr1).NotTo(HaveOccurred())

	if writeErr1 != nil {
		return
	}

	// Memory without maintenance history.
	noHistoryPath := filepath.Join(dir, "no_history.toml")
	noHistory := memory.MemoryRecord{
		Title:   "no history",
		Content: "other content",
	}

	var buf2 bytes.Buffer

	err2 := toml.NewEncoder(&buf2).Encode(noHistory)
	g.Expect(err2).NotTo(HaveOccurred())

	if err2 != nil {
		return
	}

	writeErr2 := os.WriteFile(noHistoryPath, buf2.Bytes(), 0o644)
	g.Expect(writeErr2).NotTo(HaveOccurred())

	if writeErr2 != nil {
		return
	}

	memories := []*memory.Stored{
		{FilePath: withHistoryPath},
		{FilePath: noHistoryPath},
	}

	records := loadMeasurableRecords(memories)

	g.Expect(records).To(HaveLen(1))
	g.Expect(records[0].Path).To(Equal(withHistoryPath))
	g.Expect(records[0].Record.MaintenanceHistory).To(HaveLen(1))
}

func TestLoadMeasurableRecords_SkipsMissingFiles(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	dir := t.TempDir()

	memories := []*memory.Stored{
		{FilePath: filepath.Join(dir, "nonexistent.toml")},
	}

	records := loadMeasurableRecords(memories)

	g.Expect(records).To(BeEmpty())
}

func TestMaintenancePolicyToGeneratorOpts(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	policyFile := &policy.File{
		Policies: []policy.Policy{
			{
				Dimension: policy.DimensionMaintenance,
				Status:    policy.StatusActive,
				Parameter: "stalenessThresholdDays",
				Value:     60.0,
			},
			{
				Dimension: policy.DimensionMaintenance,
				Status:    policy.StatusActive,
				Parameter: "refineKeywordsIrrelevanceThreshold",
				Value:     0.4,
			},
		},
	}

	opts := maintenancePolicyToGeneratorOpts(policyFile)
	g.Expect(opts).To(HaveLen(2))
}

func TestMaintenancePolicyToGeneratorOpts_NilFile(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	opts := maintenancePolicyToGeneratorOpts(nil)
	g.Expect(opts).To(BeEmpty())
}

func TestMaintenancePolicyToReviewOpts(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	policyFile := &policy.File{
		Policies: []policy.Policy{
			{
				Dimension: policy.DimensionMaintenance,
				Status:    policy.StatusActive,
				Parameter: "effectivenessThreshold",
				Value:     45.0,
			},
			{
				Dimension: policy.DimensionMaintenance,
				Status:    policy.StatusActive,
				Parameter: "flagThreshold",
				Value:     30.0,
			},
			{
				Dimension: policy.DimensionMaintenance,
				Status:    policy.StatusActive,
				Parameter: "minEvaluations",
				Value:     3.0,
			},
			{
				Dimension: policy.DimensionSurfacing,
				Status:    policy.StatusActive,
				Parameter: "wEff",
				Value:     0.5,
			},
		},
	}

	opts := maintenancePolicyToReviewOpts(policyFile)
	g.Expect(opts).To(HaveLen(3)) // only maintenance policies
}

func TestMaintenancePolicyToReviewOpts_NilFile(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	opts := maintenancePolicyToReviewOpts(nil)
	g.Expect(opts).To(BeEmpty())
}

func TestMarkValidatedPolicies(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	policyFile := &policy.File{
		Policies: []policy.Policy{
			{ID: "pol-001", Status: policy.StatusActive},
			{ID: "pol-002", Status: policy.StatusActive},
		},
	}

	snap := adapt.CorpusSnapshot{
		FollowRate:        0.65,
		IrrelevanceRatio:  0.08,
		MeanEffectiveness: 72.0,
	}

	markValidatedPolicies(policyFile, []string{"pol-001"}, snap)

	g.Expect(policyFile.Policies[0].Effectiveness.Validated).To(BeTrue())
	g.Expect(policyFile.Policies[0].Effectiveness.AfterFollowRate).To(BeNumerically("~", 0.65, 0.001))
	g.Expect(policyFile.Policies[0].Effectiveness.AfterIrrelevanceRatio).To(BeNumerically("~", 0.08, 0.001))
	g.Expect(policyFile.Policies[0].Effectiveness.AfterMeanEffectiveness).To(BeNumerically("~", 72.0, 0.001))
	g.Expect(policyFile.Policies[1].Effectiveness.Validated).To(BeFalse())
}

func TestMarkValidatedPolicies_MultiplePoliciesValidated(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	policyFile := &policy.File{
		Policies: []policy.Policy{
			{ID: "pol-001", Status: policy.StatusActive},
			{ID: "pol-002", Status: policy.StatusActive},
			{ID: "pol-003", Status: policy.StatusActive},
		},
	}

	snap := adapt.CorpusSnapshot{
		FollowRate:        0.70,
		IrrelevanceRatio:  0.05,
		MeanEffectiveness: 80.0,
	}

	markValidatedPolicies(policyFile, []string{"pol-001", "pol-003"}, snap)

	g.Expect(policyFile.Policies[0].Effectiveness.Validated).To(BeTrue())
	g.Expect(policyFile.Policies[1].Effectiveness.Validated).To(BeFalse())
	g.Expect(policyFile.Policies[2].Effectiveness.Validated).To(BeTrue())
}

func TestMarkValidatedPolicies_UnknownIDIgnored(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	policyFile := &policy.File{
		Policies: []policy.Policy{
			{ID: "pol-001", Status: policy.StatusActive},
		},
	}

	snap := adapt.CorpusSnapshot{FollowRate: 0.5}

	markValidatedPolicies(policyFile, []string{"pol-999"}, snap)

	g.Expect(policyFile.Policies[0].Effectiveness.Validated).To(BeFalse())
}
