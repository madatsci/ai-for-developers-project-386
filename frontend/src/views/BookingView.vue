<script setup lang="ts">
// Throwaway UI prototype. Variant B is the selected direction; A and C remain as comparison evidence.
import { computed, reactive, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import PrototypeSwitcher from './booking-prototype/PrototypeSwitcher.vue'
import VariantA from './booking-prototype/VariantA.vue'
import VariantB from './booking-prototype/VariantB.vue'
import VariantC from './booking-prototype/VariantC.vue'
import {
  buildCalendarDays,
  callTypes,
  disabledCallTypes,
  type BookingPrototypeState,
  type PrototypeVariant,
} from './booking-prototype/model'

interface StoredBookingState {
  selectedTypeSlug: string | null
  selectedDateKey: string | null
  selectedTime: string | null
  visitorName: string
  visitorEmail: string
  visitorNotes: string
}

const draftStorageKey = 'booking-prototype-draft'
const confirmationStorageKey = 'booking-prototype-confirmation'
const route = useRoute()
const router = useRouter()
const dates = buildCalendarDays()
const showPrototypeTools = import.meta.env.DEV
const initialTypeSlug = typeof route.params.typeSlug === 'string' ? route.params.typeSlug : null

function readSessionState(key: string): StoredBookingState | null {
  if (typeof window === 'undefined') return null

  const value = window.sessionStorage.getItem(key)
  if (!value) return null

  try {
    return JSON.parse(value) as StoredBookingState
  } catch {
    window.sessionStorage.removeItem(key)
    return null
  }
}

function writeSessionState(key: string, value: StoredBookingState) {
  window.sessionStorage.setItem(key, JSON.stringify(value))
}

const storedDraft = readSessionState(draftStorageKey)
const storedConfirmation = readSessionState(confirmationStorageKey)
const restoredConfirmation =
  storedConfirmation?.selectedTypeSlug === initialTypeSlug ? storedConfirmation : null
const restoredSelection = storedDraft?.selectedTypeSlug === initialTypeSlug ? storedDraft : null
const restoredContact = restoredConfirmation ?? storedDraft

const state = reactive<BookingPrototypeState>({
  stage: restoredConfirmation
    ? 'confirmation'
    : initialTypeSlug
      ? restoredSelection?.selectedTime
        ? 'details'
        : 'slot'
      : 'type',
  selectedDateKey:
    restoredConfirmation?.selectedDateKey ?? restoredSelection?.selectedDateKey ?? null,
  selectedTime: restoredConfirmation?.selectedTime ?? restoredSelection?.selectedTime ?? null,
  visitorName: restoredContact?.visitorName ?? '',
  visitorEmail: restoredContact?.visitorEmail ?? '',
  visitorNotes: restoredContact?.visitorNotes ?? '',
  errors: {},
})

const variant = computed<PrototypeVariant>(() => {
  const requested = route.query.variant
  return requested === 'A' || requested === 'C' ? requested : 'B'
})

const variantComponent = computed(() => {
  return { A: VariantA, B: VariantB, C: VariantC }[variant.value]
})

const typeSlug = computed(() =>
  typeof route.params.typeSlug === 'string' ? route.params.typeSlug : null,
)
const activeCallTypes = computed(() =>
  [...callTypes].sort(
    (left, right) =>
      left.durationMinutes - right.durationMinutes || left.name.localeCompare(right.name, 'ru'),
  ),
)
const selectedType = computed(() =>
  activeCallTypes.value.find((type) => type.slug === typeSlug.value),
)
const disabledType = computed(() => disabledCallTypes.find((type) => type.slug === typeSlug.value))
const selectedDay = computed(() => dates.find((day) => day.key === state.selectedDateKey))
const hasUnknownType = computed(() =>
  Boolean(typeSlug.value && !selectedType.value && !disabledType.value),
)

const prototypeState = computed<Record<string, unknown>>(() => ({
  route: route.path,
  variant: variant.value,
  stage: state.stage,
  selectedTypeSlug: selectedType.value?.slug ?? null,
  selectedDateKey: state.selectedDateKey,
  selectedTime: state.selectedTime,
  visitorName: state.visitorName,
  visitorEmail: state.visitorEmail,
  visitorNotes: state.visitorNotes,
  confirmationSurvivesRefresh: state.stage === 'confirmation',
  draftClearedAfterSuccess: state.stage === 'confirmation',
}))

function clearSelection(stage: BookingPrototypeState['stage']) {
  state.stage = stage
  state.selectedDateKey = null
  state.selectedTime = null
  state.errors = {}
}

function selectType(slug: string) {
  window.sessionStorage.removeItem(confirmationStorageKey)
  clearSelection('slot')
  void router.push({ name: 'booking-type', params: { typeSlug: slug }, query: route.query })
}

function selectDate(key: string) {
  state.selectedDateKey = key
  state.selectedTime = null
  state.stage = 'slot'
  state.errors = {}
}

function selectTime(time: string) {
  if (!state.selectedDateKey) return
  state.selectedTime = time
  state.stage = 'details'
  state.errors = {}
}

function backToTypes() {
  clearSelection('type')
  void router.push({ name: 'booking', query: route.query })
}

function backToSlot() {
  state.stage = 'slot'
  state.errors = {}
}

function updateVisitorName(value: string) {
  state.visitorName = value
  delete state.errors.visitorName
}

function updateVisitorEmail(value: string) {
  state.visitorEmail = value
  delete state.errors.visitorEmail
}

function updateVisitorNotes(value: string) {
  state.visitorNotes = value
  delete state.errors.visitorNotes
}

function restart() {
  window.sessionStorage.removeItem(confirmationStorageKey)
  state.visitorName = ''
  state.visitorEmail = ''
  state.visitorNotes = ''
  backToTypes()
}

function submitBooking() {
  const visitorName = state.visitorName.trim()
  const visitorEmail = state.visitorEmail.trim()
  const visitorNotes = state.visitorNotes.trim()
  const atIndex = visitorEmail.lastIndexOf('@')
  const hasAsciiOnly = /^[\x00-\x7F]*$/.test(visitorEmail)

  state.errors = {}

  if (!state.selectedDateKey || !state.selectedTime) return
  if (!visitorName) state.errors.visitorName = 'Укажите имя'
  if (
    atIndex <= 0 ||
    atIndex === visitorEmail.length - 1 ||
    !visitorEmail.slice(atIndex + 1).includes('.') ||
    visitorEmail.length > 254 ||
    !hasAsciiOnly ||
    /\s/.test(visitorEmail)
  ) {
    state.errors.visitorEmail = 'Укажите корректный email'
  }
  if (visitorNotes.length > 1000) {
    state.errors.visitorNotes = 'Не больше 1000 символов'
  }

  if (Object.keys(state.errors).length > 0) return

  const localPart = visitorEmail.slice(0, atIndex)
  const domain = visitorEmail.slice(atIndex + 1).toLowerCase()
  state.visitorName = visitorName
  state.visitorEmail = `${localPart}@${domain}`
  state.visitorNotes = visitorNotes
  writeSessionState(confirmationStorageKey, {
    selectedTypeSlug: selectedType.value?.slug ?? null,
    selectedDateKey: state.selectedDateKey,
    selectedTime: state.selectedTime,
    visitorName: state.visitorName,
    visitorEmail: state.visitorEmail,
    visitorNotes: state.visitorNotes,
  })
  window.sessionStorage.removeItem(draftStorageKey)
  state.stage = 'confirmation'
}

function changeVariant(nextVariant: PrototypeVariant) {
  void router.replace({
    path: route.path,
    query: { ...route.query, variant: nextVariant },
  })
}

watch(typeSlug, (nextSlug, previousSlug) => {
  if (!nextSlug) {
    state.stage = 'type'
  } else if (nextSlug !== previousSlug) {
    window.sessionStorage.removeItem(confirmationStorageKey)
    clearSelection('slot')
  }
})

watch(
  [state, typeSlug],
  () => {
    if (state.stage === 'confirmation') {
      window.sessionStorage.removeItem(draftStorageKey)
      return
    }

    writeSessionState(draftStorageKey, {
      selectedTypeSlug: typeSlug.value,
      selectedDateKey: state.selectedDateKey,
      selectedTime: state.selectedTime,
      visitorName: state.visitorName,
      visitorEmail: state.visitorEmail,
      visitorNotes: state.visitorNotes,
    })
  },
  { deep: true },
)
</script>

<template>
  <div
    v-if="disabledType"
    class="grid min-h-svh place-items-center bg-slate-950 px-6 text-center text-white"
  >
    <div class="max-w-xl border-l-2 border-cyan-400 pl-6 text-left">
      <p class="text-sm font-semibold text-cyan-300">Тип звонка временно недоступен</p>
      <h1 class="mt-3 text-4xl font-bold tracking-tight">{{ disabledType.name }}</h1>
      <p class="mt-4 font-mono text-sm text-slate-400">{{ disabledType.durationMinutes }} минут</p>
      <p class="mt-5 leading-7 text-slate-300">
        Сейчас этот тип нельзя забронировать. Выберите другой доступный формат звонка.
      </p>
      <RouterLink
        :to="{ name: 'booking', query: route.query }"
        class="mt-8 inline-flex rounded-xl bg-cyan-600 px-5 py-3 font-semibold text-white hover:bg-cyan-700"
      >
        К активным типам звонков
      </RouterLink>
    </div>
  </div>

  <div
    v-else-if="hasUnknownType"
    class="grid min-h-svh place-items-center bg-slate-50 px-6 text-center"
  >
    <div class="max-w-lg">
      <p class="text-sm font-semibold text-blue-700">Тип звонка не найден</p>
      <h1 class="mt-3 text-4xl font-bold tracking-tight">Эта ссылка больше не работает</h1>
      <p class="mt-5 leading-7 text-slate-600">
        Возможно, тип звонка был удалён. Выберите один из доступных вариантов.
      </p>
      <RouterLink
        :to="{ name: 'booking', query: route.query }"
        class="mt-8 inline-flex rounded-xl bg-blue-600 px-5 py-3 font-semibold text-white hover:bg-blue-700"
      >
        К активным типам звонков
      </RouterLink>
    </div>
  </div>

  <component
    :is="variantComponent"
    v-else
    :types="activeCallTypes"
    :dates="dates"
    :selected-type="selectedType"
    :selected-day="selectedDay"
    :state="state"
    @select-type="selectType"
    @select-date="selectDate"
    @select-time="selectTime"
    @back-to-types="backToTypes"
    @back-to-slot="backToSlot"
    @submit="submitBooking"
    @restart="restart"
    @update-visitor-name="updateVisitorName"
    @update-visitor-email="updateVisitorEmail"
    @update-visitor-notes="updateVisitorNotes"
  />

  <PrototypeSwitcher
    v-if="showPrototypeTools"
    :current="variant"
    :state="prototypeState"
    @change="changeVariant"
  />
</template>
