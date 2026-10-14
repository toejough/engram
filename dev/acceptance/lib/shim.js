// Fault shims (793e8d): a program named engram that stands before the real one on the session's PATH and fails in a
// real way instead of a product-side fault switch (793e2i step 6). `error` exits with a message, `hang` lets a lookup
// (engram query) run past every deadline, `delay` makes the commands of the System's background work (every command
// but a query, and but `auto`, which runs the mod's jobs and calls the commands of the work itself) take a number of
// seconds before the real program runs. Every call is logged beside the script.
import fs from 'node:fs'
import path from 'node:path'

export const MODES = ['error', 'hang', 'delay']

function script({ realEngram, dir, mode, seconds }) {
  const real = JSON.stringify(realEngram)
  const log = JSON.stringify(path.join(dir, 'calls.log'))
  const lines = ['#!/bin/sh', `echo "$(date +%s) ${mode} $*" >> ${log}`]
  if (mode === 'error') lines.push('echo "engram: injected failure" >&2', 'exit 70')
  else if (mode === 'hang') lines.push('if [ "$1" = query ]; then exec sleep 3600; fi', `exec ${real} "$@"`)
  else lines.push(`if [ "$1" != query ] && [ "$1" != auto ]; then echo "$(date +%s) slept-start $1" >> ${log}; sleep ${seconds}; fi`, `exec ${real} "$@"`)
  return `${lines.join('\n')}\n`
}

// installShim writes the shim into dir and returns its directory, to put first on PATH.
export function installShim({ dir, realEngram, mode, seconds = 0 }) {
  if (!MODES.includes(mode)) throw new Error(`no shim mode "${mode}"`)
  const shimDir = path.join(dir, 'shim')
  fs.mkdirSync(shimDir, { recursive: true })
  const file = path.join(shimDir, 'engram')
  fs.writeFileSync(file, script({ realEngram, dir: shimDir, mode, seconds }), { mode: 0o755 })
  return shimDir
}

// calls are the lines the shim logged: { at (seconds), what }.
export function shimCalls(shimDir) {
  if (!shimDir) return []
  try {
    return fs.readFileSync(path.join(shimDir, 'calls.log'), 'utf8').split('\n').filter((line) => line !== '').map((line) => {
      const [at, ...rest] = line.split(' ')
      return { at: Number(at), what: rest.join(' ') }
    })
  } catch {
    return []
  }
}
