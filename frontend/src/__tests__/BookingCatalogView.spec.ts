import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { listPublicCallTypes } = vi.hoisted(() => ({
  listPublicCallTypes: vi.fn(),
}))

vi.mock('@/api/generated', () => ({ listPublicCallTypes }))

import BookingCatalogView from '@/views/BookingCatalogView.vue'

describe('BookingCatalogView', () => {
  beforeEach(() => {
    listPublicCallTypes.mockReset()
  })

  it('shows loading state and then renders call types returned by the generated SDK', async () => {
    let resolveRequest!: (value: unknown) => void
    listPublicCallTypes.mockReturnValueOnce(
      new Promise((resolve) => {
        resolveRequest = resolve
      }),
    )

    const wrapper = mount(BookingCatalogView, {
      global: { stubs: { RouterLink: RouterLinkStub } },
    })

    expect(wrapper.get('[aria-busy="true"]').text()).toContain('Загружаем типы звонков')

    resolveRequest({
      data: {
        items: [
          {
            slug: 'zvonok-na-15-minut',
            name: 'Звонок на 15 минут',
            description: null,
            duration_minutes: 15,
          },
        ],
      },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Звонок на 15 минут')
    expect(wrapper.text()).toContain('15 мин')
    expect(wrapper.text()).not.toContain('Загружаем типы звонков')
  })

  it('shows a load error and retries through the generated SDK', async () => {
    listPublicCallTypes
      .mockResolvedValueOnce({ error: { code: 'internal_error' } })
      .mockResolvedValueOnce({ data: { items: [] } })

    const wrapper = mount(BookingCatalogView, {
      global: { stubs: { RouterLink: RouterLinkStub } },
    })
    await flushPromises()

    expect(wrapper.get('[role="alert"]').text()).toContain('Не удалось загрузить типы звонков')
    await wrapper.get('button').trigger('click')
    await flushPromises()

    expect(listPublicCallTypes).toHaveBeenCalledTimes(2)
    expect(wrapper.text()).toContain('Сейчас нет доступных типов звонков')
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
  })

  it('shows an honest empty state when the API returns no active call types', async () => {
    listPublicCallTypes.mockResolvedValueOnce({ data: { items: [] } })

    const wrapper = mount(BookingCatalogView, {
      global: { stubs: { RouterLink: RouterLinkStub } },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Сейчас нет доступных типов звонков')
    expect(wrapper.text()).toContain('Попробуйте вернуться позже')
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
  })
})
