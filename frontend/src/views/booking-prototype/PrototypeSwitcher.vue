<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted } from 'vue'
import type { PrototypeVariant } from './model'
import { prototypeVariants } from './model'

const props = defineProps<{
  current: PrototypeVariant
  state: Record<string, unknown>
}>()

const emit = defineEmits<{
  change: [variant: PrototypeVariant]
}>()

const currentIndex = computed(() =>
  prototypeVariants.findIndex((variant) => variant.key === props.current),
)

function cycle(direction: -1 | 1) {
  const nextIndex =
    (currentIndex.value + direction + prototypeVariants.length) % prototypeVariants.length
  emit('change', prototypeVariants[nextIndex]!.key)
}

function handleKeydown(event: KeyboardEvent) {
  const target = event.target as HTMLElement | null
  if (target?.matches('input, textarea, [contenteditable="true"]')) return

  if (event.key === 'ArrowLeft') {
    event.preventDefault()
    cycle(-1)
  }

  if (event.key === 'ArrowRight') {
    event.preventDefault()
    cycle(1)
  }
}

onMounted(() => window.addEventListener('keydown', handleKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', handleKeydown))
</script>

<template>
  <aside
    class="pointer-events-none fixed inset-x-0 bottom-4 z-50 flex flex-col items-center gap-2 px-4"
  >
    <details
      class="pointer-events-auto max-h-52 w-full max-w-xl overflow-auto rounded-xl border border-white/20 bg-slate-950/95 px-4 py-2 text-xs text-slate-200 shadow-2xl backdrop-blur"
    >
      <summary class="cursor-pointer font-semibold text-white">Состояние прототипа</summary>
      <pre class="mt-2 whitespace-pre-wrap break-all">{{ JSON.stringify(state, null, 2) }}</pre>
    </details>

    <div
      class="pointer-events-auto flex items-center gap-1 rounded-full border border-white/20 bg-slate-950/95 p-1.5 text-white shadow-2xl backdrop-blur"
    >
      <button
        type="button"
        class="grid size-9 place-items-center rounded-full transition hover:bg-white/15 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white"
        aria-label="Предыдущий вариант"
        @click="cycle(-1)"
      >
        ←
      </button>
      <span class="min-w-48 px-3 text-center text-xs font-semibold sm:text-sm">
        {{ current }} · {{ prototypeVariants[currentIndex]!.name }}
      </span>
      <button
        type="button"
        class="grid size-9 place-items-center rounded-full transition hover:bg-white/15 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white"
        aria-label="Следующий вариант"
        @click="cycle(1)"
      >
        →
      </button>
    </div>
  </aside>
</template>
