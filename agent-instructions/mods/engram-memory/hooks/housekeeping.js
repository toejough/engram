// The pure parts of the housekeeping job (793b202, 793b201): the index, the scan for pending offers, the reading of an
// offer and the commands that settle it as curate does. Every call on `$` is in register.js.

// a note's file name as engram writes it; a name that could be read as a flag is never passed to a command
const NOTE_NAME = /^[0-9A-Za-z][0-9A-Za-z._-]*\.md$/
export const MAX_OFFERS = 10
export const NEAR_COUNT = 5
const TEXT_LIMIT = 300

export const ingestArgv = () => ['engram', 'ingest', '--auto']

// vaultFrom is the vault engram uses: ENGRAM_VAULT_PATH, else $XDG_DATA_HOME/engram/vault, else under the home.
export function vaultFrom({ vault, xdg, home }) {
  if (vault) return vault
  return `${xdg || `${home}/.local/share`}/engram/vault`
}

// pendingArgv lists the notes marked pending, the way curate finds them (the query excludes them by design).
export function pendingArgv(vault) {
  return ['grep', '-rl', '--include=*.md', '-e', '^pending: true$', vault]
}

// parseOffers: the file names of the grep output, the vault's README apart, at most MAX_OFFERS, in name order.
export function parseOffers(output) {
  return output
    .split('\n')
    .map((line) => line.trim().split('/').at(-1))
    .filter((name) => NOTE_NAME.test(name) && name !== 'README.md')
    .sort()
    .slice(0, MAX_OFFERS)
}

export const showArgv = (name) => ['engram', 'show', name]

// parseShow reads `engram show`: the exchange hash of the version shown (absent for a note with no xid), the front
// matter fields curation needs, and the note's text.
export function parseShow(output) {
  const hash = /^# exchange_hash: (\S+)/m.exec(output)?.[1] ?? null
  const parts = output.split(/^---$/m)
  const front = parts[1] ?? ''
  const body = (parts.slice(2).join('---').split(/^# outbound links/m)[0] ?? '').replace(/\s+/g, ' ').trim()
  const field = (name) => new RegExp(`^${name}: (.*)$`, 'm').exec(front)?.[1]?.trim().replace(/^["']|["']$/g, '') ?? ''
  return {
    hash,
    kind: field('type'),
    situation: field('situation'),
    pending: /^pending: true$/m.test(front),
    skill: /^skill_hash:/m.test(front),
    forNote: /^ {2,}for: (\S+)/m.exec(front)?.[1] ?? null,
    body,
  }
}

// nearestArgv looks up the existing notes nearest to an offer; the query leaves pending offers out.
export function nearestArgv(offer) {
  return ['engram', 'query', '--limit', String(NEAR_COUNT), '--text', `${offer.situation}. ${offer.body}`.slice(0, TEXT_LIMIT), '--phrase', offer.situation.slice(0, TEXT_LIMIT)]
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
