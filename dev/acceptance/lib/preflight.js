// The programs the live suites need (793e2d). A missing one fails with its name; nothing is skipped.
import { execFileSync } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'

export const MIN_CLAUDE = [2, 1, 287]
const PROGRAMS = ['go', 'git']

export function parseClaudeVersion(text) {
  const match = /(\d+)\.(\d+)\.(\d+)/.exec(text)
  return match ? match.slice(1).map(Number) : null
}

function atLeast(version, minimum) {
  for (let i = 0; i < minimum.length; i += 1) {
    if (version[i] !== minimum[i]) return version[i] > minimum[i]
  }
  return true
}

export function checkPrograms({ which, run }, { needClaude }) {
  for (const name of needClaude ? [...PROGRAMS, 'claude'] : PROGRAMS) {
    if (which(name) === null) throw new Error(`${name} was not found on PATH: install it and run again`)
  }
  if (!needClaude) return null
  const claudePath = which('claude')
  const version = parseClaudeVersion(run(claudePath, ['--version']))
  if (version === null || !atLeast(version, MIN_CLAUDE)) {
    throw new Error(`claude ${version ? version.join('.') : '(unreadable version)'} is older than ${MIN_CLAUDE.join('.')}, the first with mods`)
  }
  return { path: claudePath, version }
}

export function whichOnPath(name, envPath = process.env.PATH ?? '') {
  for (const dir of envPath.split(path.delimiter)) {
    const candidate = path.join(dir, name)
    try {
      fs.accessSync(candidate, fs.constants.X_OK)
      if (fs.statSync(candidate).isFile()) return candidate
    } catch {
      // keep looking
    }
  }
  return null
}

export function preflight({ needClaude }) {
  return checkPrograms({ which: whichOnPath, run: (file, args) => execFileSync(file, args, { encoding: 'utf8' }) }, { needClaude })
}
