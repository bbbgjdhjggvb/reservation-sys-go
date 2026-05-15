<script setup lang="ts">
import { ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useCalendar } from '@/composables/useCalendar'
import { useToast } from '@/composables/useToast'
import { api } from '@/api/client'
import CalendarGrid from '@/components/CalendarGrid.vue'
import ReserveForm from '@/components/ReserveForm.vue'
import ConfirmModal from '@/components/ConfirmModal.vue'
import ToastNotification from '@/components/ToastNotification.vue'
import type { SubmitReq } from '@reservation/shared'

const auth = useAuthStore()
const cal = useCalendar()
const { selectedSlots } = cal
const { showToast } = useToast()

const currentStep = ref<1 | 2>(1)
const showConfirm = ref(false)
const pendingFormData = ref<SubmitReq | null>(null)
const submitting = ref(false)

function goToForm() {
  if (selectedSlots.value.length === 0) {
    showToast('请至少选择一个时段', 'warning')
    return
  }
  currentStep.value = 2
}

function goToCalendar() {
  currentStep.value = 1
}

function onFormConfirm(data: SubmitReq) {
  pendingFormData.value = data
  showConfirm.value = true
}

function closeConfirm() {
  showConfirm.value = false
}

async function doSubmit() {
  if (!pendingFormData.value) return
  submitting.value = true
  try {
    await api.submit(pendingFormData.value)
    showConfirm.value = false
    showToast(`预约提交成功，共${pendingFormData.value.slots.length}个时段，请等待审核`, 'success')

    pendingFormData.value = null
    currentStep.value = 1
    await cal.resetAndRefresh()
  } catch (e: any) {
    showToast(e.message || '提交失败，请重试', 'error')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="min-h-screen">
    <header class="sticky top-0 z-40 bg-white shadow-sm">
      <div class="max-w-4xl mx-auto px-4 py-3 flex items-center justify-between">
        <h1 class="text-lg font-bold text-primary-500">场地预约系统</h1>
        <router-link to="/myorders" class="text-sm text-primary-500 hover:text-primary-600 font-medium">
          我的预约
        </router-link>
      </div>
    </header>

    <main class="max-w-4xl mx-auto px-4 py-6">
      <!-- Token Error -->
      <div v-if="!auth.isAuthenticated" class="bg-red-50 border border-red-200 rounded-lg p-4 text-center text-red-600">
        未授权访问，请从微信服务号进入预约界面
      </div>

      <div v-else class="bg-white rounded-lg shadow">
        <!-- Step 1: Calendar -->
        <div v-if="currentStep === 1" class="p-4 sm:p-6">
          <CalendarGrid />

          <!-- Legend -->
          <div class="flex flex-wrap gap-3 mt-4 text-xs text-gray-500">
            <span class="flex items-center gap-1"><span class="w-3 h-3 rounded bg-primary-50 border border-primary-200" />可选</span>
            <span class="flex items-center gap-1"><span class="w-3 h-3 rounded bg-primary-500" />已选</span>
            <span class="flex items-center gap-1"><span class="w-3 h-3 rounded bg-yellow-100 border border-yellow-300" />待审核</span>
            <span class="flex items-center gap-1"><span class="w-3 h-3 rounded bg-red-100 border border-red-300" />已占用</span>
            <span class="flex items-center gap-1"><span class="w-3 h-3 rounded bg-gray-100" />不可选</span>
          </div>

          <!-- Selected Slots Info -->
          <div v-if="selectedSlots.length > 0" class="mt-4 p-3 bg-primary-50 rounded-lg">
            <p class="text-sm font-medium text-primary-700 mb-1">已选时段：</p>
            <p v-for="slot in selectedSlots" :key="slot.date + slot.startTime" class="text-sm text-primary-600">
              {{ cal.formatSlotDisplay(slot) }}
            </p>
          </div>

          <!-- Next Button -->
          <button
            :disabled="selectedSlots.length === 0"
            class="mt-4 w-full px-4 py-3 rounded-lg bg-primary-500 text-white font-semibold hover:bg-primary-600 disabled:opacity-40 disabled:cursor-not-allowed transition shadow-md"
            @click="goToForm"
          >
            下一步：填写信息
            <span v-if="selectedSlots.length > 0">（已选 {{ selectedSlots.length }}/{{ cal.MAX_SLOTS }}）</span>
          </button>
        </div>

        <!-- Step 2: Form -->
        <div v-if="currentStep === 2" class="p-4 sm:p-6">
          <ReserveForm @back="goToCalendar" @confirm="onFormConfirm" />
        </div>
      </div>
    </main>

    <ConfirmModal
      :visible="showConfirm"
      :pending-data="pendingFormData"
      @cancel="closeConfirm"
      @confirm="doSubmit"
    />

    <ToastNotification />
  </div>
</template>
