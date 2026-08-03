<script setup lang="ts">
// ---------------------------------------------------------------------------
// PLACEHOLDER — mock/hardcoded account, but a REAL (if client-side-only)
// per-role evidence signal as of T8.8.
//
// Implements the kickoff note's decision #1 in docs/process/t7-sprint-plan.md
// ("Role-switching UX. Decided contextual, with a lightweight persistent
// indicator... the app shell shows a role indicator in the nav ('Viewing
// as: Player ▾') that only lists roles the signed-in account actually holds
// evidence for"). There is no real Identity/Users/Auth context in this repo
// yet (see the sprint plan's "starting facts"), so the account itself is
// still hardcoded mock data, NOT read from any backend. It must not be
// treated as an authorization boundary — same caveat class as
// T5.5/T6.3/T6.7's ActorUserID pattern (UI state only, server-unverified).
//
// T8's kickoff note (docs/process/t8-sprint-plan.md) decision #1 extends
// this exact mechanism with a second, real role: "Host" is now listed once
// this browser session has actually called `CreateGame` successfully (see
// src/state/roleEvidence.ts and src/views/GameCreation.vue), rather than
// being hardcoded present unconditionally as it was pre-T8.
//
// Wire the account itself to a real account/session store once
// Identity/Users exists (tracked as a T7.4+ / Identity-context follow-up,
// not solved here).
// ---------------------------------------------------------------------------
import { computed, ref, watch } from 'vue'
import { hasHostEvidence } from '../state/roleEvidence'

export interface MockRole {
  id: 'player' | 'host' | 'owner' | 'club'
  label: string
}

// Player is always available (the mock account's baseline role); Host is
// added once this session has real evidence for it (T8.8). Owner/Club stay
// absent — no onboarding-evidence signal exists for either yet, per
// decision #1's "only lists roles the signed-in account actually holds
// evidence for" rule.
const availableRoles = computed<MockRole[]>(() => {
  const roles: MockRole[] = [{ id: 'player', label: 'Player' }]
  if (hasHostEvidence.value) {
    roles.push({ id: 'host', label: 'Host' })
  }
  return roles
})

const currentRoleId = ref<MockRole['id']>('player')

// If the currently-selected role ever stops being available (only possible
// today via the test-only reset helper, but kept as a real guard rather
// than an assumption), fall back to Player instead of pointing the select
// at an option that no longer renders.
watch(availableRoles, (roles) => {
  if (!roles.some((r) => r.id === currentRoleId.value)) {
    currentRoleId.value = 'player'
  }
})

function selectRole(id: MockRole['id']) {
  currentRoleId.value = id
}

function currentRoleLabel(): string {
  return availableRoles.value.find((r) => r.id === currentRoleId.value)?.label ?? 'Player'
}
</script>

<template>
  <div class="role-indicator">
    <label for="role-indicator-select" class="role-indicator__label">Viewing as</label>
    <select
      id="role-indicator-select"
      class="role-indicator__select"
      :value="currentRoleId"
      aria-label="Switch which role you are viewing the app as (mock data — placeholder until Identity/Users exists)"
      @change="selectRole(($event.target as HTMLSelectElement).value as MockRole['id'])"
    >
      <option v-for="role in availableRoles" :key="role.id" :value="role.id">
        {{ role.label }}
      </option>
    </select>
    <span class="role-indicator__badge" :data-role="currentRoleId">{{ currentRoleLabel() }}</span>
  </div>
</template>

<style scoped>
.role-indicator {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-family: var(--font-family-ui);
  font-size: var(--font-size-sm);
}

.role-indicator__label {
  color: var(--ink-soft);
}

.role-indicator__select {
  font: inherit;
  color: var(--ink);
  background: var(--paper-raised);
  border: 1px solid var(--hs-border);
  border-radius: var(--radius-sm);
  padding: 0.25rem 0.5rem;
  min-height: 44px;
}

.role-indicator__badge {
  border-radius: var(--radius-pill);
  padding: 0.15rem 0.75rem;
  font-weight: 600;
  background: var(--pill-success-bg);
  color: var(--pill-fg);
}
</style>
