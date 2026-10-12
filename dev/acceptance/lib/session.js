// One Claude Code session driven through the Agent SDK with streaming input (793b276): turns are sent one at a
// time, a turn ends at the SDK result message, and a mod that failed to load fails the session at once.
import fs from 'node:fs'
import path from 'node:path'

import { query } from '@anthropic-ai/claude-agent-sdk'

import { redact } from './creds.js'

const NOT_LOADED = /hooks module .*not loaded/i

export class Session {
  constructor({ env, cwd, home, claudePath, creds, model = 'sonnet', plugins = [], resume, maxBudgetUsd = 5 }) {
    this.home = home
    this.creds = creds
    this.messages = []
    this.turns = []
    this.stderr = ''
    this.sessionId = null
    this.init = null
    this.failure = null
    this.waiting = null
    this.inputs = []
    this.inputWaiter = null
    this.closed = false
    this.query = query({
      prompt: this.#inputStream(),
      options: {
        cwd,
        env,
        model,
        plugins,
        resume,
        maxBudgetUsd,
        pathToClaudeCodeExecutable: claudePath,
        permissionMode: 'default',
        permissionPrompts: 'none',
        stderr: (data) => this.#onStderr(data),
      },
    })
    this.reader = this.#read()
  }

  // send submits one user turn and resolves with its summary when the SDK result message arrives.
  send(text) {
    if (this.failure) return Promise.reject(this.failure)
    return new Promise((resolve, reject) => {
      this.waiting = { resolve, reject, startedAt: Date.now() }
      this.#push({ type: 'user', message: { role: 'user', content: text }, parent_tool_use_id: null })
    })
  }

  // close ends the input stream and waits for the SDK to finish.
  async close() {
    this.closed = true
    this.#push(null)
    await this.reader.catch(() => undefined)
  }

  transcriptPath() {
    const projects = path.join(this.home, '.claude', 'projects')
    if (!this.sessionId || !fs.existsSync(projects)) return null
    for (const dir of fs.readdirSync(projects)) {
      const file = path.join(projects, dir, `${this.sessionId}.jsonl`)
      if (fs.existsSync(file)) return file
    }
    return null
  }

  async *#inputStream() {
    for (;;) {
      while (this.inputs.length === 0) {
        await new Promise((resolve) => { this.inputWaiter = resolve })
      }
      const next = this.inputs.shift()
      if (next === null) return
      yield { ...next, session_id: this.sessionId ?? '' }
    }
  }

  #push(item) {
    this.inputs.push(item)
    this.inputWaiter?.()
  }

  #onStderr(data) {
    this.stderr += data
    if (NOT_LOADED.test(this.stderr) && !this.failure) {
      this.#fail(new Error(`a mod did not load (fail fast): ${redact(this.stderr.trim(), this.creds)}`))
    }
  }

  #fail(error) {
    this.failure = error
    this.waiting?.reject(error)
    this.waiting = null
    this.#push(null)
  }

  async #read() {
    try {
      for await (const message of this.query) {
        this.messages.push(message)
        if (message.type === 'system' && message.subtype === 'init') {
          this.sessionId = message.session_id
          this.init = message
        } else if (message.type === 'result' && this.waiting) {
          const turn = summarize(message, this.waiting.startedAt)
          this.turns.push(turn)
          const { resolve } = this.waiting
          this.waiting = null
          resolve(turn)
        }
      }
    } catch (error) {
      if (!this.closed) this.#fail(error)
    }
    if (this.waiting) this.#fail(new Error(`the session ended before the turn completed: ${redact(this.stderr.trim(), this.creds)}`))
  }
}

function summarize(result, startedAt) {
  return {
    text: result.result ?? '',
    isError: Boolean(result.is_error),
    subtype: result.subtype,
    costUsd: result.total_cost_usd ?? 0,
    durationMs: result.duration_ms ?? Date.now() - startedAt,
    wallMs: Date.now() - startedAt,
    usage: result.usage,
    modelUsage: result.modelUsage,
  }
}
