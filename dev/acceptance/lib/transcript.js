// Readers for the session transcript JSONL that Claude Code writes (793b246). Row shapes are those observed in the
// fixture test/fixtures/engram-query-turn.jsonl (Claude Code 2.1.296).
import fs from 'node:fs'

const MEMORY_SKILLS = new Set(['recall', 'learn', 'curate', 'write-memory'])

export function readRows(file) {
  const rows = []
  for (const line of fs.readFileSync(file, 'utf8').split('\n')) {
    if (line.trim() === '') continue
    try {
      rows.push(JSON.parse(line))
    } catch {
      // a partially written last line is not a row
    }
  }
  return rows
}

function blocks(row) {
  const content = row?.message?.content
  return Array.isArray(content) ? content : []
}

export function userPrompts(rows) {
  const prompts = []
  for (const row of rows) {
    if (row.type !== 'user') continue
    const content = row.message?.content
    if (typeof content === 'string') prompts.push(content)
    else {
      const text = blocks(row).filter((block) => block.type === 'text').map((block) => block.text).join('\n')
      if (text !== '') prompts.push(text)
    }
  }
  return prompts
}

function resultText(block) {
  if (typeof block.content === 'string') return block.content
  if (Array.isArray(block.content)) return block.content.map((part) => part.text ?? '').join('\n')
  return ''
}

// toolCalls pairs each tool_use with its tool_result: { id, name, input, result: { text, isError } | null }.
export function toolCalls(rows) {
  const calls = new Map()
  for (const row of rows) {
    for (const block of blocks(row)) {
      if (block.type === 'tool_use') calls.set(block.id, { id: block.id, name: block.name, input: block.input ?? {}, result: null })
      else if (block.type === 'tool_result' && calls.has(block.tool_use_id)) {
        calls.get(block.tool_use_id).result = { text: resultText(block), isError: Boolean(block.is_error) }
      }
    }
  }
  return [...calls.values()]
}

// memoryCalls are the agent's own runs of a memory command: an engram command through Bash, or a memory skill.
export function memoryCalls(rows) {
  return toolCalls(rows).filter((call) => {
    if (call.name === 'Bash') return /(^|[\s;&|(])engram\s/.test(call.input.command ?? '')
    if (call.name === 'Skill') return MEMORY_SKILLS.has(call.input.skill)
    return false
  })
}

// countContent counts the rows whose serialised form holds the text.
export function countContent(rows, needle) {
  return rows.filter((row) => JSON.stringify(row).includes(needle)).length
}

export function finalReply(rows) {
  for (let i = rows.length - 1; i >= 0; i -= 1) {
    if (rows[i].type !== 'assistant') continue
    const text = blocks(rows[i]).filter((block) => block.type === 'text').map((block) => block.text).join('\n')
    if (text !== '') return text
  }
  return ''
}
