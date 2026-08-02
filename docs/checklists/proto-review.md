# Proto review checklist

Durable review checklist for every `.proto` change, per T6.4 (issue #19) and
`docs/process/t6-sprint-plan.md`'s kickoff note on P1 #13
(`docs/requirements/research-security-compliance.md` §2). Apply this to any
proto touching payment/card flows before it merges — start with Payments,
extend as new contexts add payment-adjacent messages.

## PCI guardrail (mandatory, every proto review)

- [ ] **No raw PAN, card number, CVV/CVC, or track-data field on any
      request or response message, in proto or the REST mapping it
      generates.** Card data is tokenized client-side (Stripe.js/Elements/
      Checkout on web, the equivalent mobile SDKs on Swift/Kotlin) and
      never reaches this backend. This keeps the platform in SAQ A scope
      (the lightest PCI-DSS self-assessment tier) — accepting even one raw
      card field anywhere in the contract, even a field nobody populates
      yet, changes that scope.
- [ ] Payment-processor references crossing the wire (e.g. a Stripe
      `PaymentIntent` id) are opaque strings, never a nested object that
      could grow a card-shaped field later. Mirrors
      `port.PaymentProcessor`'s own processor-agnostic shape
      (`internal/payments/port/payment_processor.go`).
- [ ] If a future proto needs to represent "the client tokenized a card
      client-side, here is the token", the field is named/documented as an
      opaque processor token (e.g. `payment_method_token`), never anything
      implying it carries real card data.

## Review record

Every PR that adds or changes a proto touching Payments must state in its
description: which proto file(s) were reviewed against this checklist, and
confirmation that the PCI guardrail above holds. See the T6.4 PR
description for the first application of this checklist, against
`proto/pickleball/payments/v1/payments.proto`.
