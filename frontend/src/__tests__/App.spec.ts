import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getPublicCallType, getPublicCallTypeAvailability, listPublicCallTypes } = vi.hoisted(
  () => ({
    getPublicCallType: vi.fn(),
    getPublicCallTypeAvailability: vi.fn(),
    listPublicCallTypes: vi.fn(),
  }),
)

vi.mock('@/api/generated', () => ({
  getPublicCallType,
  getPublicCallTypeAvailability,
  listPublicCallTypes,
}))

import { flushPromises, mount } from '@vue/test-utils'
import App from '../App.vue'
import router from '../router'

describe('App', () => {
  beforeEach(() => {
    window.sessionStorage.clear()
    getPublicCallType.mockReset()
    getPublicCallTypeAvailability.mockReset()
    listPublicCallTypes.mockReset()
    listPublicCallTypes.mockResolvedValue({
      data: {
        items: [
          {
            slug: 'zvonok-na-15-minut',
            name: 'Звонок на 15 минут',
            description: null,
            duration_minutes: 15,
          },
          {
            slug: 'zvonok-na-30-minut',
            name: 'Звонок на 30 минут',
            description: null,
            duration_minutes: 30,
          },
          {
            slug: 'zvonok-na-60-minut',
            name: 'Звонок на 60 минут',
            description: null,
            duration_minutes: 60,
          },
        ],
      },
    })
    getPublicCallType.mockResolvedValue({
      data: {
        slug: 'zvonok-na-15-minut',
        name: 'Звонок на 15 минут',
        description: null,
        duration_minutes: 15,
        status: 'active',
        call_type_version: '1',
      },
    })
    getPublicCallTypeAvailability.mockResolvedValue({
      data: {
        generated_at: '2026-10-21T10:02:34+03:00',
        timezone: 'Europe/Moscow',
        call_type: {
          slug: 'zvonok-na-15-minut',
          name: 'Звонок на 15 минут',
          description: null,
          duration_minutes: 15,
          call_type_version: '1',
        },
        days: [
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
        ].map((date, index) => ({
          date,
          slots:
            index === 0
              ? [
                  {
                    starts_at: '2026-10-21T11:30:00+03:00',
                    ends_at: '2026-10-21T11:45:00+03:00',
                  },
                ]
              : [],
        })),
      },
    })
  })

  it('explains the service and links every call to action to booking', async () => {
    await router.push('/')
    await router.isReady()

    const wrapper = mount(App, { global: { plugins: [router] } })
    const bookingLinks = wrapper.findAll('a[href="/booking"]')

    expect(wrapper.get('h1').text()).toBe('Удобное время для звонка — без долгих переписок')
    expect(wrapper.text()).toContain('Как это работает')
    expect(bookingLinks).toHaveLength(3)
    expect(bookingLinks.every((link) => link.text() === 'Записаться на звонок')).toBe(true)

    await bookingLinks[0]!.trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.path).toBe('/booking')
    expect(wrapper.get('h1').text()).toBe('Бронирование звонка')
    expect(wrapper.text()).toContain('Звонок на 15 минут')
    wrapper.unmount()
  })

  it('renders the active call types and opens the selected public booking route', async () => {
    await router.push('/booking')
    await router.isReady()

    const wrapper = mount(App, { global: { plugins: [router] } })

    expect(wrapper.get('h1').text()).toBe('Бронирование звонка')
    expect(wrapper.get('a[href="/"]').text()).toBe('Календарь звонков')

    await flushPromises()
    await wrapper.get('a[href="/booking/zvonok-na-15-minut"]').trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.path).toBe('/booking/zvonok-na-15-minut')
    expect(wrapper.text()).toContain('Свободное время')
    wrapper.unmount()
  })

  it('reloads the planner when an unavailable type links to an active alternative', async () => {
    getPublicCallType.mockImplementation(({ path }: { path: { slug: string } }) =>
      Promise.resolve({
        data: {
          slug: path.slug,
          name: path.slug === 'otklyuchennyy' ? 'Отключённый тип' : 'Звонок на 15 минут',
          description: null,
          duration_minutes: 15,
          status: path.slug === 'otklyuchennyy' ? 'disabled' : 'active',
          call_type_version: '1',
        },
      }),
    )
    await router.push('/booking/otklyuchennyy')
    await router.isReady()

    const wrapper = mount(App, { global: { plugins: [router] } })
    await flushPromises()

    expect(wrapper.text()).toContain('Тип звонка временно недоступен')
    await wrapper.get('a[href="/booking/zvonok-na-15-minut"]').trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.path).toBe('/booking/zvonok-na-15-minut')
    expect(getPublicCallTypeAvailability).toHaveBeenCalledWith({
      path: { slug: 'zvonok-na-15-minut' },
    })
    expect(wrapper.findAll('[data-availability-day]')).toHaveLength(14)
    wrapper.unmount()
  })
})
