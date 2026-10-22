import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getPublicCallType, getPublicCallTypeAvailability, listPublicCallTypes, routeState } =
  vi.hoisted(() => ({
    getPublicCallType: vi.fn(),
    getPublicCallTypeAvailability: vi.fn(),
    listPublicCallTypes: vi.fn(),
    routeState: { typeSlug: 'zvonok-na-30-minut' },
  }))

vi.mock('@/api/generated', () => ({
  getPublicCallType,
  getPublicCallTypeAvailability,
  listPublicCallTypes,
}))

vi.mock('vue-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-router')>()
  return {
    ...actual,
    useRoute: () => ({ params: { typeSlug: routeState.typeSlug } }),
  }
})

import BookingView from '@/views/BookingView.vue'

const activeCallType = {
  slug: 'zvonok-na-30-minut',
  name: 'Звонок на 30 минут',
  description: 'Обсудим задачу',
  duration_minutes: 30,
  status: 'active',
  call_type_version: '1',
}

const activeTypes = [
  {
    slug: 'zvonok-na-15-minut',
    name: 'Звонок на 15 минут',
    description: null,
    duration_minutes: 15,
  },
]

function availability(slotsOnFirstDay = true) {
  const dates = [
    '2026-10-21',
    '2026-10-22',
    '2026-10-23',
    '2026-10-24',
    '2026-10-25',
    '2026-10-26',
    '2026-10-27',
    '2026-10-28',
    '2026-10-29',
    '2026-10-30',
    '2026-10-31',
    '2026-11-01',
    '2026-11-02',
    '2026-11-03',
  ]
  return {
    generated_at: '2026-10-21T10:02:34+03:00',
    timezone: 'Europe/Moscow',
    call_type: {
      slug: activeCallType.slug,
      name: activeCallType.name,
      description: activeCallType.description,
      duration_minutes: activeCallType.duration_minutes,
      call_type_version: activeCallType.call_type_version,
    },
    days: dates.map((date, index) => ({
      date,
      slots:
        index === 0 && slotsOnFirstDay
          ? [
              {
                starts_at: '2026-10-21T11:30:00+03:00',
                ends_at: '2026-10-21T12:00:00+03:00',
              },
            ]
          : [],
    })),
  }
}

function mountView() {
  return mount(BookingView, {
    global: { stubs: { RouterLink: RouterLinkStub } },
  })
}

describe('BookingView', () => {
  beforeEach(() => {
    routeState.typeSlug = 'zvonok-na-30-minut'
    getPublicCallType.mockReset()
    getPublicCallTypeAvailability.mockReset()
    listPublicCallTypes.mockReset()
    getPublicCallType.mockResolvedValue({ data: activeCallType })
    listPublicCallTypes.mockResolvedValue({ data: { items: activeTypes } })
  })

  it('keeps a scheduler skeleton until it renders all 14 dates and their slots', async () => {
    let resolveAvailability!: (value: unknown) => void
    getPublicCallTypeAvailability.mockReturnValueOnce(
      new Promise((resolve) => {
        resolveAvailability = resolve
      }),
    )

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.get('[aria-busy="true"]').text()).toContain('Загружаем свободное время')

    resolveAvailability({ data: availability() })
    await flushPromises()

    expect(wrapper.findAll('[data-availability-day]')).toHaveLength(14)
    expect(wrapper.text()).toContain('11:30')
    expect(wrapper.text()).toContain('Звонок на 30 минут')
    expect(wrapper.find('[aria-busy="true"]').exists()).toBe(false)
  })

  it('shows a local availability error and retries without losing the call type', async () => {
    getPublicCallTypeAvailability
      .mockResolvedValueOnce({ error: { code: 'internal_error' } })
      .mockResolvedValueOnce({ data: availability(false) })

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('Звонок на 30 минут')
    expect(wrapper.get('[role="alert"]').text()).toContain('Не удалось загрузить свободное время')

    await wrapper.get('[data-retry-availability]').trigger('click')
    await flushPromises()

    expect(getPublicCallTypeAvailability).toHaveBeenCalledTimes(2)
    expect(wrapper.findAll('[data-availability-day]')).toHaveLength(14)
    expect(wrapper.text()).toContain('В ближайшие 14 дней свободного времени нет')
  })

  it.each([
    {
      name: 'disabled',
      callTypeResult: { data: { ...activeCallType, status: 'disabled' } },
      state: 'disabled',
      message: 'Тип звонка временно недоступен',
    },
    {
      name: 'unknown',
      callTypeResult: { error: { code: 'call_type_not_found' } },
      state: 'not-found',
      message: 'Тип звонка не найден',
    },
  ])('distinguishes a $name type and offers active alternatives', async (example) => {
    getPublicCallType.mockResolvedValueOnce(example.callTypeResult)

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.get(`[data-state="${example.state}"]`).text()).toContain(example.message)
    expect(wrapper.text()).toContain('Звонок на 15 минут')
    expect(getPublicCallTypeAvailability).not.toHaveBeenCalled()
  })
})
