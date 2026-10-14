// The run-wide, read-only part of the World (793e2b): the engram binary built once to a temporary path, and the
// embedder cache unpacked once so that no scenario pays the unpacking. Nothing is installed (no go install) and
// nothing is written outside the run directory except the Go build cache, which `go build` always uses.
import { execFile } from 'node:child_process'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { promisify } from 'node:util'

import { REPO_ROOT } from './extract.js'

const run = promisify(execFile)

export async function createRunContext({ claudePath, creds }) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'engram-acceptance-'))
  try {
    const binDir = path.join(dir, 'bin')
    fs.mkdirSync(binDir)
    const engramPath = path.join(binDir, 'engram')
    // go build -o to the run directory: never go install, which would write GOBIN.
    await run('go', ['build', '-o', engramPath, './cmd/engram'], { cwd: REPO_ROOT, env: process.env })
    const cacheDir = path.join(dir, 'cache')
    const scratch = path.join(dir, 'prewarm')
    fs.mkdirSync(scratch)
    await run(engramPath, ['query', '--phrase', 'prewarm', '--limit', '1'], {
      env: baseEnv({ home: scratch, cacheDir, binDir }),
    })
    return { dir, binDir, engramPath, cacheDir, creds, claudePath }
  } catch (error) {
    fs.rmSync(dir, { recursive: true, force: true })
    throw error
  }
}

export function disposeRunContext(context) {
  if (context) fs.rmSync(context.dir, { recursive: true, force: true })
}

// baseEnv is the explicit environment of every process the harness starts for a scenario: nothing is inherited from
// the calling session, and every data, cache and config directory is inside the scenario's home.
export function baseEnv({ home, cacheDir, binDir, tmp = path.join(home, 'tmp') }) {
  return {
    PATH: [binDir, '/usr/bin', '/bin', '/usr/sbin', '/sbin', '/opt/homebrew/bin', '/usr/local/bin'].join(
      path.delimiter,
    ),
    HOME: home,
    TMPDIR: tmp,
    LANG: 'en_US.UTF-8',
    XDG_DATA_HOME: path.join(home, '.local', 'share'),
    XDG_CONFIG_HOME: path.join(home, '.config'),
    XDG_CACHE_HOME: cacheDir,
  }
}
