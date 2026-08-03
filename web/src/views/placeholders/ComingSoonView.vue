<script setup lang="ts">
// Generic placeholder for routes whose real screen hasn't been built yet
// (T8.1, docs/process/t8-sprint-plan.md's T8.1 Instructions #1: "Routes
// for T8.8-T8.10's screens are added as empty placeholder components in
// *this* ticket... the tickets that build each screen for real replace
// the placeholder, they don't invent the route.").
//
// One shared component (rather than four/six near-identical files) reads
// its heading from the matched route's `meta.title` — see
// src/router/index.ts's route table. A later ticket replacing a
// placeholder swaps that route's `component:` entry; it does not need to
// touch this file.
import { useRoute } from 'vue-router'

const route = useRoute()
const title = (route.meta.title as string | undefined) ?? 'This screen'
</script>

<template>
  <section class="coming-soon" role="status">
    <h1 class="coming-soon__title">{{ title }}</h1>
    <p class="coming-soon__body">Coming soon.</p>
  </section>
</template>

<style scoped>
.coming-soon {
  padding: 2rem 1rem;
  color: var(--ink-soft);
}

.coming-soon__title {
  font-size: var(--font-size-lg);
  color: var(--ink);
  margin: 0 0 0.5rem;
}

.coming-soon__body {
  margin: 0;
}
</style>
