package chunk_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/onsi/gomega"
	"github.com/toejough/engram/internal/chunk"
)

func TestDecodeRecordsPreservesIngestedAt(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	ingestedAt := time.Date(2026, 3, 10, 8, 0, 0, 0, time.UTC)
	rec := chunk.Record{
		Source:      "s.jsonl",
		Anchor:      "turn-2",
		ContentHash: "sha256:def",
		Text:        "world",
		Vector:      []float32{0.2},
		IngestedAt:  ingestedAt,
	}

	encoded, err := chunk.EncodeRecords([]chunk.Record{rec})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	decoded, err := chunk.DecodeRecords(encoded)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(decoded).To(gomega.HaveLen(1))

	if len(decoded) < 1 {
		return
	}

	g.Expect(decoded[0].IngestedAt).To(gomega.Equal(ingestedAt))
}

func TestDecodeRecordsZeroIngestedAtWhenAbsent(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	// JSONL without ingested_at (legacy record — migration backfill case)
	line := `{"source":"s.jsonl","anchor":"turn-1","content_hash":"sha256:abc","text":"hi","vector":[0.1]}` + "\n"

	decoded, err := chunk.DecodeRecords([]byte(line))
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(decoded).To(gomega.HaveLen(1))

	if len(decoded) < 1 {
		return
	}

	g.Expect(decoded[0].IngestedAt.IsZero()).To(gomega.BeTrue(),
		"absent ingested_at must decode as zero time (migration sentinel)")
}

func TestRecordIngestedAtRoundTrips(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	ingestedAt := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	rec := chunk.Record{
		Source:      "s.jsonl",
		Anchor:      "turn-1",
		ContentHash: "sha256:abc",
		Text:        "hello",
		Vector:      []float32{0.1},
		IngestedAt:  ingestedAt,
	}

	data, err := json.Marshal(rec)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	var got chunk.Record

	err = json.Unmarshal(data, &got)
	g.Expect(err).NotTo(gomega.HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(got.IngestedAt).To(gomega.Equal(ingestedAt))
}
