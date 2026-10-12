// The report of a run, in the results of 793e2h, with the cost of each scenario beside it.
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
    entry.usage ? entry.usage.costUsd.toFixed(3) : '-',
    entry.usage ? (entry.usage.wallMs / 1000).toFixed(1) : '-',
    entry.message ?? '',
  ])
  const widths = HEADERS.map((header, i) => Math.max(header.length, ...rows.map((row) => row[i].length > 60 && i === 6 ? 60 : row[i].length)))
  const line = (cells) => cells.map((cell, i) => (cell.length > widths[i] ? `${cell.slice(0, widths[i] - 1)}…` : cell).padEnd(widths[i])).join('  ').trimEnd()
  const tally = {}
  for (const entry of results) tally[entry.result] = (tally[entry.result] ?? 0) + 1
  const total = results.reduce((sum, entry) => sum + (entry.usage?.costUsd ?? 0), 0)
  return [
    line(HEADERS),
    line(widths.map((width) => '-'.repeat(width))),
    ...rows.map(line),
    '',
    `${results.length} scenario runs: ${Object.entries(tally).map(([name, count]) => `${count} ${name}`).join(', ') || 'none'}; cost ${total.toFixed(3)} USD (the SDK's client-side estimate)`,
  ].join('\n')
}

// passed is true when nothing is Red or a harness error; Pending and Not run are reported, never hidden.
export function passed(results) {
  return results.every((entry) => !/^(Red|Harness error)/.test(entry.result))
}
