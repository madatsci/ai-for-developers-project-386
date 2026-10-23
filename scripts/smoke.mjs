import assert from 'node:assert/strict'
import { spawn } from 'node:child_process'
import { randomUUID } from 'node:crypto'
import { once } from 'node:events'
import { setTimeout as delay } from 'node:timers/promises'

// Separate ports allow this check to run alongside the development servers.
const backendPort = process.env.SMOKE_BACKEND_PORT ?? '18080'
const frontendPort = process.env.SMOKE_FRONTEND_PORT ?? '14173'
const backendURL = `http://127.0.0.1:${backendPort}`
const frontendURL = `http://127.0.0.1:${frontendPort}`
const children = []

function start(command, args, options = {}) {
  const child = spawn(command, args, { stdio: 'inherit', ...options })
  child.on('error', (error) => {
    child.startError = error
  })
  children.push(child)
  return child
}

function assertRunning(child) {
  if (child.startError) throw child.startError
  assert.equal(child.exitCode, null, 'Server exited before the smoke test finished')
  assert.equal(child.signalCode, null, 'Server was terminated during the smoke test')
}

async function waitFor(url, child) {
  for (let attempt = 0; attempt < 60; attempt++) {
    assertRunning(child)
    try {
      const response = await fetch(url, { signal: AbortSignal.timeout(1000) })
      if (response.ok) return response
    } catch {
      // A refused connection is expected while the server is starting.
    }
    await delay(250)
  }
  throw new Error(`Server did not become ready: ${url}`)
}

async function stop(child) {
  if (!child.pid || child.exitCode !== null || child.signalCode !== null) return
  const exited = once(child, 'exit')
  child.kill('SIGTERM')
  const timeout = setTimeout(() => child.kill('SIGKILL'), 5000)
  try {
    await exited
  } finally {
    clearTimeout(timeout)
  }
}

for (const signal of ['SIGINT', 'SIGTERM']) {
  process.once(signal, async () => {
    await Promise.all(children.map(stop))
    process.exit(signal === 'SIGINT' ? 130 : 143)
  })
}

