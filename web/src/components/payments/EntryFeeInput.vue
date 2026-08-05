<script setup lang="ts">
// The Host's entry-fee field (T9.2), extracted from GameCreation.vue by
// T9.6 so the Competition creation flow REUSES it rather than growing a
// second, subtly-different money input.
//
// T9.6's Instructions say "reuse T9.2's fee input component". It wasn't a
// component yet — it was inline markup inside GameCreation.vue — so this is
// that markup lifted out verbatim (same element ids, same `data-testid`s,
// same aria wiring), with `GameCreation.vue` rewired onto it. Its existing
// spec (`GameCreation.spec.ts`'s "entry fee (T9.2)" block, which asserts on
// `#entry-fee-hint`, `#entry-fee-error`, `[data-testid="entry-fee-input"]`
// and the aria-describedby switch) is what guards the extraction: it passes
// unchanged, which is the point.
//
// What the component owns, so both callers get it identically:
//   - dollars-in / integer-minor-units-out conversion (a Host types
//     dollars because that is what they think in; the wire wants cents);
//   - the client-side mirror of `domain.Money.Validate`'s negative-amount
//     rule, shown live rather than only on submit;
//   - WCAG 3.3.1 error text next to the field, and WCAG 1.4.1's "the free
//     state is stated in words" hint.
//
// What it does NOT own: the server's authoritative rejection. Each caller
// still maps its own context's `invalid money amount` sentinel back onto
// this field via the `serverError` prop (competitions and socialplay have
// separate domains and separate error strings — CLAUDE.md rules 2/3).
import { computed } from 'vue'
import { entryFeeLabel } from '../../models/payment'

const props = withDefaults(
  defineProps<{
    /** The typed amount, in DOLLARS. `v-model`-able. A `<input type=
     * "number">` hands back a string in the browser but a number when set
     * programmatically (and by @vue/test-utils' `setValue`), hence the
     * union. */
    modelValue: string | number
    /** Field label. */
    label?: string
    /** The word for what is being priced, used in the "leave at 0" hint —
     * "game" or "competition". */
    subject?: string
    /** A server-side rejection routed back onto this field by the caller
     * (e.g. domain `ErrInvalidMoney`). Shown in the same message slot as
     * the live client-side check. */
    serverError?: string
  }>(),
  { label: 'Entry fee per player', subject: 'game', serverError: '' },
)

const emit = defineEmits<{
  (event: 'update:modelValue', value: string | number): void
  /** Fires whenever validity changes, so a parent can gate its "Next"
   * button without re-deriving the rule. */
  (event: 'validity', valid: boolean): void
}>()

/**
 * The typed fee in integer minor units. Rounds rather than truncates so a
 * Host typing "12.005" cannot silently lose a cent, and returns NaN for
 * input that isn't a number at all so the caller can tell "not a valid
 * amount" apart from "zero".
 */
const cents = computed(() => {
  const raw = String(props.modelValue ?? '').trim()
  if (raw === '') return 0
  const parsed = Number(raw)
  if (Number.isNaN(parsed)) return Number.NaN
  return Math.round(parsed * 100)
})

/** Client-side mirror of `domain.Money.Validate`'s negative-amount rule.
 * The server is still authoritative; this only gets the message in front of
 * the Host sooner. */
const valid = computed(() => !Number.isNaN(cents.value) && cents.value >= 0)

/**
 * The error actually shown next to the field. Two sources, one message
 * slot: the LIVE client-side check (so a Host who types a negative amount
 * is told why immediately — leaving them staring at a disabled "Next" with
 * no explanation would fail WCAG 3.3.1 just as surely as showing nothing at
 * all), and the server's own rejection routed here by the caller.
 */
const error = computed(() => {
  if (!valid.value) return "Entry fee can't be negative."
  return props.serverError ?? ''
})

function onInput(event: Event) {
  emit('update:modelValue', (event.target as HTMLInputElement).value)
  // Emitted after the model change so a parent reading it sees the value
  // and its validity agree.
  emit('validity', valid.value)
}

/** Exposed so a parent can submit the converted amount and gate on
 * validity without duplicating either rule. */
defineExpose({ cents, valid })
</script>

<template>
  <div class="entry-fee">
    <label for="entry-fee">{{ label }}</label>
    <input
      id="entry-fee"
      data-testid="entry-fee-input"
      type="number"
      min="0"
      step="0.01"
      inputmode="decimal"
      :value="modelValue"
      :aria-invalid="error ? 'true' : undefined"
      :aria-describedby="error ? 'entry-fee-error' : 'entry-fee-hint'"
      @input="onInput"
    />
    <!-- WCAG 1.4.1: the "this is free" state is stated in text, not implied
         by an empty or differently-styled field. -->
    <p id="entry-fee-hint" class="entry-fee__hint">
      Leave at 0 to make this {{ subject }} free.<template v-if="valid">
        Players will see “{{ entryFeeLabel(cents) }}”.</template>
    </p>
    <!-- WCAG 3.3.1 Error Identification: the validation error is visible
         text next to the field, never colour alone. -->
    <p v-if="error" id="entry-fee-error" class="entry-fee__error" role="alert">
      {{ error }}
    </p>
  </div>
</template>

<style scoped>
.entry-fee {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  font-size: var(--font-size-sm);
}

.entry-fee input {
  font: inherit;
  /* >=44px touch target (Apple HIG, stricter than WCAG 2.5.8's 24px). */
  min-height: 44px;
  padding: 0.5rem 0.65rem;
  border-radius: var(--radius-sm);
  border: 1px solid var(--hs-border);
  background: var(--paper);
  color: var(--ink);
}

.entry-fee__hint {
  color: var(--ink-soft);
  font-size: var(--font-size-sm);
  margin: 0;
}

.entry-fee__error {
  color: var(--ink-warning);
  font-size: var(--font-size-xs);
  margin: 0;
}
</style>
