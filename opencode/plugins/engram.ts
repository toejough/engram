import type { Plugin } from "@opencode-ai/plugin"
import { tool } from "@opencode-ai/plugin"
import * as fs from "fs"
import * as path from "path"
import * as os from "os"

const ENGRAM_BIN = path.join(os.homedir(), ".local", "bin", "engram")
const COMPANION_MODEL = "opencode/qwen3.6-plus"

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

interface CycleResult {
  learned: any[]
  recalled: { query: string; report: string }[]
}

async function runEngramCycle(projectDir: string): Promise<CycleResult> {
  await ensureBinary()
  const llmCmd = `opencode run -m ${COMPANION_MODEL}`
  const proc = Bun.spawn(
    [ENGRAM_BIN, "cycle", "--llm-cmd", llmCmd, "--project-dir", projectDir],
    {
      stdout: "pipe",
      stderr: "pipe",
      env: { ...process.env, ENGRAM_COMPANION_MODE: "1" },
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
    return { learned: [], recalled: [] }
  }

  const trimmed = stdout.trim()
  if (!trimmed) {
    return { learned: [], recalled: [] }
  }

  try {
    return JSON.parse(trimmed) as CycleResult
  } catch (parseErr) {
    console.error(`[engram] cycle JSON parse failed: ${String(parseErr).slice(0, 500)}`)
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
      const before = output.system[0]
      const reminder = await getReminder("system")

      if (process.env.ENGRAM_COMPANION_MODE === "1") {
        output.system[0] = before + reminder
        return
      }

      const projectDir = input?.directory ?? process.cwd()

      try {
        const cycleResult = await runEngramCycle(projectDir)
        const block = formatCycleResult(cycleResult)
        output.system[0] = before + reminder + (block ? "\n\n" + block : "")
      } catch (err) {
        console.error(`[engram] cycle invocation failed: ${String(err).slice(0, 500)}`)
        output.system[0] = before + reminder
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
