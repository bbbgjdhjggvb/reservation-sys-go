<script setup lang="ts">
import { useAdminStore } from '@/stores/admin'
import { useRouter } from 'vue-router'

const admin = useAdminStore()
const router = useRouter()

function handleLogout() {
  admin.logout()
  router.replace('/admin/login')
}
</script>

<template>
  <header class="sticky top-0 z-40 bg-white shadow-sm">
    <div class="max-w-6xl mx-auto px-4 py-3 flex items-center justify-between">
      <div class="flex items-center gap-3">
        <div class="w-8 h-8 rounded-lg bg-primary-500 flex items-center justify-center">
          <svg class="w-5 h-5 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
          </svg>
        </div>
        <h1 class="text-lg font-bold text-primary-500">预约审核中心</h1>
      </div>
      <div class="flex items-center gap-3">
        <div v-if="admin.info" class="text-sm text-gray-600">
          <span class="font-medium">{{ admin.info.real_name }}</span>
          <span class="text-gray-400 ml-2">{{ admin.info.role_text }}</span>
        </div>
        <button class="text-sm text-gray-400 hover:text-red-500 transition" @click="handleLogout">
          退出
        </button>
      </div>
    </div>
  </header>
</template>
