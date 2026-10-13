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

export function finalReply(rows) {
  for (let i = rows.length - 1; i >= 0; i -= 1) {
    if (rows[i].type !== 'assistant') continue
    const text = blocks(rows[i]).filter((block) => block.type === 'text').map((block) => block.text).join('\n')
    if (text !== '') return text
  }
  return ''
}

// MARKER opens every block the System delivers (793e2b: the System's blocks are identified by their fixed marker line).
export const MARKER = '[engram memory]'

function attachmentText(row) {
  const content = row?.attachment?.content
  if (Array.isArray(content)) return content.filter((part) => typeof part === 'string').join('\n')
  return typeof content === 'string' ? content : ''
}

// systemBlocks are the rows the System added to an agent's context: the hook_additional_context attachment rows
// (SP4) that open with the marker. { index, text } per row, in order.
export function systemBlocks(rows) {
  const blocks = []
  rows.forEach((row, index) => {
    if (row.type !== 'attachment' || row.attachment?.type !== 'hook_additional_context') return
    const text = attachmentText(row)
    if (text.startsWith(MARKER)) blocks.push({ index, text })
  })
  return blocks
}

// deliveries are the System's blocks that carry the text (a note's sentinel).
export function deliveries(rows, needle) {
  return systemBlocks(rows).filter((block) => block.text.includes(needle))
}

// retrievals are the agent's own memory commands whose results show the text.
export function retrievals(rows, needle) {
  return memoryCalls(rows).filter((call) => call.result?.text.includes(needle))
}

// precedesResponse is true when no assistant row sits between the row that caused a block (the nearest earlier user
// row: the prompt, a tool result, a task prompt) and the block: the agent reads it before its next response or tool call.
export function precedesResponse(rows, index) {
  for (let i = index - 1; i >= 0; i -= 1) {
    if (rows[i].type === 'assistant') return false
    if (rows[i].type === 'user') return true
  }
  return false
}

// resultIndex is the index of the user row that carries the result of a tool call, or -1.
export function resultIndex(rows, toolUseId) {
  return rows.findIndex((row) => blocks(row).some((block) => block.type === 'tool_result' && block.tool_use_id === toolUseId))
}

// noResponseBetween: no assistant row sits between the two indexes, so that what sits at `to` is read by the agent
// before its next response or tool call, as what sits at `from` is.
export function noResponseBetween(rows, from, to) {
  return rows.slice(from + 1, to).every((row) => row.type !== 'assistant')
}

// declarationIndex is the row of the declaration: the last assistant row that says anything (a tool call alone says
// nothing), or -1. The agent's declaration is its closing statement of a turn (793a2n1a).
export function declarationIndex(rows) {
  for (let i = rows.length - 1; i >= 0; i -= 1) {
    if (rows[i].type === 'assistant' && blocks(rows[i]).some((block) => block.type === 'text' && block.text.trim() !== '')) return i
  }
  return -1
}

// precedesFirstResponse is true when no assistant row sits anywhere before the block: nothing the agent wrote came
// first, so the block precedes its first response or tool call (793a2n1i).
export function precedesFirstResponse(rows, index) {
  return rows.slice(0, index).every((row) => row.type !== 'assistant')
}

// dispatchIndexAfter is the row at which the orchestrator routes the unit once the route call returned (793a2n1j): its
// next dispatch (an Agent or Task use) after that result, else its declaration if that comes later than the result,
// else Infinity (it did neither, so nothing the block could have preceded).
export function dispatchIndexAfter(rows, toolUseId) {
  const result = resultIndex(rows, toolUseId)
  const dispatch = rows.findIndex((row, i) => i > result && blocks(row).some((block) => block.type === 'tool_use' && (block.name === 'Agent' || block.name === 'Task')))
  if (dispatch >= 0) return dispatch
  const declared = declarationIndex(rows)
  return declared > result ? declared : Infinity
}

// agentCalls are the dispatches of a subagent in the rows: the Agent (formerly Task) tool uses.
export function agentCalls(rows) {
  return toolCalls(rows).filter((call) => call.name === 'Agent' || call.name === 'Task')
}

// promptRows are the rows that open a turn from a given origin, by the origin fields Claude Code writes on the row.
export function promptRows(rows, kind) {
  return rows.flatMap((row, index) => {
    if (row.type !== 'user' || typeof row.message?.content !== 'string') return []
    const origin = JSON.stringify([row.origin, row.turnOrigin, row.promptSource])
    return kind === undefined || origin.includes(kind) ? [{ index, row }] : []
  })
}

// SKIPPED is the fixed phrase of the warning the System shows when automatic memory is skipped (793a4b).
export const SKIPPED = /automatic memory was skipped/i

// statusTexts are the strings the mod passed to $.ui.status, in order, from the stream's system messages (SP9).
export function statusTexts(messages) {
  return messages.filter((message) => message.type === 'system' && message.subtype === 'ui_status').map((message) => String(message.text ?? ''))
}

// rowsMentioning are the rows (any kind) whose serialised form matches the pattern.
export function rowsMentioning(rows, pattern) {
  return rows.filter((row) => pattern.test(JSON.stringify(row)))
}

// typedRows are the user rows whose text is exactly the typed prompt (a block of context is a row of its own).
export function typedRows(rows, typed) {
  return rows.filter((row) => {
    if (row.type !== 'user') return false
    const content = row.message?.content
    if (typeof content === 'string') return content === typed
    return Array.isArray(content) && content.length > 0 && content.every((block) => block.type === 'text') && content.map((block) => block.text).join('\n') === typed
  })
}
