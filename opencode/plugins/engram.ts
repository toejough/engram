import type { Plugin } from "@opencode-ai/plugin"
import { tool } from "@opencode-ai/plugin"
import * as fs from "fs"
import * as path from "path"
import * as os from "os"

const ENGRAM_BIN = path.join(os.homedir(), ".local", "bin", "engram")

function findPluginRoot(): string | null {
  let dir = import.meta.dirname || __dirname
  while (dir !== path.dirname(dir)) {
    if (fs.existsSync(path.join(dir, "go.mod"))) return dir
    dir = path.dirname(dir)
  }
  return null
}

async function ensureBinary(): Promise<void> {
  const pluginRoot = findPluginRoot()
  if (!pluginRoot) return

  if (!fs.existsSync(ENGRAM_BIN)) {
    const binDir = path.dirname(ENGRAM_BIN)
    if (!fs.existsSync(binDir)) fs.mkdirSync(binDir, { recursive: true })
    const proc = Bun.spawn(["go", "build", "-o", ENGRAM_BIN, "./cmd/engram/"], {
      cwd: pluginRoot, stdout: "pipe", stderr: "pipe",
    })
    await proc.exited
    if (proc.exitCode !== 0) {
      console.error("[engram] bootstrap go build failed:", await proc.stderr.text())
    }
    return
  }

  const proc = Bun.spawn([ENGRAM_BIN, "build-self", "--if-stale",
    "--plugin-root", pluginRoot, "--bin-path", ENGRAM_BIN],
    { stdout: "pipe", stderr: "pipe" })
  await proc.exited
  if (proc.exitCode !== 0) {
    console.error("[engram] build-self failed:", await proc.stderr.text())
  }
}

async function getReminder(kind: "system" | "session-start" | "user-prompt" | "post-tool"): Promise<string> {
  const proc = Bun.spawn([ENGRAM_BIN, "reminder", kind], { stdout: "pipe", stderr: "pipe" })
  await proc.exited
  if (proc.exitCode !== 0) {
    console.error(`[engram] reminder ${kind} failed:`, await proc.stderr.text())
    return ""
  }
  return (await proc.stdout.text()).trim()
}

const DEBUG_LOG = path.join(os.homedir(), ".local", "share", "engram", "debug-system-transform.log")
const COMPANION_TRACE = path.join(os.homedir(), ".local", "share", "engram", "companion-trace.jsonl")
const COMPANION_SESSION_DIR = path.join(os.homedir(), ".local", "share", "engram", "companion-session")
const COMPANION_MODEL = "opencode/qwen3.6-plus"
const COMPANION_PROMPT_PREFIX = `You are a memory steward observing a primary AI agent's project session. Your job: read the recent project history below and emit a concise block of "memories worth injecting" — facts, prior corrections, and project context the primary agent should see for its upcoming turn.

Format your output exactly like this, and emit nothing else:

## Recalled memories
- <one sentence per relevant memory or fact, drawn directly from the project history below>
- <another, if relevant>

If nothing in the history seems relevant for the upcoming turn, output exactly: NO RELEVANT MEMORIES

PROJECT HISTORY (most recent message at end):
`

function logTransform(before: string, reminder: string, after: string): void {
  try {
    const dir = path.dirname(DEBUG_LOG)
    if (!fs.existsSync(dir)) fs.mkdirSync(dir, { recursive: true })
    const entry = [
      "=== SYSTEM TRANSFORM ===",
      `--- BEFORE (${before.length} chars) ---`,
      before,
      `--- ENGRAM REMINDER (${reminder.length} chars) ---`,
      reminder,
      `--- AFTER (${after.length} chars) ---`,
      after,
      `=== END ===\n`,
      "",
    ].join("\n")
    fs.appendFileSync(DEBUG_LOG, entry, "utf8")
  } catch {
    // logging failure is non-fatal
  }
}

function companionTrace(stage: string, info: Record<string, any>): void {
  try {
    const dir = path.dirname(COMPANION_TRACE)
    if (!fs.existsSync(dir)) fs.mkdirSync(dir, { recursive: true })
    const entry = JSON.stringify({ ts: new Date().toISOString(), stage, ...info }) + "\n"
    fs.appendFileSync(COMPANION_TRACE, entry, "utf8")
  } catch {
    // tracing failure is non-fatal
  }
}

async function runEngramRecall(): Promise<string> {
  const proc = Bun.spawn([ENGRAM_BIN, "recall", "--no-external-sources"], { stdout: "pipe", stderr: "pipe" })
  await proc.exited
  return (await proc.stdout.text()).trim()
}

function readCompanionSession(primarySessionID: string): string | null {
  try {
    const p = path.join(COMPANION_SESSION_DIR, `${primarySessionID}.txt`)
    if (!fs.existsSync(p)) return null
    return fs.readFileSync(p, "utf8").trim()
  } catch {
    return null
  }
}

