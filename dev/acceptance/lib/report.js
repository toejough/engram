// The report of a run, in the results of 793e2h, with the cost of each scenario beside it.
import { MS_PER_SECOND } from './units.js'
const COST_DECIMALS = 3
const DETAIL_COLUMN = 6
const DETAIL_WIDTH = 60
const HEADERS = ['Scenario', 'Result', 'Retried', 'Sessions', 'Cost (USD)', 'Wall (s)', 'Detail']

function labels(results) {
  const total = {}
  for (const entry of results) total[entry.id] = (total[entry.id] ?? 0) + 1
  const seen = {}
  return results.map((entry) => {
    seen[entry.id] = (seen[entry.id] ?? 0) + 1
    return total[entry.id] > 1 ? `${entry.id} (${seen[entry.id]}/${total[entry.id]})` : entry.id
  })
}

export function formatReport(results) {
  const names = labels(results)
  const rows = results.map((entry, index) => [
    names[index],
    entry.result,
    String(entry.retries),
    entry.usage ? String(entry.usage.sessions) : '-',
    entry.usage ? entry.usage.costUsd.toFixed(COST_DECIMALS) : '-',
    entry.usage ? (entry.usage.wallMs / MS_PER_SECOND).toFixed(1) : '-',
    entry.message ?? '',
  ])
  const widths = HEADERS.map((header, index) =>
    Math.max(
      header.length,
      ...rows.map((row) => (index === DETAIL_COLUMN ? Math.min(row[index].length, DETAIL_WIDTH) : row[index].length)),
    ),
  )
  const line = (cells) =>
    cells
      .map((cell, index) =>
        (cell.length > widths[index] ? `${cell.slice(0, widths[index] - 1)}…` : cell).padEnd(widths[index]),
      )
      .join('  ')
      .trimEnd()
  const tally = {}
  for (const entry of results) tally[entry.result] = (tally[entry.result] ?? 0) + 1
  const total = results.reduce((sum, entry) => sum + (entry.usage?.costUsd ?? 0), 0)
  // the table cuts a detail to fit; a result that is not Green or Pending has its whole first line below the table
  const details = results
    .map((entry, index) => ({ entry, name: names[index] }))
    .filter(({ entry }) => /^(Red|Harness error)/.test(entry.result) && entry.message)
    .map(({ entry, name }) => `${name}: ${entry.message}`)
  return [
    line(HEADERS),
    line(widths.map((width) => '-'.repeat(width))),
    ...rows.map(line),
    ...(details.length > 0 ? ['', ...details] : []),
    '',
    `${results.length} scenario runs: ${
      Object.entries(tally)
        .map(([name, count]) => `${count} ${name}`)
        .join(', ') || 'none'
    }; cost ${total.toFixed(COST_DECIMALS)} USD (the SDK's client-side estimate)`,
  ].join('\n')
}

// passed is true when nothing is Red or a harness error; Pending and Not run are reported, never hidden.
export function passed(results) {
  return results.every((entry) => !/^(Red|Harness error)/.test(entry.result))
}
