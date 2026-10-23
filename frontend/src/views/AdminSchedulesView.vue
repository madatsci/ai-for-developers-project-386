<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { onBeforeRouteLeave, RouterLink } from 'vue-router'
import AdminSidebar from '@/components/AdminSidebar.vue'
import {
  createAdminSchedule,
  deleteAdminSchedule,
  getAdminWorkingHours,
  listAdminSchedules,
  updateAdminSchedule,
  type AdminSchedule,
  type WeeklyDay,
  type WorkingHours,
} from '@/api/generated'

const dayNames = ['Понедельник', 'Вторник', 'Среда', 'Четверг', 'Пятница', 'Суббота', 'Воскресенье']

type ScheduleForm = { name: string; days: AdminSchedule['days'] }
type ApiError = {
  code?: string
  field_errors?: Array<{ path: string; message: string }>
}

const loading = ref(true)
const loadFailed = ref(false)
const schedules = ref<AdminSchedule[]>([])
const workingHours = ref<WorkingHours['days'] | null>(null)
const selectedId = ref<string | null>(null)
const creating = ref(false)
const form = ref<ScheduleForm | null>(null)
const confirmedForm = ref<ScheduleForm | null>(null)
const saving = ref(false)
const deleting = ref(false)
const validationAttempted = ref(false)
const nameError = ref('')
const dayErrors = ref<Record<number, string>>({})
const actionError = ref('')
const notice = ref('')

const selectedSchedule = computed(
  () => schedules.value.find((schedule) => schedule.id === selectedId.value) ?? null,
)
const dirty = computed(
  () =>
    form.value !== null &&
    confirmedForm.value !== null &&
    JSON.stringify(form.value) !== JSON.stringify(confirmedForm.value),
)

function cloneDays(source: AdminSchedule['days'] | WorkingHours['days']): AdminSchedule['days'] {
  return source.map((day) => ({
    day_of_week: day.day_of_week,
    intervals: day.intervals.map((interval) => ({ ...interval })),
  })) as AdminSchedule['days']
}

function cloneForm(source: ScheduleForm): ScheduleForm {
  return { name: source.name, days: cloneDays(source.days) }
}

function defaultDays(): AdminSchedule['days'] {
  return Array.from({ length: 7 }, (_, index) => ({
    day_of_week: index + 1,
    intervals: index < 5 ? [{ start_time: '09:00', end_time: '18:00' }] : [],
  })) as AdminSchedule['days']
}

function acceptSchedule(schedule: AdminSchedule) {
  const next = { name: schedule.name, days: cloneDays(schedule.days) }
  selectedId.value = schedule.id
  creating.value = false
  form.value = next
  confirmedForm.value = cloneForm(next)
  validationAttempted.value = false
  nameError.value = ''
  dayErrors.value = {}
}

function startCreating() {
  if (!confirmDiscard()) return
  const next = { name: '', days: defaultDays() }
  selectedId.value = null
  creating.value = true
  form.value = next
  confirmedForm.value = cloneForm(next)
  resetFeedback()
}

function selectSchedule(schedule: AdminSchedule) {
  if (schedule.id === selectedId.value || !confirmDiscard()) return
  resetFeedback()
  acceptSchedule(schedule)
}

function resetFeedback() {
  actionError.value = ''
  notice.value = ''
  validationAttempted.value = false
  nameError.value = ''
  dayErrors.value = {}
}

function sortSchedules(items: AdminSchedule[]) {
  return [...items].sort((left, right) => left.name.localeCompare(right.name, 'ru'))
}

async function loadSchedules() {
  loading.value = true
  loadFailed.value = false
  try {
    const [scheduleResult, workingHoursResult] = await Promise.all([
      listAdminSchedules(),
      getAdminWorkingHours(),
    ])
    if (!scheduleResult.data || !workingHoursResult.data) {
      loadFailed.value = true
      return
    }
    schedules.value = sortSchedules(scheduleResult.data.items)
    workingHours.value = cloneDays(workingHoursResult.data.days)
    if (schedules.value[0]) acceptSchedule(schedules.value[0])
  } catch {
    loadFailed.value = true
  } finally {
    loading.value = false
  }
}

