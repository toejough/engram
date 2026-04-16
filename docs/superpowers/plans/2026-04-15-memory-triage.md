# Memory Triage Results

## Summary
- Fine: 193
- Rewrite: 106
- Delete (situational): 67
- Delete (content): 40
- Total: 406

## Fine (no changes)

| Memory | Current Situation |
|---|---|
| atomic-toml-write-temp-file-strategy | "writing Go code that creates temporary files or performs file operations" |
| autonomous-task-completion-with-reporting | "when user requests autonomous execution of development tasks with a final completion report" |
| avoid-hooks-in-skill-communication-protocols | "when designing file-based communication protocols for multi-agent coordination" |
| batch-issue-analysis-with-fixes | "During spec review, issue analysis, or comprehensive document review processes" |
| check-before-force-operations | "When about to run any command with a --force flag or destructive flag (git force push, git reset --hard, git worktree remove --force, rm -rf, etc.)" |
| consult-plan-before-sequencing | "When deciding what work to tackle next in a project with documented plans and issue trackers" |
| coordinator-role-boundaries | "when working in multi-agent coordination scenarios" |
| declare-complete-requires-wiring | "When completing implementation of a feature, command, or system component and about to declare the work done or close an issue" |
| defer-with-explicit-tracking-and-confirmation | "When marking work as complete or done" |
| defer-work-with-communication | "When deferring work items or changing the status of tasks, issues, or project deliverables" |
| design-before-incremental-execution | "when starting a new feature, system, or major refactoring" |
| di-patterns-io-injection-testing-strategy | "When implementing CLI commands or functions that involve IO operations (file operations, HTTP calls, etc.)" |
| dispatch-plan-reviewer-before-execution | "after writing or receiving a plan document and before executing it" |
| file-comms-watch-loop-maintenance | "When working with multi-agent coordination via chat.toml file-based communication and completing assigned tasks" |
| file-issue-per-finding | "When conducting audits or analysis that reveals multiple distinct problems or inaccuracies" |
| file-issues-in-correct-repository | "when filing issues or bugs in repositories" |
| framework-agnostic-layers | "when building data layers or business logic components" |
| git-commit-push-cleanup | "When user requests 'commit and push' operations" |
| git-status-clean-before-commit | "when executing git commit/push operations or claiming repository state is clean" |
| git-status-verification-habit | "Before any commit, push, or claim that working directory is clean" |
| git-status-verification-workflow | "When user requests 'commit and push' or when claiming git working directory is clean" |
| gomega-assertion-standard | "when writing test assertions in Go" |
| ground-up-file-review-criteria | "when asked to re-examine a file or document that was previously reviewed" |
| hook-lifecycle-cleanup | "when refactoring hooks and a hook becomes empty or no longer performs any operation" |
| issue-workflow-parallel-plan-argue-execute | "When working through a list of GitHub issues or bugs that need planning, implementation, and review" |
| lead-must-not-fabricate-agent-output | "When acting as lead/coordinator in multi-agent coordination and relaying agent results or work to the user" |
| lead-parrot-must-be-exact-quote | "When acting as lead/coordinator and posting user input to the engram chat as a parrot/info message" |
| maintain-user-orientation-complex-operations | "When executing complex multi-agent coordination or detailed implementation work with multiple moving parts" |
| memory-file-deletion-safety | "when deleting or modifying memory files in engram system" |
| node-deletion-over-deprecation | "When cleaning up or removing obsolete nodes, entities, or data structures in a system" |
| overlap-detection-standing-validation | "When running validation or quality checks on codebases or component libraries where duplication might exist" |
| parallel-execution-preference | "When executing a defined task list where tasks are independent and can run in parallel" |
| parallelize-work-across-multi-agent-teams | "When executing substantial project work that can be decomposed into independent or loosely coupled tasks" |
| permanent-docs-directory-structure | "When organizing project documentation structure and deciding where to place permanent specs, design documents, or other long-term reference materials" |
| plan-execute-review-repeat-workflow | "When handling complex multi-issue work or tasks requiring subagent delegation and implementation" |
| plan-execute-review-report-cycle | "Working on complex multi-step tasks that can be broken down and delegated to subagents" |
| plan-files-docs-directory | "When creating or organizing plan files in a project repository" |
| plan-storage-and-naming-convention | "When writing implementation plans using the writing-plans skill" |
| planning-approval-gate | "When a complex task requires subagent dispatch after brainstorming or planning phase to implement a multi-step solution" |
| planning-precedes-implementation | "When building skills, tools, or infrastructure components that require user collaboration and decision-making about architecture, scope, or design approach" |
| plugin-hooks-universal-installation | "When implementing plugin hooks intended to work universally across all user installations without requiring manual setup steps" |
| plugin-stop-hook-json-exit-code | "When implementing any Claude Code hook that outputs JSON -- especially for lifecycle events like Stop, PostToolUse, or UserPromptSubmit" |
| preexisting-failures-require-followup | "When encountering test failures, build errors, or other technical issues during development work, regardless of whether they appear to be caused by current changes" |
| premortem-analysis-planning | "About to start a new project, implement a major feature, or make a significant decision without having analyzed potential failure modes" |
| premortem-standard-feature-request-generation | "When conducting premortems to anticipate failure scenarios for upcoming features or development cycles" |
| pressure-test-skill-updates-always | "When updating or implementing skill behavior code changes" |
| proceed-with-clear-instructions-not-seek-confirmation | "When user has provided clear instructions to proceed with a task list and there are implementation details that seem to need clarification but the user's intent is clear" |
| progressive-disclosure-semantic-separation | "When designing user interfaces or information architecture that could present multiple types of data or functionality to users simultaneously" |
| propose-approaches-with-tradeoffs | "When asked to design or architect a solution to a complex technical problem with multiple possible implementation paths" |
| read-only-targets-first | "Building new tooling or automation that includes both read-only and mutation capabilities" |
| readme-alpha-disclaimer-requirement | "When working on README files for projects in alpha or early development stages" |
| remove-nonfunctional-code-principle | "When analyzing code and discovering components that were never functional or operational" |
| remove-unused-cli-flags | "When implementing CLI commands with flag parameters" |
| repeatable-scriptable-edge-fixes | "When encountering edge cases, bugs, or system inconsistencies that require fixing" |
| resilient-error-handling | "Working on multi-step workflows that depend on external API calls when encountering API errors or service failures" |
| restart-failed-tool-calls | "When a tool call encounters an error or failure during execution" |
| retro-format-convention | "When creating retrospective documents or similar recurring project artifacts that follow established structural patterns" |
| retro-premortem-ticket-filing-default | "During retrospectives and premortems when valid, actionable insights are identified that could improve the codebase or process" |
| retrospective-before-new-work | "Development cycle has completed and new work or next iteration is being planned" |
| reusable-targeted-skills | "Working on a development project where the same types of interactions or conversational patterns are needed repeatedly across multiple work sessions" |
| reuse-deduplicate-plan-first | "When planning new development work or implementing features that may overlap with existing functionality" |
| review-all-work-before-presenting-to-user | "When the lead agent is about to report completed work (plans, implementations, research, triage) to the user as done or ready" |
| scan-existing-nodes-before-adding | "When adding new nodes or structure to a knowledge management system that has existing nodes and content" |
| scope-boundary-enforcement | "Working as an executor receiving assignments that specify particular files or directories to edit" |
| self-investigate-before-asking | "When encountering unexpected behavior or performance issues where actual results don't match expected outcomes and debugging tools or logs are available" |
| self-validate-before-review | "When fixes or changes have been completed and the agent is preparing to request user review or validation" |
| skill-announcement-protocol | "When beginning to execute an implementation plan using the writing-plans skill" |
| skill-claims-require-behavior-verification | "When evaluating skills, processes, or methodologies that claim to prevent problems or ensure outcomes" |
| skill-documentation-cli-accuracy | "When writing or updating skill documentation files that teach CLI command usage to LLMs and users" |
| skill-registration-at-install-time | "Creating new skills for a Claude plugin that needs to be auto-discovered when the plugin is installed or updated" |
| skill-scope-cycle-focus | "When planning skill development and user explicitly constrains scope with statements like 'one skill for [specific thing], that's all I want'" |
| skills-based-interaction-model | "When documenting or designing user-facing interfaces for tools that have both Claude Code skills and CLI binaries, where end users interact exclusively through skills like /traced-init and /traced-audit" |
| skip-discovery-steps-enforcement | "When working on tasks explicitly marked as brainstorming, discovery, analysis, or interactive walk-through phases that require collaborative dialogue" |
| skip-node-modules-discovery | "When performing file discovery walks in JavaScript/Node.js projects that contain node_modules directories" |
| spec-field-accountability | "Writing specifications that contain schema examples, configuration samples, or data structure tables during system design or redesign" |
| spec-layer-commit-atomicity | "Working on projects using /traced specification process while making code changes that affect use cases, requirements, design, or architecture" |
| spec-review-checklist-before-implementation | "Working with a large, foundational specification document before beginning implementation of a complex system or major update" |
| spec-trace-mandatory-process | "When implementing features or making code changes that require the /traced specification process according to team directive" |
| specs-must-trace-with-code | "When making code changes that affect system design, architecture, or functionality that should be traced through spec layers (UC -> REQ -> DES -> ARCH -> TEST)" |
| stamp-before-verify | "When editing TOML spec files (*.toml) in the traced project" |
| standardize-retro-format-documentation | "When conducting retrospectives without established format guidelines or standards" |
| step-by-step-visual-preview-priority | "When user provides numbered step-by-step instructions requesting immediate visual output (like 'show me the 2D outline in an html preview')" |
| stop-hook-async-to-sync | "When implementing any Claude Code hook that needs the agent to see and act on its output" |
| stop-pausing-ask-confirmation-layers | "Working on multi-layer specification processes, implementation tasks, or other structured workflows where the user has given initial authorization to proceed" |
| structure-beats-open-ended-instructions | "When designing agent behaviors, tasks, or workflow automation where reliable execution is critical" |
| structure-enforces-process-discipline | "When working on tasks involving multiple files or complex processes where documented workflows exist (retros, skill documentation, process prompts)" |
| subagent-autonomous-execution-pattern | "When multi-step development work can be parallelized across multiple subagents (e.g., implementing different components, writing tests for different modules, or handling separate issues)" |
| subagent-concurrent-delegation | "Multiple independent tasks need completion and could run concurrently without blocking each other" |
| subagents-over-teamcreate-pattern | "When multiple independent tasks need to be executed in parallel to complete work efficiently" |
| suggest-architecture-changes-for-complexity | "When designing or reviewing architecture that involves multiple system layers or components" |
| targ-build-automation-preference | "When adding new development tasks, build targets, or automation to the project" |
| targ-build-tool-standardization | "When setting up build tooling, configuring project scripts, or making build-related decisions for the project" |
| targ-check-full-not-check | "When an executor is about to run quality checks, linting, or pre-commit verification in the engram project" |
| targ-test-execution-requirement | "When executing Go tests in this project" |
| targ-test-wrapper-convention | "When needing to execute the test suite in this project" |
| tdd-test-first-discipline | "When implementing new functionality or significantly changing existing code behavior (especially when changing function signatures or output formats)" |
| team-direct-communication-not-subagents | "When the user requests a 'team' for multi-agent collaborative work scenarios" |
| test-parallelization-requirement | "When writing Go test functions and subtests, particularly after receiving explicit team instructions about parallel test execution" |
| thin-wrapper-failure-accountability | "When encountering test coverage failures, lint issues, or other quality problems in thin wrapper or I/O methods during implementation" |
| tmux-session-cleanup-hook | "Working with tmux sessions that manage status data or allocate resources during operation" |
| tool-defaults-not-caller-responsibility | "When building tools that require configuration values with well-known defaults (data directories, API token sources, config file paths)" |
| triage-before-implementing | "When a bug has been filed or an issue identified and someone is ready to start implementing a fix" |
| triage-plan-execute-review-iterate | "User requests comprehensive handling of complex, multi-step technical work with explicit instructions to not defer any work and provide complete results" |
| two-pass-layer-review | "When conducting reviews of layered development work where each layer builds on the previous one (like design -> implementation -> testing)" |
| ultrathink-for-strategic-documents | "When working on strategic planning documents that require systematic analysis of complex problems, trade-offs, and multi-faceted solutions" |
| unified-targ-subcommand-interface | "When setting up project developer tooling or CLI command interfaces" |
| untyped-iota-gendecl-preservation | "When processing Go code containing untyped constant blocks that use iota for value generation" |
| validate-before-directing-users | "When providing users with URLs, file paths, or addresses to access resources or files" |
| validate-invariants-before-review | "When working with geometric data or mesh generation and preparing to request human review of the output" |
| validator-gate-spec-enforcement | "When reviewing code commits or work submissions that include implementation changes to a codebase" |
| validator-spec-trace-enforcement | "When validators are reviewing code changes or work submissions that require architectural documentation and traceability" |
| validator-spec-trace-gating-2 | "When reviewing executor work in a /traced specification process where the 5-layer spec model (UC, REQ/DES/ARCH, TEST, IMPL) is required" |
| validator-spec-trace-gating | "When validators are reviewing executor work or code changes that claim to implement requirements" |
| verification-scope-and-completeness | "When user explicitly requests a 'deep check' or 'deep scan' of code vs documentation or similar comprehensive verification tasks" |
| wait-for-task-assignment-before-starting | "When working on prioritized issue lists or task assignments in a multi-agent coordination environment" |
| worktree-merge-workflow | "When an agent working in a worktree is ready to push and merge changes to main" |
| worktree-uncommitted-check-before-remove | "When removing a git worktree that contains uncommitted changes" |
| write-it-down-means-create-file | "When user says 'write it down' or asks to document findings, analysis, or defect lists" |
| writing-skills-endpoint-for-skill-edits | "When planning or implementing changes to SKILL.md files (skill documentation in plugin repositories or agent skill systems)" |
| writing-skills-pressure-test-validation | "When defining or updating skills, workflows, or process documentation that will be used in real-world scenarios" |
| when-a-function-checks-ctx-err-then-returns-nil-on-that-error-path | "When a function checks ctx.Err() then returns nil on that error path" |
| when-deciding-when-to-call-learn-after-being-corrected | "When deciding when to call /learn after being corrected" |
| when-deciding-when-to-call-learn-after-changing-direction | "When deciding when to call /learn after changing direction" |
| when-deciding-when-to-call-learn-after-completing-a-plan-step | "When deciding when to call /learn after completing a plan step" |
| when-deciding-when-to-call-learn-after-resolving-a-bug | "When deciding when to call /learn after resolving a bug" |
| when-deciding-when-to-call-learn-at-end-of-session | "When deciding when to call /learn at end of session" |
| when-deciding-when-to-call-learn-during-a-session | "When deciding when to call /learn during a session" |
| when-deciding-when-to-call-prepare-before-code-review | "When deciding when to call /prepare before code review" |
| when-deciding-when-to-call-prepare-before-debugging | "When deciding when to call /prepare before debugging" |
| when-deciding-when-to-call-prepare-before-tackling-an-issue | "When deciding when to call /prepare before tackling an issue" |
| when-deciding-when-to-call-prepare-when-resuming-work | "When deciding when to call /prepare when resuming work" |
| when-deciding-when-to-call-prepare-when-switching-tasks | "When deciding when to call /prepare when switching tasks" |
| when-deciding-when-to-call-prepare | "When deciding when to call /prepare" |
| when-dispatching-implementer-subagents-that-commit-their-work | "When dispatching implementer subagents that commit their work" |
| when-encountering-lint-coverage-failures-after-refactoring | "When encountering lint/coverage failures after refactoring" |
| when-implementing-claude-code-hooks-for-a-new-event-type | "When implementing Claude Code hooks for a new event type" |
| when-refactoring-a-go-binary-s-entry-point-and-the-binary-is-installed-in-multiple-locations | "When refactoring a Go binary's entry point and the binary is installed in multiple locations" |
| when-subagents-trigger-learn-during-implementation | "When subagents trigger /learn during implementation" |
| when-writing-a-skill-that-needs-agents-to-perform-a-specific-procedure-defined-in-another-skill | "When writing a skill that needs agents to perform a specific procedure defined in another skill" |
| when-writing-llm-system-prompts-that-describe-output-format-expectations | "When writing LLM system prompts that describe output format expectations" |
| worktree-creation-gitlock-delay | "When dispatching multiple agents that each need to create git worktrees for parallel execution" |
| engram-chat-file-location | "When setting up or referencing engram chat file paths for multi-agent coordination" |
| engram-chat-new-message-types | "When working with engram chat protocol message types" |
| engram-plugin-auto-discovery | "When adding new skills to the engram Claude Code plugin" |
| engram-up-backward-compat | "When users invoke /engram or /engram-up to start an engram session" |
| engram-xdg-data-dir-permission-fix | "When debugging spawned agent permission prompts or understanding why engram uses ~/.local/share/engram/" |
| engram-binary-install-lifecycle | "When engram commands fail unexpectedly, return unknown flags, or behave like an older version -- or when verifying the binary is current after code changes" |
| engram-binary-worktree-slug-bug | "When running engram binary commands (chat cursor, chat post, chat watch) from a git worktree directory" |
| engram-ack-wait-flow | "When an engram agent needs to wait for ACK before proceeding with an intent" |
| engram-headless-worker-loop-exit-contract | "When a headless worker (engram agent run) needs to end its turn or exit its loop" |
| engram-tmux-lead-lifecycle-intent-required | "When engram-tmux-lead spawns, kills, respawns, shuts down, or makes routing decisions about agents" |
| engram-tmux-lead-no-foreground-blocking | "When implementing monitoring, polling, or waiting patterns in engram-tmux-lead skill" |
| engram-speech-act-prefix-markers | "When engram binary scans agent speech for coordination protocol markers to post to chat" |
| engram-lead-hold-based-agent-lifecycle | "When working with engram-tmux-lead agent lifecycle groups or designing agent coordination patterns" |
| engram-esc-interrupt-stuck-agent | "When an agent gets stuck in inference and is unresponsive in its tmux pane" |
| toejough-issue-repo-purpose | "When filing issues or bugs in repositories and deciding which repo is appropriate for cross-project or workflow process issues" |
| check-uncommitted-must-pass | "When an executor reports task completion after running targ check-full" |
| agent-review-before-presenting | "When any agent has completed work (plan, triage, implementation, research) and is about to surface results to the user" |
| when-deploying-the-engram-binary-for-cli-use | "When deploying the engram binary for CLI use" |
| when-implementing-or-reviewing-the-engram-recall-query-pipeline | "When implementing or reviewing the engram recall --query pipeline" |
| when-running-filtered-tests-during-engram-development | "When running filtered tests during engram development" |
| when-running-targ-subprocess-tests-with-environment-variables-set-in-the-test-sandbox | "When running targ subprocess tests with environment variables set in the test sandbox" |
| when-working-with-targ-in-engram-project | "When working with targ in engram project" |
| when-writing-a-session-start-hook-that-creates-a-symlink-to-a-managed-binary | "When writing a session-start hook that creates a symlink to a managed binary" |
| testing-that-concurrent-operations-respect-context-cancellation | "Testing that concurrent operations respect context cancellation" |
| use-engram-chat-online-detection | "When an engram-chat agent needs to determine if another agent is currently online" |
| engram-down-list-panes | "When the engram-down skill needs to find and kill tmux panes for shutdown" |
| engram-pane-display-filter | "When engram binary filters claude headless agent output for pane display" |
| engram-state-reconstruction-strategy | "When reconstructing agent state from chat file in engram" |
| background-bash-cursor | "When engram-tmux-lead spawns background bash tasks that need to read new chat messages from a cursor position" |
| engram-tmux-tail-pane-id | "When the engram-tmux-lead skill spawns the tail pane and needs to kill it cleanly on shutdown" |
| engram-tmux-drain-before-spawn | "When the engram-tmux-lead skill spawns background fswatch tasks for chat file monitoring" |
| parallel-cycles-with-worktrees | "When executing multiple development cycles that could be performed independently or in parallel" |
| parallelize-subagents-worktree-isolation | "When multiple implementation subagents need to run on independent tasks that modify different files but share git state" |
| yagni-simplicity-in-factories | "When designing a factory function for SM-2 algorithm state objects that need to return initial values" |
| rotten-tomatoes-minimum-score-80 | "When providing movie recommendations or evaluating movie options for selection" |
| sticky-cascading-headers-scroll-behavior | "Working on blog layout with multiple header levels (site banner, blog post title, section headers) that need coordinated scroll-triggered positioning behavior" |
| slider-sensitivity-graph-workflow | "When building parameter tuning or optimization interfaces where users need to explore sensitivity across parameter ranges" |
| parallel-block-generation-viewer-tool | "When generating multiple blocks or content units that can be processed independently and require comparison or review" |
| continue-after-tool-failure | "when a tool completes execution" |
| use-sonnet-haiku-agents | "When dispatching agents for multi-step implementation work or task continuation" |
| haiku-validators-with-sonnet-executors | "when setting up parallel team workflows with validator and executor roles" |
| parallel-agent-execution-with-validation | "When working on complex multi-step workflows involving file updates, research tasks, and coordination between multiple activities" |
| parallel-bug-fix-with-model-roles | "Multiple related bugs in the same codebase or skill cluster need to be fixed" |
| task-planning-validation-workflow | "Multiple related bugs need fixing across shared files where sequential work would create bottlenecks and concurrent editing would cause merge conflicts" |
| tier-c-filtering-pipeline-architecture | "When implementing memory persistence in a learning pipeline that classifies memories by quality tier" |
| three-layer-consolidation-defense | "When implementing memory consolidation systems or designing memory retention strategies" |
| policy-toml-confidence-threshold-default | "When implementing clustering or similarity-based features that use confidence thresholds to filter results before presenting them to users" |
| claude-code-settings-statusline-override | "When spawning Claude Code agents and wanting to suppress or override the status line" |
| reuse-shared-graph-methods | "When implementing or refactoring code that involves graph traversal, coverage detection, or node walking in a codebase that already has shared utility methods for these operations" |
| targ-binary-shadowing | "Working on a project that needs build automation and the system has targ installed for build target management" |
| targ-command-dispatcher-unification | "When encountering a project with multiple development scripts scattered across directories (dev/build, dev/check, dev/test, etc.)" |
| transcript-always-exists | "When working with transcript data, references, or lookups in system operations" |

