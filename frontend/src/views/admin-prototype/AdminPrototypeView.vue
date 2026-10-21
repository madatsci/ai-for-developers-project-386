<script setup lang="ts">
// Three throwaway variants of the unprotected admin, switchable via ?variant= on /admin/prototype.
import { computed, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import PrototypeSwitcher from './PrototypeSwitcher.vue'
import VariantA from './VariantA.vue'
import VariantB from './VariantB.vue'
import VariantC from './VariantC.vue'
import { cloneCallTypes, cloneSchedules, workingHours, type PrototypeVariant } from './model'

const route = useRoute()
const router = useRouter()
const callTypes = reactive(cloneCallTypes())
const schedules = reactive(cloneSchedules())
const message = ref('')
const showPrototypeTools = import.meta.env.DEV

const variant = computed<PrototypeVariant>(() => {
  const requested = route.query.variant
  return requested === 'B' || requested === 'C' ? requested : 'A'
})

const variantComponent = computed(() => ({ A: VariantA, B: VariantB, C: VariantC })[variant.value])

function changeVariant(nextVariant: PrototypeVariant) {
  void router.replace({ query: { ...route.query, variant: nextVariant } })
}

function toggleType(id: number) {
  const callType = callTypes.find((item) => item.id === id)
  if (!callType) return
  callType.active = !callType.active
  message.value = `${callType.name}: ${callType.active ? 'тип включён' : 'тип отключён'}. Публичная ссылка не изменилась.`
}

function reassignType(typeId: number, scheduleId: number) {
  const callType = callTypes.find((item) => item.id === typeId)
  const schedule = schedules.find((item) => item.id === scheduleId)
  if (!callType || !schedule) return
  callType.scheduleId = scheduleId
  message.value = `${callType.name} теперь использует расписание «${schedule.name}». Существующие звонки не меняются.`
}

function deleteType(id: number) {
  const index = callTypes.findIndex((item) => item.id === id)
  if (index === -1) return
  const callType = callTypes[index]!
  if (callType.hasBookings) {
    message.value = `Нельзя удалить «${callType.name}»: по этому типу уже создавались бронирования. Его можно отключить.`
    return
  }
  callTypes.splice(index, 1)
  message.value = `Тип «${callType.name}» удалён. Его публичный идентификатор больше не используется.`
}

function addType() {
  const id = Math.max(0, ...callTypes.map((item) => item.id)) + 1
  callTypes.push({
    id,
    name: `Новый тип ${id}`,
    durationMinutes: 45,
    scheduleId: schedules[0]!.id,
    active: true,
    hasBookings: false,
  })
  message.value = `Создан активный «Новый тип ${id}» с расписанием «${schedules[0]!.name}».`
}

function addSchedule() {
  const id = Math.max(0, ...schedules.map((item) => item.id)) + 1
  schedules.push({
    id,
    name: `Новое расписание ${id}`,
    availability: ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс'].map((day, index) => ({
      day,
      intervals: index < 5 ? ['09:00–18:00'] : [],
    })),
  })
  message.value = `Создано расписание «Новое расписание ${id}»: Пн–Пт 09:00–18:00, Сб–Вс выходные. Оно пока не назначено типам звонков.`
}

function deleteSchedule(id: number) {
  const schedule = schedules.find((item) => item.id === id)
  if (!schedule) return
  const linkedTypes = callTypes.filter((item) => item.scheduleId === id)
  if (schedules.length === 1) {
    message.value = `Нельзя удалить «${schedule.name}»: у организатора должно остаться хотя бы одно расписание.`
    return
  }
  if (linkedTypes.length > 0) {
    message.value = `Нельзя удалить «${schedule.name}»: оно назначено ${linkedTypes.length} типам звонков. Сначала переназначьте их.`
    return
  }
  schedules.splice(schedules.indexOf(schedule), 1)
  message.value = `Расписание «${schedule.name}» удалено.`
}

function showValidation() {
  message.value =
    'Интервалы 09:00–13:00 и 12:30–15:00 пересекаются. Измените начало или конец второго интервала.'
}

const prototypeState = computed<Record<string, unknown>>(() => ({
  route: route.path,
  variant: variant.value,
  workingHours,
  callTypes: callTypes.map((item) => ({
    name: item.name,
    active: item.active,
    schedule: schedules.find((schedule) => schedule.id === item.scheduleId)?.name,
  })),
  schedules: schedules.map((schedule) => ({
    name: schedule.name,
    assignedTypes: callTypes.filter((item) => item.scheduleId === schedule.id).length,
  })),
  lastResult: message.value || null,
}))
</script>

<template>
  <component
    :is="variantComponent"
    :call-types="callTypes"
    :schedules="schedules"
    :working-hours="workingHours"
    :message="message"
    @toggle-type="toggleType"
    @reassign-type="reassignType"
    @delete-type="deleteType"
    @add-type="addType"
    @add-schedule="addSchedule"
    @delete-schedule="deleteSchedule"
    @show-validation="showValidation"
  />

  <PrototypeSwitcher
    v-if="showPrototypeTools"
    :current="variant"
    :state="prototypeState"
    @change="changeVariant"
  />
</template>
