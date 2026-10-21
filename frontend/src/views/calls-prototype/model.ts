export type PrototypeVariant = 'A' | 'B' | 'C'
export type PrototypeScenario = 'ready' | 'loading' | 'error' | 'empty' | 'load-more-error'
export type CallPeriod = 'upcoming' | 'past'

export interface CallItem {
  id: string
  startsAt: string
  endsAt: string
  typeName: string
  durationMinutes: number
  visitorName: string
  visitorEmail: string
  visitorNotes: string | null
}

export interface CallFilters {
  period: CallPeriod
  typeName: string
  query: string
}

export const prototypeNow = new Date('2026-10-20T12:00:00+03:00')

export const prototypeCalls: CallItem[] = [
  {
    id: 'call-101',
    startsAt: '2026-10-20T11:45:00+03:00',
    endsAt: '2026-10-20T12:15:00+03:00',
    typeName: 'Звонок на 30 минут',
    durationMinutes: 30,
    visitorName: 'Анна Смирнова',
    visitorEmail: 'anna@example.com',
    visitorNotes: 'Хочу обсудить переход команды на новый процесс релизов.',
  },
  {
    id: 'call-102',
    startsAt: '2026-10-20T13:00:00+03:00',
    endsAt: '2026-10-20T14:00:00+03:00',
    typeName: 'Звонок на 60 минут',
    durationMinutes: 60,
    visitorName: 'Михаил Орлов',
    visitorEmail: 'm.orlov@example.com',
    visitorNotes: 'Покажу схему сервиса и список открытых вопросов.\nНужна обратная связь по API.',
  },
  {
    id: 'call-103',
    startsAt: '2026-10-20T16:30:00+03:00',
    endsAt: '2026-10-20T16:45:00+03:00',
    typeName: 'Звонок на 15 минут',
    durationMinutes: 15,
    visitorName: 'Елена Ким',
    visitorEmail: 'elena.kim@example.com',
    visitorNotes: null,
  },
  {
    id: 'call-104',
    startsAt: '2026-10-21T09:30:00+03:00',
    endsAt: '2026-10-21T10:00:00+03:00',
    typeName: 'Звонок на 30 минут',
    durationMinutes: 30,
    visitorName: 'Алексей Волков',
    visitorEmail: 'a.volkov@example.com',
    visitorNotes: 'Нужно выбрать стратегию миграции данных.',
  },
  {
    id: 'call-105',
    startsAt: '2026-10-22T15:00:00+03:00',
    endsAt: '2026-10-22T15:30:00+03:00',
    typeName: 'Звонок на 30 минут',
    durationMinutes: 30,
    visitorName: 'Ольга Петрова',
    visitorEmail: 'olga@example.com',
    visitorNotes: null,
  },
  {
    id: 'call-106',
    startsAt: '2026-10-26T10:00:00+03:00',
    endsAt: '2026-10-26T11:00:00+03:00',
    typeName: 'Звонок на 60 минут',
    durationMinutes: 60,
    visitorName: 'Роман Беляев',
    visitorEmail: 'roman@example.com',
    visitorNotes: 'Архитектурный разбор перед запуском проекта.',
  },
  {
    id: 'call-099',
    startsAt: '2026-10-19T17:00:00+03:00',
    endsAt: '2026-10-19T17:15:00+03:00',
    typeName: 'Звонок на 15 минут',
    durationMinutes: 15,
    visitorName: 'Дарья Кузнецова',
    visitorEmail: 'daria@example.com',
    visitorNotes: null,
  },
  {
    id: 'call-098',
    startsAt: '2026-10-16T12:00:00+03:00',
    endsAt: '2026-10-16T12:30:00+03:00',
    typeName: 'Звонок на 30 минут',
    durationMinutes: 30,
    visitorName: 'Иван Соколов',
    visitorEmail: 'ivan@example.com',
    visitorNotes: 'Обсудить результаты технического интервью.',
  },
]

export const callTypeNames = Array.from(new Set(prototypeCalls.map((call) => call.typeName)))

export function getCallStatus(call: CallItem) {
  const startsAt = new Date(call.startsAt)
  const endsAt = new Date(call.endsAt)

  if (startsAt <= prototypeNow && prototypeNow < endsAt) return 'in-progress'
  if (endsAt <= prototypeNow) return 'past'
  return 'upcoming'
}

export function filterCalls(calls: CallItem[], filters: CallFilters) {
  const query = filters.query.trim().toLocaleLowerCase('ru')

  return calls
    .filter((call) => {
      const status = getCallStatus(call)
      const belongsToPeriod = filters.period === 'past' ? status === 'past' : status !== 'past'
      const matchesType = !filters.typeName || call.typeName === filters.typeName
      const matchesQuery =
        !query ||
        [call.visitorName, call.visitorEmail, call.typeName, call.visitorNotes ?? ''].some(
          (value) => value.toLocaleLowerCase('ru').includes(query),
        )

      return belongsToPeriod && matchesType && matchesQuery
    })
    .sort((left, right) => {
      const difference = new Date(left.startsAt).getTime() - new Date(right.startsAt).getTime()
      return filters.period === 'past' ? -difference : difference
    })
}

export function formatDate(value: string, options?: Intl.DateTimeFormatOptions) {
  return new Intl.DateTimeFormat('ru-RU', {
    timeZone: 'Europe/Moscow',
    day: 'numeric',
    month: 'long',
    ...options,
  }).format(new Date(value))
}

export function formatTime(value: string) {
  return new Intl.DateTimeFormat('ru-RU', {
    timeZone: 'Europe/Moscow',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value))
}

export function formatWeekday(value: string) {
  return new Intl.DateTimeFormat('ru-RU', {
    timeZone: 'Europe/Moscow',
    weekday: 'short',
  }).format(new Date(value))
}

export function dateKey(value: string) {
  return new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Europe/Moscow',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(new Date(value))
}
