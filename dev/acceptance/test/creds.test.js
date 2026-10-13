import assert from 'node:assert/strict'
import test from 'node:test'

import { resolveCredentials, redact, redactDeep, secretValues, TOKEN_FILE } from '../lib/creds.js'

const SECRET = 'sk-ant-oat01-SECRETVALUE'

function fakeFs(content) {
  const reads = []
  return { reads, readFile: (file) => { reads.push(file); if (content === null) throw Object.assign(new Error('ENOENT'), { code: 'ENOENT' }); return content } }
}

test('the environment token wins and the token file is not read', () => {
  const fs = fakeFs(`CLAUDE_CODE_OAUTH_TOKEN=${SECRET}-FILE\n`)
  const creds = resolveCredentials({ CLAUDE_CODE_OAUTH_TOKEN: SECRET }, fs)
  assert.deepEqual(creds.env, { CLAUDE_CODE_OAUTH_TOKEN: SECRET })
  assert.deepEqual(fs.reads, [])
})

test('an API key in the environment is used without reading the token file', () => {
  const fs = fakeFs(`CLAUDE_CODE_OAUTH_TOKEN=${SECRET}\n`)
  const creds = resolveCredentials({ ANTHROPIC_API_KEY: 'sk-ant-api-KEY' }, fs)
  assert.deepEqual(creds.env, { ANTHROPIC_API_KEY: 'sk-ant-api-KEY' })
  assert.deepEqual(fs.reads, [])
})

test('with neither variable set the token file is read, with or without export and quotes', () => {
  for (const line of [`CLAUDE_CODE_OAUTH_TOKEN=${SECRET}`, `export CLAUDE_CODE_OAUTH_TOKEN="${SECRET}"`, `CLAUDE_CODE_OAUTH_TOKEN='${SECRET}'`]) {
    const fs = fakeFs(`# a comment\nOTHER=1\n${line}\n`)
    const creds = resolveCredentials({}, fs)
    assert.deepEqual(creds.env, { CLAUDE_CODE_OAUTH_TOKEN: SECRET }, line)
    assert.deepEqual(fs.reads, [TOKEN_FILE])
  }
})

test('with no credential anywhere the error names the variables and the file and holds no secret', () => {
  assert.throws(
    () => resolveCredentials({}, fakeFs(null)),
    (error) => /CLAUDE_CODE_OAUTH_TOKEN/.test(error.message) && /ANTHROPIC_API_KEY/.test(error.message) && error.message.includes(TOKEN_FILE),
  )
  assert.throws(
    () => resolveCredentials({}, fakeFs(`NOTHING=${SECRET}\n`)),
    (error) => /CLAUDE_CODE_OAUTH_TOKEN/.test(error.message) && !error.message.includes(SECRET),
  )
})

test('redact removes every credential value from a text', () => {
  const creds = resolveCredentials({ CLAUDE_CODE_OAUTH_TOKEN: SECRET }, fakeFs(''))
  assert.equal(redact(`token ${SECRET} and again ${SECRET}`, creds), 'token [redacted] and again [redacted]')
})

test('redactDeep removes every credential value from nested evidence', () => {
  const creds = resolveCredentials({ CLAUDE_CODE_OAUTH_TOKEN: SECRET }, fakeFs(''))
  const evidence = { a: [`x ${SECRET}`, 1], b: { c: SECRET, d: null } }
  assert.deepEqual(redactDeep(evidence, creds), { a: ['x [redacted]', 1], b: { c: '[redacted]', d: null } })
})

test('the values to remove from a message are those of the environment and of the token file, whether or not credentials resolved', () => {
  const file = fakeFs(`CLAUDE_CODE_OAUTH_TOKEN=${SECRET}-FILE\n`)
  assert.deepEqual(secretValues({ ANTHROPIC_API_KEY: SECRET, CLAUDE_CODE_OAUTH_TOKEN: `${SECRET}-ENV` }, file).sort(), [SECRET, `${SECRET}-ENV`, `${SECRET}-FILE`].sort())
  assert.deepEqual(secretValues({}, fakeFs(null)), [])
})
