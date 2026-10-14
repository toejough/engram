// The mod is an adapter (793e8m): event wiring and `$` calls, and nothing else. Decisions (moment detection, presence
// and relevance filtering, window building, redaction, reconcile and write, backlog selection, curation outcomes,
// summary counts, failure classification) are in the Go binary, which the mod runs through `$.process.run` and whose
// judgement requests it relays through `$.model.complete`. These rules forbid, in the hooks, the constructs that carry
// logic: a module other than register.js imports nothing (it is a pure helper or it is not there), no regular
// expression, no parsing of JSON except the output of the Go process, and functions that are straight-line wiring.
// Every rule is red until the logic has moved; the numbers are recorded in 793b361.
import globals from 'globals'

export const ADAPTER = {
  complexity: 5,
  maxStatements: 15,
  maxDepth: 2,
  maxParams: 3,
}
const PARSE = "CallExpression[callee.object.name='JSON'][callee.property.name='parse']"

export default [
  { ignores: ['**/node_modules/**'] },
  {
    files: ['agent-instructions/mods/engram-memory/hooks/*.js'],
    languageOptions: { ecmaVersion: 'latest', sourceType: 'module', globals: { ...globals.node } },
    rules: {
      complexity: ['error', ADAPTER.complexity],
      'max-statements': ['error', ADAPTER.maxStatements],
      'max-depth': ['error', ADAPTER.maxDepth],
      'max-params': ['error', ADAPTER.maxParams],
      'no-restricted-syntax': [
        'error',
        { selector: 'Literal[regex]', message: 'a regular expression is parsing or matching: it belongs in Go' },
        { selector: "NewExpression[callee.name='RegExp']", message: 'a regular expression belongs in Go' },
        {
          selector: `${PARSE} > .arguments:not(MemberExpression[property.name='stdout'])`,
          message: 'only the output of the Go process is parsed in JS; a model reply is relayed unread',
        },
      ],
    },
  },
  {
    files: ['agent-instructions/mods/engram-memory/hooks/*.js'],
    ignores: ['agent-instructions/mods/engram-memory/hooks/register.js'],
    rules: {
      'no-restricted-imports': [
        'error',
        { patterns: [{ regex: '.*', message: 'only register.js imports: any other module is a pure helper' }] },
      ],
    },
  },
]
