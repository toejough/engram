import type { Plugin } from "@opencode-ai/plugin"
import { tool } from "@opencode-ai/plugin"
import * as fs from "fs"
import * as path from "path"
import * as os from "os"

const ENGRAM_BIN = path.join(os.homedir(), ".local", "bin", "engram")
const COMPANION_MODEL = "opencode/qwen3.6-plus"
const DEBUG_LOG_PATH = process.env.ENGRAM_DEBUG_LOG ||
  path.join(os.homedir(), ".local", "share", "engram", "cycle-debug.log")
const TRUNCATE_PREVIEW = 200

function truncate(s: string, max: number): string {
  const collapsed = s.replace(/[\r\n]+/g, " ")
  if (collapsed.length <= max) return collapsed
  return collapsed.slice(0, Math.max(0, max - 3)) + "..."
}

function debugLog(stage: string, fields: Record<string, any> = {}): void {
  try {
    const ts = new Date().toISOString()
    const fieldStr = Object.entries(fields)
      .map(([k, v]) => {
        const s = typeof v === "string" ? v : JSON.stringify(v)
        return `${k}=${truncate(s, TRUNCATE_PREVIEW)}`
      })
      .join(" ")
    fs.appendFileSync(DEBUG_LOG_PATH, `${ts} [plugin] ${stage}: ${fieldStr}\n`)
  } catch {
    // Never let logging break the plugin path.
  }
}

function findPluginRoot(): string | null {
  let dir = import.meta.dirname || __dirname
  while (dir !== path.dirname(dir)) {
    if (fs.existsSync(path.join(dir, "go.mod"))) return dir
    dir = path.dirname(dir)
  }
  return null
}

async function ensureBinary(): Promise<void> {
  const start = Date.now()
  const pluginRoot = findPluginRoot()
  if (!pluginRoot) {
    debugLog("ensureBinary.skip", { reason: "no plugin root" })
    return
  }

  if (!fs.existsSync(ENGRAM_BIN)) {
    debugLog("ensureBinary.bootstrap.start", { binPath: ENGRAM_BIN, pluginRoot })
    const binDir = path.dirname(ENGRAM_BIN)
    if (!fs.existsSync(binDir)) fs.mkdirSync(binDir, { recursive: true })
    const proc = Bun.spawn(["go", "build", "-o", ENGRAM_BIN, "./cmd/engram/"], {
      cwd: pluginRoot, stdout: "pipe", stderr: "pipe",
    })
    await proc.exited
    if (proc.exitCode !== 0) {
      console.error("[engram] bootstrap go build failed:", await proc.stderr.text())
    }
    debugLog("ensureBinary.bootstrap.end", { exit: proc.exitCode, took_ms: Date.now() - start })
    return
  }

  debugLog("ensureBinary.buildSelf.start", { binPath: ENGRAM_BIN })
  const proc = Bun.spawn([ENGRAM_BIN, "build-self", "--if-stale",
    "--plugin-root", pluginRoot, "--bin-path", ENGRAM_BIN],
    { stdout: "pipe", stderr: "pipe" })
  await proc.exited
  if (proc.exitCode !== 0) {
    console.error("[engram] build-self failed:", await proc.stderr.text())
  }
  debugLog("ensureBinary.buildSelf.end", { exit: proc.exitCode, took_ms: Date.now() - start })
}

async function getReminder(kind: "system" | "session-start" | "user-prompt" | "post-tool"): Promise<string> {
  const start = Date.now()
  const proc = Bun.spawn([ENGRAM_BIN, "reminder", kind], { stdout: "pipe", stderr: "pipe" })
  await proc.exited
  if (proc.exitCode !== 0) {
    console.error(`[engram] reminder ${kind} failed:`, await proc.stderr.text())
    debugLog("getReminder.error", { kind, exit: proc.exitCode, took_ms: Date.now() - start })
    return ""
  }
  const text = (await proc.stdout.text()).trim()
  debugLog("getReminder.end", { kind, took_ms: Date.now() - start })
  return text
}

interface CycleResult {
  learned: any[]
  recalled: { query: string; report: string }[]
}

