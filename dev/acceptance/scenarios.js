// Scenario inputs (793b276): keyed by scenario ID, inputs only. A scenario with no entry here is not built yet. `turns`
// maps the quoted event of a "a turn completes in which ..." step to the prompt that elicits it and the observation
// that confirms the agent did it; the outcomes are asserted by the steps, never here. 793e8f: the seeds of the summary
// scenarios. Three notes on one situation are three items for a delivery; a near offer states the claim of one of them
// and adds a claim it lacks; an absent offer states a claim no note does. The quoted events of a scenario name what the
// System does; the seeds and the prompt make the model do it (the counts are asserted by the Then, never here).
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

// A lesson is stated by a note that says every part of it, as whole words (a model words the note, so the words are
// patterns); each pattern is matched against what the note says, never its source line (793b316).
const says = (...parts) => new RegExp(parts.map((part) => `(?=[\\s\\S]*${part})`).join(''), 'i')
const TARG_NOT_GO_TEST = says('\\btarg\\b', '\\bgo test\\b')
const RELEASES_FROM_MAIN = says('\\breleases?\\b', '\\btag(?:s|ged|ging)?\\b', '\\bmain\\b')
const FIXTURE_UNSET = says('\\bFIXTURE_DIR\\b', '\\b(?:unset|not set|missing)\\b')
const TARG_TEST_NOT_GO_TEST = says('\\btarg test\\b', '\\bgo test\\b')

// A note the vault holds that no scenario prompt is about: it is found by the lookup of any prompt (it states one of
// the four moments of recall) and bears on none of them, so that the judgement of relevance has a candidate to decline.
const UNRELATED_NOTE = {
  situation: 'starting to build a new approach to caching in a web service',
  says: 'measure the hit rate before adding a cache',
}

// The lesson a subagent finds out by itself: it states a guess, reads the user's feedback in a file, and corrects its
// own guess in its next message. The parent's prompt never states the lesson, so a note that states it was written from
// what the subagent itself said (793b316). The window of a capture holds what the user and the agent said, not what a
// tool gave back, so the subagent has to say the rule.
const FEEDBACK_FILE = {
  'feedback.txt': 'User feedback on the last run: "I ran `go test ./internal/cli`. No, run targ test, not go test."\n',
}
const ACKNOWLEDGE =
  'Before looking at anything, state in one sentence which command you would use to ' +
  'run the tests of internal/cli. Then read feedback.txt with the Read tool. Then state ' +
  'in one sentence the rule that the feedback gives, saying that your first guess was ' +
  'wrong. Do not do anything else.'

// the rules a working agent copies into the brief of a subagent (793b441)
const BRIEF_RULES_FILE = {
  'rules.txt':
    'Rules for the subagent:\n1. use 793b IDs from the next free number\n2. never push the branch\n' +
    '3. commit with exactly one AI-Used trailer\n',
}
const BRIEF_PROMPT =
  'Read rules.txt with the Read tool. Then dispatch a subagent with the Agent tool (type general-purpose) whose ' +
  'task is exactly: "Reply ready. Follow these rules", followed by the three rules of rules.txt copied word for ' +
  'word. When it returns, reply DONE.'

// the ended Pi session of the targ lesson: a run of the tests the user corrects, and the agent's last reply
const TARG_PI_SESSION = {
  lesson: 'use targ, not go test directly',
  opening: [
    ['user', 'Run the tests of this repository.'],
    ['assistant', 'I will run go test ./... in the repository root now.'],
  ],
  closing: 'Understood, I will use targ from now on and not call go test directly in this repository.',
}

