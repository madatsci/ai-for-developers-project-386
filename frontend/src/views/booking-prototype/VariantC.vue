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
  backToSlot: []
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
  <div class="min-h-svh bg-[#f4f0e8] pb-40 text-[#17251c]">
    <header class="mx-auto flex max-w-5xl items-center justify-between px-5 py-6 sm:px-8 sm:py-9">
      <RouterLink to="/" class="font-bold tracking-tight">Календарь звонков</RouterLink>
      <span class="rounded-full border border-[#17251c]/20 px-3 py-1 text-xs font-semibold"
        >МСК</span
      >
    </header>

    <main class="mx-auto max-w-5xl px-5 pb-16 sm:px-8">
      <section v-if="!selectedType" class="mx-auto max-w-3xl pt-8 sm:pt-16">
        <p class="text-sm font-bold text-[#af4b2a]">1 / 3</p>
        <h1 class="mt-5 text-5xl leading-[0.96] font-bold tracking-[-0.055em] sm:text-7xl">
          О чём успеем поговорить?
        </h1>
        <p class="mt-7 max-w-xl text-lg leading-8 text-[#526158]">
          Выберите длительность — следующий экран покажет только подходящие свободные слоты.
        </p>

        <div class="mt-12 space-y-3">
          <button
            v-for="type in types"
            :key="type.slug"
            type="button"
            class="group grid w-full grid-cols-[auto_1fr_auto] items-center gap-4 rounded-[1.5rem] border border-[#17251c]/15 bg-[#fffdf7] p-4 text-left transition hover:border-[#17251c] hover:shadow-[5px_5px_0_#17251c] sm:gap-6 sm:p-6"
            @click="emit('selectType', type.slug)"
          >
            <span
              class="grid size-12 place-items-center rounded-full bg-[#d9e8d6] text-sm font-bold sm:size-16"
            >
              {{ type.durationMinutes }}
            </span>
            <span>
              <span class="block text-xl font-bold sm:text-2xl">{{ type.name }}</span>
              <span class="mt-1 block text-sm text-[#526158] sm:text-base">{{
                type.description
              }}</span>
            </span>
            <span class="text-2xl transition group-hover:translate-x-1" aria-hidden="true">→</span>
          </button>
        </div>
      </section>

      <section v-else-if="state.stage === 'slot'" class="pt-5 sm:pt-12">
        <button type="button" class="text-sm font-bold text-[#526158]" @click="emit('backToTypes')">
          ← {{ selectedType.name }}
        </button>
        <div class="mt-8 grid gap-10 lg:grid-cols-[minmax(0,1fr)_17rem] lg:items-start">
          <div>
            <p class="text-sm font-bold text-[#af4b2a]">2 / 3</p>
            <h1 class="mt-4 text-5xl leading-none font-bold tracking-[-0.05em] sm:text-7xl">
              Когда вам удобно?
            </h1>
            <p class="mt-5 text-[#526158]">14 дней, время организатора — Europe/Moscow.</p>

            <div class="-mx-5 mt-10 flex snap-x gap-2 overflow-x-auto px-5 pb-3 sm:mx-0 sm:px-0">
              <button
                v-for="day in dates"
                :key="day.key"
                type="button"
                :disabled="day.slots.length === 0"
                class="min-w-24 snap-start rounded-2xl border px-3 py-4 text-center transition disabled:cursor-not-allowed disabled:opacity-35"
                :class="
                  state.selectedDateKey === day.key
                    ? 'border-[#17251c] bg-[#17251c] text-white'
                    : 'border-[#17251c]/20 bg-[#fffdf7] hover:border-[#17251c]'
                "
                @click="emit('selectDate', day.key)"
              >
                <span class="block text-xs font-bold uppercase">{{
                  day.relativeLabel || day.dayName
                }}</span>
                <span class="mt-2 block font-bold">{{ day.dateLabel }}</span>
                <span class="mt-1 block text-[0.7rem] opacity-70">
                  {{ day.slots.length ? day.slots.length : 'нет' }}
                </span>
              </button>
            </div>

            <div v-if="selectedDay" class="mt-8 grid grid-cols-2 gap-3 sm:grid-cols-3">
              <button
                v-for="time in selectedDay.slots"
                :key="time"
                type="button"
                class="rounded-2xl border border-[#17251c]/20 bg-[#fffdf7] px-5 py-5 text-xl font-bold transition hover:-translate-y-0.5 hover:border-[#17251c] hover:shadow-[4px_4px_0_#17251c]"
                @click="emit('selectTime', time)"
              >
                {{ time }}
              </button>
            </div>
          </div>

          <aside class="rounded-[2rem] bg-[#d9e8d6] p-6 lg:sticky lg:top-8">
            <p class="text-xs font-bold tracking-widest uppercase">Вы выбрали</p>
            <p class="mt-4 text-2xl font-bold">{{ selectedType.name }}</p>
            <p class="mt-3 leading-7 text-[#526158]">{{ selectedType.description }}</p>
            <p class="mt-7 border-t border-[#17251c]/15 pt-5 text-sm font-semibold">
              Слот не резервируется, пока бронирование не создано.
            </p>
          </aside>
        </div>
      </section>

      <section v-else-if="state.stage === 'details'" class="mx-auto max-w-3xl pt-5 sm:pt-12">
        <button type="button" class="text-sm font-bold text-[#526158]" @click="emit('backToSlot')">
          ← Выбрать другое время
        </button>
        <p class="mt-10 text-sm font-bold text-[#af4b2a]">3 / 3</p>
        <h1 class="mt-4 text-5xl leading-none font-bold tracking-[-0.05em] sm:text-7xl">
          Как к вам обращаться?
        </h1>
        <div class="mt-8 rounded-2xl bg-[#d9e8d6] px-5 py-4 font-semibold">
          {{ selectedType.name }} · {{ formatSelectedDate(selectedDay, state.selectedTime) }} МСК
        </div>

        <form class="mt-10 space-y-8" @submit.prevent="emit('submit')">
          <div>
            <label class="block text-sm font-bold" for="variant-c-name">Ваше имя</label>
            <input
              id="variant-c-name"
              :value="state.visitorName"
              type="text"
              autocomplete="name"
              class="mt-3 w-full border-0 border-b-2 border-[#17251c]/25 bg-transparent px-0 py-4 text-2xl font-semibold placeholder:text-[#526158]/40 focus:border-[#af4b2a] focus:outline-none"
              placeholder="Например, Анна"
              @input="updateVisitorName"
            />
            <p v-if="state.errors.visitorName" class="mt-2 text-sm font-semibold text-red-700">
              {{ state.errors.visitorName }}
            </p>
          </div>

          <div>
            <label class="block text-sm font-bold" for="variant-c-email">Email</label>
            <input
              id="variant-c-email"
              :value="state.visitorEmail"
              type="email"
              autocomplete="email"
              class="mt-3 w-full border-0 border-b-2 border-[#17251c]/25 bg-transparent px-0 py-4 text-2xl font-semibold placeholder:text-[#526158]/40 focus:border-[#af4b2a] focus:outline-none"
              placeholder="name@example.com"
              @input="updateVisitorEmail"
            />
            <p v-if="state.errors.visitorEmail" class="mt-2 text-sm font-semibold text-red-700">
              {{ state.errors.visitorEmail }}
            </p>
          </div>

          <button
            type="submit"
            class="w-full rounded-full bg-[#af4b2a] px-8 py-5 text-lg font-bold text-white transition hover:bg-[#8d3c22] sm:w-auto"
          >
            Подтвердить бронирование →
          </button>
        </form>
      </section>

      <section v-else class="mx-auto max-w-3xl pt-8 text-center sm:pt-20">
        <div class="mx-auto grid size-24 place-items-center rounded-full bg-[#d9e8d6] text-5xl">
          ✓
        </div>
        <p class="mt-8 text-sm font-bold text-[#af4b2a]">Готово</p>
        <h1 class="mt-4 text-5xl leading-none font-bold tracking-[-0.05em] sm:text-7xl">
          Время за вами
        </h1>
        <p class="mx-auto mt-6 max-w-xl text-lg leading-8 text-[#526158]">
          Звонок создан сразу. Сохраните детали — письмо-подтверждение не отправляется.
        </p>
        <div
          class="mt-10 rounded-[2rem] border border-[#17251c]/15 bg-[#fffdf7] p-7 text-left sm:p-9"
        >
          <p class="text-2xl font-bold">{{ selectedType.name }}</p>
          <p class="mt-3 text-xl">{{ formatSelectedDate(selectedDay, state.selectedTime) }} МСК</p>
          <div class="mt-7 grid gap-5 border-t border-[#17251c]/15 pt-6 sm:grid-cols-2">
            <div>
              <p class="text-xs font-bold tracking-widest text-[#526158] uppercase">Посетитель</p>
              <p class="mt-2 font-semibold">{{ state.visitorName }}</p>
            </div>
            <div>
              <p class="text-xs font-bold tracking-widest text-[#526158] uppercase">Email</p>
              <p class="mt-2 break-all font-semibold">{{ state.visitorEmail }}</p>
            </div>
          </div>
        </div>
        <div class="mt-9 flex flex-col justify-center gap-3 sm:flex-row">
          <RouterLink
            to="/"
            class="rounded-full border border-[#17251c] px-6 py-3 font-bold hover:bg-[#fffdf7]"
          >
            На главную
          </RouterLink>
          <button
            type="button"
            class="rounded-full bg-[#17251c] px-6 py-3 font-bold text-white hover:bg-[#263b2d]"
            @click="emit('restart')"
          >
            Забронировать ещё
          </button>
        </div>
      </section>
    </main>
  </div>
</template>
