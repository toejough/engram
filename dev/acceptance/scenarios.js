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
}
