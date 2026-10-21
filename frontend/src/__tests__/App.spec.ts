import { beforeEach, describe, expect, it } from 'vitest'

import { flushPromises, mount } from '@vue/test-utils'
import App from '../App.vue'
import router from '../router'

describe('App', () => {
  beforeEach(() => {
    window.sessionStorage.clear()
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

    await wrapper.get('button').trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.path).toBe('/booking/zvonok-na-15-minut')
    expect(wrapper.text()).toContain('Свободное время')
    wrapper.unmount()
  })

  it('keeps the selected planner flow interactive and restores its confirmation', async () => {
    await router.push('/booking?variant=B')
    await router.isReady()

    const wrapper = mount(App, { global: { plugins: [router] } })

    await wrapper
      .findAll('button')
      .find((button) => button.text().includes('Звонок на 15 минут'))!
      .trigger('click')
    await flushPromises()

    await wrapper
      .findAll('button')
      .find((button) => button.text() === '09:00')!
      .trigger('click')

    await wrapper.get('input[type="text"]').setValue('Тестовый посетитель')
    await wrapper.get('input[type="email"]').setValue('test@EXAMPLE.COM')
    await wrapper.get('textarea').setValue('Нужно обсудить архитектуру.\nПодготовлю схему заранее.')
    await wrapper.get('form').trigger('submit')

    expect(wrapper.text()).toContain('Звонок забронирован')
    expect(wrapper.text()).toContain('test@example.com')
    expect(wrapper.text()).toContain('Нужно обсудить архитектуру.')
    wrapper.unmount()

    const refreshedWrapper = mount(App, { global: { plugins: [router] } })
    await flushPromises()

    expect(refreshedWrapper.text()).toContain('Звонок забронирован')
    expect(refreshedWrapper.text()).toContain('Нужно обсудить архитектуру.')
    refreshedWrapper.unmount()
  })

  it('preserves contact details while the visitor changes the call type', async () => {
    await router.push('/booking?variant=B')
    await router.isReady()

    const wrapper = mount(App, { global: { plugins: [router] } })

    await wrapper
      .findAll('button')
      .find((button) => button.text().includes('Звонок на 15 минут'))!
      .trigger('click')
    await flushPromises()
    await wrapper
      .findAll('button')
      .find((button) => button.text() === '09:00')!
      .trigger('click')
    await wrapper.get('input[type="text"]').setValue('Ирина')
    await wrapper.get('input[type="email"]').setValue('irina@example.com')
    await wrapper.get('textarea').setValue('Хочу обсудить интеграцию')

    await wrapper
      .findAll('button')
      .find((button) => button.text().includes('Все типы'))!
      .trigger('click')
    await flushPromises()
    await wrapper
      .findAll('button')
      .find((button) => button.text().includes('Звонок на 30 минут'))!
      .trigger('click')
    await flushPromises()

    expect((wrapper.get('input[type="text"]').element as HTMLInputElement).value).toBe('Ирина')
    expect((wrapper.get('input[type="email"]').element as HTMLInputElement).value).toBe(
      'irina@example.com',
    )
    expect((wrapper.get('textarea').element as HTMLTextAreaElement).value).toBe(
      'Хочу обсудить интеграцию',
    )
    expect(wrapper.findAll('button').some((button) => button.text() === '09:00')).toBe(true)
    expect(wrapper.findAll('button[aria-pressed="true"]')).toHaveLength(0)
    wrapper.unmount()
  })

  it.each([
    ['/booking/razbor-proekta?variant=B', 'Тип звонка временно недоступен', 'Разбор проекта'],
    ['/booking/neizvestnyy-tip?variant=B', 'Тип звонка не найден', 'Эта ссылка больше не работает'],
  ])('renders the expected unavailable state for %s', async (path, status, heading) => {
    await router.push(path)
    await router.isReady()

    const wrapper = mount(App, { global: { plugins: [router] } })

    expect(wrapper.text()).toContain(status)
    expect(wrapper.get('h1').text()).toBe(heading)
    expect(wrapper.text()).toContain('К активным типам звонков')
    wrapper.unmount()
  })

  it.each([
    ['B', 'Свободное время'],
    ['C', 'Когда вам удобно?'],
  ])('renders variant %s on a direct public booking URL', async (variant, heading) => {
    await router.push(`/booking/zvonok-na-30-minut?variant=${variant}`)
    await router.isReady()

    const wrapper = mount(App, { global: { plugins: [router] } })

    expect(wrapper.text()).toContain(heading)
    expect(wrapper.text()).toContain('Звонок на 30 минут')
    wrapper.unmount()
  })
})
