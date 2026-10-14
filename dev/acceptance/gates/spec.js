// The spec-conformance gate (793e8q): the requirements of the change's technical spec against the mod's tests, the Go
// tests and the product spec, wherever the specs are (the change directory, or the canonical files after promotion:
// see lib/extract.js). Every requirement has a `Supports` and a `Verified by` line; every product scenario named by a
// `Supports` line exists, and every product scenario is supported; every contract test a `Verified by` line names
// exists and, for the mod's tests, passes. A mod test is named by its file and the start of its name ("…" stands for
// any text); a Go test by its file and its exact function name. The security table (mitigation, requirement, tests)
// is held to the same rule: its requirements exist and carry a `Security` line, and its tests exist by exact name.
import fs from 'node:fs'
import path from 'node:path'

const REQUIREMENT = /^Requirement: (?:\[)?(793d\d+[a-z]\d+[a-z0-9]*) /
const ANY_HEADING = /^#{1,6} /

// parseSpec reads every requirement of the text: { id, supports: [scenario IDs], verifiedBy: text|null, security:
// text|null }. A requirement ends at the next heading that is not a scenarios heading under it.
export function parseSpec(text) {
  const requirements = []
  let current = null
  for (const line of text.split('\n')) {
    const found = REQUIREMENT.exec(line)
    if (found) {
      current = { id: found[1], supports: null, verifiedBy: null, security: null }
      requirements.push(current)
    } else if (current && line.startsWith('Supports:')) {
      current.supports = [...line.matchAll(/\[(793a[0-9a-z]+) /g)].map((match) => match[1])
    } else if (current && line.startsWith('Verified by:')) {
      current.verifiedBy = line.slice('Verified by:'.length).trim()
    } else if (current && line.startsWith('Security:')) {
      current.security = line.slice('Security:'.length).trim()
    } else if (current && ANY_HEADING.test(line) && !/ Scenarios\s*$/.test(line)) {
      current = null
    }
  }
  return requirements
}

const MOD_REF = /`([\w-]+\.test\.ts)` "([^"]+)"/g
const GO_REF = /`([\w/.-]+_test\.go)`: ([^;`]+)/g
const GO_NAME = /\bTest\w+/g

// references lists the contract tests a `Verified by` line names: { kind: 'mod', file, name } and { kind: 'go', file,
// name }; a Go name is the exact name of the test function, never the start of a name.
export function references(line) {
  const refs = []
  for (const match of line.matchAll(MOD_REF)) refs.push({ kind: 'mod', file: match[1], name: match[2] })
  for (const match of line.matchAll(GO_REF))
    refs.push(...[...match[2].matchAll(GO_NAME)].map((name) => ({ kind: 'go', file: match[1], name: name[0] })))
  return refs
}

// matchesName: a cited name matches a test when its pieces, split at "…", appear in the test's own name in order, the
// first at the start.
export function matchesName(cited, testName) {
  const cut = testName.lastIndexOf(SEPARATOR)
  const own = cut < 0 ? testName : testName.slice(cut + SEPARATOR.length)
  const pieces = cited.split('…').map((piece) => piece.trim())
  let from = 0
  for (const [index, piece] of pieces.entries()) {
    if (piece === '') continue
    const found = own.indexOf(piece, from)
    if (found < 0 || (index === 0 && found !== 0)) return false
    from = found + piece.length
  }
  return true
}

const SEPARATOR = ' > '
const IDENTIFIER = /^[A-Za-z_]\w*(?:\.[A-Za-z_]\w*)?$/

// realizations lists, per `Realization:` line of the specs, the files and the names it gives in code spans: { section,
// files: [...paths], names: [...identifiers] }. The sentence ends at the first full stop that starts a new one.
export function realizations(text) {
  const found = []
  let section = ''
  for (const line of text.split('\n')) {
    const head = /^#{2,4} (793d\d+[a-z]?\d*) /.exec(line)
    if (head) section = head[1]
    const marker = line.indexOf('Realization:')
    if (marker < 0) continue
    const sentence = line.slice(marker + 'Realization:'.length).split(/\.\s+(?=[A-Z])/)[0]
    const spans = [...sentence.matchAll(/`([^`]+)`/g)].map((match) => match[1])
    found.push({
      section,
      files: spans.filter((span) => /\.(?:js|go)$/.test(span)),
      names: spans.filter((span) => IDENTIFIER.test(span) && !/\.(?:js|go)$/.test(span)),
    })
  }
  return found
}

// checkRealizations: every file a Realization line names exists, and every name is declared in the files it names (or,
// when it names none, in a file of the mod's hooks). `read(path)` returns the text of a repository path or null.
export function checkRealizations(list, { read, hooks }) {
  const problems = []
  for (const entry of list) {
    const texts = []
    for (const file of entry.files) {
      const path_ = file.startsWith('internal/') ? file : `agent-instructions/mods/engram-memory/${file}`
      const text = read(path_)
      if (text === null) problems.push(`${entry.section}: Realization names ${file}, which does not exist`)
      else texts.push(text)
    }
    const searched =
      texts.length > 0 && entry.files.every((file) => file.startsWith('internal/')) ? texts : [...texts, ...hooks]
    for (const name of entry.names) {
      const word = new RegExp(`\\b${name.split('.').at(-1)}\\b`)
      if (!searched.some((text) => word.test(text)))
        problems.push(`${entry.section}: Realization names ${name}, which no file it names declares`)
    }
  }
  return problems
}

