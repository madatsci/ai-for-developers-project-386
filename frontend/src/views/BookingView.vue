<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import {
  getPublicCallType,
  getPublicCallTypeAvailability,
  listPublicCallTypes,
  type AvailabilityResponseBody,
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

const hasNoSlots = computed(
  () =>
    availability.value !== null && availability.value.days.every((day) => day.slots.length === 0),
)

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

    <div
      v-if="state === 'disabled' || state === 'not-found'"
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

          <div class="mt-6 grid gap-4 sm:grid-cols-2">
            <article
              v-for="day in availability.days"
              :key="day.date"
              data-availability-day
              class="rounded-xl border border-slate-200 bg-white p-5"
            >
              <h3 class="font-bold capitalize">{{ formatDate(day.date) }}</h3>
              <p class="mt-1 font-mono text-xs text-slate-400">{{ day.date }}</p>
              <div v-if="day.slots.length" class="mt-4 flex flex-wrap gap-2">
                <time
                  v-for="slot in day.slots"
                  :key="slot.starts_at"
                  :datetime="slot.starts_at"
                  class="rounded-lg bg-blue-50 px-3 py-2 font-mono text-sm font-semibold text-blue-800"
                >
                  {{ formatTime(slot.starts_at) }}–{{ formatTime(slot.ends_at) }}
                </time>
              </div>
              <p v-else class="mt-4 text-sm text-slate-500">Нет свободного времени</p>
            </article>
          </div>
        </template>
      </section>
    </div>
  </main>
</template>
