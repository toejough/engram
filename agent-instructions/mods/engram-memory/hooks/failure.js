// How the System's own work fails and what the user is told (793a4b, 793b205). A failing step is `unavailable` (the
// engram program cannot start, or the vault cannot be read), `timeout` (a named deadline passed, or a model call was
// cut) or `error` (anything else). Every call on `$` is in register.js.

// The deadlines of the steps, in milliseconds (initial values, tuned under #850): the lookup of a moment holds the
// agent's prompt or tool result, so it is the shortest and, with the relevance judgement's own deadline (judge.js), it
// holds the agent for 8 s at most (793b317); the lookups of background work, the other commands, the scan for pending
// offers and the index have their own.
export const DEADLINE_MS = { moment: 4000, lookup: 25000, command: 30000, scan: 30000, ingest: 300000 }

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

// What the messages of a failure say: `$.process.run` rejecting ("... aborted: still running after Nms", "... failed to
// start: ENOENT ..."), a model call that was cut ("aborted"), what engram or grep print when the vault directory cannot
// be read.
const TIMED_OUT = /still running after|timed out|^aborted$/i
const UNAVAILABLE = /failed to start|ENOENT|permission denied|no such file or directory|read-only file system|input\/output error/i

// failureKind is the class of a failure from the text it left: the message of a rejected call, the stderr of a command,
// the reason a model call gave.
export function failureKind(text) {
  if (TIMED_OUT.test(text)) return 'timeout'
  return UNAVAILABLE.test(text) ? 'unavailable' : 'error'
}

export function kindOf(error) {
  return error instanceof StepError ? error.kind : 'error'
}

// warning is the one line the user is shown, on the status line and nowhere else: what was skipped and why.
export function warning(step, kind) {
  return `Automatic memory was skipped because ${REASON[kind] ?? REASON.error} (${step})`
}
