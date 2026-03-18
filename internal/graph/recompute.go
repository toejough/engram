package graph

import (
	"fmt"

	"engram/internal/registry"
)

// RecomputeMergeLinks re-computes concept_overlap and content_similarity links
// for the merged memory, removes stale links pointing to the absorbed memory,
// and preserves co_surfacing and evaluation_correlation links (P5f).
func (b *Builder) RecomputeMergeLinks(result MergeResult, linker RegistryLinker) error {
	entries, err := linker.List()
	if err != nil {
		return fmt.Errorf("listing registry entries: %w", err)
	}

	// Step 1: Remove absorbed-memory links from all entries.
	if result.AbsorbedMemoryID != "" {
		err = removeAbsorbedLinks(result.AbsorbedMemoryID, entries, linker)
		if err != nil {
			return err
		}
	}

	// Step 2: Find the merged entry's current links and build corpus.
	var existingLinks []registry.Link

	corpus := make([]registry.InstructionEntry, 0, len(entries))

	for _, entry := range entries {
		if entry.ID == result.MergedMemoryID {
			existingLinks = entry.Links
		} else {
			corpus = append(corpus, entry)
		}
	}

	// Step 3: Keep only co_surfacing and evaluation_correlation links.
	preserved := make([]registry.Link, 0, len(existingLinks))

	for _, link := range existingLinks {
		if link.Basis == "co_surfacing" || link.Basis == "evaluation_correlation" {
			preserved = append(preserved, link)
		}
	}

	// Step 4: Build merged entry for link computation.
	// Keywords are set from MergedConceptSet so BuildConceptOverlap uses the
	// post-merge keyword set for Jaccard, not just tokenized title+content (REQ-138).
	mergedEntry := registry.InstructionEntry{
		ID:       result.MergedMemoryID,
		Title:    result.MergedTitle,
		Content:  result.MergedContent,
		Keywords: result.MergedConceptSet,
	}

	// Step 5: Compute new concept_overlap and content_similarity links.
	newConceptLinks := b.BuildConceptOverlap(mergedEntry, corpus)
	newContentLinks := b.BuildContentSimilarity(mergedEntry, corpus)

	// Step 6: Merge preserved + new links.
	updatedLinks := make([]registry.Link, 0, len(preserved)+len(newConceptLinks)+len(newContentLinks))
	updatedLinks = append(updatedLinks, preserved...)
	updatedLinks = append(updatedLinks, newConceptLinks...)
	updatedLinks = append(updatedLinks, newContentLinks...)

	// Step 7: Update links for merged entry.
	err = linker.UpdateLinks(result.MergedMemoryID, updatedLinks)
	if err != nil {
		return fmt.Errorf("updating merged memory links: %w", err)
	}

	return nil
}

// MergeResult carries the outcome of a merge-on-write operation (P5f).
// It is the contract passed to the link recomputer after processMerge completes.
type MergeResult struct {
	MergedMemoryID   string   // registry ID (file path) of the surviving merged memory
	AbsorbedMemoryID string   // registry ID of the absorbed memory, or empty if never registered
	MergedTitle      string   // post-merge title
	MergedContent    string   // post-merge principle text
	MergedConceptSet []string // post-merge concept set
}

// RegistryLinker is the DI boundary for registry access during link recomputation (ARCH-7).
type RegistryLinker interface {
	List() ([]registry.InstructionEntry, error)
	UpdateLinks(id string, links []registry.Link) error
}

// removeAbsorbedLinks removes all links pointing to absorbedID from affected entries.
func removeAbsorbedLinks(absorbedID string, entries []registry.InstructionEntry, linker RegistryLinker) error {
	for _, entry := range entries {
		if entry.ID == absorbedID {
			continue
		}

		filtered := make([]registry.Link, 0, len(entry.Links))
		removed := false

		for _, link := range entry.Links {
			if link.Target == absorbedID {
				removed = true

				continue
			}

			filtered = append(filtered, link)
		}

		if !removed {
			continue
		}

		err := linker.UpdateLinks(entry.ID, filtered)
		if err != nil {
			return fmt.Errorf("removing absorbed links from %s: %w", entry.ID, err)
		}
	}

	return nil
}
