<script setup lang="ts">
import { ref, watch } from 'vue'
import { adminApi } from '@/api/client'
import type { OrderResp } from '@reservation/shared'

type SaveState = 'idle' | 'saving' | 'success' | 'error'

const props = defineProps<{
  visible: boolean
  order: OrderResp | null
}>()

const emit = defineEmits<{
  close: []
  saved: []
}>()

const passwordInputs = ref<Record<number, string>>({})
const overallState = ref<SaveState>('idle')
const errorMsg = ref('')

watch(() => props.order, (o) => {
  overallState.value = 'idle'
  errorMsg.value = ''
  passwordInputs.value = {}
  if (o?.slots) {
    for (const s of o.slots) {
      passwordInputs.value[s.id] = s.password || ''
    }
  }
})

async function saveAll() {
  if (!props.order) return

  // 校验：至少有一个密码不为空
  const slots = props.order.slots
  const hasAny = slots.some(s => (passwordInputs.value[s.id] || '').trim())
  if (!hasAny) {
    overallState.value = 'error'
    errorMsg.value = '请至少输入一个密码'
    return
  }

  overallState.value = 'saving'
  errorMsg.value = ''

  let firstError = ''
  for (const s of slots) {
    const pwd = (passwordInputs.value[s.id] || '').trim()
    if (!pwd) continue // 跳过空的，只保存填了的
    try {
      await adminApi.setPassword(props.order.id, s.id, pwd)
    } catch (e: any) {
      firstError = e.message || '保存失败'
    }
  }

  if (firstError) {
    overallState.value = 'error'
    errorMsg.value = firstError
  } else {
    overallState.value = 'success'
    emit('saved')
    // 成功后 1.5s 自动关闭弹窗
    setTimeout(() => {
      emit('close')
    }, 1500)
  }
}

function buttonLabel(state: SaveState): string {
  switch (state) {
    case 'saving': return '保存中…'
    case 'success': return '全部已保存 ✓'
    case 'error': return '保存失败'
    default: return '保存密码'
  }
}

function close() {
  emit('close')
}
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="visible && order" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black bg-opacity-50 overscroll-contain" @click.self="close">
        <div class="bg-white rounded-xl shadow-xl w-full max-w-md">
          <div class="px-6 py-4 border-b border-gray-100 flex items-center justify-between">
            <h3 class="text-lg font-semibold text-gray-800">设置门锁密码</h3>
            <button class="text-gray-400 hover:text-gray-600 transition" aria-label="关闭" @click="close">
              <svg class="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          <div class="px-6 py-4 space-y-4">
            <p class="text-xs text-gray-400 font-mono">{{ order.order_no }}</p>

            <!-- 密码输入行 -->
            <div class="space-y-3">
              <div v-for="s in order.slots" :key="s.id" class="flex items-center gap-3">
                <span class="text-sm text-gray-600 w-36 flex-shrink-0">{{ s.start_time.slice(5) }} ~ {{ s.end_time.slice(11) }}</span>
                <input
                  v-model="passwordInputs[s.id]"
                  type="text"
                  maxlength="20"
                  class="flex-1 px-3 py-2 text-sm border border-gray-200 rounded-lg focus:border-primary-400 outline-none transition-colors"
                  :class="{ 'border-green-400 bg-green-50': overallState === 'success' }"
                  placeholder="输入密码…"
                  :disabled="overallState === 'saving' || overallState === 'success'"
                />
              </div>
            </div>

            <!-- 全局保存按钮 + 状态反馈 -->
            <div class="space-y-2">
              <button
                :disabled="overallState === 'saving' || overallState === 'success'"
                class="w-full py-2.5 rounded-lg text-sm font-medium transition-all"
                :class="{
                  'bg-primary-500 text-white hover:bg-primary-600': overallState === 'idle',
                  'bg-gray-400 text-white cursor-not-allowed': overallState === 'saving',
                  'bg-green-500 text-white': overallState === 'success',
                  'bg-red-500 text-white hover:bg-red-600': overallState === 'error',
                }"
                @click="saveAll"
              >
                {{ buttonLabel(overallState) }}
              </button>
              <p v-if="errorMsg" class="text-xs text-red-500 text-center">{{ errorMsg }}</p>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.modal-enter-active { animation: modalIn 0.2s ease; }
.modal-leave-active { animation: modalIn 0.2s ease reverse; }
@keyframes modalIn {
  from { opacity: 0; transform: scale(1) translateY(10px); }
  to   { opacity: 1; transform: scale(1) translateY(0); }
}
</style>
