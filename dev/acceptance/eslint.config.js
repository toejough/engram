// Standards for the JavaScript and TypeScript this change owns (793e8n): the mod
// (agent-instructions/mods/engram-memory), the acceptance harness and the probes. The rules follow the spirit of the Go
// rules in .claude/rules/go.md: named constants instead of magic numbers, descriptive names, lines under 120
// characters, errors that keep their cause, nothing unused, and consistent error handling. Complexity and size are a
// gate of their own (eslint.complexity.config.js).
import js from '@eslint/js'
import globals from 'globals'
import tseslint from 'typescript-eslint'

export const LINE_LENGTH = 120
const SCOPES = [
  'dev/acceptance/**/*.js',
  'agent-instructions/mods/engram-memory/hooks/*.js',
  'agent-instructions/mods/engram-memory/tests/*.ts',
]

const shared = {
  // consistent error handling
  'no-empty': ['error', { allowEmptyCatch: false }],
  'no-useless-catch': 'error',
  'no-throw-literal': 'error',
  'prefer-promise-reject-errors': 'error',
  'preserve-caught-error': 'error',
  'no-promise-executor-return': 'error',
  'require-atomic-updates': 'error',
  // nothing unused
  'no-unused-vars': ['error', { args: 'after-used', caughtErrors: 'all', ignoreRestSiblings: false }],
  // names and constants
  'no-magic-numbers': ['error', { ignore: [0, 1, -1], ignoreArrayIndexes: true, enforceConst: true }],
  'id-length': ['error', { min: 3, exceptions: ['$', '_', 'on', 'fs', 'os', 'id', 'to', 'ok', 'js'] }],
  'max-len': ['error', { code: LINE_LENGTH, ignoreUrls: true }],
  // consistency
  eqeqeq: ['error', 'always'],
  'no-var': 'error',
  'prefer-const': 'error',
  'no-shadow': 'error',
  'no-param-reassign': 'error',
  'no-implicit-coercion': 'error',
  'no-nested-ternary': 'error',
}

export default tseslint.config(
  { ignores: ['**/node_modules/**', 'dev/acceptance/types/**'] },
  {
    files: SCOPES,
    extends: [js.configs.recommended],
    languageOptions: { ecmaVersion: 'latest', sourceType: 'module', globals: { ...globals.node } },
    rules: shared,
  },
  {
    // as the Go configuration excludes mnd (and funlen, goconst) from _test.go files
    files: ['**/*.test.js', 'dev/acceptance/test/**/*.js', 'agent-instructions/mods/engram-memory/tests/*.ts'],
    rules: { 'no-magic-numbers': 'off' },
  },
  {
    files: ['agent-instructions/mods/engram-memory/tests/*.ts'],
    extends: [...tseslint.configs.recommended],
    languageOptions: { parserOptions: { projectService: false } },
    rules: {
      ...shared,
      'no-magic-numbers': 'off',
      'no-unused-vars': 'off',
      '@typescript-eslint/no-unused-vars': shared['no-unused-vars'],
    },
  },
)