const hasTest = (source, name) => new RegExp(`^func ${name}\\(`, 'm').test(source)

const SECURITY_COLUMNS = 3
const SECURITY_HEADER = /^\|\s*Mitigation\s*\|\s*Requirement\s*\|\s*Tests\s*\|/
const REQUIREMENT_LINK = /\[(793d\d+[a-z]\d+[a-z0-9]*) /g

// securityRows reads the rows of the security table (header Mitigation | Requirement | Tests): { mitigation,
// requirements: [IDs linked], tests: [exact Go test names] }.
export function securityRows(text) {
  const rows = []
  let inTable = false
  for (const line of text.split('\n')) {
    if (SECURITY_HEADER.test(line)) {
      inTable = true
      continue
    }
    if (!inTable) continue
    if (!line.startsWith('|')) {
      inTable = false
      continue
    }
    const cells = line.split('|').slice(1, -1)
    if (cells.length < SECURITY_COLUMNS || /^\s*-+\s*$/.test(cells[0])) continue
    rows.push({
      mitigation: cells[0].trim(),
      requirements: [...cells[1].matchAll(REQUIREMENT_LINK)].map((match) => match[1]),
      tests: [...cells[2].matchAll(GO_NAME)].map((match) => match[0]),
    })
  }
  return rows
}

// checkSecurity: every row names a requirement and a test, every requirement exists and has a `Security` line, and
// every test is a function of some Go test file by its exact name; and every `Security` line links requirements that
// exist.
export function checkSecurity(rows, { requirements, goSources }) {
  const problems = []
  const byId = new Map(requirements.map((requirement) => [requirement.id, requirement]))
  for (const row of rows) {
    if (row.requirements.length === 0) problems.push(`security table: "${row.mitigation}" names no requirement`)
    if (row.tests.length === 0) problems.push(`security table: "${row.mitigation}" names no test`)
    for (const id of row.requirements) {
      if (!byId.has(id)) problems.push(`security table: ${id} is no requirement`)
      else if (byId.get(id).security === null) problems.push(`security table: ${id} has no Security line`)
    }
    for (const name of row.tests)
      if (!goSources.some((source) => hasTest(source, name))) problems.push(`security table: no Go test ${name}`)
  }
  for (const requirement of requirements)
    for (const [, id] of (requirement.security ?? '').matchAll(REQUIREMENT_LINK))
      if (!byId.has(id)) problems.push(`${requirement.id}: its Security line links ${id}, which is no requirement`)
  return problems
}

// check returns the list of problems; none means the specs conform.
export function check({ requirements, productIds, tests, goFiles }) {
  const problems = []
  const supported = new Set()
  for (const requirement of requirements) {
    if (requirement.supports === null || requirement.supports.length === 0)
      problems.push(`${requirement.id}: no Supports line`)
    for (const id of requirement.supports ?? []) {
      supported.add(id)
      if (!productIds.includes(id)) problems.push(`${requirement.id}: supports ${id}, which is no product scenario`)
    }
    if (requirement.verifiedBy === null) {
      problems.push(`${requirement.id}: no Verified by line`)
      continue
    }
    const refs = references(requirement.verifiedBy)
    if (refs.length === 0) problems.push(`${requirement.id}: its Verified by line names no contract test`)
    for (const ref of refs) problems.push(...checkReference(requirement.id, ref, { tests, goFiles }))
  }
  for (const id of productIds)
    if (!supported.has(id)) problems.push(`${id}: no requirement supports this product scenario`)
  return problems
}

function checkReference(owner, ref, { tests, goFiles }) {
  if (ref.kind === 'go') {
    const text = goFiles[ref.file]
    if (text === undefined) return [`${owner}: ${ref.file} does not exist`]
    return hasTest(text, ref.name) ? [] : [`${owner}: ${ref.file} has no test ${ref.name}`]
  }
  const matching = tests.filter((test) => test.file === ref.file && matchesName(ref.name, test.name))
  if (matching.length === 0) return [`${owner}: ${ref.file} has no test named "${ref.name}"`]
  return matching.filter((test) => !test.ok).map((test) => `${owner}: ${ref.file} "${test.name}" fails`)
}

export function loadGoFiles(repo, requirements) {
  const files = {}
  for (const requirement of requirements) {
    for (const ref of references(requirement.verifiedBy ?? '')) {
      if (ref.kind !== 'go' || ref.file in files) continue
      const full = path.join(repo, ref.file)
      if (fs.existsSync(full)) files[ref.file] = fs.readFileSync(full, 'utf8')
    }
  }
  return files
}

// loadGoTests: the text of every Go test file under internal/, for the names the security table cites.
export function loadGoTests(repo) {
  const sources = []
  const visit = (directory) => {
    for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
      const full = path.join(directory, entry.name)
      if (entry.isDirectory()) visit(full)
      else if (entry.name.endsWith('_test.go')) sources.push(fs.readFileSync(full, 'utf8'))
    }
  }
  visit(path.join(repo, 'internal'))
  return sources
}
