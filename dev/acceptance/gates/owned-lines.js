// The minimality gate (793e8m): the owned lines of each JavaScript file of the mod against a budget table. A line is
// owned when it is neither blank nor a comment; the count is the same for every file so that a budget is comparable.
import fs from 'node:fs'
import path from 'node:path'

// ownedLines counts the lines of a JavaScript text that are neither blank nor only a comment (`//` lines and the lines
// of a `/* ... */` block).
export function ownedLines(text) {
  let inBlock = false
  let owned = 0
  for (const raw of text.split('\n')) {
    const line = raw.trim()
    if (inBlock) {
      if (line.includes('*/')) inBlock = false
      continue
    }
    if (line === '' || line.startsWith('//')) continue
    if (line.startsWith('/*')) {
      inBlock = !line.includes('*/')
      continue
    }
    owned += 1
  }
  return owned
}

const FILE_WIDTH = 16
const OWNED_WIDTH = 11
const BUDGET_WIDTH = 6
const RELATIVE_IMPORT = /^\s*(?:import\b[^'"]*|export\b[^'"]*from\s*)['"]([^'"]+)['"]/gm

// externalImports lists the module specifiers a text imports that are neither relative nor a node built-in: the mod has
// no dependency (it runs in the hook runtime with nothing installed).
export function externalImports(text) {
  return [...text.matchAll(RELATIVE_IMPORT)]
    .map((match) => match[1])
    .filter((name) => !name.startsWith('.') && !name.startsWith('node:'))
}

// jsFilesOf lists the JavaScript files of the mod under dir, as paths relative to it (`hooks/register.js`): the mod's
// tests are TypeScript and are not the mod.
export function jsFilesOf(dir, relative = '') {
  const found = []
  for (const entry of fs.readdirSync(path.join(dir, relative), { withFileTypes: true })) {
    const name = path.join(relative, entry.name)
    if (entry.isDirectory() && entry.name !== 'node_modules') found.push(...jsFilesOf(dir, name))
    else if (entry.name.endsWith('.js')) found.push(name)
  }
  return found.sort()
}

// measure reads the JavaScript files of the mod under dir and returns { relative path: ownedLines }.
export function measure(dir) {
  const sizeOf = (file) => ownedLines(fs.readFileSync(path.join(dir, file), 'utf8'))
  return Object.fromEntries(jsFilesOf(dir).map((file) => [file, sizeOf(file)]))
}

// compare holds the measured sizes against the budget table: { rows, total, over, unbudgeted, imports }. A file with no
// row is unbudgeted (it fails, so that a new file is given a budget on purpose); a row with no file is ignored.
export function compare(sizes, table) {
  const rows = Object.entries(sizes).map(([file, owned]) => {
    const budget = table.files[file]?.budget
    return { file, owned, budget: budget ?? null, excess: budget === undefined ? null : owned - budget }
  })
  const total = rows.reduce((sum, row) => sum + row.owned, 0)
  return {
    rows,
    total,
    totalBudget: table.total,
    over: rows.filter((row) => row.excess !== null && row.excess > 0),
    unbudgeted: rows.filter((row) => row.budget === null),
    totalExcess: total - table.total,
  }
}

// goal is the size the owned JavaScript lines a Go component replaces allow at the ratio (793b369).
export const goal = (row, ratio) => Math.round(row.js * ratio)

// goTable is the budget table of the Go components: a component may not grow past its ceiling, the size it has reached;
// the goal, the owned JavaScript lines it replaces times the ratio, is what the ceiling is brought down to (793b369).
export function goTable(goSection) {
  const files = Object.fromEntries(
    Object.entries(goSection.components).map(([name, row]) => [
      name,
      { budget: row.ceiling ?? goal(row, goSection.ratio) },
    ]),
  )
  return { total: Object.values(files).reduce((sum, row) => sum + row.budget, 0), files }
}

// measureGo reads the non-test Go files of dir and returns { component: ownedLines }; a file that belongs to no
// component and is not plumbing is listed under its own name, so that the comparison finds it unbudgeted.
export function measureGo(dir, goSection) {
  const owner = new Map(
    Object.entries(goSection.components).flatMap(([name, row]) => row.files.map((file) => [file, name])),
  )
  const sizes = Object.fromEntries(Object.keys(goSection.components).map((name) => [name, 0]))
  for (const file of fs
    .readdirSync(dir)
    .filter((entry) => entry.endsWith('.go') && !entry.endsWith('_test.go'))
    .sort()) {
    if (goSection.plumbing.includes(file)) continue
    const owned = ownedLines(fs.readFileSync(path.join(dir, file), 'utf8'))
    const name = owner.get(file) ?? file
    sizes[name] = (sizes[name] ?? 0) + owned
  }
  return sizes
}

export function formatTable(result) {
  const lines = ['file             owned lines   budget   over']
  for (const row of result.rows) {
    const over = row.excess === null ? 'no budget' : String(Math.max(row.excess, 0))
    lines.push(
      `${row.file.padEnd(FILE_WIDTH)} ${String(row.owned).padStart(OWNED_WIDTH)}   ` +
        `${String(row.budget ?? '-').padStart(BUDGET_WIDTH)}   ${over}`,
    )
  }
  lines.push(
    `${'total'.padEnd(FILE_WIDTH)} ${String(result.total).padStart(OWNED_WIDTH)}   ` +
      `${String(result.totalBudget).padStart(BUDGET_WIDTH)}   ${Math.max(result.totalExcess, 0)}`,
  )
  return lines.join('\n')
}

export function passes(result, imports) {
  return result.over.length === 0 && result.unbudgeted.length === 0 && result.totalExcess <= 0 && imports.length === 0
}
