package graph

import (
	"fmt"

	"engram/internal/memory"
)

// RecomputeMergeLinks re-computes concept_overlap and content_similarity links
// for the merged memory, removes stale links pointing to the absorbed memory,
// and preserves co_surfacing and evaluation_correlation links (P5f).
func (b *Builder) RecomputeMergeLinks(result MergeResult, lister MemoryLister, writer LinkWriter) error {
	records, err := lister.ListAll()
	if err != nil {
		return fmt.Errorf("listing memory records: %w", err)
	}

	// Step 1: Remove absorbed-memory links from all entries.
	if result.AbsorbedMemoryID != "" {
		err = removeAbsorbedLinks(result.AbsorbedMemoryID, records, writer)
		if err != nil {
			return err
		}
	}

	// Step 2: Find the merged entry's current links and build corpus.
	var existingLinks []memory.LinkRecord

	corpus := make([]memory.StoredRecord, 0, len(records))

	for _, stored := range records {
		if stored.Path == result.MergedMemoryID {
			existingLinks = stored.Record.Links
		} else {
			corpus = append(corpus, stored)
		}
	}

	// Step 3: Keep only co_surfacing and evaluation_correlation links.
	preserved := make([]memory.LinkRecord, 0, len(existingLinks))

	for _, link := range existingLinks {
		if link.Basis == "co_surfacing" || link.Basis == "evaluation_correlation" {
			preserved = append(preserved, link)
		}
	}

	// Step 4: Build merged entry for link computation.
	// Keywords are set from MergedConceptSet so BuildConceptOverlap uses the
	// post-merge keyword set for Jaccard, not just tokenized title+content (REQ-138).
	mergedRecord := memory.MemoryRecord{
		Title:    result.MergedTitle,
		Content:  result.MergedContent,
		Keywords: result.MergedConceptSet,
	}

	// Step 5: Compute new concept_overlap and content_similarity links.
	newConceptLinks := b.BuildConceptOverlap(result.MergedMemoryID, mergedRecord, corpus)
	newContentLinks := b.BuildContentSimilarity(result.MergedMemoryID, mergedRecord, corpus)

	// Step 6: Merge preserved + new links.
	updatedLinks := make([]memory.LinkRecord, 0, len(preserved)+len(newConceptLinks)+len(newContentLinks))
	updatedLinks = append(updatedLinks, preserved...)
	updatedLinks = append(updatedLinks, newConceptLinks...)
	updatedLinks = append(updatedLinks, newContentLinks...)

	// Step 7: Update links for merged entry.
	err = writer.WriteLinks(result.MergedMemoryID, updatedLinks)
	if err != nil {
		return fmt.Errorf("updating merged memory links: %w", err)
	}

	return nil
}

// LinkWriter persists links for a memory file.
type LinkWriter interface {
	WriteLinks(path string, links []memory.LinkRecord) error
}

// MemoryLister lists all stored memory records.
type MemoryLister interface {
	ListAll() ([]memory.StoredRecord, error)
}

// MergeResult carries the outcome of a merge-on-write operation (P5f).
// It is the contract passed to the link recomputer after processMerge completes.
type MergeResult struct {
	MergedMemoryID   string   // file path of the surviving merged memory
	AbsorbedMemoryID string   // file path of the absorbed memory, or empty if never registered
	MergedTitle      string   // post-merge title
	MergedContent    string   // post-merge principle text
	MergedConceptSet []string // post-merge concept set
}

// removeAbsorbedLinks removes all links pointing to absorbedID from affected entries.
func removeAbsorbedLinks(absorbedID string, records []memory.StoredRecord, writer LinkWriter) error {
	for _, stored := range records {
		if stored.Path == absorbedID {
			continue
		}

		filtered := make([]memory.LinkRecord, 0, len(stored.Record.Links))
		removed := false

		for _, link := range stored.Record.Links {
			if link.Target == absorbedID {
				removed = true

				continue
			}

			filtered = append(filtered, link)
		}

		if !removed {
			continue
		}

		err := writer.WriteLinks(stored.Path, filtered)
		if err != nil {
			return fmt.Errorf("removing absorbed links from %s: %w", stored.Path, err)
		}
	}

	return nil
}
