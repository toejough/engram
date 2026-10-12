// The arguments of run.js: an optional scenario ID prefix and --dry-run. Anything else is an error, so that a
// mistyped flag never starts a paid live run.
export function parseArgs(argv) {
  const parsed = { filter: '', dryRun: false }
  let filters = 0
  for (const arg of argv) {
    if (arg === '--dry-run') parsed.dryRun = true
    else if (arg.startsWith('--')) throw new Error(`unknown flag ${arg}`)
    else {
      filters += 1
      parsed.filter = arg
    }
  }
  if (filters > 1) throw new Error('give at most one scenario ID prefix')
  return parsed
}
