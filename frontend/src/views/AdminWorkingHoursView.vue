<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { onBeforeRouteLeave } from 'vue-router'
import AdminSidebar from '@/components/AdminSidebar.vue'
import {
  getAdminWorkingHours,
  updateAdminWorkingHours,
  type WeeklyDay,
  type WorkingHours,
} from '@/api/generated'

const dayNames = ['Понедельник', 'Вторник', 'Среда', 'Четверг', 'Пятница', 'Суббота', 'Воскресенье']

const loading = ref(true)
const saving = ref(false)
const loadFailed = ref(false)
const saveError = ref('')
const savedNotice = ref(false)
const days = ref<WorkingHours['days'] | null>(null)
const confirmedDays = ref<WorkingHours['days'] | null>(null)
const version = ref('')
const etag = ref('')
const validationAttempted = ref(false)
const dayErrors = ref<Record<number, string>>({})

const dirty = computed(
  () =>
    days.value !== null &&
    confirmedDays.value !== null &&
    JSON.stringify(days.value) !== JSON.stringify(confirmedDays.value),
)

function responseETag(response: Response | undefined) {
  const value = response?.headers.get('ETag')
  if (!value) throw new Error('Working-hours response has no ETag')
  return value
}

function cloneDays(source: WorkingHours['days']): WorkingHours['days'] {
  return source.map((day) => ({
    day_of_week: day.day_of_week,
    intervals: day.intervals.map((interval) => ({ ...interval })),
  })) as WorkingHours['days']
}

function acceptConfirmedState(data: WorkingHours, response: Response | undefined) {
  const nextDays = cloneDays(data.days)
  days.value = nextDays
  confirmedDays.value = cloneDays(nextDays)
  version.value = data.version
  etag.value = responseETag(response)
  validationAttempted.value = false
  dayErrors.value = {}
}

function timeMinutes(value: string) {
  const match = /^(\d{2}):(\d{2})$/.exec(value)
  if (!match) return null
  const hours = Number(match[1])
  const minutes = Number(match[2])
  if (hours > 23 || minutes > 59 || minutes % 5 !== 0) return null
  return hours * 60 + minutes
}

function validateDays() {
  const errors: Record<number, string> = {}
  for (const [dayIndex, day] of (days.value ?? []).entries()) {
    const intervals = day.intervals
      .map((interval) => ({
        start: timeMinutes(interval.start_time),
        end: timeMinutes(interval.end_time),
      }))
      .sort((left, right) => (left.start ?? -1) - (right.start ?? -1))
    for (const [intervalIndex, interval] of intervals.entries()) {
      if (interval.start === null || interval.end === null) {
        errors[dayIndex] = 'Границы интервала должны быть на пятиминутной сетке.'
        break
      }
      if (interval.end <= interval.start) {
        errors[dayIndex] = 'Конец интервала должен быть позже начала.'
        break
      }
      if (interval.end - interval.start < 15) {
        errors[dayIndex] = 'Интервал должен длиться не меньше 15 минут.'
        break
      }
      const previous = intervals[intervalIndex - 1]
      if (previous?.end !== null && previous?.end !== undefined && interval.start < previous.end) {
        errors[dayIndex] = 'Интервалы одного дня не должны пересекаться.'
        break
      }
    }
  }
  dayErrors.value = errors
  return Object.keys(errors).length === 0
}

async function loadWorkingHours() {
  loading.value = true
  loadFailed.value = false
  saveError.value = ''
  try {
    const result = await getAdminWorkingHours()
    if (!result.data) {
      loadFailed.value = true
      return
    }
    acceptConfirmedState(result.data, result.response)
  } catch {
    loadFailed.value = true
  } finally {
    loading.value = false
  }
}

function addInterval(day: WeeklyDay) {
  day.intervals.push({ start_time: '09:00', end_time: '18:00' })
  markEdited()
}

function removeInterval(day: WeeklyDay, intervalIndex: number) {
  day.intervals.splice(intervalIndex, 1)
  markEdited()
}

