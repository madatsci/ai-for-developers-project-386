import { defineConfig } from '@hey-api/openapi-ts'

const outputPath = process.env.OPENAPI_TS_OUTPUT ?? 'src/api/generated'

export default defineConfig({
  input: '../api/tsp-output/openapi.yaml',
  output: {
    path: outputPath,
    postProcess: [
      {
        command: 'prettier',
        args: ['--config', '.prettierrc.json', '--write', '{{path}}'],
      },
    ],
  },
  plugins: [
    '@hey-api/client-fetch',
    '@hey-api/typescript',
    {
      name: '@hey-api/sdk',
      paramsStructure: 'grouped',
    },
  ],
})
