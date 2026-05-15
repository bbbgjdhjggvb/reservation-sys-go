<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAdminStore } from '@/stores/admin'

const router = useRouter()
const admin = useAdminStore()

const username = ref('')
const password = ref('')
const errorMsg = ref('')
const loading = ref(false)

onMounted(() => {
  if (admin.isAuthenticated) {
    router.replace('/admin/dashboard')
  }
})

async function handleLogin() {
  errorMsg.value = ''
  if (!username.value.trim() || !password.value.trim()) {
    errorMsg.value = '请输入用户名和密码'
    return
  }
  loading.value = true
  try {
    await admin.login(username.value.trim(), password.value)
    router.push('/admin/dashboard')
  } catch (e: any) {
    errorMsg.value = e.message || '登录失败，请重试'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="min-h-screen bg-[#FDF6F6] flex items-center justify-center p-4">
    <div class="bg-white rounded-2xl shadow-lg p-8 w-full max-w-md">
      <div class="text-center mb-8">
        <div class="inline-flex items-center justify-center w-16 h-16 rounded-full bg-primary-500 mb-4">
          <svg class="w-8 h-8 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
          </svg>
        </div>
        <h1 class="text-xl font-bold text-gray-800">场地预约管理系统</h1>
        <p class="text-sm text-gray-500 mt-1">管理员审核后台</p>
      </div>

      <form class="space-y-5" @submit.prevent="handleLogin">
        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1" for="login-username">用户名</label>
          <input
            id="login-username"
            v-model="username"
            type="text"
            name="username"
            autocomplete="username"
            class="w-full px-4 py-3 rounded-lg border border-gray-200 focus:border-red-400 focus:ring-2 focus:ring-red-100 outline-none transition text-base"
            style="font-size: 16px;"
            placeholder="请输入用户名…"
          />
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1" for="login-password">密码</label>
          <input
            id="login-password"
            v-model="password"
            type="password"
            name="password"
            autocomplete="current-password"
            class="w-full px-4 py-3 rounded-lg border border-gray-200 focus:border-red-400 focus:ring-2 focus:ring-red-100 outline-none transition text-base"
            style="font-size: 16px;"
            placeholder="请输入密码…"
          />
        </div>

        <div v-if="errorMsg" class="bg-red-50 p-3 rounded-lg text-center text-sm text-red-600">
          {{ errorMsg }}
        </div>

        <button
          :disabled="loading"
          class="w-full py-3 rounded-lg bg-primary-500 text-white font-semibold text-base hover:bg-primary-600 disabled:opacity-60 transition shadow-md"
        >
          {{ loading ? '登录中…' : '登录' }}
        </button>
      </form>

      <p class="text-center text-xs text-gray-400 mt-8">
        深圳大学校友场地预约系统 &copy; 2026
      </p>
    </div>
  </div>
</template>
