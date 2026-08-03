// Concrete typed client for the Payments bounded context. T8.10's Payments
// UI (`CreateOnlinePayment`, `ConfirmOnlinePayment`, `RecordOfflinePayment`)
// is the first consumer — mirrors bookingClient.ts/facilitiesClient.ts/
// socialplayClient.ts's shape exactly.
//
// NOTE: this file only type-checks once `npm run generate:client` has been
// run (it needs `./generated/payments` to exist — see web/README.md).
// `import type` is erased at build time (no runtime import), so this module
// is safe to import from component code that Vitest exercises on a fresh
// clone before `generate:client` has ever run; only `npm run build`'s
// type-check step needs the generated file present.
import type { paths as PaymentsPaths } from './generated/payments'
import { createTypedClient, type CreateTypedClientOptions } from './client'

export function createPaymentsClient(options: CreateTypedClientOptions = {}) {
  return createTypedClient<PaymentsPaths>(options)
}

export const paymentsClient = createPaymentsClient()

/** The shape a component accepts to allow injecting a mock client in tests
 * without needing to mock the ambient `fetch` — mirrors BookingClient/
 * SocialPlayClient's identical role. */
export type PaymentsClient = ReturnType<typeof createPaymentsClient>
