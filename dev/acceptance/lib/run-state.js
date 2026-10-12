// The run context is created once, by the first scenario that needs a session, and disposed once by the process
// that started the run.
import fs from 'node:fs'

import { createRunContext, disposeRunContext } from './run-context.js'
import { resolveCredentials } from './creds.js'
import { preflight } from './preflight.js'

let pending = null
let created = null

export function getRunContext() {
  pending ??= (async () => {
    const claude = preflight({ needClaude: true })
    const creds = resolveCredentials(process.env, { readFile: (file) => fs.readFileSync(file, 'utf8') })
    created = await createRunContext({ claudePath: claude.path, creds })
    return created
  })()
  return pending
}

export async function disposeRun() {
  if (pending) await pending.catch(() => undefined)
  disposeRunContext(created)
}
