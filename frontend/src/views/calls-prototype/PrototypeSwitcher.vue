<script setup lang="ts">
import { onBeforeUnmount, onMounted } from 'vue'
import type { PrototypeScenario, PrototypeVariant } from './model'

const props = defineProps<{
  current: PrototypeVariant
  scenario: PrototypeScenario
  state: Record<string, unknown>
}>()

const emit = defineEmits<{
  change: [variant: PrototypeVariant]
  changeScenario: [scenario: PrototypeScenario]
}>()

const variants: Array<{ key: PrototypeVariant; name: string }> = [
  { key: 'A', name: 'Повестка' },
  { key: 'B', name: 'Журнал' },
  { key: 'C', name: 'День' },
]

function cycle(offset: number) {
  const currentIndex = variants.findIndex((variant) => variant.key === props.current)
  const nextIndex = (currentIndex + offset + variants.length) % variants.length
  emit('change', variants[nextIndex]!.key)
}

function onKeydown(event: KeyboardEvent) {
  const target = event.target as HTMLElement | null
  if (target?.matches('input, textarea, select, [contenteditable]')) return
  if (event.key === 'ArrowLeft') cycle(-1)
  if (event.key === 'ArrowRight') cycle(1)
}

onMounted(() => window.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))
</script>

<template>
  <div class="fixed inset-x-0 bottom-4 z-50 flex justify-center px-4">
    <div
      class="flex max-w-full flex-wrap items-center justify-center gap-2 rounded-2xl border border-white/15 bg-slate-950 p-2 text-white shadow-2xl"
    >
      <label class="ml-1 flex items-center gap-2 text-xs text-slate-300">
        Сценарий
        <select
          :value="scenario"
          class="rounded-lg border border-white/15 bg-white/10 px-2 py-1.5 text-white"
          @change="
            emit('changeScenario', ($event.target as HTMLSelectElement).value as PrototypeScenario)
          "
        >
          <option class="text-black" value="ready">Данные</option>
          <option class="text-black" value="loading">Загрузка</option>
          <option class="text-black" value="error">Ошибка</option>
          <option class="text-black" value="empty">Пусто</option>
          <option class="text-black" value="load-more-error">Ошибка догрузки</option>
        </select>
      </label>
      <span class="mx-1 h-6 w-px bg-white/15" />
      <button
        type="button"
        class="grid size-9 place-items-center rounded-full hover:bg-white/10"
        aria-label="Предыдущий вариант"
        @click="cycle(-1)"
      >
        ←
      </button>
      <details class="group relative">
        <summary class="min-w-36 cursor-pointer list-none px-2 text-center text-sm font-semibold">
          {{ current }} · {{ variants.find((variant) => variant.key === current)?.name }}
        </summary>
        <pre
          class="absolute bottom-14 left-1/2 max-h-72 w-[min(38rem,90vw)] -translate-x-1/2 overflow-auto rounded-xl bg-slate-950 p-4 text-left text-xs text-slate-300 shadow-2xl"
          >{{ JSON.stringify(state, null, 2) }}</pre>
      </details>
      <button
        type="button"
        class="grid size-9 place-items-center rounded-full hover:bg-white/10"
        aria-label="Следующий вариант"
        @click="cycle(1)"
      >
        →
      </button>
    </div>
  </div>
</template>