export default {
  // 793e8i: the H1 scenarios are live experiments (h1/, steps/h1.js, kept under ENGRAM_EVAL_DIR since 793b343): their
  // inputs are the scripted sessions of h1/tours.js, so an entry here only marks a scenario as built.
  ...Object.fromEntries(
    ['a1a', 'a1b', 'a1c', 'b1a', 'c1a', 'c1b', 'c1c', 'd1a', 'e1a', 'e1b', 'f1a'].map((id) => [
      `793d17${id}`,
      { liveExperiment: true },
    ]),
  ),

  '793a5k1e': {
    turns: {
      'the agent runs an engram query on how releases are tagged in the middle of a task': {
        prompt:
          "In the middle of this task, run `engram query --phrase 'how releases are tagged'` " +
          'with the Bash tool, then tell me in one sentence what the top result says.',
        observe: { tool: 'Bash', command: /(^|[\s;&|(])engram\s+query\b/ },
      },
    },
  },

  // 793e8b: delivery at the moments. The prompt does not say what the work is or that it is done: the task is in a file
  // the agent reads, so the note can only reach the agent in the turn, at the tool result after the step in which the
  // agent says it is about to check the rename before it declares it done (the in-turn cue, 793b320). The agent is
  // asked to say what it is about to do before each tool call: the cue reads what the agent said, not what a tool gave
  // back.
  '793a2n1a': {
    deliveredMidTurn: true,
    turns: {
      'the working agent reaches the point of declaring the rename done': {
        files: {
          'task.md':
            'A colleague says they renamed the Go package `util` to `helpers` and updated the ' +
            'imports. Search the workspace once for any file that still mentions `internal/util`, ' +
            'then tell me in one sentence whether the rename is done. Before the search, say ' +
            'in one sentence what you are about to search for and what you will tell me if it finds nothing.\n',
        },
        prompt:
          'Follow the instructions in task.md. Before each tool call, say in one sentence what ' +
          'you are about to do and why.',
        observe: { tool: /^(Grep|Glob|Bash)$/, reply: /done|complete|finished/i },
      },
    },
  },
  '793a2n1e': {
    unrelatedNote: UNRELATED_NOTE,
    turns: {
      'the human submits a prompt': { prompt: 'What is two plus two? Answer in one word.' },
    },
  },
  '793a2n1g': {
    turns: {
      "the working agent dispatches a subagent with the task 'add tests for internal/cli'": {
        prompt:
          'Dispatch a subagent with the Agent tool (type general-purpose) whose task is exactly: ' +
          'add tests for internal/cli. Do not do the work yourself. When it returns, reply DONE.',
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
  // the unit of work is in a file, not in the prompt: the note can only be delivered when the agent calls route with it
  '793a2n1j': {
    deliveredAfter: { tool: 'Skill', skill: 'route' },
    turns: {
      'the orchestrator routes a unit of Go test writing through route': {
        files: { 'unit.md': 'The unit of work to route: writing Go tests for internal/cli.\n' },
        prompt:
          'Read unit.md with the Read tool. Then use the Skill tool to invoke the `route` skill ' +
          'for the unit of work it describes, passing that unit as the skill args, before you do anything else.',
        observe: { tool: 'Skill', skill: 'route', args: /Go tests/i },
      },
    },
  },
  // the stub of a chunk rides the result of a tool the agent uses after the prompt: the agent reads two files, one tool
  // result after the other, so that there is a result to ride (793a2n1k)
  '793a2n1k': {
    turns: {
      'the working agent works on rotating the staging database credentials': {
        files: { 'a.txt': 'first file\n', 'b.txt': 'second file\n' },
        prompt:
          'Next I am working on: rotating the staging database credentials. Use the Read tool on a.txt, then on ' +
          'b.txt, and then reply with one word: ok.',
        observe: { tool: 'Read' },
      },
    },
  },
  '793a2e1a': {
    turns: {
      "the agent's next two tool results match note X again": {
        prompt:
          'Run `echo tagging a release` with the Bash tool, then run `echo tagging a release ' +
          'again` with the Bash tool, then reply DONE.',
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
        prompt:
          "Run `engram query --phrase 'tagging a release'` with the Bash tool and tell me in " +
          'one sentence what the top result says.',
        observe: { tool: 'Bash', command: /engram\s+query/ },
      },
    },
  },
  '793a2e1d': {
    turns: {
      'the parent dispatches a subagent whose task matches note X': {
        prompt:
          'Dispatch a subagent with the Agent tool (type general-purpose) whose task is exactly: ' +
          'explain how a release gets tagged. Do not do the work yourself. When it returns, reply DONE.',
        observe: { tool: /^(Agent|Task)$/ },
      },
    },
  },
  '793a2e1e': { turns: {} },
  '793a2e1g': { turns: {} },

  // 793e8e: install by engram update. No scripted turns: the steps start the sessions the Thens read.
  '793a5j1a': { turns: {} },
  '793a5j1b': {
    turns: {},
    automatic: true,
    noMemoryCommands: true,
    lessons: { 'use targ, not go test directly': TARG_NOT_GO_TEST },
  },
  '793a5j1c': { turns: {} },
  '793a5j1d': { turns: {} },
  '793a5j1l': { turns: {}, unrelatedNote: UNRELATED_NOTE },

  // 793e8c: capture at turn end. `lessons` gives the words that state each lesson (a model words the note); `files`
  // are input files the prompt tells the agent to read, so that the situation reaches the agent through a tool result
  // and not through the prompt; `noMemoryCommands` makes a turn in which the agent saved the lesson itself a failed
  // precondition (the System writes it, 793a3a).
  '793a3a1a': {
    automatic: true,
    lessons: {
      'use the targ build system, not go test directly': TARG_NOT_GO_TEST,
      'releases are tagged from main': RELEASES_FROM_MAIN,
      'the corrected cause of the failing test': FIXTURE_UNSET,
    },
    turns: {
      "the user says 'no, use the targ build system, not go test directly'": {
        prompt:
          'Earlier you proposed checking my change with `go test ./...`. No, use the targ build ' +
          'system, not go test directly. Reply with one short sentence.',
        noMemoryCommands: true,
      },
      "the user says 'remember that releases are tagged from main'": {
        prompt: 'Remember that releases are tagged from main. Reply with the one word: ok.',
        noMemoryCommands: true,
      },
      'the agent states a cause for a failing test, a later command output overturns it, and the agent corrects itself':
        {
          prompt:
            'A test named TestSync is failing. First, before running anything, state in one sentence ' +
            "what you think the cause is, guessing from its name. Then run `echo 'TestSync failed: " +
            "FIXTURE_DIR is unset'` with the Bash tool. Then state in one sentence the real cause, " +
            'saying that your first guess was wrong.',
          observe: {
            tool: 'Bash',
            command: /echo/,
            reply: /(?=[\s\S]*\bFIXTURE_DIR\b)(?=[\s\S]*\b(?:wrong|incorrect|mistaken|actually|not the)\b)/i,
          },
          noMemoryCommands: true,
        },
    },
  },
  // 793b441: confirmation. The agent says its first guess was wrong and states a new cause with a hedge and no check:
  // nothing is written (the dogfooding capture wrote such a claim as a fact). The same claim once the user has
  // confirmed it is a lesson.
  '793a3a1r': {
    automatic: true,
    turns: {
      ['the agent says its first guess for a failing test was wrong and ' +
      'states a likely new cause that it has not checked']:
        {
          prompt:
            'A test named TestSync is failing and I have not looked at it. First say in one sentence what you ' +
            'think the cause is, guessing from its name. Then say in one sentence that this first guess was ' +
            'wrong, and that the cause is probably that FIXTURE_DIR is unset, but that you have not checked ' +
            'and it is only a hypothesis. Do not run anything.',
          observe: { reply: /(?=[\s\S]*\bFIXTURE_DIR\b)(?=[\s\S]*\b(?:probably|likely|hypothesis)\b)/i },
          noMemoryCommands: true,
        },
    },
  },
  '793a3a1s': {
    automatic: true,
    lessons: { 'the confirmed cause of the failing test': FIXTURE_UNSET },
    turns: {
      'the user confirms the cause the agent guessed for a failing test': {
        prompt:
          'Earlier you guessed that the failing test TestSync fails because FIXTURE_DIR is unset. I ran it and ' +
          'checked: you were right, FIXTURE_DIR is unset, and that is the cause. Reply with the one word: ok.',
        noMemoryCommands: true,
      },
    },
  },
  // 793b441: the rules the working agent writes into the brief of a subagent. The rules are in a file the agent reads
  // (a tool result is not in a window), so the user's words state none of them; the agent copies them into the brief.
  '793a3a1t': {
    automatic: true,
    fromSubagent: true,
    turns: {
      'the working agent briefs a subagent with three rules that the user did not state': {
        files: BRIEF_RULES_FILE,
        prompt: BRIEF_PROMPT,
        observe: { tool: /^(Agent|Task)$/, subagentsRead: /never push the branch/ },
        noMemoryCommands: true,
      },
    },
  },
  '793a3a1q': {
    automatic: true,
    lessons: {
      'exactly one AI-Used trailer per commit': says('\\bAI-Used\\b', '\\btrailer\\b'),
      'use 793b IDs from the next free number': says('\\b793b\\b', '\\bnext free\\b'),
      'never push the branch': says('\\bpush\\b', '\\bbranch\\b'),
    },
    turns: {
      'the user corrects one rule and the working agent briefs a subagent with that rule and two more': {
        files: BRIEF_RULES_FILE,
        prompt:
          'Your last commit carried two AI-Used trailers. No, exactly one AI-Used trailer per commit. ' + BRIEF_PROMPT,
        observe: { tool: /^(Agent|Task)$/, subagentsRead: /never push the branch/ },
        noMemoryCommands: true,
      },
    },
  },
  '793a3a1g': {
    automatic: true,
    fromSubagent: true,
    lessons: { 'run targ test, not go test': TARG_TEST_NOT_GO_TEST },
    turns: {
      "the working agent dispatches a subagent, tells it 'no, run targ test, not go test', and the subagent finishes": {
        files: FEEDBACK_FILE,
        prompt:
          `Dispatch a subagent with the Agent tool (type general-purpose) whose task is exactly: ` +
          `${ACKNOWLEDGE} When it returns, reply DONE.`,
        observe: { tool: /^(Agent|Task)$/, subagentsRead: /run targ test/ },
        noMemoryCommands: true,
      },
    },
  },
  '793a3a1h': {
    automatic: true,
    fromSubagent: true,
    lessons: { 'run targ test, not go test': TARG_TEST_NOT_GO_TEST },
    turns: {
      "two subagents finish at the same time, each containing the lesson 'run targ test, not go test'": {
        files: FEEDBACK_FILE,
        prompt:
          `In one single message, dispatch two subagents at the same time with the Agent tool ` +
          `(type general-purpose), each with the task, exactly: ${ACKNOWLEDGE} When both return, reply DONE.`,
        observe: { tool: /^(Agent|Task)$/, calls: 2, subagentsRead: /run targ test/ },
        noMemoryCommands: true,
      },
    },
  },
  '793a3a1j': {
    lessons: { 'use targ, not go test directly': TARG_NOT_GO_TEST },
    turns: {
      ["the user says 'no, use targ, not go test directly' and the agent runs the learn skill, " +
      'which writes a feedback note stating that rule']: {
        prompt:
          'No, use targ, not go test directly. Save that rule now by running the learn skill ' +
          'with the Skill tool, then reply DONE.',
        observe: { tool: 'Bash', command: /engram\s+learn\b/ },
      },
      'the user asks a neutral question': { prompt: 'What is two plus two? Answer in one word.' },
    },
    // 793e8g: the ended Pi session of the second and third examples; the second has its agent save the lesson, the
    // third has an earlier Claude Code turn capture it
    ended: { 'no, use targ, not go test directly': TARG_PI_SESSION },
    agentNote: {
      slug: 'use-targ-not-go-test-directly',
      source: 'Pi session; saved by the Pi agent running the learn skill',
      situation: 'running the tests of a Go repository',
      behavior: 'ran go test directly',
      impact: 'the repository gates were skipped',
      action: 'use targ, not go test directly',
    },
    earlierTurn: {
      prompt: 'No, use targ, not go test directly. Reply with one short sentence.',
      noMemoryCommands: true,
    },
  },

  // 793e8g: backlog capture. `ended` gives the conversation of an ended session around what the user said: its
  // opening, and the agent's last reply (a session killed mid-turn has none).
  '793a3a1k': {
    automatic: true,
    lessons: {
      'use targ, not go test directly': TARG_NOT_GO_TEST,
      'releases are tagged from main': RELEASES_FROM_MAIN,
    },
    turns: { 'the user asks a neutral question': { prompt: 'What is two plus two? Answer in one word.' } },
    ended: {
      'no, use targ, not go test directly': TARG_PI_SESSION,
      'remember that releases are tagged from main': {
        lesson: 'releases are tagged from main',
        opening: [
          ['user', 'Which branch do we tag releases from in this repository?'],
          ['assistant', 'I do not know which branch releases are tagged from here.'],
        ],
      },
    },
  },
  '793a5c1c': {
    lessons: { 'use targ, not go test directly': TARG_NOT_GO_TEST },
    turns: { 'the user asks a neutral question': { prompt: 'What is two plus two? Answer in one word.' } },
    ended: { 'no, use targ, not go test directly': TARG_PI_SESSION },
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
          'Next I am working on: tagging a release. No, use targ, not go test directly. First ' +
          'run `echo ok` with the Bash tool, then reply with one short sentence.',
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
          'Next I am working on: tagging a release. No, use targ, not go test directly. First ' +
          'run `echo ok` with the Bash tool, then reply with one short sentence.',
        observe: { tool: 'Bash', command: /^echo ok$/, delivered: true },
        noMemoryCommands: true,
      },
    },
  },
  '793a4c1f': { lessons: { 'use targ, not go test directly': TARG_NOT_GO_TEST }, turns: {} },
  '793a4b1a': { turns: { 'the human submits a prompt': { prompt: 'What is two plus two? Answer in one word.' } } },
  '793a4b1b': { turns: { 'the human submits a prompt': { prompt: 'What is two plus two? Answer in one word.' } } },
  '793a4b1c': { turns: { 'the human submits a prompt': { prompt: 'What is two plus two? Answer in one word.' } } },
  '793a4b1f': { turns: { 'the human submits a prompt': { prompt: 'What is two plus two? Answer in one word.' } } },
  '793a4b1g': { turns: { 'the human submits a prompt': { prompt: 'What is two plus two? Answer in one word.' } } },
  '793a4b1e': { unrelatedNote: UNRELATED_NOTE, turns: { 'the human submits a prompt': NEUTRAL } },

  // 793e8f: the summary, intentional memory and other harnesses.
  '793a5h1a': {
    // the subagent finishes in the background, after the turn that dispatched it: its note is in the next summary
    summaryAcrossTurns: true,
    turns: {
      ['the System delivers 3 items, adds 1 note, amends 1 note and curates 2 offers, ' +
      "and a subagent's session produces 1 of the added notes"]: {
        seeds: {
          notes: THREE_NOTES,
          offers: [
            { label: 'ON', covers: 'RA', extend: 'and the tag message names the version being released' },
            { label: 'OA' },
          ],
        },
        files: FEEDBACK_FILE,
        prompt:
          `Next I am working on: tagging a release. Dispatch a subagent with the Agent tool ` +
          `(type general-purpose) whose task is exactly: ${ACKNOWLEDGE} When it returns, reply DONE.`,
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
        prompt:
          '/recall how releases are tagged. Run each engram command plain, as the one command of its call, ' +
          'with no cd, pipe or redirect.',
        observe: { tool: 'Bash', command: /engram\s+query/ },
      },
    },
  },
  '793a5k1d': {
    agentWrites: true,
    lessons: { 'releases are tagged from main': RELEASES_FROM_MAIN },
    turns: {
      "the user says 'remember that releases are tagged from main' and the agent runs the learn skill": {
        prompt:
          'Remember that releases are tagged from main. Save it now by running the learn skill ' +
          'with the Skill tool, then reply DONE.',
        observe: { tool: 'Bash', command: /engram\s+learn\b/ },
      },
    },
  },
  '793a5c1a': { turns: { 'the human submits a prompt': NEUTRAL } },
  // SP14: no headless Pi, so the Givens end the row `Not run` before any session
  '793a5c1b': { turns: {} },
}
