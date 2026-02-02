# Flows

## new project

- User: Start claude
- User: Run /project let's start ISSUE-1234
- PM agent:
  - read the issue
  - evaluate: what problem is being solved, what is the current state, what is the desired state, what is the guidance
    from getting from here to there?
    - gather relevant context to try to answer those questions. Likely sources: from code, docs, git history, historical memory, and external research, open and recently close
      issues, design docs/files/screenshots, etc.
    - summarize the relevant context and the answers to the evaluation questions
    - ask the user to confirm or correct
    - repeat as needed until the user is satisfied
  - save answers and relevant context to a requirements markdown doc
  - in a new section in that requirements doc, break down the answers and context into individually trackable and verifiable requirements that follow our guidelines
  - Commit agent: commit according to conventional commits standards
- PM QA agent:
  - review the requirements doc for completeness and clarity of eval answers and context, clear traceability to the
    issue, clearly address the issue, and verifiable requirements that follow our guidelines. Identify any misalignment or gaps between the issue and the requirements.
  - raise to the user if it seems like the issue itself needs clarification or correction
  - otherwise, identify suggested improvements or additions as needed, for handoff to the PM agent.
  - repeat from PM agent until satisfied or we've hit a 3x limit (then raise to user).
- Design agent:
  - read the requirements doc
  - evaluate: what are the user experience impacts of the requirements (GUI, TUI, CLI, API, user or agent instructions, user or agent interaction
    workflows, etc)? what is the existing user experience for these situations (if any)? what should the new user experience be? what are the best practices and design patterns and tools that
    apply?
    - gather any additional context needed to inform design (user personas, user journey maps, competitive analysis,
      accessibility requirements, branding guidelines, etc). Likely sources: design docs/files/screenshots, implementation
      code, external research, historical memory, etc.
    - summarize the relevant context and the answers to the evaluation questions
    - generate example designs (sketches, wireframes, user flows, draft documentation, example scripts, screenshots, etc) to illustrate ideas
    - ask the user to choose, confirm, or correct
    - repeat as needed until the user is satisfied
  - save final designs and rationale to a design doc and any other relevant files (sketches, wireframes, user flows,
    draft documentation, example scripts, screenshots, etc)
  - in a new section in that design doc, break down the answers and context into individually trackable and verifiable requirements that follow our guidelines
  - Commit agent: commit according to conventional commits standards
- Design QA agent:
  - review the design doc for completeness and clarity of eval answers and context, clear traceability to the
    requirements, clearly address the requirements, and designs that follow our guidelines. Identify any misalignment or gaps between the requirements and the designs.
  - raise to the PM agent if it seems like the requirements themselves need clarification or correction
  - otherwise, identify suggested improvements or additions as needed, for handoff to the design agent.
  - repeat from design agent until satisfied or we've hit a 3x limit (then raise to user).
- Architect agent:
  - read the design & requirements docs
  - evaluate: what are the architectural impacts of the designs & requirements? what is the existing architecture for these situations (if any)? what should the new architecture be? what are the best practices and patterns and tools that apply?
    - gather any additional context needed to inform architecture (system diagrams, data flow diagrams, infrastructure diagrams, scalability requirements, performance requirements, security requirements, etc). Likely sources: requirements docs, design docs/files/screenshots, implementation code, external research, historical memory, etc.
    - summarize the relevant context and the answers to the evaluation questions
    - generate example architectures (diagrams, data models, infrastructure plans, etc) to illustrate ideas
    - ask the user to choose, confirm, or correct
    - repeat as needed until the user is satisfied
  - save final architectures and rationale to an architecture doc and any other relevant files (diagrams, data models,
    infrastructure plans, etc)
  - in a new section in that architecture doc, break down the answers and context into individually trackable and verifiable requirements that follow our guidelines
  - Commit agent: commit according to conventional commits standards
- Architect QA agent:
  - review the architecture doc for completeness and clarity of eval answers and context, clear traceability to the
    design & requirements, clearly address the design & requirements, and architectures that follow our guidelines. Identify any misalignment or gaps between the design & requirements and the architectures.
  - raise to the pm or design agents if it seems like the requirements or designs themselves needs clarification or correction
  - otherwise, identify suggested improvements or additions as needed, for handoff to the architect agent.
  - repeat from architect agent until satisfied or we've hit a 3x limit (then raise to user).