function writeCompanionSession(primarySessionID: string, companionSessionID: string): void {
  try {
    if (!fs.existsSync(COMPANION_SESSION_DIR)) fs.mkdirSync(COMPANION_SESSION_DIR, { recursive: true })
    fs.writeFileSync(path.join(COMPANION_SESSION_DIR, `${primarySessionID}.txt`), companionSessionID, "utf8")
  } catch {
    // non-fatal
  }
}

async function runCompanion(primarySessionID: string, prompt: string): Promise<string> {
  const existingCompanion = readCompanionSession(primarySessionID)
  const args = ["run", "-m", COMPANION_MODEL, "--format", "json"]
  if (existingCompanion) args.push("-s", existingCompanion)
  args.push(prompt)

  // ENGRAM_COMPANION_MODE breaks the recursive companion-spawning loop:
  // when the companion's opencode process loads this plugin, the
  // system.transform hook checks the env var and skips its own companion call.
  const proc = Bun.spawn(["opencode", ...args], {
    stdout: "pipe",
    stderr: "pipe",
    env: { ...process.env, ENGRAM_COMPANION_MODE: "1" },
  })
  await proc.exited
  if (proc.exitCode !== 0) {
    companionTrace("companion-run-failed", { exitCode: proc.exitCode, stderr: (await proc.stderr.text()).slice(0, 1000) })
    return ""
  }

  const stdout = await proc.stdout.text()
  let capturedSessionID = ""
  let companionText = ""

  for (const line of stdout.split("\n")) {
    if (!line.trim()) continue
    try {
      const ev = JSON.parse(line)
      if (ev.sessionID && !capturedSessionID) capturedSessionID = ev.sessionID
      if (ev.type === "text" && ev.part?.text) companionText += ev.part.text
    } catch {
      // skip non-JSON lines
    }
  }

  if (!existingCompanion && capturedSessionID) {
    writeCompanionSession(primarySessionID, capturedSessionID)
    companionTrace("companion-session-created", { primarySessionID, companionSessionID: capturedSessionID })
  }

  return companionText.trim()
}

