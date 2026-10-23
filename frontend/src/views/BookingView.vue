<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import {
  createCall,
  getPublicCallType,
  getPublicCallTypeAvailability,
  listPublicCallTypes,
  type AvailabilitySlot,
  type AvailabilityResponseBody,
  type CreatedCall,
  type PublicCallType,
  type PublicCallTypeSummary,
} from '@/api/generated'

type PageState =
  'loading' | 'ready' | 'availability-error' | 'page-error' | 'disabled' | 'not-found'

const route = useRoute()
const typeSlug = computed(() => String(route.params.typeSlug))
const state = ref<PageState>('loading')
const callType = ref<PublicCallType | null>(null)
const availability = ref<AvailabilityResponseBody | null>(null)
const activeCallTypes = ref<PublicCallTypeSummary[]>([])
const displayedCallType = computed(() => availability.value?.call_type ?? callType.value)
const selectedSlot = ref<AvailabilitySlot | null>(null)
const formHeading = ref<HTMLElement | null>(null)
const submitting = ref(false)
const submitError = ref('')
const fieldErrors = ref<Record<string, string>>({})
const draft = ref(loadDraft())
const confirmation = ref<CreatedCall | null>(loadConfirmation())
const idempotency = ref<{ fingerprint: string; key: string } | null>(loadIdempotency())
const currentConfirmation = computed(() =>
  confirmation.value?.call_type_slug === typeSlug.value ? confirmation.value : null,
)

const hasNoSlots = computed(
  () =>
    availability.value !== null && availability.value.days.every((day) => day.slots.length === 0),
)

watch(draft, (value) => sessionStorage.setItem('booking-draft', JSON.stringify(value)), {
  deep: true,
})

function emptyDraft() {
  return { visitorName: '', visitorEmail: '', visitorNotes: '' }
}

function loadDraft() {
  try {
    const stored = JSON.parse(sessionStorage.getItem('booking-draft') ?? 'null')
    if (
      stored &&
      typeof stored.visitorName === 'string' &&
      typeof stored.visitorEmail === 'string' &&
      typeof stored.visitorNotes === 'string'
    ) {
      return stored
    }
  } catch {
    // Ignore a damaged per-tab draft.
  }
  return emptyDraft()
}

function loadConfirmation() {
  try {
    const stored = JSON.parse(sessionStorage.getItem('booking-confirmation') ?? 'null')
    return stored && typeof stored.call_type_slug === 'string' ? (stored as CreatedCall) : null
  } catch {
    return null
  }
}

function loadIdempotency() {
  try {
    const stored = JSON.parse(sessionStorage.getItem('booking-idempotency') ?? 'null')
    if (stored && typeof stored.fingerprint === 'string' && typeof stored.key === 'string') {
      return stored
    }
  } catch {
    // Ignore a damaged per-tab booking attempt.
  }
  return null
}

function rememberIdempotency(value: { fingerprint: string; key: string } | null) {
  idempotency.value = value
  if (value) {
    sessionStorage.setItem('booking-idempotency', JSON.stringify(value))
  } else {
    sessionStorage.removeItem('booking-idempotency')
  }
}

function errorCode(result: { error?: unknown }) {
  if (typeof result.error !== 'object' || result.error === null || !('code' in result.error)) {
    return null
  }
  return String(result.error.code)
}

async function loadAvailability(slug = typeSlug.value) {
  state.value = 'loading'
  availability.value = null
  try {
    const result = await getPublicCallTypeAvailability({ path: { slug } })
    if (slug !== typeSlug.value) return
    if (result.data) {
      availability.value = result.data
      state.value = 'ready'
      return
    }
    const code = errorCode(result)
    if (code === 'call_type_unavailable') {
      state.value = 'disabled'
      return
    }
    if (code === 'call_type_not_found') {
      state.value = 'not-found'
      return
    }
    state.value = 'availability-error'
  } catch {
    state.value = 'availability-error'
  }
}

async function loadPage() {
  const slug = typeSlug.value
  selectedSlot.value = null
  state.value = 'loading'
  callType.value = null
  availability.value = null
  try {
    const [callTypeResult, catalogResult] = await Promise.all([
      getPublicCallType({ path: { slug } }),
      listPublicCallTypes(),
    ])
    if (slug !== typeSlug.value) return
    activeCallTypes.value = catalogResult.data?.items ?? []

    if (!callTypeResult.data) {
      state.value = errorCode(callTypeResult) === 'call_type_not_found' ? 'not-found' : 'page-error'
      return
    }
    callType.value = callTypeResult.data
    if (callTypeResult.data.status === 'disabled') {
      state.value = 'disabled'
      return
    }
    await loadAvailability(slug)
  } catch {
    state.value = 'page-error'
  }
}

