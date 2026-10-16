// The rate bar (793b421): a row that depends on a model's judgement is not Green or Red once, it is Green at a rate.
// It passes when at least 9 of 10 of its runs are Green and every Red is attributed from the activity line the System
// showed in that run: a warning on the line says the System failed (the lookup was cut, the program crashed), none
// says the judgement missed. `targ acceptance --runs N` runs a row N times and prints this.
import { MS_PER_SECOND } from './units.js'

export const RATE_BAR = 0.9
const DENIAL = /Permission for this tool use was denied|permission denied/i
const WARNING = /^Automatic memory skipped:/
const COST_DECIMALS = 3
const RATE_DECIMALS = 2
const HEADERS = ['Scenario', 'Runs', 'Green', 'Red', 'Harness error', 'Not run', 'Rate', 'Bar 9 of 10']

// attribution says why a run was Red, from the lines its sessions showed.
export function attribution(entry) {
  if (entry.activity === undefined) return { kind: 'unattributed', detail: 'the activity line was not kept' }
  // a refused tool call is the harness's condition, not a judgement of the System
  if (DENIAL.test(`${entry.result} ${entry.message ?? ''}`))
    return { kind: 'Harness error', detail: 'a tool use was denied by the permission rules' }
  const warning = entry.activity.find((text) => WARNING.test(text))
  return warning
    ? { kind: 'System failure', detail: warning }
    : { kind: 'System miss', detail: 'no warning on the activity line: the judgement missed' }
}

// summarize reads the results of N runs of the same rows, one array per run, as one row per scenario.
export function summarize(rounds) {
  const byId = new Map()
  rounds.forEach((results, round) => {
    for (const entry of results) {
      const row = byId.get(entry.id) ?? { id: entry.id, runs: 0, green: 0, red: 0, harness: 0, notRun: 0, reds: [] }
      row.runs += 1
      if (entry.result === 'Green') row.green += 1
      else if (entry.result === 'Red') {
        row.red += 1
        row.reds.push({ run: round + 1, entry, ...attribution(entry) })
      } else if (/^Harness error/.test(entry.result)) row.harness += 1
      else row.notRun += 1
      byId.set(entry.id, row)
    }
  })
  return [...byId.values()].map((row) => {
    const ran = row.runs - row.notRun
    const rate = ran > 0 ? row.green / ran : null
    const attributed = row.reds.every((red) => red.kind !== 'unattributed')
    const met = rate >= RATE_BAR && attributed && row.harness === 0
    return { ...row, rate, met: rate === null ? null : met }
  })
}

// ratePassed is true when no row failed the bar; a row that was not run is reported, never hidden.
export function ratePassed(rows) {
  return rows.every((row) => row.met !== false)
}

// formatRates is the table of rates, then each Red with its attribution, then the totals.
export function formatRates(rows, rounds) {
  const cells = rows.map((row) => [
    row.id,
    String(row.runs),
    String(row.green),
    String(row.red),
    String(row.harness),
    String(row.notRun),
    row.rate === null ? '-' : row.rate.toFixed(RATE_DECIMALS),
    { true: 'yes', false: 'no', null: '-' }[row.met],
  ])
  const widths = HEADERS.map((header, index) => Math.max(header.length, ...cells.map((row) => row[index].length)))
  const line = (row) => row.map((cell, index) => cell.padEnd(widths[index])).join('  ').trimEnd()
  const reds = rows.flatMap((row) =>
    row.reds.map(
      (red) => `${row.id} run ${red.run}: ${red.kind}: ${red.detail} (${red.entry.message || 'no message'})`,
    ),
  )
  const cost = rounds.flat().reduce((sum, entry) => sum + (entry.usage?.costUsd ?? 0), 0)
  const wall = rounds.flat().reduce((sum, entry) => sum + (entry.usage?.wallMs ?? 0), 0) / MS_PER_SECOND
  return [
    line(HEADERS),
    line(widths.map((width) => '-'.repeat(width))),
    ...cells.map(line),
    ...(reds.length > 0 ? ['', ...reds] : []),
    '',
    `${rounds.length} runs of ${rows.length} rows; cost ${cost.toFixed(COST_DECIMALS)} USD ` +
      `(the SDK's client-side estimate); ${wall.toFixed(1)} s of session time`,
  ].join('\n')
}
