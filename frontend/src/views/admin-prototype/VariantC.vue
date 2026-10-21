<script setup lang="ts">
import type { CallType, Schedule } from './model'

defineProps<{
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
</script>

<template>
  <div class="min-h-svh bg-[#111b18] pb-28 text-[#eef8f3]">
    <header class="border-b border-white/10 px-6 py-5">
      <div class="mx-auto flex max-w-[90rem] flex-wrap items-center justify-between gap-4">
        <div>
          <p class="text-xs font-bold tracking-[0.18em] text-[#86efac] uppercase">
            Доступность организатора
          </p>
          <h1 class="mt-1 text-2xl font-semibold">Доска расписаний</h1>
        </div>
        <div class="rounded-xl bg-white/5 px-4 py-2 text-sm text-white/70">
          Режим работы: <span class="font-semibold text-white">{{ workingHours }}</span>
        </div>
      </div>
    </header>

    <main class="mx-auto max-w-[90rem] px-6 py-8">
      <div class="mb-6 flex flex-wrap items-center justify-between gap-4">
        <p class="max-w-2xl text-sm leading-6 text-white/60">
          Расписание — главный объект. Внутри сразу видно недельную доступность и назначенные типы
          звонков.
        </p>
        <div class="flex gap-2">
          <button
            class="rounded-xl border border-white/20 px-4 py-2 text-sm font-semibold"
            @click="emit('addType')"
          >
            + Тип звонка
          </button>
          <button
            class="rounded-xl bg-[#86efac] px-4 py-2 text-sm font-bold text-[#102019]"
            @click="emit('addSchedule')"
          >
            + Расписание
          </button>
        </div>
      </div>

      <div class="grid gap-5 xl:grid-cols-2">
        <article
          v-for="schedule in schedules"
          :key="schedule.id"
          class="overflow-hidden rounded-3xl border border-white/10 bg-[#192723] shadow-2xl"
        >
          <div class="border-b border-white/10 p-6">
            <div class="flex items-start justify-between gap-4">
              <div>
                <p class="text-xs font-bold tracking-wider text-[#86efac] uppercase">Расписание</p>
                <h2 class="mt-1 text-2xl font-semibold">{{ schedule.name }}</h2>
              </div>
              <button class="text-sm text-rose-300" @click="emit('deleteSchedule', schedule.id)">
                Удалить
              </button>
            </div>
            <div class="mt-5 flex flex-wrap gap-2">
              <div
                v-for="callType in callTypes.filter((item) => item.scheduleId === schedule.id)"
                :key="callType.id"
                class="group rounded-xl bg-white/8 p-3"
              >
                <div class="flex items-center gap-2">
                  <span
                    class="size-2 rounded-full"
                    :class="callType.active ? 'bg-[#86efac]' : 'bg-white/20'"
                  />
                  <strong class="text-sm">{{ callType.name }}</strong>
                </div>
                <div class="mt-2 flex items-center gap-3 text-xs text-white/50">
                  <span>{{ callType.durationMinutes }} мин.</span>
                  <button class="text-white/70 underline" @click="emit('toggleType', callType.id)">
                    {{ callType.active ? 'Отключить' : 'Включить' }}
                  </button>
                  <select
                    :value="callType.scheduleId"
                    class="max-w-32 rounded bg-black/25 px-1 py-0.5 text-white"
                    aria-label="Перенести в расписание"
                    @change="
                      emit(
                        'reassignType',
                        callType.id,
                        Number(($event.target as HTMLSelectElement).value),
                      )
                    "
                  >
                    <option
                      v-for="target in schedules"
                      :key="target.id"
                      :value="target.id"
                      class="text-black"
                    >
                      {{ target.name }}
                    </option>
                  </select>
                </div>
              </div>
              <p
                v-if="!callTypes.some((item) => item.scheduleId === schedule.id)"
                class="text-sm text-white/40"
              >
                Типы звонков не назначены
              </p>
            </div>
          </div>

          <div class="p-6">
            <div class="grid grid-cols-7 gap-2">
              <div v-for="day in schedule.availability" :key="day.day" class="min-w-0">
                <p class="mb-2 text-center text-xs font-bold text-white/50">{{ day.day }}</p>
                <div class="min-h-28 rounded-xl bg-black/20 p-2 text-center">
                  <span
                    v-for="interval in day.intervals"
                    :key="interval"
                    class="mb-1 block rounded-lg bg-[#86efac]/15 px-1 py-2 text-[11px] font-semibold text-[#bbf7d0]"
                  >
                    {{ interval }}
                  </span>
                  <span v-if="day.intervals.length === 0" class="text-lg text-white/15">—</span>
                </div>
              </div>
            </div>
            <button
              class="mt-4 w-full rounded-xl border border-white/15 py-2.5 text-sm font-semibold"
              @click="emit('showValidation')"
            >
              Редактировать недельную доступность
            </button>
          </div>
        </article>
      </div>

      <div
        v-if="message"
        class="mt-6 rounded-2xl border border-amber-300/30 bg-amber-300/10 p-4 text-sm text-amber-100"
      >
        {{ message }}
      </div>
    </main>
  </div>
</template>
