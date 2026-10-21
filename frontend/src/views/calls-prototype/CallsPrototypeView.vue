<!-- Three variants of the organizer call list, switchable via ?variant= on /admin/calls-prototype. -->
<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import PrototypeSwitcher from './PrototypeSwitcher.vue'
import VariantA from './VariantA.vue'
import VariantB from './VariantB.vue'
import VariantC from './VariantC.vue'
import {
  filterCalls,
  prototypeCalls,
  type CallFilters,
  type PrototypeScenario,
  type PrototypeVariant,
} from './model'

const route = useRoute()
const router = useRouter()
const showPrototypeSwitcher = !import.meta.env.PROD

const variant = computed<PrototypeVariant>(() => {
  const value = route.query.variant
  return value === 'B' || value === 'C' ? value : 'A'
})

const scenario = computed<PrototypeScenario>(() => {
  const value = route.query.scenario
  return value === 'loading' ||
    value === 'error' ||
    value === 'empty' ||
    value === 'load-more-error'
    ? value
    : 'ready'
})

const filters = ref<CallFilters>({
  period: 'upcoming',
  typeName: '',
  query: '',
})
const selectedCallId = ref<string | null>('call-101')
const visibleCalls = computed(() =>
  scenario.value === 'empty' ? [] : filterCalls(prototypeCalls, filters.value),
)

watch(visibleCalls, (calls) => {
  if (!calls.some((call) => call.id === selectedCallId.value)) {
    selectedCallId.value = calls[0]?.id ?? null
  }
})

function replaceQuery(change: { variant?: PrototypeVariant; scenario?: PrototypeScenario }) {
  void router.replace({
    query: {
      ...route.query,
      ...(change.variant ? { variant: change.variant } : {}),
      ...(change.scenario && change.scenario !== 'ready'
        ? { scenario: change.scenario }
        : { scenario: undefined }),
    },
  })
}

const exposedState = computed(() => ({
  variant: variant.value,
  scenario: scenario.value,
  filters: filters.value,
  selectedCallId: selectedCallId.value,
  visibleCallIds: visibleCalls.value.map((call) => call.id),
}))
</script>

<template>
  <component
    :is="variant === 'A' ? VariantA : variant === 'B' ? VariantB : VariantC"
    :calls="visibleCalls"
    :filters="filters"
    :scenario="scenario"
    :selected-call-id="selectedCallId"
    @update:filters="filters = $event"
    @select-call="selectedCallId = $event"
  />

  <PrototypeSwitcher
    v-if="showPrototypeSwitcher"
    :current="variant"
    :scenario="scenario"
    :state="exposedState"
    @change="replaceQuery({ variant: $event })"
    @change-scenario="replaceQuery({ scenario: $event })"
  />
</template>
