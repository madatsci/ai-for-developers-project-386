export type PrototypeVariant = 'A' | 'B' | 'C'

export type BookingStage = 'type' | 'slot' | 'details' | 'confirmation'

export interface CallType {
  slug: string
  name: string
  description: string
  durationMinutes: number
}

export interface CalendarDay {
  key: string
  dayName: string
  dateLabel: string
  relativeLabel: string | null
  slots: string[]
}

export interface BookingPrototypeState {
  stage: BookingStage
  selectedDateKey: string | null
  selectedTime: string | null
  visitorName: string
  visitorEmail: string
  visitorNotes: string
  errors: {
    visitorName?: string
    visitorEmail?: string
    visitorNotes?: string
  }
}

export const prototypeVariants: Array<{
  key: PrototypeVariant
  name: string
}> = [
  { key: 'A', name: 'Спокойный маршрут' },
  { key: 'B', name: 'Планировщик' },
  { key: 'C', name: 'Один вопрос' },
]

export const callTypes: CallType[] = [
  {
    slug: 'zvonok-na-15-minut',
    name: 'Звонок на 15 минут',
    description: 'Коротко обсудить один конкретный вопрос.',
    durationMinutes: 15,
  },
  {
    slug: 'zvonok-na-30-minut',
    name: 'Звонок на 30 минут',
    description: 'Разобрать задачу и наметить следующие шаги.',
    durationMinutes: 30,
  },
  {
    slug: 'zvonok-na-60-minut',
    name: 'Звонок на 60 минут',
    description: 'Подробно погрузиться в тему без спешки.',
    durationMinutes: 60,
  },
]

export const disabledCallTypes: CallType[] = [
  {
    slug: 'razbor-proekta',
    name: 'Разбор проекта',
    description: 'Подробный разбор проекта и возможных следующих шагов.',
    durationMinutes: 45,
  },
]

const slotPatterns = [
  ['09:00', '09:30', '11:15', '14:00', '16:30'],
  ['10:00', '10:45', '12:30', '15:00', '17:15'],
  ['09:15', '11:00', '13:30', '16:00'],
  [],
  ['09:30', '10:30', '14:15', '15:45'],
  [],
  [],
]

function getMoscowDateParts() {
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Europe/Moscow',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).formatToParts(new Date())

  return {
    year: Number(parts.find((part) => part.type === 'year')?.value),
    month: Number(parts.find((part) => part.type === 'month')?.value),
    day: Number(parts.find((part) => part.type === 'day')?.value),
  }
}

export function buildCalendarDays(): CalendarDay[] {
  const { year, month, day } = getMoscowDateParts()
  const dayFormatter = new Intl.DateTimeFormat('ru-RU', { weekday: 'short', timeZone: 'UTC' })
  const dateFormatter = new Intl.DateTimeFormat('ru-RU', {
    day: 'numeric',
    month: 'short',
    timeZone: 'UTC',
  })

  return Array.from({ length: 14 }, (_, index) => {
    const date = new Date(Date.UTC(year, month - 1, day + index, 12))
    const slots = slotPatterns[index % slotPatterns.length] ?? []

    return {
      key: date.toISOString().slice(0, 10),
      dayName: dayFormatter.format(date).replace('.', ''),
      dateLabel: dateFormatter.format(date).replace('.', ''),
      relativeLabel: index === 0 ? 'Сегодня' : index === 1 ? 'Завтра' : null,
      slots,
    }
  })
}

export function formatSelectedDate(day: CalendarDay | undefined, time: string | null) {
  if (!day || !time) return 'Время ещё не выбрано'

  const prefix = day.relativeLabel ? `${day.relativeLabel}, ` : ''
  return `${prefix}${day.dateLabel} в ${time}`
}
