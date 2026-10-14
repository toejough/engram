// Complexity and size of the mod (793e8m): per-file and per-function limits on the hooks module. The limits follow the
// Go rules (cyclomatic complexity 10 as cyclop, function length as funlen) and add a file size, which Go has no
// linter for. Every limit is a red gate until the mod is within it; the numbers are recorded in 793b361.
import globals from 'globals'

export const LIMITS = {
  complexity: 10,
  maxDepth: 4,
  maxParams: 5,
  maxStatements: 40,
  maxFunctionLines: 60,
  maxFileLines: 300,
}

export default [
  { ignores: ['**/node_modules/**'] },
  {
    files: ['agent-instructions/mods/engram-memory/hooks/*.js'],
    languageOptions: { ecmaVersion: 'latest', sourceType: 'module', globals: { ...globals.node } },
    linterOptions: { reportUnusedDisableDirectives: 'error' },
    rules: {
      complexity: ['error', LIMITS.complexity],
      'max-depth': ['error', LIMITS.maxDepth],
      'max-params': ['error', LIMITS.maxParams],
      'max-statements': ['error', LIMITS.maxStatements],
      'max-lines-per-function': ['error', { max: LIMITS.maxFunctionLines, skipBlankLines: true, skipComments: true }],
      'max-lines': ['error', { max: LIMITS.maxFileLines, skipBlankLines: true, skipComments: true }],
    },
  },
]
