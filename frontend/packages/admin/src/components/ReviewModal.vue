<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{
  visible: boolean
  action: number // 1=pass, 2=reject
  actionLabel: string
}>()

const emit = defineEmits<{
  close: []
  confirm: [comment: string]
}>()

const comment = ref('')
const submitting = ref(false)

function handleConfirm() {
  emit('confirm', comment.value.trim())
}

function handleClose() {
  comment.value = ''
  emit('close')
}
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="visible" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black bg-opacity-50 overscroll-contain" @click.self="handleClose">
        <div class="bg-white rounded-xl shadow-xl w-full max-w-sm p-6">
          <h3 class="text-lg font-semibold text-gray-800 mb-2">
            确认{{ actionLabel }}
          </h3>
          <p class="text-sm text-gray-500 mb-4">
            {{ action === 1 ? '确定通过此预约申请？' : '确定拒绝此预约申请？' }}
          </p>
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-1" for="review-comment">{{ actionLabel }}意见（可选）：</label>
            <textarea
              id="review-comment"
              v-model="comment"
              name="comment"
              rows="2"
              maxlength="500"
              class="w-full px-3 py-2 rounded-lg border border-gray-200 text-sm focus:border-red-400 focus:ring-2 focus:ring-red-100 outline-none resize-none"
              :placeholder="action === 1 ? '如：材料齐全，同意通过…' : '如：材料不全，请补充后重新申请…'"
            />
          </div>
          <div class="flex gap-3">
            <button class="flex-1 px-4 py-2.5 rounded-lg border border-gray-200 text-gray-600 hover:bg-gray-50 transition" @click="handleClose">
              取消
            </button>
            <button
              :class="action === 1
                ? 'bg-green-500 hover:bg-green-600'
                : 'bg-red-500 hover:bg-red-600'"
              class="flex-1 px-4 py-2.5 rounded-lg text-white font-medium transition"
              @click="handleConfirm"
            >
              {{ actionLabel }}
            </button>
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
