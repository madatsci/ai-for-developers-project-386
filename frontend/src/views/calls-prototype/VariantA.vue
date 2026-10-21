<script setup lang="ts">
import { computed } from 'vue'
import {
  callTypeNames,
  dateKey,
  formatDate,
  formatTime,
  formatWeekday,
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
</script>

<template>
  <div class="min-h-svh bg-[#f5f7fb] pb-28 text-slate-950">
    <header class="border-b border-slate-200 bg-white">
      <div class="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-4 px-5 py-4">
        <div class="flex items-center gap-3">
          <div class="grid size-10 place-items-center rounded-xl bg-blue-600 font-bold text-white">
            КЗ
          </div>
          <div>
            <p class="font-semibold">Календарь звонков</p>
            <p class="text-xs text-slate-500">Время по Москве</p>
          </div>
        </div>
        <nav class="flex gap-1 rounded-xl bg-slate-100 p-1 text-sm">
          <a class="rounded-lg bg-white px-3 py-2 font-semibold text-blue-700 shadow-sm">Звонки</a>
          <a class="px-3 py-2 text-slate-600">Типы и расписания</a>
          <a class="px-3 py-2 text-slate-600">Режим работы</a>
        </nav>
      </div>
    </header>

    <main class="mx-auto max-w-5xl px-5 py-8">
      <div class="flex flex-wrap items-end justify-between gap-5">
        <div>
          <p class="text-sm font-semibold text-blue-700">Вариант A · Повестка</p>
          <h1 class="mt-1 text-3xl font-bold tracking-tight">Звонки</h1>
          <p class="mt-2 text-sm text-slate-500">
            Хронологическая лента, сгруппированная по локальным датам.
          </p>
        </div>
        <div class="flex rounded-xl border border-slate-200 bg-white p-1 shadow-sm">
          <button
            class="rounded-lg px-4 py-2 text-sm font-semibold"
            :class="filters.period === 'upcoming' ? 'bg-blue-600 text-white' : 'text-slate-600'"
            @click="updateFilters({ period: 'upcoming' })"
          >
            Предстоящие
          </button>
          <button
            class="rounded-lg px-4 py-2 text-sm font-semibold"
            :class="filters.period === 'past' ? 'bg-blue-600 text-white' : 'text-slate-600'"
            @click="updateFilters({ period: 'past' })"
          >
            Прошедшие
          </button>
        </div>
      </div>

      <div
        class="mt-7 grid gap-3 rounded-2xl border border-slate-200 bg-white p-4 shadow-sm sm:grid-cols-[1fr_15rem]"
      >
        <label class="text-xs font-semibold text-slate-500">
          Найти посетителя, email или комментарий
          <input
            :value="filters.query"
            type="search"
            placeholder="Например, Анна или API"
            class="mt-1.5 w-full rounded-xl border border-slate-200 px-3 py-2.5 text-sm text-slate-950"
            @input="updateFilters({ query: ($event.target as HTMLInputElement).value })"
          />
        </label>
        <label class="text-xs font-semibold text-slate-500">
          Тип звонка
          <select
            :value="filters.typeName"
            class="mt-1.5 w-full rounded-xl border border-slate-200 px-3 py-2.5 text-sm text-slate-950"
            @change="updateFilters({ typeName: ($event.target as HTMLSelectElement).value })"
          >
            <option value="">Все типы</option>
            <option v-for="typeName in callTypeNames" :key="typeName" :value="typeName">
              {{ typeName }}
            </option>
          </select>
        </label>
      </div>

      <div v-if="scenario === 'loading'" class="mt-8 space-y-8" aria-label="Загрузка звонков">
        <div v-for="day in 2" :key="day">
          <div class="h-5 w-40 animate-pulse rounded bg-slate-200" />
          <div class="mt-3 space-y-3">
            <div v-for="row in 2" :key="row" class="h-28 animate-pulse rounded-2xl bg-white" />
          </div>
        </div>
      </div>

      <div
        v-else-if="scenario === 'error'"
        class="mt-8 rounded-2xl border border-red-200 bg-red-50 p-7"
      >
        <h2 class="text-lg font-bold text-red-950">Не удалось загрузить звонки</h2>
        <p class="mt-2 text-sm text-red-800">Проверьте соединение и повторите запрос.</p>
        <button class="mt-5 rounded-xl bg-red-700 px-4 py-2 text-sm font-semibold text-white">
          Повторить
        </button>
      </div>

      <div
        v-else-if="calls.length === 0"
        class="mt-8 rounded-2xl border border-dashed border-slate-300 bg-white px-6 py-14 text-center"
      >
        <div class="mx-auto grid size-12 place-items-center rounded-full bg-slate-100 text-xl">
          ◷
        </div>
        <h2 class="mt-4 text-lg font-bold">
          {{ filters.query || filters.typeName ? 'Ничего не найдено' : 'Звонков пока нет' }}
        </h2>
        <p class="mx-auto mt-2 max-w-md text-sm text-slate-500">
          {{
            filters.query || filters.typeName
              ? 'Измените поиск или сбросьте фильтр типа звонка.'
              : filters.period === 'upcoming'
                ? 'Новые бронирования появятся здесь в порядке времени.'
                : 'Здесь появятся завершившиеся звонки.'
          }}
        </p>
        <button
          v-if="filters.query || filters.typeName"
          class="mt-5 text-sm font-semibold text-blue-700"
          @click="updateFilters({ query: '', typeName: '' })"
        >
          Сбросить фильтры
        </button>
      </div>

      <div v-else class="mt-8 space-y-9">
        <section v-for="group in groupedCalls" :key="group.key">
          <div class="mb-3 flex items-baseline gap-2">
            <h2 class="text-lg font-bold capitalize">
              {{ formatDate(group.calls[0]!.startsAt, { weekday: 'long' }) }}
            </h2>
            <span class="text-sm text-slate-400">{{
              formatWeekday(group.calls[0]!.startsAt)
            }}</span>
            <span class="ml-auto text-xs text-slate-400">{{ group.calls.length }} звонка</span>
          </div>
          <div class="space-y-3">
            <button
              v-for="call in group.calls"
              :key="call.id"
              type="button"
              class="grid w-full gap-4 rounded-2xl border bg-white p-5 text-left shadow-sm transition hover:-translate-y-0.5 hover:shadow-md sm:grid-cols-[7rem_1fr_auto]"
              :class="
                selectedCallId === call.id
                  ? 'border-blue-400 ring-2 ring-blue-100'
                  : 'border-slate-200'
              "
              @click="emit('selectCall', call.id)"
            >
              <div>
                <p class="text-xl font-bold tabular-nums">{{ formatTime(call.startsAt) }}</p>
                <p class="mt-1 text-xs text-slate-400">до {{ formatTime(call.endsAt) }}</p>
                <span
                  v-if="getCallStatus(call) === 'in-progress'"
                  class="mt-2 inline-flex rounded-full bg-emerald-100 px-2 py-1 text-[11px] font-bold text-emerald-800"
                >
                  Идёт сейчас
                </span>
              </div>
              <div>
                <p class="font-semibold">{{ call.visitorName }}</p>
                <p class="mt-1 text-sm text-slate-500">{{ call.visitorEmail }}</p>
                <p v-if="call.visitorNotes" class="mt-3 line-clamp-2 text-sm text-slate-600">
                  {{ call.visitorNotes }}
                </p>
                <p v-else class="mt-3 text-sm text-slate-400">Без комментария</p>
              </div>
              <div class="sm:text-right">
                <p class="text-sm font-semibold text-blue-700">{{ call.typeName }}</p>
                <p class="mt-1 text-xs text-slate-400">{{ call.durationMinutes }} минут</p>
              </div>
            </button>
          </div>
        </section>
      </div>
    </main>
  </div>
</template>