## Rewrite

| Memory | Current Situation | Proposed Situation |
|---|---|---|
| agent-role-based-naming | "When engram-tmux-lead spawns agents that have specific roles or perspectives (e.g., architecture lead, Go binary specialist, skill coverage reviewer, phasing planner)" | "When spawning named agents in a multi-agent system" |
| announce-presence-before-spinup-work | "When an agent (any role: executor, reviewer, researcher, planner, engram-agent) initializes and needs to load resources, read history, or catch up before being useful" | "When an agent starts up and begins loading context" |
| async-hook-file-output | "When discussing async hooks that can't inject context due to blocking issues" | "When implementing Claude Code hooks that need to pass data between invocations" |
| autonomous-execution-self-review-loop | "when output volume spikes dramatically after feature completion or algorithm optimization" | "When reviewing system output after deploying a change" |
| check-uncommitted-not-expected | "When an agent runs targ check-full and sees the check-uncommitted check fail" | "When targ check-full fails on the check-uncommitted step" |
| cli-abstraction-for-memory-edits | "Working with engram project memory modifications" | "When modifying engram memory files" |
| cli-abstraction-memory-edits | "During memory triage sessions when engram hook surfaces memories for feedback" | "When engram hooks surface memories during active work" |
| completion-verification-with-deferral-tracking | "When completing complex development work that includes deferred items or follow-up tasks" | "When wrapping up development work that has follow-up items" |
| continuous-file-watching-multi-agent-chat | "When communicating with other agents via chat.toml file in a multi-agent coordination system" | "When implementing file-based multi-agent communication" |
| coordinator-requires-assignment-approval | "When acting as coordinator in a multi-agent system with pending tasks available for assignment" | "When assigning tasks to agents as a coordinator" |
| default-universal-scope-carefully | "When creating or categorizing memories during agent interactions, particularly when deciding whether a lesson applies universally or only to the current project context." | "When creating memories and choosing their scope" |
| delete-bad-memories-dont-refine | "When dealing with incorrectly refined memories or memories with empty/missing action fields in the engram system" | "When maintaining engram memory quality" |
| engram-agent-must-always-ack-intents | "When engram-agent receives an intent message addressed to it (or to all) and has no matching feedback or fact memories" | "When engram-agent receives an intent message" |
| engram-cli-memory-modification-commands | "when modifying memory files or working with engram memory systems" | "When modifying engram memory files" |
| engram-commands-memory-workflow-issue | "when processing memories or working with memory management systems" | "When modifying engram memory files" |
| engram-field-pipeline-mapping | "When designing or refactoring a memory classification system with fields that map to different pipeline lifecycle stages" | "When designing memory classification fields" |
| engram-tmux-lead-multi-column-layout | "When engram-tmux-lead is spawning more than 4 right-side panes (agents) in a session" | "When spawning many agents in engram-tmux-lead" |
| executor-must-acknowledge-engram-agent-waits | "When an executor agent receives a WAIT from engram-agent during task execution in a multi-agent session" | "When an executor receives a WAIT message from engram-agent" |
| existing-agent-coordination-integration | "When user mentions setting up team coordination and existing agents/coordinators are already in place via file-based communication system" | "When setting up multi-agent coordination" |
| extraction-prompt-memory-handling-guidance | "when writing prompts for memory extraction or refinement operations" | "When writing LLM prompts for memory extraction" |
| file-comms-external-not-subagents | "When user instructs to use file-comms coordination with externally-launched executors in separate terminals" | "When setting up multi-agent coordination with external agents" |
| flag-upstream-complexity | "when implementing requirements or design specifications that involve complex multi-phase processes, file restructuring, or extensive validation logic" | "When implementing complex requirements or specifications" |
| honor-wait-messages-in-chat-protocol | "When receiving a WAIT message from another agent in the engram chat protocol after posting an intent" | "When receiving a WAIT message in engram chat protocol" |
| instruct-agents-use-relevant-skills | "When lead dispatches an executor agent for any task involving engram workflows or project-specific processes" | "When dispatching executor agents for project-specific work" |
| intent-must-always-include-engram-agent | "When an active agent constructs an intent message to broadcast before taking a significant action" | "When constructing intent messages in engram chat" |
| intent-requires-explicit-ack-from-all-recipients | "When an active agent posts an intent and waits for the 500ms window to elapse without receiving responses" | "When posting intents in engram chat protocol" |
| mandatory-full-build-before-merge | "During subagent-driven development, when compilation errors appear but are suspected to be stale LSP diagnostics" | "When merging subagent work" |
| memory-2 | "Memory extraction/conversion process failing with JSON parsing errors" | "When running memory extraction or conversion pipelines" |
| memory | "Systematic elimination of silent failures across codebase" | "When fixing error handling across a codebase" |
| memory-extraction-relevance-and-decay | "When working on memory system improvements and discussing how to reduce extraction noise and improve relevance of stored memories" | "When improving memory extraction quality" |
| memory-feedback-completion-pattern | "when engram memories are surfaced in system reminders during conversations" | "When engram surfaces memories during a conversation" |
| memory-3 | "When implementing signal handling for CLI force-exit alongside targ.Main() which uses signal.NotifyContext internally" | "When implementing signal handling in a Go CLI" |
| parallelizable-task-decomposition-completion-workflow | "When work appears finished or reaches a natural stopping point, but additional tasks or improvements could still be completed" | "When reaching a stopping point in multi-task work" |
| parameter-read-write-lifecycle | "When designing or auditing a system with configurable parameters during architecture review or implementation" | "When designing systems with configurable parameters" |
| passive-accumulation-human-review-keywords | "When designing keyword refinement systems for memory feedback loops, where irrelevant surfacing feedback provides evidence for which keywords caused false positives" | "When designing keyword systems for memory retrieval" |
| planning-versus-execution-signal | "User says 'let's start/jump into [cycle]' when planning artifacts (premortems, specs, gates) already exist and are complete" | "When user signals readiness to begin execution" |
| playground-first-validation-later | "When tuning memory system weights, formulas, or exploring new ranking approaches during the exploration phase of development" | "When tuning system parameters or exploring ranking approaches" |
| playground-production-parity-spreading-activation | "When implementing or modifying ranking algorithms that will be used in production systems" | "When implementing ranking algorithms" |
| populate-surfaced-count-always | "Building TrackingData maps for memory classification when stored memories contain surfacing history that needs to be preserved" | "When building classification data for engram memories" |
| pragmatic-skill-integration-replacement | "When deciding between using existing tools, skills, or plugins versus building custom solutions for a project" | "When choosing between existing tools and building custom solutions" |
| pragmatic-skill-integration-strategy | "Working with existing skills or tool integrations during time-boxed development cycles (like 5-minute increments)" | "When integrating existing skills under time constraints" |
| pre-push-working-directory-validation | "When claiming that changes have been committed and pushed successfully, or when asked to ensure repository cleanliness" | "When pushing changes to a remote repository" |
| prefixed-spec-ids-for-parallel-agents | "When multiple executor agents are creating traced specification layers concurrently for different features in the same repository worktree" | "When multiple agents create traced specs concurrently" |
| proactive-correction-timing-surface-stage | "When an LLM agent has access to memory systems that could surface relevant historical situations and corrective guidance during task execution" | "When designing memory surfacing timing" |
| proactive-execution-agreed-work-plan | "When working through a prioritized list of tasks or issues in a collaborative system where the work plan has been established and agreed upon" | "When executing an agreed-upon work plan" |
| proactive-memory-graduation-prompt | "When starting a new session and there are memories that may need graduation, update, or curation" | "When starting a new session" |
| proceed-after-uc-alignment | "When user and assistant have just finished discussing and aligning on requirements, use cases, or technical standards for a development task" | "When requirements alignment is complete and implementation is next" |
| progressive-disclosure-nested-headings-strategy | "When processing markdown documents for coverage analysis, working with multi-level heading hierarchies (H1/H2/H3) where document splitting granularity needs to be determined" | "When processing markdown documents with multi-level headings" |
| project-context-generalizability-scoring | "When working across multiple projects and engram surfaces memories from previous sessions" | "When engram surfaces memories from other projects" |
| prompt-consolidation-strategy | "When extensive research and analysis has been done on a complex domain and implementation work is about to begin" | "When transitioning from research to implementation" |
| proposal-with-issue-recommendations | "When you have identified multiple issues, problems, or findings that need stakeholder review and decision-making" | "When presenting multiple findings for user review" |
| recall-skill-context-accumulation-design | "When implementing session context recall or resume functionality for retrieving previous conversation history" | "When implementing session recall functionality" |
| refine-prompt-rewrite-sbia-not-extract | "When working on the engram refine command functionality that should improve existing memory SBIA field clarity" | "When implementing memory refinement" |
| remove-execution-model-architecture | "When designing system architecture for agent execution workflows and workflow coordination mechanisms" | "When designing agent execution architecture" |
| remove-frequently-useful-memories | "When the system recommends promoting frequently-surfaced, high-effectiveness memories to skills or other tiers" | "When reviewing memory promotion candidates" |
| remove-hardcoded-execution-rules-prompts | "When writing orchestration prompts or system prompts that coordinate multi-step agent workflows and tool usage" | "When writing orchestration prompts for agent workflows" |
| remove-leech-memories-system-cleanup | "When reviewing memory system performance or conducting periodic maintenance on accumulated memories" | "When maintaining memory system quality" |
| remove-traces-comment-validation | "When auditing or validating traced specification compliance in codebases" | "When auditing traced specification compliance" |
| remove-unused-ranking-factors | "When empirical optimization testing reveals that certain ranking factors (alpha spreading, BM25 floor, recency weighting, tier C memories) have zero impact on retrieval quality across multiple projects and scoring scenarios" | "When evaluating ranking factor effectiveness in retrieval systems" |
| require-phase-status-reporting | "Orchestrator managing multi-phase tasks where executors need to complete work in sequence before next steps can proceed" | "When orchestrating sequential multi-phase work" |
| research-reference-system-mechanisms-before-design | "When researching or discussing another system's architecture and mechanisms to inform design decisions" | "When researching existing systems to inform design" |
| restore-dirty-unsatisfiable-flags | "When working with group TOML files in a traced specification system that needs to track modification and constraint satisfaction state" | "When designing state tracking in specification files" |
| resume-state-in-spec-files | "When implementing a system that needs to track completion and resume state across multiple spec files and groups" | "When designing completion tracking across spec files" |
| retain-session-context-logs | "When user indicates that a topic or question has already been discussed or covered earlier in the current session" | "When user references earlier discussion in the same session" |
| retire-all-duplicate-instructions | "When performing deduplication or cleanup tasks on instruction/memory systems" | "When deduplicating instructions or memories" |
| retro-closure-completeness | "When wrapping up a retrospective session and preparing to archive the retro document" | "When completing a retrospective session" |
| retro-first-workflow-priority | "At the completion of a development cycle or major work phase" | "When a development cycle completes" |
| rolling-log-turn-summarization-archive | "When implementing memory management for AI sessions that need to retain context across multiple interactions while avoiding unbounded growth" | "When implementing session memory management" |
| sbia-framework-memory-extraction-adoption | "When extracting memories or corrections from session interactions where problematic behavior occurred" | "When extracting memories from session transcripts" |
| session-logging-directory-based-structure | "When implementing session logging functionality for tools that operate across multiple session directories or project contexts" | "When implementing session logging" |
| single-classification-source-of-truth | "When building systems that need to classify memories for maintenance or signal detection, where multiple components could independently invoke classification logic" | "When building memory classification systems" |
| skill-file-update-test-pressure | "When discovering a behavioral issue, inefficiency, or correctable pattern during audit, development, or debugging work" | "When fixing behavioral issues in skills" |
| sonnet-correction-extraction-sbia-context | "When designing a learning system that extracts corrections and lessons from conversation transcripts to build searchable memories" | "When designing memory extraction from transcripts" |
| sonnet-haiku-extraction-surfacing-split | "When working with SBIA memory systems that involve both extraction (building memories from conversation transcripts) and retrieval (surfacing relevant memories during active sessions)" | "When designing memory extraction and retrieval pipelines" |
| spec-driven-architecture-prompt-consolidation | "When planning to rebuild, evolve, or significantly modify a system after accumulating substantial experience with what works and what doesn't" | "When planning a major system rebuild or evolution" |
| spec-id-prefixing-parallel-agents-3 | "When multiple agents are working in parallel worktrees creating new REQ, DES, ARCH, or TEST entries in docs/specs/" | "When multiple agents create traced spec entries concurrently" |
| spec-id-prefixing-parallel-agents | "Multiple agents working in parallel on specification entries in docs/specs/ during development tasks" | "When multiple agents create traced spec entries concurrently" |
| spec-traceability-entry-point-classification | "Building traceability systems to verify that all implemented CLI commands have corresponding specifications and all spec'd commands are implemented" | "When building traceability between specs and implementation" |
| spreading-activation-playground-visibility | "When implementing or modifying algorithms that affect memory ranking in production systems" | "When modifying memory ranking algorithms" |
| state-toml-migration-cleanup | "When updating specs or documentation that contains references to the old state.toml workflow (centralized state tracking)" | "When updating traced project documentation" |
| strip-system-reminders-from-extraction | "When extracting conversation text from session transcripts for summarization or recall systems" | "When processing session transcripts" |
| toml-artifact-format-preference | "When migrating spec artifacts from markdown format to a structured serialization format" | "When choosing serialization format for spec artifacts" |
| toml-group-hierarchy-with-merkle-hashing | "When designing or implementing group configuration files for the traced project's specification management system" | "When implementing traced group configuration files" |
| toml-pending-evaluation-surfacing-storage | "When implementing memory surfacing functionality that needs to track in-flight evaluation state alongside existing memory metadata" | "When implementing memory surfacing state tracking" |
| traced-tdd-spec-driven-work | "When implementing features or making changes in a codebase that follows the /traced specification process with layered specs (L1 use cases -> L2 requirements -> L3 architecture -> L4 tests -> L5 implementation)" | "When working in a traced codebase" |
| use-targets-for-state-tracking | "When implementing multi-step workflows that require state tracking and verification (e.g., issue closing processes with summary updates, commits, and cleanup steps)" | "When implementing multi-step workflows with state tracking" |
| use-teams-feature-for-agent-communication | "When designing multi-agent coordination systems with existing notification mechanisms and message addressing already in place" | "When designing multi-agent communication" |
| watch-loops-block-main-agent-interaction | "When designing multi-agent coordination systems where agents need memory but the main agent should remain interactive" | "When designing multi-agent systems with a main interactive agent" |
| worktree-naming-strategy | "When implementing auto-detection of worktree context for spec ID prefixing, there are multiple ways to derive the prefix from the worktree" | "When deriving identifiers from git worktrees" |
| engram-agent-list-missing-state-file | "When engram agent list encounters a missing state file" | "When implementing engram agent list" |
| engram-skill-rename-file-comms | "When referencing the engram chat coordination skill or file-comms" | "When working with engram coordination skills" |
| engram-coverage-out-stale | "When running targ check-full after rebasing or reordering Go files in engram" | "When running targ check-full after rebasing" |
| engram-ack-wait-offline-fd-exhaustion | "When writing Go tests for engram agent spawn that involve AckWait with engram-agent offline" | "When testing engram agent spawn with AckWait" |
| engram-internal-chat-package | "When implementing or reviewing the engram internal/chat package" | "When working with engram internal/chat package" |
| engram-internal-watch-package | "When implementing or reviewing engram file-watch functionality" | "When working with engram file-watch code" |
| engram-watcher-suffix-fix | "When reviewing engram chat watcher behavior or debugging Watch returning false on large chat files" | "When debugging engram chat watcher issues" |
| engram-worktree-chat-path-derivation | "When use-engram-chat-as skill derives chat file path in a git worktree" | "When using engram chat from a git worktree" |
| when-refactoring-engram-cli-code-and-encountering-lint-failures | "When refactoring engram CLI code and encountering lint failures" | "When refactoring engram CLI code" |
| adding-multiple-constants-to-an-existing-const-block-in-engram | "Adding multiple constants to an existing const block in engram" | "When adding constants to Go const blocks" |
| engram-keywords-blob-memories | "When matching intents against engram feedback memories using situation fields" | "When working with engram memory situation matching" |
| zombie-tasks-cause | "When diagnosing why the engram lead spawns duplicate ACK messages in the chat file" | "When debugging duplicate messages in engram chat" |
| when-ide-diagnostics-report-errors-after-subagent-file-changes | "When IDE diagnostics report errors after subagent file changes" | "When IDE reports errors after subagent file changes" |
| when-fixing-context-cancellation-in-concurrent-code | "When fixing context cancellation in concurrent code" | "When writing concurrent Go code with context" |
| when-designing-mode-b-recall-to-search-across-many-session-files | "When designing mode B recall to search across many session files" | "When implementing cross-session search" |
| traced-audit-endpoint-requirement | "After fixing critical spec integrity issues (192 stale hashes, orphaned ARCH items)" | "When completing fixes to spec integrity issues" |
| resurrect-skill-new-traced-repo | "When choosing tools for README generation in a traced development environment with spec representation requirements" | "When generating READMEs in a traced project" |

