<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useCalendar } from '@/composables/useCalendar'
import { useToast } from '@/composables/useToast'
import { useReservationSSE } from '@/composables/useSSE'
import { api } from '@/api/client'
import DateAxis from '@/components/DateAxis.vue'
import TimeSlotCard from '@/components/TimeSlotCard.vue'
import ReserveForm from '@/components/ReserveForm.vue'
import ConfirmModal from '@/components/ConfirmModal.vue'
import ToastNotification from '@/components/ToastNotification.vue'
import VenueInfo from '@/components/VenueInfo.vue'
import szuHomeLogo from '@/assets/szu_home_logo.jpg'
import type { SubmitReq } from '@reservation/shared'

const auth = useAuthStore()
const cal = useCalendar()
const { selectedSlots, activeDayIndex, visibleDays, activeDateStr, TIME_SLOTS } = cal
const { showToast } = useToast()

// 建立 SSE 实时推送连接，监听订单变更后自动刷新日历
// 若 SSE 不可用，自动降级为 15s 轮询
useReservationSSE()

const currentStep = ref<1 | 2>(1)
const showConfirm = ref(false)
const pendingFormData = ref<SubmitReq | null>(null)
const submitting = ref(false)
const lastMaxSlotToastTime = ref(0)

// Fetch occupied slots on mount
onMounted(() => {
  if (auth.isAuthenticated) {
    cal.fetchOccupiedSlots()
  }
})

// Re-fetch if auth state becomes true later
watch(() => auth.isAuthenticated, (val) => {
  if (val) cal.fetchOccupiedSlots()
})

function onSelectDay(index: number) {
  activeDayIndex.value = index
}

function onToggleSlot(startTime: string, endTime: string) {
  const ok = cal.toggleSlot(activeDateStr.value, startTime, endTime)
  if (!ok) {
    const now = Date.now()
    if (now - lastMaxSlotToastTime.value > 3000) {
      lastMaxSlotToastTime.value = now
      showToast(`最多只能选择 ${cal.MAX_SLOTS} 个时段`, 'warning')
    }
  }
}

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
  <div class="min-h-screen bg-gray-100 py-0 md:py-8">
    <!-- Responsive card container -->
    <div class="w-full max-w-md md:max-w-xl mx-auto bg-white min-h-screen md:min-h-[840px] flex flex-col shadow-lg relative pb-28 md:rounded-3xl md:overflow-hidden md:border md:border-gray-100">

      <!-- 1. Header -->
      <header class="bg-white border-b border-gray-100 px-6 py-4 sticky top-0 z-50 flex items-center justify-between">
        <div class="flex items-center space-x-2">
          <img :src="szuHomeLogo" alt="校友之家" class="h-10" />
        </div>
        <router-link
          to="/myorders"
          class="text-xs font-bold text-szu-red px-4 py-2 border border-szu-red rounded-full hover:bg-szu-red-light transition-colors"
        >
          我的预约
        </router-link>
      </header>

      <!-- Token Error Banner -->
      <div v-if="!auth.isAuthenticated" class="bg-red-50 border-b border-red-200 px-6 py-4 text-center text-red-600 text-sm">
        未授权访问，请从微信服务号进入预约界面
      </div>

      <template v-else>
        <!-- 2. Venue Info Card -->
        <VenueInfo />

        <!-- Step 1: Calendar + Slots -->
        <template v-if="currentStep === 1">
          <!-- 3. Date Axis — sticky below header -->
          <section class="py-3 bg-white border-b border-gray-100 sticky top-[72px] z-40 shadow-sm overflow-hidden">
            <div class="px-6">
              <DateAxis
                :days="visibleDays"
                :active-index="activeDayIndex"
                @select-day="onSelectDay"
              />
            </div>
          </section>

          <!-- 4. Time Slot List -->
          <section class="p-6 flex-1">
            <div class="flex items-center justify-between mb-4">
              <h3 class="text-xs md:text-sm font-bold text-gray-400 tracking-wider uppercase">
                {{ cal.activeDayLabel.value }} · 预约状态
              </h3>
              <span
                class="text-[10px] md:text-xs px-2 py-0.5 rounded font-bold"
                :class="cal.isNextWeek.value
                  ? 'bg-gray-100 text-gray-500'
                  : 'bg-szu-red/10 text-szu-red'"
              >
                {{ cal.isNextWeek.value ? '下周' : '本周' }}
              </span>
            </div>

            <div class="space-y-3.5">
              <TimeSlotCard
                v-for="slot in TIME_SLOTS"
                :key="`${activeDateStr}-${slot.start}-${slot.end}`"
                :date="activeDateStr"
                :start-time="slot.start"
                :end-time="slot.end"
                :cell-state="cal.getCellState(activeDateStr, slot.start, slot.end, visibleDays[activeDayIndex])"
                @toggle="onToggleSlot(slot.start, slot.end)"
              />
            </div>
          </section>

          <!-- 5. Floating Bottom Action Bar -->
          <section class="absolute bottom-0 left-0 right-0 bg-white border-t border-gray-100 shadow-[0_-4px_12px_rgba(0,0,0,0.04)] px-6 py-4 z-50 md:rounded-b-3xl">
            <!-- Legend -->
            <div class="flex items-center justify-between text-[10px] text-gray-400 mb-3 px-1 flex-wrap gap-y-1">
              <div class="flex items-center space-x-1">
                <span class="w-2.5 h-2.5 rounded bg-white border border-gray-200" />
                <span>可选</span>
              </div>
              <div class="flex items-center space-x-1">
                <span class="w-2.5 h-2.5 rounded bg-szu-red" />
                <span class="text-szu-red font-bold">已选</span>
              </div>
              <div class="flex items-center space-x-1">
                <span class="w-2.5 h-2.5 rounded bg-amber-50 border border-amber-200" />
                <span class="text-amber-600 font-bold">待审核</span>
              </div>
              <div class="flex items-center space-x-1">
                <span class="w-2.5 h-2.5 rounded stripe-bg border border-red-100" />
                <span class="text-red-400 font-bold">已占用</span>
              </div>
              <div class="flex items-center space-x-1">
                <span class="w-2.5 h-2.5 rounded bg-emerald-50 border border-emerald-200" />
                <span class="text-emerald-600 font-bold">已通过</span>
              </div>
              <div class="flex items-center space-x-1">
                <span class="w-2.5 h-2.5 rounded bg-gray-100 border border-gray-200" />
                <span>不可选</span>
              </div>
            </div>

            <!-- CTA Button -->
            <button
              :disabled="selectedSlots.length === 0"
              class="w-full py-4 bg-szu-red text-white hover:bg-szu-red-hover active:scale-[0.98] transition-all rounded-xl font-bold tracking-widest text-sm shadow-md shadow-[#901111]/20 disabled:opacity-40 disabled:cursor-not-allowed disabled:active:scale-100"
              @click="goToForm"
            >
              下一步：填写信息
              <span v-if="selectedSlots.length > 0">（已选 {{ selectedSlots.length }}/{{ cal.MAX_SLOTS }}）</span>
            </button>
          </section>
        </template>

        <!-- Step 2: ReserveForm -->
        <template v-if="currentStep === 2">
          <section class="p-6 flex-1">
            <ReserveForm @back="goToCalendar" @confirm="onFormConfirm" />
          </section>
        </template>
      </template>

      <!-- ConfirmModal and ToastNotification -->
      <ConfirmModal
        :visible="showConfirm"
        :pending-data="pendingFormData"
        @cancel="closeConfirm"
        @confirm="doSubmit"
      />

      <ToastNotification />
    </div>
  </div>
</template>
