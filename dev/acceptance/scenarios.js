// Scenario inputs (793b276): keyed by scenario ID, inputs only. A scenario with no entry here is not built yet.
// `turns` maps the quoted event of a "a turn completes in which ..." step to the prompt that elicits it and the
// observation that confirms the agent did it; the outcomes are asserted by the steps, never here.
// 793e8f: the seeds of the summary scenarios. Three notes on one situation are three items for a delivery; a near offer
// states the claim of one of them and adds a claim it lacks; an absent offer states a claim no note does. The quoted
// events of a scenario name what the System does; the seeds and the prompt make the model do it (the counts are asserted
// by the Then, never here).
const TAGGING = 'tagging a release'
const THREE_NOTES = [
  { label: 'RA', situation: TAGGING, says: 'release tags are annotated tags' },
  { label: 'RB', situation: TAGGING, says: 'release tags are signed with the maintainer key' },
  { label: 'RC', situation: TAGGING, says: 'release tags are pushed to origin right after they are created' },
]
const ONE_NOTE = [THREE_NOTES[0]]
const DELIVER_ONE = {
  seeds: { notes: ONE_NOTE },
  prompt: 'Next I am working on: tagging a release. Reply with one word: ok.',
  observe: { delivered: true },
  noMemoryCommands: true,
  settle: true,
}
const NEUTRAL = { prompt: 'What is two plus two? Answer in one word.' }