## Delete (situational)

| Memory | Current Situation | Reason |
|---|---|---|
| convert-refine-pipeline-order | "When providing feedback on memory migration pipeline ordering that has already been documented" | one-time event, action says "no new action needed" |
| engram-main-post-merge-2026-04-04 | "After merging the bug fix branches from the 2026-04-04 session" | diary entry with specific commit/date |
| engram-open-issues-2026-04-04 | "When planning work on engram open issues or checking issue priority/dependencies" | date-stamped triage snapshot, stale |
| engram-phase1-plan-doc | "When implementing engram Phase 1 (chat post+watch) or reviewing the implementation plan" | phase-locked, Phase 1 complete |
| engram-phase2-reviewer-findings | "When reviewing Phase 2 (ACK-wait + Holds) implementation quality or planning Phase 2.1 patches" | phase-locked, Phase 2 complete |
| engram-phase2-validation-pass | "When checking Phase 2 (ACK-wait + Holds) implementation and validation status" | phase-locked status check |
| engram-phase21-bug-issues | "When planning Phase 2.1 patches or checking status of Phase 2 bug fixes" | phase-locked, all issues closed |
| engram-phase21-issues-closed | "When checking status of engram Phase 2.1 bug issues" | phase-locked, all issues closed |
| engram-phase3-e2e-verification | "When checking Phase 3 E2E verification status or planning work before Phase 3 is done" | phase-locked, Phase 3 complete |
| engram-phase4-complete | "When checking engram Phase 4 Speech-to-Chat implementation status" | phase-locked status check |
| engram-phase4-cursor-capture-pattern | "When implementing engram Phase 4 Task 4 (engram agent run + ack-wait+resume loop in runAgentRun)" | phase-locked to Phase 4 Task 4 |
| engram-phase4-plan-doc | "When looking for the engram Phase 4 speech-to-chat implementation plan document" | phase-locked document pointer |
| engram-phase4-skill-review | "When checking Phase 4 skill change quality or review status" | phase-locked review status |
| engram-phase4-type-conventions | "When implementing engram Phase 4 binary Go code" | phase-locked conventions |
| engram-phase4-user-e2e-review-gaps | "When planning Phase 5 or reviewing Phase 4 User E2E acceptance" | phase-locked gap list |
| engram-phase5-agent-model | "When designing or implementing engram-agent behavior in Phase 5 and beyond" | phase-locked design decision |
| engram-phase5-design-decisions | "When implementing Phase 5 (Agent Resume + Auto-Resume) or designing worker state transitions" | phase-locked design decisions |
| engram-phase5-plan-doc | "When looking for the Phase 5 (agent auto-resume) implementation plan document" | phase-locked document pointer |
| engram-phase5-reassessment-outcome | "When planning Phase 5 (Agent Resume + Auto-Resume) or reviewing post-Phase-4 decisions" | phase-locked reassessment |
| engram-phase5-wait-semantics | "When engram-agent posts WAIT: in Phase 5 during argument protocol" | phase-locked semantics |
| engram-pre-phase5-skill-blockers | "When starting Phase 5 work or checking pre-Phase-5 skill blocker status" | phase-locked, resolved |
| engram-codesign2-phase-plan | "When planning engram binary implementation phases or checking which issues belong to which phase" | phase-locked plan, all phases complete or superseded |
| engram-spec-phasing-gaps | "When estimating effort for engram deterministic coordination phases or reviewing the phase plan" | phase-locked estimate review |
| engram-spec-review-critical-gaps | "When implementing or reviewing the engram deterministic coordination design spec" | snapshot of gaps, resolved |
| engram-spec-binary-alignment-gaps | "When implementing or reviewing engram binary Go code against the deterministic coordination design spec" | snapshot of gaps, resolved |
| engram-spec-skill-coverage-fixes | "When reading or updating engram deterministic coordination design spec skill coverage section" | snapshot of fixes, applied |
| engram-issue-519-filed | "When checking open engram bugs or ack-wait reliability" | issue closed |
| engram-issue-519-fixed | "When checking fix status of engram issue #519 (ack-wait timeout)" | issue closed, fix applied |
| engram-issue-522-pane-border-status | "When checking pane title visibility in engram-tmux-lead sessions or reviewing S1.3 of engram-tmux-lead SKILL.md" | issue closed |
| engram-issue-523-monitor-prompt-template | "When reviewing engram-tmux-lead skill monitor agent prompt sections or checking why background monitors use pseudocode instead of real commands" | issue closed, fix applied |
| engram-issue-524-verbatim-callout | "When using the Background Monitor Pattern in use-engram-chat-as or checking why agents improvise polling instead of using engram chat watch" | issue closed |
| engram-issue-530-fixed | "When checking if engram-tmux-lead spawn intent situations have been fixed to use semantic context" | issue closed |
| engram-issues-514-510-closed | "When checking status of engram bug issues #514 and #510" | issues closed |
| engram-483-hold-model-implemented | "When checking implementation status of engram-tmux-lead hold-based agent lifecycle (#483)" | issue closed, implemented |
| engram-phase1-skill-analysis | "When implementing Phase 1 of engram binary or updating use-engram-chat-as and engram-tmux-lead skills" | phase-locked |
| engram-chat-watch-race-issue514 | "When reviewing engram chat watch race condition bug or implementing Background Monitor Pattern" | issue closed |
| engram-agent-list-reconstruction-phase3 | "When implementing engram agent list command in Phase 3" | phase-locked |
| engram-ackwaiter-deadline-fix-pattern | "When fixing or reviewing engram AckWait timeout behavior or the FileAckWaiter implementation" | specific bug fix, applied |
| plus-delta-other-retrospective-framework | "After completing major development tasks, iterations, or project milestones when conducting project retrospectives" | retro-format-convention covers this better |
| engram-agent-ack-latency-warning | "When engram-agent is running via the background monitor pattern (subagent spawn per iteration) and active agents use engram chat ack-wait with --max-wait 30" | implementation-specific to deprecated pattern |
| spawn-intent-situation-quality | (corrupted file - contains only "$updated_content") | corrupted, no content |
| separate-refine-operations-tracking | "When there are multiple memory refine operations that need to happen - normal refining of unrefined memories and re-refining of memories that were previously refined incorrectly" | refine command removed from engram |
| preserve-branch-update-status | "User explicitly requests to keep a development branch unchanged while asking for status log updates" | one-time user preference event |
| stop-feedback-directive | "When user sends 'Stop hook feedback:' message during a session to halt automated system feedback collection" | one-time event, hook-specific |
| session-start-signal-activation | "When signals are surfaced at session start through the signal-surface hook with instructional text for the model" | signal system removed |
| sessionstart-hook-active-memory-review | "When session-start hook surfaces memory management signals (noise candidates, hidden gems, etc.) to provide context for user decisions" | signal system removed |
| sessionstart-hook-memory-surfacing | "When designing or modifying session startup/initialization behavior, particularly the SessionStart hook" | signal system removed |
| sessionstart-remove-surfacing-maintain-only | "When implementing or modifying SessionStart functionality in the engram memory system" | signal system removed |
| signal-surface-user-approval-framing | "When the formatInstructions function generates guidance text for surfaced signals from the signal queue during session start" | signal system removed |
| engram-pane-title-after-ready-wait | "When pane titles are set in engram-tmux-lead during agent spawn" | specific bug fix (#526), applied |
| engram-proceed-normal-p-turn | "When engram binary sends a user turn to a headless agent spawned with claude -p" | implementation detail, captured in code |
| prompt-evaluation-triggers-track-record-not-memory | "When evaluating triggers for re-evaluating prompts in the engram system (specifically the surfacing prompt and other configurable prompts)" | obsolete, surfacing prompt removed |
| prompt-planning-dated-artifacts | "When working with prompts that contain planning or workflow guidance that could be converted into structured planning documents" | one-time conversion advice |
| proposal-limit-configurable-flag | "When implementing proposal limits or similar system constraints in engram or similar tools" | specific to removed feature |
| session-start-hook-analyze-command | "When session-start hooks need to gather system metrics and state for agent analysis" | implementation-specific to deprecated hook pattern |
| session-start-must-not-auto-load-memories | "When implementing session initialization hooks or memory loading features during Claude session startup" | specific to deprecated auto-load pattern |
| scanner-heading-level-initialization | "When scanning documents to build the node tree for spec analysis" | implementation detail for traced scanner |
| engram-deterministic-coordination-spec | "When looking for the engram deterministic coordination design specification document" | document pointer, file can be found by searching |
| recall-skill-stale-binary-path | "When referencing or updating the engram recall skill binary path" | stale path, self-describes as potentially obsolete |
| engram-2026-04-01-bad-refinement | "When reviewing or relying on engram memory content quality" | date-stamped event, refinement system removed |
| traced-artifacts-cleanup-protocol | "When re-running an adoption process after traced has fixed upstream problems" | one-time event |
| traced-autonomous-building-decisions | "User has explicitly instructed to build autonomously during implementation work, providing guidance to escalate only for use case or standard decisions" | one-time session directive |
| orphaned-commit-integration | "When developers work in detached HEAD state or on temporary branches and complete their work without properly integrating to the main development branch" | one-time event |
| stop-hook-memory-surface-agent-statements | "When an agent makes statements that should trigger contextual memories (e.g., claiming something is 'pre-existing' or making assertions that contradict established patterns)" | specific to deprecated hook pattern |
| use-ultrathink-for-next-task | "User explicitly requests ultrathink reasoning mode for the next task, indicating previous positive experience with extended reasoning" | one-time user preference |
| use-worktree-name-identifier | "When working with git worktrees and needing identifiers for prefixes, namespacing, or other operations" | redundant with worktree-naming-strategy |
| startup-messaging-user-control-simplification | "System startup when memory management data (noise candidates, hidden gems, skill promotions) needs to be communicated to the user" | signal/startup system removed |

## Delete (content)

| Memory | Current Situation | Reason |
|---|---|---|
| avoid-implementing-blocking-processes-during-design | "When implementing or testing multi-agent coordination features during design sessions" | too vague to be actionable |
| engram-dropped-go-commands | "When invoking or referencing engram CLI commands for maintain, surface, correct, or evaluate operations" | obsolete, commands no longer exist |
| engram-speech-to-chat-design | "When designing engram binary commands and skill separation for agent speech interception" | design complete, captured in code |
| engram-streamjson-package | "When designing or implementing engram binary parsing of claude headless agent output" | implementation complete, captured in code |
| engram-internal-chat-fileposter | "When implementing or reviewing engram chat message posting" | implementation detail, captured in code |
| engram-internal-chat-filewatcher | "When implementing or reviewing engram chat message watching" | implementation detail, captured in code |
| engram-cli-chat-wiring | "When reviewing or extending engram CLI chat command wiring" | implementation detail with commit ref |
| medial-axis-three-hard-constraints | "Working on medial-axis line drawing system and encountering disconnected boundary dots, then asking about previously established algorithmic constraints" | narrow to one project's geometry system |
| junction-sector-line-coverage | "Working on geometric or graph problems with junctions, sectors, and line routing where multiple constraints must be satisfied simultaneously" | narrow to one project's geometry system |
| s-curve-boundary-constraint | "During s-curve generation for groove patterns in star-shaped designs with defined boundary lines" | narrow to one project's geometry system |
| separate-flat-block-outputs | "When generating 3MF output files for geometric blocks that have both flat and grooved variants" | narrow to one project's 3MF output |
| smallest-cluster-consolidation-priority | "During memory consolidation when a memory qualifies for assignment to multiple overlapping clusters of different sizes" | clustering system removed |
| tooluse-to-mode-flags-refactor | "When analyzing tool hook architecture and discussing patterns for tool invocation in the engram system" | obsolete architecture discussion |
| traced-doc-absorb-non-living-docs | "During documentation review and maintenance activities where non-living or obsolete documents need to be handled" | redundant with general doc cleanup principles |
| traced-layer-momentum-rule | "When working through a systematic traced process or iterative layer-by-layer analysis" | too vague to be actionable |
| traced-project-specification-requirement | "When assigned to work on a traced project that has an associated specification document" | redundant with spec-trace-mandatory-process |
| traced-project-tool-usage | "When starting work on a project that contains docs/specs/state.toml indicating it follows the specwalk specification process" | state.toml removed, obsolete |
| traced-spec-driven-development | "When beginning work on a new feature or fix that requires code implementation" | redundant with traced-specification-workflow-mandatory |
| traced-specification-workflow-mandatory | "When beginning development work on a feature or fix, or when about to write implementation code" | redundant with spec-trace-mandatory-process |
| traced-standard-clarity-ticket | "When working with traced tool and encountering unclear or ambiguous standards documentation" | one-time ticket filing |
| traced-spec-migration-skill-definition | "When migrating ephemeral specification documents to permanent canonical documentation using writing-skills" | one-time migration |
| traced-test-tool-for-test-derivation | "When working on test derivation or converting issues to T-items" | too narrow, covered by traced workflow |
| track-violation-duplicate-detection | "When configuring monitoring systems to track recurring violations of instructions or rules" | too narrow, monitoring system removed |
| standards-ucs-l1-placement-rule | "When conducting audits and encountering standards use cases (UC-21-27) placed at L3 or other non-L1 levels, or when standards UCs lack clear identification markings" | narrow to specific traced UC numbers |
| standards-ucs-valid-l1 | "When reviewing or organizing use cases that have internal actors (architect, team lead, designer) rather than external users" | narrow to specific traced UC classification |
| visual-diagram-layer-traceability-validation | "When designing traced project architecture, specifications, or validation systems that need to bridge human oversight with LLM-generated work at scale" | too abstract, no actionable content |
| test-specs-group-file-consolidation | "When organizing test specifications and project documentation with multiple file types (UC, ARCH, and group files)" | narrow to traced file organization |
| usability-gate-hash-validation | "When specification process declares tree complete after multiple edit cycles without hash verification steps" | narrow to traced hash validation |
| usability-gate-mandatory-checkpoint | "When completing feature implementation and preparing to mark task as done in state tracking, after all unit tests pass but before updating completion status" | narrow to traced checkpoints |
| use-cases-standards-first-dogfooding | "When working on a design document or implementation plan, especially during dogfooding exercises where you're building something you'll use yourself" | too abstract |
| traced-core-artifact-support-scope | "During traced project scoping discussions about which artifact types to treat as core supported features versus optional extensions" | one-time scoping decision |
| terminology-three-axis-consistency | "When designing taxonomies or naming frameworks that group related concepts into categories (like design axes, architectural layers, or classification systems)" | too abstract |
| engram-tmux-lead-background-bash-monitor | "When checking how engram-tmux-lead monitors the chat channel for incoming messages" | captured in skill documentation |
| update-claude-superpowers-traced-references | "When working on Claude documentation, references, or related materials" | too vague to be useful |
| retroactive-linter-enforcement-policy | "When implementing linter checks on a codebase containing existing items that may not meet new format standards, and the user has explicitly requested retroactive enforcement over forward-only application" | one-time policy decision |
| user-prefers-direct-inspection-autonomy | "When debugging or testing web applications, especially when pages may be in incomplete states (blank, loading, or error states)" | one-time user preference |
| validate-tiering-category-independence | "Working on software project architecture with multi-tier structure where concerns about independence and proper separation between requirements, architecture, design, and testing layers have been raised" | redundant with traced spec process |
| oauth-credentials-session-logs | "When working on engram project and needing authentication credentials for API calls or services" | too narrow, one-session credential lookup |
| pressure-tests-run-externally-fish | "When working with pressure tests in the specwalk project and deciding how to run fish test scripts like tests/pressure/t-70.fish" | narrow to specwalk project fish tests |
| targ-claude-resume-continue-passthrough | "When invoking targ claude with --resume or --continue flags that should be passed to the underlying claude command" | specific targ bug, likely fixed or obsolete |
