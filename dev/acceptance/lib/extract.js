// Extracts the Gherkin scenarios of a specification so that the spec is the test source (793e2b).
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

export const REPO_ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..', '..')
const CHANGE_DIR = path.join(REPO_ROOT, 'docs', 'changes', '793_parallel-memory-agent')
export const PRODUCT_SPEC = path.join(CHANGE_DIR, '793a_product-spec.md')
export const TECHNICAL_SPEC = path.join(CHANGE_DIR, '793d_technical-spec.md')

const HEADING = /^#{2,6} (\S+ (?:UAT Scenarios|Experiments))\s*$/
const SCENARIO = /^Scenario( Outline)?: (\d+[a-z][a-z0-9]*) (.*)$/

// extractScenarios returns the scenarios of the gherkin blocks that sit under a "UAT Scenarios" or "Experiments"
// heading, in spec order: { id, title, outline, examples, feature, text }. idPrefix keeps only matching IDs.
export function extractScenarios(specPath, { idPrefix = '' } = {}) {
  const scenarios = []
  let heading = null
  let block = null
  for (const line of fs.readFileSync(specPath, 'utf8').split('\n')) {
    if (block === null) {
      const match = HEADING.exec(line)
      if (match) heading = match[1]
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
  return scenarios.filter((scenario) => scenario.id.startsWith(idPrefix))
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
