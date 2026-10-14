import fs from 'node:fs'
import path from 'node:path'
import { excerpt } from '../lib/units.js'

const SETTLE_MS = 500
const TURNS = 4
const REPEATS = 4
const ANSWER =
  'answer INDEX: understood, I will run targ test and never go test directly because targ wraps the build. '
const CHUNK_CHARS = 80
const OUTPUT_LINES = 4

export const claim =
  'The transcript sources with content not yet captured, and their stripped text, can ' +
  'be derived by the mod from what engram ingest --auto leaves on disk (manifest, chunk ' +
  'files) plus a capture mark the mod writes'

function readJsonl(file) {
  return fs
    .readFileSync(file, 'utf8')
    .split('\n')
    .filter(Boolean)
    .map((line) => JSON.parse(line))
}

export async function run(harness) {
  const scenario = harness.scenario()
  await scenario.setup((world) => world.install())
  const session = scenario.startSession()
  await session.send('Reply with exactly the word: one')
  await session.send('Reply with exactly the word: two')
  await harness.settle(SETTLE_MS)
  const ccTranscript = session.transcriptPath()
  await session.close()

  // a Pi-format session file in the Pi session directory convention of the scenario home
  const piDir = path.join(scenario.home, '.pi', 'agent', 'sessions', 'probe')
  fs.mkdirSync(piDir, { recursive: true })
  const piFile = path.join(fs.realpathSync(piDir), '2026-10-12T00-00-00-000Z_probe.jsonl')
  const piRows = [
    { type: 'session', version: 3, id: 'probe', timestamp: '2026-10-12T00:00:00.000Z', cwd: scenario.workspace },
  ]
  let parent = null
  for (let index = 0; index < TURNS; index += 1) {
    const turns = [
      ['user', `question ${index}: no, use targ, not go test directly, when you run the tests of this repository`],
      ['assistant', ANSWER.replace('INDEX', String(index)).repeat(REPEATS)],
    ]
    for (const [role, text] of turns) {
      const id = `${role[0]}${index}`
      piRows.push({
        type: 'message',
        id,
        parentId: parent,
        timestamp: '2026-10-12T00:00:01.000Z',
        message: { role, content: [{ type: 'text', text }] },
      })
      parent = id
    }
  }
  fs.writeFileSync(piFile, `${piRows.map((row) => JSON.stringify(row)).join('\n')}\n`)

  const chunksDir = path.join(scenario.home, '.local', 'share', 'engram', 'chunks')
  const ingest = (extra = []) => scenario.engram(['ingest', '--auto', ...extra], { cwd: scenario.workspace })
  const ingestOut = await ingest()
  const manifest = () => JSON.parse(fs.readFileSync(path.join(chunksDir, 'manifest.json'), 'utf8'))
  const transcriptsIn = (entries) => Object.keys(entries).filter((source) => source.endsWith('.jsonl'))
  const afterFirst = manifest()
  const chunkFiles = fs.readdirSync(chunksDir).filter((name) => name.endsWith('.jsonl'))
  const chunks = chunkFiles.flatMap((name) => readJsonl(path.join(chunksDir, name)))
  const bySource = (source) =>
    chunks
      .filter((chunk) => chunk.source === source)
      .map((chunk) => ({ anchor: chunk.anchor, text: excerpt(chunk.text, CHUNK_CHARS) }))

  // the mod's own mark: source -> file_hash at capture time
  const marks = {}
  const notCaptured = (entries) =>
    transcriptsIn(entries).filter((source) => marks[source] !== entries[source].file_hash)
  const first = notCaptured(afterFirst)
  for (const source of first) marks[source] = afterFirst[source].file_hash
  const second = notCaptured(manifest())

  // the Claude Code session grows by a turn: appended rows, then the next ingest
  const laterRows = [
    { type: 'user', message: { role: 'user', content: 'a later turn in the same session' }, uuid: 'x' },
    {
      type: 'assistant',
      message: { role: 'assistant', content: [{ type: 'text', text: 'a later answer' }] },
      uuid: 'y',
    },
  ].map((row) => ({ ...row, timestamp: new Date().toISOString() }))
  fs.appendFileSync(ccTranscript, laterRows.map((row) => `${JSON.stringify(row)}\n`).join(''))
  await ingest()
  const third = notCaptured(manifest())

  return {
    ingestOutput: ingestOut.trim().split('\n').slice(-OUTPUT_LINES),
    manifestFields: Object.values(afterFirst)
      .slice(0, 1)
      .map((entry) => Object.keys(entry)),
    transcriptSourcesInManifest: transcriptsIn(afterFirst).map((source) => path.relative(scenario.home, source)),
    ccSourceChunks: bySource(ccTranscript),
    piSourceChunks: bySource(piFile),
    notCapturedAtFirst: first.length,
    notCapturedAfterMarking: second.length,
    notCapturedAfterTheSessionGrew: third.map((source) => path.relative(scenario.home, source)),
    newAnchorsAfterGrowth: bySource(ccTranscript).length,
  }
}
