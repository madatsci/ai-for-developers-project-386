<script setup lang="ts">
import { computed, nextTick, onUnmounted, ref, watch } from 'vue'
import {
  listAdminCalls,
  listAdminCallTypes,
  type AdminCall,
  type AdminCallType,
} from '@/api/generated'
import AdminSidebar from '@/components/AdminSidebar.vue'

type CallPeriod = 'upcoming' | 'past'

const period = ref<CallPeriod>('upcoming')
const callTypeId = ref('')
const search = ref('')
const callTypes = ref<AdminCallType[]>([])
const calls = ref<AdminCall[]>([])
const asOf = ref('')
const nextCursor = ref<string | null>(null)
const selectedCallId = ref<string | null>(null)
const loading = ref(true)
const initialError = ref(false)
const loadingMore = ref(false)
const loadMoreError = ref(false)
const mobileDetailsOpen = ref(false)
const mobileListScroll = ref(0)
const currentTime = ref(Date.now())
let requestId = 0
const clock = window.setInterval(() => {
  currentTime.value = Date.now()
}, 30_000)

onUnmounted(() => window.clearInterval(clock))

const selectedCall = computed(() => calls.value.find((call) => call.id === selectedCallId.value))
const filtersActive = computed(() => callTypeId.value !== '' || search.value.trim() !== '')
const groupedCalls = computed(() => {
  const groups = new Map<string, AdminCall[]>()
  for (const call of calls.value) {
    const key = dateKey(call.starts_at)
    groups.set(key, [...(groups.get(key) ?? []), call])
  }
  return Array.from(groups.entries()).map(([key, items]) => ({ key, items }))
})

void loadCallTypes()

watch([period, callTypeId, search], loadFirstPage, { immediate: true })

async function loadCallTypes() {
  const response = await listAdminCallTypes()
  if (response.data) callTypes.value = response.data.items
}

function requestQuery(cursor?: string) {
  const query: {
    period: CallPeriod
    call_type_id?: string
    q?: string
    cursor?: string
  } = { period: period.value }
  if (callTypeId.value) query.call_type_id = callTypeId.value
  const normalizedSearch = search.value.trim()
  if (normalizedSearch) query.q = normalizedSearch
  if (cursor) query.cursor = cursor
  return query
}

async function loadFirstPage() {
  const currentRequest = ++requestId
  loading.value = true
  initialError.value = false
  loadMoreError.value = false
  loadingMore.value = false
  mobileDetailsOpen.value = false
  const response = await listAdminCalls({ query: requestQuery() })
  if (currentRequest !== requestId) return
  loading.value = false
  if (!response.data) {
    calls.value = []
    nextCursor.value = null
    selectedCallId.value = null
    initialError.value = true
    return
  }
  calls.value = response.data.items
  asOf.value = response.data.as_of
  nextCursor.value = response.data.next_cursor
  selectedCallId.value = response.data.items[0]?.id ?? null
}

async function loadMore() {
  if (!nextCursor.value || loadingMore.value) return
  const currentRequest = requestId
  loadingMore.value = true
  loadMoreError.value = false
  const cursor = nextCursor.value
  const response = await listAdminCalls({ query: requestQuery(cursor) })
  if (currentRequest !== requestId) return
  loadingMore.value = false
  if (!response.data) {
    loadMoreError.value = true
    return
  }
  calls.value.push(...response.data.items)
  nextCursor.value = response.data.next_cursor
}

function selectCall(id: string, openMobile = false) {
  selectedCallId.value = id
  if (openMobile) {
    mobileListScroll.value = window.scrollY
    mobileDetailsOpen.value = true
  }
}

async function closeMobileDetails() {
  mobileDetailsOpen.value = false
  await nextTick()
  document.documentElement.scrollTop = mobileListScroll.value
}

function resetFilters() {
  callTypeId.value = ''
  search.value = ''
}

