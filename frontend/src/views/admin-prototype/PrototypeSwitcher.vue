<script setup lang="ts">
import { onBeforeUnmount, onMounted } from 'vue'
import type { PrototypeVariant } from './model'

const props = defineProps<{
  current: PrototypeVariant
  state: Record<string, unknown>
}>()

const emit = defineEmits<{
  change: [variant: PrototypeVariant]
}>()

const variants: Array<{ key: PrototypeVariant; name: string }> = [
  { key: 'A', name: 'Разделы' },
  { key: 'B', name: 'Консоль' },
  { key: 'C', name: 'Доска расписаний' },
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
  <div class="fixed inset-x-0 bottom-5 z-50 flex justify-center px-4">
    <div
      class="flex items-center gap-2 rounded-full border border-white/20 bg-slate-950 px-2 py-2 text-white shadow-2xl"
    >
      <button
        type="button"
        class="grid size-9 place-items-center rounded-full hover:bg-white/10"
        aria-label="Предыдущий вариант"
        @click="cycle(-1)"
      >
        ←
      </button>
      <details class="group relative">
        <summary class="min-w-48 cursor-pointer list-none px-3 text-center text-sm font-semibold">
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
