<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { onBeforeRouteLeave, RouterLink, useRoute } from 'vue-router'
import AdminSidebar from '@/components/AdminSidebar.vue'
import {
  createAdminCallType,
  deleteAdminCallType,
  listAdminCallTypes,
  listAdminSchedules,
  updateAdminCallType,
  type AdminCallType,
  type AdminSchedule,
  type CallTypeStatus,
} from '@/api/generated'

type CallTypeForm = {
  name: string
  description: string
  durationMinutes: number
  status: CallTypeStatus
  scheduleId: string
}

type ApiError = {
  code?: string
  field_errors?: Array<{ path: string; message: string }>
}

const loading = ref(true)
const loadFailed = ref(false)
const callTypes = ref<AdminCallType[]>([])
const schedules = ref<AdminSchedule[]>([])
const selectedId = ref<string | null>(null)
const creating = ref(false)
const mobileEditing = ref(false)
const form = ref<CallTypeForm | null>(null)
const confirmedForm = ref<CallTypeForm | null>(null)
const saving = ref(false)
const deleting = ref(false)
const validationAttempted = ref(false)
const fieldErrors = ref<Record<string, string>>({})
const actionError = ref('')
const notice = ref('')
const route = useRoute()

const selectedCallType = computed(
  () => callTypes.value.find((callType) => callType.id === selectedId.value) ?? null,
)
const dirty = computed(
  () =>
    form.value !== null &&
    confirmedForm.value !== null &&
    JSON.stringify(form.value) !== JSON.stringify(confirmedForm.value),
)
const publicLink = computed(() => {
  const callType = selectedCallType.value
  return callType ? `${window.location.origin}/booking/${callType.slug}` : ''
})

function callTypeForm(callType: AdminCallType): CallTypeForm {
  return {
    name: callType.name,
    description: callType.description ?? '',
    durationMinutes: callType.duration_minutes,
    status: callType.status,
    scheduleId: callType.schedule.id,
  }
}

function cloneForm(source: CallTypeForm): CallTypeForm {
  return { ...source }
}

function sortCallTypes(items: AdminCallType[]) {
  return [...items].sort((left, right) => {
    if (left.status !== right.status) return left.status === 'active' ? -1 : 1
    if (left.duration_minutes !== right.duration_minutes)
      return left.duration_minutes - right.duration_minutes
    return left.name.localeCompare(right.name, 'ru')
  })
}

function acceptCallType(callType: AdminCallType, openMobile = false) {
  const next = callTypeForm(callType)
  selectedId.value = callType.id
  creating.value = false
  mobileEditing.value = openMobile
  form.value = next
  confirmedForm.value = cloneForm(next)
  validationAttempted.value = false
  fieldErrors.value = {}
}

function resetFeedback() {
  actionError.value = ''
  notice.value = ''
  fieldErrors.value = {}
  validationAttempted.value = false
}

async function loadCallTypes() {
  loading.value = true
  loadFailed.value = false
  try {
    const [callTypeResult, scheduleResult] = await Promise.all([
      listAdminCallTypes(),
      listAdminSchedules(),
    ])
    if (!callTypeResult.data || !scheduleResult.data) {
      loadFailed.value = true
      return
    }
    callTypes.value = sortCallTypes(callTypeResult.data.items)
    schedules.value = [...scheduleResult.data.items].sort((left, right) =>
      left.name.localeCompare(right.name, 'ru'),
    )
    const requestedID = Array.isArray(route.query.call_type_id)
      ? route.query.call_type_id[0]
      : route.query.call_type_id
    const requestedCallType = callTypes.value.find((callType) => callType.id === requestedID)
    const initialCallType = requestedCallType ?? callTypes.value[0]
    if (initialCallType) acceptCallType(initialCallType, requestedCallType !== undefined)
  } catch {
    loadFailed.value = true
  } finally {
    loading.value = false
  }
}

function startCreating() {
  if (!confirmDiscard()) return
  const next: CallTypeForm = {
    name: '',
    description: '',
    durationMinutes: 30,
    status: 'active',
    scheduleId: schedules.value.length === 1 ? schedules.value[0]!.id : '',
  }
  selectedId.value = null
  creating.value = true
  mobileEditing.value = true
  form.value = next
  confirmedForm.value = cloneForm(next)
  resetFeedback()
}