try {
  const backend = start('./.bin/api', [], {
    env: { ...process.env, HTTP_ADDR: `127.0.0.1:${backendPort}` },
  })
  const health = await waitFor(`${backendURL}/api/health`, backend)
  assert.deepEqual(await health.json(), { status: 'ok' })

  const callTypes = await fetch(`${backendURL}/api/v1/public/call-types`, {
    signal: AbortSignal.timeout(3000),
  })
  assert.equal(callTypes.status, 200)
  assert.equal(callTypes.headers.get('cache-control'), 'no-store')
  const publicCallTypes = (await callTypes.json()).items
  for (const expected of [
    { name: 'Звонок на 15 минут', description: null, duration_minutes: 15 },
    { name: 'Звонок на 30 минут', description: null, duration_minutes: 30 },
    { name: 'Звонок на 60 минут', description: null, duration_minutes: 60 },
  ]) {
    assert.ok(
      publicCallTypes.some(
        ({ name, description, duration_minutes }) =>
          name === expected.name &&
          description === expected.description &&
          duration_minutes === expected.duration_minutes,
      ),
      `Seeded call type is missing: ${expected.name}`,
    )
  }

  const schedulesResponse = await fetch(`${backendURL}/api/v1/admin/schedules`, {
    signal: AbortSignal.timeout(3000),
  })
  assert.equal(schedulesResponse.status, 200)
  const schedule = (await schedulesResponse.json()).items[0]
  assert.ok(schedule?.id, 'Seeded schedule is missing')

  const smokeName = `Smoke ${Date.now()}`
  const createdResponse = await fetch(`${backendURL}/api/v1/admin/call-types`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      name: smokeName,
      description: 'Production smoke test',
      duration_minutes: 45,
      schedule_id: schedule.id,
    }),
    signal: AbortSignal.timeout(3000),
  })
  assert.equal(createdResponse.status, 201)
  assert.equal(createdResponse.headers.get('etag'), '"1"')
  const createdType = await createdResponse.json()

  const publicCreatedResponse = await fetch(
    `${backendURL}/api/v1/public/call-types/${createdType.slug}`,
    { signal: AbortSignal.timeout(3000) },
  )
  assert.equal(publicCreatedResponse.status, 200)
  assert.equal((await publicCreatedResponse.json()).name, smokeName)

  const disabledResponse = await fetch(`${backendURL}/api/v1/admin/call-types/${createdType.id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', 'If-Match': '"1"' },
    body: JSON.stringify({
      name: smokeName,
      description: null,
      duration_minutes: 45,
      status: 'disabled',
      schedule_id: schedule.id,
    }),
    signal: AbortSignal.timeout(3000),
  })
  assert.equal(disabledResponse.status, 200)
  assert.equal(disabledResponse.headers.get('etag'), '"2"')

  const publicAfterDisable = await fetch(`${backendURL}/api/v1/public/call-types`, {
    signal: AbortSignal.timeout(3000),
  })
  assert.equal(publicAfterDisable.status, 200)
  assert.ok(
    !(await publicAfterDisable.json()).items.some(({ slug }) => slug === createdType.slug),
    'Disabled call type is still present in the public catalog',
  )

  const bookableType = (
    await (
      await fetch(`${backendURL}/api/v1/public/call-types`, {
        signal: AbortSignal.timeout(3000),
      })
    ).json()
  ).items[0]
  const availabilityResponse = await fetch(
    `${backendURL}/api/v1/public/call-types/${bookableType.slug}/availability`,
    { signal: AbortSignal.timeout(3000) },
  )
  assert.equal(availabilityResponse.status, 200)
  const availability = await availabilityResponse.json()
  const smokeSlot = availability.days.flatMap(({ slots }) => slots)[0]
  assert.ok(smokeSlot, 'No slot is available for the calls journal smoke test')
  const bookedResponse = await fetch(`${backendURL}/api/v1/public/calls`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Idempotency-Key': randomUUID(),
    },
    body: JSON.stringify({
      call_type_slug: bookableType.slug,
      call_type_version: availability.call_type.call_type_version,
      starts_at: smokeSlot.starts_at,
      visitor_name: 'Smoke Visitor',
      visitor_email: 'smoke@example.com',
      visitor_notes: 'Calls journal production smoke',
    }),
    signal: AbortSignal.timeout(3000),
  })
  assert.equal(bookedResponse.status, 201)
  const bookedCall = await bookedResponse.json()
  const adminCallsResponse = await fetch(`${backendURL}/api/v1/admin/calls?q=smoke%40example.com`, {
    signal: AbortSignal.timeout(3000),
  })
  assert.equal(adminCallsResponse.status, 200)
  const adminCalls = await adminCallsResponse.json()
  assert.ok(
    adminCalls.items.some(({ id }) => id === bookedCall.id),
    'Booked call is missing from the admin calls journal',
  )

  const deletedResponse = await fetch(`${backendURL}/api/v1/admin/call-types/${createdType.id}`, {
    method: 'DELETE',
    headers: { 'If-Match': '"2"' },
    signal: AbortSignal.timeout(3000),
  })
  assert.equal(deletedResponse.status, 204)

  const frontend = start(
    process.execPath,
    ['node_modules/vite/bin/vite.js', 'preview', '--port', frontendPort, '--strictPort'],
    { cwd: 'frontend', env: { ...process.env, API_PROXY_TARGET: backendURL } },
  )
  const page = await waitFor(frontendURL, frontend)
  const html = await page.text()
  assert.match(html, /<div id="app"><\/div>/)
  assert.match(html, /Календарь звонков/)

  const assets = [...html.matchAll(/(?:src|href)="(\/assets\/[^\"]+)"/g)]
  assert.ok(
    assets.some(([, path]) => path.endsWith('.js')),
    'Built JavaScript is missing',
  )
  assert.ok(
    assets.some(([, path]) => path.endsWith('.css')),
    'Built CSS is missing',
  )
  for (const [, path] of assets) {
    const asset = await fetch(`${frontendURL}${path}`, {
      signal: AbortSignal.timeout(3000),
    })
    assert.equal(asset.status, 200, `Asset is unavailable: ${path}`)
    assert.ok((await asset.text()).length > 0, `Asset is empty: ${path}`)
  }

  const proxied = await fetch(`${frontendURL}/api/health`, {
    signal: AbortSignal.timeout(3000),
  })
  assert.equal(proxied.status, 200)
  assert.deepEqual(await proxied.json(), { status: 'ok' })
  children.forEach(assertRunning)
  console.log('Smoke test passed: frontend assets → API proxy → Go backend → PostgreSQL')
} catch (error) {
  console.error(error)
  process.exitCode = 1
} finally {
  await Promise.all(children.map(stop))
}
