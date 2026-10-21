export type PrototypeVariant = 'A' | 'B' | 'C'

export interface CallType {
  id: number
  name: string
  durationMinutes: number
  scheduleId: number
  active: boolean
  hasBookings: boolean
}

export interface DayAvailability {
  day: string
  intervals: string[]
}

export interface Schedule {
  id: number
  name: string
  availability: DayAvailability[]
}

export const initialCallTypes: CallType[] = [
  {
    id: 1,
    name: 'Звонок на 15 минут',
    durationMinutes: 15,
    scheduleId: 2,
    active: true,
    hasBookings: true,
  },
  {
    id: 2,
    name: 'Звонок на 30 минут',
    durationMinutes: 30,
    scheduleId: 1,
    active: true,
    hasBookings: true,
  },
  {
    id: 3,
    name: 'Звонок на 60 минут',
    durationMinutes: 60,
    scheduleId: 1,
    active: false,
    hasBookings: false,
  },
]

export const initialSchedules: Schedule[] = [
  {
    id: 1,
    name: 'Основное',
    availability: [
      { day: 'Пн', intervals: ['09:00–13:00', '14:00–18:00'] },
      { day: 'Вт', intervals: ['09:00–18:00'] },
      { day: 'Ср', intervals: ['09:00–13:00', '14:00–18:00'] },
      { day: 'Чт', intervals: ['09:00–18:00'] },
      { day: 'Пт', intervals: ['09:00–16:00'] },
      { day: 'Сб', intervals: [] },
      { day: 'Вс', intervals: [] },
    ],
  },
  {
    id: 2,
    name: 'Короткие звонки',
    availability: [
      { day: 'Пн', intervals: ['10:00–12:00'] },
      { day: 'Вт', intervals: [] },
      { day: 'Ср', intervals: ['10:00–12:00'] },
      { day: 'Чт', intervals: [] },
      { day: 'Пт', intervals: ['10:00–12:00'] },
      { day: 'Сб', intervals: [] },
      { day: 'Вс', intervals: [] },
    ],
  },
]

export const workingHours = 'Пн–Пт, 09:00–18:00 (Europe/Moscow)'

export function cloneCallTypes(): CallType[] {
  return initialCallTypes.map((callType) => ({ ...callType }))
}

export function cloneSchedules(): Schedule[] {
  return initialSchedules.map((schedule) => ({
    ...schedule,
    availability: schedule.availability.map((day) => ({
      ...day,
      intervals: [...day.intervals],
    })),
  }))
}