function timeMinutes(value: string) {
  const match = /^(\d{2}):(\d{2})$/.exec(value)
  if (!match) return null
  const hours = Number(match[1])
  const minutes = Number(match[2])
  if (hours > 23 || minutes > 59 || minutes % 5 !== 0) return null
  return hours * 60 + minutes
}

function formatMinutes(value: number) {
  return `${String(Math.floor(value / 60)).padStart(2, '0')}:${String(value % 60).padStart(2, '0')}`
}

function validateForm() {
  if (!form.value) return false
  const trimmedName = form.value.name.trim()
  nameError.value =
    [...trimmedName].length >= 1 && [...trimmedName].length <= 80
      ? ''
      : 'Введите название длиной от 1 до 80 символов.'
  const errors: Record<number, string> = {}
  for (const [dayIndex, day] of form.value.days.entries()) {
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
  return nameError.value === '' && Object.keys(errors).length === 0
}

function markEdited() {
  notice.value = ''
  actionError.value = ''
  if (validationAttempted.value) validateForm()
}

function addInterval(day: WeeklyDay) {
  day.intervals.push({ start_time: '09:00', end_time: '18:00' })
  markEdited()
}

function removeInterval(day: WeeklyDay, intervalIndex: number) {
  day.intervals.splice(intervalIndex, 1)
  markEdited()
}

function ineffectiveParts(dayIndex: number, intervalIndex: number) {
  const interval = form.value?.days[dayIndex]?.intervals[intervalIndex]
  if (!interval || !workingHours.value) return []
  const start = timeMinutes(interval.start_time)
  const end = timeMinutes(interval.end_time)
  if (start === null || end === null || end <= start) return []
  const boundaries =
    workingHours.value
      .find((day) => day.day_of_week === dayIndex + 1)
      ?.intervals.map((item) => ({
        start: timeMinutes(item.start_time) ?? 0,
        end: timeMinutes(item.end_time) ?? 0,
      }))
      .sort((left, right) => left.start - right.start) ?? []
  const ineffective: Array<{ start: number; end: number }> = []
  let cursor = start
  for (const boundary of boundaries) {
    if (boundary.end <= cursor) continue
    if (boundary.start >= end) break
    if (boundary.start > cursor)
      ineffective.push({ start: cursor, end: Math.min(boundary.start, end) })
    cursor = Math.max(cursor, boundary.end)
    if (cursor >= end) break
  }
  if (cursor < end) ineffective.push({ start: cursor, end })
  return ineffective.map((part) => `${formatMinutes(part.start)}–${formatMinutes(part.end)}`)
}

function applyServerErrors(error: ApiError | undefined) {
  if (!error?.field_errors) return
  for (const fieldError of error.field_errors) {
    if (fieldError.path === 'body.name') nameError.value = fieldError.message
    const match = /^body\.days\.(\d+)/.exec(fieldError.path)
    if (match) dayErrors.value[Number(match[1])] = fieldError.message
  }
}

async function saveSchedule() {
  if (saving.value || !form.value) return
  validationAttempted.value = true
  if (!validateForm()) return
  saving.value = true
  actionError.value = ''
  notice.value = ''
  const submitted = cloneForm(form.value)
  const current = selectedSchedule.value
  const wasCreating = creating.value
  try {
    const result = creating.value
      ? await createAdminSchedule({ body: submitted })
      : await updateAdminSchedule({
          path: { id: current!.id },
          headers: { 'If-Match': `"${current!.version}"` },
          body: submitted,
        })
    if (!result.data) {
      const error = result.error as ApiError | undefined
      applyServerErrors(error)
      actionError.value =
        error?.code === 'version_conflict'
          ? 'Расписание уже изменено. Обновите страницу и повторите изменения.'
          : error?.code === 'schedule_name_conflict'
            ? 'Расписание с таким названием уже существует.'
            : 'Не удалось сохранить расписание. Исправьте ошибки или повторите попытку.'
      return
    }
    const schedule = result.data
    const index = schedules.value.findIndex((item) => item.id === schedule.id)
    if (index >= 0) schedules.value.splice(index, 1, schedule)
    else schedules.value.push(schedule)
    schedules.value = sortSchedules(schedules.value)
    if (JSON.stringify(form.value) === JSON.stringify(submitted)) {
      acceptSchedule(schedule)
    } else {
      selectedId.value = schedule.id
      creating.value = false
      confirmedForm.value = { name: schedule.name, days: cloneDays(schedule.days) }
      validateForm()
    }
    notice.value = wasCreating ? 'Расписание создано' : 'Расписание сохранено'
  } catch {
    actionError.value = 'Не удалось связаться с сервером. Введённые изменения сохранены в форме.'
    if (creating.value) {
      try {
        const refreshed = await listAdminSchedules()
        if (refreshed.data) schedules.value = sortSchedules(refreshed.data.items)
      } catch {
        // The preserved form remains the authoritative UI state after an uncertain create.
      }
    }
  } finally {
    saving.value = false
  }
}

async function removeSchedule() {
  const schedule = selectedSchedule.value
  if (!schedule?.can_delete || deleting.value) return
  if (!window.confirm(`Удалить расписание «${schedule.name}»?`)) return
  deleting.value = true
  actionError.value = ''
  notice.value = ''
  try {
    const result = await deleteAdminSchedule({
      path: { id: schedule.id },
      headers: { 'If-Match': `"${schedule.version}"` },
    })
    const error = result.error as ApiError | undefined
    if (result.response?.status !== 204) {
      actionError.value =
        error?.code === 'version_conflict'
          ? 'Расписание уже изменено. Обновите страницу перед удалением.'
          : error?.code === 'schedule_in_use'
            ? 'Расписание уже назначено типу звонка и не может быть удалено.'
            : error?.code === 'last_schedule'
              ? 'Нельзя удалить последнее расписание.'
              : 'Не удалось удалить расписание.'
      return
    }
    const index = schedules.value.findIndex((item) => item.id === schedule.id)
    schedules.value.splice(index, 1)
    const next = schedules.value[Math.min(index, schedules.value.length - 1)]
    if (next) acceptSchedule(next)
    notice.value = 'Расписание удалено'
  } catch {
    actionError.value = 'Не удалось связаться с сервером. Расписание не удалено.'
  } finally {
    deleting.value = false
  }
}

function assignmentLabel(schedule: AdminSchedule) {
  const count = schedule.assigned_call_types.length
  const suffix =
    count % 10 === 1 && count % 100 !== 11
      ? 'тип'
      : count % 10 >= 2 && count % 10 <= 4 && (count % 100 < 10 || count % 100 >= 20)
        ? 'типа'
        : 'типов'
  if (count === 0) return 'Нет назначенных типов'
  return `${count} ${suffix}: ${schedule.assigned_call_types.map((item) => item.name).join(', ')}`
}

function deletionReason(schedule: AdminSchedule) {
  if (schedule.delete_blocked_reason === 'schedule_in_use')
    return 'Расписание назначено типам звонков'
  if (schedule.delete_blocked_reason === 'last_schedule')
    return 'Нельзя удалить последнее расписание'
  return ''
}

function confirmDiscard() {
  return !dirty.value || window.confirm('Отменить несохранённые изменения расписания?')
}

function handleBeforeUnload(event: BeforeUnloadEvent) {
  if (!dirty.value) return
  event.preventDefault()
}

onBeforeRouteLeave(confirmDiscard)
onMounted(() => {
  window.addEventListener('beforeunload', handleBeforeUnload)
  void loadSchedules()
})
onBeforeUnmount(() => window.removeEventListener('beforeunload', handleBeforeUnload))
</script>

<template>
  <div class="min-h-svh bg-slate-100 text-slate-950 lg:grid lg:grid-cols-[17rem_1fr]">
    <AdminSidebar
      active="types-and-schedules"
      note="Расписание задаёт доступность типов звонков внутри общего режима работы."
    />

    <main class="px-4 py-8 sm:px-8 lg:px-12 lg:py-10">
      <div class="mx-auto max-w-7xl">
        <header
          class="border-b border-slate-300 pb-7 sm:flex sm:items-end sm:justify-between sm:gap-6"
        >
          <div>
            <p class="text-xs font-bold tracking-[0.2em] text-blue-700 uppercase">Организатор</p>
            <h1 class="mt-3 text-3xl font-bold tracking-[-0.035em] sm:text-4xl">Расписания</h1>
            <p class="mt-3 max-w-2xl leading-7 text-slate-600">
              Создавайте именованные наборы недельной доступности и назначайте их типам звонков.
            </p>
          </div>
          <div class="mt-5 flex flex-wrap items-center gap-3 sm:mt-0">
            <RouterLink
              to="/admin/call-types"
              class="rounded-lg border border-slate-300 px-4 py-2.5 text-sm font-semibold"
            >
              Типы звонков
            </RouterLink>
            <button
              type="button"
              data-create-schedule
              class="rounded-lg bg-slate-950 px-4 py-2.5 text-sm font-semibold text-white"
              @click="startCreating"
            >
              + Новое расписание
            </button>
          </div>
        </header>

        <div v-if="loading" class="mt-8 rounded-xl bg-white p-8" aria-busy="true">
          Загружаем расписания…
        </div>
        <div
          v-else-if="loadFailed"
          class="mt-8 rounded-xl border border-red-200 bg-red-50 p-6"
          role="alert"
        >
          <h2 class="text-lg font-bold">Не удалось загрузить расписания</h2>
          <button
            type="button"
            class="mt-5 rounded-lg bg-slate-950 px-4 py-2 text-sm font-semibold text-white"
            @click="loadSchedules"
          >
            Повторить
          </button>
        </div>

        <div v-else class="mt-8 grid gap-6 lg:grid-cols-[minmax(16rem,22rem)_minmax(0,1fr)]">
          <section aria-label="Список расписаний" class="space-y-3">
            <button
              v-for="schedule in schedules"
              :key="schedule.id"
              type="button"
              :data-select-schedule="schedule.id"
              class="w-full rounded-xl border p-4 text-left shadow-sm"
              :class="
                selectedId === schedule.id
                  ? 'border-blue-500 bg-blue-50'
                  : 'border-slate-200 bg-white hover:border-slate-300'
              "
              @click="selectSchedule(schedule)"
            >
              <span class="block font-bold">{{ schedule.name }}</span>
              <span class="mt-1 block text-sm text-slate-600">{{ assignmentLabel(schedule) }}</span>
              <span
                v-if="!schedule.can_delete"
                class="mt-2 block text-xs font-medium text-amber-700"
              >
                {{ deletionReason(schedule) }}
              </span>
            </button>
          </section>

          <form v-if="form" class="min-w-0" @submit.prevent="saveSchedule">
            <section class="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
              <div class="border-b border-slate-200 px-5 py-5">
                <div class="flex flex-wrap items-start justify-between gap-3">
                  <div class="min-w-[14rem] flex-1">
                    <label for="schedule-name" class="text-sm font-semibold">Название</label>
                    <input
                      id="schedule-name"
                      v-model="form.name"
                      name="schedule-name"
                      maxlength="80"
                      class="mt-2 w-full rounded-lg border border-slate-300 px-3 py-2.5"
                      @input="markEdited"
                    />
                    <p v-if="nameError" class="mt-2 text-sm font-medium text-red-700">
                      {{ nameError }}
                    </p>
                  </div>
                  <p v-if="selectedSchedule" class="pt-7 font-mono text-xs text-slate-500">
                    Подтверждённая версия: {{ selectedSchedule.version }}
                  </p>
                </div>
                <div
                  v-if="selectedSchedule?.assigned_call_types.length"
                  class="mt-5 rounded-lg bg-slate-50 p-4"
                >
                  <p class="text-xs font-bold tracking-wide text-slate-500 uppercase">
                    Назначенные типы
                  </p>
                  <div class="mt-2 flex flex-wrap gap-2">
                    <RouterLink
                      v-for="callType in selectedSchedule.assigned_call_types"
                      :key="callType.id"
                      :to="{
                        name: 'admin-call-types',
                        query: { call_type_id: callType.id },
                      }"
                      :data-assigned-call-type="callType.id"
                      class="rounded-full border border-slate-200 bg-white px-3 py-1 text-sm text-slate-700"
                    >
                      {{ callType.name }} ·
                      {{ callType.status === 'active' ? 'активен' : 'отключён' }}
                    </RouterLink>
                  </div>
                </div>
              </div>

              <div class="divide-y divide-slate-200">
                <article
                  v-for="(day, dayIndex) in form.days"
                  :key="day.day_of_week"
                  data-schedule-day
                  class="grid gap-4 px-5 py-5 md:grid-cols-[9rem_1fr_auto] md:items-start"
                >
                  <div>
                    <h3 class="font-semibold">{{ dayNames[dayIndex] }}</h3>
                    <p v-if="day.intervals.length === 0" class="mt-1 text-sm text-slate-500">
                      Выходной
                    </p>
                  </div>
                  <div class="space-y-3">
                    <div v-for="(interval, intervalIndex) in day.intervals" :key="intervalIndex">
                      <div class="flex flex-wrap items-center gap-2">
                        <label class="sr-only" :for="`schedule-${dayIndex}-start-${intervalIndex}`"
                          >Начало</label
                        >
                        <input
                          :id="`schedule-${dayIndex}-start-${intervalIndex}`"
                          v-model="interval.start_time"
                          :name="`days.${dayIndex}.intervals.${intervalIndex}.start_time`"
                          type="time"
                          step="300"
                          class="rounded-lg border border-slate-300 px-3 py-2 font-mono text-sm"
                          @input="markEdited"
                        />
                        <span class="text-slate-400">—</span>
                        <label class="sr-only" :for="`schedule-${dayIndex}-end-${intervalIndex}`"
                          >Конец</label
                        >
                        <input
                          :id="`schedule-${dayIndex}-end-${intervalIndex}`"
                          v-model="interval.end_time"
                          :name="`days.${dayIndex}.intervals.${intervalIndex}.end_time`"
                          type="time"
                          step="300"
                          class="rounded-lg border border-slate-300 px-3 py-2 font-mono text-sm"
                          @input="markEdited"
                        />
                        <button
                          type="button"
                          class="rounded-lg px-3 py-2 text-sm font-semibold text-red-700 hover:bg-red-50"
                          @click="removeInterval(day, intervalIndex)"
                        >
                          Удалить
                        </button>
                      </div>
                      <p
                        v-if="ineffectiveParts(dayIndex, intervalIndex).length"
                        :data-ineffective="`${dayIndex}-${intervalIndex}`"
                        class="mt-2 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-800"
                      >
                        Неэффективно при текущем режиме работы:
                        {{ ineffectiveParts(dayIndex, intervalIndex).join(', ') }}. Эти части
                        сохранятся и снова создадут слоты после расширения режима.
                      </p>
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
                    :data-add-interval="dayIndex"
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
              <p v-if="actionError" class="mb-3 text-sm font-medium text-red-700" role="alert">
                {{ actionError }}
              </p>
              <p v-if="notice" class="mb-3 text-sm font-medium text-emerald-700" role="status">
                {{ notice }}
              </p>
              <div class="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <button
                    v-if="selectedSchedule"
                    type="button"
                    data-delete-schedule
                    class="rounded-lg border border-red-200 px-4 py-2.5 text-sm font-semibold text-red-700 disabled:cursor-not-allowed disabled:opacity-45"
                    :disabled="deleting || !selectedSchedule.can_delete"
                    :title="deletionReason(selectedSchedule)"
                    @click="removeSchedule"
                  >
                    {{ deleting ? 'Удаляем…' : 'Удалить расписание' }}
                  </button>
                </div>
                <div class="flex items-center gap-4">
                  <p class="text-sm text-slate-500">
                    {{ dirty ? 'Есть несохранённые изменения' : 'Все изменения сохранены' }}
                  </p>
                  <button
                    type="submit"
                    class="rounded-lg bg-blue-600 px-5 py-2.5 text-sm font-semibold text-white disabled:cursor-not-allowed disabled:opacity-45"
                    :disabled="saving || (!creating && !dirty)"
                  >
                    {{
                      saving
                        ? 'Сохраняем…'
                        : creating
                          ? 'Создать расписание'
                          : 'Сохранить расписание'
                    }}
                  </button>
                </div>
              </div>
            </div>
          </form>
        </div>
      </div>
    </main>
  </div>
</template>