export const EngramPlugin: Plugin = async ({ client, $ }) => {
  await ensureBinary()

  return {
    event: async ({ event }) => {
      if (event.type === "session.created") {
        await ensureBinary()
      }
    },

    "experimental.chat.system.transform": async (input: any, output) => {
      const before = output.system[0]
      const reminder = await getReminder("system")
      const sessionID = input?.sessionID

      // Guard against recursion: when this plugin is loaded inside the
      // companion's own opencode process, ENGRAM_COMPANION_MODE is set and
      // we must NOT spawn another companion. We still inject the reminder.
      if (process.env.ENGRAM_COMPANION_MODE === "1") {
        companionTrace("system.transform-skipped-companion", { sessionID, reason: "ENGRAM_COMPANION_MODE" })
        output.system[0] = before + reminder
        logTransform(before, reminder, output.system[0])
        return
      }

      let companionBlock = ""
      try {
        companionTrace("system.transform-start", { sessionID })

        const recallStart = Date.now()
        const recallOutput = await runEngramRecall()
        companionTrace("recall-complete", { sessionID, recallMs: Date.now() - recallStart, recallLen: recallOutput.length })

        const prompt = COMPANION_PROMPT_PREFIX + recallOutput
        const companionStart = Date.now()
        const companionOutput = await runCompanion(sessionID || "default", prompt)
        companionTrace("companion-complete", { sessionID, companionMs: Date.now() - companionStart, companionOutLen: companionOutput.length, companionOutSample: companionOutput.slice(0, 500) })

        if (companionOutput && !companionOutput.includes("NO RELEVANT MEMORIES")) {
          companionBlock = "\n\n" + companionOutput
          companionTrace("companion-injected", { sessionID, blockLen: companionBlock.length })
        } else {
          companionTrace("companion-skipped", { sessionID, reason: companionOutput ? "no-memories" : "empty-output" })
        }
      } catch (err: any) {
        companionTrace("companion-error", { sessionID, error: String(err) })
      }

      const injected = reminder + companionBlock
      output.system[0] = before + injected
      logTransform(before, injected, output.system[0])
    },

    "tool.execute.after": async (_input, output) => {
      const reminder = await getReminder("post-tool")
      output.output += "\n\n" + reminder
    },

    tool: {
      engram_recall: tool({
        description: "Recall recent session context or search memories using engram",
        args: {
          query: tool.schema.string().optional().describe("Search query (omit for summary mode)"),
        },
        async execute(args) {
          const cmdArgs = ["recall"]
          if (args.query) cmdArgs.push("--query", args.query)
          const proc = Bun.spawn([ENGRAM_BIN, ...cmdArgs], { stdout: "pipe", stderr: "pipe" })
          const [stdout, stderr] = await Promise.all([proc.stdout.text(), proc.stderr.text()])
          return (stdout + (stderr ? "\n" + stderr : "")).trim()
        },
      }),

      engram_learn_feedback: tool({
        description: "Learn from behavioral feedback using SBIA format",
        args: {
          situation: tool.schema.string().describe("Context when this applies"),
          behavior: tool.schema.string().describe("Observed behavior"),
          impact: tool.schema.string().describe("Impact of the behavior"),
          action: tool.schema.string().describe("Recommended action"),
          source: tool.schema.string().optional().describe("Human or agent"),
        },
        async execute(args) {
          const cmdArgs = ["learn", "feedback",
            "--situation", args.situation,
            "--behavior", args.behavior,
            "--impact", args.impact,
            "--action", args.action,
          ]
          if (args.source) cmdArgs.push("--source", args.source)
          const proc = Bun.spawn([ENGRAM_BIN, ...cmdArgs], { stdout: "pipe", stderr: "pipe" })
          const [stdout, stderr] = await Promise.all([proc.stdout.text(), proc.stderr.text()])
          return (stdout + (stderr ? "\n" + stderr : "")).trim()
        },
      }),

      engram_learn_fact: tool({
        description: "Learn a factual statement using SPO format",
        args: {
          situation: tool.schema.string().describe("Context when this applies"),
          subject: tool.schema.string().describe("Subject of the fact"),
          predicate: tool.schema.string().describe("Relationship or verb"),
          object: tool.schema.string().describe("Object of the fact"),
          source: tool.schema.string().optional().describe("Human or agent"),
        },
        async execute(args) {
          const cmdArgs = ["learn", "fact",
            "--situation", args.situation,
            "--subject", args.subject,
            "--predicate", args.predicate,
            "--object", args.object,
          ]
          if (args.source) cmdArgs.push("--source", args.source)
          const proc = Bun.spawn([ENGRAM_BIN, ...cmdArgs], { stdout: "pipe", stderr: "pipe" })
          const [stdout, stderr] = await Promise.all([proc.stdout.text(), proc.stderr.text()])
          return (stdout + (stderr ? "\n" + stderr : "")).trim()
        },
      }),

      engram_update: tool({
        description: "Update fields on an existing memory",
        args: {
          name: tool.schema.string().describe("Memory slug to update"),
          situation: tool.schema.string().optional().describe("Context when this applies"),
          behavior: tool.schema.string().optional().describe("Observed behavior"),
          impact: tool.schema.string().optional().describe("Impact of the behavior"),
          action: tool.schema.string().optional().describe("Recommended action"),
          subject: tool.schema.string().optional().describe("Subject of the fact"),
          predicate: tool.schema.string().optional().describe("Relationship or verb"),
          object: tool.schema.string().optional().describe("Object of the fact"),
          source: tool.schema.string().optional().describe("Human or agent"),
        },
        async execute(args) {
          const cmdArgs = ["update", "--name", args.name]
          if (args.situation) cmdArgs.push("--situation", args.situation)
          if (args.behavior) cmdArgs.push("--behavior", args.behavior)
          if (args.impact) cmdArgs.push("--impact", args.impact)
          if (args.action) cmdArgs.push("--action", args.action)
          if (args.subject) cmdArgs.push("--subject", args.subject)
          if (args.predicate) cmdArgs.push("--predicate", args.predicate)
          if (args.object) cmdArgs.push("--object", args.object)
          if (args.source) cmdArgs.push("--source", args.source)
          const proc = Bun.spawn([ENGRAM_BIN, ...cmdArgs], { stdout: "pipe", stderr: "pipe" })
          const [stdout, stderr] = await Promise.all([proc.stdout.text(), proc.stderr.text()])
          return (stdout + (stderr ? "\n" + stderr : "")).trim()
        },
      }),

      engram_show: tool({
        description: "Display full memory details",
        args: {
          name: tool.schema.string().describe("Memory slug to display"),
        },
        async execute(args) {
          const proc = Bun.spawn([ENGRAM_BIN, "show", "--name", args.name], { stdout: "pipe", stderr: "pipe" })
          const [stdout, stderr] = await Promise.all([proc.stdout.text(), proc.stderr.text()])
          return (stdout + (stderr ? "\n" + stderr : "")).trim()
        },
      }),

      engram_list: tool({
        description: "List all memories with type, name, and situation",
        args: {},
        async execute() {
          const proc = Bun.spawn([ENGRAM_BIN, "list"], { stdout: "pipe", stderr: "pipe" })
          const [stdout, stderr] = await Promise.all([proc.stdout.text(), proc.stderr.text()])
          return (stdout + (stderr ? "\n" + stderr : "")).trim()
        },
      }),
    },
  }
}
