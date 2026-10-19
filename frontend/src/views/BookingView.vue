<script setup lang="ts">
import { RouterLink } from 'vue-router'
</script>

<template>
  <div class="flex min-h-svh flex-col bg-white text-slate-950">
    <header class="border-b border-slate-200">
      <div
        class="mx-auto w-full max-w-7xl px-6 py-5 text-sm font-bold tracking-tight sm:text-lg lg:px-8"
      >
        Календарь звонков
      </div>
    </header>

    <main class="mx-auto flex w-full max-w-3xl flex-1 items-center px-6 py-20 text-center">
      <div class="w-full">
        <h1 class="text-4xl font-bold tracking-[-0.04em] text-slate-950 sm:text-6xl">
          Бронирование звонка
        </h1>
        <p class="mt-6 text-lg leading-8 text-slate-600 sm:text-xl">Выбор времени появится здесь</p>
        <RouterLink
          to="/"
          class="mt-10 inline-flex items-center justify-center rounded-xl border border-slate-300 bg-white px-6 py-3 text-sm font-semibold text-slate-950 shadow-sm transition-colors hover:bg-slate-50 focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-blue-600 sm:text-base"
        >
          Вернуться на главную
        </RouterLink>
      </div>
    </main>
  </div>
</template>
