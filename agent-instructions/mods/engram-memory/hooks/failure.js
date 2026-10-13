// How the System's own work fails and what the user is told (793a4b, 793b205). A failing step is `unavailable` (the
// engram program cannot start, or the vault cannot be read), `timeout` (a named deadline passed, or a model call was
// cut) or `error` (anything else). Every call on `$` is in register.js.

// The deadlines of the steps, in milliseconds (initial values, tuned under #850): a lookup holds the agent's prompt,
// so it is the shortest; the commands of background work, the scan for pending offers and the index have their own.
export const DEADLINE_MS = { lookup: 25000, command: 30000, scan: 30000, ingest: 300000 }

const REASON = {
  unavailable: 'the vault is unavailable',
  timeout: 'the System timed out',
  error: 'of an error',
}

// StepError is a failure with its class; any other error a step throws is an `error`.
export class StepError extends Error {
  constructor(message, kind) {
    super(message)
    this.kind = kind
  }
}

// The messages of `$.process.run` when it rejects: "... aborted: still running after Nms", "... failed to start: ENOENT ..."
const TIMED_OUT = /still running after|timed out/i
const NOT_STARTED = /failed to start|ENOENT/i
// What engram or grep print when the vault directory cannot be read.
const UNREADABLE = /permission denied|no such file or directory|read-only file system|input\/output error/i

export function spawnKind(message) {
  if (TIMED_OUT.test(message)) return 'timeout'
  return NOT_STARTED.test(message) ? 'unavailable' : 'error'
}

export function exitKind(stderr) {
  return UNREADABLE.test(stderr) ? 'unavailable' : 'error'
}

// replyKind: a model call that did not answer was either cut (its deadline passed) or refused by the provider.
export function replyKind(reply) {
  return reply.reason === 'aborted' ? 'timeout' : 'error'
}

export function kindOf(error) {
  return error instanceof StepError ? error.kind : 'error'
}

// warning is the one line the user is shown, on the status line and nowhere else: what was skipped and why.
export function warning(step, kind) {
  return `Automatic memory was skipped because ${REASON[kind] ?? REASON.error} (${step})`
}
