import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const {
  createAdminSchedule,
  deleteAdminSchedule,
  getAdminWorkingHours,
  listAdminSchedules,
  updateAdminSchedule,
  routeLeave,
} = vi.hoisted(() => ({
  createAdminSchedule: vi.fn(),
  deleteAdminSchedule: vi.fn(),
  getAdminWorkingHours: vi.fn(),
  listAdminSchedules: vi.fn(),
  updateAdminSchedule: vi.fn(),
  routeLeave: { guard: undefined as undefined | (() => boolean) },
}))

vi.mock('@/api/generated', () => ({
  createAdminSchedule,
  deleteAdminSchedule,
  getAdminWorkingHours,
  listAdminSchedules,
  updateAdminSchedule,
}))

vi.mock('vue-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-router')>()
  return {
    ...actual,
    onBeforeRouteLeave: (guard: () => boolean) => {
      routeLeave.guard = guard
    },
  }
})

import AdminSchedulesView from '@/views/AdminSchedulesView.vue'

const mainSchedule = {
  id: '018f0000-0000-7000-8000-000000000101',
  name: 'Основное',
  version: '1',
  assigned_call_types: [
    { id: '018f0000-0000-7000-8000-000000000201', name: 'Короткий звонок', status: 'active' },
    { id: '018f0000-0000-7000-8000-000000000202', name: 'Длинный звонок', status: 'disabled' },
  ],
  can_delete: false,
  delete_blocked_reason: 'schedule_in_use',
  days: Array.from({ length: 7 }, (_, index) => ({
    day_of_week: index + 1,
    intervals: index < 5 ? [{ start_time: '09:00', end_time: '18:00' }] : [],
  })),
}

const workingHours = {
  timezone: 'Europe/Moscow',
  version: '1',
  days: Array.from({ length: 7 }, (_, index) => ({
    day_of_week: index + 1,
    intervals: index < 5 ? [{ start_time: '10:00', end_time: '17:00' }] : [],
  })),
}

function mountView() {
  return mount(AdminSchedulesView, {
    global: { stubs: { RouterLink: RouterLinkStub } },
  })
}

