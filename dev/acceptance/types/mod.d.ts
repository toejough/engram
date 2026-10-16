// The shapes the mod's JavaScript (agent-instructions/mods/engram-memory/hooks) is type-checked against (checkJs, 793e8n).
// The platform writes its own type files only in an interactive session, so the parts of the context `$` that the mod
// uses are declared here, beside the declaration of the test kit; the events and the engine's responses stay open.

interface ModProcessResult {
  exitCode: number
  stdout: string
  stderr: string
}

// The context a hook receives. Everything the mod does to the world goes through it.
interface Ctx {
  process: { run(argv: string[], options: { stdin: string; timeoutMs: number }): Promise<ModProcessResult> }
  clock: {
    now(): Promise<number>
    sleep(milliseconds: number): Promise<void>
    after(milliseconds: number, run: () => unknown): unknown
  }
  model: { complete(request: unknown): Promise<unknown> }
  session: {
    messages(): Promise<{ role: string; text: string }[]>
    id(): Promise<string>
    turns(): Promise<number>
  }
  ui: { log(line: string, options?: { to: string }): void }
  fs: { write(path: string, text: string): Promise<void> }
  env: { get(name: string): Promise<string | undefined> }
}

// A job of `engram auto run`: the fields the mod names, and the rest the engine reads.
interface Job {
  kind: string
  [field: string]: any
}
// The response of a round of a job.
interface Round {
  log?: string[]
  warnings?: string[]
  counts?: Record<string, number>
  done?: any
  request?: unknown
  waitMs?: number
  state?: unknown
  truncated?: boolean
}
// The hold of an event on a job: set when the hold has passed.
interface Hold {
  cut: boolean
}
// The in-turn cue the mod keeps while its job runs.
interface Cue {
  done: boolean
  notes: unknown[]
  startedAt: number
  promise?: Promise<void>
}
// A hook of the mod: the context, the event as received, and the rest of the chain.
type Hook = ($: Ctx, event: any, next: (event: any) => any) => Promise<any>
type PassOn = (context: unknown, event: any, next: (event: any) => any) => any
type StreamHook = ($: Ctx, event: any, next: (event: any) => any) => AsyncGenerator<any, any, any>
// What the mod keeps between events.
interface ModState {
  ledger: unknown
  work: unknown
  transcript: string
  lastPrompt: string
  activity: string[]
  step: Cue | null
  deep: Cue | null
  hooks: Promise<void>
  tally: Record<string, number>[]
  spawns: string[]
  queue: { tail: Promise<void>; pending: number; seam: string[]; turnEnded: boolean }
}
