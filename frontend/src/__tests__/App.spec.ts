import { describe, expect, it } from 'vitest'

import { flushPromises, mount } from '@vue/test-utils'
import App from '../App.vue'
import router from '../router'

describe('App', () => {
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
    expect(wrapper.text()).toContain('Выбор времени появится здесь')
    wrapper.unmount()
  })

  it('renders the booking destination directly and links back home', async () => {
    await router.push('/booking')
    await router.isReady()

    const wrapper = mount(App, { global: { plugins: [router] } })

    expect(wrapper.get('h1').text()).toBe('Бронирование звонка')
    expect(wrapper.get('a[href="/"]').text()).toBe('Вернуться на главную')
    wrapper.unmount()
  })
})
