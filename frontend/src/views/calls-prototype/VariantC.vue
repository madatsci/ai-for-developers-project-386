<script setup lang="ts">
import { computed, ref, watch } from 'vue'
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

const activeDate = ref<string | null>(null)
const dates = computed(() => {
  const groups = new Map<string, CallItem[]>()
  for (const call of props.calls) {
    const key = dateKey(call.startsAt)
    groups.set(key, [...(groups.get(key) ?? []), call])
  }
  return Array.from(groups.entries()).map(([key, calls]) => ({ key, calls }))
})
const activeCalls = computed(
  () =>
    dates.value.find((group) => group.key === activeDate.value)?.calls ??
    dates.value[0]?.calls ??
    [],
)

watch(
  dates,
  (groups) => {
    if (!groups.some((group) => group.key === activeDate.value)) {
      activeDate.value = groups[0]?.key ?? null
    }
  },
  { immediate: true },
)

function updateFilters(change: Partial<CallFilters>) {
  emit('update:filters', { ...props.filters, ...change })
}
</script>

<template>
  <div class="min-h-svh bg-[#f8f6f0] pb-28 text-[#29251f]">
    <header class="border-b-2 border-[#29251f] bg-[#f8f6f0] px-5 py-5">
      <div class="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-4">
        <div>
          <p class="text-xs font-black tracking-[0.18em] uppercase">Календарь звонков / admin</p>
          <h1 class="mt-1 font-serif text-3xl font-bold">Рабочий день организатора</h1>
        </div>
        <nav class="flex gap-5 text-sm font-bold">
          <a class="border-b-2 border-[#d84a2f] pb-1 text-[#d84a2f]">Звонки</a>
          <a>Типы и расписания</a>
          <a>Режим работы</a>
        </nav>
      </div>
    </header>

    <main class="mx-auto max-w-6xl px-5 py-7">
      <div
        class="flex flex-wrap items-center justify-between gap-4 border-b border-[#29251f]/25 pb-5"
      >
        <div>
          <p class="text-sm font-bold text-[#d84a2f]">Вариант C · День</p>
          <p class="mt-1 max-w-2xl text-sm text-[#6f675d]">
            Сначала дата, затем компактная повестка одного дня. Без глобального текстового поиска.
          </p>
        </div>
        <div class="flex items-center gap-2">
          <button
            class="rounded-full border-2 border-[#29251f] px-4 py-2 text-sm font-bold"
            :class="filters.period === 'upcoming' ? 'bg-[#29251f] text-white' : ''"
            @click="updateFilters({ period: 'upcoming', query: '' })"
          >
            Будущие
          </button>
          <button
            class="rounded-full border-2 border-[#29251f] px-4 py-2 text-sm font-bold"
            :class="filters.period === 'past' ? 'bg-[#29251f] text-white' : ''"
            @click="updateFilters({ period: 'past', query: '' })"
          >
            Архив
          </button>
        </div>
      </div>

      <div class="mt-5 flex flex-wrap gap-2">
        <button
          class="rounded-full border px-3 py-1.5 text-xs font-bold"
          :class="
            !filters.typeName ? 'border-[#d84a2f] bg-[#d84a2f] text-white' : 'border-[#29251f]/25'
          "
          @click="updateFilters({ typeName: '' })"
        >
          Все типы
        </button>
        <button
          v-for="typeName in callTypeNames"
          :key="typeName"
          class="rounded-full border px-3 py-1.5 text-xs font-bold"
          :class="
            filters.typeName === typeName
              ? 'border-[#d84a2f] bg-[#d84a2f] text-white'
              : 'border-[#29251f]/25'
          "
          @click="updateFilters({ typeName })"
        >
          {{ typeName }}
        </button>
      </div>

      <div v-if="scenario === 'loading'" class="mt-8 grid gap-5 md:grid-cols-[15rem_1fr]">
        <div class="h-96 animate-pulse rounded-2xl bg-[#e8e3d8]" />
        <div class="h-96 animate-pulse rounded-2xl bg-[#e8e3d8]" />
      </div>

      <div
        v-else-if="scenario === 'error'"
        class="mt-8 border-2 border-[#29251f] bg-[#ffd9c7] p-7 shadow-[6px_6px_0_#29251f]"
      >
        <p class="text-xs font-black tracking-wider uppercase">Ошибка загрузки</p>
        <h2 class="mt-2 font-serif text-2xl font-bold">Повестка временно недоступна</h2>
        <button
          class="mt-5 border-2 border-[#29251f] bg-[#29251f] px-4 py-2 text-sm font-bold text-white"
        >
          Попробовать снова
        </button>
      </div>

      <div
        v-else-if="calls.length === 0"
        class="mt-8 border-2 border-dashed border-[#29251f]/40 px-6 py-16 text-center"
      >
        <p class="font-serif text-5xl">—</p>
        <h2 class="mt-4 text-xl font-bold">
          {{ filters.typeName ? 'Для этого типа звонков нет' : 'Дней со звонками пока нет' }}
        </h2>
        <p class="mt-2 text-sm text-[#6f675d]">
          Выберите другой тип или переключитесь между будущими звонками и архивом.
        </p>
      </div>

      <div v-else class="mt-8 grid gap-6 md:grid-cols-[15rem_minmax(0,1fr)]">
        <aside class="border-2 border-[#29251f] bg-white p-3 shadow-[5px_5px_0_#29251f]">
          <p class="px-2 pb-3 text-xs font-black tracking-wider uppercase">Даты со звонками</p>
          <button
            v-for="group in dates"
            :key="group.key"
            class="mb-2 flex w-full items-center justify-between border-2 px-3 py-3 text-left"
            :class="
              activeDate === group.key
                ? 'border-[#d84a2f] bg-[#fff0e9]'
                : 'border-transparent hover:border-[#29251f]/30'
            "
            @click="activeDate = group.key"
          >
            <span>
              <span class="block text-xs font-bold capitalize text-[#6f675d]">
                {{ formatWeekday(group.calls[0]!.startsAt) }}
              </span>
              <span class="mt-0.5 block font-serif text-lg font-bold">
                {{ formatDate(group.calls[0]!.startsAt) }}
              </span>
            </span>
            <span
              class="grid size-8 place-items-center rounded-full bg-[#29251f] text-xs font-bold text-white"
            >
              {{ group.calls.length }}
            </span>
          </button>
        </aside>

        <section>
          <div class="flex items-end justify-between gap-4">
            <div>
              <p class="text-xs font-black tracking-wider text-[#d84a2f] uppercase">
                Выбранный день
              </p>
              <h2 class="mt-1 font-serif text-3xl font-bold capitalize">
                {{
                  activeCalls[0]
                    ? formatDate(activeCalls[0].startsAt, { weekday: 'long' })
                    : 'Нет звонков'
                }}
              </h2>
            </div>
            <p class="text-sm font-bold">{{ activeCalls.length }} звонка</p>
          </div>

          <div class="mt-5 border-t-2 border-[#29251f]">
            <article
              v-for="call in activeCalls"
              :key="call.id"
              class="grid cursor-pointer gap-4 border-b-2 border-[#29251f]/20 py-5 md:grid-cols-[7rem_1fr_auto]"
              :class="selectedCallId === call.id ? 'bg-[#fff0e9] px-4' : ''"
              @click="emit('selectCall', call.id)"
            >
              <div>
                <p class="font-serif text-2xl font-bold">{{ formatTime(call.startsAt) }}</p>
                <p class="text-xs text-[#6f675d]">{{ call.durationMinutes }} минут</p>
              </div>
              <div>
                <div class="flex flex-wrap items-center gap-2">
                  <h3 class="font-bold">{{ call.visitorName }}</h3>
                  <span
                    v-if="getCallStatus(call) === 'in-progress'"
                    class="rounded-full bg-[#167a5b] px-2 py-0.5 text-[10px] font-black text-white"
                  >
                    ИДЁТ СЕЙЧАС
                  </span>
                </div>
                <p class="mt-1 text-sm text-[#6f675d]">{{ call.visitorEmail }}</p>
                <p v-if="call.visitorNotes" class="mt-3 whitespace-pre-line text-sm leading-6">
                  {{ call.visitorNotes }}
                </p>
                <p v-else class="mt-3 text-sm text-[#9b9185]">Без комментария посетителя</p>
              </div>
              <p class="text-sm font-bold text-[#d84a2f] md:text-right">{{ call.typeName }}</p>
            </article>
          </div>
        </section>
      </div>
    </main>
  </div>
</template>
