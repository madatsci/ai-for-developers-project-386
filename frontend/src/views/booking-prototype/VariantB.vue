<script setup lang="ts">
import { computed, nextTick, ref } from 'vue'
import { RouterLink } from 'vue-router'
import type { BookingPrototypeState, CalendarDay, CallType } from './model'
import { formatSelectedDate } from './model'

const props = defineProps<{
  types: CallType[]
  dates: CalendarDay[]
  selectedType?: CallType
  selectedDay?: CalendarDay
  state: BookingPrototypeState
}>()

const emit = defineEmits<{
  selectType: [slug: string]
  selectDate: [key: string]
  selectTime: [time: string]
  backToTypes: []
  submit: []
  restart: []
  updateVisitorName: [value: string]
  updateVisitorEmail: [value: string]
  updateVisitorNotes: [value: string]
}>()

const detailsHeading = ref<HTMLElement | null>(null)
const hasAvailableSlots = computed(() => props.dates.some((day) => day.slots.length > 0))

async function selectSlot(dayKey: string, time: string) {
  emit('selectDate', dayKey)
  emit('selectTime', time)

  if (
    typeof window.matchMedia !== 'function' ||
    !window.matchMedia('(max-width: 1023px)').matches
  ) {
    return
  }

  await nextTick()
  detailsHeading.value?.focus({ preventScroll: true })
  detailsHeading.value?.scrollIntoView?.({ behavior: 'smooth', block: 'start' })
}

function updateVisitorName(event: Event) {
  emit('updateVisitorName', (event.target as HTMLInputElement).value)
}

function updateVisitorEmail(event: Event) {
  emit('updateVisitorEmail', (event.target as HTMLInputElement).value)
}

function updateVisitorNotes(event: Event) {
  emit('updateVisitorNotes', (event.target as HTMLTextAreaElement).value)
}
</script>

