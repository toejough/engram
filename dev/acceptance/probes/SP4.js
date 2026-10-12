import fs from 'node:fs'
import path from 'node:path'

import { readRows } from '../lib/transcript.js'
import { rowsContaining } from './lib.js'

export const claim = 'Added context appears in the transcript as a separate, attributable row (prompt.submit context, classic.PostToolUse and classic.SubagentStart additionalContext); also settles SP5 (subagent start context, dispatch prompt, transcript at stop) and SP6 (main transcript path at session start), which share this session'

const TOKENS = { prompt: 'CTX-PROMPT-7f3a', tool: 'CTX-TOOL-9c21', sub: 'CTX-SUB-b84e' }

function jsonlFiles(dir) {
  if (!fs.existsSync(dir)) return []
  return fs.readdirSync(dir, { withFileTypes: true }).flatMap((entry) => (entry.isDirectory() ? jsonlFiles(path.join(dir, entry.name)) : entry.name.endsWith('.jsonl') ? [path.join(dir, entry.name)] : []))
}

export async function run(h) {
  const scenario = h.scenario()
  h.recorder(scenario, { context: { promptSubmit: TOKENS.prompt, postToolUse: TOKENS.tool, subagentStart: TOKENS.sub } })
  const session = scenario.startSession()
  const turn = await session.send(
    'Do these in order. First run the shell command `echo probe-ran` with the Bash tool. Then dispatch a subagent with the Agent tool (type general-purpose) whose task is: Reply with exactly the word PARROT. Then reply DONE.',
  )
  await h.settle(1500)
  const events = h.events(scenario)
  const projects = path.join(scenario.home, '.claude', 'projects')
  const files = jsonlFiles(projects)
  const mainFile = session.transcriptPath()
  const subFiles = files.filter((file) => file !== mainFile)
  const where = (file) => Object.fromEntries(Object.entries(TOKENS).map(([name, token]) => [name, rowsContaining(readRows(file), token).map(({ index, type, attachmentType, role }) => ({ index, type, attachmentType, role }))]))
  const start = events.find((event) => event.kind === 'classic.SessionStart')
  const stop = events.find((event) => event.kind === 'classic.SubagentStop')
  const spawn = events.find((event) => event.kind === 'agent.spawn')
  const subStart = events.find((event) => event.kind === 'classic.SubagentStart')
  return {
    reply: turn.text.slice(0, 80),
    sp4: { main: where(mainFile), subagentFiles: subFiles.map((file) => ({ file: path.relative(projects, file), where: where(file) })) },
    sp5: {
      spawnPrompt: spawn?.data.prompt,
      spawnDescription: spawn?.data.description,
      subagentStartKeys: subStart ? Object.keys(subStart.data) : null,
      stop: stop && {
        agent_transcript_path: path.relative(projects, stop.data.agent_transcript_path ?? ''),
        matchesDiscoveredFile: subFiles.includes(stop.data.agent_transcript_path),
        last_assistant_message: stop.data.last_assistant_message,
        linesAtStop: stop.data.transcriptLinesAtStop,
        hasLastMessageAtStop: stop.data.transcriptHasLastMessageAtStop,
        assistantRowsAtStop: stop.data.assistantRowsAtStop,
        assistantRowHoldsLastMessageAtStop: stop.data.assistantRowHoldsLastMessageAtStop,
        rowTypesAfterStop: stop.data.transcriptLinesAtStop ? readRows(stop.data.agent_transcript_path).slice(stop.data.transcriptLinesAtStop).map((row) => row.type) : null,
        linesAtEnd: subFiles.find((file) => file === stop.data.agent_transcript_path) ? readRows(stop.data.agent_transcript_path).length : null,
      },
    },
    sp6: { classicSessionStartTranscriptPath: start && path.relative(projects, start.data.transcript_path), driverTranscriptPath: path.relative(projects, mainFile), same: start?.data.transcript_path === mainFile, source: start?.data.source },
  }
}
