import type { Plugin } from "@opencode-ai/plugin"
import { tool } from "@opencode-ai/plugin"
import * as fs from "fs"
import * as path from "path"
import * as os from "os"

const ENGRAM_BIN = path.join(os.homedir(), ".local", "bin", "engram")

function findPluginRoot(): string | null {
  const possible = [
    path.join(import.meta.dirname || __dirname, "..", "..", ".."),
    path.join(os.homedir(), ".config", "opencode"),
    path.join(process.cwd(), ".opencode"),
  ]
  for (const p of possible) {
    if (fs.existsSync(path.join(p, "go.mod"))) return p
  }
  return null
}

async function buildIfNeeded(): Promise<boolean> {
  return !fs.existsSync(ENGRAM_BIN) || !fs.statSync(ENGRAM_BIN).isFile()
}

async function ensureBinary(): Promise<void> {
  const shouldBuild = await buildIfNeeded()
  if (!shouldBuild) return

  const pluginRoot = findPluginRoot()
  if (!pluginRoot) return

  const binDir = path.dirname(ENGRAM_BIN)
  try {
    await Bun.spawn(["mkdir", "-p", binDir], { cwd: pluginRoot, stdout: "pipe", stderr: "pipe" }).exited
    await Bun.spawn(["rm", "-f", ENGRAM_BIN, ENGRAM_BIN + ".tmp"], { cwd: pluginRoot, stdout: "pipe", stderr: "pipe" }).exited
    const buildProc = Bun.spawn(["go", "build", "-o", ENGRAM_BIN + ".tmp", "./cmd/engram/"], { cwd: pluginRoot, stdout: "pipe", stderr: "pipe" })
    await buildProc.exited
    await Bun.spawn(["mv", ENGRAM_BIN + ".tmp", ENGRAM_BIN], { cwd: pluginRoot, stdout: "pipe", stderr: "pipe" }).exited
  } catch (err) {
    console.error("[engram] binary build failed:", err)
  }
}

async function getReminder(kind: "system" | "session-start" | "user-prompt" | "post-tool"): Promise<string> {
  const proc = Bun.spawn([ENGRAM_BIN, "reminder", kind], { stdout: "pipe", stderr: "pipe" })
  await proc.exited
  return (await proc.stdout.text()).trim()
}

const DEBUG_LOG = path.join(os.homedir(), ".local", "share", "engram", "debug-system-transform.log")

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

export const EngramPlugin: Plugin = async ({ client, $ }) => {
  await ensureBinary()

  return {
    event: async ({ event }) => {
      if (event.type === "session.created") {
        await ensureBinary()
      }
    },

    "experimental.chat.system.transform": async (_input, output) => {
      const before = output.system[0]
      const reminder = await getReminder("system")
      output.system[0] = before + reminder
      logTransform(before, reminder, output.system[0])
    },

    "chat.message": async (_input, output) => {
      const reminder = await getReminder("user-prompt")
      output.parts.push({ type: "text", text: reminder })
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
