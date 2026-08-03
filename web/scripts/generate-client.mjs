#!/usr/bin/env node
// Runs `make generate` at the repo root (buf + sqlc -> internal/gen and
// openapi/, per CLAUDE.md), then feeds each bounded context's
// openapi/pickleball/<context>/v1/*.swagger.json through `openapi-typescript`
// to produce typed `paths`/`components` modules under
// web/src/api/generated/ (gitignored — see web/README.md).
//
// Actual on-disk filenames were confirmed by inspection (T7.1 instructions
// say not to guess): `openapi/pickleball/{booking,payments,socialplay}/v1/
// {booking,payments,socialplay}.swagger.json`, i.e. one
// `<context>/v1/<context>.swagger.json` per bounded context. This script
// discovers that shape rather than hardcoding the three current context
// names, so a future bounded context's swagger.json is picked up
// automatically once `make generate` produces it.
//
// Extra wrinkle discovered by actually running this (not assumed):
// `protoc-gen-openapiv2` (buf.gen.yaml's OpenAPI plugin) emits **Swagger
// 2.0**, but `openapi-typescript` 7.x only accepts OpenAPI 3.x and fails
// fast ("Unsupported Swagger version: 2.x") on a raw swagger.json. So each
// file is piped through `swagger2openapi` first to upconvert 2.0 -> 3.0
// in-memory, and the *converted* document is what's handed to
// openapi-typescript. See web/README.md's "Typed client generation"
// section for the full why-this-shape rationale.
import { execFileSync } from 'node:child_process'
import { existsSync, mkdirSync, readdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import path from 'node:path'
import { convertObj } from 'swagger2openapi'

const webRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const repoRoot = path.resolve(webRoot, '..')
const openapiPickleballDir = path.join(repoRoot, 'openapi', 'pickleball')
const outDir = path.join(webRoot, 'src', 'api', 'generated')

function run(command, args, cwd) {
  console.log(`[generate:client] ${command} ${args.join(' ')}`)
  execFileSync(command, args, { cwd, stdio: 'inherit' })
}

function toPascalCase(name) {
  return name.replace(/(^|[-_])([a-z])/g, (_match, _sep, ch) => ch.toUpperCase())
}

function convertSwagger2ToOpenApi3(swagger2Doc) {
  return new Promise((resolve, reject) => {
    convertObj(swagger2Doc, { patch: true, warnOnly: true }, (err, options) => {
      if (err) reject(err)
      else resolve(options.openapi)
    })
  })
}

console.log('[generate:client] Step 1/2: `make generate` at repo root (buf + sqlc)...')
try {
  run('make', ['generate'], repoRoot)
} catch (err) {
  console.error(
    '[generate:client] `make generate` failed. This step needs buf + sqlc (and the local ' +
      'protoc-gen-* plugins) on PATH — see CLAUDE.md\'s Gotchas section and ' +
      "docs/process/t7-sprint-plan.md's Toolchain note. If this environment doesn't have " +
      'them installed, install buf/sqlc (`go install github.com/bufbuild/buf/cmd/buf@latest`, ' +
      '`go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`, plus the four protoc-gen-* plugins ' +
      "CLAUDE.md names) and ensure $(go env GOPATH)/bin is on PATH, then re-run " +
      '`npm run generate:client`.',
  )
  throw err
}

if (!existsSync(openapiPickleballDir)) {
  console.error(
    `[generate:client] Expected OpenAPI output at ${openapiPickleballDir} but it doesn't exist ` +
      "after `make generate` ran without error — check protoc-gen-openapiv2 is on PATH and " +
      'buf.gen.yaml is wired to it.',
  )
  process.exit(1)
}

console.log('[generate:client] Step 2/2: openapi-typescript per bounded context -> web/src/api/generated/...')
rmSync(outDir, { recursive: true, force: true })
mkdirSync(outDir, { recursive: true })

const contexts = readdirSync(openapiPickleballDir, { withFileTypes: true })
  .filter((entry) => entry.isDirectory())
  .map((entry) => entry.name)
  .sort()

const generatedContexts = []

for (const context of contexts) {
  const v1Dir = path.join(openapiPickleballDir, context, 'v1')
  if (!existsSync(v1Dir)) continue

  const swaggerFile = readdirSync(v1Dir).find((f) => f.endsWith('.swagger.json'))
  if (!swaggerFile) continue

  const inputPath = path.join(v1Dir, swaggerFile)
  const outputPath = path.join(outDir, `${context}.d.ts`)

  console.log(`[generate:client] converting Swagger 2.0 -> OpenAPI 3.0: ${path.relative(repoRoot, inputPath)}`)
  const swagger2Doc = JSON.parse(readFileSync(inputPath, 'utf-8'))
  const openapi3Doc = await convertSwagger2ToOpenApi3(swagger2Doc)
  const convertedPath = path.join(outDir, `.${context}.openapi3.json`)
  writeFileSync(convertedPath, JSON.stringify(openapi3Doc))

  run('npx', ['--yes', 'openapi-typescript', convertedPath, '-o', outputPath], webRoot)
  rmSync(convertedPath)
  generatedContexts.push(context)
}

if (generatedContexts.length === 0) {
  console.error(
    `[generate:client] No *.swagger.json files found under ${openapiPickleballDir}/<context>/v1/ ` +
      '— nothing to generate types from.',
  )
  process.exit(1)
}

const indexContents = `// AUTO-GENERATED by \`npm run generate:client\`. Do not hand-edit — see web/README.md.
${generatedContexts.map((ctx) => `export type * as ${toPascalCase(ctx)} from './${ctx}'`).join('\n')}
`
writeFileSync(path.join(outDir, 'index.ts'), indexContents)

console.log(
  `[generate:client] Done. Generated ${generatedContexts.length} module(s): ${generatedContexts.join(', ')} -> ${path.relative(webRoot, outDir)}/`,
)
