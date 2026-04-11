package chat_test

import (
	"encoding/json"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/chat"
)

func TestEvaluateCondition_DoneAgent_BeforeAcquiredTS_NotMet(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	acquiredAt := time.Now()
	hold := chat.HoldRecord{HoldID: "h1", Condition: "done:reviewer-1", AcquiredTS: acquiredAt}
	messages := []chat.Message{
		// done message BEFORE hold was acquired — should not satisfy condition
		{From: "reviewer-1", Type: "done", TS: acquiredAt.Add(-1 * time.Second)},
	}
	met, _ := chat.EvaluateCondition(hold, messages)
	g.Expect(met).To(BeFalse())
}

// --- Step 7: EvaluateCondition ---

func TestEvaluateCondition_DoneAgent_ConditionMet(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	acquiredAt := time.Now().Add(-1 * time.Minute)
	hold := chat.HoldRecord{HoldID: "h1", Condition: "done:reviewer-1", AcquiredTS: acquiredAt}
	messages := []chat.Message{
		{From: "reviewer-1", Type: "done", TS: time.Now()},
	}
	met, reason := chat.EvaluateCondition(hold, messages)
	g.Expect(met).To(BeTrue())
	g.Expect(reason).NotTo(BeEmpty())
}

func TestEvaluateCondition_DoneAgent_ConditionNotMet(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	acquiredAt := time.Now().Add(-1 * time.Minute)
	hold := chat.HoldRecord{HoldID: "h1", Condition: "done:reviewer-1", AcquiredTS: acquiredAt}
	messages := []chat.Message{
		{From: "reviewer-1", Type: "info", TS: time.Now()},
	}
	met, _ := chat.EvaluateCondition(hold, messages)
	g.Expect(met).To(BeFalse())
}

func TestEvaluateCondition_EmptyCondition_NeverMet(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	hold := chat.HoldRecord{
		HoldID:     "h1",
		Condition:  "",
		AcquiredTS: time.Now().Add(-1 * time.Minute),
	}
	met, _ := chat.EvaluateCondition(hold, nil)
	g.Expect(met).To(BeFalse())
}

func TestEvaluateCondition_FirstIntent_Met(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	acquiredAt := time.Now().Add(-1 * time.Minute)
	hold := chat.HoldRecord{HoldID: "h1", Condition: "first-intent:exec-1", AcquiredTS: acquiredAt}
	messages := []chat.Message{
		{From: "exec-1", Type: "intent", TS: time.Now()},
	}
	met, reason := chat.EvaluateCondition(hold, messages)
	g.Expect(met).To(BeTrue())
	g.Expect(reason).NotTo(BeEmpty())
}

func TestEvaluateCondition_FirstIntent_NotMet_WrongType(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	acquiredAt := time.Now().Add(-1 * time.Minute)
	hold := chat.HoldRecord{HoldID: "h1", Condition: "first-intent:exec-1", AcquiredTS: acquiredAt}
	messages := []chat.Message{
		{From: "exec-1", Type: "info", TS: time.Now()},
	}
	met, _ := chat.EvaluateCondition(hold, messages)
	g.Expect(met).To(BeFalse())
}

func TestEvaluateCondition_LeadRelease_NeverAutoMet(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	hold := chat.HoldRecord{
		HoldID:     "h1",
		Condition:  "lead-release:tag",
		AcquiredTS: time.Now().Add(-1 * time.Minute),
	}
	messages := []chat.Message{
		{From: "lead", Type: "done", TS: time.Now()},
	}
	met, _ := chat.EvaluateCondition(hold, messages)
	g.Expect(met).To(BeFalse(), "lead-release conditions never auto-evaluate to true")
}

func TestEvaluateCondition_UnknownCondition_NeverMet(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	hold := chat.HoldRecord{
		HoldID:     "h1",
		Condition:  "unknown:whatever",
		AcquiredTS: time.Now().Add(-1 * time.Minute),
	}
	messages := []chat.Message{
		{From: "whatever", Type: "done", TS: time.Now()},
	}
	met, _ := chat.EvaluateCondition(hold, messages)
	g.Expect(met).To(BeFalse())
}

// --- Step 5: HoldRecord JSON round-trip ---

func TestHoldRecord_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	original := chat.HoldRecord{
		HoldID:     "h-12345",
		Holder:     "reviewer-1",
		Target:     "executor-1",
		Condition:  "done:reviewer-1",
		AcquiredTS: time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC),
	}
	data, err := json.Marshal(original)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var got chat.HoldRecord
	g.Expect(json.Unmarshal(data, &got)).To(Succeed())
	g.Expect(got).To(Equal(original))
}

