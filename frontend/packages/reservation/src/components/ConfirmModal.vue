<script setup lang="ts">
import { ref } from 'vue'
import type { SubmitReq } from '@reservation/shared'

const props = defineProps<{
  visible: boolean
  pendingData: SubmitReq | null
}>()

const emit = defineEmits<{
  confirm: []
  cancel: []
}>()

const submitting = ref(false)
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="visible" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black bg-opacity-50 overscroll-contain" @click.self="$emit('cancel')">
        <div class="bg-white rounded-xl shadow-xl w-full max-w-md overflow-hidden">
          <div class="p-6">
            <div class="flex items-center gap-3 mb-4">
              <div class="w-8 h-8 rounded-full bg-primary-100 flex items-center justify-center flex-shrink-0">
                <svg class="w-5 h-5 text-primary-500" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z" />
                </svg>
              </div>
              <div>
                <h3 class="text-lg font-semibold text-gray-800">确认提交预约</h3>
                <p class="text-sm text-gray-500">请仔细核对以下信息，提交后需等待审核</p>
              </div>
            </div>

            <div v-if="pendingData" class="bg-gray-50 rounded-lg p-4 space-y-3 text-sm">
              <div>
                <p class="text-gray-500 mb-1">预约时段（{{ pendingData.slots.length }}个）：</p>
                <p v-for="(s, i) in pendingData.slots" :key="i" class="text-gray-800">
                  {{ s.start_time.slice(0, 16) }} ~ {{ s.end_time.slice(11, 16) }}
                </p>
              </div>
              <hr class="border-gray-200" />
              <div class="grid grid-cols-2 gap-2">
                <p class="text-gray-500">申请人：</p>
                <p class="text-gray-800">{{ pendingData.applicant_name }}</p>
                <p class="text-gray-500">年级：</p>
                <p class="text-gray-800">{{ pendingData.year }}</p>
                <p class="text-gray-500">校友会：</p>
                <p class="text-gray-800">{{ pendingData.alumni_association }}</p>
                <p class="text-gray-500">专业：</p>
                <p class="text-gray-800">{{ pendingData.major }}</p>
                <p class="text-gray-500">手机：</p>
                <p class="text-gray-800">{{ pendingData.phone }}</p>
              </div>
              <div v-if="pendingData.reason">
                <p class="text-gray-500">事由：</p>
                <p class="text-gray-800">{{ pendingData.reason }}</p>
              </div>
            </div>

            <div class="flex gap-3 mt-6">
              <button
                :disabled="submitting"
                class="flex-1 px-4 py-3 rounded-lg border border-gray-200 text-gray-600 font-medium hover:bg-gray-50 transition"
                @click="$emit('cancel')"
              >
                返回修改
              </button>
              <button
                :disabled="submitting"
                class="flex-1 px-4 py-3 rounded-lg bg-primary-500 text-white font-semibold hover:bg-primary-600 disabled:opacity-60 transition shadow-md flex items-center justify-center gap-2"
                @click="$emit('confirm')"
              >
                <svg v-if="submitting" class="animate-spin w-4 h-4" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                </svg>
                {{ submitting ? '提交中…' : '确认提交' }}
              </button>
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
