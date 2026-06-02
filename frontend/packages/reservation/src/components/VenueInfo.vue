<script setup lang="ts">
import { ref } from 'vue'
import alumniGate from '@/assets/alumni_gate.png'
import guidImg from '@/assets/guid.jpg'

const showGuide = ref(false)
const showRules = ref(false)

function openModal(type: 'guide' | 'rules') {
  if (type === 'guide') showGuide.value = true
  else showRules.value = true
}

function closeAll() {
  showGuide.value = false
  showRules.value = false
}
</script>

<template>
  <!-- ===== Info Card ===== -->
  <section class="px-4 pt-6 pb-4">
    <div class="bg-white rounded-2xl overflow-hidden border border-szu-red/15 flex flex-col shadow-[0_8px_24px_rgba(144,17,17,0.03)]">

      <!-- Full-width image (16:9 ratio) -->
      <div class="relative h-32 md:h-36 w-full overflow-hidden bg-gray-50 flex-shrink-0">
        <img :src="alumniGate" alt="校友之家" class="w-full h-full object-cover" />

        <!-- "开放中" status badge -->
        <div class="absolute top-3 left-3 bg-black/60 text-[9px] text-white px-2.5 py-1 rounded-full flex items-center space-x-1.5 backdrop-blur-[2px]">
          <span class="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
          <span class="font-bold">开放中</span>
        </div>
      </div>

      <!-- Action buttons -->
      <div class="p-3 bg-gradient-to-br from-szu-red/[0.02] to-white">
        <div class="grid grid-cols-2 gap-3">
          <!-- Venue guide button: white bg, red border -->
          <button
            @click="openModal('guide')"
            class="group flex items-center justify-center space-x-2 py-2.5 rounded-xl border border-szu-red/30 bg-white hover:bg-szu-red-light/30 transition-all active:scale-[0.97] shadow-sm"
          >
            <span class="text-xs">🗺️</span>
            <span class="text-xs font-bold text-szu-red">场地平面指引</span>
          </button>

          <!-- Rules button: red bg, white text -->
          <button
            @click="openModal('rules')"
            class="group flex items-center justify-center space-x-2 py-2.5 rounded-xl bg-szu-red hover:bg-szu-red-hover text-white transition-all active:scale-[0.97] shadow-sm"
          >
            <span class="text-xs">📜</span>
            <span class="text-xs font-bold">预约使用须知</span>
          </button>
        </div>
      </div>

    </div>
  </section>

  <!-- ===== Modal A: Venue Guide (地图指引) ===== -->
  <Teleport to="body">
    <div
      v-if="showGuide"
      class="fixed inset-0 bg-black/70 flex items-center justify-center z-[100]"
      @click.self="closeAll"
    >
      <div class="bg-white rounded-3xl p-5 max-w-sm w-11/12 text-center relative shadow-2xl">
        <!-- Close button -->
        <button
          @click="closeAll"
          class="absolute -top-3 -right-3 w-8 h-8 rounded-full bg-white text-gray-500 hover:text-szu-red font-bold shadow-md flex items-center justify-center"
        >
          ✕
        </button>

        <h3 class="text-base font-bold text-gray-900 mb-3">多功能会议厅 · 场地指引</h3>

        <!-- Map image (max-height to prevent overscroll on desktop) -->
        <div class="bg-gray-100 rounded-2xl w-full mb-4 overflow-hidden max-h-[60vh] overflow-y-auto">
          <img
            :src="guidImg"
            alt="场地指引地图"
            draggable="false"
            class="w-full h-auto object-contain select-none pointer-events-none"
          />
        </div>

        <button
          @click="closeAll"
          class="w-full py-3 bg-szu-red text-white rounded-xl text-xs font-bold hover:bg-szu-red-hover"
        >
          我知道了
        </button>
      </div>
    </div>
  </Teleport>

  <!-- ===== Modal B: Usage Rules (使用须知) ===== -->
  <Teleport to="body">
    <div
      v-if="showRules"
      class="fixed inset-0 bg-black/70 flex items-center justify-center z-[100]"
      @click.self="closeAll"
    >
      <div class="bg-white rounded-3xl p-6 max-w-sm w-11/12 relative shadow-2xl text-left">
        <!-- Close button -->
        <button
          @click="closeAll"
          class="absolute -top-3 -right-3 w-8 h-8 rounded-full bg-white text-gray-500 hover:text-szu-red font-bold shadow-md flex items-center justify-center"
        >
          ✕
        </button>

        <h3 class="text-base font-bold text-gray-900 mb-4 text-center">多功能会议厅 · 使用须知</h3>

        <div class="max-h-60 overflow-y-auto pr-2 space-y-3.5 text-xs text-gray-600 leading-relaxed mb-6">
          <p class="font-bold text-gray-800">尊敬的校友、师生：</p>
          <p>为保障深圳大学校友之家多功能会议厅合理、高效、安全地使用，请在预约前遵守以下须知：</p>
          <div class="space-y-3">
            <p><span class="font-bold text-szu-red">1. 预约对象</span><br />场地主要面向深圳大学全体校友、在校师生开放，用于校友联谊、学术交流、班级聚会或社团活动。</p>
            <p><span class="font-bold text-szu-red">2. 安全与秩序</span><br />请爱护场内所有公物与电子设备。活动期间严禁携带易燃易爆等危险品入场。禁止高声喧哗，以免影响周围办公秩序。</p>
            <p><span class="font-bold text-szu-red">3. 卫生与复原</span><br />活动结束后，请使用者自行带走产生的垃圾，并将桌椅设施恢复原样。如有损坏，需照价赔偿。</p>
            <p><span class="font-bold text-szu-red">4. 爽约限制</span><br />如因故无法正常进行，请至少提前 24 小时取消预约。若无故爽约超过 2 次，该账号将被暂停预约权限 30 天。</p>
          </div>
        </div>

        <button
          @click="closeAll"
          class="w-full py-3 bg-szu-red text-white rounded-xl text-xs font-bold hover:bg-szu-red-hover text-center"
        >
          我已阅读并同意
        </button>
      </div>
    </div>
  </Teleport>
</template>