function selectCallType(callType: AdminCallType) {
  if (callType.id === selectedId.value) {
    mobileEditing.value = true
    return
  }
  if (!confirmDiscard()) return
  resetFeedback()
  acceptCallType(callType, true)
}

function validateForm() {
  if (!form.value) return false
  const errors: Record<string, string> = {}
  const nameLength = [...form.value.name.trim()].length
  const descriptionLength = [...form.value.description.trim()].length
  if (nameLength < 1 || nameLength > 80)
    errors.name = 'Введите название длиной от 1 до 80 символов.'
  if (descriptionLength > 500) errors.description = 'Описание должно быть не длиннее 500 символов.'
  if (
    !Number.isInteger(form.value.durationMinutes) ||
    form.value.durationMinutes < 15 ||
    form.value.durationMinutes > 240 ||
    form.value.durationMinutes % 5 !== 0
  )
    errors.duration_minutes = 'Укажите от 15 до 240 минут с шагом 5.'
  if (!form.value.scheduleId) errors.schedule_id = 'Выберите расписание.'
  fieldErrors.value = errors
  return Object.keys(errors).length === 0
}

function markEdited() {
  notice.value = ''
  actionError.value = ''
  if (validationAttempted.value) validateForm()
}

function applyServerErrors(error: ApiError | undefined) {
  if (!error?.field_errors) return
  const errors = { ...fieldErrors.value }
  for (const fieldError of error.field_errors) {
    const field = fieldError.path.replace('body.', '')
    errors[field] = fieldError.message
  }
  fieldErrors.value = errors
}

async function saveCallType() {
  if (!form.value || saving.value) return
  validationAttempted.value = true
  if (!validateForm()) return
  saving.value = true
  actionError.value = ''
  notice.value = ''
  const submitted = cloneForm(form.value)
  const body = {
    name: submitted.name,
    description: submitted.description.trim() || null,
    duration_minutes: submitted.durationMinutes,
    schedule_id: submitted.scheduleId,
  }
  const current = selectedCallType.value
  const wasCreating = creating.value
  try {
    const result = creating.value
      ? await createAdminCallType({ body })
      : await updateAdminCallType({
          path: { id: current!.id },
          headers: { 'If-Match': `"${current!.version}"` },
          body: { ...body, status: submitted.status },
        })
    if (!result.data) {
      const error = result.error as ApiError | undefined
      applyServerErrors(error)
      actionError.value =
        error?.code === 'version_conflict'
          ? 'Тип звонка уже изменён. Обновите страницу и повторите изменения.'
          : error?.code === 'call_type_name_conflict'
            ? 'Тип звонка с таким названием уже существует.'
            : error?.code === 'schedule_unavailable'
              ? 'Выбранное расписание больше недоступно.'
              : 'Не удалось сохранить тип звонка. Исправьте ошибки или повторите попытку.'
      return
    }
    const saved = result.data
    const index = callTypes.value.findIndex((item) => item.id === saved.id)
    if (index >= 0) callTypes.value.splice(index, 1, saved)
    else callTypes.value.push(saved)
    callTypes.value = sortCallTypes(callTypes.value)
    if (JSON.stringify(form.value) === JSON.stringify(submitted)) {
      acceptCallType(saved, true)
    } else {
      selectedId.value = saved.id
      creating.value = false
      confirmedForm.value = callTypeForm(saved)
      validateForm()
    }
    notice.value = wasCreating ? 'Тип звонка создан' : 'Тип звонка сохранён'
  } catch {
    actionError.value = 'Не удалось связаться с сервером. Введённые изменения сохранены в форме.'
    if (creating.value) {
      try {
        const refreshed = await listAdminCallTypes()
        if (refreshed.data) callTypes.value = sortCallTypes(refreshed.data.items)
      } catch {
        // Keep the form after an uncertain create outcome.
      }
    }
  } finally {
    saving.value = false
  }
}

