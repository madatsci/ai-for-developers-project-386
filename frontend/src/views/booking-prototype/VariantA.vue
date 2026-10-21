<script setup lang="ts">
import { RouterLink } from 'vue-router'
import type { BookingPrototypeState, CalendarDay, CallType } from './model'
import { formatSelectedDate } from './model'

defineProps<{
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
}>()

function updateVisitorName(event: Event) {
  emit('updateVisitorName', (event.target as HTMLInputElement).value)
}

function updateVisitorEmail(event: Event) {
  emit('updateVisitorEmail', (event.target as HTMLInputElement).value)
}
</script>

<template>
  <div class="min-h-svh bg-slate-50 pb-40 text-slate-950">
    <header class="border-b border-slate-200 bg-white">
      <div class="mx-auto flex max-w-6xl items-center justify-between px-5 py-5 sm:px-8">
        <RouterLink to="/" class="font-bold tracking-tight">Календарь звонков</RouterLink>
        <span class="text-sm text-slate-500">Время указано по Москве</span>
      </div>
    </header>

    <main v-if="!selectedType" class="mx-auto max-w-6xl px-5 py-12 sm:px-8 sm:py-20">
      <div class="max-w-2xl">
        <p class="text-sm font-semibold text-blue-700">Шаг 1 из 3</p>
        <h1 class="mt-3 text-4xl font-bold tracking-[-0.04em] sm:text-6xl">Бронирование звонка</h1>
        <p class="mt-5 text-lg leading-8 text-slate-600">
          Выберите формат. На следующем шаге покажем свободное время на 14 дней.
        </p>
      </div>

      <div class="mt-12 grid gap-5 lg:grid-cols-3">
        <button
          v-for="type in types"
          :key="type.slug"
          type="button"
          class="group flex min-h-64 flex-col rounded-3xl border border-slate-200 bg-white p-7 text-left shadow-sm transition hover:-translate-y-1 hover:border-blue-300 hover:shadow-xl focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-blue-600"
          @click="emit('selectType', type.slug)"
        >
          <span class="text-sm font-semibold text-blue-700">{{ type.durationMinutes }} минут</span>
          <span class="mt-5 text-2xl font-bold tracking-tight">{{ type.name }}</span>
          <span class="mt-3 leading-7 text-slate-600">{{ type.description }}</span>
          <span class="mt-auto pt-8 font-semibold text-blue-700">
            Выбрать <span aria-hidden="true">→</span>
          </span>
        </button>
      </div>
    </main>

    <main v-else class="mx-auto max-w-6xl px-5 py-8 sm:px-8 sm:py-12">
      <button
        v-if="state.stage !== 'confirmation'"
        type="button"
        class="text-sm font-semibold text-slate-600 hover:text-slate-950"
        @click="emit('backToTypes')"
      >
        ← Изменить тип звонка
      </button>

      <section
        v-if="state.stage === 'confirmation'"
        class="mx-auto mt-8 max-w-2xl rounded-[2rem] border border-emerald-200 bg-white p-7 text-center shadow-xl sm:p-12"
      >
        <div
          class="mx-auto grid size-16 place-items-center rounded-full bg-emerald-100 text-3xl text-emerald-700"
        >
          ✓
        </div>
        <p class="mt-7 text-sm font-semibold text-emerald-700">Бронирование создано</p>
        <h1 class="mt-3 text-3xl font-bold tracking-tight sm:text-5xl">До встречи на звонке</h1>
        <dl class="mt-8 divide-y divide-slate-200 rounded-2xl bg-slate-50 px-6 text-left">
          <div class="flex justify-between gap-6 py-4">
            <dt class="text-slate-500">Тип</dt>
            <dd class="text-right font-semibold">{{ selectedType.name }}</dd>
          </div>
          <div class="flex justify-between gap-6 py-4">
            <dt class="text-slate-500">Когда</dt>
            <dd class="text-right font-semibold">
              {{ formatSelectedDate(selectedDay, state.selectedTime) }} (МСК)
            </dd>
          </div>
          <div class="flex justify-between gap-6 py-4">
            <dt class="text-slate-500">Посетитель</dt>
            <dd class="text-right font-semibold">{{ state.visitorName }}</dd>
          </div>
          <div class="flex justify-between gap-6 py-4">
            <dt class="text-slate-500">Email</dt>
            <dd class="break-all text-right font-semibold">{{ state.visitorEmail }}</dd>
          </div>
        </dl>
        <p class="mt-6 text-sm leading-6 text-slate-500">
          Сохраните эти данные: отдельной страницы звонка и письма-подтверждения в прототипе нет.
        </p>
        <div class="mt-8 flex flex-col justify-center gap-3 sm:flex-row">
          <RouterLink
            to="/"
            class="rounded-xl border border-slate-300 px-5 py-3 font-semibold hover:bg-slate-50"
          >
            На главную
          </RouterLink>
          <button
            type="button"
            class="rounded-xl bg-blue-600 px-5 py-3 font-semibold text-white hover:bg-blue-700"
            @click="emit('restart')"
          >
            Забронировать ещё
          </button>
        </div>
      </section>

      <template v-else>
        <div class="mt-6 grid gap-8 lg:grid-cols-[minmax(0,1fr)_20rem] lg:items-start">
          <section class="rounded-[2rem] border border-slate-200 bg-white p-5 shadow-sm sm:p-8">
            <div class="flex flex-col justify-between gap-5 sm:flex-row sm:items-end">
              <div>
                <p class="text-sm font-semibold text-blue-700">Шаг 2 из 3</p>
                <h1 class="mt-2 text-3xl font-bold tracking-tight sm:text-4xl">Выберите время</h1>
              </div>
              <div class="sm:text-right">
                <p class="font-semibold">{{ selectedType.name }}</p>
                <p class="mt-1 text-sm text-slate-500">{{ selectedType.durationMinutes }} минут</p>
              </div>
            </div>

            <div class="mt-8 grid grid-cols-2 gap-2 sm:grid-cols-4 lg:grid-cols-7">
              <button
                v-for="day in dates"
                :key="day.key"
                type="button"
                :disabled="day.slots.length === 0"
                class="rounded-2xl border px-2 py-3 text-center transition disabled:cursor-not-allowed disabled:bg-slate-50 disabled:text-slate-400"
                :class="
                  state.selectedDateKey === day.key
                    ? 'border-blue-600 bg-blue-600 text-white'
                    : 'border-slate-200 bg-white hover:border-blue-400'
                "
                @click="emit('selectDate', day.key)"
              >
                <span class="block text-xs font-semibold uppercase">{{ day.dayName }}</span>
                <span class="mt-1 block font-bold">{{ day.dateLabel }}</span>
                <span class="mt-1 block text-[0.7rem] opacity-75">
                  {{ day.slots.length ? `${day.slots.length} слотов` : 'Нет мест' }}
                </span>
              </button>
            </div>

            <div class="mt-8 border-t border-slate-200 pt-7">
              <p class="text-sm font-semibold text-slate-600">
                {{ selectedDay?.relativeLabel || selectedDay?.dateLabel || 'Выберите дату' }}
              </p>
              <div v-if="selectedDay" class="mt-4 flex flex-wrap gap-3">
                <button
                  v-for="time in selectedDay.slots"
                  :key="time"
                  type="button"
                  class="min-w-24 rounded-xl border px-4 py-3 font-semibold transition"
                  :class="
                    state.selectedTime === time
                      ? 'border-blue-600 bg-blue-50 text-blue-800 ring-2 ring-blue-600'
                      : 'border-slate-300 hover:border-blue-500'
                  "
                  @click="emit('selectTime', time)"
                >
                  {{ time }}
                </button>
              </div>
            </div>
          </section>

          <aside
            class="rounded-[2rem] border border-slate-200 bg-white p-6 shadow-sm lg:sticky lg:top-6"
          >
            <p class="text-xs font-semibold tracking-widest text-slate-500 uppercase">Ваш выбор</p>
            <p class="mt-3 text-lg font-bold">{{ selectedType.name }}</p>
            <p class="mt-2 text-sm leading-6 text-slate-600">
              {{ formatSelectedDate(selectedDay, state.selectedTime) }}
            </p>
            <p class="mt-1 text-xs text-slate-500">Europe/Moscow · МСК</p>

            <form
              v-if="state.stage === 'details'"
              class="mt-7 border-t border-slate-200 pt-6"
              @submit.prevent="emit('submit')"
            >
              <p class="text-sm font-semibold text-blue-700">Шаг 3 из 3</p>
              <h2 class="mt-1 text-xl font-bold">Ваши данные</h2>

              <label class="mt-5 block text-sm font-semibold" for="variant-a-name">Имя</label>
              <input
                id="variant-a-name"
                :value="state.visitorName"
                type="text"
                autocomplete="name"
                class="mt-2 w-full rounded-xl border border-slate-300 px-4 py-3 focus:border-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-200"
                placeholder="Как к вам обращаться"
                @input="updateVisitorName"
              />
              <p v-if="state.errors.visitorName" class="mt-1 text-sm text-red-600">
                {{ state.errors.visitorName }}
              </p>

              <label class="mt-5 block text-sm font-semibold" for="variant-a-email">Email</label>
              <input
                id="variant-a-email"
                :value="state.visitorEmail"
                type="email"
                autocomplete="email"
                class="mt-2 w-full rounded-xl border border-slate-300 px-4 py-3 focus:border-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-200"
                placeholder="name@example.com"
                @input="updateVisitorEmail"
              />
              <p v-if="state.errors.visitorEmail" class="mt-1 text-sm text-red-600">
                {{ state.errors.visitorEmail }}
              </p>

              <button
                type="submit"
                class="mt-6 w-full rounded-xl bg-blue-600 px-4 py-3 font-semibold text-white hover:bg-blue-700"
              >
                Подтвердить бронирование
              </button>
            </form>

            <p v-else class="mt-6 border-t border-slate-200 pt-6 text-sm leading-6 text-slate-500">
              Сначала выберите дату и свободный слот.
            </p>
          </aside>
        </div>
      </template>
    </main>
  </div>
</template>
