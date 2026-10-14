// Seeding a pending offer through a real path (SP10, 793e2i step 5): `engram serve` on the scenario's vault and a
// child vault's `engram learn` with ENGRAM_PARENT, which lands the child's note as a pending offer. The offer states
// the claim of a seeded note (it is covered) or a claim no note states.
import { spawn } from 'node:child_process'
import fs from 'node:fs'
import net from 'node:net'
import path from 'node:path'
import { setTimeout as sleep } from 'node:timers/promises'

import { sentinelFor } from './vault.js'

const UP_TRIES = 100
const UP_WAIT_MS = 100

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

export function pendingNotes(vault) {
  if (!fs.existsSync(vault)) return []
  return fs
    .readdirSync(vault)
    .filter((name) => name.endsWith('.md') && /^pending: true$/m.test(fs.readFileSync(path.join(vault, name), 'utf8')))
}

async function waitUp(port, server) {
  for (let index = 0; index < UP_TRIES; index += 1) {
    if (server.exitCode !== null) throw new Error('engram serve exited before it answered')
    if (
      await fetch(`http://127.0.0.1:${port}/`).then(
        () => true,
        () => false,
      )
    )
      return
    await sleep(UP_WAIT_MS)
  }
  throw new Error('engram serve did not answer')
}

// seedOffer returns { label, sentinel, situation, covers, basename } of the offer that landed.
export async function seedOffer(scenario, label, covered) {
  const claim = covered ?? {
    situation: `rotating the ${label.toLowerCase()} staging database credentials`,
    object: sentinelFor(label),
  }
  const before = new Set(pendingNotes(scenario.vault))
  const port = await freePort()
  const server = spawn(
    scenario.context.engramPath,
    ['serve', '--addr', `127.0.0.1:${port}`, '--vault', scenario.vault],
    {
      env: scenario.env(),
      stdio: 'ignore',
    },
  )
  try {
    await waitUp(port, server)
    const child = path.join(scenario.dir, `child-vault-${label.toLowerCase()}`)
    await scenario.engram(
      [
        'learn',
        'fact',
        '--slug',
        `offer-${label.toLowerCase()}`,
        '--position',
        'top',
        '--source',
        'acceptance harness',
        '--vault',
        child,
        '--situation',
        claim.situation,
        '--subject',
        claim.situation,
        '--predicate',
        'is recorded as',
        '--object',
        claim.object,
      ],
      { extraEnv: { ENGRAM_PARENT: `http://127.0.0.1:${port}` } },
    )
  } finally {
    server.kill()
  }
  const landed = pendingNotes(scenario.vault).filter((name) => !before.has(name))
  if (landed.length !== 1) throw new Error(`seeding offer "${label}" landed ${landed.length} pending notes, expected 1`)
  const sentinel = /SENTINEL-[A-Za-z0-9]+-[0-9a-f]{6}/.exec(claim.object)[0]
  return { label, sentinel, situation: claim.situation, covers: covered ? true : false, basename: landed[0] }
}
