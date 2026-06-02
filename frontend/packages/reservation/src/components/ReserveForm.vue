<script setup lang="ts">
import { ref } from 'vue'
import { useCalendar } from '@/composables/useCalendar'
import { useToast } from '@/composables/useToast'
import { ALUMNI_OPTIONS } from '@reservation/shared'
import AlumniAutocomplete from './AlumniAutocomplete.vue'
import type { SubmitReq } from '@reservation/shared'

const emit = defineEmits<{
  back: []
  confirm: [data: SubmitReq]
}>()

const cal = useCalendar()
const { selectedSlots } = cal
const { showToast } = useToast()

const name = ref('')
const year = ref('')
const alumniValue = ref('')
const major = ref('')
const phone = ref('')
const attendeeCount = ref('')
const reason = ref('')

function handleSubmit() {
  if (!name.value.trim()) { showToast('请填写申请人姓名', 'error'); return }
  if (!year.value.trim() || !/^\d{4}$/.test(year.value)) { showToast('请填写正确的入学年级（4位数字）', 'error'); return }
  if (!alumniValue.value || !ALUMNI_OPTIONS.includes(alumniValue.value)) { showToast('请从列表中选择校友会', 'error'); return }
  if (!major.value.trim()) { showToast('请填写专业', 'error'); return }
  if (!phone.value.trim() || !/^\d{11}$/.test(phone.value)) { showToast('请填写正确的手机号码（11位数字）', 'error'); return }
  if (!reason.value.trim()) { showToast('请填写借用事由', 'error'); return }
  if (!attendeeCount.value.trim() || !/^\d+$/.test(attendeeCount.value) || parseInt(attendeeCount.value) < 1) { showToast('请填写正确的会议人数', 'error'); return }

  const slots = selectedSlots.value.map(s => ({
    start_time: `${s.date} ${s.startTime}:00`,
    end_time: `${s.date} ${s.endTime}:00`,
  }))

  emit('confirm', {
    applicant_name: name.value.trim(),
    year: parseInt(year.value),
    alumni_association: alumniValue.value,
    major: major.value.trim(),
    phone: phone.value.trim(),
    reason: reason.value.trim(),
    attendee_count: parseInt(attendeeCount.value),
    slots,
  })
}
</script>

<template>
  <div>
    <div class="mb-4 p-3 bg-primary-50 rounded-lg">
      <p class="text-sm font-medium text-primary-700 mb-2">已选时段：</p>
      <p v-for="slot in selectedSlots" :key="slot.date + slot.startTime" class="text-sm text-primary-600">
        {{ cal.formatSlotDisplay(slot) }}
      </p>
    </div>

    <form class="space-y-4" @submit.prevent="handleSubmit">
      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1" for="reserve-name">申请人姓名</label>
        <input
          id="reserve-name"
          v-model="name"
          type="text"
          name="name"
          autocomplete="name"
          required
          class="w-full px-4 py-3 rounded-lg border border-gray-200 text-base focus:border-red-400 focus:ring-2 focus:ring-red-100 outline-none transition"
          style="font-size: 16px;"
          placeholder="请输入姓名…"
        />
      </div>

      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1" for="reserve-year">入学年级</label>
        <input
          id="reserve-year"
          v-model="year"
          type="text"
          name="year"
          inputmode="numeric"
          maxlength="4"
          pattern="\d{4}"
          class="w-full px-4 py-3 rounded-lg border border-gray-200 text-base focus:border-red-400 focus:ring-2 focus:ring-red-100 outline-none transition"
          style="font-size: 16px;"
          placeholder="如 2020…"
        />
      </div>

      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1">校友会</label>
        <AlumniAutocomplete v-model="alumniValue" />
      </div>

      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1" for="reserve-major">专业</label>
        <input
          id="reserve-major"
          v-model="major"
          type="text"
          name="major"
          autocomplete="organization-title"
          required
          class="w-full px-4 py-3 rounded-lg border border-gray-200 text-base focus:border-red-400 focus:ring-2 focus:ring-red-100 outline-none transition"
          style="font-size: 16px;"
          placeholder="请输入专业名称…"
        />
      </div>

      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1" for="reserve-phone">手机号码</label>
        <input
          id="reserve-phone"
          v-model="phone"
          type="tel"
          name="phone"
          autocomplete="tel"
          inputmode="numeric"
          maxlength="11"
          pattern="[0-9]{11}"
          class="w-full px-4 py-3 rounded-lg border border-gray-200 text-base focus:border-red-400 focus:ring-2 focus:ring-red-100 outline-none transition"
          style="font-size: 16px;"
          placeholder="请输入11位手机号码…"
        />
      </div>

      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1" for="reserve-attendees">会议人数</label>
        <input
          id="reserve-attendees"
          v-model="attendeeCount"
          type="text"
          name="attendees"
          inputmode="numeric"
          maxlength="4"
          pattern="[0-9]+"
          class="w-full px-4 py-3 rounded-lg border border-gray-200 text-base focus:border-red-400 focus:ring-2 focus:ring-red-100 outline-none transition"
          style="font-size: 16px;"
          placeholder="请输入参会人数…"
        />
      </div>

      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1" for="reserve-reason">
          借用事由 <span class="text-xs text-gray-400">{{ reason.length }}/500</span>
        </label>
        <textarea
          id="reserve-reason"
          v-model="reason"
          name="reason"
          maxlength="500"
          rows="3"
          class="w-full px-4 py-3 rounded-lg border border-gray-200 text-base focus:border-red-400 focus:ring-2 focus:ring-red-100 outline-none transition resize-none"
          style="font-size: 16px;"
          placeholder="请简要描述活动内容…"
        />
      </div>

      <div class="flex gap-3 pt-2">
        <button
          type="button"
          class="flex-1 px-4 py-3 rounded-lg border border-gray-200 text-gray-600 font-medium hover:bg-gray-50 transition"
          @click="$emit('back')"
        >
          返回选择时间
        </button>
        <button
          type="submit"
          class="flex-1 px-4 py-3 rounded-lg bg-primary-500 text-white font-semibold hover:bg-primary-600 transition shadow-md"
        >
          确定提交
        </button>
      </div>
      <p class="mt-3 text-xs text-gray-400 text-center">
        请勿频繁操作
      </p>
    </form>
  </div>
</template>
