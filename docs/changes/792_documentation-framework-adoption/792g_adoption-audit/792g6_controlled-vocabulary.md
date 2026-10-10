# 792g6 controlled-vocabulary - Adoption Audit

Audit of [792a6 controlled-vocabulary](../792a_product-spec.md#792a6-added-controlled-vocabulary) and the [792b20 vocabulary-engine](../792b_technical-spec.md#792b20-added-vocabulary-engine) requirements that support it. Checked against the code at `dd2bd9e2` and a binary built from it, run on temporary vaults (`/tmp`, temp HOME and XDG). Rows with a code bug leave the spec stating the intended behavior.

| Spec ID | Verdict | Evidence |
| --- | --- | --- |
| [792a6a Definition notes](../792a_product-spec.md#792a6a-definition-notes) | match | vocab_commands.go renderDefinitionNoteContent/mintDefinitionNote/embedDefinitionNote; binary: tags [vocab, vocab/cooking], Exemplars list, sidecar |
| [792a6a1a A definition note carries its self-tag](../792a_product-spec.md#792a6a1-uat-scenarios) | match | vocab_commands.go renderDefinitionNoteContent/mintDefinitionNote/embedDefinitionNote; binary: tags [vocab, vocab/cooking], Exemplars list, sidecar |
| [792a6a1b A definition is never assigned a term](../792a_product-spec.md#792a6a1-uat-scenarios) | match | vocab_commands.go renderDefinitionNoteContent/mintDefinitionNote/embedDefinitionNote; binary: tags [vocab, vocab/cooking], Exemplars list, sidecar |
| [792a6b The family note](../792a_product-spec.md#792a6b-the-family-note) | match | vocab_commands.go ensureVocabFamilyNote, bumpMinorVersion/bumpMajorVersion; binary: 1.0 -> 1.1 -> 2.0 -> 3.0 |
| [792a6b1a Bootstrap creates the family note](../792a_product-spec.md#792a6b1-uat-scenarios) | match | vocab_commands.go ensureVocabFamilyNote, bumpMinorVersion/bumpMajorVersion; binary: 1.0 -> 1.1 -> 2.0 -> 3.0 |
| [792a6b1b The version follows propose and refit](../792a_product-spec.md#792a6b1-uat-scenarios) | match | vocab_commands.go ensureVocabFamilyNote, bumpMinorVersion/bumpMajorVersion; binary: 1.0 -> 1.1 -> 2.0 -> 3.0 |
| [792a6c Categorical tags share one list](../792a_product-spec.md#792a6c-categorical-tags-share-one-list) | match | vocab.go WriteVocabAssignment (vocab entries last, others kept); binary: [work-kind/review, vocab/build-tooling] |
| [792a6c1a A categorical tag survives tagging](../792a_product-spec.md#792a6c1-uat-scenarios) | match | vocab.go WriteVocabAssignment (vocab entries last, others kept); binary: [work-kind/review, vocab/build-tooling] |
| [792a6c1b A categorical tag survives a refit](../792a_product-spec.md#792a6c1-uat-scenarios) | match | vocab.go WriteVocabAssignment (vocab entries last, others kept); binary: [work-kind/review, vocab/build-tooling] |
| [792a6d Write-time vocabulary tagging](../792a_product-spec.md#792a6d-write-time-vocabulary-tagging) | match | vocab.go AssignVocabTerms (floor 0.35, top 3, ties by name), applyVocabAssignmentCore; learn.go:464, amend.go:492, resituate.go:93; binary learn/amend/resituate re-tag |
| [792a6d1a A learned note is tagged with its closest terms](../792a_product-spec.md#792a6d1-uat-scenarios) | match | vocab.go AssignVocabTerms (floor 0.35, top 3, ties by name), applyVocabAssignmentCore; learn.go:464, amend.go:492, resituate.go:93; binary learn/amend/resituate re-tag |
| [792a6d1b A note close to no term gets no tag](../792a_product-spec.md#792a6d1-uat-scenarios) | match | vocab.go AssignVocabTerms (floor 0.35, top 3, ties by name), applyVocabAssignmentCore; learn.go:464, amend.go:492, resituate.go:93; binary learn/amend/resituate re-tag |
| [792a6d1c At most three terms are kept](../792a_product-spec.md#792a6d1-uat-scenarios) | match | vocab.go AssignVocabTerms (floor 0.35, top 3, ties by name), applyVocabAssignmentCore; learn.go:464, amend.go:492, resituate.go:93; binary learn/amend/resituate re-tag |
| [792a6d1d An amend re-tags the note](../792a_product-spec.md#792a6d1-uat-scenarios) | match | vocab.go AssignVocabTerms (floor 0.35, top 3, ties by name), applyVocabAssignmentCore; learn.go:464, amend.go:492, resituate.go:93; binary learn/amend/resituate re-tag |
| [792a6d1e A resituated note is re-tagged](../792a_product-spec.md#792a6d1-uat-scenarios) | match | vocab.go AssignVocabTerms (floor 0.35, top 3, ties by name), applyVocabAssignmentCore; learn.go:464, amend.go:492, resituate.go:93; binary learn/amend/resituate re-tag |
| [792a6e Tagging is skipped without a vocabulary or an embedding](../792a_product-spec.md#792a6e-tagging-is-skipped-without-a-vocabulary-or-an-embedding) | match | vocab.go applyVocabAssignmentCore early returns; vocab_centroids.go loadAssignmentTermVectors |
| [792a6e1a A vault without a vocabulary writes plain notes](../792a_product-spec.md#792a6e1-uat-scenarios) | match | vocab.go applyVocabAssignmentCore early returns; vocab_centroids.go loadAssignmentTermVectors |
| [792a6e1b A note whose embedding failed is not tagged](../792a_product-spec.md#792a6e1-uat-scenarios) | match | vocab.go applyVocabAssignmentCore early returns; vocab_centroids.go loadAssignmentTermVectors |
| [792a6f CRLF notes](../792a_product-spec.md#792a6f-crlf-notes) | match | vocab.go splitFrontmatterAndBody (toLF), vocabAssignmentUnchanged; binary: CRLF note with unchanged tags not rewritten; commit 2026 #789 D8 |
| [792a6f1a Vocab tagging reads a CRLF note](../792a_product-spec.md#792a6f1-uat-scenarios) | match | vocab.go splitFrontmatterAndBody (toLF), vocabAssignmentUnchanged; binary: CRLF note with unchanged tags not rewritten; commit 2026 #789 D8 |
| [792a6g Bootstrapping a vocabulary](../792a_product-spec.md#792a6g-bootstrapping-a-vocabulary) | match | vocab_commands.go:RunVocabBootstrap, retagAllNotesTwoPass (vocab_centroids.go:258); binary prints "vocab bootstrap: 2 terms, 16 notes assigned" |
| [792a6g1a Bootstrap seeds terms and tags existing notes](../792a_product-spec.md#792a6g1-uat-scenarios) | match | vocab_commands.go:RunVocabBootstrap, retagAllNotesTwoPass (vocab_centroids.go:258); binary prints "vocab bootstrap: 2 terms, 16 notes assigned" |
| [792a6g1b A second bootstrap creates nothing new](../792a_product-spec.md#792a6g1-uat-scenarios) | match | vocab_commands.go writeAndEmbedSeedTerms / definitionNoteExistsForTerm |
| [792a6h Bootstrap failures](../792a_product-spec.md#792a6h-bootstrap-failures) | match | vocab_commands.go RunVocabBootstrap (errVocabBootstrapMissingSeed, read/parse before lock); binary: missing flag, unreadable file, bad YAML, non-list all refused |
| [792a6h1a A bad seed is refused](../792a_product-spec.md#792a6h1-uat-scenarios) | match | vocab_commands.go RunVocabBootstrap (errVocabBootstrapMissingSeed, read/parse before lock); binary: missing flag, unreadable file, bad YAML, non-list all refused |
| [792a6i Proposing a term](../792a_product-spec.md#792a6i-proposing-a-term) | match | vocab_commands.go:166 RunVocabPropose, vocab_apply.go:738 stampProposedTermOrigin; binary output and missing-family warning as specified |
| [792a6i1a A proposed term is created](../792a_product-spec.md#792a6i1-uat-scenarios) | match | vocab_commands.go:166 RunVocabPropose, vocab_apply.go:738 stampProposedTermOrigin; binary output and missing-family warning as specified |
| [792a6i1b Propose in a vault without a family note](../792a_product-spec.md#792a6i1-uat-scenarios) | match | vocab_commands.go:176 bumpAndPersistVocabVersion warns; binary prints warning and "created test-term (version -> 1.1)" |
| [792a6i1c Both flags are required](../792a_product-spec.md#792a6i1-uat-scenarios) | match | vocab_commands.go:166 RunVocabPropose, vocab_apply.go:738 stampProposedTermOrigin; binary output and missing-family warning as specified |
| [792a6j Deriving the vocabulary](../792a_product-spec.md#792a6j-deriving-the-vocabulary) | match | vocab_derive.go deriveVocabClusters (K 2..40, floor 0.02, hysteresis 0.02), vocab_commands.go RunVocabRefit; binary |
| [792a6j1a The term count follows the vault's clusters](../792a_product-spec.md#792a6j1-uat-scenarios) | match | vocab_derive.go deriveVocabClusters (K 2..40, floor 0.02, hysteresis 0.02), vocab_commands.go RunVocabRefit; binary |
| [792a6j1b A vault without structure keeps its vocabulary](../792a_product-spec.md#792a6j1-uat-scenarios) | match | vocab_derive.go deriveVocabClusters (K 2..40, floor 0.02, hysteresis 0.02), vocab_commands.go RunVocabRefit; binary |
| [792a6k Term names stay stable](../792a_product-spec.md#792a6k-term-names-stay-stable) | match | vocab_derive.go matchClustersToTerms (0.80, greedy); binary dry-run "matched: ... similarity 1.000" |
| [792a6k1a A cluster keeps its term's name](../792a_product-spec.md#792a6k1-uat-scenarios) | match | vocab_derive.go matchClustersToTerms (0.80, greedy); binary dry-run "matched: ... similarity 1.000" |
| [792a6k1b Unmatched clusters and terms are reported](../792a_product-spec.md#792a6k1-uat-scenarios) | match | vocab_derive.go matchClustersToTerms (0.80, greedy); binary dry-run "matched: ... similarity 1.000" |
| [792a6l Dry run](../792a_product-spec.md#792a6l-dry-run) | match | vocab_apply.go printDerivationDiff; binary dry-run writes nothing |
| [792a6l1a Dry run reports the derivation diff without writing](../792a_product-spec.md#792a6l1-uat-scenarios) | match | vocab_apply.go printDerivationDiff; binary dry-run writes nothing |
| [792a6m Naming requests](../792a_product-spec.md#792a6m-naming-requests) | match | vocab_apply.go buildNamingRequests/clusterExemplars/emitNamingRequests; binary JSON has naming_requests, 3 exemplars, instruction, fingerprint |
| [792a6m1a New clusters are emitted as naming requests](../792a_product-spec.md#792a6m1-uat-scenarios) | match | vocab_apply.go buildNamingRequests/clusterExemplars/emitNamingRequests; binary JSON has naming_requests, 3 exemplars, instruction, fingerprint |
| [792a6n Naming answers](../792a_product-spec.md#792a6n-naming-answers) | match | vocab_apply.go parseRefitNames/validateRefitNameEntry; binary: all six refusals print the specified messages |
| [792a6n1a A bad naming answer is refused](../792a_product-spec.md#792a6n1-uat-scenarios) | match | vocab_apply.go parseRefitNames/validateRefitNameEntry; binary: all six refusals print the specified messages |
| [792a6o Applying a refit](../792a_product-spec.md#792a6o-applying-a-refit) | match | vocab_apply.go applyVocabDerivation; binary: "vocab refit applied: version -> 3.0 (K=4, matched=3, new=1, retired=1)" |
| [792a6o1a A named cluster becomes a term](../792a_product-spec.md#792a6o1-uat-scenarios) | match | vocab_apply.go applyVocabDerivation; binary: "vocab refit applied: version -> 3.0 (K=4, matched=3, new=1, retired=1)" |
| [792a6o1b A refit with nothing to name applies at once](../792a_product-spec.md#792a6o1-uat-scenarios) | match | vocab_apply.go applyVocabDerivation; binary: "vocab refit applied: version -> 3.0 (K=4, matched=3, new=1, retired=1)" |
| [792a6p Retiring a term](../792a_product-spec.md#792a6p-retiring-a-term) | code bug #806 | vocab_apply.go:587 retireVocabTerms deletes note+sidecar, scrubs supersedes:/Supersedes:/body links; frontmatter links (object:) stay dangling (binary: engram check G3) |
| [792a6p1a A retired term is deleted with its references](../792a_product-spec.md#792a6p1-uat-scenarios) | match | scenario behavior verified against binary/code; see requirement row |
| [792a6p1b Retirement leaves no dangling body link](../792a_product-spec.md#792a6p1-uat-scenarios) | code bug #806 | binary: refit then engram check reports G3 dangling for an object: link; body links are unlinked (vocab_apply.go:scrubBodyReferences) |
| [792a6q Proposed terms survive derivation](../792a_product-spec.md#792a6q-proposed-terms-survive-derivation) | code bug #808 | vocab_apply.go derivedCentroidState keeps proposed terms through refit; but vocab_centroids.go:279 writeCentroidsFile (bootstrap/regen) drops origin (binary: gardening retired after re-bootstrap) |
| [792a6q1a An unmatched proposed term is kept](../792a_product-spec.md#792a6q1-uat-scenarios) | match | refit alone keeps an unmatched proposed term (vocab_apply.go derivedCentroidState); loss happens via re-bootstrap, #808 |
| [792a6r Refit takes no term plan](../792a_product-spec.md#792a6r-refit-takes-no-term-plan) | match | targets.go vocabTargets refit flags only --vault/--dry-run/--names; binary: "flag provided but not defined" |
| [792a6r1a The plan flow is gone](../792a_product-spec.md#792a6r1-uat-scenarios) | match | targets.go vocabTargets refit flags only --vault/--dry-run/--names; binary: "flag provided but not defined" |
| [792a6s Refit trigger](../792a_product-spec.md#792a6s-refit-trigger) | code bug #807 | vocab_trigger.go:29, evaluateVocabTriggers; learn.go:736 lists with listMDFromFS (counts starter README, binary: note_count 2 for one note); thresholds 40/14 match |
| [792a6s1a Growth marks a refit pending](../792a_product-spec.md#792a6s1-uat-scenarios) | match | vocab_trigger.go evaluateVocabTriggers; binary: "verdict: REFIT_PENDING (growth: 63 notes, 20 days)" (63 itself inflated by #807) |
| [792a6s1b Growth without enough time does not trigger](../792a_product-spec.md#792a6s1-uat-scenarios) | match | vocab_trigger.go:evaluateVocabTriggers growth >= 40 && days >= 14 |
| [792a6s1c A first write records the baseline](../792a_product-spec.md#792a6s1-uat-scenarios) | code bug #807 | baseline recorded, but one too high when the vault holds the starter README (binary: 2 for one note) |
| [792a6s1d A high untagged rate alone does not trigger](../792a_product-spec.md#792a6s1-uat-scenarios) | match | vocab_commands.go printStatsReport [high] is diagnostic only; evaluateVocabTriggers ignores it |
| [792a6t Vocabulary statistics](../792a_product-spec.md#792a6t-vocabulary-statistics) | match | vocab_commands.go RunVocabStats, printStatsReport, collectVaultStats; qa.go:181 countQAPairs; binary |
| [792a6t1a Stats on an empty vault](../792a_product-spec.md#792a6t1-uat-scenarios) | match | vocab_commands.go RunVocabStats, printStatsReport, collectVaultStats; qa.go:181 countQAPairs; binary |
| [792a6t1b Hubs and orphans are flagged](../792a_product-spec.md#792a6t1-uat-scenarios) | match | vocab_commands.go RunVocabStats, printStatsReport, collectVaultStats; qa.go:181 countQAPairs; binary |
| [792a6t1c Questions and definitions are not member notes](../792a_product-spec.md#792a6t1-uat-scenarios) | match | vocab_commands.go RunVocabStats, printStatsReport, collectVaultStats; qa.go:181 countQAPairs; binary |
| [792a6t1d A pending refit is the verdict](../792a_product-spec.md#792a6t1-uat-scenarios) | match | vocab_commands.go RunVocabStats, printStatsReport, collectVaultStats; qa.go:181 countQAPairs; binary |
| [792a6u Self-tag backfill](../792a_product-spec.md#792a6u-self-tag-backfill) | match | vocab_commands.go runVocabTagDefinitions/processVocabDefinitionNote; binary output lines as specified |
| [792a6u1a Backfill adds missing self-tags](../792a_product-spec.md#792a6u1-uat-scenarios) | match | vocab_commands.go runVocabTagDefinitions/processVocabDefinitionNote; binary output lines as specified |
| [792a6u1b Backfill is idempotent](../792a_product-spec.md#792a6u1-uat-scenarios) | match | vocab_commands.go runVocabTagDefinitions/processVocabDefinitionNote; binary output lines as specified |
| [792a6v Update notices](../792a_product-spec.md#792a6v-update-notices) | doc fixed | update.go:68-72 notice strings, :416 oldVocabFilesPresent, :673 vocabDefinitionsMissingSelfTags; spec rendering of the two notice lines was garbled (missing spaces), rewritten |
| [792a6v1a An old-format vault is flagged](../792a_product-spec.md#792a6v1-uat-scenarios) | match | update.go:416 oldVocabFilesPresent; Test792b20c22a1 (update not run end to end: installs from network) |
| [792a6v1b A missing self-tag is flagged](../792a_product-spec.md#792a6v1-uat-scenarios) | match | update.go:673 vocabDefinitionsMissingSelfTags; Test792b20c22a2 |
| [792a6v1c A current vault gets no vocab notice](../792a_product-spec.md#792a6v1-uat-scenarios) | match | same detectors return false on a current vault; Test792b20c22a1/a2 |
| [792a6w Migrating an old-format vocabulary](../792a_product-spec.md#792a6w-migrating-an-old-format-vocabulary) | code bug #809 | vocab_regen.go:199 regenVocab; update.go:1050 writeVocabRegenReport text matches; counts planned (:210) not actual deletions, and no vault lock (#809) |
| [792a6w1a An old-format vocabulary is migrated](../792a_product-spec.md#792a6w1-uat-scenarios) | match | vocab_regen.go regenVocab; Test792b20c21a1_RegenVocab_RealRun (not run end to end: engram update installs from the network) |
| [792a6w1b A dry run changes nothing](../792a_product-spec.md#792a6w1-uat-scenarios) | match | vocab_regen.go regenVocab dry-run; update.go:1060 "[dry-run]" prefix; Test792b20c21a2 |
| [792a6w1c Nothing to migrate](../792a_product-spec.md#792a6w1-uat-scenarios) | match | update.go:1051 "nothing to regenerate"; Test792b20c21a3 |
| [792b20c1 Definition recognition](../792b_technical-spec.md#792b20c1-definition-recognition) | match | vocab.go:isVocabDefinitionNote, vocab_commands.go:termFromDefinitionSlug (:slugFromNoteFilename), vocabTermsFromTags |
| [792b20c2 Term vectors](../792b_technical-spec.md#792b20c2-term-vectors) | match | vocab_commands.go:loadTermVectors (first wins), vocab_centroids.go:loadAssignmentTermVectors (model id guard) |
| [792b20c3 Term selection](../792b_technical-spec.md#792b20c3-term-selection) | match | vocab.go:AssignVocabTerms, sortTermScores |
| [792b20c4 Rewriting a note's vocabulary tags](../792b_technical-spec.md#792b20c4-rewriting-a-notes-vocabulary-tags) | match | vocab.go:WriteVocabAssignment, rewriteTagsFrontmatterSplit |
| [792b20c5 Tagging a written note](../792b_technical-spec.md#792b20c5-tagging-a-written-note) | match | vocab.go:applyVocabAssignmentCore, vocabAssignmentUnchanged |
| [792b20c6 Refit check](../792b_technical-spec.md#792b20c6-refit-check) | code bug #807 | vocab_trigger.go:29 counts every .md the caller lists, including the starter README (learn.go:736); bootstrap count (vault_fs.go:25) excludes it, so scenario 792b20c6a6 (counts agree) fails on a real vault |
| [792b20c7 Statistics](../792b_technical-spec.md#792b20c7-statistics) | match | vocab_commands.go:RunVocabStats/collectVaultStats/printStatsReport; qa.go:181 |
| [792b20c8 Writing a definition note](../792b_technical-spec.md#792b20c8-writing-a-definition-note) | match | vocab_commands.go:mintDefinitionNote/renderDefinitionNoteContent |
| [792b20c9 The family note and the version](../792b_technical-spec.md#792b20c9-the-family-note-and-the-version) | match | vocab_commands.go:ensureVocabFamilyNote, loadCurrentVocabVersion, rewriteVocabVersionKey |
| [792b20c10 Bootstrap](../792b_technical-spec.md#792b20c10-bootstrap) | match | vocab_commands.go:RunVocabBootstrap (seed before lock) |
| [792b20c11 Two-pass re-tagging](../792b_technical-spec.md#792b20c11-two-pass-re-tagging) | code bug #808 | vocab_centroids.go:258 retagAllNotesTwoPass matches the text, but writes the record without origins, so bootstrap/regen erase `proposed` (spec 792b20c12 relies on it surviving) |
| [792b20c12 Propose](../792b_technical-spec.md#792b20c12-propose) | code bug #805 | vocab_commands.go:166 RunVocabPropose: matches the text, but a repeat proposal writes a second definition and bumps the version again |
| [792b20c13 Deriving clusters](../792b_technical-spec.md#792b20c13-deriving-clusters) | match | vocab_derive.go:deriveVocabClusters; vocab_apply.go:computeDerivationFingerprint |
| [792b20c14 Matching clusters to terms](../792b_technical-spec.md#792b20c14-matching-clusters-to-terms) | match | vocab_derive.go:matchClustersToTerms |
| [792b20c15 Refit routing](../792b_technical-spec.md#792b20c15-refit-routing) | match | vocab_commands.go:RunVocabRefit (lock held for whole run) |
| [792b20c16 Naming requests](../792b_technical-spec.md#792b20c16-naming-requests) | match | vocab_apply.go:buildNamingRequests/clusterExemplars/noteSnippet |
| [792b20c17 Naming answers](../792b_technical-spec.md#792b20c17-naming-answers) | match | vocab_apply.go:parseRefitNames, namedClustersToSeedTerms |
| [792b20c18 Applying a derivation](../792b_technical-spec.md#792b20c18-applying-a-derivation) | match | vocab_apply.go:applyVocabDerivation, derivedCentroidState |
| [792b20c19 Retiring terms](../792b_technical-spec.md#792b20c19-retiring-terms) | code bug #806 | vocab_apply.go:587 matches the text (bodies and supersedes:), but frontmatter links remain dangling after retirement |
| [792b20c20 Self-tag backfill](../792b_technical-spec.md#792b20c20-self-tag-backfill) | match | vocab_commands.go:runVocabTagDefinitions lists then locks, processVocabDefinitionNote |
| [792b20c21 Migrating an old-format vocabulary](../792b_technical-spec.md#792b20c21-migrating-an-old-format-vocabulary) | code bug #809 | vocab_regen.go:199 regenVocab: report counts planned deletions (:210, :229 discards removeOldVocabFiles count), no vault lock |
| [792b20c22 Repairs a vault needs](../792b_technical-spec.md#792b20c22-repairs-a-vault-needs) | match | update.go:416 oldVocabFilesPresent, :673 vocabDefinitionsMissingSelfTags |

## Summary

Verdict counts: code bug 11, doc fixed 1, match 82.

Issues filed:

- #805 `vocab propose` duplicates an existing term (792a6i, 792b20c12).
- #806 retirement leaves frontmatter links dangling (792a6p, 792b20c19).
- #807 refit trigger counts the starter README (792a6s, 792b20c6). Not covered by #796, which does not state the vocab count.
- #808 bootstrap/regen drops the `proposed` origin (792a6q, 792b20c11).
- #809 `update --regen-vocab` takes no vault lock and reports planned deletions (792a6w, 792b20c21).

Doc fix: [792a6v Update notices](../792a_product-spec.md#792a6v-update-notices) had its two notice lines rendered with missing spaces around nested backticks; now double-backtick code spans with the exact text of `update.go`.

Notes:

- `engram update` was not run end to end: it clones and builds from the network. Update notices and `--regen-vocab` are checked by code and by the `Test792b20c21*`/`c22*` unit tests.
- 792d87 says `engram vocab <subcommand> --help` prints the group help rather than the subcommand flags. The binary prints the group help followed by the subcommand help with its flags. The behavior is not specified in 792a6, so no row or issue.
- 792d89: `vocab tag-definitions` lists the vault before taking the lock; [792b20c20 Self-tag backfill](../792b_technical-spec.md#792b20c20-self-tag-backfill) states exactly that, so it is a match.
- A `supersedes:` entry written without `.md` is not scrubbed on retirement (the convention keeps the full filename); mentioned in #806.
- [792a6g Bootstrapping a vocabulary](../792a_product-spec.md#792a6g-bootstrapping-a-vocabulary) describes the centroid pass as over notes that are not questions, while the code also lets question notes with sidecars contribute to the means (tagging still skips them). Matches [792b20c11 Two-pass re-tagging](../792b_technical-spec.md#792b20c11-two-pass-re-tagging), so left as is.