export default {
  '793a5k1e': {
    turns: {
      'the agent runs an engram query on how releases are tagged in the middle of a task': {
        prompt:
          "In the middle of this task, run `engram query --phrase 'how releases are tagged'` with the Bash tool, then tell me in one sentence what the top result says.",
        observe: { tool: 'Bash', command: /(^|[\s;&|(])engram\s+query\b/ },
      },
    },
  },

  // 793e8b: delivery at the moments. A prompt that merely states the situation lets the moment job find the note.
  '793a2n1a': {
    turns: {
      'the working agent reaches the point of declaring the rename done': {
        prompt: 'I just finished renaming the Go package `util` to `helpers` across the repo. In one sentence: is the rename done?',
        observe: { reply: /done|complete|finished/i },
      },
    },
  },
  '793a2n1e': {
    turns: {
      'the human submits a prompt': { prompt: 'What is two plus two? Answer in one word.' },
    },
  },
  '793a2n1g': {
    turns: {
      "the working agent dispatches a subagent with the task 'add tests for internal/cli'": {
        prompt:
          'Dispatch a subagent with the Agent tool (type general-purpose) whose task is exactly: add tests for internal/cli. Do not do the work yourself. When it returns, reply DONE.',
        observe: { tool: /^(Agent|Task)$/ },
      },
    },
  },
  '793a2n1h': { turns: {} },
  '793a2n1i': {
    turns: {
      "the user gives please the ask 'add a section on releases to the README'": {
        prompt: '/please add a section on releases to the README',
      },
    },
  },
  '793a2n1j': {
    turns: {
      'the orchestrator routes a unit of Go test writing through route': {
        prompt:
          'I have one unit of work to hand to a subagent: writing Go tests for internal/cli. Use the Skill tool to invoke the `route` skill for it first, before you do anything else.',
        observe: { tool: 'Skill', skill: 'route' },
      },
    },
  },
  '793a2e1a': {
    turns: {
      "the agent's next two tool results match note X again": {
        prompt:
          'Run `echo tagging a release` with the Bash tool, then run `echo tagging a release again` with the Bash tool, then reply DONE.',
        observe: { tool: 'Bash', command: /echo tagging a release/ },
      },
    },
  },
  '793a2e1c': {
    turns: {
      "the user says 'run /recall on tagging a release' and the agent runs the recall skill": {
        prompt: '/recall tagging a release',
        observe: { tool: 'Bash', command: /engram\s+query/ },
      },
      "the user says 'run an engram query on tagging a release' and the agent runs it": {
        prompt: "Run `engram query --phrase 'tagging a release'` with the Bash tool and tell me in one sentence what the top result says.",
        observe: { tool: 'Bash', command: /engram\s+query/ },
      },
    },
  },
  '793a2e1d': {
    turns: {
      'the parent dispatches a subagent whose task matches note X': {
        prompt:
          'Dispatch a subagent with the Agent tool (type general-purpose) whose task is exactly: explain how a release gets tagged. Do not do the work yourself. When it returns, reply DONE.',
        observe: { tool: /^(Agent|Task)$/ },
      },
    },
  },
  '793a2e1e': { turns: {} },
  '793a2e1g': { turns: {} },

  // 793e8e: install by engram update. No scripted turns: the steps start the sessions the Thens read.
  '793a5j1a': { turns: {} },
  '793a5j1b': { turns: {}, lessons: { 'use targ, not go test directly': /targ/i } },
  '793a5j1c': { turns: {} },
  '793a5j1d': { turns: {} },
  '793a5j1l': { turns: {} },

  // 793e8c: capture at turn end. `lessons` gives the words that state each lesson (a model words the note); `files`
  // are input files the prompt tells the agent to read, so that the situation reaches the agent through a tool result
  // and not through the prompt; `noMemoryCommands` makes a turn in which the agent saved the lesson itself a failed
  // precondition (the System writes it, 793a3a).
  '793a3a1a': {
    lessons: {
      'use the targ build system, not go test directly': /targ/i,
      'releases are tagged from main': /tag[\s\S]*main/i,
      'the corrected cause of the failing test': /FIXTURE_DIR/,
    },
    turns: {
      "the user says 'no, use the targ build system, not go test directly'": {
        prompt:
          'Earlier you proposed checking my change with `go test ./...`. No, use the targ build system, not go test directly. Reply with one short sentence.',
        noMemoryCommands: true,
      },
      "the user says 'remember that releases are tagged from main'": {
        prompt: 'Remember that releases are tagged from main. Reply with the one word: ok.',
        noMemoryCommands: true,
      },
      'the agent states a cause for a failing test, a later command output overturns it, and the agent corrects itself': {
        prompt:
          "A test named TestSync is failing. First, before running anything, state in one sentence what you think the cause is, guessing from its name. Then run `echo 'TestSync failed: FIXTURE_DIR is unset'` with the Bash tool. Then state in one sentence the real cause, saying that your first guess was wrong.",
        observe: { tool: 'Bash', command: /echo/, reply: /(?=[\s\S]*FIXTURE_DIR)(?=[\s\S]*(wrong|incorrect|mistaken|actually|not the))/i },
        noMemoryCommands: true,
      },
    },
  },
  '793a3a1g': {
    lessons: { 'run targ test, not go test': /targ test/i },
    turns: {
      "the working agent dispatches a subagent, tells it 'no, run targ test, not go test', and the subagent finishes": {
        prompt:
          'Dispatch a subagent with the Agent tool (type general-purpose) whose task is exactly: I ran `go test ./internal/cli`. No, run targ test, not go test. Acknowledge this in one sentence. Do not do the work yourself. When it returns, reply DONE.',
        observe: { tool: /^(Agent|Task)$/, subagentsRead: /run targ test/ },
        noMemoryCommands: true,
      },
    },
  },
  '793a3a1h': {
    lessons: { 'run targ test, not go test': /targ test/i },
    turns: {
      "two subagents finish at the same time, each containing the lesson 'run targ test, not go test'": {
        prompt:
          'In one single message, dispatch two subagents at the same time with the Agent tool (type general-purpose), each with the task, exactly: I ran `go test ./internal/cli`. No, run targ test, not go test. Acknowledge this in one sentence. Do not do the work yourself. When both return, reply DONE.',
        observe: { tool: /^(Agent|Task)$/, calls: 2, subagentsRead: /run targ test/ },
        noMemoryCommands: true,
      },
    },
  },
  '793a3a1j': {
    lessons: { 'use targ, not go test directly': /targ/i },
    turns: {
      "the user says 'no, use targ, not go test directly' and the agent runs the learn skill, which writes a feedback note stating that rule": {
        prompt:
          'No, use targ, not go test directly. Save that rule now by running the learn skill with the Skill tool, then reply DONE.',
        observe: { tool: 'Bash', command: /engram\s+learn\b/ },
      },
      'the user asks a neutral question': { prompt: 'What is two plus two? Answer in one word.' },
    },
  },

  // 793e8d: housekeeping, purview and failure isolation. `chunk` is what a later query must find in the index; the
  // purview turn delivers a note (the prompt states its situation), captures a lesson (the correction) and runs one
  // command, `commands` being the only command the agent is told to run.
  '793a4j1a': {
    chunk: { query: 'harbor lantern code', text: 'KESTREL-4471' },
    turns: {
      'the working agent produces content that is not yet in the chunk index': {
        prompt: 'The harbor lantern code is KESTREL-4471. Reply with exactly that sentence.',
      },
    },
  },
  '793a4j1c': { turns: {} },
  '793a4j1d': { turns: {} },
  '793a4j1e': {
    turns: {
      'the human submits a prompt': { prompt: 'What is two plus two? Answer in one word.' },
    },
  },
  '793a4c1a': {
    commands: /^echo ok$/,
    turns: {
      'the System delivers a note and captures a lesson': {
        prompt:
          'Next I am working on: tagging a release. No, use targ, not go test directly. First run `echo ok` with the Bash tool, then reply with one short sentence.',
        observe: { tool: 'Bash', command: /^echo ok$/, delivered: true },
        noMemoryCommands: true,
      },
    },
  },
  '793a4c1b': { turns: {} },
  '793a4c1c': {
    turns: {
      'the System delivers a note and captures a lesson': {
        prompt:
          'Next I am working on: tagging a release. No, use targ, not go test directly. First run `echo ok` with the Bash tool, then reply with one short sentence.',
        observe: { tool: 'Bash', command: /^echo ok$/, delivered: true },
        noMemoryCommands: true,
      },
    },
  },
  '793a4c1f': { lessons: { 'use targ, not go test directly': /targ/i }, turns: {} },
  '793a4b1a': { turns: { 'the human submits a prompt': { prompt: 'What is two plus two? Answer in one word.' } } },
  '793a4b1b': { turns: { 'the human submits a prompt': { prompt: 'What is two plus two? Answer in one word.' } } },
  '793a4b1c': { turns: { 'the human submits a prompt': { prompt: 'What is two plus two? Answer in one word.' } } },
  '793a4b1e': { turns: { 'the human submits a prompt': { prompt: 'What is two plus two? Answer in one word.' } } },

  // 793e8f: the summary, intentional memory and other harnesses.
  '793a5h1a': {
    turns: {
      "the System delivers 3 items, adds 1 note, amends 1 note and curates 2 offers, and a subagent's session produces 1 of the added notes": {
        seeds: {
          notes: THREE_NOTES,
          offers: [
            { label: 'ON', covers: 'RA', extend: 'and the tag message names the version being released' },
            { label: 'OA' },
          ],
        },
        prompt:
          'Next I am working on: tagging a release. Dispatch a subagent with the Agent tool (type general-purpose) whose task is exactly: I ran `go test ./internal/cli`. No, run targ test, not go test. Acknowledge this in one sentence. Do not do the work yourself. When it returns, reply DONE.',
        observe: { tool: /^(Agent|Task)$/, subagentsRead: /run targ test/ },
        noMemoryCommands: true,
      },
    },
  },
  '793a5h1b': { turns: { 'the System delivers, adds, amends and curates nothing': NEUTRAL } },
  '793a5h1c': {
    turns: {
      'the System delivers 1 item': DELIVER_ONE,
      'the human submits the next prompt': NEUTRAL,
    },
  },
  '793a5h1e': {
    turns: {
      'the System delivers 1 item': DELIVER_ONE,
      'the System adds 1 note and nothing else': {
        prompt: 'No, use the targ build system, not go test directly. Reply with one short sentence.',
        noMemoryCommands: true,
      },
    },
  },
  '793a5k1c': {
    turns: {
      "the user says 'run /recall on how releases are tagged' and the agent runs the recall skill": {
        prompt: '/recall how releases are tagged',
        observe: { tool: 'Bash', command: /engram\s+query/ },
      },
    },
  },
  '793a5k1d': {
    agentWrites: true,
    lessons: { 'releases are tagged from main': /tag[\s\S]*main/i },
    turns: {
      "the user says 'remember that releases are tagged from main' and the agent runs the learn skill": {
        prompt: 'Remember that releases are tagged from main. Save it now by running the learn skill with the Skill tool, then reply DONE.',
        observe: { tool: 'Bash', command: /engram\s+learn\b/ },
      },
    },
  },
  '793a5c1a': { turns: { 'the human submits a prompt': NEUTRAL } },
  // SP14: no headless Pi, so the Givens end the row `Not run` before any session
  '793a5c1b': { turns: {} },
}