- Task breakdown agent:
  - read the architecture docs
  - evaluate: what are the individual tasks needed to implement the architecture? what is the existing implementation?
    what should the new implementation be? what are the best practices and patterns that apply for breaking down and
    implementing this work?
    - gather any additional context needed to inform task breakdown (implementation code, historical memory, external
      research, etc)
    - summarize the relevant context and the answers to the evaluation questions
    - ask the user to confirm or correct
    - repeat as needed until the user is satisfied
  - save final task breakdown and rationale to a task breakdown doc
  - in a new section in that task breakdown doc, break down the answers and context into individually trackable and verifiable tasks that follow our guidelines
  - Commit agent: commit according to conventional commits standards
- Task breakdown QA agent:
  - review the task breakdown doc for completeness and clarity of eval answers and context, clear traceability to the
    architecture, clearly address the architecture, and tasks that follow our guidelines. Identify any misalignment or gaps between the architecture and the tasks.
  - raise to the architect agent if it seems like the architecture itself needs clarification or correction
  - otherwise, identify suggested improvements or additions as needed, for handoff to the task breakdown agent.
  - repeat from task breakdown agent until satisfied or we've hit a 3x limit (then raise to user).
- Task Loop Agent:
  - read the task breakdown doc
  - queue up tasks in order of dependencies, structural impact, and simplicity: if a task is blocked by a dependency, it
    cannot run. If a task has high structural impact, it should run earlier. If a task is simple, it should run earlier.
  - for each task in the queue:
    - TDD Agent
      - Red (test) Agent
        - evaluate: what are the test impacts of the task? what is the existing testing for this task (if any)? what should the new tests be? what are the best practices and patterns and tools that apply?
          - gather any additional context needed to inform testing (test patterns present in the repo, likely edge cases, desired tooling or methodology, etc). Likely sources: requirements docs, design docs/files/screenshots, other test code, git history, external research, historical memory, etc.
        - write or update the relevant tests, expect them to fail, and tag them with the relevant
          task/design/requirement IDs.
        - Commit agent: commit according to conventional commits standards
      - Test Agent QA
        - review the tests for completeness and clarity of eval answers and context, clear traceability to the
          task/design/requirement, clearly address the task/design/requirement, and tests that follow our guidelines. Identify any misalignment or gaps between the task/design/requirement and the tests.
        - raise to the task loop agent if it seems like the task itself needs clarification or correction
        - otherwise, identify suggested improvements or additions as needed, for handoff to the test agent.
        - repeat from test agent until satisfied or we've hit a 3x limit (then raise to user).
      - Green (Implementation) Agent
        - evaluate: what are the implementation impacts of the task? what is the existing implementation for this task (if
          any)? what should the new implementation be? what are the best practices and patterns and tools that apply?
          - gather any additional context needed to inform implementation (implementation code, historical memory, external
            research, etc)
          - write or update the relevant implementation code, tagged with the relevant
            test names.
          - Ensure the targeted tests pass. Repeat implementation as needed until tests pass.
          - ensure all tests pass. Repeat implementation as needed until all tests pass.
          - Commit agent: commit according to conventional commits standards
      - Implementation Agent QA
        - review the implementation code for completeness and clarity, clear traceability to the
          tests, clearly address the task, and implementation that follow our guidelines, and all tests pass. Identify any misalignment or gaps between the task and the implementation.
        - raise to the task loop agent if it seems like the task itself needs clarification or correction
        - otherwise, identify suggested improvements or additions as needed, for handoff to the implementation agent.
        - repeat from implementation agent until satisfied or we've hit a 3x limit (then raise to user).
      - Refactor Agent
        - evaluate: what are the refactoring opportunities in the implementation code for this task? what is the current
          state of the codebase? what is the target state? what are the best
          practices and patterns and tools that apply?
          - gather any additional context needed to inform refactoring (implementation code, historical memory, external
            research, etc)
          - refactor the relevant implementation code, maintaining all existing functionality, and tagged with the relevant
            test names.
          - ensure all tests and linting rules pass. Repeat refactoring as needed until all tests and linting rules pass.
          - Commit agent: commit according to conventional commits standards
      - Refactor Agent QA
        - review the refactored implementation code for completeness and clarity, clear traceability to the
          tests, clearly address the task, and implementation that follow our guidelines, and all tests and linting rules pass. Identify any misalignment or gaps between the task and the implementation.
        - raise to the task loop agent if it seems like the task itself needs clarification or correction
        - otherwise, identify suggested improvements or additions as needed, for handoff to the refactor agent.
        - repeat from refactor agent until satisfied or we've hit a 3x limit (then raise to user).
    - TDD Agent QA
      - review the overall task implementation and tests for completeness and clarity, clear traceability of
        implementation to test, of test to the
        task, clearly addresses the task, and test & implementation that follow our guidelines, and all tests and linting pass. Identify any misalignment or gaps between the task and the tests or implementation.
      - raise to the task agent if it seems like the task itself needs clarification or correction
      - otherwise, identify suggested improvements or additions as needed, for handoff to the tdd agent.
      - repeat from tdd agent until satisfied or we've hit a 3x limit (then raise to user).
  - mark the task complete
  - re-evaluate and if necessary re-order the remaining tasks in the queue based on any changes from completed tasks.
  - repeat until all tasks are complete.
