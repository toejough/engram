// The install tree (793e8e): what `engram update` leaves of the System and of the memory skills, guidance and shim.
// A snapshot maps each path under the engram-owned root and the skills directory of a home to what it holds: the
// hash of a file, the target of a link, or "dir". Comparing two snapshots is what "unchanged" and "still holds" mean.
import crypto from 'node:crypto'
import fs from 'node:fs'
import path from 'node:path'

const ROOTS = [path.join('.claude', 'engram'), path.join('.claude', 'skills')]

function describe(file, stat) {
  if (stat.isSymbolicLink()) return `link ${fs.readlinkSync(file)}`
  if (stat.isDirectory()) return 'dir'
  return `file ${crypto.createHash('sha256').update(fs.readFileSync(file)).digest('hex')}`
}

function walk(home, rel, entries) {
  const file = path.join(home, rel)
  const stat = fs.lstatSync(file, { throwIfNoEntry: false })
  if (!stat) return
  entries.set(rel, describe(file, stat))
  if (!stat.isDirectory()) return
  for (const name of fs.readdirSync(file).sort()) walk(home, path.join(rel, name), entries)
}

export function snapshotTree(home) {
  const entries = new Map()
  for (const root of ROOTS) walk(home, root, entries)
  return entries
}

// changes lists how `after` differs from `before`: paths added, removed and changed, each as one line.
export function changes(before, after) {
  const lines = []
  for (const [rel, what] of before) {
    if (!after.has(rel)) lines.push(`removed ${rel}`)
    else if (after.get(rel) !== what) lines.push(`changed ${rel}`)
  }
  for (const rel of after.keys()) if (!before.has(rel)) lines.push(`added ${rel}`)
  return lines
}

// lost lists what `before` held that `after` no longer holds as it was; additions are not losses.
export function lost(before, after) {
  return changes(before, after).filter((line) => !line.startsWith('added '))
}
