<script setup lang="ts">
import { computed, ref } from 'vue'
import type { CallType, Schedule } from './model'

const props = defineProps<{
  callTypes: CallType[]
  schedules: Schedule[]
  workingHours: string
  message: string
}>()

const emit = defineEmits<{
  toggleType: [id: number]
  reassignType: [typeId: number, scheduleId: number]
  deleteType: [id: number]
  addType: []
  addSchedule: []
  deleteSchedule: [id: number]
  showValidation: []
}>()

const selectedKind = ref<'type' | 'schedule'>('type')
const selectedId = ref(props.callTypes[0]?.id ?? 0)
const selectedType = computed(() => props.callTypes.find((item) => item.id === selectedId.value))
const selectedSchedule = computed(() =>
  props.schedules.find((item) => item.id === selectedId.value),
)

function selectType(id: number) {
  selectedKind.value = 'type'
  selectedId.value = id
}

function selectSchedule(id: number) {
  selectedKind.value = 'schedule'
  selectedId.value = id
}
</script>

<template>
  <div class="min-h-svh bg-[#f4f3ef] pb-28 text-[#20201d]">
    <header class="border-b border-black/10 bg-[#f8f7f3] px-5 py-3">
      <div class="flex items-center justify-between">
        <strong class="font-mono text-sm tracking-tight">CALENDAR / ADMIN</strong>
        <span class="font-mono text-xs text-black/50">{{ workingHours }}</span>
      </div>
    </header>

    <main class="grid min-h-[calc(100svh-53px)] lg:grid-cols-[13rem_1fr_24rem]">
      <nav class="border-r border-black/10 p-4">
        <p class="px-3 py-2 font-mono text-[10px] tracking-wider text-black/40 uppercase">
          Управление
        </p>
        <button
          class="w-full rounded-lg bg-black px-3 py-2.5 text-left text-sm font-semibold text-white"
        >
          Типы и расписания
        </button>
        <button class="mt-1 w-full rounded-lg px-3 py-2.5 text-left text-sm text-black/50">
          Режим работы
        </button>
        <div
          class="mt-8 rounded-xl border border-black/10 bg-white/60 p-3 text-xs leading-5 text-black/60"
        >
          Изменения влияют только на новые свободные слоты.
        </div>
      </nav>

      <section class="min-w-0 border-r border-black/10 p-5 lg:p-7">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div>
            <p class="font-mono text-xs text-black/45">ЕДИНАЯ РАБОЧАЯ ОБЛАСТЬ</p>
            <h1 class="mt-1 text-2xl font-bold">Типы звонков и расписания</h1>
          </div>
          <div class="flex gap-2">
            <button
              class="rounded-lg border border-black/15 bg-white px-3 py-2 text-sm font-semibold"
              @click="emit('addSchedule')"
            >
              + Расписание
            </button>
            <button
              class="rounded-lg bg-[#ff5c35] px-3 py-2 text-sm font-semibold text-white"
              @click="emit('addType')"
            >
              + Тип звонка
            </button>
          </div>
        </div>

        <div class="mt-7 overflow-hidden rounded-xl border border-black/10 bg-white">
          <table class="w-full text-left text-sm">
            <thead
              class="bg-black/[0.035] font-mono text-[10px] tracking-wide text-black/45 uppercase"
            >
              <tr>
                <th class="px-4 py-3">Тип звонка</th>
                <th class="px-4 py-3">Мин.</th>
                <th class="px-4 py-3">Расписание</th>
                <th class="px-4 py-3">Статус</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="callType in callTypes"
                :key="callType.id"
                class="cursor-pointer border-t border-black/10"
                :class="
                  selectedKind === 'type' && selectedId === callType.id
                    ? 'bg-orange-50'
                    : 'hover:bg-black/[0.02]'
                "
                @click="selectType(callType.id)"
              >
                <td class="px-4 py-4 font-semibold">{{ callType.name }}</td>
                <td class="px-4 py-4 font-mono">{{ callType.durationMinutes }}</td>
                <td class="px-4 py-4">
                  {{ schedules.find((schedule) => schedule.id === callType.scheduleId)?.name }}
                </td>
                <td class="px-4 py-4">
                  <span :class="callType.active ? 'text-emerald-700' : 'text-black/40'">
                    {{ callType.active ? '● Активен' : '○ Отключён' }}
                  </span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <h2 class="mt-8 font-mono text-xs tracking-wide text-black/45 uppercase">Расписания</h2>
        <div class="mt-3 grid gap-3 sm:grid-cols-2">
          <button
            v-for="schedule in schedules"
            :key="schedule.id"
            class="rounded-xl border border-black/10 bg-white p-4 text-left"
            :class="
              selectedKind === 'schedule' && selectedId === schedule.id
                ? 'ring-2 ring-[#ff5c35]'
                : ''
            "
            @click="selectSchedule(schedule.id)"
          >
            <strong>{{ schedule.name }}</strong>
            <span class="mt-1 block text-xs text-black/50">
              {{ callTypes.filter((item) => item.scheduleId === schedule.id).length }} типов ·
              {{ schedule.availability.filter((day) => day.intervals.length).length }} рабочих дней
            </span>
          </button>
        </div>

        <p
          v-if="message"
          class="mt-5 rounded-lg border border-red-300 bg-red-50 p-3 text-sm text-red-900"
        >
          {{ message }}
        </p>
      </section>

      <aside class="bg-white p-5 lg:p-7">
        <template v-if="selectedKind === 'type' && selectedType">
          <p class="font-mono text-xs text-black/40">ТИП ЗВОНКА / {{ selectedType.id }}</p>
          <h2 class="mt-2 text-xl font-bold">{{ selectedType.name }}</h2>
          <label class="mt-6 block text-xs font-bold uppercase">
            Название
            <input
              :value="selectedType.name"
              class="mt-2 w-full rounded-lg border border-black/20 px-3 py-2.5 text-sm font-normal normal-case"
            />
          </label>
          <div class="mt-4 grid grid-cols-2 gap-3">
            <label class="text-xs font-bold uppercase">
              Минуты
              <input
                :value="selectedType.durationMinutes"
                class="mt-2 w-full rounded-lg border border-black/20 px-3 py-2.5 text-sm font-normal"
              />
            </label>
            <label class="text-xs font-bold uppercase">
              Расписание
              <select
                :value="selectedType.scheduleId"
                class="mt-2 w-full rounded-lg border border-black/20 px-3 py-2.5 text-sm font-normal normal-case"
                @change="
                  emit(
                    'reassignType',
                    selectedType.id,
                    Number(($event.target as HTMLSelectElement).value),
                  )
                "
              >
                <option v-for="schedule in schedules" :key="schedule.id" :value="schedule.id">
                  {{ schedule.name }}
                </option>
              </select>
            </label>
          </div>
          <button
            class="mt-6 w-full rounded-lg bg-black px-4 py-2.5 text-sm font-semibold text-white"
          >
            Сохранить
          </button>
          <button
            class="mt-2 w-full rounded-lg border px-4 py-2.5 text-sm font-semibold"
            @click="emit('toggleType', selectedType.id)"
          >
            {{ selectedType.active ? 'Отключить' : 'Включить' }}
          </button>
          <button
            class="mt-2 w-full px-4 py-2 text-sm font-semibold text-red-700"
            @click="emit('deleteType', selectedType.id)"
          >
            Удалить
          </button>
        </template>

        <template v-else-if="selectedSchedule">
          <p class="font-mono text-xs text-black/40">РАСПИСАНИЕ / {{ selectedSchedule.id }}</p>
          <h2 class="mt-2 text-xl font-bold">{{ selectedSchedule.name }}</h2>
          <div class="mt-6 space-y-2">
            <div
              v-for="day in selectedSchedule.availability"
              :key="day.day"
              class="grid grid-cols-[2rem_1fr] gap-3 border-b py-2 text-sm"
            >
              <strong>{{ day.day }}</strong>
              <span>{{ day.intervals.join(', ') || 'Выходной' }}</span>
            </div>
          </div>
          <button
            class="mt-5 w-full rounded-lg border px-4 py-2.5 text-sm font-semibold"
            @click="emit('showValidation')"
          >
            Проверить интервалы
          </button>
          <button
            class="mt-2 w-full px-4 py-2 text-sm font-semibold text-red-700"
            @click="emit('deleteSchedule', selectedSchedule.id)"
          >
            Удалить расписание
          </button>
        </template>
      </aside>
    </main>
  </div>
</template>