async function removeCallType() {
  const callType = selectedCallType.value
  if (!callType?.can_delete || deleting.value) return
  if (!window.confirm(`Удалить тип звонка «${callType.name}»? Это действие необратимо.`)) return
  deleting.value = true
  actionError.value = ''
  notice.value = ''
  try {
    const result = await deleteAdminCallType({
      path: { id: callType.id },
      headers: { 'If-Match': `"${callType.version}"` },
    })
    const error = result.error as ApiError | undefined
    if (result.response?.status !== 204) {
      actionError.value =
        error?.code === 'version_conflict'
          ? 'Тип звонка уже изменён. Обновите страницу перед удалением.'
          : error?.code === 'call_type_has_calls'
            ? 'У типа уже есть звонки. Его можно только отключить.'
            : 'Не удалось удалить тип звонка.'
      return
    }
    const index = callTypes.value.findIndex((item) => item.id === callType.id)
    callTypes.value.splice(index, 1)
    const next = callTypes.value[Math.min(index, callTypes.value.length - 1)]
    if (next) acceptCallType(next, true)
    else {
      selectedId.value = null
      form.value = null
      confirmedForm.value = null
    }
    notice.value = 'Тип звонка удалён'
  } catch {
    actionError.value = 'Не удалось связаться с сервером. Тип звонка не удалён.'
  } finally {
    deleting.value = false
  }
}

function showMobileList() {
  if (!confirmDiscard()) return
  mobileEditing.value = false
}

async function copyPublicLink() {
  if (!publicLink.value) return
  try {
    await navigator.clipboard.writeText(publicLink.value)
    notice.value = 'Публичная ссылка скопирована'
  } catch {
    actionError.value = 'Не удалось скопировать ссылку.'
  }
}

function deletionReason(callType: AdminCallType) {
  return callType.delete_blocked_reason === 'call_type_has_calls'
    ? 'У типа есть история звонков — его можно только отключить'
    : ''
}

function confirmDiscard() {
  return !dirty.value || window.confirm('Отменить несохранённые изменения типа звонка?')
}

function handleBeforeUnload(event: BeforeUnloadEvent) {
  if (!dirty.value) return
  event.preventDefault()
}

onBeforeRouteLeave(confirmDiscard)
onMounted(() => {
  window.addEventListener('beforeunload', handleBeforeUnload)
  void loadCallTypes()
})
onBeforeUnmount(() => window.removeEventListener('beforeunload', handleBeforeUnload))
</script>

