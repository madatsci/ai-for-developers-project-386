import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const {
  createCall,
  getPublicCallType,
  getPublicCallTypeAvailability,
  listPublicCallTypes,
  routeState,
} = vi.hoisted(() => ({
  createCall: vi.fn(),
  getPublicCallType: vi.fn(),
  getPublicCallTypeAvailability: vi.fn(),
  listPublicCallTypes: vi.fn(),
  routeState: { typeSlug: 'zvonok-na-30-minut' },
}))

vi.mock('@/api/generated', () => ({
  createCall,
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

const createdCall = {
  id: '018f0000-0000-7000-8000-000000000301',
  call_type_slug: 'zvonok-na-30-minut',
  call_type_name: 'Звонок на 30 минут',
  duration_minutes: 30,
  starts_at: '2026-10-21T11:30:00+03:00',
  ends_at: '2026-10-21T12:00:00+03:00',
  timezone: 'Europe/Moscow',
  visitor_name: 'Йрина',
  visitor_email: 'irina@example.com',
  visitor_notes: 'Первая строка\nВторая строка',
}

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
    sessionStorage.clear()
    routeState.typeSlug = 'zvonok-na-30-minut'
    getPublicCallType.mockReset()
    getPublicCallTypeAvailability.mockReset()
    listPublicCallTypes.mockReset()
    createCall.mockReset()
    getPublicCallType.mockResolvedValue({ data: activeCallType })
    listPublicCallTypes.mockResolvedValue({ data: { items: activeTypes } })
  })

  it('books a selected slot through the generated SDK and shows the immutable confirmation', async () => {
    getPublicCallTypeAvailability.mockResolvedValueOnce({ data: availability() })
    createCall.mockResolvedValueOnce({ data: createdCall })

    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-slot]').trigger('click')
    await wrapper.get('input[name="visitor_name"]').setValue('  Йрина  ')
    await wrapper.get('input[name="visitor_email"]').setValue(' irina@EXAMPLE.COM ')
    await wrapper
      .get('textarea[name="visitor_notes"]')
      .setValue('  Первая строка\r\nВторая строка  ')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(createCall).toHaveBeenCalledExactlyOnceWith({
      body: {
        call_type_slug: 'zvonok-na-30-minut',
        call_type_version: '1',
        starts_at: '2026-10-21T11:30:00+03:00',
        visitor_name: 'Йрина',
        visitor_email: 'irina@example.com',
        visitor_notes: 'Первая строка\nВторая строка',
      },
      headers: {
        'Idempotency-Key': expect.stringMatching(
          /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/,
        ),
      },
    })
    const confirmation = wrapper.get('[data-confirmation]').text()
    expect(confirmation).toContain('Звонок на 30 минут')
    expect(confirmation).toContain('21 октября 2026 г.')
    expect(confirmation).toContain('11:30–12:00')
    expect(confirmation).toContain('30 минут')
    expect(confirmation).toContain('Йрина')
    expect(confirmation).toContain('irina@example.com')
    expect(confirmation).toContain('Первая строка\nВторая строка')
    expect(confirmation).toContain('Письмо с подтверждением не отправляется')
    expect(wrapper.text()).not.toContain('018f0000-0000-7000-8000-000000000301')
    expect(JSON.parse(sessionStorage.getItem('booking-confirmation') ?? 'null')).toEqual(
      expect.objectContaining({ visitor_name: 'Йрина' }),
    )
  })

  it('reuses the idempotency key for a network retry and replaces it after input changes', async () => {
    getPublicCallTypeAvailability.mockResolvedValueOnce({ data: availability() })
    createCall.mockRejectedValue(new Error('network unavailable'))
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-slot]').trigger('click')
    await wrapper.get('input[name="visitor_name"]').setValue('Анна')
    await wrapper.get('input[name="visitor_email"]').setValue('anna@example.com')

    await wrapper.get('form').trigger('submit')
    await flushPromises()
    const keyForCall = (index: number) => {
      const call = createCall.mock.calls[index]
      if (!call) throw new Error(`createCall invocation ${index} is missing`)
      return call[0].headers['Idempotency-Key']
    }
    const firstKey = keyForCall(0)
    expect(wrapper.get('[role="alert"]').text()).toContain('Повторите отправку')

    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(keyForCall(1)).toBe(firstKey)

    await wrapper.get('input[name="visitor_name"]').setValue('Мария')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(keyForCall(2)).not.toBe(firstKey)
  })

  it('keeps the idempotency key after reload and reselecting the same slot', async () => {
    getPublicCallTypeAvailability.mockResolvedValue({ data: availability() })
    createCall.mockRejectedValue(new Error('network unavailable'))
    const firstView = mountView()
    await flushPromises()
    await firstView.get('[data-slot]').trigger('click')
    await firstView.get('input[name="visitor_name"]').setValue('Анна')
    await firstView.get('input[name="visitor_email"]').setValue('anna@example.com')
    await firstView.get('form').trigger('submit')
    await flushPromises()
    const firstKey = createCall.mock.calls[0]?.[0].headers['Idempotency-Key']
    firstView.unmount()

    const reloadedView = mountView()
    await flushPromises()
    await reloadedView.get('[data-slot]').trigger('click')
    await reloadedView.get('form').trigger('submit')
    await flushPromises()

    expect(createCall.mock.calls[1]?.[0].headers['Idempotency-Key']).toBe(firstKey)
  })

  it('restores the confirmation in the same tab without exposing the internal id', async () => {
    sessionStorage.setItem('booking-confirmation', JSON.stringify(createdCall))
    getPublicCallTypeAvailability.mockResolvedValueOnce({ data: availability() })

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.get('[data-confirmation]').text()).toContain('Звонок забронирован')
    expect(wrapper.get('[data-confirmation]').text()).toContain('Йрина')
    expect(wrapper.text()).not.toContain(createdCall.id)
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

  it('renders call type metadata from the same availability snapshot as the slots', async () => {
    const currentAvailability = availability()
    currentAvailability.call_type.name = 'Обновлённый тип звонка'
    currentAvailability.call_type.description = 'Актуальное описание'
    currentAvailability.call_type.duration_minutes = 45
    currentAvailability.call_type.call_type_version = '2'
    getPublicCallTypeAvailability.mockResolvedValueOnce({ data: currentAvailability })

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('Обновлённый тип звонка')
    expect(wrapper.text()).toContain('Актуальное описание')
    expect(wrapper.text()).toContain('45 минут')
    expect(wrapper.text()).not.toContain('Обсудим задачу')
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
