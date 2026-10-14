// Extracts the Gherkin scenarios of a specification so that the spec is the test source (793e2b).
//
// Where the specs are read from is the spec root (the repository, or the directory ENGRAM_SPEC_ROOT names): the change
// directory of the proof of concept while it exists, and the canonical files docs/specs/product/*.md and
// docs/specs/technical/*.md that promotion (793e8r) writes. The same gates therefore read the specs before and after
// the change directory is deleted.
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

export const REPO_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..', '..')
export const SPEC_ROOT_VARIABLE = 'ENGRAM_SPEC_ROOT'
const CHANGE_DIRECTORY = path.join('docs', 'changes', '793_parallel-memory-agent')
const PRODUCT_DIRECTORY = path.join('docs', 'specs', 'product')
const TECHNICAL_DIRECTORY = path.join('docs', 'specs', 'technical')
const PRODUCT_CHANGE_FILE = '793a_product-spec.md'
const TECHNICAL_CHANGE_FILE = '793d_technical-spec.md'
const PRODUCT_ID_PREFIX = '793a'
const TECHNICAL_SECTION = /^## 793d\d+/m

export const specRoot = (env = process.env) => env[SPEC_ROOT_VARIABLE] ?? REPO_ROOT

// changeDirectory is the directory of the change while it exists, else null.
export function changeDirectory(root = specRoot()) {
  const directory = path.join(root, CHANGE_DIRECTORY)
  return fs.existsSync(directory) ? directory : null
}

function markdownFiles(directory) {
  return fs.existsSync(directory)
    ? fs
        .readdirSync(directory)
        .filter((name) => name.endsWith('.md'))
        .sort()
        .map((name) => path.join(directory, name))
    : []
}

const present = (file) => (fs.existsSync(file) ? [file] : [])

// productSpecFiles: the product spec of the change while it exists, then every canonical product file.
export function productSpecFiles(root = specRoot()) {
  return [
    ...present(path.join(root, CHANGE_DIRECTORY, PRODUCT_CHANGE_FILE)),
    ...markdownFiles(path.join(root, PRODUCT_DIRECTORY)),
  ]
}

// technicalSpecFiles: the technical spec of the change while it exists, then the canonical component and flow files
// that hold sections of the change (a section `## 793d<number> ...`); the other files are other changes' specs.
export function technicalSpecFiles(root = specRoot()) {
  const canonical = markdownFiles(path.join(root, TECHNICAL_DIRECTORY)).filter((file) =>
    TECHNICAL_SECTION.test(fs.readFileSync(file, 'utf8')),
  )
  return [...present(path.join(root, CHANGE_DIRECTORY, TECHNICAL_CHANGE_FILE)), ...canonical]
}

// productScenarios: the scenarios of this change's product spec, wherever it is (the IDs of other changes' scenarios
// in the canonical files are not this suite's), each ID once.
export function productScenarios(root = specRoot()) {
  const seen = new Set()
  return extractScenarios(productSpecFiles(root), { idPrefix: PRODUCT_ID_PREFIX }).filter(
    (scenario) => !seen.has(scenario.id) && seen.add(scenario.id),
  )
}

// technicalText: the text of every technical file of the change, in one string.
export function technicalText(root = specRoot()) {
  return technicalSpecFiles(root)
    .map((file) => fs.readFileSync(file, 'utf8'))
    .join('\n')
}

const HEADING = /^#{2,6} (\S+ (?:UAT Scenarios|Experiments))\s*$/
const ANY_HEADING = /^#{1,6} /
const SCENARIO = /^Scenario( Outline)?: (\d+[a-z][a-z0-9]*) (.*)$/

// extractScenarios returns the scenarios of the gherkin blocks that sit under a "UAT Scenarios" or "Experiments"
// heading of a file (or of each of a list of files), in spec order: { id, title, outline, examples, feature, text }.
// idPrefix keeps only matching IDs.
export function extractScenarios(specPaths, { idPrefix = '' } = {}) {
  return [specPaths]
    .flat()
    .flatMap((file) => scenariosOfFile(file))
    .filter((scenario) => scenario.id.startsWith(idPrefix))
}

function scenariosOfFile(specPath) {
  const scenarios = []
  let heading = null
  let block = null
  for (const line of fs.readFileSync(specPath, 'utf8').split('\n')) {
    if (block === null) {
      const match = HEADING.exec(line)
      if (match) heading = match[1]
      else if (ANY_HEADING.test(line)) heading = null
      else if (line.startsWith('```gherkin')) block = []
      continue
    }
    if (line.startsWith('```')) {
      if (heading !== null) scenarios.push(...scenariosOf(block, heading))
      block = null
      continue
    }
    block.push(line)
  }
  return scenarios
}

function scenariosOf(lines, feature) {
  const out = []
  let current = null
  for (const line of lines) {
    const match = SCENARIO.exec(line)
    if (match) {
      current = { id: match[2], title: match[3].trim(), outline: Boolean(match[1]), examples: 0, feature, lines: [] }
      out.push(current)
    }
    if (current !== null) current.lines.push(line)
  }
  for (const scenario of out) {
    while (scenario.lines.at(-1)?.trim() === '') scenario.lines.pop()
    const rows = scenario.lines.filter((line) => line.trim().startsWith('|')).length
    scenario.examples = scenario.outline ? Math.max(rows - 1, 0) : 0
    scenario.text = scenario.lines.join('\n')
    delete scenario.lines
  }
  return out
}

// selectScenarios keeps the scenarios whose ID starts with filter ("" keeps all), in spec order.
export function selectScenarios(scenarios, filter) {
  return scenarios.filter((scenario) => scenario.id.startsWith(filter))
}

// writeFeatures writes one feature file per heading into dir (cucumber-js reads features from paths) and returns
// the paths. The only text added to the spec's scenarios is the Feature line.
export function writeFeatures(scenarios, dir) {
  const byFeature = new Map()
  for (const scenario of scenarios) {
    if (!byFeature.has(scenario.feature)) byFeature.set(scenario.feature, [])
    byFeature.get(scenario.feature).push(scenario)
  }
  const files = []
  for (const [feature, members] of byFeature) {
    const file = path.join(dir, `${feature.split(' ')[0]}.feature`)
    fs.writeFileSync(file, `Feature: ${feature}\n\n${members.map((member) => member.text).join('\n\n')}\n`)
    files.push(file)
  }
  return files
}
