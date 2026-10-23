import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const {
  createAdminCallType,
  deleteAdminCallType,
  listAdminCallTypes,
  listAdminSchedules,
  route,
  updateAdminCallType,
  routeLeave,
} = vi.hoisted(() => ({
  createAdminCallType: vi.fn(),
  deleteAdminCallType: vi.fn(),
  listAdminCallTypes: vi.fn(),
  listAdminSchedules: vi.fn(),
  route: { query: {} as Record<string, string> },
  updateAdminCallType: vi.fn(),
  routeLeave: { guard: undefined as undefined | (() => boolean) },
}))

vi.mock('@/api/generated', () => ({
  createAdminCallType,
  deleteAdminCallType,
  listAdminCallTypes,
  listAdminSchedules,
  updateAdminCallType,
}))

vi.mock('vue-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-router')>()
  return {
    ...actual,
    useRoute: () => route,
    onBeforeRouteLeave: (guard: () => boolean) => {
      routeLeave.guard = guard
    },
  }
})

import AdminCallTypesView from '@/views/AdminCallTypesView.vue'

const schedule = {
  id: '018f0000-0000-7000-8000-000000000101',
  name: 'Основное',
  version: '1',
  assigned_call_types: [],
  can_delete: false,
  delete_blocked_reason: 'schedule_in_use',
  days: Array.from({ length: 7 }, (_, index) => ({
    day_of_week: index + 1,
    intervals: index < 5 ? [{ start_time: '09:00', end_time: '18:00' }] : [],
  })),
}

const callType = {
  id: '018f0000-0000-7000-8000-000000000201',
  slug: 'zvonok-na-30-minut',
  name: 'Звонок на 30 минут',
  description: null,
  duration_minutes: 30,
  status: 'active',
  schedule: { id: schedule.id, name: schedule.name },
  version: '1',
  can_delete: true,
  delete_blocked_reason: null,
}

function mountView() {
  return mount(AdminCallTypesView, {
    global: { stubs: { RouterLink: RouterLinkStub } },
  })
}

describe('AdminCallTypesView', () => {
  beforeEach(() => {
    createAdminCallType.mockReset()
    deleteAdminCallType.mockReset()
    listAdminCallTypes.mockReset()
    listAdminSchedules.mockReset()
    updateAdminCallType.mockReset()
    route.query = {}
    routeLeave.guard = undefined
    listAdminCallTypes.mockResolvedValue({ data: { items: [structuredClone(callType)] } })
    listAdminSchedules.mockResolvedValue({ data: { items: [structuredClone(schedule)] } })
  })

  it('shows the confirmed list and creates an active type with the single schedule selected', async () => {
    const created = {
      ...structuredClone(callType),
      id: '018f0000-0000-7000-8000-000000000202',
      slug: 'razbor-proekta',
      name: 'Разбор проекта',
      description: 'Обсудим архитектуру',
      duration_minutes: 45,
    }
    createAdminCallType.mockResolvedValue({ data: created })
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('Звонок на 30 минут')
    expect(wrapper.text()).toContain('30 минут · Основное · Активен')

    await wrapper.get('[data-create-call-type]').trigger('click')
    expect(wrapper.get('input[name="call-type-duration"]').element).toHaveProperty('value', '30')
    expect(wrapper.get('select[name="call-type-schedule"]').element).toHaveProperty(
      'value',
      schedule.id,
    )
    await wrapper.get('input[name="call-type-name"]').setValue('Разбор проекта')
    await wrapper.get('textarea[name="call-type-description"]').setValue('Обсудим архитектуру')
    await wrapper.get('input[name="call-type-duration"]').setValue('45')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(createAdminCallType).toHaveBeenCalledExactlyOnceWith({
      body: {
        name: 'Разбор проекта',
        description: 'Обсудим архитектуру',
        duration_minutes: 45,
        schedule_id: schedule.id,
      },
    })
    expect(wrapper.text()).toContain('Тип звонка создан')
    expect(wrapper.text()).toContain('razbor-proekta')
  })

  it('uses separate list and editor states on mobile', async () => {
    const wrapper = mountView()
    await flushPromises()

    const list = wrapper.get('[aria-label="Список типов звонков"]')
    const form = wrapper.get('form')
    expect(list.classes()).toContain('block')
    expect(form.classes()).toContain('hidden')

    await wrapper.get(`[data-select-call-type="${callType.id}"]`).trigger('click')
    expect(list.classes()).toContain('hidden')
    expect(form.classes()).toContain('block')

    await wrapper.get('[data-mobile-call-type-back]').trigger('click')
    expect(list.classes()).toContain('block')
    expect(form.classes()).toContain('hidden')
  })

  it('opens the call type editor selected by the route', async () => {
    const selectedCallType = {
      ...structuredClone(callType),
      id: '018f0000-0000-7000-8000-000000000202',
      slug: 'dlinnaya-konsultatsiya',
      name: 'Длинная консультация',
      duration_minutes: 60,
    }
    route.query = { call_type_id: selectedCallType.id }
    listAdminCallTypes.mockResolvedValue({
      data: { items: [structuredClone(callType), selectedCallType] },
    })

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.get('input[name="call-type-name"]').element).toHaveProperty(
      'value',
      'Длинная консультация',
    )
    expect(wrapper.get('[aria-label="Список типов звонков"]').classes()).toContain('hidden')
    expect(wrapper.get('form').classes()).toContain('block')
  })

  it('updates with If-Match, preserves edits on conflict, and protects dirty navigation', async () => {
    updateAdminCallType.mockResolvedValue({
      error: { code: 'version_conflict', message: 'stale' },
      response: { status: 412 },
    })
    vi.spyOn(window, 'confirm').mockReturnValue(false)
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('input[name="call-type-name"]').setValue('Новое название')
    expect(routeLeave.guard?.()).toBe(false)
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(updateAdminCallType).toHaveBeenCalledExactlyOnceWith({
      path: { id: callType.id },
      headers: { 'If-Match': '"1"' },
      body: {
        name: 'Новое название',
        description: null,
        duration_minutes: 30,
        status: 'active',
        schedule_id: schedule.id,
      },
    })
    expect(wrapper.get('input[name="call-type-name"]').element).toHaveProperty(
      'value',
      'Новое название',
    )
    expect(wrapper.text()).toContain('Тип звонка уже изменён')
    vi.restoreAllMocks()
  })

  it('copies the stable link and confirms deletion with the current version', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.defineProperty(navigator, 'clipboard', { configurable: true, value: { writeText } })
    deleteAdminCallType.mockResolvedValue({ response: { status: 204 } })
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-copy-call-type-link]').trigger('click')
    expect(writeText).toHaveBeenCalledWith(`${window.location.origin}/booking/${callType.slug}`)
    await wrapper.get('[data-delete-call-type]').trigger('click')
    await flushPromises()

    expect(deleteAdminCallType).toHaveBeenCalledExactlyOnceWith({
      path: { id: callType.id },
      headers: { 'If-Match': '"1"' },
    })
    expect(wrapper.text()).toContain('Тип звонка удалён')
    vi.restoreAllMocks()
  })
})