async function selectSlot(slot: AvailabilitySlot) {
  selectedSlot.value = slot
  submitError.value = ''
  fieldErrors.value = {}
  if (window.innerWidth >= 640) return
  await nextTick()
  formHeading.value?.focus()
}

function normalizeEmail(value: string) {
  const trimmed = value.trim()
  const separator = trimmed.lastIndexOf('@')
  return separator < 0
    ? trimmed
    : `${trimmed.slice(0, separator + 1)}${trimmed.slice(separator + 1).toLowerCase()}`
}

function normalizedVisitorData() {
  const notes = draft.value.visitorNotes.replace(/\r\n?/g, '\n').trim().normalize('NFC')
  return {
    visitor_name: draft.value.visitorName.trim().normalize('NFC'),
    visitor_email: normalizeEmail(draft.value.visitorEmail),
    visitor_notes: notes === '' ? null : notes,
  }
}

function validateVisitorData(data: ReturnType<typeof normalizedVisitorData>) {
  const errors: Record<string, string> = {}
  const nameLength = Array.from(data.visitor_name).length
  if (nameLength < 1 || nameLength > 100 || /\p{Cc}/u.test(data.visitor_name)) {
    errors.visitor_name = 'Укажите однострочное имя длиной до 100 символов'
  }
  const emailIsASCII = [...data.visitor_email].every((character) => character.charCodeAt(0) < 128)
  if (
    data.visitor_email.length > 254 ||
    !emailIsASCII ||
    !/^[^\s@]+@[^\s@]+$/.test(data.visitor_email)
  ) {
    errors.visitor_email = 'Укажите корректный email латинскими символами'
  }
  if (data.visitor_notes !== null && Array.from(data.visitor_notes).length > 1000) {
    errors.visitor_notes = 'Комментарий должен быть не длиннее 1000 символов'
  }
  fieldErrors.value = errors
  return Object.keys(errors).length === 0
}

async function submitBooking() {
  if (submitting.value || !selectedSlot.value || !availability.value) return
  const visitor = normalizedVisitorData()
  if (!validateVisitorData(visitor)) return
  const body = {
    call_type_slug: availability.value.call_type.slug,
    call_type_version: availability.value.call_type.call_type_version,
    starts_at: selectedSlot.value.starts_at,
    ...visitor,
  }
  const fingerprint = JSON.stringify(body)
  let attempt = idempotency.value
  if (attempt?.fingerprint !== fingerprint) {
    attempt = { fingerprint, key: crypto.randomUUID() }
    rememberIdempotency(attempt)
  }
  submitting.value = true
  submitError.value = ''
  try {
    const result = await createCall({
      body,
      headers: { 'Idempotency-Key': attempt.key },
    })
    if (result.data) {
      confirmation.value = result.data
      sessionStorage.setItem('booking-confirmation', JSON.stringify(result.data))
      draft.value = emptyDraft()
      await nextTick()
      sessionStorage.removeItem('booking-draft')
      rememberIdempotency(null)
      return
    }
    if (result.response && result.response.status >= 400 && result.response.status < 500) {
      rememberIdempotency(null)
    }
    const code = errorCode(result)
    if (code === 'slot_unavailable' || code === 'call_type_changed') {
      submitError.value = 'Условия изменились. Выберите свободное время ещё раз.'
      selectedSlot.value = null
      await loadAvailability()
      return
    }
    if (code === 'call_type_unavailable') {
      state.value = 'disabled'
      return
    }
    if (
      code === 'validation_error' &&
      typeof result.error === 'object' &&
      result.error !== null &&
      'field_errors' in result.error &&
      Array.isArray(result.error.field_errors)
    ) {
      fieldErrors.value = Object.fromEntries(
        result.error.field_errors.map((error) => [
          String(error.path).replace('body.', ''),
          String(error.message),
        ]),
      )
      return
    }
    submitError.value = 'Не удалось завершить бронирование. Попробуйте ещё раз.'
  } catch {
    submitError.value = 'Не удалось связаться с сервером. Повторите отправку.'
  } finally {
    submitting.value = false
  }
}

async function bookAnother() {
  confirmation.value = null
  sessionStorage.removeItem('booking-confirmation')
  selectedSlot.value = null
  await loadAvailability()
}

