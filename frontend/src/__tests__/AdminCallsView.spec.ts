import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { AdminCall } from '@/api/generated'

const { listAdminCalls, listAdminCallTypes } = vi.hoisted(() => ({
  listAdminCalls: vi.fn(),
  listAdminCallTypes: vi.fn(),
}))

vi.mock('@/api/generated', () => ({ listAdminCalls, listAdminCallTypes }))

import AdminCallsView from '@/views/AdminCallsView.vue'

const callType = {
  id: '018f0000-0000-7000-8000-000000000201',
  slug: 'zvonok-na-30-minut',
  name: 'Звонок на 30 минут',
  description: null,
  duration_minutes: 30,
  status: 'active',
  schedule: { id: '018f0000-0000-7000-8000-000000000101', name: 'Основное' },
  version: '1',
  can_delete: false,
  delete_blocked_reason: 'call_type_has_calls',
}

const firstCall: AdminCall = {
  id: '018f0000-0000-7000-8000-000000000301',
  call_type_id: callType.id,
  call_type_name: 'Историческое название',
  duration_minutes: 30,
  starts_at: '2026-10-20T11:45:00+03:00',
  ends_at: '2026-10-20T12:15:00+03:00',
  visitor_name: 'Анна Смирнова',
  visitor_email: 'anna@example.com',
  visitor_notes: 'Обсудить API',
}

const secondCall: AdminCall = {
  ...firstCall,
  id: '018f0000-0000-7000-8000-000000000302',
  starts_at: '2026-10-21T13:00:00+03:00',
  ends_at: '2026-10-21T13:30:00+03:00',
  visitor_name: 'Михаил Орлов',
  visitor_email: 'mikhail@example.com',
  visitor_notes: null,
}

function page(items: AdminCall[] = [firstCall], nextCursor: string | null = null) {
  return {
    data: {
      as_of: '2026-10-20T12:00:00+03:00',
      timezone: 'Europe/Moscow',
      items,
      next_cursor: nextCursor,
    },
  }
}

function mountView() {
  return mount(AdminCallsView, { global: { stubs: { RouterLink: RouterLinkStub } } })
}

describe('AdminCallsView', () => {
  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
    vi.setSystemTime(new Date('2026-10-20T12:00:00+03:00'))
    listAdminCalls.mockReset()
    listAdminCallTypes.mockReset()
    listAdminCallTypes.mockResolvedValue({ data: { items: [structuredClone(callType)] } })
    listAdminCalls.mockResolvedValue(page())
  })

  it('shows grouped snapshot rows and selects the first call without exposing its UUID', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(listAdminCalls).toHaveBeenCalledExactlyOnceWith({ query: { period: 'upcoming' } })
    expect(wrapper.text()).toContain('20 октября')
    expect(wrapper.text()).toContain('1 звонок')
    expect(wrapper.text()).toContain('Анна Смирнова')
    expect(wrapper.text()).toContain('Историческое название')
    expect(wrapper.text()).toContain('Сейчас')
    expect(wrapper.get('[data-call-details]').text()).toContain('Обсудить API')
    expect(wrapper.text()).not.toContain(firstCall.id)
  })

  it('shows skeletons during the first request and a retry action after a primary error', async () => {
    let resolveRequest: (value: unknown) => void = () => undefined
    listAdminCalls.mockReturnValueOnce(
      new Promise((resolve) => {
        resolveRequest = resolve
      }),
    )
    const wrapper = mountView()
    expect(wrapper.find('[aria-label="Загрузка звонков"]').exists()).toBe(true)
    expect(wrapper.find('[aria-label="Загрузка деталей"]').exists()).toBe(true)

    resolveRequest({ error: { code: 'internal_error' }, response: { status: 500 } })
    await flushPromises()
    expect(wrapper.text()).toContain('Не удалось загрузить звонки')

    listAdminCalls.mockResolvedValueOnce(page())
    await wrapper.get('[data-retry-initial]').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('Анна Смирнова')
  })

  it('keeps search and stable type filter between tabs while creating a new snapshot', async () => {
    listAdminCalls.mockResolvedValue(page([]))
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('input[type="search"]').setValue('  АННА  ')
    await wrapper.get('select[name="call-type-filter"]').setValue(callType.id)
    await flushPromises()
    await wrapper.get('[data-period="past"]').trigger('click')
    await flushPromises()

    expect(listAdminCalls).toHaveBeenLastCalledWith({
      query: { period: 'past', call_type_id: callType.id, q: 'АННА' },
    })
    expect(wrapper.get('input[type="search"]').element).toHaveProperty('value', '  АННА  ')
    expect(wrapper.get('select[name="call-type-filter"]').element).toHaveProperty(
      'value',
      callType.id,
    )
    expect(window.location.search).toBe('')
  })

  it('preserves rows when loading more fails and retries only the cursor page', async () => {
    listAdminCalls
      .mockResolvedValueOnce(page([firstCall], 'next-page'))
      .mockResolvedValueOnce({ error: { code: 'internal_error' }, response: { status: 500 } })
      .mockResolvedValueOnce(page([secondCall]))
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-load-more]').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('Анна Смирнова')
    expect(wrapper.text()).toContain('Не удалось загрузить следующую страницу')

    await wrapper.get('[data-retry-load-more]').trigger('click')
    await flushPromises()
    expect(listAdminCalls).toHaveBeenLastCalledWith({
      query: { period: 'upcoming', cursor: 'next-page' },
    })
    expect(wrapper.text()).toContain('Анна Смирнова')
    expect(wrapper.text()).toContain('Михаил Орлов')
  })

  it('distinguishes empty states, resets filters, and uses a separate mobile details screen', async () => {
    listAdminCalls.mockResolvedValue(page([]))
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.text()).toContain('Предстоящих звонков нет')

    await wrapper.get('input[type="search"]').setValue('нет совпадений')
    await flushPromises()
    expect(wrapper.text()).toContain('Ничего не найдено')
    await wrapper.get('[data-reset-filters]').trigger('click')
    await flushPromises()
    expect(wrapper.get('input[type="search"]').element).toHaveProperty('value', '')

    await wrapper.get('[data-period="past"]').trigger('click')
    await flushPromises()
    expect(wrapper.text()).toContain('Прошедших звонков нет')

    listAdminCalls.mockResolvedValue(page([firstCall]))
    await wrapper.get('[data-period="upcoming"]').trigger('click')
    await flushPromises()
    await wrapper.get(`[data-select-call="${firstCall.id}"]`).trigger('click')
    expect(wrapper.get('[data-call-list]').classes()).toContain('hidden')
    expect(wrapper.get('[data-call-details]').classes()).toContain('block')
    await wrapper.get('[data-mobile-calls-back]').trigger('click')
    expect(wrapper.get('[data-call-list]').classes()).toContain('block')
  })
})