function dateKey(value: string) {
  const parts = new Intl.DateTimeFormat('en', {
    timeZone: 'Europe/Moscow',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).formatToParts(new Date(value))
  const part = (type: Intl.DateTimeFormatPartTypes) =>
    parts.find((candidate) => candidate.type === type)?.value ?? ''
  return `${part('year')}-${part('month')}-${part('day')}`
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat('ru-RU', {
    timeZone: 'Europe/Moscow',
    weekday: 'long',
    day: 'numeric',
    month: 'long',
  }).format(new Date(value))
}

function formatTime(value: string) {
  return new Intl.DateTimeFormat('ru-RU', {
    timeZone: 'Europe/Moscow',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value))
}

function formatCallCount(count: number) {
  const plural = new Intl.PluralRules('ru-RU').select(count)
  if (plural === 'one') return `${count} звонок`
  if (plural === 'few') return `${count} звонка`
  return `${count} звонков`
}

function isCurrent(call: AdminCall) {
  const now = currentTime.value
  return new Date(call.starts_at).getTime() <= now && now < new Date(call.ends_at).getTime()
}
</script>

<template>
  <div class="min-h-svh bg-slate-100 text-slate-950 lg:grid lg:grid-cols-[15rem_1fr]">
    <AdminSidebar active="calls" note="Все даты и время показаны по Москве." />

    <main class="min-w-0">
      <header class="border-b border-slate-200 bg-white px-5 py-5 lg:px-7">
        <div class="flex flex-wrap items-center justify-between gap-4">
          <div>
            <p class="text-xs font-bold tracking-wider text-blue-700 uppercase">Журнал</p>
            <h1 class="mt-1 text-2xl font-bold">Звонки</h1>
          </div>
          <div class="flex rounded-lg border border-slate-200 p-1">
            <button
              data-period="upcoming"
              class="rounded-md px-3 py-1.5 text-sm font-semibold"
              :class="period === 'upcoming' ? 'bg-slate-900 text-white' : 'text-slate-500'"
              @click="period = 'upcoming'"
            >
              Предстоящие
            </button>
            <button
              data-period="past"
              class="rounded-md px-3 py-1.5 text-sm font-semibold"
              :class="period === 'past' ? 'bg-slate-900 text-white' : 'text-slate-500'"
              @click="period = 'past'"
            >
              Прошедшие
            </button>
          </div>
        </div>
      </header>

      <div class="grid min-h-[calc(100svh-85px)] xl:grid-cols-[minmax(0,1fr)_22rem]">
        <section
          data-call-list
          class="min-w-0 p-5 lg:p-7"
          :class="mobileDetailsOpen ? 'hidden xl:block' : 'block'"
        >
          <div class="flex flex-wrap gap-3">
            <input
              v-model="search"
              type="search"
              maxlength="200"
              placeholder="Поиск по имени, email или комментарию"
              class="min-w-64 flex-1 rounded-xl border border-slate-300 bg-white px-3 py-2.5 text-sm"
            />
            <select
              v-model="callTypeId"
              name="call-type-filter"
              aria-label="Фильтр по типу звонка"
              class="rounded-xl border border-slate-300 bg-white px-3 py-2.5 text-sm"
            >
              <option value="">Все типы</option>
              <option v-for="callType in callTypes" :key="callType.id" :value="callType.id">
                {{ callType.name }}
              </option>
            </select>
          </div>

          <div
            v-if="loading"
            class="mt-5 overflow-hidden rounded-2xl border border-slate-200 bg-white"
            aria-label="Загрузка звонков"
          >
            <div v-for="row in 6" :key="row" class="grid grid-cols-4 gap-5 border-b p-4">
              <span
                v-for="column in 4"
                :key="column"
                class="h-4 animate-pulse rounded bg-slate-200"
              />
            </div>
          </div>

          <div v-else-if="initialError" class="mt-5 rounded-2xl border border-red-200 bg-white p-6">
            <p class="font-bold text-red-900">Не удалось загрузить звонки</p>
            <p class="mt-2 text-sm text-slate-600">Проверьте соединение и повторите запрос.</p>
            <button
              data-retry-initial
              class="mt-4 rounded-lg bg-slate-900 px-4 py-2 text-sm font-semibold text-white"
              @click="loadFirstPage"
            >
              Повторить
            </button>
          </div>

          <div
            v-else-if="calls.length === 0"
            class="mt-5 rounded-2xl border border-slate-200 bg-white p-10 text-center"
          >
            <p class="text-3xl text-slate-300">∅</p>
            <h2 class="mt-3 font-bold">
              {{
                filtersActive
                  ? 'Ничего не найдено'
                  : period === 'upcoming'
                    ? 'Предстоящих звонков нет'
                    : 'Прошедших звонков нет'
              }}
            </h2>
            <p class="mt-2 text-sm text-slate-500">
              {{
                filtersActive
                  ? 'Сбросьте фильтры и попробуйте снова.'
                  : period === 'upcoming'
                    ? 'Новые звонки появятся здесь после бронирования.'
                    : 'Завершившиеся звонки появятся здесь.'
              }}
            </p>
            <button
              v-if="filtersActive"
              data-reset-filters
              class="mt-4 text-sm font-semibold text-blue-700"
              @click="resetFilters"
            >
              Сбросить фильтры
            </button>
          </div>

          <div
            v-else
            class="mt-5 overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm"
          >
            <table class="hidden w-full min-w-[48rem] border-collapse text-left text-sm xl:table">
              <thead class="bg-slate-50 text-xs font-semibold text-slate-500 uppercase">
                <tr>
                  <th class="px-4 py-3">Время</th>
                  <th class="px-4 py-3">Посетитель</th>
                  <th class="px-4 py-3">Тип звонка</th>
                  <th class="px-4 py-3">Комментарий</th>
                </tr>
              </thead>
              <tbody v-for="group in groupedCalls" :key="group.key">
                <tr class="border-y border-slate-200 bg-slate-100">
                  <th colspan="4" class="px-4 py-3 text-left text-slate-700">
                    <span class="flex items-baseline justify-between gap-4">
                      <span class="text-base font-bold capitalize">{{
                        formatDate(group.items[0]!.starts_at)
                      }}</span>
                      <span class="text-xs font-medium text-slate-400">{{
                        formatCallCount(group.items.length)
                      }}</span>
                    </span>
                  </th>
                </tr>
                <tr
                  v-for="call in group.items"
                  :key="call.id"
                  class="cursor-pointer border-t border-slate-100 hover:bg-blue-50/60"
                  :class="selectedCallId === call.id ? 'bg-blue-50' : ''"
                  @click="selectCall(call.id)"
                >
                  <td class="whitespace-nowrap px-4 py-4">
                    <div class="flex items-center gap-2 font-semibold">
                      {{ formatTime(call.starts_at) }}–{{ formatTime(call.ends_at) }}
                      <span
                        v-if="isCurrent(call)"
                        class="rounded-full bg-emerald-100 px-2 py-0.5 text-[10px] font-bold text-emerald-800"
                        >Сейчас</span
                      >
                    </div>
                    <div class="mt-1 text-xs text-slate-400">Москва</div>
                  </td>
                  <td class="px-4 py-4">
                    <div class="font-semibold">{{ call.visitor_name }}</div>
                    <div class="mt-1 text-xs text-slate-500">{{ call.visitor_email }}</div>
                  </td>
                  <td class="px-4 py-4">
                    <div>{{ call.call_type_name }}</div>
                    <div class="mt-1 text-xs text-slate-400">{{ call.duration_minutes }} минут</div>
                  </td>
                  <td class="max-w-56 px-4 py-4 text-slate-600">
                    <span v-if="call.visitor_notes" class="line-clamp-2">{{
                      call.visitor_notes
                    }}</span>
                    <span v-else class="text-slate-300">—</span>
                  </td>
                </tr>
              </tbody>
            </table>

            <div class="divide-y divide-slate-200 xl:hidden">
              <section v-for="group in groupedCalls" :key="group.key">
                <div class="flex items-center justify-between bg-slate-100 px-4 py-3">
                  <h2 class="font-bold capitalize">{{ formatDate(group.items[0]!.starts_at) }}</h2>
                  <span class="text-xs text-slate-500">{{
                    formatCallCount(group.items.length)
                  }}</span>
                </div>
                <button
                  v-for="call in group.items"
                  :key="call.id"
                  :data-select-call="call.id"
                  class="block w-full border-t border-slate-100 p-4 text-left"
                  @click="selectCall(call.id, true)"
                >
                  <span class="flex items-center justify-between gap-3">
                    <span class="font-semibold"
                      >{{ formatTime(call.starts_at) }}–{{ formatTime(call.ends_at) }}</span
                    >
                    <span
                      v-if="isCurrent(call)"
                      class="rounded-full bg-emerald-100 px-2 py-0.5 text-[10px] font-bold text-emerald-800"
                      >Сейчас</span
                    >
                  </span>
                  <span class="mt-2 block font-semibold">{{ call.visitor_name }}</span>
                  <span class="mt-1 block text-sm text-slate-500">{{ call.visitor_email }}</span>
                  <span class="mt-2 block text-sm text-blue-700"
                    >{{ call.call_type_name }} · {{ call.duration_minutes }} минут</span
                  >
                  <span
                    v-if="call.visitor_notes"
                    class="mt-2 line-clamp-2 block text-sm text-slate-600"
                    >{{ call.visitor_notes }}</span
                  >
                </button>
              </section>
            </div>

            <div class="border-t border-slate-100 px-4 py-3">
              <div class="flex flex-wrap items-center justify-between gap-3">
                <p class="text-xs text-slate-400">
                  {{ formatCallCount(calls.length) }} · время по Москве
                </p>
                <button
                  v-if="nextCursor"
                  data-load-more
                  class="rounded-lg border border-slate-200 px-3 py-1.5 text-xs font-semibold text-slate-700 disabled:opacity-50"
                  :disabled="loadingMore"
                  @click="loadMore"
                >
                  {{ loadingMore ? 'Загружаем…' : 'Загрузить ещё' }}
                </button>
              </div>
              <div
                v-if="loadMoreError"
                class="mt-3 flex flex-wrap items-center justify-between gap-3 rounded-xl bg-red-50 px-3 py-2 text-xs text-red-800"
              >
                <span>Не удалось загрузить следующую страницу. Показанные звонки сохранены.</span>
                <button data-retry-load-more class="font-bold" @click="loadMore">Повторить</button>
              </div>
            </div>
          </div>
        </section>

        <aside
          data-call-details
          class="border-t border-slate-200 bg-white p-6 xl:static xl:block xl:border-t-0 xl:border-l"
          :class="
            mobileDetailsOpen ? 'fixed inset-0 z-40 block overflow-y-auto' : 'hidden xl:block'
          "
        >
          <button
            data-mobile-calls-back
            class="mb-6 inline-flex items-center gap-2 text-sm font-semibold text-blue-700 xl:hidden"
            @click="closeMobileDetails"
          >
            ← Назад к списку
          </button>
          <div v-if="loading" class="space-y-5" aria-label="Загрузка деталей">
            <div class="h-3 w-24 animate-pulse rounded bg-slate-200" />
            <div class="h-7 w-48 animate-pulse rounded bg-slate-200" />
            <div class="h-4 w-40 animate-pulse rounded bg-slate-200" />
          </div>
          <template v-else-if="selectedCall">
            <p class="text-xs font-bold tracking-wider text-slate-400 uppercase">Детали звонка</p>
            <div class="mt-5 flex items-center justify-between gap-3">
              <h2 class="text-xl font-bold">{{ selectedCall.visitor_name }}</h2>
              <span
                v-if="isCurrent(selectedCall)"
                class="rounded-full bg-emerald-100 px-2.5 py-1 text-xs font-bold text-emerald-800"
                >Сейчас</span
              >
            </div>
            <a
              :href="`mailto:${selectedCall.visitor_email}`"
              class="mt-1 block text-sm text-blue-700"
              >{{ selectedCall.visitor_email }}</a
            >
            <dl class="mt-7 space-y-5 text-sm">
              <div>
                <dt class="text-xs font-semibold text-slate-400 uppercase">Когда</dt>
                <dd class="mt-1 font-semibold">
                  {{ formatDate(selectedCall.starts_at) }},
                  {{ formatTime(selectedCall.starts_at) }}–{{ formatTime(selectedCall.ends_at) }}
                </dd>
              </div>
              <div>
                <dt class="text-xs font-semibold text-slate-400 uppercase">Тип звонка</dt>
                <dd class="mt-1 font-semibold">
                  {{ selectedCall.call_type_name }} · {{ selectedCall.duration_minutes }} минут
                </dd>
              </div>
              <div>
                <dt class="text-xs font-semibold text-slate-400 uppercase">
                  Комментарий посетителя
                </dt>
                <dd class="mt-2 whitespace-pre-line leading-6 text-slate-600">
                  {{ selectedCall.visitor_notes || 'Комментарий не оставлен' }}
                </dd>
              </div>
            </dl>
            <p v-if="asOf" class="mt-8 text-xs leading-5 text-slate-400">
              Список сформирован по состоянию на {{ formatTime(asOf) }}.
            </p>
          </template>
          <div
            v-else-if="!loading && !initialError"
            class="rounded-xl bg-slate-50 p-4 text-sm text-slate-400"
          >
            Выберите звонок, чтобы увидеть детали.
          </div>
        </aside>
      </div>
    </main>
  </div>
</template>
