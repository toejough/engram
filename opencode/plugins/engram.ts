import type { Plugin } from "@opencode-ai/plugin"
import { tool } from "@opencode-ai/plugin"
import * as fs from "fs"
import * as path from "path"
import * as os from "os"

const ENGRAM_BIN = path.join(os.homedir(), ".local", "share", "engram", "bin", "engram")
const SYMLINK_PATH = path.join(os.homedir(), ".local", "bin", "engram")

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

async function buildIfNeeded($: typeof Bun.$): Promise<boolean> {
  const needsBuild =
    !fs.existsSync(ENGRAM_BIN) ||
    !fs.statSync(ENGRAM_BIN).isFile()

  if (!needsBuild) {
    const pluginRoot = findPluginRoot()
    if (pluginRoot) {
      try {
        const { execSync } = await import("child_process")
        const result = execSync(
          `find "${pluginRoot}" -name '*.go' -newer "${ENGRAM_BIN}" -print -quit`,
          { encoding: "utf8" },
        )
        if (result.trim().length > 0) {
          return true
        }
      } catch {
        // find failed, assume no build needed
      }
    }
    return false
  }

  return needsBuild
}

async function ensureBinary($: typeof Bun.$): Promise<void> {
  const shouldBuild = await buildIfNeeded($)
  if (!shouldBuild) return

  const pluginRoot = findPluginRoot()
  if (!pluginRoot) return

  const binDir = path.dirname(ENGRAM_BIN)
  try {
    await $`mkdir -p ${binDir}`.cwd(pluginRoot)
    await $`rm -f ${ENGRAM_BIN} ${ENGRAM_BIN}.tmp`.cwd(pluginRoot)
    await $`go build -o ${ENGRAM_BIN}.tmp ./cmd/engram/`.cwd(pluginRoot)
    await $`mv ${ENGRAM_BIN}.tmp ${ENGRAM_BIN}`.cwd(pluginRoot)

    const localBin = path.dirname(SYMLINK_PATH)
    await $`mkdir -p ${localBin}`
    await $`ln -sf ${ENGRAM_BIN} ${SYMLINK_PATH}`
  } catch (err) {
    // Build failed — log silently, user can build manually
    console.error("[engram] binary build failed:", err)
  }
}

function runEngramCommand(
  args: string[],
): (ctx: { directory: string; worktree?: string }) => Promise<string> {
  return async () => {
    try {
      const result = await Bun.$`${ENGRAM_BIN} ${args}`
      return result.text().trim()
    } catch (err: unknown) {
      if (err instanceof Error && "stderr" in err) {
        return `engram error: ${(err as { stderr?: string }).stderr || err.message}`
      }
      return `engram error: ${err instanceof Error ? err.message : String(err)}`
    }
  }
}

const seenSessions = new Set<string>()

export const EngramPlugin: Plugin = async ({ $ }) => {
  return {
    event: async ({ event }) => {
      if (event.type === "session.created") {
        ensureBinary($)
      }
    },

    "experimental.chat.system.transform": async (_input, output) => {
      const sessionID = (_input as { sessionID?: string })?.sessionID
      if (sessionID && !seenSessions.has(sessionID)) {
        seenSessions.add(sessionID)
        const reminder =
          "## Engram Memory\n" +
          "Use /prepare before starting new work. Use /learn after completing work to capture lessons.\n" +
          "Use /recall to load previous session context. Use /remember to save something explicitly."
        output.system[0] = output.system[0] + "\n\n" + reminder
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
          cmdArgs.push("--no-external-sources")
          const result = await Bun.$`${ENGRAM_BIN} ${cmdArgs}`
          return result.text().trim()
        },
      }),

      engram_learn_feedback: tool({
        description: "Learn from behavioral feedback using SBIA format",
        args: {
          situation: tool.schema.string().describe("Context when this applies"),
          behavior: tool.schema.string().describe("Observed behavior"),
          impact: tool.schema.string().describe("Impact of the behavior"),
          action: tool.schema.string().describe("Recommended action"),
          source: tool.schema.string().optional().describe("Human or agent (default: human)"),
        },
        async execute(args) {
          const result = await Bun.$`${ENGRAM_BIN} learn feedback \
            --situation ${args.situation} \
            --behavior ${args.behavior} \
            --impact ${args.impact} \
            --action ${args.action} \
            --source ${args.source || "human"}`
          return result.text().trim()
        },
      }),

      engram_learn_fact: tool({
        description: "Learn a factual statement using SPO format",
        args: {
          situation: tool.schema.string().describe("Context when this applies"),
          subject: tool.schema.string().describe("Subject of the fact"),
          predicate: tool.schema.string().describe("Relationship or verb"),
          object: tool.schema.string().describe("Object of the fact"),
          source: tool.schema.string().optional().describe("Human or agent (default: human)"),
        },
        async execute(args) {
          const result = await Bun.$`${ENGRAM_BIN} learn fact \
            --situation ${args.situation} \
            --subject ${args.subject} \
            --predicate ${args.predicate} \
            --object ${args.object} \
            --source ${args.source || "human"}`
          return result.text().trim()
        },
      }),

      engram_show: tool({
        description: "Display full memory details",
        args: {
          name: tool.schema.string().describe("Memory slug to display"),
        },
        async execute(args) {
          const result = await Bun.$`${ENGRAM_BIN} show --name ${args.name}`
          return result.text().trim()
        },
      }),

      engram_list: tool({
        description: "List all memories with type, name, and situation",
        args: {},
        async execute() {
          const result = await Bun.$`${ENGRAM_BIN} list`
          return result.text().trim()
        },
      }),
    },
  }
}
