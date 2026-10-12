import { spawn } from 'node:child_process'
import fs from 'node:fs'
import net from 'node:net'
import path from 'node:path'

export const claim = 'A pending offer can be seeded in an isolated vault through a real path (the registration offers of engram update, or the serve write path)'

function freePort() {
  return new Promise((resolve, reject) => {
    const server = net.createServer()
    server.listen(0, '127.0.0.1', () => {
      const { port } = server.address()
      server.close(() => resolve(port))
    })
    server.on('error', reject)
  })
}

const pendingNotes = (vault) =>
  fs.readdirSync(vault).filter((name) => name.endsWith('.md') && /^pending: true$/m.test(fs.readFileSync(path.join(vault, name), 'utf8')))

export async function run(h) {
  const evidence = {}
  const scenario = h.scenario()
  await scenario.setup((s) => s.install())

  const registered = await scenario.engram(['register-skills', '--accept', 'claude:recall']).catch((error) => error.message)
  evidence.registerSkills = { pendingAfter: pendingNotes(scenario.vault), queryFlag: /pending_offers: true/.test(await scenario.engram(['query', '--phrase', 'x', '--limit', '1'])), output: String(registered).slice(0, 120) }

  const port = await freePort()
  const server = spawn(h.context.engramPath, ['serve', '--addr', `127.0.0.1:${port}`, '--vault', scenario.vault], { env: scenario.env(), stdio: 'ignore' })
  try {
    for (let i = 0; i < 50; i += 1) {
      const up = await fetch(`http://127.0.0.1:${port}/`).then(() => true, () => false)
      if (up) break
      await h.settle(100)
    }
    const childVault = path.join(scenario.dir, 'child-vault')
    const before = pendingNotes(scenario.vault).length
    await scenario.engram(
      ['learn', 'fact', '--slug', 'offered', '--position', 'top', '--source', 'probe', '--vault', childVault, '--situation', 'tagging a release', '--subject', 'releases', '--predicate', 'are tagged from', '--object', 'main'],
      { extraEnv: { ENGRAM_PARENT: `http://127.0.0.1:${port}` } },
    ).catch((error) => { evidence.serveLearnError = error.message.slice(0, 300) })
    const after = pendingNotes(scenario.vault)
    evidence.serve = { pendingBefore: before, pendingAfter: after.length, newest: after.at(-1) }
  } finally {
    server.kill()
  }
  return evidence
}