<template>
  <div class="min-h-svh bg-slate-100 text-slate-950 lg:grid lg:grid-cols-[17rem_1fr]">
    <AdminSidebar
      active="types-and-schedules"
      note="Типы звонков используют именованные расписания внутри общего режима работы."
    />

    <main class="px-4 py-8 sm:px-8 lg:px-12 lg:py-10">
      <div class="mx-auto max-w-7xl">
        <header
          class="border-b border-slate-300 pb-7 sm:flex sm:items-end sm:justify-between sm:gap-6"
        >
          <div>
            <p class="text-xs font-bold tracking-[0.2em] text-blue-700 uppercase">Организатор</p>
            <h1 class="mt-3 text-3xl font-bold tracking-[-0.035em] sm:text-4xl">Типы звонков</h1>
            <p class="mt-3 max-w-2xl leading-7 text-slate-600">
              Управляйте длительностью, расписанием и доступностью публичных вариантов звонка.
            </p>
          </div>
          <div class="mt-5 flex flex-wrap items-center gap-3 sm:mt-0">
            <RouterLink
              to="/admin/schedules"
              class="rounded-lg border border-slate-300 px-4 py-2.5 text-sm font-semibold"
            >
              Расписания
            </RouterLink>
            <button
              type="button"
              data-create-call-type
              class="rounded-lg bg-slate-950 px-4 py-2.5 text-sm font-semibold text-white disabled:opacity-45"
              :disabled="schedules.length === 0"
              @click="startCreating"
            >
              + Создать тип звонка
            </button>
          </div>
        </header>

        <p v-if="actionError && !form" class="mt-6 text-sm font-medium text-red-700" role="alert">
          {{ actionError }}
        </p>
        <p v-if="notice" class="mt-6 text-sm font-medium text-emerald-700" role="status">
          {{ notice }}
        </p>

        <div v-if="loading" class="mt-8 rounded-xl bg-white p-8" aria-busy="true">
          Загружаем типы звонков…
        </div>
        <div
          v-else-if="loadFailed"
          class="mt-8 rounded-xl border border-red-200 bg-red-50 p-6"
          role="alert"
        >
          <h2 class="text-lg font-bold">Не удалось загрузить типы звонков</h2>
          <button
            type="button"
            class="mt-5 rounded-lg bg-slate-950 px-4 py-2 text-sm font-semibold text-white"
            @click="loadCallTypes"
          >
            Повторить
          </button>
        </div>

        <div v-else class="mt-8 grid gap-6 lg:grid-cols-[minmax(17rem,23rem)_minmax(0,1fr)]">
          <section
            aria-label="Список типов звонков"
            class="space-y-3"
            :class="mobileEditing ? 'hidden lg:block' : 'block'"
          >
            <div
              v-if="callTypes.length === 0"
              class="rounded-xl border border-dashed border-slate-300 bg-white p-6"
            >
              <h2 class="font-bold">Типов звонков пока нет</h2>
              <p class="mt-2 text-sm leading-6 text-slate-600">
                Создайте первый тип, чтобы он появился в публичном каталоге.
              </p>
              <button
                type="button"
                class="mt-4 text-sm font-semibold text-blue-700"
                @click="startCreating"
              >
                Создать тип звонка
              </button>
            </div>
            <button
              v-for="callType in callTypes"
              :key="callType.id"
              type="button"
              :data-select-call-type="callType.id"
              class="w-full rounded-xl border p-4 text-left shadow-sm"
              :class="
                selectedId === callType.id
                  ? 'border-blue-500 bg-blue-50'
                  : 'border-slate-200 bg-white hover:border-slate-300'
              "
              @click="selectCallType(callType)"
            >
              <span class="block font-bold">{{ callType.name }}</span>
              <span class="mt-1 block text-sm text-slate-600"
                >{{ callType.duration_minutes }} минут · {{ callType.schedule.name }} ·
                {{ callType.status === 'active' ? 'Активен' : 'Отключён' }}</span
              >
              <span
                v-if="!callType.can_delete"
                class="mt-2 block text-xs font-medium text-amber-700"
                >{{ deletionReason(callType) }}</span
              >
            </button>
          </section>

          <form
            v-if="form"
            class="min-w-0"
            :class="mobileEditing ? 'block' : 'hidden lg:block'"
            @submit.prevent="saveCallType"
          >
            <button
              type="button"
              data-mobile-call-type-back
              class="mb-4 text-sm font-semibold text-blue-700 lg:hidden"
              @click="showMobileList"
            >
              ← К списку типов
            </button>
            <section class="rounded-xl border border-slate-200 bg-white p-5 shadow-sm sm:p-7">
              <div class="grid gap-5 sm:grid-cols-2">
                <div class="sm:col-span-2">
                  <label for="call-type-name" class="text-sm font-semibold">Название</label>
                  <input
                    id="call-type-name"
                    v-model="form.name"
                    name="call-type-name"
                    maxlength="80"
                    class="mt-2 w-full rounded-lg border border-slate-300 px-3 py-2.5"
                    @input="markEdited"
                  />
                  <p v-if="fieldErrors.name" class="mt-2 text-sm font-medium text-red-700">
                    {{ fieldErrors.name }}
                  </p>
                </div>
                <div class="sm:col-span-2">
                  <label for="call-type-description" class="text-sm font-semibold"
                    >Описание <span class="font-normal text-slate-500">(необязательно)</span></label
                  >
                  <textarea
                    id="call-type-description"
                    v-model="form.description"
                    name="call-type-description"
                    maxlength="500"
                    rows="4"
                    class="mt-2 w-full rounded-lg border border-slate-300 px-3 py-2.5"
                    @input="markEdited"
                  />
                  <p v-if="fieldErrors.description" class="mt-2 text-sm font-medium text-red-700">
                    {{ fieldErrors.description }}
                  </p>
                </div>
                <div>
                  <label for="call-type-duration" class="text-sm font-semibold"
                    >Длительность, минут</label
                  >
                  <input
                    id="call-type-duration"
                    v-model.number="form.durationMinutes"
                    name="call-type-duration"
                    type="number"
                    min="15"
                    max="240"
                    step="5"
                    class="mt-2 w-full rounded-lg border border-slate-300 px-3 py-2.5"
                    @input="markEdited"
                  />
                  <p
                    v-if="fieldErrors.duration_minutes"
                    class="mt-2 text-sm font-medium text-red-700"
                  >
                    {{ fieldErrors.duration_minutes }}
                  </p>
                </div>
                <div>
                  <label for="call-type-schedule" class="text-sm font-semibold">Расписание</label>
                  <select
                    id="call-type-schedule"
                    v-model="form.scheduleId"
                    name="call-type-schedule"
                    class="mt-2 w-full rounded-lg border border-slate-300 bg-white px-3 py-2.5"
                    @change="markEdited"
                  >
                    <option value="" disabled>Выберите расписание</option>
                    <option v-for="schedule in schedules" :key="schedule.id" :value="schedule.id">
                      {{ schedule.name }}
                    </option>
                  </select>
                  <p v-if="fieldErrors.schedule_id" class="mt-2 text-sm font-medium text-red-700">
                    {{ fieldErrors.schedule_id }}
                  </p>
                </div>
                <div v-if="!creating">
                  <label for="call-type-status" class="text-sm font-semibold">Статус</label>
                  <select
                    id="call-type-status"
                    v-model="form.status"
                    name="call-type-status"
                    class="mt-2 w-full rounded-lg border border-slate-300 bg-white px-3 py-2.5"
                    @change="markEdited"
                  >
                    <option value="active">Активен</option>
                    <option value="disabled">Отключён</option>
                  </select>
                </div>
              </div>

              <div v-if="selectedCallType" class="mt-7 rounded-lg bg-slate-50 p-4">
                <p class="text-xs font-bold tracking-wide text-slate-500 uppercase">
                  Стабильная публичная ссылка
                </p>
                <div class="mt-2 flex flex-col gap-3 sm:flex-row sm:items-center">
                  <code class="min-w-0 flex-1 overflow-hidden text-ellipsis text-sm">{{
                    publicLink
                  }}</code>
                  <button
                    type="button"
                    data-copy-call-type-link
                    class="rounded-lg border border-slate-300 px-3 py-2 text-sm font-semibold"
                    @click="copyPublicLink"
                  >
                    Скопировать
                  </button>
                </div>
                <p class="mt-2 text-xs text-slate-500">
                  Slug не изменится при переименовании: {{ selectedCallType.slug }}
                </p>
              </div>
            </section>

            <div
              class="mt-6 border-t border-slate-300 bg-slate-100/95 py-5 backdrop-blur sm:sticky sm:bottom-0"
            >
              <p v-if="actionError" class="mb-3 text-sm font-medium text-red-700" role="alert">
                {{ actionError }}
              </p>
              <div class="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <button
                    v-if="selectedCallType"
                    type="button"
                    data-delete-call-type
                    class="rounded-lg border border-red-200 px-4 py-2.5 text-sm font-semibold text-red-700 disabled:cursor-not-allowed disabled:opacity-45"
                    :disabled="deleting || !selectedCallType.can_delete"
                    :title="deletionReason(selectedCallType)"
                    @click="removeCallType"
                  >
                    {{ deleting ? 'Удаляем…' : 'Удалить тип' }}
                  </button>
                  <p
                    v-if="selectedCallType && !selectedCallType.can_delete"
                    class="mt-2 max-w-md text-xs text-amber-700"
                  >
                    {{ deletionReason(selectedCallType) }}
                  </p>
                </div>
                <div class="flex items-center gap-4">
                  <p class="text-sm text-slate-500">
                    {{ dirty ? 'Есть несохранённые изменения' : 'Все изменения сохранены' }}
                  </p>
                  <button
                    type="submit"
                    class="rounded-lg bg-blue-600 px-5 py-2.5 text-sm font-semibold text-white disabled:cursor-not-allowed disabled:opacity-45"
                    :disabled="saving || (!creating && !dirty)"
                  >
                    {{ saving ? 'Сохраняем…' : creating ? 'Создать тип' : 'Сохранить тип' }}
                  </button>
                </div>
              </div>
            </div>
          </form>
        </div>
      </div>
    </main>
  </div>
</template>
