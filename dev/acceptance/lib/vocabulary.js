// The vocabulary check (793e2i): a cucumber dry run that reports, per step line, how many definitions match it.
import { loadConfiguration, runCucumber } from '@cucumber/cucumber/api'
import { Writable } from 'node:stream'

export async function checkVocabulary({ features, support, cwd }) {
  const { runConfiguration } = await loadConfiguration(
    { provided: { paths: features, import: support, dryRun: true, strict: true, format: [] } },
    { cwd, env: process.env },
  )
  const sink = new Writable({ write: (chunk, encoding, done) => done() })
  const pickles = new Map()
  const definitions = new Map()
  const testCases = []
  await runCucumber(runConfiguration, { cwd, stdout: sink, stderr: sink, env: process.env }, (envelope) => {
    if (envelope.pickle) pickles.set(envelope.pickle.id, envelope.pickle)
    if (envelope.stepDefinition) definitions.set(envelope.stepDefinition.id, envelope.stepDefinition.pattern.source)
    if (envelope.testCase) testCases.push(envelope.testCase)
  })
  const used = new Set()
  const undefinedSteps = []
  const ambiguousSteps = []
  for (const testCase of testCases) {
    const pickle = pickles.get(testCase.pickleId)
    for (const testStep of testCase.testSteps) {
      if (!testStep.pickleStepId) continue
      const text = pickle.steps.find((step) => step.id === testStep.pickleStepId).text
      const matches = testStep.stepDefinitionIds ?? []
      matches.forEach((id) => used.add(id))
      if (matches.length === 0) undefinedSteps.push(`${pickle.name}: ${text}`)
      if (matches.length > 1) ambiguousSteps.push(`${pickle.name}: ${text}`)
    }
  }
  return {
    scenarios: testCases.length,
    definitions: definitions.size,
    undefinedSteps,
    ambiguousSteps,
    unusedDefinitions: [...definitions].filter(([id]) => !used.has(id)).map(([, source]) => source),
  }
}
