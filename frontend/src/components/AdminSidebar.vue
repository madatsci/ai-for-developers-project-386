<script setup lang="ts">
import { RouterLink } from 'vue-router'

defineProps<{
  active: 'calls' | 'types-and-schedules' | 'working-hours'
  note: string
}>()

const activeClass = 'bg-white font-semibold text-slate-950'
const inactiveClass = 'text-slate-300 hover:bg-slate-900'
</script>

<template>
  <aside class="bg-slate-950 px-6 py-6 text-white lg:min-h-svh lg:px-7 lg:py-8">
    <p class="text-lg font-bold tracking-tight">Календарь звонков</p>
    <p class="mt-1 text-xs font-semibold tracking-[0.16em] text-slate-400 uppercase">Управление</p>
    <nav class="mt-8 grid gap-1" aria-label="Административные разделы">
      <RouterLink
        to="/admin/calls"
        class="rounded-lg px-3 py-2.5 text-sm"
        :class="active === 'calls' ? activeClass : inactiveClass"
      >
        Звонки
      </RouterLink>
      <RouterLink
        to="/admin/call-types"
        class="rounded-lg px-3 py-2.5 text-sm"
        :class="active === 'types-and-schedules' ? activeClass : inactiveClass"
      >
        Типы и расписания
      </RouterLink>
      <RouterLink
        to="/admin/working-hours"
        class="rounded-lg px-3 py-2.5 text-sm"
        :class="active === 'working-hours' ? activeClass : inactiveClass"
      >
        Режим работы
      </RouterLink>
    </nav>
    <p class="mt-8 border-t border-slate-800 pt-5 text-xs leading-5 text-slate-400">
      {{ note }}
    </p>
  </aside>
</template>
