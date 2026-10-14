// Package auto is the logic of the engram memory mod (793b368, 793e8m): moment detection, presence and relevance
// filtering, window building, redaction, reconcile and write, backlog selection, curation outcomes, summary counts,
// failure classification and the strict parsing of every model reply. The mod is a thin adapter that relays the
// requests this package returns to a model and calls this package again with the replies (793b197): Run advances one
// job one round. A job is written as straight-line code; to make a round stateless, every effect it has (a command, a
// file, the clock, a model reply) is recorded, and the next round replays the job from its start, answering each
// effect from the record until it reaches the first one not yet performed. The record is the opaque state the mod
// sends back and never reads.
package auto
