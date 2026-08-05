// Concrete typed client for the Competitions bounded context (T9.4's
// CreateCompetition/GetCompetition/ListCompetitions/EnterCompetition/
// CancelCompetition/ListEntriesForCompetition REST surface, plus T9.5's
// GetCompetitionByShareToken). Mirrors socialplayClient.ts's shape exactly.
//
// NOTE: this file only type-checks once `npm run generate:client` has been
// run (it needs `./generated/competitions` to exist — see web/README.md;
// the generator discovers bounded contexts from openapi/pickleball/<ctx>/v1
// rather than hardcoding them, so Competitions is picked up automatically).
// `import type` is erased at build time (no runtime import), so this module
// is safe to import from component code that Vitest exercises on a fresh
// clone before `generate:client` has ever run; only `npm run build`'s
// type-check step needs the generated file present.
import type { paths as CompetitionsPaths } from './generated/competitions'
import { createTypedClient, type CreateTypedClientOptions } from './client'

export function createCompetitionsClient(options: CreateTypedClientOptions = {}) {
  return createTypedClient<CompetitionsPaths>(options)
}

export const competitionsClient = createCompetitionsClient()

/** The shape a component accepts to allow injecting a mock client in tests
 * (e.g. CompetitionCreation.spec.ts) without needing to mock the ambient
 * `fetch` — mirrors SocialPlayClient/FacilitiesClient's identical role. */
export type CompetitionsClient = ReturnType<typeof createCompetitionsClient>
