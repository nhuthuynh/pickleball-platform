<script setup lang="ts">
// ---------------------------------------------------------------------------
// PLACEHOLDER — mock/hardcoded role data only.
//
// Implements the kickoff note's decision #1 in docs/process/t7-sprint-plan.md
// ("Role-switching UX. Decided contextual, with a lightweight persistent
// indicator... the app shell shows a role indicator in the nav ('Viewing
// as: Player ▾') that only lists roles the signed-in account actually holds
// evidence for"). There is no real Identity/Users/Auth context in this repo
// yet (see the sprint plan's "starting facts"), so this component's account
// and role list are hardcoded mock data, NOT read from any backend. It must
// not be treated as an authorization boundary — same caveat class as
// T5.5/T6.3/T6.7's ActorUserID pattern (UI state only, server-unverified).
//
// Wire this to a real account/session store once Identity/Users exists
// (tracked as a T7.4+ / Identity-context follow-up, not solved here).
// ---------------------------------------------------------------------------
import { ref } from 'vue'

export interface MockRole {
  id: 'player' | 'host' | 'owner' | 'club'
  label: string
}

// A player who has also hosted a game (so both roles have "evidence") but
// has never onboarded a facility, per decision #1's "only lists roles the
// signed-in account actually holds evidence for" rule.
const MOCK_AVAILABLE_ROLES: MockRole[] = [
  { id: 'player', label: 'Player' },
  { id: 'host', label: 'Host' },
]

const currentRoleId = ref<MockRole['id']>(MOCK_AVAILABLE_ROLES[0]!.id)

function selectRole(id: MockRole['id']) {
  currentRoleId.value = id
}

function currentRoleLabel(): string {
  return MOCK_AVAILABLE_ROLES.find((r) => r.id === currentRoleId.value)?.label ?? 'Player'
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
      <option v-for="role in MOCK_AVAILABLE_ROLES" :key="role.id" :value="role.id">
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
