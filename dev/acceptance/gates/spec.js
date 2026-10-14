// The spec-conformance gate (793e8q): the component and flow specs of the change (793d21 onward) against the mod's
// tests and the product spec. Every requirement has a `Supports` and a `Verified by` line; every product scenario named
// by a `Supports` line exists, and every product scenario is supported; every contract test a `Verified by` line names
// exists and, for the mod's tests, passes. A mod test is named by its file and the start of its name ("…" stands for
// any text); a Go test by its file and the number-and-letter suffix of its name.
import fs from 'node:fs'
import path from 'node:path'

const REQUIREMENT = /^Requirement: (793d\d+[a-z]\d+[a-z0-9]*) /
const FIRST_COMPONENT = 22
const FIRST_SECTION = /^## 793d(\d+) /

// parseSpec reads the requirements of the sections 793d22 and later: { id, supports: [scenario IDs], verifiedBy:
// text|null }.
export function parseSpec(text) {
  const requirements = []
  let inRange = false
  let current = null
  for (const line of text.split('\n')) {
    const section = FIRST_SECTION.exec(line)
    if (section) {
      inRange = Number(section[1]) >= FIRST_COMPONENT
      current = null
    }
    if (!inRange) continue
    const found = REQUIREMENT.exec(line)
    if (found) {
      current = { id: found[1], supports: null, verifiedBy: null }
      requirements.push(current)
    } else if (current && line.startsWith('Supports:')) {
      current.supports = [...line.matchAll(/\[(793a[0-9a-z]+) /g)].map((match) => match[1])
    } else if (current && line.startsWith('Verified by:')) {
      current.verifiedBy = line.slice('Verified by:'.length).trim()
    }
  }
  return requirements
}

const MOD_REF = /`([\w-]+\.test\.ts)` "([^"]+)"/g
const GO_REF = /`([\w/.-]+_test\.go)`: ([^;`]+)/g

// references lists the contract tests a `Verified by` line names: { kind: 'mod', file, name } and { kind: 'go', file,
// name } (a Go name is the prefix "Test793e8e1a" for each suffix of "1a, 1b and 1c").
export function references(line) {
  const refs = []
  for (const match of line.matchAll(MOD_REF)) refs.push({ kind: 'mod', file: match[1], name: match[2] })
  for (const match of line.matchAll(GO_REF)) {
    const names = match[2]
      .replace(/\s*\(.*?\)/g, '')
      .split(/,\s*|\s+and\s+/)
      .map((name) => name.trim())
      .filter(Boolean)
    const stem = names[0].replace(/\d+[a-z]$/, '')
    refs.push(
      ...names.map((name) => ({ kind: 'go', file: match[1], name: /^Test/.test(name) ? name : `${stem}${name}` })),
    )
  }
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

// the reading guide states how many contract tests the mod has; the number must be the number that ran
export function statedTestCount(text) {
  const match = /\((?:run by `claude plugin test`; )?(\d+) tests\)/.exec(text)
  return match ? Number(match[1]) : null
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
    return new RegExp(`func ${ref.name}\\w*\\(`).test(text) ? [] : [`${owner}: ${ref.file} has no test ${ref.name}`]
  }
  const matching = tests.filter((test) => test.file === ref.file && matchesName(ref.name, test.name))
  if (matching.length === 0) return [`${owner}: ${ref.file} has no test named "${ref.name}"`]
  return matching.filter((test) => !test.ok).map((test) => `${owner}: ${ref.file} "${test.name}" fails`)
}

export function checkTestCount(stated, actual) {
  if (stated === null) return []
  return stated === actual ? [] : [`the reading guide states ${stated} contract tests and the mod has ${actual}`]
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
