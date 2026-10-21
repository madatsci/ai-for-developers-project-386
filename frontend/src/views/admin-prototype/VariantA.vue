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

const section = ref<'types' | 'schedules'>('types')
const selectedTypeId = ref(props.callTypes[0]?.id ?? 0)
const selectedType = computed(() =>
  props.callTypes.find((item) => item.id === selectedTypeId.value),
)
</script>

<template>
  <div class="min-h-svh bg-slate-50 pb-28 text-slate-950">
    <header class="border-b border-slate-200 bg-white">
      <div class="mx-auto flex max-w-7xl items-center justify-between px-6 py-5">
        <div>
          <p class="text-xs font-bold tracking-[0.18em] text-blue-700 uppercase">
            Календарь звонков
          </p>
          <h1 class="mt-1 text-xl font-bold">Настройки бронирования</h1>
        </div>
        <span class="rounded-full bg-emerald-50 px-3 py-1.5 text-xs font-semibold text-emerald-700">
          Europe/Moscow
        </span>
      </div>
    </header>

    <main class="mx-auto max-w-7xl px-6 py-8">
      <div class="mb-8 rounded-2xl border border-blue-200 bg-blue-50 p-5">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div>
            <p class="text-sm font-bold text-blue-950">Режим работы</p>
            <p class="mt-1 text-sm text-blue-800">{{ workingHours }}</p>
          </div>
          <button class="text-sm font-semibold text-blue-800 underline underline-offset-4">
            Изменить границы
          </button>
        </div>
      </div>

      <nav class="mb-6 flex gap-2 border-b border-slate-200" aria-label="Разделы настроек">
        <button
          v-for="item in [
            { key: 'types', label: 'Типы звонков' },
            { key: 'schedules', label: 'Расписания' },
          ] as const"
          :key="item.key"
          type="button"
          class="border-b-2 px-4 py-3 text-sm font-semibold"
          :class="
            section === item.key
              ? 'border-blue-600 text-blue-700'
              : 'border-transparent text-slate-500'
          "
          @click="section = item.key"
        >
          {{ item.label }}
        </button>
      </nav>

      <section v-if="section === 'types'" class="grid gap-6 lg:grid-cols-[1fr_23rem]">
        <div>
          <div class="mb-4 flex items-end justify-between">
            <div>
              <h2 class="text-2xl font-bold">Типы звонков</h2>
              <p class="mt-1 text-sm text-slate-500">
                Публичные варианты, которые видит посетитель
              </p>
            </div>
            <button
              class="rounded-xl bg-blue-600 px-4 py-2.5 text-sm font-semibold text-white"
              @click="emit('addType')"
            >
              + Новый тип
            </button>
          </div>
          <div class="grid gap-3 sm:grid-cols-2">
            <button
              v-for="callType in callTypes"
              :key="callType.id"
              type="button"
              class="rounded-2xl border bg-white p-5 text-left shadow-sm"
              :class="
                selectedTypeId === callType.id
                  ? 'border-blue-500 ring-2 ring-blue-100'
                  : 'border-slate-200'
              "
              @click="selectedTypeId = callType.id"
            >
              <span
                class="rounded-full px-2.5 py-1 text-xs font-semibold"
                :class="
                  callType.active ? 'bg-emerald-50 text-emerald-700' : 'bg-slate-100 text-slate-500'
                "
              >
                {{ callType.active ? 'Активен' : 'Отключён' }}
              </span>
              <strong class="mt-4 block">{{ callType.name }}</strong>
              <span class="mt-2 block text-sm text-slate-500"
                >{{ callType.durationMinutes }} минут</span
              >
            </button>
          </div>
        </div>

        <aside
          v-if="selectedType"
          class="h-fit rounded-2xl border border-slate-200 bg-white p-6 shadow-sm"
        >
          <p class="text-xs font-bold tracking-wider text-slate-400 uppercase">Редактирование</p>
          <label class="mt-5 block text-sm font-semibold">
            Название
            <input
              :value="selectedType.name"
              class="mt-2 w-full rounded-xl border px-3 py-2.5 font-normal"
            />
          </label>
          <label class="mt-4 block text-sm font-semibold">
            Длительность
            <select class="mt-2 w-full rounded-xl border px-3 py-2.5 font-normal">
              <option>{{ selectedType.durationMinutes }} минут</option>
              <option>45 минут</option>
            </select>
          </label>
          <label class="mt-4 block text-sm font-semibold">
            Расписание
            <select
              :value="selectedType.scheduleId"
              class="mt-2 w-full rounded-xl border px-3 py-2.5 font-normal"
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
          <div class="mt-6 grid gap-2">
            <button class="rounded-xl bg-slate-950 px-4 py-2.5 text-sm font-semibold text-white">
              Сохранить
            </button>
            <button
              class="rounded-xl border px-4 py-2.5 text-sm font-semibold"
              @click="emit('toggleType', selectedType.id)"
            >
              {{ selectedType.active ? 'Отключить' : 'Включить' }}
            </button>
            <button
              class="px-4 py-2 text-sm font-semibold text-rose-700"
              @click="emit('deleteType', selectedType.id)"
            >
              Удалить
            </button>
          </div>
        </aside>
      </section>

      <section v-else>
        <div class="mb-5 flex items-end justify-between">
          <div>
            <h2 class="text-2xl font-bold">Расписания</h2>
            <p class="mt-1 text-sm text-slate-500">Недельная доступность для типов звонков</p>
          </div>
          <button
            class="rounded-xl bg-blue-600 px-4 py-2.5 text-sm font-semibold text-white"
            @click="emit('addSchedule')"
          >
            + Новое расписание
          </button>
        </div>
        <div class="grid gap-4 lg:grid-cols-2">
          <article
            v-for="schedule in schedules"
            :key="schedule.id"
            class="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm"
          >
            <div class="flex items-start justify-between">
              <div>
                <h3 class="font-bold">{{ schedule.name }}</h3>
                <p class="mt-1 text-xs text-slate-500">
                  {{ callTypes.filter((item) => item.scheduleId === schedule.id).length }} типов
                  звонков
                </p>
              </div>
              <button
                class="text-sm font-semibold text-rose-700"
                @click="emit('deleteSchedule', schedule.id)"
              >
                Удалить
              </button>
            </div>
            <dl class="mt-5 grid grid-cols-[2rem_1fr] gap-x-3 gap-y-2 text-sm">
              <template v-for="day in schedule.availability" :key="day.day">
                <dt class="font-semibold text-slate-500">{{ day.day }}</dt>
                <dd>{{ day.intervals.join(', ') || 'Выходной' }}</dd>
              </template>
            </dl>
          </article>
        </div>
      </section>

      <div
        v-if="message"
        class="mt-6 rounded-xl border border-amber-300 bg-amber-50 p-4 text-sm text-amber-950"
      >
        {{ message }}
      </div>
      <button
        class="mt-4 text-sm font-semibold text-rose-700 underline"
        @click="emit('showValidation')"
      >
        Показать пример ошибки пересечения интервалов
      </button>
    </main>
  </div>
</template>
