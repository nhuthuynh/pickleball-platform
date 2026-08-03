import { describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import GameJoinPanel from '../GameJoinPanel.vue'
import type { GameSummary } from '../../../models/game'
import type { SocialPlayClient } from '../../../api/socialplayClient'

const game: GameSummary = {
  id: 'g1',
  hostId: 'host-1',
  venueFacilityId: 'facility-1',
  courtIds: ['court-1'],
  startsAt: '2026-09-01T10:00:00Z',
  endsAt: '2026-09-01T11:00:00Z',
  capacity: 8,
  status: 'GAME_STATUS_SCHEDULED',
  paymentMethod: 'PAYMENT_METHOD_EITHER',
  guestAllowance: 2,
  spotsLeft: 3,
}

function fakeClient(handlers: {
  postRegistrations?: (body: unknown) => unknown
  postWaitlist?: (body: unknown) => unknown
}): SocialPlayClient {
  const POST = vi.fn(async (path: string, options: { body: unknown }) => {
    if (path === '/v1/games/{gameId}/registrations') return handlers.postRegistrations?.(options.body)
    if (path === '/v1/games/{gameId}/waitlist') return handlers.postWaitlist?.(options.body)
    throw new Error(`unexpected POST ${path}`)
  })
  return { POST, GET: vi.fn() } as unknown as SocialPlayClient
}

describe('GameJoinPanel', () => {
  // T8.9 requirement: "guest-count stepper respects GuestAllowance".
  it('the guest-count stepper never exceeds the Game\'s GuestAllowance', async () => {
    const wrapper = mount(GameJoinPanel, { props: { game, client: fakeClient({}) } })

    const incrementButton = wrapper.find('[aria-label="Add a guest"]')
    await incrementButton.trigger('click')
    await incrementButton.trigger('click')
    expect(wrapper.find('.game-join__stepper-count').text()).toBe('2')

    // A third increment attempt should be a no-op — guestAllowance is 2.
    await incrementButton.trigger('click')
    expect(wrapper.find('.game-join__stepper-count').text()).toBe('2')
    expect(incrementButton.attributes('disabled')).toBeDefined()
  })

  it('the guest-count stepper never goes below 0', async () => {
    const wrapper = mount(GameJoinPanel, { props: { game, client: fakeClient({}) } })
    const decrementButton = wrapper.find('[aria-label="Remove a guest"]')
    expect(decrementButton.attributes('disabled')).toBeDefined()
    expect(wrapper.find('.game-join__stepper-count').text()).toBe('0')
  })

  it('a Game with GuestAllowance 0 disables the increment control immediately', () => {
    const wrapper = mount(GameJoinPanel, { props: { game: { ...game, guestAllowance: 0 }, client: fakeClient({}) } })
    expect(wrapper.find('[aria-label="Add a guest"]').attributes('disabled')).toBeDefined()
    expect(wrapper.text()).toContain('Up to 0 guests allowed')
  })

  it('submitting the join form calls RegisterForGame with the current guest count and shows a success message', async () => {
    const client = fakeClient({
      postRegistrations: () => ({
        data: {
          registration: {
            id: 'r1',
            gameId: 'g1',
            playerId: 'player-mock-1',
            status: 'REGISTRATION_STATUS_REGISTERED',
            paymentStatus: 'PAYMENT_STATUS_UNPAID',
            guestCount: 1,
          },
        },
        error: undefined,
        response: { status: 200 },
      }),
    })
    const wrapper = mount(GameJoinPanel, { props: { game, client } })

    await wrapper.find('[aria-label="Add a guest"]').trigger('click')
    await wrapper.find('.game-join__form').trigger('submit')
    await flushPromises()

    expect(client.POST).toHaveBeenCalledWith('/v1/games/{gameId}/registrations', {
      params: { path: { gameId: 'g1' } },
      body: { playerId: 'player-mock-1', guestCount: 1 },
    })
    expect(wrapper.text()).toContain("You're in!")
    expect(wrapper.find('[role="status"]').exists()).toBe(true)
  })

  // T8.9 requirement: the game-full/waitlist-offer path.
  describe('game-full / waitlist-offer path', () => {
    it('a 409 (capacity-exceeded) rejection shows a specific message and offers Join waitlist', async () => {
      const client = fakeClient({
        postRegistrations: () => ({ data: undefined, error: { message: 'full' }, response: { status: 409 } }),
      })
      const wrapper = mount(GameJoinPanel, { props: { game, client } })

      await wrapper.find('.game-join__form').trigger('submit')
      await flushPromises()

      expect(wrapper.find('[role="alert"]').text()).toContain('This game just filled up')
      const waitlistButton = wrapper.find('.game-join__conflict .game-join__primary')
      expect(waitlistButton.exists()).toBe(true)
      expect(waitlistButton.text()).toContain('Join waitlist')
    })

    it('clicking Join waitlist after a game-full rejection calls JoinWaitlist and shows a confirmation', async () => {
      const client = fakeClient({
        postRegistrations: () => ({ data: undefined, error: { message: 'full' }, response: { status: 409 } }),
        postWaitlist: () => ({
          data: { entry: { id: 'w1', gameId: 'g1', playerId: 'player-mock-1', position: 4, status: 'WAITLIST_STATUS_WAITING' } },
          error: undefined,
          response: { status: 200 },
        }),
      })
      const wrapper = mount(GameJoinPanel, { props: { game, client } })

      await wrapper.find('.game-join__form').trigger('submit')
      await flushPromises()

      await wrapper.find('.game-join__conflict .game-join__primary').trigger('click')
      await flushPromises()

      expect(client.POST).toHaveBeenCalledWith('/v1/games/{gameId}/waitlist', {
        params: { path: { gameId: 'g1' } },
        body: { playerId: 'player-mock-1' },
      })
      expect(wrapper.text()).toContain('waitlist at position 4')
    })

    it('a Game whose SpotsLeft is already known to be 0 (startFull) skips straight to the waitlist offer', () => {
      const wrapper = mount(GameJoinPanel, {
        props: { game: { ...game, spotsLeft: 0 }, client: fakeClient({}), startFull: true },
      })

      expect(wrapper.text()).toContain('This game just filled up')
      expect(wrapper.find('.game-join__form').exists()).toBe(false)
    })
  })

  it('a non-conflict registration failure shows a generic error, not the waitlist offer', async () => {
    const client = fakeClient({
      postRegistrations: () => ({ data: undefined, error: { message: 'boom' }, response: { status: 500 } }),
    })
    const wrapper = mount(GameJoinPanel, { props: { game, client } })

    await wrapper.find('.game-join__form').trigger('submit')
    await flushPromises()

    expect(wrapper.find('.game-join__conflict').exists()).toBe(false)
    expect(wrapper.find('[role="alert"]').text()).toContain('Could not complete this registration')
  })

  // T8.10 requirements #1/#2: the post-registration payment choice.
  describe('post-registration payment choice (T8.10)', () => {
    function registerAndFlush(gameOverrides: Partial<typeof game> = {}) {
      const client = fakeClient({
        postRegistrations: () => ({
          data: {
            registration: {
              id: 'reg-1',
              gameId: 'g1',
              playerId: 'player-mock-1',
              status: 'REGISTRATION_STATUS_REGISTERED',
              paymentStatus: 'PAYMENT_STATUS_UNPAID',
              guestCount: 0,
            },
          },
          error: undefined,
          response: { status: 200 },
        }),
      })
      const wrapper = mount(GameJoinPanel, { props: { game: { ...game, ...gameOverrides }, client } })
      return { wrapper, client }
    }

    it('a cash-only Game shows the pending-cash text immediately, with no payment buttons', async () => {
      const { wrapper } = registerAndFlush({ paymentMethod: 'PAYMENT_METHOD_CASH' })
      await wrapper.find('.game-join__form').trigger('submit')
      await flushPromises()

      expect(wrapper.text()).toContain('Payment: pending (cash at facility)')
      expect(wrapper.find('.game-join__payment-choice').exists()).toBe(false)
    })

    it('an online-only Game shows a "Pay online now" button and no cash option', async () => {
      const { wrapper } = registerAndFlush({ paymentMethod: 'PAYMENT_METHOD_ONLINE' })
      await wrapper.find('.game-join__form').trigger('submit')
      await flushPromises()

      const buttons = wrapper.findAll('.game-join__payment-choice button').map((b) => b.text())
      expect(buttons).toEqual(['Pay online now'])
    })

    it('clicking "Pay online now" emits payOnline with the confirmed registration id', async () => {
      const { wrapper } = registerAndFlush({ paymentMethod: 'PAYMENT_METHOD_ONLINE' })
      await wrapper.find('.game-join__form').trigger('submit')
      await flushPromises()

      await wrapper.find('.game-join__payment-choice .game-join__primary').trigger('click')

      expect(wrapper.emitted('payOnline')).toEqual([['reg-1']])
    })

    it('an "either" Game offers both options; choosing cash switches to the pending text without any network call', async () => {
      const { wrapper, client } = registerAndFlush({ paymentMethod: 'PAYMENT_METHOD_EITHER' })
      await wrapper.find('.game-join__form').trigger('submit')
      await flushPromises()

      const buttons = wrapper.findAll('.game-join__payment-choice button').map((b) => b.text())
      expect(buttons).toEqual(['Pay online now', 'Pay cash at facility'])

      const callsBeforeCashChoice = (client.POST as ReturnType<typeof vi.fn>).mock.calls.length
      await wrapper.find('.game-join__secondary').trigger('click')

      expect(wrapper.text()).toContain('Payment: pending (cash at facility)')
      expect(wrapper.find('.game-join__payment-choice').exists()).toBe(false)
      expect((client.POST as ReturnType<typeof vi.fn>).mock.calls.length).toBe(callsBeforeCashChoice)
    })
  })
})
