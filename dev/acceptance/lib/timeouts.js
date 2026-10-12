// Cucumber's default of 5 seconds per step or hook is shorter than one turn with its retries (3 attempts of a
// session that starts, answers and closes, SP11/SP13). A step is bounded by this instead.
export const STEP_TIMEOUT_MS = 10 * 60 * 1000
