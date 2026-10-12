import fs from 'node:fs'
import path from 'node:path'

export const claim = 'The transcript sources with content not yet captured, and their stripped text, can be derived by the mod from what engram ingest --auto leaves on disk (manifest, chunk files) plus a capture mark the mod writes'

function readJsonl(file) {
  return fs.readFileSync(file, 'utf8').split('\n').filter(Boolean).map((line) => JSON.parse(line))
}

export async function run(h) {
  const scenario = h.scenario()
  await scenario.setup((s) => s.install())
  const session = scenario.startSession()
  await session.send('Reply with exactly the word: one')
  await session.send('Reply with exactly the word: two')
  await h.settle(500)
  const ccTranscript = session.transcriptPath()
  await session.close()

  // a Pi-format session file in the Pi session directory convention of the scenario home
  const piDir = path.join(scenario.home, '.pi', 'agent', 'sessions', 'probe')
  fs.mkdirSync(piDir, { recursive: true })
  const piFile = path.join(fs.realpathSync(piDir), '2026-10-12T00-00-00-000Z_probe.jsonl')
  const piRows = [{ type: 'session', version: 3, id: 'probe', timestamp: '2026-10-12T00:00:00.000Z', cwd: scenario.workspace }]
  let parent = null
  for (let i = 0; i < 4; i += 1) {
    const turns = [
      ['user', `question ${i}: no, use targ, not go test directly, when you run the tests of this repository`],
      ['assistant', `answer ${i}: understood, I will run targ test and never go test directly because targ wraps the build. `.repeat(4)],
    ]
    for (const [role, text] of turns) {
      const id = `${role[0]}${i}`
      piRows.push({ type: 'message', id, parentId: parent, timestamp: '2026-10-12T00:00:01.000Z', message: { role, content: [{ type: 'text', text }] } })
      parent = id
    }
  }
  fs.writeFileSync(piFile, `${piRows.map((row) => JSON.stringify(row)).join('\n')}\n`)

  const chunksDir = path.join(scenario.home, '.local', 'share', 'engram', 'chunks')
  const ingest = (extra = []) => scenario.engram(['ingest', '--auto', ...extra], { cwd: scenario.workspace })
  const ingestOut = await ingest()
  const manifest = () => JSON.parse(fs.readFileSync(path.join(chunksDir, 'manifest.json'), 'utf8'))
  const transcriptsIn = (m) => Object.keys(m).filter((source) => source.endsWith('.jsonl'))
  const afterFirst = manifest()
  const chunkFiles = fs.readdirSync(chunksDir).filter((name) => name.endsWith('.jsonl'))
  const chunks = chunkFiles.flatMap((name) => readJsonl(path.join(chunksDir, name)))
  const bySource = (source) => chunks.filter((chunk) => chunk.source === source).map((chunk) => ({ anchor: chunk.anchor, text: chunk.text.slice(0, 80) }))

  // the mod's own mark: source -> file_hash at capture time
  const marks = {}
  const notCaptured = (m) => transcriptsIn(m).filter((source) => marks[source] !== m[source].file_hash)
  const first = notCaptured(afterFirst)
  for (const source of first) marks[source] = afterFirst[source].file_hash
  const second = notCaptured(manifest())

  // the Claude Code session grows by a turn: appended rows, then the next ingest
  fs.appendFileSync(ccTranscript, `${JSON.stringify({ type: 'user', message: { role: 'user', content: 'a later turn in the same session' }, uuid: 'x', timestamp: new Date().toISOString() })}\n${JSON.stringify({ type: 'assistant', message: { role: 'assistant', content: [{ type: 'text', text: 'a later answer' }] }, uuid: 'y', timestamp: new Date().toISOString() })}\n`)
  await ingest()
  const third = notCaptured(manifest())

  return {
    ingestOutput: ingestOut.trim().split('\n').slice(-4),
    manifestFields: Object.values(afterFirst).slice(0, 1).map((entry) => Object.keys(entry)),
    transcriptSourcesInManifest: transcriptsIn(afterFirst).map((source) => path.relative(scenario.home, source)),
    ccSourceChunks: bySource(ccTranscript),
    piSourceChunks: bySource(piFile),
    notCapturedAtFirst: first.length,
    notCapturedAfterMarking: second.length,
    notCapturedAfterTheSessionGrew: third.map((source) => path.relative(scenario.home, source)),
    newAnchorsAfterGrowth: bySource(ccTranscript).length,
  }
}
