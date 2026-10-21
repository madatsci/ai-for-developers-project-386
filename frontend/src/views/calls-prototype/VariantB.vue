<script setup lang="ts">
import { computed, ref } from 'vue'
import {
  callTypeNames,
  dateKey,
  formatDate,
  formatTime,
  getCallStatus,
  type CallFilters,
  type CallItem,
  type PrototypeScenario,
} from './model'

const props = defineProps<{
  calls: CallItem[]
  filters: CallFilters
  scenario: PrototypeScenario
  selectedCallId: string | null
}>()

const emit = defineEmits<{
  'update:filters': [filters: CallFilters]
  selectCall: [id: string]
}>()

const mobileDetailsOpen = ref(false)
const selectedCall = computed(() => props.calls.find((call) => call.id === props.selectedCallId))
const groupedCalls = computed(() => {
  const groups = new Map<string, CallItem[]>()
  for (const call of props.calls) {
    const key = dateKey(call.startsAt)
    groups.set(key, [...(groups.get(key) ?? []), call])
  }
  return Array.from(groups.entries()).map(([key, calls]) => ({ key, calls }))
})

function updateFilters(change: Partial<CallFilters>) {
  emit('update:filters', { ...props.filters, ...change })
}

function selectCall(id: string) {
  emit('selectCall', id)
  mobileDetailsOpen.value = true
}

function formatCallCount(count: number) {
  const pluralRule = new Intl.PluralRules('ru').select(count)
  if (pluralRule === 'one') return `${count} звонок`
  if (pluralRule === 'few') return `${count} звонка`
  return `${count} звонков`
}
</script>

