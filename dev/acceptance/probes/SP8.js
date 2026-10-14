import fs from 'node:fs'
import path from 'node:path'

const PEER_SETTLE_MS = 15000
const CRON_SETTLE_MS = 85000

export const claim =
  'The non-human prompt origins (task notification, scheduled trigger, peer message) ' +
  'can be produced in an SDK session and fire the hook the mod uses'

function origins(events) {
  return events
    .filter((event) => event.kind === 'prompt.submit')
    .map((event) => ({ origin: event.data.origin?.kind, text: event.data.text }))
}

export async function run(harness) {
  const evidence = {}

  const background = harness.scenario()
  harness.recorder(background, {})
  const backgroundSession = background.startSession()
  await backgroundSession.send(
    'Run the shell command `sleep 3; echo bg-finished` in the background (the Bash tool ' +
      'with run_in_background true), then reply with the single word started.',
  )
  await harness.settle(PEER_SETTLE_MS)
  await backgroundSession.close()
  evidence.taskNotification = {
    prompts: origins(harness.events(background)),
    receives: harness
      .events(background)
      .filter((entry) => entry.kind === 'session.receive')
      .map((entry) => ({ origin: entry.data.origin, text: entry.data.text })),
  }

  const cron = harness.scenario()
  harness.recorder(cron, {})
  const cronSession = cron.startSession()
  await cronSession.send(
    'Use the CronCreate tool to schedule the prompt "scheduled-ok: say the word fired" ' +
      'to run once, one minute from now. Then reply with the single word scheduled.',
  )
  evidence.scheduledTrigger = {
    cronTools: (cronSession.init.tools || []).filter((name) => /cron|schedule/i.test(name)),
    turnsAfterWait: null,
  }
  await harness.settle(CRON_SETTLE_MS)
  evidence.scheduledTrigger.prompts = origins(harness.events(cron))
  evidence.scheduledTrigger.receives = harness
    .events(cron)
    .filter((entry) => entry.kind === 'session.receive')
    .map((entry) => ({ origin: entry.data.origin, text: entry.data.text }))
  await cronSession.close()

  const peers = harness.scenario()
  harness.recorder(peers, {})
  const receiver = peers.startSession()
  await receiver.send('Reply with the word ready.')
  const dirB = path.join(peers.dir, 'probe-events-b')
  fs.mkdirSync(dirB)
  const sender = peers.startSession({
    env: {
      ...peers.env({ withCredentials: true }),
      PROBE_DIR: dirB,
      PROBE_CONFIG: JSON.stringify({ sendTo: receiver.sessionId, sendText: 'peer ping from the probe' }),
    },
  })
  await sender.send('Reply with the word sent.')
  await harness.settle(PEER_SETTLE_MS)
  const sent = JSON.parse(
    fs
      .readdirSync(dirB)
      .filter((name) => name.endsWith('session.send.json'))
      .map((name) => fs.readFileSync(path.join(dirB, name), 'utf8'))[0] ?? 'null',
  )
  evidence.peerMessage = {
    sendResult: sent?.data ?? null,
    receiverPrompts: origins(harness.events(peers)),
    receiverReceives: harness
      .events(peers)
      .filter((entry) => entry.kind === 'session.receive')
      .map((entry) => ({ origin: entry.data.origin, text: entry.data.text })),
  }
  return evidence
}
