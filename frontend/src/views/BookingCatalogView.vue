<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { listPublicCallTypes, type PublicCallTypeSummary } from '@/api/generated'

const callTypes = ref<PublicCallTypeSummary[]>([])
const loading = ref(true)
const loadFailed = ref(false)

async function loadCallTypes() {
  loading.value = true
  loadFailed.value = false
  try {
    const result = await listPublicCallTypes()
    if (!result.data) {
      loadFailed.value = true
      return
    }
    callTypes.value = result.data.items
  } catch {
    loadFailed.value = true
  } finally {
    loading.value = false
  }
}

onMounted(loadCallTypes)
</script>

<template>
  <main class="grid min-h-svh bg-slate-100 text-slate-950 lg:grid-cols-[22rem_1fr]">
    <aside class="flex flex-col bg-slate-950 p-8 text-white sm:p-12">
      <RouterLink to="/" class="text-lg font-bold tracking-tight">Календарь звонков</RouterLink>
      <div class="my-auto py-16">
        <p class="text-sm font-semibold text-cyan-300">Онлайн-бронирование</p>
        <h1 class="mt-4 text-4xl font-bold tracking-[-0.04em]">Бронирование звонка</h1>
        <p class="mt-5 leading-7 text-slate-300">
          Все доступные типы звонков в одном списке. Выберите подходящую длительность.
        </p>
      </div>
      <p class="text-sm text-slate-400">Europe/Moscow · без пересчёта часового пояса</p>
    </aside>

    <section class="flex items-center px-5 py-12 sm:px-12 lg:px-20">
      <div class="w-full max-w-4xl">
        <p class="text-xs font-bold tracking-[0.2em] text-slate-500 uppercase">
          Активные типы звонков
        </p>

        <div v-if="loading" class="mt-6 space-y-4" aria-busy="true">
          <p class="sr-only">Загружаем типы звонков</p>
          <div v-for="index in 3" :key="index" class="h-24 rounded-xl bg-white" />
        </div>

        <div
          v-else-if="loadFailed"
          class="mt-6 border-y border-red-200 bg-red-50 px-5 py-8"
          role="alert"
        >
          <h2 class="text-xl font-bold">Не удалось загрузить типы звонков</h2>
          <p class="mt-2 text-slate-600">Проверьте соединение и попробуйте ещё раз.</p>
          <button
            type="button"
            class="mt-5 rounded-lg bg-slate-950 px-4 py-2 font-semibold text-white hover:bg-cyan-700"
            @click="loadCallTypes"
          >
            Повторить
          </button>
        </div>

        <div
          v-else-if="callTypes.length"
          class="mt-6 divide-y divide-slate-200 border-y border-slate-300"
        >
          <RouterLink
            v-for="callType in callTypes"
            :key="callType.slug"
            :to="{ name: 'booking-type', params: { typeSlug: callType.slug } }"
            class="group grid gap-4 py-7 text-left transition hover:px-4 hover:bg-white sm:grid-cols-[1fr_auto] sm:items-center"
          >
            <span>
              <span class="block text-2xl font-bold tracking-tight">{{ callType.name }}</span>
              <span v-if="callType.description" class="mt-2 block text-slate-600">
                {{ callType.description }}
              </span>
            </span>
            <span class="flex items-center gap-5">
              <span class="font-mono text-sm text-slate-500">
                {{ callType.duration_minutes }} мин
              </span>
              <span
                class="grid size-11 place-items-center rounded-full bg-slate-950 text-white transition group-hover:bg-cyan-600"
                aria-hidden="true"
              >
                →
              </span>
            </span>
          </RouterLink>
        </div>

        <div v-else class="mt-6 border-y border-slate-300 py-10">
          <h2 class="text-2xl font-bold">Сейчас нет доступных типов звонков</h2>
          <p class="mt-3 text-slate-600">Попробуйте вернуться позже.</p>
        </div>
      </div>
    </section>
  </main>
</template>