<template>
  <div class="min-h-svh bg-slate-100 pb-28 text-slate-950">
    <div class="grid min-h-svh lg:grid-cols-[15rem_1fr]">
      <aside class="hidden border-r border-slate-800 bg-slate-950 p-5 text-slate-300 lg:block">
        <div class="flex items-center gap-3 text-white">
          <div class="grid size-9 place-items-center rounded-lg bg-blue-600 text-sm font-bold">
            КЗ
          </div>
          <span class="font-semibold">Календарь звонков</span>
        </div>
        <nav class="mt-10 space-y-1 text-sm">
          <a class="block rounded-xl bg-white/10 px-3 py-2.5 font-semibold text-white">Звонки</a>
          <a class="block rounded-xl px-3 py-2.5">Типы и расписания</a>
          <a class="block rounded-xl px-3 py-2.5">Режим работы</a>
        </nav>
        <div class="mt-10 border-t border-white/10 pt-5 text-xs leading-5 text-slate-500">
          Один организатор<br />Europe/Moscow
        </div>
      </aside>

      <main class="min-w-0">
        <header class="border-b border-slate-200 bg-white px-5 py-5 lg:px-7">
          <div class="flex flex-wrap items-center justify-between gap-4">
            <div>
              <p class="text-xs font-bold tracking-wider text-blue-700 uppercase">
                Вариант B · Журнал
              </p>
              <h1 class="mt-1 text-2xl font-bold">Звонки</h1>
            </div>
            <div class="flex rounded-lg border border-slate-200 p-1">
              <button
                class="rounded-md px-3 py-1.5 text-sm font-semibold"
                :class="
                  filters.period === 'upcoming' ? 'bg-slate-900 text-white' : 'text-slate-500'
                "
                @click="updateFilters({ period: 'upcoming' })"
              >
                Предстоящие
              </button>
              <button
                class="rounded-md px-3 py-1.5 text-sm font-semibold"
                :class="filters.period === 'past' ? 'bg-slate-900 text-white' : 'text-slate-500'"
                @click="updateFilters({ period: 'past' })"
              >
                Прошедшие
              </button>
            </div>
          </div>
        </header>

        <div class="grid min-h-[calc(100svh-85px)] xl:grid-cols-[minmax(0,1fr)_22rem]">
          <section class="min-w-0 p-5 lg:p-7">
            <div class="flex flex-wrap gap-3">
              <input
                :value="filters.query"
                type="search"
                placeholder="Поиск по посетителю, email, комментарию"
                class="min-w-64 flex-1 rounded-xl border border-slate-300 bg-white px-3 py-2.5 text-sm"
                @input="updateFilters({ query: ($event.target as HTMLInputElement).value })"
              />
              <select
                :value="filters.typeName"
                aria-label="Фильтр по типу звонка"
                class="rounded-xl border border-slate-300 bg-white px-3 py-2.5 text-sm"
                @change="updateFilters({ typeName: ($event.target as HTMLSelectElement).value })"
              >
                <option value="">Все типы</option>
                <option v-for="typeName in callTypeNames" :key="typeName" :value="typeName">
                  {{ typeName }}
                </option>
              </select>
            </div>

            <div
              v-if="scenario === 'loading'"
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

            <div
              v-else-if="scenario === 'error'"
              class="mt-5 rounded-2xl border border-red-200 bg-white p-6"
            >
              <p class="font-bold text-red-900">Список звонков недоступен</p>
              <p class="mt-2 text-sm text-slate-600">Данные не изменены. Повторите запрос.</p>
              <button
                class="mt-4 rounded-lg bg-slate-900 px-4 py-2 text-sm font-semibold text-white"
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
                  filters.query || filters.typeName
                    ? 'Ничего не найдено'
                    : filters.period === 'upcoming'
                      ? 'Предстоящих звонков нет'
                      : 'Прошедших звонков нет'
                }}
              </h2>
              <p class="mt-2 text-sm text-slate-500">
                {{
                  filters.query || filters.typeName
                    ? 'Сбросьте фильтры и попробуйте снова.'
                    : filters.period === 'upcoming'
                      ? 'Новые звонки появятся здесь после бронирования.'
                      : 'Завершившиеся звонки появятся здесь.'
                }}
              </p>
              <button
                v-if="filters.query || filters.typeName"
                class="mt-4 text-sm font-semibold text-blue-700"
                @click="updateFilters({ query: '', typeName: '' })"
              >
                Сбросить фильтры
              </button>
            </div>

            <div
              v-else
              class="mt-5 overflow-x-auto rounded-2xl border border-slate-200 bg-white shadow-sm"
            >
              <table class="w-full min-w-[48rem] border-collapse text-left text-sm">
                <thead class="bg-slate-50 text-xs font-semibold text-slate-500 uppercase">
                  <tr>
                    <th class="px-4 py-3">Время ↑</th>
                    <th class="px-4 py-3">Посетитель</th>
                    <th class="px-4 py-3">Тип звонка</th>
                    <th class="px-4 py-3">Комментарий</th>
                  </tr>
                </thead>
                <tbody v-for="group in groupedCalls" :key="group.key">
                  <tr class="border-y border-slate-200 bg-slate-100">
                    <th colspan="4" class="px-4 py-3 text-left text-slate-700">
                      <span class="flex items-baseline justify-between gap-4">
                        <span class="text-base font-bold capitalize">
                          {{ formatDate(group.calls[0]!.startsAt, { weekday: 'long' }) }}
                        </span>
                        <span class="text-xs font-medium text-slate-400">
                          {{ formatCallCount(group.calls.length) }}
                        </span>
                      </span>
                    </th>
                  </tr>
                  <tr
                    v-for="call in group.calls"
                    :key="call.id"
                    class="cursor-pointer border-t border-slate-100 hover:bg-blue-50/60"
                    :class="selectedCallId === call.id ? 'bg-blue-50' : ''"
                    @click="selectCall(call.id)"
                  >
                    <td class="whitespace-nowrap px-4 py-4">
                      <div class="flex items-center gap-2 font-semibold">
                        {{ formatTime(call.startsAt) }}–{{ formatTime(call.endsAt) }}
                        <span
                          v-if="getCallStatus(call) === 'in-progress'"
                          class="rounded-full bg-emerald-100 px-2 py-0.5 text-[10px] font-bold text-emerald-800"
                        >
                          СЕЙЧАС
                        </span>
                      </div>
                      <div class="mt-1 text-xs text-slate-400">Москва</div>
                    </td>
                    <td class="px-4 py-4">
                      <div class="font-semibold">{{ call.visitorName }}</div>
                      <div class="mt-1 text-xs text-slate-500">{{ call.visitorEmail }}</div>
                    </td>
                    <td class="px-4 py-4">
                      <div>{{ call.typeName }}</div>
                      <div class="mt-1 text-xs text-slate-400">
                        {{ call.durationMinutes }} минут
                      </div>
                    </td>
                    <td class="max-w-56 px-4 py-4 text-slate-600">
                      <span v-if="call.visitorNotes" class="line-clamp-2">{{
                        call.visitorNotes
                      }}</span>
                      <span v-else class="text-slate-300">—</span>
                    </td>
                  </tr>
                </tbody>
              </table>
              <div class="border-t border-slate-100 px-4 py-3">
                <div class="flex flex-wrap items-center justify-between gap-3">
                  <p class="text-xs text-slate-400">
                    {{ formatCallCount(calls.length) }} · сортировка по времени
                  </p>
                  <button
                    v-if="scenario !== 'load-more-error'"
                    class="rounded-lg border border-slate-200 px-3 py-1.5 text-xs font-semibold text-slate-700"
                  >
                    Загрузить ещё
                  </button>
                </div>
                <div
                  v-if="scenario === 'load-more-error'"
                  class="mt-3 flex flex-wrap items-center justify-between gap-3 rounded-xl bg-red-50 px-3 py-2 text-xs text-red-800"
                >
                  <span>Не удалось загрузить следующую страницу. Показанные звонки сохранены.</span>
                  <button class="font-bold">Повторить</button>
                </div>
              </div>
            </div>
          </section>

          <aside
            class="border-t border-slate-200 bg-white p-6 xl:static xl:block xl:border-t-0 xl:border-l"
            :class="
              mobileDetailsOpen ? 'fixed inset-0 z-40 block overflow-y-auto' : 'hidden xl:block'
            "
          >
            <button
              class="mb-6 inline-flex items-center gap-2 text-sm font-semibold text-blue-700 xl:hidden"
              @click="mobileDetailsOpen = false"
            >
              ← Назад к списку
            </button>

            <div v-if="scenario === 'loading'" class="space-y-5" aria-label="Загрузка деталей">
              <div class="h-3 w-24 animate-pulse rounded bg-slate-200" />
              <div class="h-7 w-48 animate-pulse rounded bg-slate-200" />
              <div class="h-4 w-40 animate-pulse rounded bg-slate-200" />
              <div class="mt-8 h-24 animate-pulse rounded-xl bg-slate-100" />
            </div>

            <template
              v-else-if="selectedCall && (scenario === 'ready' || scenario === 'load-more-error')"
            >
              <p class="text-xs font-bold tracking-wider text-slate-400 uppercase">Детали звонка</p>
              <div class="mt-5 flex items-center justify-between gap-3">
                <h2 class="text-xl font-bold">{{ selectedCall.visitorName }}</h2>
                <span
                  v-if="getCallStatus(selectedCall) === 'in-progress'"
                  class="rounded-full bg-emerald-100 px-2.5 py-1 text-xs font-bold text-emerald-800"
                >
                  Идёт сейчас
                </span>
              </div>
              <a class="mt-1 block text-sm text-blue-700">{{ selectedCall.visitorEmail }}</a>

              <dl class="mt-7 space-y-5 text-sm">
                <div>
                  <dt class="text-xs font-semibold text-slate-400 uppercase">Когда</dt>
                  <dd class="mt-1 font-semibold">
                    {{ formatDate(selectedCall.startsAt, { weekday: 'long' }) }},
                    {{ formatTime(selectedCall.startsAt) }}–{{ formatTime(selectedCall.endsAt) }}
                  </dd>
                </div>
                <div>
                  <dt class="text-xs font-semibold text-slate-400 uppercase">Тип звонка</dt>
                  <dd class="mt-1 font-semibold">{{ selectedCall.typeName }}</dd>
                </div>
                <div>
                  <dt class="text-xs font-semibold text-slate-400 uppercase">
                    Комментарий посетителя
                  </dt>
                  <dd class="mt-2 whitespace-pre-line leading-6 text-slate-600">
                    {{ selectedCall.visitorNotes || 'Комментарий не оставлен' }}
                  </dd>
                </div>
              </dl>
              <p class="mt-8 rounded-xl bg-slate-100 p-3 text-xs leading-5 text-slate-500">
                Прототип намеренно не предлагает отмену, перенос или редактирование: эти действия
                вне текущего объёма.
              </p>
            </template>
            <div v-else class="rounded-xl bg-slate-50 p-4 text-sm text-slate-400">
              Выберите звонок, чтобы увидеть детали.
            </div>
          </aside>
        </div>
      </main>
    </div>
  </div>
</template>
