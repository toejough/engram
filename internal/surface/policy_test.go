package surface_test

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"engram/internal/frecency"
	"engram/internal/policy"
	"engram/internal/surface"
)

// TestSurfacingPolicyToFrecencyOpts verifies that active surfacing policies are
// converted to the correct frecency options.
func TestSurfacingPolicyToFrecencyOpts(t *testing.T) {
	t.Parallel()

	t.Run("empty file returns no options", func(t *testing.T) {
		t.Parallel()

		g := NewGomegaWithT(t)
		policyFile := &policy.File{}

		opts := surface.ExportSurfacingPolicyToFrecencyOpts(policyFile)

		g.Expect(opts).To(BeEmpty())
	})

	t.Run("inactive surfacing policy is ignored", func(t *testing.T) {
		t.Parallel()

		g := NewGomegaWithT(t)
		policyFile := &policy.File{
			Policies: []policy.Policy{
				{
					Dimension: policy.DimensionSurfacing,
					Parameter: "wEff",
					Value:     0.9,
					Status:    policy.StatusProposed,
				},
			},
		}

		opts := surface.ExportSurfacingPolicyToFrecencyOpts(policyFile)

		g.Expect(opts).To(BeEmpty())
	})

	t.Run("non-surfacing active policy is ignored", func(t *testing.T) {
		t.Parallel()

		g := NewGomegaWithT(t)
		policyFile := &policy.File{
			Policies: []policy.Policy{
				{
					Dimension: policy.DimensionExtraction,
					Parameter: "wEff",
					Value:     0.9,
					Status:    policy.StatusActive,
				},
			},
		}

		opts := surface.ExportSurfacingPolicyToFrecencyOpts(policyFile)

		g.Expect(opts).To(BeEmpty())
	})

	t.Run("active wEff policy produces WithWEff option", func(t *testing.T) {
		t.Parallel()

		g := NewGomegaWithT(t)
		policyFile := &policy.File{
			Policies: []policy.Policy{
				{
					Dimension: policy.DimensionSurfacing,
					Parameter: "wEff",
					Value:     0.9,
					Status:    policy.StatusActive,
				},
			},
		}

		opts := surface.ExportSurfacingPolicyToFrecencyOpts(policyFile)

		g.Expect(opts).To(HaveLen(1))

		scorer := frecency.New(time.Now(), 10, opts...)
		g.Expect(scorer).NotTo(BeNil())
	})

	t.Run("active wFreq policy produces WithWFreq option", func(t *testing.T) {
		t.Parallel()

		g := NewGomegaWithT(t)
		policyFile := &policy.File{
			Policies: []policy.Policy{
				{
					Dimension: policy.DimensionSurfacing,
					Parameter: "wFreq",
					Value:     0.5,
					Status:    policy.StatusActive,
				},
			},
		}

		opts := surface.ExportSurfacingPolicyToFrecencyOpts(policyFile)

		g.Expect(opts).To(HaveLen(1))
	})

	t.Run("active wTier policy produces WithWTier option", func(t *testing.T) {
		t.Parallel()

		g := NewGomegaWithT(t)
		policyFile := &policy.File{
			Policies: []policy.Policy{
				{
					Dimension: policy.DimensionSurfacing,
					Parameter: "wTier",
					Value:     0.3,
					Status:    policy.StatusActive,
				},
			},
		}

		opts := surface.ExportSurfacingPolicyToFrecencyOpts(policyFile)

		g.Expect(opts).To(HaveLen(1))
	})

	t.Run("active tierABoost policy produces WithTierABoost option", func(t *testing.T) {
		t.Parallel()

		g := NewGomegaWithT(t)
		policyFile := &policy.File{
			Policies: []policy.Policy{
				{
					Dimension: policy.DimensionSurfacing,
					Parameter: "tierABoost",
					Value:     1.5,
					Status:    policy.StatusActive,
				},
			},
		}

		opts := surface.ExportSurfacingPolicyToFrecencyOpts(policyFile)

		g.Expect(opts).To(HaveLen(1))
	})

	t.Run("active tierBBoost policy produces WithTierBBoost option", func(t *testing.T) {
		t.Parallel()

		g := NewGomegaWithT(t)
		policyFile := &policy.File{
			Policies: []policy.Policy{
				{
					Dimension: policy.DimensionSurfacing,
					Parameter: "tierBBoost",
					Value:     1.2,
					Status:    policy.StatusActive,
				},
			},
		}

		opts := surface.ExportSurfacingPolicyToFrecencyOpts(policyFile)

		g.Expect(opts).To(HaveLen(1))
	})

	t.Run("unknown parameter is skipped", func(t *testing.T) {
		t.Parallel()

		g := NewGomegaWithT(t)
		policyFile := &policy.File{
			Policies: []policy.Policy{
				{
					Dimension: policy.DimensionSurfacing,
					Parameter: "unknownParam",
					Value:     1.0,
					Status:    policy.StatusActive,
				},
			},
		}

		opts := surface.ExportSurfacingPolicyToFrecencyOpts(policyFile)

		g.Expect(opts).To(BeEmpty())
	})

	t.Run("multiple active surfacing policies produce multiple options", func(t *testing.T) {
		t.Parallel()

		g := NewGomegaWithT(t)
		policyFile := &policy.File{
			Policies: []policy.Policy{
				{
					Dimension: policy.DimensionSurfacing,
					Parameter: "wEff",
					Value:     0.9,
					Status:    policy.StatusActive,
				},
				{
					Dimension: policy.DimensionSurfacing,
					Parameter: "wFreq",
					Value:     0.4,
					Status:    policy.StatusActive,
				},
				{
					Dimension: policy.DimensionExtraction,
					Parameter: "wEff",
					Value:     0.7,
					Status:    policy.StatusActive,
				},
			},
		}

		opts := surface.ExportSurfacingPolicyToFrecencyOpts(policyFile)

		g.Expect(opts).To(HaveLen(2))
	})
}