function markEdited() {
  savedNotice.value = false
  saveError.value = ''
  if (validationAttempted.value) validateDays()
}

async function saveWorkingHours() {
  if (saving.value || !dirty.value || !days.value) return
  validationAttempted.value = true
  if (!validateDays()) return
  saving.value = true
  saveError.value = ''
  savedNotice.value = false
  const submittedDays = cloneDays(days.value)
  try {
    const result = await updateAdminWorkingHours({
      body: { days: submittedDays },
      headers: { 'If-Match': etag.value },
    })
    if (!result.data) {
      if (result.error && 'field_errors' in result.error && result.error.field_errors) {
        const errors: Record<number, string> = {}
        for (const error of result.error.field_errors) {
          const match = /^body\.days\.(\d+)/.exec(error.path)
          if (match) errors[Number(match[1])] = error.message
        }
        dayErrors.value = errors
      }
      saveError.value =
        result.error?.code === 'version_conflict'
          ? 'Режим работы уже изменён. Обновите страницу и повторите изменения.'
          : 'Не удалось сохранить режим работы. Исправьте ошибки или повторите попытку.'
      return
    }
    if (JSON.stringify(days.value) === JSON.stringify(submittedDays)) {
      acceptConfirmedState(result.data, result.response)
    } else {
      confirmedDays.value = cloneDays(result.data.days)
      version.value = result.data.version
      etag.value = responseETag(result.response)
      validateDays()
    }
    savedNotice.value = true
  } catch {
    saveError.value = 'Не удалось связаться с сервером. Введённые изменения сохранены в форме.'
  } finally {
    saving.value = false
  }
}

function confirmDiscard() {
  return !dirty.value || window.confirm('Отменить несохранённые изменения режима работы?')
}

function handleBeforeUnload(event: BeforeUnloadEvent) {
  if (!dirty.value) return
  event.preventDefault()
}

onBeforeRouteLeave(confirmDiscard)
onMounted(() => {
  window.addEventListener('beforeunload', handleBeforeUnload)
  void loadWorkingHours()
})
onBeforeUnmount(() => window.removeEventListener('beforeunload', handleBeforeUnload))
</script>