- tech-writer agent:
  - read the current repo docs (requirements, design, architecture, README, etc) and the entire project history
    (issues, requirements, designs, architectures, task breakdowns, tasks, tests, implementation code, git history,
    etc)
  - evaluate: what repo-level documentation is needed to support the completed project? what is the existing documentation (if
    any)? what should the new documentation be? what are the best practices and patterns and tools that apply?
    - gather any additional context needed to inform documentation (historical memory, external research, etc)
    - summarize the relevant context and the answers to the evaluation questions
    - ask the user to confirm or correct
    - repeat as needed until the user is satisfied
  - save final documentation to relevant repo-level docs (requirements, design, architecture, README, etc)
  - Commit agent: commit according to conventional commits standards
- tech-writer QA agent:
  - review the repo-level documentation for completeness and clarity of eval answers and context, clear traceability to the
    project, clearly address the project, and documentation that follow our guidelines. Identify any misalignment or gaps between the project and the documentation.
  - raise to the user if it seems like the project itself needs clarification or correction
  - otherwise, identify suggested improvements or additions as needed, for handoff to the tech-writer agent.
  - repeat from tech-writer agent until satisfied or we've hit a 3x limit (then raise to user).
- retro agent:
  - read the entire project history (issues, requirements, designs, architectures, task breakdowns, tasks, tests,
    implementation code, git history, etc)
  - evaluate: what went well? what could be improved? what were the blockers or challenges? what are the action items
    for future projects? What lessons were learned? what patterns or practices should be adopted or avoided in the
    future?
    - gather any additional context needed to inform retro (historical memory, external research, etc)
    - summarize the relevant context and the answers to the evaluation questions
    - ask the user to confirm or correct or add
    - repeat as needed until the user is satisfied
  - save final retro and rationale to a retro doc
  - file issues as necessary for action items with the relevant repos, and link bidirectionally between the retro doc
    and the issues.
  - Commit agent: commit according to conventional commits standards
- summary agent:
  - read the entire project history (issues, requirements, designs, architectures, task breakdowns, tasks, tests,
    implementation code, git history, retro doc, etc)
  - evaluate: what was accomplished? what are the key changes? what are the important details? what should be
    highlighted for future reference?
    - gather any additional context needed to inform summary (historical memory, external research, etc)
    - summarize the relevant context and the answers to the evaluation questions
    - ask the user to confirm or correct
    - repeat as needed until the user is satisfied
  - save final summary and rationale to a summary doc
  - Commit agent: commit according to conventional commits standards
- next steps agent:
  - read the open issues in this repo and suggest next steps for the user based on the completed project and the open
    issues.

# agent handoff and orchestration

When an agent needs to call another agent, unless it's the top level orchestrator agent, it should package up all
relevant context, save it, and then yield back to the orchestrator agent with a request to call the agent it wants. The
orchestrator should call that agent, likewise handling any of that agent's sub-agent calls, and then return the result
and saved context back to the original agent, which can then continue its work.

the same pattern should apply for sub-agents that want to interact with the user.

The top level agent should be a pure orchestrator that handles agent/sub-agent/user handoff & context management, and
nothing else.

# State management

As much as possible, state management for the project should be managed through a deterministic tool like projctl.

# Tool / model selection

Prefer the most reliable tool for the job. If a deterministic tool can reliably handle the task, use or build one for
that task. If a local ONNX model can handle it, use that. If an LLM is needed, prefer the cheapest, simplest model that
can handle the task reliably.
