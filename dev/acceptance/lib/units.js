// Units shared by the harness and the probes, so that no number stands for them unnamed.
export const MS_PER_SECOND = 1000
export const SECONDS_PER_MINUTE = 60
export const SHORT_EXCERPT = 200
export const LONG_EXCERPT = 300

// excerpt is the first characters of a text, for a message that quotes what went wrong.
export function excerpt(text, characters = SHORT_EXCERPT) {
  return String(text).slice(0, characters)
}

export const JSON_INDENT = 2
export const EXIT_FAILURE = 2
export const ARGS_FROM = 2

const TENTH_SECOND_MS = 100
const TENTHS = 10

// tenthsOfSeconds is a duration in milliseconds as seconds with one decimal.
export function tenthsOfSeconds(milliseconds) {
  return Math.round(milliseconds / TENTH_SECOND_MS) / TENTHS
}
