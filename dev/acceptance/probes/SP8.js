import fs from 'node:fs'
import path from 'node:path'

export const claim = 'The non-human prompt origins (task notification, scheduled trigger, peer message) can be produced in an SDK session and fire the hook the mod uses'

function origins(events) {
  return events.filter((event) => event.kind === 'prompt.submit').map((event) => ({ origin: event.data.origin?.kind, text: event.data.text }))
}

export async function run(h) {
  const evidence = {}

  const background = h.scenario()
  h.recorder(background, {})
  const a = background.startSession()
  await a.send('Run the shell command `sleep 3; echo bg-finished` in the background (the Bash tool with run_in_background true), then reply with the single word started.')
  await h.settle(15000)
  await a.close()
  evidence.taskNotification = { prompts: origins(h.events(background)), receives: h.events(background).filter((e) => e.kind === 'session.receive').map((e) => ({ origin: e.data.origin, text: e.data.text })) }

  const cron = h.scenario()
  h.recorder(cron, {})
  const c = cron.startSession()
  const toolNames = c.init?.tools
  await c.send('Use the CronCreate tool to schedule the prompt "scheduled-ok: say the word fired" to run once, one minute from now. Then reply with the single word scheduled.')
  evidence.scheduledTrigger = { cronTools: (c.init.tools || []).filter((name) => /cron|schedule/i.test(name)), turnsAfterWait: null }
  await h.settle(85000)
  evidence.scheduledTrigger.prompts = origins(h.events(cron))
  evidence.scheduledTrigger.receives = h.events(cron).filter((e) => e.kind === 'session.receive').map((e) => ({ origin: e.data.origin, text: e.data.text }))
  await c.close()

  const peers = h.scenario()
  const dirA = h.recorder(peers, {})
  const receiver = peers.startSession()
  await receiver.send('Reply with the word ready.')
  const dirB = path.join(peers.dir, 'probe-events-b')
  fs.mkdirSync(dirB)
  const sender = peers.startSession({
    env: { ...peers.env({ withCredentials: true }), PROBE_DIR: dirB, PROBE_CONFIG: JSON.stringify({ sendTo: receiver.sessionId, sendText: 'peer ping from the probe' }) },
  })
  await sender.send('Reply with the word sent.')
  await h.settle(15000)
  const sent = JSON.parse(fs.readdirSync(dirB).filter((f) => f.endsWith('session.send.json')).map((f) => fs.readFileSync(path.join(dirB, f), 'utf8'))[0] ?? 'null')
  evidence.peerMessage = { sendResult: sent?.data ?? null, receiverPrompts: origins(h.events(peers)), receiverReceives: h.events(peers).filter((e) => e.kind === 'session.receive').map((e) => ({ origin: e.data.origin, text: e.data.text })) }
  return evidence
}
