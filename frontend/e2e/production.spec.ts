import { expect, test } from '@playwright/test'

test.describe.serial('production acceptance', () => {
  test('visitor books a seeded call type and the organizer finds the call', async ({ page }) => {
    await page.goto('/booking')
    await page.getByRole('link', { name: /Звонок на 15 минут/ }).click()

    await expect(page.locator('[data-availability-day]')).toHaveCount(14)
    const firstDayWithSlots = page
      .locator('[data-availability-day]')
      .filter({ has: page.locator('[data-slot]') })
      .first()
    const selectedDate = (await firstDayWithSlots
      .locator('[data-availability-date]')
      .textContent())!.trim()
    const selectedDateLabel = new Intl.DateTimeFormat('ru-RU', {
      weekday: 'long',
      day: 'numeric',
      month: 'long',
      year: 'numeric',
      timeZone: 'Europe/Moscow',
    }).format(new Date(`${selectedDate}T12:00:00+03:00`))
    const selectedSlot = firstDayWithSlots.locator('[data-slot]').last()
    const selectedSlotLabel = await selectedSlot.textContent()
    await selectedSlot.click()
    await page.getByLabel('Имя').fill('Посетитель E2E')
    await page.getByLabel('Email').fill('visitor.e2e@example.com')
    await page.getByLabel('Дополнительная информация').fill('Публичный production E2E')
    await page.getByRole('button', { name: 'Забронировать звонок' }).click()

    const confirmation = page.locator('[data-confirmation]')
    await expect(confirmation).toContainText('Звонок забронирован')
    await expect(confirmation).toContainText('Звонок на 15 минут')
    await expect(confirmation).toContainText(selectedDateLabel)
    await expect(confirmation.getByText('15 минут', { exact: true })).toBeVisible()
    await expect(confirmation).toContainText('Посетитель E2E')
    await expect(confirmation).toContainText('visitor.e2e@example.com')
    await expect(confirmation).toContainText('Публичный production E2E')
    await expect(confirmation).toContainText(selectedSlotLabel!)

    await expect
      .poll(async () => {
        const response = await page.request.get('/api/v1/admin/calls?period=upcoming&q=visitor.e2e')
        expect(response.ok()).toBe(true)
        return (await response.json()).items
      })
      .toEqual([
        expect.objectContaining({
          call_type_name: 'Звонок на 15 минут',
          duration_minutes: 15,
          visitor_name: 'Посетитель E2E',
          visitor_email: 'visitor.e2e@example.com',
          visitor_notes: 'Публичный production E2E',
        }),
      ])

    await page.goto('/admin/calls')
    await page.getByPlaceholder('Поиск по имени, email или комментарию').fill('visitor.e2e')
    await expect(page.getByText('visitor.e2e@example.com').first()).toBeVisible()
  })

  test('organizer changes working hours, schedule and call type before public booking', async ({
    page,
  }) => {
    await page.goto('/admin/working-hours')
    const workingMonday = page.locator('[data-working-day]').first()
    await workingMonday.locator('input[type="time"]').nth(0).fill('10:00')
    await workingMonday.locator('input[type="time"]').nth(1).fill('16:00')
    await page.getByRole('button', { name: 'Сохранить режим работы' }).click()
    await expect(page.getByRole('status')).toHaveText('Режим работы сохранён')

    await page.goto('/admin/schedules')
    const scheduleMonday = page.locator('[data-schedule-day]').first()
    await scheduleMonday.locator('input[type="time"]').nth(0).fill('10:30')
    await scheduleMonday.locator('input[type="time"]').nth(1).fill('15:00')
    await page.getByRole('button', { name: 'Сохранить расписание' }).click()
    await expect(page.getByRole('status')).toHaveText('Расписание сохранено')

    await page.goto('/admin/call-types')
    await page.locator('[data-create-call-type]').click()
    await page.getByLabel('Название').fill('Production E2E')
    await page.getByLabel('Описание').fill('Проверка контейнерного контура')
    await page.getByLabel('Длительность, минут').fill('15')
    await page.getByRole('button', { name: 'Создать тип', exact: true }).click()
    await expect(page.getByRole('status')).toHaveText('Тип звонка создан')

    await page.getByLabel('Название').fill('Production E2E обновлён')
    await page.getByRole('button', { name: 'Сохранить тип' }).click()
    await expect(page.getByRole('status')).toHaveText('Тип звонка сохранён')

    await page.goto('/booking')
    await page.getByRole('link', { name: /Production E2E обновлён/ }).click()
    await expect(page.locator('[data-availability-day]')).toHaveCount(14)
    const mondayWithSlots = page
      .locator('[data-availability-day]')
      .filter({ hasText: /понедельник/i })
      .filter({ has: page.locator('[data-slot]') })
      .first()
    await expect(mondayWithSlots).toBeVisible()
    await expect(mondayWithSlots.locator('[data-slot]').first()).toHaveText('10:30–10:45')
    await mondayWithSlots.locator('[data-slot]').first().click()
    await page.getByLabel('Имя').fill('Администратор E2E')
    await page.getByLabel('Email').fill('admin.e2e@example.com')
    await page.getByRole('button', { name: 'Забронировать звонок' }).click()
    await expect(page.locator('[data-confirmation]')).toContainText('Production E2E обновлён')

    await expect
      .poll(async () => {
        const response = await page.request.get('/api/v1/admin/calls?period=upcoming&q=admin.e2e')
        expect(response.ok()).toBe(true)
        return ((await response.json()).items as unknown[]).length
      })
      .toBe(1)

    await page.goto('/admin/calls')
    await page.getByPlaceholder('Поиск по имени, email или комментарию').fill('admin.e2e')
    await expect(page.getByText('admin.e2e@example.com').first()).toBeVisible()
  })
})
