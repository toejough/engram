// Credentials for the sessions (793b245, 793b267): the environment first; the owner's token file only when
// neither variable is set. No value is ever put in a message, a log or a file.
import os from 'node:os'
import path from 'node:path'

export const TOKEN_FILE = path.join(os.homedir(), '.config', 'claude-key.env')
const OAUTH = 'CLAUDE_CODE_OAUTH_TOKEN'
const API_KEY = 'ANTHROPIC_API_KEY'

// resolveCredentials returns { env, values }: env is what a session gets, values what redact removes.
export function resolveCredentials(environment, fs) {
  for (const name of [OAUTH, API_KEY]) {
    if (environment[name]) return build({ [name]: environment[name] })
  }
  let content = ''
  try {
    content = fs.readFile(TOKEN_FILE)
  } catch {
    content = ''
  }
  const token = parseTokenFile(content)
  if (token === null) {
    throw new Error(`no credentials: set ${OAUTH} or ${API_KEY}, or put ${OAUTH}=<token> in ${TOKEN_FILE}`)
  }
  return build({ [OAUTH]: token })
}

function parseTokenFile(content) {
  for (const line of content.split('\n')) {
    const match = /^\s*(?:export\s+)?CLAUDE_CODE_OAUTH_TOKEN=(.*?)\s*$/.exec(line)
    if (match) {
      const value = match[1].replace(/^(["'])(.*)\1$/, '$2')
      if (value !== '') return value
    }
  }
  return null
}

function build(env) {
  return { env, values: Object.values(env) }
}

export function redact(text, creds) {
  return creds.values.reduce((out, value) => out.split(value).join('[redacted]'), text)
}

// redactDeep removes the credential values from any JSON-serialisable value (a probe's evidence, a report).
export function redactDeep(value, creds) {
  return JSON.parse(redact(JSON.stringify(value) ?? 'null', creds))
}

// secretValues are all the credential values that can be named in a message, found without requiring that any resolves:
// both variables and the token file. An error that is printed before credentials were resolved is redacted with these.
export function secretValues(environment, fs) {
  let content = ''
  try {
    content = fs.readFile(TOKEN_FILE)
  } catch {
    content = ''
  }
  return [...new Set([environment[OAUTH], environment[API_KEY], parseTokenFile(content)].filter(Boolean))]
}
