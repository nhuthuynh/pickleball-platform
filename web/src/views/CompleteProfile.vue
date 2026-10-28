<script setup lang="ts">
// #340. The screen that turns "signed in" into "has an account".
//
// Before this, a first-time player with a valid token could do nothing:
// every write returned `403 <context>: user not found`, and the product had
// no path to `CreateUser` anywhere — `Profile.vue` recorded that absence as
// deliberate scope, and it stayed that way for ten sprints because the only
// thing that could notice is a browser, and nothing opened one.
//
// WHY IT ASKS RATHER THAN GUESSES. The Product Owner chose an explicit
// screen over silent creation (2026-10-08). The self-reported starting
// level is the one field `CLAUDE.md`'s locked cold-start decision names —
// "new players seeded by a self-reported starting level" — and T65.2's
// Player Level formula seeds from it, blending it with real results until
// the 20-game confidence ramp hands over. A default would not be a neutral
// placeholder; it would be a claim the player never made, already feeding
// the rating.
//
// ROLE. `ROLE_PLAYER` only, and that is not a UI choice to revisit: the
// server refuses anything else on this path (`ErrRoleNotSelfAssignable`),
// because holding a valid token proves who you are, not that you may
// appoint yourself a platform admin.
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'

import { identityClient, type IdentityClient } from '../api/identityClient'
import { clearProfileMissing, rememberUserId, isSignedIn } from '../state/authSession'
import { ensureCurrentUserId } from '../api/currentUser'
import { MIN_SELF_REPORTED_LEVEL, MAX_SELF_REPORTED_LEVEL } from '../models/identityLevel'

const props = defineProps<{
  /** Injectable for tests; defaults to the real identityClient. */
  client?: IdentityClient
}>()

const router = useRouter()
const client = computed(() => props.client ?? identityClient)

const displayName = ref('')
const level = ref<string>('')
const saving = ref(false)
const error = ref<string | null>(null)

const levels = Array.from(
  { length: MAX_SELF_REPORTED_LEVEL - MIN_SELF_REPORTED_LEVEL + 1 },
  (_, i) => MIN_SELF_REPORTED_LEVEL + i,
)

const canSubmit = computed(() => displayName.value.trim() !== '' && level.value !== '' && !saving.value)

async function submit(): Promise<void> {
  if (!canSubmit.value) return
  saving.value = true
  error.value = null

  const { data, error: apiError } = await client.value.POST('/v1/users', {
    body: {
      displayName: displayName.value.trim(),
      roles: ['ROLE_PLAYER'],
      selfReportedStartingLevel: Number(level.value),
    },
  })

  saving.value = false

  if (apiError !== undefined || data?.user?.id === undefined) {
    // The account may already exist for this verified subject — that is
    // what a 409 means, and it is the ordinary case for a player whose
    // browser lost its stored id (#344). It is not an error to report: ask
    // the server who we are, and if it answers, this screen has nothing
    // left to do.
    if ((await ensureCurrentUserId()) === 'resolved') {
      clearProfileMissing()
      await router.push('/profile')
      return
    }

    // The server's own message, not a sentence invented here — if it says
    // the account already exists, that is worth reading verbatim.
    error.value =
      (apiError as { message?: string } | undefined)?.message ??
      'Could not create your profile. Please try again.'
    return
  }

  rememberUserId(data.user.id)
  clearProfileMissing()
  await router.push('/profile')
}
</script>

<template>
  <section class="complete-profile">
    <h1>Finish setting up your profile</h1>

    <p v-if="!isSignedIn()" class="complete-profile__hint" role="alert">
      You are not signed in. <RouterLink to="/sign-in">Sign in</RouterLink> first — creating a
      profile needs a verified account.
    </p>

    <p class="complete-profile__hint">
      You are signed in, but you do not have a player profile yet. Everything that creates or joins
      something needs one.
    </p>

    <form novalidate @submit.prevent="submit">
      <div class="complete-profile__field">
        <label for="profile-display-name">Display name</label>
        <input
          id="profile-display-name"
          v-model="displayName"
          autocomplete="name"
          aria-describedby="profile-display-name-hint"
        />
        <p id="profile-display-name-hint" class="complete-profile__hint">
          What other players see on a game roster.
        </p>
      </div>

      <div class="complete-profile__field">
        <label for="profile-level">Starting level</label>
        <select id="profile-level" v-model="level" aria-describedby="profile-level-hint">
          <option value="" disabled>Choose a level</option>
          <option v-for="l in levels" :key="l" :value="String(l)">{{ l }}</option>
        </select>
        <p id="profile-level-hint" class="complete-profile__hint">
          Your own honest assessment, from {{ MIN_SELF_REPORTED_LEVEL }} (new to the sport) to
          {{ MAX_SELF_REPORTED_LEVEL }} (competitive). It is only a starting point — once you have
          played a few games your level follows your results, and you can change this at any time
          on your profile.
        </p>
      </div>

      <p v-if="error !== null" class="complete-profile__error" role="alert">{{ error }}</p>

      <button type="submit" data-testid="create-profile" :disabled="!canSubmit">
        {{ saving ? 'Creating…' : 'Create my profile' }}
      </button>
    </form>
  </section>
</template>

<style scoped>
.complete-profile {
  max-width: 34rem;
}
.complete-profile__field {
  margin-block: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}
.complete-profile__hint {
  font-size: 0.875rem;
  color: #4a5a52;
  margin: 0;
}
.complete-profile__error {
  color: #a4202b;
}
</style>
