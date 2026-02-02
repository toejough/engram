# question responses

1. I imagine ID's being created in the relevant do-er agent when they break their responsibilities down - for
   instance when the design agent breaks down answers into individually trackable requirements. Then it should use
   projctl to get the next design ID for the repo and assign it.
2. I'd like you to think about what an appropriate handoff format would be.
3. we have guidelines on good requirements, designs, tests, implementation, tools, process, etc, scattered throughout
   our skills and projctl and claude.md. Can you find the ones you think are relevant and consolidate them into a
   reference for me to review in your unified doc?
4. We could extract the QA loops into a set of agents... probably should: X Loop: X-implementer, X-QA. X Loop agent
   tracks the state, via updates via projctl.
5. I'm imagining the /commit skill.
6. What research exists for this? I think "relevant context" is ambiguous, and I would like LLM help even figuring out
   what to look for, or how to constrain it. We can/should provide defautls and constraints for common situations, but
   this will always need a little judgement applied.
7. automated analysis. Dependencies should be based on how the tasks are marked. structural impact and simplicity will
   require LLM review. I'd like the cheapest possible LLM analysis that will get the job done.

# suggested reorg

## 1. core patterns

PHASES:

1. PM
1. DESIGN
1. ARCHITECTURE
1. TASK BREAKDOWN
1. IMPLEMENTATION
1. DOCUMENTATION
1. RETROSPECTIVE
1. SUMMARY
1. NEXT STEPS

LOOPER AGENT:

1. Create/Recreate Queue (tasks to do)
1. if task in Queue:
   1. Run PAIR LOOP with first task
   1. Return to start and recreate Queue as needed
1. Stop & return only when queue is empty or entirely blocked.

PAIR LOOP:

1. PRODUCER agent
1. QA agent
1. Evaluate outcome:
   1. If approved, return outcome to LOOPER AGENT
   1. If needs improvement, return to PRODUCER
   1. If needs escalation, return to prior PHASE or user

PRODUCER AGENT:

1. Evaluate (questions to answer)
1. Gather (sources to consult)
1. Synthesize (summarize findings)
1. Confirm (user or prior agent approval)
1. Produce (artifact with traceable IDs)
1. Commit (checkpoint)

QA LOOP:

1. Review (against criteria)
1. Return (approved, improve, escalate)

## 2 Phase definitions

I like this structure.

## 3 Yield protocol

I prefer TOML

## 4 state serialization

this looks fine. I want to be able to pick up where I left off if interrupted at any point.

## 5 traceability chain

Downward, for handling new work based on issues:

- ISSUE -> REQUIREMENT -> DESIGN -> ARCHITECTURE -> TASK -> test -> implementation
- Every issue should get broken down into requirements. Every requirement should have a user experience impact (to be
  designed) and a technical impact (to be architected).
- ID's should point back up the stack not down. Requirements point to issues, designs point to requirements,
  architecture points to designs, tasks point to architecture, tests point to tasks, implementations point to tests.
- Implementation is in code/docs and should name the test it satisfies.
- tests are in code, and should name the task they satisfy.

Upward, for understanding existing work:

- IMPLEMENTATION -> TEST -> (TASK) -> ARCHITECTURE -> DESIGN -> REQUIREMENT -> (ISSUE)
- higher level items may not exist for existing implementations. We should be able to infer the non-optional ones
- TASKs and ISSUEs are optional. There's no reason to create them if the lower level items don't map nicely to existing
  tasks or issues.
- Tests, architecture, designs, and requirements should always be inferred and created if they don't exist or don't have
  clean mappings. When exploring existing work, escalate progressively to higher level agents to fill in missing context
  if the current level is not inferrable. For example, if an implementation has no obvious test, and it's not clear what
  the intended property to validate is, escalate to the architecture level to try to get more context. If the
  architecture level can't infer reasonable context, escalate to design, and so on. Ultimately, if it is unclear why
  something exists, escalate to the user.

## 6. Support systems

- also the image diffing / analysis in projctl

## 7 workflows

### 7.1 New Project (this document)

PHASES:

1. PM
1. DESIGN
1. ARCHITECTURE
1. TASK BREAKDOWN
1. IMPLEMENTATION
1. DOCUMENTATION
1. RETROSPECTIVE
1. SUMMARY
1. NEXT STEPS

### 7.2 Adopt Existing (TODO)

PHASES:

1. EXPLORE IMPLEMENTATION
1. INFER & INTEGRATE TESTS
1. INFER & INTEGRATE ARCHITECTURE
1. INFER & INTEGRATE DESIGN
1. INFER & INTEGRATE REQUIREMENTS
1. INFER & INTEGRATE DOCUMENTATION
1. RETROSPECTIVE
1. SUMMARY
1. NEXT STEPS

### 7.3 Align Drift (TODO)

Same as adopt

### 7.4 Single Task (TODO)

1. IMPLEMENTATION
1. DOCUMENTATION
1. RETROSPECTIVE
1. SUMMARY
1. NEXT STEPS

### 7.5 intake

1. EVALUATE REQUEST
1. CREATE NECESSARY ISSUES
   1. if request was just tto file an issue, DONE
1. DISPATCH TO WORKFLOW
1. CLOSE NECESSARY ISSUES

New work should get a new issue before we execute, unless they exist already.

The evaluation phase should determine if this is likely to become a multi-task project, and if so, use the `new project`
workflow. If it's likely to be a single task, use the `single task` workflow. If it's likely to involve existing work,
resume that open project/task.

Missing from Unified Design
┌─────────────────────────────────────────┬──────────────────────────────────────┐
│ In Unified Design │ In Redesign Thoughts │
├─────────────────────────────────────────┼──────────────────────────────────────┤
│ Yield protocol TOML formats │ Mentioned but not specified │ (use unified design)
├─────────────────────────────────────────┼──────────────────────────────────────┤
│ Context serialization (role-state.toml) │ Implicit │ (use unified design)
├─────────────────────────────────────────┼──────────────────────────────────────┤
│ Territory mapping phase │ Implicit in "gather context" │ (use territory mapping in the context gathering)
├─────────────────────────────────────────┼──────────────────────────────────────┤
│ Memory system (ONNX + SQLite-vec) │ "historical memory" with no detail │ (use the memory system from unified design)
├─────────────────────────────────────────┼──────────────────────────────────────┤
│ Model routing per role/mode │ Not mentioned │ (use model routing from unified design)
├─────────────────────────────────────────┼──────────────────────────────────────┤
│ Adopt/align workflows │ Only "new" workflow shown │ (filled in above)
├─────────────────────────────────────────┼──────────────────────────────────────┤
│ State machine diagram │ Linear prose, no state visualization │ (create a new state machine diagram)
├─────────────────────────────────────────┼──────────────────────────────────────┤
│ File layout │ Not specified │ (all repo level docs should go in repo/docs, except README.md, which should be in the
top level. project-specific docs should go in repo/docs/projects/<project-name>/)
├─────────────────────────────────────────┼──────────────────────────────────────┤
│ CLI commands │ Not specified │ (re-evaluate and re-define projctl commands as needed. Highlight the changes for
review/discussion/rationale)
├─────────────────────────────────────────┼──────────────────────────────────────┤
│ Error/retry protocol │ 3x limit mentioned but no format │ (use the error/retry protocol from unified design, with the
3x limit)
├─────────────────────────────────────────┼──────────────────────────────────────┤
│ Role modes (interview/infer/audit) │ Separate agents instead of modes │ (use separate agents via skills that launch
them with clean context, like we do today)
└─────────────────────────────────────────┴──────────────────────────────────────┘
