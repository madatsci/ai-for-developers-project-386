import { spawnSync } from 'node:child_process'
import { mkdtemp, readFile, readdir, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'

const projectRoot = process.cwd()
const generatedDirectory = join(projectRoot, 'frontend/src/api/generated')
const temporaryDirectory = await mkdtemp(join(tmpdir(), 'call-calendar-sdk-'))

async function collectFiles(directory, relativeDirectory = '') {
  const files = new Map()
  const currentDirectory = join(directory, relativeDirectory)
  const entries = await readdir(currentDirectory, { withFileTypes: true })

  for (const entry of entries) {
    const relativePath = join(relativeDirectory, entry.name)

    if (entry.isDirectory()) {
      const nestedFiles = await collectFiles(directory, relativePath)
      for (const [path, contents] of nestedFiles) {
        files.set(path, contents)
      }
    } else if (entry.isFile()) {
      files.set(relativePath, await readFile(join(directory, relativePath)))
    }
  }

  return files
}

try {
  const npmCommand = process.platform === 'win32' ? 'npm.cmd' : 'npm'
  const generation = spawnSync(
    npmCommand,
    ['--prefix', 'frontend', 'run', 'generate:api', '--', '--silent'],
    {
      cwd: projectRoot,
      env: {
        ...process.env,
        OPENAPI_TS_OUTPUT: temporaryDirectory,
      },
      stdio: 'inherit',
    },
  )

  if (generation.error) {
    throw generation.error
  }
  if (generation.status !== 0) {
    throw new Error(`Hey API exited with status ${generation.status}`)
  }

  const committedFiles = await collectFiles(generatedDirectory)
  const generatedFiles = await collectFiles(temporaryDirectory)
  const paths = new Set([...committedFiles.keys(), ...generatedFiles.keys()])
  const differences = [...paths]
    .filter((path) => {
      const committed = committedFiles.get(path)
      const generated = generatedFiles.get(path)

      return !committed || !generated || !committed.equals(generated)
    })
    .sort()

  if (differences.length > 0) {
    console.error('Generated client SDK is out of date. Run `make generate-client`.')
    for (const path of differences) {
      console.error(`- ${path}`)
    }
    process.exitCode = 1
  }
} finally {
  await rm(temporaryDirectory, { force: true, recursive: true })
}
