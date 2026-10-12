// Contract tests of the delivery moments (793e8b): the process and the model are stubbed under the mod, and record
// the query arguments, the request and the order of the calls.
import { describe, expect, test } from 'claude-code/testing'

const QUERY_OUTPUT = `version: 1
phrases:
  - declaring work done
items:
  - path: 1.2026-10-12.note-n.md
    kind: fact
    score: 0.49
    provenances:
      - direct
    content: |+
      ---
      type: fact
      situation: declaring a Go package rename done
      object: run go build first (SENTINEL-N)
      ---

      Information learned: run go build first (SENTINEL-N).

clusters:
`

const EMPTY_OUTPUT = 'version: 1\nphrases: []\nitems: []\nclusters: []\n'

// bottom stands for the engine: the transcript file does not exist yet, the log goes nowhere, and the prompt enters.
function bottom(on: any, transcript: { text: string | null } = { text: null }) {
  on('fs.read', async () => (transcript.text === null ? { deny: 'ENOENT' } : { value: transcript.text }))
  on('ui.log', async () => ({ value: undefined }))
  on('prompt.submit', async (_$: any, e: any) => ({ text: e.text, context: e.context }))
}

function stubs(on: any, output: string, reply: string, transcript?: { text: string | null }) {
  const calls: string[] = []
  const requests: any[] = []
  on('process.run', async (_$: any, e: any) => {
    calls.push(e.argv.join(' '))
    return { value: { exitCode: 0, stdout: output, stderr: '' } }
  })
  on('model.complete', async (_$: any, e: any) => {
    calls.push('model')
    requests.push(e)
    return { value: { isAnswered: true, text: reply, usage: { input_tokens: 1, output_tokens: 1 } } }
  })
  bottom(on, transcript)
  return { calls, requests }
}

const PROMPT = { text: 'I just finished renaming the package. Is it done?', wait: false, origin: { kind: 'sdk' } }

describe('prompt delivery', () => {
  test('a relevant note rides the prompt context with the apply instruction, after the query and one judgement', async ($: any, on: any) => {
    const { calls, requests } = stubs(on, QUERY_OUTPUT, '[1]')
    const result = await $.prompt.submit(PROMPT)
    expect(result.text).toBe(PROMPT.text)
    expect(result.context.length).toBe(1)
    expect(result.context[0]).toContain('[engram memory]')
    expect(result.context[0]).toContain('Apply what bears on your next step.')
    expect(result.context[0]).toContain('SENTINEL-N')
    expect(calls.length).toBe(2)
    expect(calls[0].startsWith('engram query --limit 10 --text')).toBe(true)
    expect(calls[0]).toContain('--phrase declaring work done')
    expect(calls[1]).toBe('model')
    expect(requests[0].model).toBe('haiku')
  })

  test('nothing relevant is nothing delivered, and a query with no result makes no judgement', async ($: any, on: any) => {
    const { calls } = stubs(on, EMPTY_OUTPUT, '[]')
    const result = await $.prompt.submit(PROMPT)
    expect(result.context ?? []).toEqual([])
    expect(calls.length).toBe(1)
  })

  test('a judgement that selects nothing delivers nothing', async ($: any, on: any) => {
    stubs(on, QUERY_OUTPUT, '[]')
    const result = await $.prompt.submit(PROMPT)
    expect(result.context ?? []).toEqual([])
  })

  test('a note delivered once is not delivered again in the same session', async ($: any, on: any) => {
    const { calls } = stubs(on, QUERY_OUTPUT, '[1]')
    await $.prompt.submit(PROMPT)
    const second = await $.prompt.submit(PROMPT)
    expect(second.context ?? []).toEqual([])
    expect(calls.filter((call) => call === 'model').length).toBe(1)
  })

  test('a failing query leaves the prompt as typed', async ($: any, on: any) => {
    on('process.run', async () => ({ value: { exitCode: 1, stdout: '', stderr: 'boom' } }))
    bottom(on)
    const result = await $.prompt.submit(PROMPT)
    expect(result.text).toBe(PROMPT.text)
    expect(result.context ?? []).toEqual([])
  })
})

const TRANSCRIPT = '/home/.claude/projects/p/s.jsonl'

describe('presence', () => {
  test('a note the context already holds is not delivered, and one a compaction removed is', async ($: any, on: any) => {
    const file = { text: '{"type":"user","message":"1.2026-10-12.note-n.md was retrieved by the agent"}' }
    const { calls } = stubs(on, QUERY_OUTPUT, '[1]', file)
    on('classic.SessionStart', async () => ({}))
    await $.classic.SessionStart({ source: 'startup', transcript_path: TRANSCRIPT })
    const held = await $.prompt.submit(PROMPT)
    expect(held.context ?? []).toEqual([])
    expect(calls.filter((call) => call === 'model').length).toBe(0)
    file.text += '\n{"type":"system","subtype":"compact_boundary"}\n{"type":"user","isCompactSummary":true,"message":"1.2026-10-12.note-n.md summarised"}'
    const after = await $.prompt.submit(PROMPT)
    expect(after.context.length).toBe(1)
  })

  test('the ledger of the working agent is emptied by a compaction', async ($: any, on: any) => {
    stubs(on, QUERY_OUTPUT, '[1]')
    on('session.compact', async (_$: any, e: any) => ({ messages: e.messages }))
    await $.prompt.submit(PROMPT)
    await $.session.compact({ trigger: 'manual', messages: [{ role: 'user', text: 'hi', toolUses: [] }] })
    const again = await $.prompt.submit(PROMPT)
    expect(again.context.length).toBe(1)
  })
})

describe('cues', () => {
  test('the Skill call of route is delivered next to its tool result, and no other skill is', async ($: any, on: any) => {
    const { calls } = stubs(on, QUERY_OUTPUT, '[1]')
    on('classic.PostToolUse', async () => ({}))
    const other = await $.classic.PostToolUse({ tool_name: 'Skill', tool_input: { skill: 'recall' }, tool_response: '', tool_use_id: 't1' })
    expect(other.additionalContext ?? []).toEqual([])
    expect(calls.length).toBe(0)
    const route = await $.classic.PostToolUse({ tool_name: 'Skill', tool_input: { skill: 'route', args: 'write Go tests' }, tool_response: '', tool_use_id: 't2' })
    expect(route.additionalContext.length).toBe(1)
    expect(route.additionalContext[0]).toContain('[engram memory]')
    expect(calls[0]).toContain('--phrase routing a unit of work to a model tier')
  })

  test('a subagent receives the notes for its dispatch prompt at its start, whatever its parent holds', async ($: any, on: any) => {
    const { calls } = stubs(on, QUERY_OUTPUT, '[1]')
    on('agent.spawn', async () => ({ model: 'sonnet' }))
    on('classic.SubagentStart', async () => ({}))
    await $.prompt.submit(PROMPT)
    await $.agent.spawn({ tool_use_id: 't', prompt: 'add tests for internal/cli', description: 'tests', subagentType: 'general-purpose' })
    const start = await $.classic.SubagentStart({ agent_id: 'a1', agent_type: 'general-purpose' })
    expect(start.additionalContext.length).toBe(1)
    expect(calls.filter((call) => call.includes('--text add tests for internal/cli')).length).toBe(1)
  })
})