func TestHoldRecord_JSONRoundTrip_WithTag(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	original := chat.HoldRecord{
		HoldID:     "h-99999",
		Holder:     "lead",
		Target:     "exec-1",
		Condition:  "lead-release:codesign-1",
		Tag:        "codesign-1",
		AcquiredTS: time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC),
	}
	data, err := json.Marshal(original)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var got chat.HoldRecord
	g.Expect(json.Unmarshal(data, &got)).To(Succeed())
	g.Expect(got.Tag).To(Equal("codesign-1"))
	g.Expect(got).To(Equal(original))
}

func TestScanActiveHolds_AcquireAndRelease_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	record := chat.HoldRecord{HoldID: "h2", Holder: "lead", Target: "exec-1"}
	text, err := json.Marshal(record)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	messages := []chat.Message{
		{From: "lead", To: "exec-1", Type: "hold-acquire", Text: string(text)},
		{From: "lead", To: "exec-1", Type: "hold-release", Text: string(text)},
	}
	holds := chat.ScanActiveHolds(messages)
	g.Expect(holds).To(BeEmpty())
}

func TestScanActiveHolds_AcquireWithNoRelease(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	record := chat.HoldRecord{
		HoldID:    "h1",
		Holder:    "lead",
		Target:    "exec-1",
		Condition: "done:lead",
	}
	text, err := json.Marshal(record)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	messages := []chat.Message{
		{From: "lead", To: "exec-1", Type: "hold-acquire", Text: string(text)},
	}
	holds := chat.ScanActiveHolds(messages)
	g.Expect(holds).To(HaveLen(1))
	g.Expect(holds[0].HoldID).To(Equal("h1"))
}

// --- Step 6: ScanActiveHolds ---

func TestScanActiveHolds_EmptyMessages(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	holds := chat.ScanActiveHolds(nil)
	g.Expect(holds).To(BeEmpty())
}

func TestScanActiveHolds_InvalidJSON_Skipped(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	record := chat.HoldRecord{HoldID: "h-valid", Holder: "lead", Target: "exec-1"}
	text, err := json.Marshal(record)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	messages := []chat.Message{
		{From: "lead", To: "all", Type: "hold-acquire", Text: "not-json"},
		{From: "lead", To: "exec-1", Type: "hold-acquire", Text: string(text)},
	}
	holds := chat.ScanActiveHolds(messages)
	g.Expect(holds).To(HaveLen(1))
	g.Expect(holds[0].HoldID).To(Equal("h-valid"))
}

func TestScanActiveHolds_MultipleHolds_IndependentTracking(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	recordA := chat.HoldRecord{HoldID: "h-a", Holder: "lead", Target: "exec-1"}
	recordB := chat.HoldRecord{HoldID: "h-b", Holder: "lead", Target: "exec-2"}

	textA, errA := json.Marshal(recordA)
	g.Expect(errA).NotTo(HaveOccurred())

	if errA != nil {
		return
	}

	textB, errB := json.Marshal(recordB)
	g.Expect(errB).NotTo(HaveOccurred())

	if errB != nil {
		return
	}

	messages := []chat.Message{
		{From: "lead", To: "exec-1", Type: "hold-acquire", Text: string(textA)},
		{From: "lead", To: "exec-2", Type: "hold-acquire", Text: string(textB)},
		{From: "lead", To: "exec-1", Type: "hold-release", Text: string(textA)},
	}
	holds := chat.ScanActiveHolds(messages)
	g.Expect(holds).To(HaveLen(1))
	g.Expect(holds[0].HoldID).To(Equal("h-b"))
}

func TestScanActiveHolds_ReleaseOnlyHoldID_Cancels(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Release message may only contain {"hold-id":"..."} per plan note
	acquire := chat.HoldRecord{
		HoldID:    "h-min",
		Holder:    "lead",
		Target:    "exec-1",
		Condition: "done:lead",
	}
	acquireText, err := json.Marshal(acquire)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Minimal release — only hold-id
	releaseText := `{"hold-id":"h-min"}`

	messages := []chat.Message{
		{From: "lead", To: "exec-1", Type: "hold-acquire", Text: string(acquireText)},
		{From: "lead", To: "exec-1", Type: "hold-release", Text: releaseText},
	}
	holds := chat.ScanActiveHolds(messages)
	g.Expect(holds).To(BeEmpty())
}
