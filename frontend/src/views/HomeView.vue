<script setup lang="ts">
import { RouterLink } from 'vue-router'
</script>

<template>
  <div class="min-h-svh bg-white text-slate-950">
    <header class="border-b border-slate-200">
      <div class="mx-auto flex max-w-7xl items-center justify-between gap-4 px-6 py-5 lg:px-8">
        <span class="text-sm font-bold tracking-tight sm:text-lg">Календарь звонков</span>
        <RouterLink
          to="/booking"
          class="shrink-0 rounded-xl bg-blue-600 px-3 py-2 text-xs font-semibold text-white shadow-sm transition-colors hover:bg-blue-700 focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-blue-600 sm:px-5 sm:py-3 sm:text-sm"
        >
          Записаться на звонок
        </RouterLink>
      </div>
    </header>

    <main>
      <section
        class="mx-auto grid max-w-7xl gap-14 px-6 py-16 lg:grid-cols-[1fr_1.05fr] lg:items-center lg:px-8 lg:py-28"
      >
        <div class="max-w-2xl">
          <h1
            class="text-4xl leading-[1.06] font-bold tracking-[-0.04em] text-slate-950 sm:text-5xl lg:text-6xl xl:text-7xl"
          >
            Удобное время для звонка — без долгих переписок
          </h1>
          <p class="mt-7 max-w-xl text-lg leading-8 text-slate-600 sm:text-xl">
            Выберите свободный слот в календаре и забронируйте звонок за пару минут
          </p>
          <RouterLink
            to="/booking"
            class="mt-9 inline-flex w-full items-center justify-center rounded-xl bg-blue-600 px-6 py-3.5 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-blue-700 focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-blue-600 sm:w-auto sm:min-w-64 sm:text-base"
          >
            Записаться на звонок
          </RouterLink>
        </div>

        <div
          aria-hidden="true"
          class="overflow-hidden rounded-[1.25rem] border border-slate-200 bg-white p-5 shadow-[0_18px_55px_-32px_rgba(15,23,42,0.35)] sm:p-8"
        >
          <p class="text-xl font-bold tracking-tight text-slate-950 sm:text-2xl">
            Свободное время на этой неделе
          </p>
          <div class="mt-7 grid grid-cols-3 divide-x divide-slate-200">
            <div class="px-2 text-center sm:px-4">
              <p class="text-sm font-semibold text-slate-950 sm:text-base">Пн</p>
              <div class="mt-4 space-y-3">
                <span
                  class="block rounded-lg bg-blue-600 px-2 py-3 text-sm font-semibold text-white"
                  >10:00</span
                >
                <span class="block rounded-lg bg-slate-50 px-2 py-3 text-sm text-slate-700"
                  >12:00</span
                >
                <span class="block rounded-lg bg-slate-50 px-2 py-3 text-sm text-slate-700"
                  >14:30</span
                >
              </div>
            </div>
            <div class="px-2 text-center sm:px-4">
              <p class="text-sm font-semibold text-slate-950 sm:text-base">Вт</p>
              <div class="mt-4 space-y-3">
                <span class="block rounded-lg bg-slate-50 px-2 py-3 text-sm text-slate-700"
                  >09:30</span
                >
                <span class="block rounded-lg bg-slate-50 px-2 py-3 text-sm text-slate-700"
                  >11:00</span
                >
                <span class="block rounded-lg bg-slate-50 px-2 py-3 text-sm text-slate-700"
                  >16:00</span
                >
              </div>
            </div>
            <div class="px-2 text-center sm:px-4">
              <p class="text-sm font-semibold text-slate-950 sm:text-base">Ср</p>
              <div class="mt-4 space-y-3">
                <span class="block rounded-lg bg-slate-50 px-2 py-3 text-sm text-slate-700"
                  >10:00</span
                >
                <span class="block rounded-lg bg-slate-50 px-2 py-3 text-sm text-slate-700"
                  >13:00</span
                >
                <span class="block rounded-lg bg-slate-50 px-2 py-3 text-sm text-slate-700"
                  >15:30</span
                >
              </div>
            </div>
          </div>
        </div>
      </section>

      <section class="border-y border-slate-200 bg-white">
        <div class="mx-auto max-w-7xl px-6 py-20 lg:px-8 lg:py-28">
          <h2 class="text-3xl font-bold tracking-[-0.03em] text-slate-950 sm:text-5xl">
            Как это работает
          </h2>
          <ol class="mt-14 grid gap-10 lg:grid-cols-3 lg:gap-0">
            <li class="relative lg:pr-12">
              <div class="flex items-center gap-5">
                <span class="text-6xl leading-none font-semibold tracking-[-0.06em] text-blue-600"
                  >01</span
                >
                <span class="hidden h-px flex-1 bg-slate-300 lg:block" aria-hidden="true"></span>
              </div>
              <p class="mt-7 text-xl font-semibold tracking-tight text-slate-950 sm:text-2xl">
                Откройте календарь
              </p>
            </li>
            <li class="relative lg:px-12">
              <div class="flex items-center gap-5">
                <span class="text-6xl leading-none font-semibold tracking-[-0.06em] text-blue-600"
                  >02</span
                >
                <span class="hidden h-px flex-1 bg-slate-300 lg:block" aria-hidden="true"></span>
              </div>
              <p class="mt-7 text-xl font-semibold tracking-tight text-slate-950 sm:text-2xl">
                Выберите свободное время
              </p>
            </li>
            <li class="relative lg:pl-12">
              <span class="text-6xl leading-none font-semibold tracking-[-0.06em] text-blue-600"
                >03</span
              >
              <p class="mt-7 text-xl font-semibold tracking-tight text-slate-950 sm:text-2xl">
                Подтвердите бронирование
              </p>
            </li>
          </ol>
        </div>
      </section>

      <section class="mx-auto max-w-4xl px-6 py-20 text-center lg:py-32">
        <h2 class="text-3xl font-bold tracking-[-0.03em] text-slate-950 sm:text-5xl">
          Выберите удобное время для звонка
        </h2>
        <RouterLink
          to="/booking"
          class="mt-9 inline-flex w-full items-center justify-center rounded-xl bg-blue-600 px-6 py-3.5 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-blue-700 focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-blue-600 sm:w-auto sm:min-w-80 sm:text-base"
        >
          Записаться на звонок
        </RouterLink>
      </section>
    </main>

    <footer class="border-t border-slate-200">
      <div
        class="mx-auto max-w-7xl px-6 py-7 text-center text-sm font-medium text-slate-600 lg:px-8"
      >
        Календарь звонков
      </div>
    </footer>
  </div>
</template>