<template>
  <div class="min-h-svh bg-slate-100 pb-40 text-slate-950">
    <main v-if="!selectedType" class="grid min-h-svh lg:grid-cols-[22rem_1fr]">
      <aside class="flex flex-col bg-slate-950 p-8 text-white sm:p-12">
        <RouterLink to="/" class="text-lg font-bold tracking-tight">Календарь звонков</RouterLink>
        <div class="my-auto py-16">
          <p class="text-sm font-semibold text-cyan-300">Онлайн-бронирование</p>
          <h1 class="mt-4 text-4xl font-bold tracking-[-0.04em]">Бронирование звонка</h1>
          <p class="mt-5 leading-7 text-slate-300">
            Все доступные форматы в одном списке. Выберите подходящую длительность.
          </p>
        </div>
        <p class="text-sm text-slate-400">Europe/Moscow · без пересчёта часового пояса</p>
      </aside>

      <section class="flex items-center px-5 py-12 sm:px-12 lg:px-20">
        <div class="w-full max-w-4xl">
          <p class="text-xs font-bold tracking-[0.2em] text-slate-500 uppercase">
            Активные типы звонков
          </p>
          <div v-if="types.length" class="mt-6 divide-y divide-slate-200 border-y border-slate-300">
            <button
              v-for="type in types"
              :key="type.slug"
              type="button"
              class="group grid w-full gap-4 py-7 text-left transition hover:px-4 hover:bg-white sm:grid-cols-[1fr_auto] sm:items-center"
              @click="emit('selectType', type.slug)"
            >
              <span>
                <span class="block text-2xl font-bold tracking-tight">{{ type.name }}</span>
                <span class="mt-2 block text-slate-600">{{ type.description }}</span>
              </span>
              <span class="flex items-center gap-5">
                <span class="font-mono text-sm text-slate-500">{{ type.durationMinutes }} мин</span>
                <span
                  class="grid size-11 place-items-center rounded-full bg-slate-950 text-white transition group-hover:bg-cyan-600"
                  aria-hidden="true"
                >
                  →
                </span>
              </span>
            </button>
          </div>
          <div v-else class="mt-6 border-y border-slate-300 py-10">
            <h2 class="text-2xl font-bold">Сейчас нет доступных типов звонков</h2>
            <p class="mt-3 text-slate-600">Попробуйте вернуться позже.</p>
          </div>
        </div>
      </section>
    </main>

    <main v-else class="min-h-svh lg:grid lg:grid-cols-[17rem_minmax(0,1fr)_22rem]">
      <aside class="bg-slate-950 p-6 text-white lg:sticky lg:top-0 lg:h-svh lg:p-8">
        <RouterLink to="/" class="font-bold tracking-tight">Календарь звонков</RouterLink>
        <button
          v-if="state.stage !== 'confirmation'"
          type="button"
          class="mt-10 block text-sm font-semibold text-slate-300 hover:text-white"
          @click="emit('backToTypes')"
        >
          ← Все типы
        </button>
        <div class="mt-8 border-l-2 border-cyan-400 pl-5">
          <p class="text-xs font-bold tracking-widest text-cyan-300 uppercase">Вы выбрали</p>
          <h1 class="mt-3 text-2xl font-bold leading-tight">{{ selectedType.name }}</h1>
          <p class="mt-3 text-sm leading-6 text-slate-300">{{ selectedType.description }}</p>
          <p class="mt-5 font-mono text-sm text-slate-400">
            {{ selectedType.durationMinutes }} минут
          </p>
        </div>

        <ol class="mt-12 space-y-5 text-sm">
          <li
            class="flex items-center gap-3"
            :class="state.selectedTime ? 'text-white' : 'text-cyan-300'"
          >
            <span class="grid size-7 place-items-center rounded-full border border-current">1</span>
            Время
          </li>
          <li
            class="flex items-center gap-3"
            :class="state.stage === 'details' ? 'text-cyan-300' : 'text-slate-500'"
          >
            <span class="grid size-7 place-items-center rounded-full border border-current">2</span>
            Данные
          </li>
          <li
            class="flex items-center gap-3"
            :class="state.stage === 'confirmation' ? 'text-cyan-300' : 'text-slate-500'"
          >
            <span class="grid size-7 place-items-center rounded-full border border-current">3</span>
            Готово
          </li>
        </ol>
      </aside>

      <section v-if="state.stage !== 'confirmation'" class="p-5 sm:p-8 lg:p-10">
        <div class="flex flex-col justify-between gap-3 sm:flex-row sm:items-end">
          <div>
            <p class="text-xs font-bold tracking-[0.2em] text-slate-500 uppercase">
              Ближайшие 14 дней
            </p>
            <h2 class="mt-2 text-3xl font-bold tracking-tight">Свободное время</h2>
          </div>
          <p class="text-sm text-slate-500">Московское время</p>
        </div>

        <div
          v-if="!hasAvailableSlots"
          class="mt-7 rounded-2xl border border-amber-300 bg-amber-50 p-5"
          role="status"
        >
          <p class="font-bold">В ближайшие 14 дней свободного времени нет</p>
          <p class="mt-2 text-sm leading-6 text-slate-600">
            Можно выбрать другой тип звонка или вернуться позже.
          </p>
          <button
            type="button"
            class="mt-4 text-sm font-semibold text-cyan-700 hover:text-cyan-900"
            @click="emit('backToTypes')"
          >
            Изменить тип звонка
          </button>
        </div>

        <div class="mt-7 overflow-hidden rounded-2xl border border-slate-300 bg-white">
          <div
            v-for="day in dates"
            :key="day.key"
            class="grid border-b border-slate-200 last:border-b-0 sm:grid-cols-[9rem_1fr]"
            :class="state.selectedDateKey === day.key ? 'bg-cyan-50' : ''"
          >
            <div
              class="border-b border-slate-200 px-5 py-4 text-left sm:border-r sm:border-b-0"
              :class="day.slots.length === 0 ? 'text-slate-400' : ''"
            >
              <span class="block text-xs font-bold uppercase">{{ day.dayName }}</span>
              <span class="mt-1 block font-semibold">{{ day.relativeLabel || day.dateLabel }}</span>
              <span v-if="day.relativeLabel" class="mt-1 block text-xs text-slate-500">
                {{ day.dateLabel }}
              </span>
            </div>
            <div class="flex min-h-20 flex-wrap items-center gap-2 px-4 py-3">
              <template v-if="day.slots.length">
                <button
                  v-for="time in day.slots"
                  :key="time"
                  type="button"
                  :aria-label="`${day.dateLabel}, ${time}`"
                  :aria-pressed="state.selectedDateKey === day.key && state.selectedTime === time"
                  class="rounded-lg border px-3 py-2 font-mono text-sm font-semibold transition"
                  :class="
                    state.selectedDateKey === day.key && state.selectedTime === time
                      ? 'border-slate-950 bg-slate-950 text-white'
                      : 'border-slate-300 bg-white hover:border-cyan-600 hover:text-cyan-700'
                  "
                  @click="selectSlot(day.key, time)"
                >
                  {{ time }}
                </button>
              </template>
              <span v-else class="text-sm text-slate-400">Свободных слотов нет</span>
            </div>
          </div>
        </div>
      </section>

      <aside
        v-if="state.stage !== 'confirmation'"
        class="border-t border-slate-300 bg-white p-6 lg:sticky lg:top-0 lg:h-svh lg:overflow-y-auto lg:border-t-0 lg:border-l lg:p-8"
      >
        <p class="text-xs font-bold tracking-[0.2em] text-slate-500 uppercase">Оформление</p>
        <h2 ref="detailsHeading" tabindex="-1" class="mt-3 text-2xl font-bold outline-none">
          Ваши данные
        </h2>
        <p class="mt-3 min-h-12 text-sm leading-6 text-slate-600">
          {{ formatSelectedDate(selectedDay, state.selectedTime) }}
        </p>

        <form class="mt-7" @submit.prevent="emit('submit')">
          <label class="block text-sm font-semibold" for="variant-b-name">
            Имя <span aria-hidden="true">*</span>
          </label>
          <input
            id="variant-b-name"
            :value="state.visitorName"
            type="text"
            autocomplete="name"
            class="mt-2 w-full border-b-2 border-slate-300 bg-transparent px-0 py-3 focus:border-cyan-600 focus:outline-none"
            placeholder="Иван"
            @input="updateVisitorName"
          />
          <p v-if="state.errors.visitorName" class="mt-1 text-sm text-red-600">
            {{ state.errors.visitorName }}
          </p>

          <label class="mt-7 block text-sm font-semibold" for="variant-b-email">
            Email <span aria-hidden="true">*</span>
          </label>
          <input
            id="variant-b-email"
            :value="state.visitorEmail"
            type="email"
            autocomplete="email"
            class="mt-2 w-full border-b-2 border-slate-300 bg-transparent px-0 py-3 focus:border-cyan-600 focus:outline-none"
            placeholder="name@example.com"
            @input="updateVisitorEmail"
          />
          <p v-if="state.errors.visitorEmail" class="mt-1 text-sm text-red-600">
            {{ state.errors.visitorEmail }}
          </p>

          <div class="mt-7 flex items-baseline justify-between gap-3">
            <label class="block text-sm font-semibold" for="variant-b-notes">
              Дополнительная информация
              <span class="font-normal text-slate-500">(необязательно)</span>
            </label>
            <span class="shrink-0 font-mono text-xs text-slate-400">
              {{ state.visitorNotes.length }}/1000
            </span>
          </div>
          <textarea
            id="variant-b-notes"
            :value="state.visitorNotes"
            maxlength="1000"
            rows="4"
            class="mt-2 w-full resize-y rounded-xl border-2 border-slate-300 bg-transparent px-3 py-3 leading-6 focus:border-cyan-600 focus:outline-none"
            placeholder="Расскажите всё, что поможет подготовиться к звонку"
            @input="updateVisitorNotes"
          />
          <p v-if="state.errors.visitorNotes" class="mt-1 text-sm text-red-600">
            {{ state.errors.visitorNotes }}
          </p>

          <button
            type="submit"
            :disabled="!state.selectedTime"
            class="mt-9 w-full rounded-xl bg-cyan-600 px-4 py-3 font-semibold text-white transition hover:bg-cyan-700 disabled:cursor-not-allowed disabled:bg-slate-300"
          >
            Забронировать звонок
          </button>
          <p class="mt-4 text-xs leading-5 text-slate-500">
            Нажимая кнопку, вы сразу создаёте звонок. Дополнительного подтверждения нет.
          </p>
        </form>
      </aside>

      <section
        v-else
        class="col-span-2 flex min-h-svh items-center justify-center bg-cyan-50 p-5 sm:p-10"
      >
        <div
          class="w-full max-w-3xl border border-slate-950 bg-white p-7 shadow-[14px_14px_0_#0f172a] sm:p-12"
        >
          <p class="font-mono text-sm font-bold text-cyan-700">201 CREATED</p>
          <h1 class="mt-4 text-4xl font-bold tracking-[-0.04em] sm:text-6xl">
            Звонок забронирован
          </h1>
          <p class="mt-4 leading-7 text-slate-600">
            Сохраните детали: письмо с подтверждением не отправляется.
          </p>
          <div class="mt-10 grid gap-6 border-y border-slate-300 py-8 sm:grid-cols-2">
            <div>
              <p class="text-xs font-bold tracking-widest text-slate-500 uppercase">Когда</p>
              <p class="mt-2 text-xl font-bold">
                {{ formatSelectedDate(selectedDay, state.selectedTime) }}
              </p>
              <p class="mt-1 text-sm text-slate-500">Europe/Moscow</p>
            </div>
            <div>
              <p class="text-xs font-bold tracking-widest text-slate-500 uppercase">Что</p>
              <p class="mt-2 text-xl font-bold">{{ selectedType.name }}</p>
              <p class="mt-1 text-sm text-slate-500">{{ selectedType.durationMinutes }} минут</p>
            </div>
            <div>
              <p class="text-xs font-bold tracking-widest text-slate-500 uppercase">Имя</p>
              <p class="mt-2 font-semibold">{{ state.visitorName }}</p>
            </div>
            <div>
              <p class="text-xs font-bold tracking-widest text-slate-500 uppercase">Email</p>
              <p class="mt-2 break-all font-semibold">{{ state.visitorEmail }}</p>
            </div>
            <div v-if="state.visitorNotes" class="sm:col-span-2">
              <p class="text-xs font-bold tracking-widest text-slate-500 uppercase">
                Дополнительная информация
              </p>
              <p class="mt-2 whitespace-pre-wrap">{{ state.visitorNotes }}</p>
            </div>
          </div>
          <div class="mt-9 flex flex-col gap-3 sm:flex-row">
            <RouterLink
              to="/"
              class="border border-slate-950 px-6 py-3 text-center font-semibold hover:bg-slate-100"
            >
              На главную
            </RouterLink>
            <button
              type="button"
              class="bg-slate-950 px-6 py-3 font-semibold text-white hover:bg-slate-800"
              @click="emit('restart')"
            >
              Забронировать ещё
            </button>
          </div>
        </div>
      </section>
    </main>
  </div>
</template>
