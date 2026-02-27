package memory

import (
	"context"
	"maps"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func TestCallAnthropicAPI_APIErrorBody(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content": [], "error": {"message": "API error occurred"}}`))
	}))

	defer ts.Close()

	client := &http.Client{
		Transport: &mockAPITransport{serverURL: ts.URL},
	}

	_, err := callAnthropicAPI(context.Background(), client, "test-key", "system", "user")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("API error"))
}

func TestCallAnthropicAPI_EmptyContent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content": []}`))
	}))

	defer ts.Close()

	client := &http.Client{
		Transport: &mockAPITransport{serverURL: ts.URL},
	}

	_, err := callAnthropicAPI(context.Background(), client, "test-key", "system", "user")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("empty response"))
}

func TestCallAnthropicAPI_ErrorStatus(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))

	defer ts.Close()

	client := &http.Client{
		Transport: &mockAPITransport{serverURL: ts.URL},
	}

	_, err := callAnthropicAPI(context.Background(), client, "test-key", "system", "user")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("status 401"))
}

func TestCallAnthropicAPI_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content": [{"text": "test response text"}]}`))
	}))

	defer ts.Close()

	client := &http.Client{
		Transport: &mockAPITransport{serverURL: ts.URL},
	}

	resp, err := callAnthropicAPI(context.Background(), client, "test-key", "system prompt", "user prompt")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(resp).To(Equal("test response text"))
}

func TestGenerateSkillContent_NilCompiler(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cluster := []ClusterEntry{
		{ID: 1, Content: "entry one"},
		{ID: 2, Content: "entry two"},
	}

	content, err := generateSkillContent(context.Background(), "Test Theme", cluster, nil)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(content).To(ContainSubstring("Test Theme"))
	g.Expect(content).To(ContainSubstring("entry one"))
}

func TestGenerateTriggerDescription_LongTheme(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Theme long enough to produce desc > 1024 chars; truncation kicks in.
	// fmt.Sprintf("Use when the user encounters %s-related patterns or needs guidance on %s.", theme, theme)
	// base = 68 chars + 2*len(theme), so len(theme) > 478 pushes past 1024.
	longTheme := strings.Repeat("x", 500)
	desc := generateTriggerDescription(longTheme, "")
	g.Expect(desc).To(HaveLen(1024))
}

func TestGenerateTriggerDescription_Normal(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	desc := generateTriggerDescription("error handling", "some content")
	g.Expect(desc).To(HavePrefix("Use when"))
	g.Expect(len(desc)).To(BeNumerically("<=", 1024))
}