<template>
  <div class="min-h-svh bg-slate-100 text-slate-950 lg:grid lg:grid-cols-[17rem_1fr]">
    <AdminSidebar
      active="working-hours"
      note="Общая граница времени для всех расписаний и типов звонков."
    />

    <main class="px-4 py-8 sm:px-8 lg:px-12 lg:py-10">
      <div class="mx-auto max-w-5xl">
        <header class="border-b border-slate-300 pb-7">
          <p class="text-xs font-bold tracking-[0.2em] text-blue-700 uppercase">Организатор</p>
          <h1 class="mt-3 text-3xl font-bold tracking-[-0.035em] sm:text-4xl">Режим работы</h1>
          <p class="mt-3 max-w-2xl leading-7 text-slate-600">
            Укажите общие часы, когда вы проводите звонки. Расписания могут быть шире, но свободные
            слоты появляются только внутри режима работы.
          </p>
        </header>

        <div v-if="loading" class="mt-8 rounded-xl bg-white p-8" aria-busy="true">
          Загружаем режим работы…
        </div>

        <div
          v-else-if="loadFailed"
          class="mt-8 rounded-xl border border-red-200 bg-red-50 p-6"
          role="alert"
        >
          <h2 class="text-lg font-bold">Не удалось загрузить режим работы</h2>
          <p class="mt-2 text-sm text-slate-600">Проверьте соединение и попробуйте ещё раз.</p>
          <button
            type="button"
            class="mt-5 rounded-lg bg-slate-950 px-4 py-2 text-sm font-semibold text-white"
            @click="loadWorkingHours"
          >
            Повторить
          </button>
        </div>

        <form v-else-if="days" class="mt-8" @submit.prevent="saveWorkingHours">
          <section class="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
            <div
              class="border-b border-slate-200 px-5 py-4 sm:flex sm:items-center sm:justify-between"
            >
              <div>
                <h2 class="font-bold">Недельные интервалы</h2>
                <p class="mt-1 text-sm text-slate-500">Europe/Moscow · шаг 5 минут</p>
              </div>
              <p class="mt-2 font-mono text-xs text-slate-500 sm:mt-0">
                Подтверждённая версия: {{ version }}
              </p>
            </div>

            <div class="divide-y divide-slate-200">
              <article
                v-for="(day, dayIndex) in days"
                :key="day.day_of_week"
                data-working-day
                class="grid gap-4 px-5 py-5 md:grid-cols-[10rem_1fr_auto] md:items-start"
              >
                <div>
                  <h3 class="font-semibold">{{ dayNames[dayIndex] }}</h3>
                  <p v-if="day.intervals.length === 0" class="mt-1 text-sm text-slate-500">
                    Выходной
                  </p>
                </div>

                <div class="space-y-3">
                  <div
                    v-for="(interval, intervalIndex) in day.intervals"
                    :key="intervalIndex"
                    class="flex flex-wrap items-center gap-2"
                  >
                    <label class="sr-only" :for="`day-${dayIndex}-start-${intervalIndex}`"
                      >Начало</label
                    >
                    <input
                      :id="`day-${dayIndex}-start-${intervalIndex}`"
                      v-model="interval.start_time"
                      :name="`days.${dayIndex}.intervals.${intervalIndex}.start_time`"
                      type="time"
                      step="300"
                      class="rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-sm"
                      @input="markEdited"
                    />
                    <span class="text-slate-400">—</span>
                    <label class="sr-only" :for="`day-${dayIndex}-end-${intervalIndex}`"
                      >Конец</label
                    >
                    <input
                      :id="`day-${dayIndex}-end-${intervalIndex}`"
                      v-model="interval.end_time"
                      :name="`days.${dayIndex}.intervals.${intervalIndex}.end_time`"
                      type="time"
                      step="300"
                      class="rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-sm"
                      @input="markEdited"
                    />
                    <button
                      type="button"
                      class="rounded-lg px-3 py-2 text-sm font-semibold text-red-700 hover:bg-red-50"
                      :aria-label="`Удалить интервал: ${dayNames[dayIndex]}`"
                      @click="removeInterval(day, intervalIndex)"
                    >
                      Удалить
                    </button>
                  </div>
                  <p
                    v-if="dayErrors[dayIndex]"
                    :data-day-error="dayIndex"
                    class="text-sm font-medium text-red-700"
                  >
                    {{ dayErrors[dayIndex] }}
                  </p>
                </div>

                <button
                  type="button"
                  data-add-interval
                  class="justify-self-start rounded-lg border border-slate-300 px-3 py-2 text-sm font-semibold hover:bg-slate-50"
                  @click="addInterval(day)"
                >
                  + Интервал
                </button>
              </article>
            </div>
          </section>

          <div
            class="mt-6 border-t border-slate-300 bg-slate-100/95 py-5 backdrop-blur sm:sticky sm:bottom-0"
          >
            <p v-if="saveError" class="mb-3 text-sm font-medium text-red-700" role="alert">
              {{ saveError }}
            </p>
            <p v-if="savedNotice" class="mb-3 text-sm font-medium text-emerald-700" role="status">
              Режим работы сохранён
            </p>
            <div class="flex flex-wrap items-center justify-between gap-3">
              <p class="text-sm text-slate-500">
                {{ dirty ? 'Есть несохранённые изменения' : 'Все изменения сохранены' }}
              </p>
              <button
                type="submit"
                class="rounded-lg bg-blue-600 px-5 py-2.5 text-sm font-semibold text-white disabled:cursor-not-allowed disabled:opacity-45"
                :disabled="saving || !dirty"
              >
                {{ saving ? 'Сохраняем…' : 'Сохранить режим работы' }}
              </button>
            </div>
          </div>
        </form>
      </div>
    </main>
  </div>
</template>