function formatDate(date: string) {
  return new Intl.DateTimeFormat('ru-RU', {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
    timeZone: 'Europe/Moscow',
  }).format(new Date(`${date}T12:00:00+03:00`))
}

function formatTime(value: string) {
  return value.slice(11, 16)
}

function formatConfirmationDate(value: string) {
  return new Intl.DateTimeFormat('ru-RU', {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
    year: 'numeric',
    timeZone: 'Europe/Moscow',
  }).format(new Date(value))
}

watch(typeSlug, loadPage, { immediate: true })
</script>

<template>
  <main class="min-h-svh bg-slate-100 text-slate-950">
    <header class="border-b border-slate-200 bg-white px-5 py-5 sm:px-10">
      <div class="mx-auto flex max-w-7xl items-center justify-between gap-4">
        <RouterLink to="/" class="font-bold tracking-tight">Календарь звонков</RouterLink>
        <RouterLink to="/booking" class="text-sm font-semibold text-blue-700 hover:text-blue-900">
          Сменить тип звонка
        </RouterLink>
      </div>
    </header>

    <div v-if="currentConfirmation" data-confirmation class="mx-auto max-w-3xl px-5 py-16 sm:py-24">
      <p class="text-sm font-bold text-emerald-700">Звонок забронирован</p>
      <h1 class="mt-3 text-4xl font-bold tracking-tight sm:text-5xl">
        {{ currentConfirmation.call_type_name }}
      </h1>
      <dl class="mt-10 divide-y divide-slate-200 border-y border-slate-300 bg-white px-5 sm:px-8">
        <div class="grid gap-1 py-5 sm:grid-cols-[11rem_1fr]">
          <dt class="text-sm font-semibold text-slate-500">Дата и время</dt>
          <dd class="font-semibold capitalize">
            {{ formatConfirmationDate(currentConfirmation.starts_at) }},
            {{ formatTime(currentConfirmation.starts_at) }}–{{
              formatTime(currentConfirmation.ends_at)
            }}
          </dd>
        </div>
        <div class="grid gap-1 py-5 sm:grid-cols-[11rem_1fr]">
          <dt class="text-sm font-semibold text-slate-500">Длительность</dt>
          <dd>{{ currentConfirmation.duration_minutes }} минут</dd>
        </div>
        <div class="grid gap-1 py-5 sm:grid-cols-[11rem_1fr]">
          <dt class="text-sm font-semibold text-slate-500">Имя</dt>
          <dd>{{ currentConfirmation.visitor_name }}</dd>
        </div>
        <div class="grid gap-1 py-5 sm:grid-cols-[11rem_1fr]">
          <dt class="text-sm font-semibold text-slate-500">Email</dt>
          <dd>{{ currentConfirmation.visitor_email }}</dd>
        </div>
        <div
          v-if="currentConfirmation.visitor_notes"
          class="grid gap-1 py-5 sm:grid-cols-[11rem_1fr]"
        >
          <dt class="text-sm font-semibold text-slate-500">Дополнительная информация</dt>
          <dd class="whitespace-pre-wrap">{{ currentConfirmation.visitor_notes }}</dd>
        </div>
      </dl>
      <p class="mt-6 text-slate-600">Письмо с подтверждением не отправляется.</p>
      <div class="mt-8 flex flex-wrap gap-3">
        <RouterLink
          to="/"
          class="rounded-lg bg-slate-950 px-5 py-3 font-semibold text-white hover:bg-blue-700"
        >
          На главную
        </RouterLink>
        <button
          type="button"
          class="rounded-lg border border-slate-300 bg-white px-5 py-3 font-semibold hover:border-blue-500 hover:text-blue-700"
          @click="bookAnother"
        >
          Забронировать ещё
        </button>
      </div>
    </div>

    <div
      v-else-if="state === 'disabled' || state === 'not-found'"
      :data-state="state"
      class="mx-auto grid max-w-5xl gap-12 px-5 py-16 lg:grid-cols-[1fr_22rem] lg:px-10 lg:py-24"
    >
      <section>
        <p class="text-sm font-bold text-blue-700">
          {{ state === 'disabled' ? 'Тип звонка временно недоступен' : 'Тип звонка не найден' }}
        </p>
        <h1 class="mt-3 text-4xl font-bold tracking-tight sm:text-5xl">
          {{ state === 'disabled' ? callType?.name : 'Эта ссылка больше не работает' }}
        </h1>
        <p v-if="state === 'disabled'" class="mt-4 font-mono text-sm text-slate-500">
          {{ callType?.duration_minutes }} минут · Europe/Moscow
        </p>
        <p class="mt-6 max-w-xl text-lg leading-8 text-slate-600">
          <template v-if="state === 'disabled'">
            Сейчас этот тип нельзя забронировать. Выберите другой доступный формат звонка.
          </template>
          <template v-else> Возможно, тип звонка был удалён или адрес указан неверно. </template>
        </p>
      </section>

      <aside>
        <h2 class="text-sm font-bold tracking-wider text-slate-500 uppercase">Активные типы</h2>
        <div v-if="activeCallTypes.length" class="mt-4 divide-y divide-slate-200 border-y">
          <RouterLink
            v-for="item in activeCallTypes"
            :key="item.slug"
            :to="{ name: 'booking-type', params: { typeSlug: item.slug } }"
            class="flex items-center justify-between gap-4 py-4 font-semibold hover:text-blue-700"
          >
            <span>{{ item.name }}</span>
            <span class="font-mono text-xs text-slate-500">{{ item.duration_minutes }} мин</span>
          </RouterLink>
        </div>
        <RouterLink v-else to="/booking" class="mt-5 inline-flex font-semibold text-blue-700">
          К списку типов звонков
        </RouterLink>
      </aside>
    </div>

    <div
      v-else-if="state === 'page-error'"
      class="mx-auto max-w-3xl px-5 py-20 text-center"
      role="alert"
    >
      <h1 class="text-3xl font-bold">Не удалось загрузить тип звонка</h1>
      <p class="mt-3 text-slate-600">Проверьте соединение и попробуйте ещё раз.</p>
      <button
        type="button"
        class="mt-7 rounded-lg bg-slate-950 px-5 py-3 font-semibold text-white hover:bg-blue-700"
        @click="loadPage"
      >
        Повторить
      </button>
    </div>

    <div v-else class="mx-auto grid max-w-7xl gap-8 px-5 py-10 lg:grid-cols-[18rem_1fr] lg:px-10">
      <aside class="lg:sticky lg:top-8 lg:self-start">
        <p class="text-xs font-bold tracking-[0.18em] text-blue-700 uppercase">Планировщик</p>
        <h1 class="mt-3 text-3xl font-bold tracking-tight">
          {{ displayedCallType?.name ?? 'Выберите свободное время' }}
        </h1>
        <p v-if="displayedCallType?.description" class="mt-4 leading-7 text-slate-600">
          {{ displayedCallType.description }}
        </p>
        <p v-if="displayedCallType" class="mt-5 font-mono text-sm text-slate-500">
          {{ displayedCallType.duration_minutes }} минут · Europe/Moscow
        </p>
      </aside>

      <section aria-labelledby="availability-heading">
        <div class="flex flex-wrap items-end justify-between gap-3">
          <div>
            <p class="text-sm font-semibold text-slate-500">Сегодня и следующие 13 дней</p>
            <h2 id="availability-heading" class="mt-1 text-2xl font-bold">Свободное время</h2>
          </div>
          <p class="text-sm text-slate-500">Время указано по Москве</p>
        </div>

        <div v-if="state === 'loading'" class="mt-6 grid gap-4 sm:grid-cols-2" aria-busy="true">
          <p class="sr-only">Загружаем свободное время</p>
          <div v-for="index in 14" :key="index" class="h-36 animate-pulse rounded-xl bg-white" />
        </div>

        <div
          v-else-if="state === 'availability-error'"
          class="mt-6 border-y border-red-200 bg-red-50 px-5 py-8"
          role="alert"
        >
          <h3 class="text-xl font-bold">Не удалось загрузить свободное время</h3>
          <p class="mt-2 text-slate-600">
            Тип звонка загружен. Повторите только запрос расписания.
          </p>
          <button
            type="button"
            data-retry-availability
            class="mt-5 rounded-lg bg-slate-950 px-4 py-2 font-semibold text-white hover:bg-blue-700"
            @click="loadAvailability()"
          >
            Повторить
          </button>
        </div>

        <template v-else-if="availability">
          <div v-if="hasNoSlots" class="mt-6 border-y border-amber-200 bg-amber-50 px-5 py-5">
            <p class="font-semibold">В ближайшие 14 дней свободного времени нет</p>
            <RouterLink to="/booking" class="mt-2 inline-flex text-sm font-semibold text-blue-700">
              Сменить тип звонка
            </RouterLink>
          </div>

          <p
            v-if="submitError && !selectedSlot"
            class="mt-6 border-y border-red-200 bg-red-50 px-5 py-4 text-sm font-semibold text-red-700"
            role="alert"
          >
            {{ submitError }}
          </p>

          <div class="mt-6 grid gap-4 sm:grid-cols-2">
            <article
              v-for="day in availability.days"
              :key="day.date"
              data-availability-day
              class="rounded-xl border border-slate-200 bg-white p-5"
            >
              <h3 class="font-bold capitalize">{{ formatDate(day.date) }}</h3>
              <p data-availability-date class="mt-1 font-mono text-xs text-slate-400">
                {{ day.date }}
              </p>
              <div v-if="day.slots.length" class="mt-4 flex flex-wrap gap-2">
                <button
                  v-for="slot in day.slots"
                  :key="slot.starts_at"
                  type="button"
                  data-slot
                  :aria-pressed="selectedSlot?.starts_at === slot.starts_at"
                  class="rounded-lg px-3 py-2 font-mono text-sm font-semibold transition"
                  :class="
                    selectedSlot?.starts_at === slot.starts_at
                      ? 'bg-slate-950 text-white'
                      : 'bg-blue-50 text-blue-800 hover:bg-blue-100'
                  "
                  @click="selectSlot(slot)"
                >
                  {{ formatTime(slot.starts_at) }}–{{ formatTime(slot.ends_at) }}
                </button>
              </div>
              <p v-else class="mt-4 text-sm text-slate-500">Нет свободного времени</p>
            </article>
          </div>

          <form
            v-if="selectedSlot"
            class="mt-10 border-t border-slate-300 pt-8"
            @submit.prevent="submitBooking"
          >
            <p class="text-sm font-semibold text-blue-700">
              {{ formatDate(selectedSlot.starts_at.slice(0, 10)) }} ·
              {{ formatTime(selectedSlot.starts_at) }}–{{ formatTime(selectedSlot.ends_at) }}
            </p>
            <h2 ref="formHeading" tabindex="-1" class="mt-2 text-2xl font-bold outline-none">
              Ваши данные
            </h2>
            <div class="mt-6 grid gap-5 sm:grid-cols-2">
              <label class="grid gap-2 font-semibold">
                Имя
                <input
                  v-model="draft.visitorName"
                  name="visitor_name"
                  autocomplete="name"
                  class="rounded-lg border border-slate-300 bg-white px-4 py-3 font-normal outline-none focus:border-blue-600 focus:ring-2 focus:ring-blue-100"
                  :aria-invalid="Boolean(fieldErrors.visitor_name)"
                  required
                />
                <span v-if="fieldErrors.visitor_name" class="text-sm font-normal text-red-700">
                  {{ fieldErrors.visitor_name }}
                </span>
              </label>
              <label class="grid gap-2 font-semibold">
                Email
                <input
                  v-model="draft.visitorEmail"
                  name="visitor_email"
                  type="email"
                  autocomplete="email"
                  class="rounded-lg border border-slate-300 bg-white px-4 py-3 font-normal outline-none focus:border-blue-600 focus:ring-2 focus:ring-blue-100"
                  :aria-invalid="Boolean(fieldErrors.visitor_email)"
                  required
                />
                <span v-if="fieldErrors.visitor_email" class="text-sm font-normal text-red-700">
                  {{ fieldErrors.visitor_email }}
                </span>
              </label>
            </div>
            <label class="mt-5 grid gap-2 font-semibold">
              Дополнительная информация
              <textarea
                v-model="draft.visitorNotes"
                name="visitor_notes"
                rows="5"
                class="resize-y rounded-lg border border-slate-300 bg-white px-4 py-3 font-normal outline-none focus:border-blue-600 focus:ring-2 focus:ring-blue-100"
                :aria-invalid="Boolean(fieldErrors.visitor_notes)"
              />
              <span v-if="fieldErrors.visitor_notes" class="text-sm font-normal text-red-700">
                {{ fieldErrors.visitor_notes }}
              </span>
            </label>
            <p v-if="submitError" class="mt-5 text-sm font-semibold text-red-700" role="alert">
              {{ submitError }}
            </p>
            <button
              type="submit"
              class="mt-6 rounded-lg bg-slate-950 px-6 py-3 font-semibold text-white hover:bg-blue-700 disabled:cursor-wait disabled:opacity-60"
              :disabled="submitting"
            >
              {{ submitting ? 'Бронируем…' : 'Забронировать звонок' }}
            </button>
          </form>
        </template>
      </section>
    </div>
  </main>
</template>
