// The arguments of run.js: an optional scenario ID prefix, --dry-run and --runs N (a row run N times for its rate,
// 793b421). Anything else is an error, so that a mistyped flag never starts a paid live run.
const MAX_RUNS = 100

function runsOf(text) {
  const runs = Number(text)
  if (text === undefined || !Number.isInteger(runs) || runs < 1 || runs > MAX_RUNS)
    throw new Error(`--runs takes a whole number from 1 to ${MAX_RUNS}, not ${text === undefined ? 'nothing' : text}`)
  return runs
}

export function parseArgs(argv) {
  const parsed = { filter: '', dryRun: false, runs: 1 }
  let filters = 0
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index]
    if (arg === '--dry-run') parsed.dryRun = true
    else if (arg === '--runs') {
      index += 1
      parsed.runs = runsOf(argv[index])
    } else if (arg.startsWith('--runs=')) parsed.runs = runsOf(arg.slice('--runs='.length))
    else if (arg.startsWith('--')) throw new Error(`unknown flag ${arg}`)
    else {
      filters += 1
      parsed.filter = arg
    }
  }
  if (filters > 1) throw new Error('give at most one scenario ID prefix')
  return parsed
}
