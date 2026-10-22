import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getAdminWorkingHours, updateAdminWorkingHours, routeLeave } = vi.hoisted(() => ({
  getAdminWorkingHours: vi.fn(),
  updateAdminWorkingHours: vi.fn(),
  routeLeave: { guard: undefined as undefined | (() => boolean) },
}))

vi.mock('@/api/generated', () => ({ getAdminWorkingHours, updateAdminWorkingHours }))

vi.mock('vue-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-router')>()
  return {
    ...actual,
    onBeforeRouteLeave: (guard: () => boolean) => {
      routeLeave.guard = guard
    },
  }
})

import AdminWorkingHoursView from '@/views/AdminWorkingHoursView.vue'

const initialWorkingHours = {
  timezone: 'Europe/Moscow' as const,
  version: '1',
  days: Array.from({ length: 7 }, (_, index) => ({
    day_of_week: index + 1,
    intervals:
      index < 5
        ? [
            {
              start_time: '09:00',
              end_time: '18:00',
            },
          ]
        : [],
  })),
}

function mountView() {
  return mount(AdminWorkingHoursView, {
    global: { stubs: { RouterLink: RouterLinkStub } },
  })
}

describe('AdminWorkingHoursView', () => {
  beforeEach(() => {
    getAdminWorkingHours.mockReset()
    updateAdminWorkingHours.mockReset()
    routeLeave.guard = undefined
    getAdminWorkingHours.mockResolvedValue({
      data: structuredClone(initialWorkingHours),
      response: { headers: new Headers({ ETag: '"1"' }) },
    })
  })

  it('loads all seven days and saves the complete edited week explicitly', async () => {
    updateAdminWorkingHours.mockResolvedValue({
      data: { ...structuredClone(initialWorkingHours), version: '2' },
      response: { headers: new Headers({ ETag: '"2"' }) },
    })
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.findAll('[data-working-day]')).toHaveLength(7)
    expect(wrapper.text()).toContain('Понедельник')
    expect(wrapper.text()).toContain('Воскресенье')
    expect(wrapper.text()).toContain('Подтверждённая версия: 1')

    const mondayEnd = wrapper.get('input[name="days.0.intervals.0.end_time"]')
    await mondayEnd.setValue('17:00')
    expect(wrapper.text()).toContain('Есть несохранённые изменения')
    expect(updateAdminWorkingHours).not.toHaveBeenCalled()

    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(updateAdminWorkingHours).toHaveBeenCalledExactlyOnceWith({
      body: {
        days: expect.arrayContaining([
          expect.objectContaining({
            day_of_week: 1,
            intervals: [{ start_time: '09:00', end_time: '17:00' }],
          }),
        ]),
      },
      headers: { 'If-Match': '"1"' },
    })
    expect(wrapper.text()).toContain('Режим работы сохранён')
    expect(wrapper.text()).toContain('Подтверждённая версия: 2')
  })

  it('shows interval errors by day and updates them while the organizer corrects input', async () => {
    updateAdminWorkingHours.mockResolvedValue({
      data: { ...structuredClone(initialWorkingHours), version: '2' },
      response: { headers: new Headers({ ETag: '"2"' }) },
    })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('input[name="days.0.intervals.0.end_time"]').setValue('09:10')
    await wrapper.get('form').trigger('submit')

    expect(updateAdminWorkingHours).not.toHaveBeenCalled()
    expect(wrapper.get('[data-day-error="0"]').text()).toContain('не меньше 15 минут')

    await wrapper.get('input[name="days.0.intervals.0.end_time"]').setValue('09:15')
    expect(wrapper.find('[data-day-error="0"]').exists()).toBe(false)
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(updateAdminWorkingHours).toHaveBeenCalledOnce()
  })

  it('keeps edited input and confirmed state after network and version errors', async () => {
    updateAdminWorkingHours
      .mockRejectedValueOnce(new Error('network unavailable'))
      .mockResolvedValueOnce({
        error: { code: 'version_conflict', message: 'stale' },
        response: { status: 412 },
      })
    const wrapper = mountView()
    await flushPromises()

    const mondayEnd = wrapper.get('input[name="days.0.intervals.0.end_time"]')
    await mondayEnd.setValue('17:00')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(mondayEnd.element).toHaveProperty('value', '17:00')
    expect(wrapper.text()).toContain('Подтверждённая версия: 1')
    expect(wrapper.get('[role="alert"]').text()).toContain('Введённые изменения сохранены')

    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(wrapper.get('[role="alert"]').text()).toContain('уже изменён')
    expect(mondayEnd.element).toHaveProperty('value', '17:00')
    expect(wrapper.text()).toContain('Подтверждённая версия: 1')
  })

  it('blocks repeated saves and confirms navigation away from a dirty form', async () => {
    let resolveSave!: (value: unknown) => void
    updateAdminWorkingHours.mockReturnValueOnce(
      new Promise((resolve) => {
        resolveSave = resolve
      }),
    )
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(false)
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('input[name="days.0.intervals.0.end_time"]').setValue('17:00')
    expect(routeLeave.guard?.()).toBe(false)
    expect(confirm).toHaveBeenCalledOnce()

    await wrapper.get('form').trigger('submit')
    await wrapper.get('form').trigger('submit')
    expect(updateAdminWorkingHours).toHaveBeenCalledOnce()
    expect(wrapper.get('button[type="submit"]').attributes('disabled')).toBeDefined()

    await wrapper.get('input[name="days.0.intervals.0.end_time"]').setValue('16:00')

    resolveSave({
      data: {
        ...structuredClone(initialWorkingHours),
        version: '2',
        days: initialWorkingHours.days.map((day, index) =>
          index === 0 ? { ...day, intervals: [{ start_time: '09:00', end_time: '17:00' }] } : day,
        ),
      },
      response: { headers: new Headers({ ETag: '"2"' }) },
    })
    await flushPromises()
    expect(wrapper.get('input[name="days.0.intervals.0.end_time"]').element).toHaveProperty(
      'value',
      '16:00',
    )
    expect(wrapper.text()).toContain('Подтверждённая версия: 2')
    expect(wrapper.text()).toContain('Есть несохранённые изменения')
    expect(routeLeave.guard?.()).toBe(false)
    confirm.mockRestore()
  })
})
