// The pure parts of the housekeeping job (793b202, 793b201): the index, the scan for pending offers, the reading of an
// offer and the commands that settle it as curate does. Every call on `$` is in register.js.

// a note's file name as engram writes it; a name that could be read as a flag is never passed to a command
const NOTE_NAME = /^[0-9A-Za-z][0-9A-Za-z._-]*\.md$/
export const MAX_OFFERS = 10
export const NEAR_COUNT = 5
const TEXT_LIMIT = 300

export const ingestArgv = () => ['engram', 'ingest', '--auto']

// dataDirectory is where engram keeps its data: $XDG_DATA_HOME/engram, else under the home. The vault and the chunk
// index are in it unless the variable of each says otherwise.
export const dataDirectory = ({ xdg, home }) => `${xdg || `${home}/.local/share`}/engram`

// vaultFrom is the vault engram uses: ENGRAM_VAULT_PATH, else the vault under the data directory.
export const vaultFrom = ({ vault, xdg, home }) => vault || `${dataDirectory({ xdg, home })}/vault`

// pendingArgv lists the notes marked pending, the way curate finds them (the query excludes them by design).
export function pendingArgv(vault) {
  return ['grep', '-rl', '--include=*.md', '-e', '^pending: true$', vault]
}

// parseOffers: the file names of the grep output, the vault's README apart, in name order. The cap of MAX_OFFERS applies
// to the offers settled in a run, not to the names listed: an offer that is skipped does not use it up.
export function parseOffers(output) {
  return output
    .split('\n')
    .map((line) => line.trim().split('/').at(-1))
    .filter((name) => NOTE_NAME.test(name) && name !== 'README.md')
    .sort()
}

// noteName is the name if it is a note file name engram could have written, else null.
const noteName = (name) => (name && NOTE_NAME.test(name) ? name : null)

export const showArgv = (name) => ['engram', 'show', name]

// parseShow reads `engram show`: the exchange hash of the version shown (absent for a note with no xid), the front
// matter fields curation needs, and the note's text.
export function parseShow(output) {
  const parts = output.split(/^---$/m)
  const hash = /^# exchange_hash: (\S+)/m.exec(parts[0])?.[1] ?? null
  const front = parts[1] ?? ''
  const body = (parts.slice(2).join('---').split(/^# outbound links/m)[0] ?? '').replace(/\s+/g, ' ').trim()
  const field = (name) => new RegExp(`^${name}: (.*)$`, 'm').exec(front)?.[1]?.trim().replace(/^["']|["']$/g, '') ?? ''
  return {
    hash,
    kind: field('type'),
    situation: field('situation'),
    pending: /^pending: true$/m.test(front),
    skill: /^skill_hash:/m.test(front),
    forNote: noteName(/^ {2,}for: (\S+)/m.exec(front)?.[1]),
    body,
  }
}

// nearestArgv looks up the existing notes nearest to an offer; the query leaves pending offers out. Free text is
// `--flag=value`: an offer's situation that starts with a dash must not be read as a flag.
export function nearestArgv(offer) {
  return ['engram', 'query', '--limit', String(NEAR_COUNT), `--text=${`${offer.situation}. ${offer.body}`.slice(0, TEXT_LIMIT)}`, `--phrase=${offer.situation.slice(0, TEXT_LIMIT)}`]
}

// settleArgvs are the commands of an outcome, in curate's order: a covered offer reinforces the note and is folded
// into it (`--discard --into` keeps the offer's identity); a near offer is folded away first and its new claim then
// written into the note; an absent one clears its own marker; a rejected one is discarded bare, never `--into`. A
// served offer's exchange hash is passed, as curate does, so that an offer changed since it was read is not settled.
export function settleArgvs(decision, offer, hash) {
  const guard = hash ? [`--expect-hash=${hash}`] : []
  const amend = (target, ...rest) => ['engram', 'amend', '--target', target, ...rest]
  switch (decision.outcome) {
    case 'covered':
      return [amend(decision.target, '--activate'), amend(offer, '--discard', '--into', decision.target, ...guard)]
    case 'near':
      return [amend(offer, '--discard', '--into', decision.target, ...guard), amend(decision.target, ...Object.entries(decision.fields).map(([name, value]) => `--${name}=${value}`))]
    case 'absent':
      return [amend(offer, '--clear-pending', ...guard)]
    default:
      return [amend(offer, '--discard', ...guard)]
  }
}