func TestInsertSkill_DuplicateSlug(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	skill := &GeneratedSkill{
		Slug:            "dup-slug",
		Theme:           "Duplicate",
		Description:     "desc",
		Content:         "content",
		SourceMemoryIDs: "[]",
		Alpha:           1.0,
		Beta:            1.0,
		Utility:         0.5,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	_, err = insertSkill(db, skill)
	g.Expect(err).ToNot(HaveOccurred())

	// Second insert with same slug should fail (UNIQUE constraint)
	_, err = insertSkill(db, skill)
	g.Expect(err).To(HaveOccurred())
}

func TestListSkillsPublic_Empty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	skills, err := ListSkillsPublic(db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(skills).To(BeEmpty())
}

func TestListSkillsPublic_Populated(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	skill := &GeneratedSkill{
		Slug:            "test-skill",
		Theme:           "Testing",
		Description:     "test",
		Content:         "content",
		SourceMemoryIDs: "[]",
		Alpha:           1.0,
		Beta:            1.0,
		Utility:         0.5,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	_, err = insertSkill(db, skill)
	g.Expect(err).ToNot(HaveOccurred())

	skills, err := ListSkillsPublic(db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(skills).To(HaveLen(1))

	if len(skills) == 0 {
		t.Fatal("expected at least one skill")
	}

	g.Expect(skills[0].Slug).To(Equal("test-skill"))
}

func TestListSkills_Empty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	skills, err := listSkills(db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(skills).To(BeEmpty())
}

func TestListSkills_WithEmbeddingAndLastRetrieved(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	skill := &GeneratedSkill{
		Slug:            "listed-skill",
		Theme:           "Listed",
		Description:     "desc",
		Content:         "content",
		SourceMemoryIDs: "[]",
		Alpha:           1.0,
		Beta:            1.0,
		Utility:         0.5,
		EmbeddingID:     42,
		LastRetrieved:   now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	_, err = insertSkill(db, skill)
	g.Expect(err).ToNot(HaveOccurred())

	skills, err := listSkills(db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(skills).To(HaveLen(1))

	if len(skills) > 0 {
		g.Expect(skills[0].EmbeddingID).To(Equal(int64(42)))
		g.Expect(skills[0].LastRetrieved).ToNot(BeEmpty())
	}
}

func TestNullInt64_Nonzero(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(nullInt64(5)).To(Equal(int64(5)))
}

func TestNullInt64_Zero(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(nullInt64(0)).To(BeNil())
}

func TestRecordSkillFeedback_MissingSkill(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	err = RecordSkillFeedback(db, "nonexistent", true)
	g.Expect(err).To(HaveOccurred())
}

func TestRecordSkillFeedback_ValidSkill(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	skill := &GeneratedSkill{
		Slug:            "feedback-skill",
		Theme:           "Feedback",
		Description:     "desc",
		Content:         "content",
		SourceMemoryIDs: "[]",
		Alpha:           1.0,
		Beta:            1.0,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	_, err = insertSkill(db, skill)
	g.Expect(err).ToNot(HaveOccurred())

	err = RecordSkillFeedback(db, "feedback-skill", true)
	g.Expect(err).ToNot(HaveOccurred())

	fetched, err := getSkillBySlug(db, "feedback-skill")
	g.Expect(err).ToNot(HaveOccurred())

	if fetched == nil {
		t.Fatal("expected skill but got nil")
	}

	g.Expect(fetched.Alpha).To(BeNumerically(">", 1.0))
}

func TestRecordSkillUsage_MissingSkill(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	err = RecordSkillUsage(db, "nonexistent", false)
	g.Expect(err).To(HaveOccurred())
}

func TestRecordSkillUsage_ValidSkill(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	skill := &GeneratedSkill{
		Slug:            "usage-skill",
		Theme:           "Usage",
		Description:     "desc",
		Content:         "content",
		SourceMemoryIDs: "[]",
		Alpha:           1.0,
		Beta:            1.0,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	_, err = insertSkill(db, skill)
	g.Expect(err).ToNot(HaveOccurred())

	err = RecordSkillUsage(db, "usage-skill", false)
	g.Expect(err).ToNot(HaveOccurred())

	fetched, err := getSkillBySlug(db, "usage-skill")
	g.Expect(err).ToNot(HaveOccurred())

	if fetched == nil {
		t.Fatal("expected skill but got nil")
	}

	g.Expect(fetched.RetrievalCount).To(Equal(1))
}

func TestRecordSkillUsage_WithSuccess(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	skill := &GeneratedSkill{
		Slug:            "usage-success-skill",
		Theme:           "Usage Success",
		Description:     "desc",
		Content:         "content",
		SourceMemoryIDs: "[]",
		Alpha:           1.0,
		Beta:            1.0,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	_, err = insertSkill(db, skill)
	g.Expect(err).ToNot(HaveOccurred())

	// success=true also calls RecordSkillFeedback (increments alpha)
	err = RecordSkillUsage(db, "usage-success-skill", true)
	g.Expect(err).ToNot(HaveOccurred())

	fetched, err := getSkillBySlug(db, "usage-success-skill")
	g.Expect(err).ToNot(HaveOccurred())

	if fetched == nil {
		t.Fatal("expected skill but got nil")
	}

	g.Expect(fetched.RetrievalCount).To(Equal(1))
	g.Expect(fetched.Alpha).To(BeNumerically(">", 1.0))
}

func TestRunSingleTest_APIError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	defer ts.Close()

	client := &http.Client{
		Transport: &mockAPITransport{serverURL: ts.URL},
	}

	scenario := TestScenario{
		Description:     "test scenario",
		SkillContent:    "skill content",
		SuccessCriteria: "success",
		FailureCriteria: "failure",
	}

	result := runSingleTest(context.Background(), client, "test-key", scenario, false)
	g.Expect(result.Error).ToNot(BeEmpty())
}

func TestRunSingleTest_FailureCriteriaMet(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content": [{"text": "failure pattern detected in response"}]}`))
	}))

	defer ts.Close()

	client := &http.Client{
		Transport: &mockAPITransport{serverURL: ts.URL},
	}

	scenario := TestScenario{
		Description:     "test scenario",
		SkillContent:    "skill content",
		SuccessCriteria: "success criteria not in response",
		FailureCriteria: "failure pattern",
	}

	result := runSingleTest(context.Background(), client, "test-key", scenario, false)
	g.Expect(result.Error).To(BeEmpty())
	g.Expect(result.FailureCriteriaMet).To(BeTrue())
	g.Expect(result.SuccessCriteriaMet).To(BeFalse())
}

func TestRunSingleTest_WithSkill(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"content": [{"text": "success criteria met here"}]}`))
	}))

	defer ts.Close()

	client := &http.Client{
		Transport: &mockAPITransport{serverURL: ts.URL},
	}

	scenario := TestScenario{
		Description:     "test scenario description",
		SkillContent:    "skill content goes here",
		SuccessCriteria: "success criteria",
		FailureCriteria: "failure pattern",
	}

	result := runSingleTest(context.Background(), client, "test-key", scenario, true)
	g.Expect(result.Error).To(BeEmpty())
	g.Expect(result.WithSkill).To(BeTrue())
	g.Expect(result.SuccessCriteriaMet).To(BeTrue())
}

func TestScoreCluster_Empty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	score, err := scoreCluster(db, []ClusterEntry{})
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(score).To(Equal(0.0))
}

func TestScoreCluster_WithEmbeddings(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	_, err = db.Exec(
		`INSERT INTO embeddings (content, source, embedding_id, confidence, retrieval_count) VALUES ('test content', 'test', 1, 0.8, 5)`)
	g.Expect(err).ToNot(HaveOccurred())

	var embedID int64

	err = db.QueryRow("SELECT id FROM embeddings WHERE source = 'test'").Scan(&embedID)
	g.Expect(err).ToNot(HaveOccurred())

	cluster := []ClusterEntry{{ID: embedID, Content: "test content"}}
	score, err := scoreCluster(db, cluster)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(score).To(BeNumerically(">", 0.0))
}

func TestSkillConfidenceMean(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		sc   SkillConfidence
		want float64
	}{
		{"zero", SkillConfidence{0, 0}, 0},
		{"half", SkillConfidence{1, 1}, 0.5},
		{"high", SkillConfidence{3, 1}, 0.75},
		{"all alpha", SkillConfidence{1, 0}, 1.0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewWithT(t)
			g.Expect(tc.sc.Mean()).To(BeNumerically("~", tc.want, 1e-9))
		})
	}
}

func TestSoftDeleteSkill_NonexistentID(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	// Should succeed (UPDATE with no matching row is not an error in SQLite)
	err = softDeleteSkill(db, 9999)
	g.Expect(err).ToNot(HaveOccurred())
}

func TestSoftDeleteSkill_Valid(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	db, err := initEmbeddingsDB(filepath.Join(t.TempDir(), "test.db"))
	g.Expect(err).ToNot(HaveOccurred())

	defer db.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	skill := &GeneratedSkill{
		Slug:            "delete-skill",
		Theme:           "Delete",
		Description:     "desc",
		Content:         "content",
		SourceMemoryIDs: "[]",
		Alpha:           1.0,
		Beta:            1.0,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	id, err := insertSkill(db, skill)
	g.Expect(err).ToNot(HaveOccurred())

	err = softDeleteSkill(db, id)
	g.Expect(err).ToNot(HaveOccurred())

	// Pruned skills not returned by listSkills
	skills, err := listSkills(db)
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(skills).To(BeEmpty())
}

// mockAPITransport redirects all HTTP requests to a test server URL.
type mockAPITransport struct {
	serverURL string
}

func (m *mockAPITransport) RoundTrip(req *http.Request) (*http.Response, error) {
	newURL := m.serverURL + req.URL.Path

	newReq, err := http.NewRequestWithContext(req.Context(), req.Method, newURL, req.Body)
	if err != nil {
		return nil, err
	}

	maps.Copy(newReq.Header, req.Header)

	return http.DefaultTransport.RoundTrip(newReq)
}
