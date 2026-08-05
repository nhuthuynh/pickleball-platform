// T9.7's routes. Kept in its own spec file rather than added to a shared
// router spec because T9.6 lands concurrently against the same route table —
// separate files keep the merge to router/index.ts itself.
import { describe, it, expect } from 'vitest'
import { createRouter, createMemoryHistory } from 'vue-router'
import { routes } from '../index'

function router() {
  return createRouter({ history: createMemoryHistory(), routes })
}

describe('competition routes (T9.7)', () => {
  it('/competitions resolves to the discover screen', () => {
    const resolved = router().resolve('/competitions')
    expect(resolved.name).toBe('competitions')
    expect(resolved.matched).toHaveLength(1)
  })

  it('/competitions/:id resolves and exposes the competition id', () => {
    const resolved = router().resolve('/competitions/c1')
    expect(resolved.name).toBe('competition-detail')
    expect(resolved.params.id).toBe('c1')
  })

  // The deep-link path a shared link lands on. Deliberately short (/c/…) —
  // it is pasted into social posts and messages.
  it('/c/:shareToken resolves to the share-link landing and exposes the token', () => {
    const resolved = router().resolve('/c/abc123-token')
    expect(resolved.name).toBe('competition-share-link')
    expect(resolved.params.shareToken).toBe('abc123-token')
  })

  it('the share-link route does not collide with /competitions', () => {
    expect(router().resolve('/competitions').name).not.toBe('competition-share-link')
  })
})
