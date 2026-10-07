<script setup lang="ts">
/**
 * Dev-only sign-in — T65.1.
 *
 * The server has enforced authentication on 31 RPCs since T55.1 and the client
 * could not send a token until now, so every write path in this UI answered
 * `Unauthenticated`. This view is the smallest thing that fixes that without
 * inventing an identity provider: paste a token minted by `make dev-token`.
 *
 * **Why pasting is the right shape here, not a shortcut.** `dev/auth/` holds a
 * committed keypair precisely so a human can mint a local token
 * (`TOKEN=$(make -s dev-token)`); the keypair is public and therefore
 * worthless, and `dev/auth/README.md` says never to point a deployment at it.
 * A real redirect flow needs an identity provider this environment cannot
 * provision (ADR-0013, #145, #137).
 *
 * **Why the user id is asked for separately.** The server resolves the token's
 * `sub` to a `User.ID` itself (ADR-0014's seam), and **no RPC tells a client
 * its own user id** — Identity exposes `CreateUser`, `GetUser` and
 * `UpdateSelfReportedLevel`, and no `GetMe`. Domain fields that name a player
 * or an owner need that id, so it is captured here until a `GetMe` exists.
 * That gap is filed, not hidden.
 */
import { ref } from 'vue'
import { useRouter } from 'vue-router'

import {
  authToken,
  authUserId,
  clearAuthRejected,
  signIn,
  signOut,
} from '../state/authSession'

const router = useRouter()
const token = ref('')
const userId = ref(authUserId.value ?? '')
const error = ref<string | null>(null)

function submit(): void {
  const trimmed = token.value.trim()
  if (trimmed === '') {
    error.value = 'Paste a token first — run `make dev-token` to mint one.'
    return
  }
  // A JWT is three dot-separated segments. This is a typo check, not
  // validation: the server verifies the signature, and a client that pretended
  // to validate would be claiming an authority it does not have.
  if (trimmed.split('.').length !== 3) {
    error.value = 'That does not look like a JWT (expected three dot-separated parts).'
    return
  }
  error.value = null
  signIn(trimmed, userId.value)
  clearAuthRejected()
  void router.push('/facilities')
}

function clear(): void {
  signOut()
  token.value = ''
  userId.value = ''
  error.value = null
}
</script>

<template>
  <section class="dev-sign-in">
    <h1>Sign in (development)</h1>

    <p class="dev-sign-in__intro">
      This screen exists so the browser can send an <code>Authorization</code>
      header. Mint a token in the repository root and paste it below:
    </p>
    <pre class="dev-sign-in__cmd"><code>TOKEN=$(make -s dev-token); echo "$TOKEN"</code></pre>

    <p class="dev-sign-in__hint">
      <strong>A freshly minted token has no user yet.</strong> The server
      resolves your token's subject to an <code>identity_users</code> row, and
      a brand-new subject has none — so the first authenticated call answers
      <code>403 "user not found"</code> rather than 401. Create one once (that
      call is itself authenticated, so it needs this token):
    </p>
    <pre class="dev-sign-in__cmd"><code>curl -X POST -H "Authorization: Bearer $TOKEN" -H 'content-type: application/json' \
  -d '{"displayName":"Dev User","selfReportedStartingLevel":3,"roles":["ROLE_PLAYER"]}' \
  localhost:8080/v1/users</code></pre>
    <p class="dev-sign-in__hint">
      Paste the <code>id</code> it returns into the field below. Verified
      end-to-end at T65.1 against a real server: with no header that read
      answers 401, with a token but no user row 403, and after this call 200.
    </p>

    <p v-if="authToken !== null" class="dev-sign-in__status" role="status">
      Signed in for this browser.
      <span v-if="authUserId !== null"> Acting as <code>{{ authUserId }}</code>.</span>
    </p>

    <form novalidate @submit.prevent="submit">
      <div class="dev-sign-in__field">
        <label for="dev-token">Bearer token</label>
        <textarea
          id="dev-token"
          v-model="token"
          rows="4"
          autocomplete="off"
          spellcheck="false"
          :aria-describedby="error !== null ? 'dev-token-error' : 'dev-token-hint'"
          :aria-invalid="error !== null ? 'true' : undefined"
        />
        <p id="dev-token-hint" class="dev-sign-in__hint">
          Verified by the server against <code>dev/auth/</code>'s committed
          (and therefore worthless) keypair. Never point a deployment at it.
        </p>
        <p v-if="error !== null" id="dev-token-error" class="dev-sign-in__error" role="alert">
          {{ error }}
        </p>
      </div>

      <div class="dev-sign-in__field">
        <label for="dev-user-id">Acting user id (optional)</label>
        <input id="dev-user-id" v-model="userId" autocomplete="off" aria-describedby="dev-user-id-hint" />
        <p id="dev-user-id-hint" class="dev-sign-in__hint">
          Used for request fields that name a player or an owner. The server
          resolves your token's subject itself; no RPC reports a caller's own
          user id yet, which is why this is asked for here.
        </p>
      </div>

      <div class="dev-sign-in__actions">
        <button type="submit">Sign in</button>
        <button type="button" @click="clear">Sign out</button>
      </div>
    </form>
  </section>
</template>

<style scoped>
.dev-sign-in {
  max-width: 44rem;
  margin: 0 auto;
  padding: 1.5rem;
}
.dev-sign-in__cmd {
  overflow-x: auto;
  padding: 0.75rem;
  background: var(--surface-sunken, #f4f4f5);
  border-radius: 0.375rem;
}
.dev-sign-in__field {
  margin-block: 1.25rem;
}
.dev-sign-in__field label {
  display: block;
  font-weight: 600;
  margin-bottom: 0.35rem;
}
.dev-sign-in__field textarea,
.dev-sign-in__field input {
  width: 100%;
  font-family: ui-monospace, monospace;
  padding: 0.5rem;
}
.dev-sign-in__hint {
  margin-top: 0.35rem;
  font-size: 0.875rem;
  color: var(--text-muted, #52525b);
}
.dev-sign-in__error {
  margin-top: 0.35rem;
  color: var(--danger, #b91c1c);
  font-weight: 600;
}
.dev-sign-in__actions {
  display: flex;
  gap: 0.75rem;
}
.dev-sign-in__actions button {
  min-height: 44px;
  padding-inline: 1rem;
}
.dev-sign-in__actions button:focus-visible {
  outline: 3px solid var(--court, #2f855a);
  outline-offset: 2px;
}
</style>
