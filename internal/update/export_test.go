package update

// Exported variables.
var (
	ExportApplyEngramSyncOps            = applyEngramSyncOps
	ExportApplyManifestModeDeletion     = applyManifestModeDeletion
	ExportApplyPrunedDirs               = applyPrunedDirs
	ExportCleanupDanglingLinksInDir     = cleanupDanglingLinksInDir
	ExportDetectHarnesses               = detectHarnesses
	ExportGuidanceImportPrefixes        = guidanceImportPrefixes
	ExportLexicallyResolveSymlinkTarget = lexicallyResolveSymlinkTarget
	ExportListSubtreeFiles              = listSubtreeFiles
	ExportMaterializeOrAdopt            = materializeOrAdopt
	ExportMaterializeSymlink            = materializeSymlink
	ExportPathWithinRoot                = pathWithinRoot
	ExportPlanEngramRootSync            = planEngramRootSync
	ExportPlanGuidanceCopies            = planGuidanceCopies
	ExportPlanSkillCopies               = planSkillCopies
	ExportPruneEmptyDirs                = pruneEmptyDirs
	ExportSurfaceStrays                 = surfaceStrays
	ExportWalkUpForModule               = walkUpForModule
)

// Exported types.
type ExportIntendedRootFile = intendedRootFile

// ExportApplyOps drives Updater.applyOps directly with a caller-supplied
// harness list, bypassing detectHarnesses/supportedHarnesses (which are
// hardcoded to the two currently-supported, currently-all-symlink-mode
// harnesses). This is the only way tests can exercise the DeployModeManifest
// fallthrough branch of applyForHarness, since Run always resolves harnesses
// through supportedHarnesses.
func ExportApplyOps(
	updater *Updater,
	harnesses []HarnessSpec,
	home string,
	skillOps, guidanceOps []CopyOp,
	guidanceManaged, dryRun bool,
) []HarnessReport {
	return updater.applyOps(harnesses, home, skillOps, guidanceOps, guidanceManaged, dryRun)
}
