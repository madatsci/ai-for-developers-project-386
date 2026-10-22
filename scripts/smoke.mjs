import assert from 'node:assert/strict'
import { spawn } from 'node:child_process'
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
  assert.deepEqual(
    (await callTypes.json()).items.map(({ name, description, duration_minutes }) => ({
      name,
      description,
      duration_minutes,
    })),
    [
      { name: 'Звонок на 15 минут', description: null, duration_minutes: 15 },
      { name: 'Звонок на 30 минут', description: null, duration_minutes: 30 },
      { name: 'Звонок на 60 минут', description: null, duration_minutes: 60 },
    ],
  )

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
