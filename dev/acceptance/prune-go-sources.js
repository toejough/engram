// Run by npm after an install (the `postinstall` script of package.json): removes the Go source files that some npm
// packages ship, so that the repository's Go gates (lint, declaration order, nil analysis, the thin-API check,
// coverage) do not read them as code of this module. eslint reaches `flatted` through its file cache, and `flatted`
// ships a Go port under `golang/`; nothing here builds it.
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

// pruneGoSources deletes every `.go` file under dir (a node_modules directory) and returns their paths.
export function pruneGoSources(dir) {
  const removed = []
  if (!fs.existsSync(dir)) return removed
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name)
    if (entry.isDirectory()) removed.push(...pruneGoSources(full))
    else if (entry.name.endsWith('.go')) {
      fs.rmSync(full)
      removed.push(full)
    }
  }
  return removed
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const removed = pruneGoSources(path.join(path.dirname(fileURLToPath(import.meta.url)), 'node_modules'))
  if (removed.length > 0) console.log(`pruned ${removed.length} Go source file(s) from node_modules`)
}
