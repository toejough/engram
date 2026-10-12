// Scenario inputs (793b276): keyed by scenario ID, inputs only. A scenario with no entry here is not built yet.
// `turns` maps the quoted event of a "a turn completes in which ..." step to the prompt that elicits it and the
// observation that confirms the agent did it; the outcomes are asserted by the steps, never here.
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
}
