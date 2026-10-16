// Helpers of the SP17 probe: the markers, a screen test and a reader of the transcript files of a home.
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

export const RECORDER_DIR_OF = path.dirname(fileURLToPath(import.meta.url))
export const MARKERS = {
  status: 'PROBE-STATUS-A',
  toast: 'PROBE-TOAST-B',
  turn: 'PROBE-TURN-C',
  band: 'PROBE-BAND-D',
  log: 'PROBE-LOG-E',
  logEnd: 'PROBE-LOG-END',
}

// screenHas tells whether a captured screen holds the text (a long line may wrap, so the whitespace is dropped).
export function screenHas(screen, text) {
  return screen.replace(/\s+/g, '').includes(text.replace(/\s+/g, ''))
}

// readTranscriptFiles joins every session file under the home's projects directory.
export function readTranscriptFiles(home) {
  const root = path.join(home, '.claude', 'projects')
  if (!fs.existsSync(root)) return ''
  const out = []
  for (const dir of fs.readdirSync(root)) {
    const full = path.join(root, dir)
    if (!fs.statSync(full).isDirectory()) continue
    for (const file of fs.readdirSync(full)) {
      if (file.endsWith('.jsonl')) out.push(fs.readFileSync(path.join(full, file), 'utf8'))
    }
  }
  return out.join('\n')
}