async function runEngramCycle(projectDir: string): Promise<CycleResult> {
  const start = Date.now()
  await ensureBinary()
  const llmCmd = `opencode run -m ${COMPANION_MODEL}`
  debugLog("runEngramCycle.start", { projectDir, llmCmd, debugLogPath: DEBUG_LOG_PATH })
  const proc = Bun.spawn(
    [ENGRAM_BIN, "cycle", "--llm-cmd", llmCmd, "--project-dir", projectDir],
    {
      stdout: "pipe",
      stderr: "pipe",
      env: {
        ...process.env,
        ENGRAM_COMPANION_MODE: "1",
        ENGRAM_DEBUG_LOG: DEBUG_LOG_PATH,
      },
    },
  )

  // Read stdout and stderr concurrently to keep the subprocess from blocking
  // on a full stderr pipe (engram recall logs ~5KB+ per session inspected).
  const [stdout, stderr] = await Promise.all([
    proc.stdout.text(),
    proc.stderr.text(),
  ])
  await proc.exited

  if (proc.exitCode !== 0) {
    console.error(`[engram] cycle exit ${proc.exitCode}: ${stderr.slice(0, 2000)}`)
    debugLog("runEngramCycle.error", {
      exit: proc.exitCode,
      stderr_head: stderr.slice(0, TRUNCATE_PREVIEW),
      took_ms: Date.now() - start,
    })
    return { learned: [], recalled: [] }
  }

  const trimmed = stdout.trim()
  if (!trimmed) {
    debugLog("runEngramCycle.end", { exit: 0, outcome: "empty_stdout", took_ms: Date.now() - start })
    return { learned: [], recalled: [] }
  }

  try {
    const result = JSON.parse(trimmed) as CycleResult
    debugLog("runEngramCycle.end", {
      exit: 0,
      outcome: "ok",
      learned: result.learned.length,
      recalled: result.recalled.length,
      took_ms: Date.now() - start,
    })
    return result
  } catch (parseErr) {
    console.error(`[engram] cycle JSON parse failed: ${String(parseErr).slice(0, 500)}`)
    debugLog("runEngramCycle.parseError", {
      err: String(parseErr).slice(0, TRUNCATE_PREVIEW),
      stdout_head: trimmed.slice(0, TRUNCATE_PREVIEW),
      took_ms: Date.now() - start,
    })
    return { learned: [], recalled: [] }
  }
}

function formatCycleResult(result: CycleResult): string {
  if (!result.recalled || result.recalled.length === 0) {
    return ""
  }

  let block = "## Recalled memories\n"
  for (const { query, report } of result.recalled) {
    block += `\n### Query: ${query}\n${report}\n`
  }

  return block.trimEnd()
}

export const EngramPlugin: Plugin = async ({ client, $ }) => {
  await ensureBinary()

  return {

    "experimental.chat.system.transform": async (input: any, output) => {
      const start = Date.now()
      const sessionID = input?.sessionID ?? "<none>"
      const recursionGuard = process.env.ENGRAM_COMPANION_MODE === "1"
      debugLog("system.transform.start", {
        sessionID,
        directory: input?.directory ?? "<none>",
        recursionGuard: recursionGuard ? "active" : "inactive",
      })

      const before = output.system[0]
      const reminder = await getReminder("system")

      if (recursionGuard) {
        output.system[0] = before + reminder
        debugLog("system.transform.end", {
          sessionID,
          path: "short-circuit",
          took_ms: Date.now() - start,
        })
        return
      }

      const projectDir = input?.directory ?? process.cwd()

      try {
        const cycleResult = await runEngramCycle(projectDir)
        const block = formatCycleResult(cycleResult)
        output.system[0] = before + reminder + (block ? "\n\n" + block : "")
        debugLog("system.transform.end", {
          sessionID,
          path: "cycle",
          learned: cycleResult.learned.length,
          recalled: cycleResult.recalled.length,
          block_present: block.length > 0,
          took_ms: Date.now() - start,
        })
      } catch (err) {
        console.error(`[engram] cycle invocation failed: ${String(err).slice(0, 500)}`)
        output.system[0] = before + reminder
        debugLog("system.transform.error", {
          sessionID,
          err: String(err).slice(0, TRUNCATE_PREVIEW),
          took_ms: Date.now() - start,
        })
      }
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
