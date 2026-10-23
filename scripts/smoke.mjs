import assert from 'node:assert/strict'
import { setTimeout as delay } from 'node:timers/promises'

const configuredBaseURL = process.env.SMOKE_BASE_URL ?? 'http://127.0.0.1:18080'
const parsedBaseURL = new URL(configuredBaseURL)
const baseURL = parsedBaseURL.origin

assert.equal(parsedBaseURL.username, '', 'Smoke URL must not contain credentials')
assert.equal(parsedBaseURL.password, '', 'Smoke URL must not contain credentials')
if (process.env.SMOKE_REQUIRE_HTTPS === '1') {
  assert.equal(parsedBaseURL.protocol, 'https:', 'Remote smoke requires an HTTPS URL')
}

const smokeCall = {
  idempotencyKey: 'ca11da7a-9b3d-4f4a-8cc7-e67a669cc032',
  visitorName: 'Посетитель remote smoke',
  visitorEmail: 'remote-smoke@example.com',
  visitorNotes: 'Проверка опубликованного production-контура',
}

function isSmokeCall(call) {
  return (
    call.visitor_name === smokeCall.visitorName &&
    call.visitor_email === smokeCall.visitorEmail &&
    call.visitor_notes === smokeCall.visitorNotes
  )
}

async function findPersistedSmokeCall() {
  for (const period of ['upcoming', 'past']) {
    const response = await fetch(
      `${baseURL}/api/v1/admin/calls?period=${period}&q=${encodeURIComponent(smokeCall.visitorEmail)}`,
      { signal: AbortSignal.timeout(3000) },
    )
    assert.equal(response.status, 200)
    const match = (await response.json()).items.find(isSmokeCall)
    if (match) return match
  }
  return undefined
}

async function waitForPersistedSmokeCall() {
  for (let attempt = 0; attempt < 8; attempt++) {
    const call = await findPersistedSmokeCall()
    if (call) return call
    await delay(250)
  }
  return undefined
}

async function waitFor(url) {
  for (let attempt = 0; attempt < 60; attempt++) {
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

try {
  const health = await waitFor(`${baseURL}/api/health`)
  assert.deepEqual(await health.json(), { status: 'ok' })

  const callTypes = await fetch(`${baseURL}/api/v1/public/call-types`, {
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

  const seededCallType = publicCallTypes.find(
    ({ name, duration_minutes }) => name === 'Звонок на 15 минут' && duration_minutes === 15,
  )
  assert.ok(seededCallType, 'Seeded 15-minute call type is missing')

  let persistedSmokeCall = await findPersistedSmokeCall()
  if (!persistedSmokeCall) {
    const availabilityResponse = await fetch(
      `${baseURL}/api/v1/public/call-types/${seededCallType.slug}/availability`,
      { signal: AbortSignal.timeout(3000) },
    )
    assert.equal(availabilityResponse.status, 200)
    const availability = await availabilityResponse.json()
    const availableSlot = availability.days.flatMap(({ slots }) => slots)[0]
    assert.ok(availableSlot?.starts_at, 'Seeded call type has no available slot')

    const bookingResponse = await fetch(`${baseURL}/api/v1/public/calls`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Idempotency-Key': smokeCall.idempotencyKey,
      },
      body: JSON.stringify({
        call_type_slug: availability.call_type.slug,
        call_type_version: availability.call_type.call_type_version,
        starts_at: availableSlot.starts_at,
        visitor_name: smokeCall.visitorName,
        visitor_email: smokeCall.visitorEmail,
        visitor_notes: smokeCall.visitorNotes,
      }),
      signal: AbortSignal.timeout(3000),
    })
    assert.equal(bookingResponse.status, 201)
    const createdCall = await bookingResponse.json()
    assert.equal(createdCall.visitor_email, smokeCall.visitorEmail)

    persistedSmokeCall = await waitForPersistedSmokeCall()
    assert.equal(
      persistedSmokeCall?.id,
      createdCall.id,
      'Created smoke call is missing from the organizer journal',
    )
  }
  assert.ok(persistedSmokeCall, 'Persisted smoke call is missing from the organizer journal')

  const schedulesResponse = await fetch(`${baseURL}/api/v1/admin/schedules`, {
    signal: AbortSignal.timeout(3000),
  })
  assert.equal(schedulesResponse.status, 200)
  const schedule = (await schedulesResponse.json()).items[0]
  assert.ok(schedule?.id, 'Seeded schedule is missing')

  const smokeName = `Smoke ${Date.now()}`
  const createdResponse = await fetch(`${baseURL}/api/v1/admin/call-types`, {
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
    `${baseURL}/api/v1/public/call-types/${createdType.slug}`,
    { signal: AbortSignal.timeout(3000) },
  )
  assert.equal(publicCreatedResponse.status, 200)
  assert.equal((await publicCreatedResponse.json()).name, smokeName)

  const disabledResponse = await fetch(`${baseURL}/api/v1/admin/call-types/${createdType.id}`, {
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

  const publicAfterDisable = await fetch(`${baseURL}/api/v1/public/call-types`, {
    signal: AbortSignal.timeout(3000),
  })
  assert.equal(publicAfterDisable.status, 200)
  assert.ok(
    !(await publicAfterDisable.json()).items.some(({ slug }) => slug === createdType.slug),
    'Disabled call type is still present in the public catalog',
  )

  const adminCallsResponse = await fetch(`${baseURL}/api/v1/admin/calls`, {
    signal: AbortSignal.timeout(3000),
  })
  assert.equal(adminCallsResponse.status, 200)
  const adminCalls = await adminCallsResponse.json()
  assert.equal(adminCalls.timezone, 'Europe/Moscow')
  assert.equal(typeof adminCalls.as_of, 'string')
  assert.ok(Array.isArray(adminCalls.items), 'Admin calls journal items are missing')

  const deletedResponse = await fetch(`${baseURL}/api/v1/admin/call-types/${createdType.id}`, {
    method: 'DELETE',
    headers: { 'If-Match': '"2"' },
    signal: AbortSignal.timeout(3000),
  })
  assert.equal(deletedResponse.status, 204)

  const page = await waitFor(baseURL)
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
    const asset = await fetch(`${baseURL}${path}`, {
      signal: AbortSignal.timeout(3000),
    })
    assert.equal(asset.status, 200, `Asset is unavailable: ${path}`)
    assert.ok((await asset.text()).length > 0, `Asset is empty: ${path}`)
  }

  const proxied = await fetch(`${baseURL}/api/health`, {
    signal: AbortSignal.timeout(3000),
  })
  assert.equal(proxied.status, 200)
  assert.deepEqual(await proxied.json(), { status: 'ok' })
  console.log(
    'Smoke test passed: frontend assets → same-origin Go API → PostgreSQL → persisted call',
  )
} catch (error) {
  console.error(error)
  process.exitCode = 1
}