describe('AdminSchedulesView', () => {
  beforeEach(() => {
    createAdminSchedule.mockReset()
    deleteAdminSchedule.mockReset()
    getAdminWorkingHours.mockReset()
    listAdminSchedules.mockReset()
    updateAdminSchedule.mockReset()
    routeLeave.guard = undefined
    listAdminSchedules.mockResolvedValue({ data: { items: [structuredClone(mainSchedule)] } })
    getAdminWorkingHours.mockResolvedValue({ data: structuredClone(workingHours) })
  })

  it('shows assignments and creates a schedule with frontend week defaults', async () => {
    const created = {
      ...structuredClone(mainSchedule),
      id: '018f0000-0000-7000-8000-000000000102',
      name: 'Вечернее',
      assigned_call_types: [],
      can_delete: true,
      delete_blocked_reason: null,
    }
    createAdminSchedule.mockResolvedValue({
      data: created,
      response: { headers: new Headers({ ETag: '"1"' }) },
    })
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('2 типа: Короткий звонок, Длинный звонок')
    expect(wrapper.text()).toContain('Расписание назначено типам звонков')

    const assignmentLinks = wrapper
      .findAllComponents(RouterLinkStub)
      .filter((link) => link.attributes('data-assigned-call-type') !== undefined)
    expect(assignmentLinks.map((link) => link.props('to'))).toEqual([
      {
        name: 'admin-call-types',
        query: { call_type_id: mainSchedule.assigned_call_types[0]!.id },
      },
      {
        name: 'admin-call-types',
        query: { call_type_id: mainSchedule.assigned_call_types[1]!.id },
      },
    ])

    await wrapper.get('[data-create-schedule]').trigger('click')
    expect(wrapper.findAll('[data-schedule-day]')).toHaveLength(7)
    expect(wrapper.get('input[name="days.0.intervals.0.start_time"]').element).toHaveProperty(
      'value',
      '09:00',
    )
    expect(wrapper.find('input[name="days.5.intervals.0.start_time"]').exists()).toBe(false)

    await wrapper.get('input[name="schedule-name"]').setValue('Вечернее')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(createAdminSchedule).toHaveBeenCalledExactlyOnceWith({
      body: {
        name: 'Вечернее',
        days: expect.arrayContaining([
          { day_of_week: 1, intervals: [{ start_time: '09:00', end_time: '18:00' }] },
          { day_of_week: 7, intervals: [] },
        ]),
      },
    })
    expect(wrapper.text()).toContain('Расписание создано')
    expect(wrapper.text()).toContain('Вечернее')
  })

  it('marks ineffective parts and reports interval errors at the affected day', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.get('[data-ineffective="0-0"]').text()).toContain('09:00–10:00')
    expect(wrapper.get('[data-ineffective="0-0"]').text()).toContain('17:00–18:00')

    await wrapper.get('[data-add-interval="0"]').trigger('click')
    await wrapper.get('form').trigger('submit')

    expect(updateAdminSchedule).not.toHaveBeenCalled()
    expect(wrapper.get('[data-day-error="0"]').text()).toContain('пересекаться')

    await wrapper.get('input[name="days.0.intervals.1.start_time"]').setValue('18:00')
    await wrapper.get('input[name="days.0.intervals.1.end_time"]').setValue('20:00')
    expect(wrapper.find('[data-day-error="0"]').exists()).toBe(false)
  })

  it('updates conditionally, preserves edits on conflicts, and explains deletion restrictions', async () => {
    updateAdminSchedule.mockResolvedValue({
      error: { code: 'version_conflict', message: 'stale' },
      response: { status: 412 },
    })
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('input[name="schedule-name"]').setValue('Изменённое')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(updateAdminSchedule).toHaveBeenCalledWith({
      path: { id: mainSchedule.id },
      headers: { 'If-Match': '"1"' },
      body: expect.objectContaining({ name: 'Изменённое' }),
    })
    expect(wrapper.get('input[name="schedule-name"]').element).toHaveProperty('value', 'Изменённое')
    expect(wrapper.text()).toContain('Расписание уже изменено')
    expect(wrapper.text()).toContain('Подтверждённая версия: 1')
    expect(wrapper.get('[data-delete-schedule]').attributes('disabled')).toBeDefined()
    expect(deleteAdminSchedule).not.toHaveBeenCalled()
  })

  it('explains that the last schedule cannot be deleted', async () => {
    listAdminSchedules.mockResolvedValue({
      data: {
        items: [
          {
            ...structuredClone(mainSchedule),
            assigned_call_types: [],
            delete_blocked_reason: 'last_schedule',
          },
        ],
      },
    })
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('Нельзя удалить последнее расписание')
    expect(wrapper.get('[data-delete-schedule]').attributes('disabled')).toBeDefined()
  })

  it('deletes an allowed schedule with its current version and selects the remaining item', async () => {
    const removable = {
      ...structuredClone(mainSchedule),
      id: '018f0000-0000-7000-8000-000000000102',
      name: 'Свободное',
      version: '4',
      assigned_call_types: [],
      can_delete: true,
      delete_blocked_reason: null,
    }
    listAdminSchedules.mockResolvedValue({
      data: { items: [structuredClone(mainSchedule), removable] },
    })
    deleteAdminSchedule.mockResolvedValue({ response: { status: 204 } })
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get(`[data-select-schedule="${removable.id}"]`).trigger('click')
    await wrapper.get('[data-delete-schedule]').trigger('click')
    await flushPromises()

    expect(deleteAdminSchedule).toHaveBeenCalledExactlyOnceWith({
      path: { id: removable.id },
      headers: { 'If-Match': '"4"' },
    })
    expect(wrapper.text()).toContain('Расписание удалено')
    expect(wrapper.get('input[name="schedule-name"]').element).toHaveProperty('value', 'Основное')
    vi.restoreAllMocks()
  })
})
