import fs from 'node:fs'
import { pathToFileURL } from 'node:url'

import { evalStepFiles } from '../lib/eval-dir.js'

import './given.js'
import './when.js'
import './then.js'
import './housekeeping.js'
import './backlog.js'

// The steps of the one-time experiments (H1: 793b343) live under ENGRAM_EVAL_DIR, not in the repository; where they are
// absent the 793d17 scenarios have undefined steps, which is the intended state once the experiment is decided.
for (const file of evalStepFiles(process.env, fs.existsSync)) await import(pathToFileURL(file).href)
