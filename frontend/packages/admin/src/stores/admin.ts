import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { AdminLoginResp, AdminInfoResp } from '../types'

export const useAdminStore = defineStore('admin', () => {
  const token = ref<string | null>(localStorage.getItem('admin_token'))
  const info = ref<AdminInfoResp | null>(null)
  const isAuthenticated = computed(() => !!token.value)

  async function login(username: string, password: string): Promise<boolean> {
    const { adminApi } = await import('../api/client')
    const data: AdminLoginResp = await adminApi.login(username, password)
    token.value = data.token
    info.value = {
      id: 0,
      username: data.username,
      real_name: data.real_name,
      role: data.role,
      role_text: data.role_text,
    }
    localStorage.setItem('admin_token', data.token)
    return true
  }

  async function fetchInfo() {
    if (!token.value) return
    const { adminApi } = await import('../api/client')
    info.value = await adminApi.getAdminInfo()
  }

  function logout() {
    token.value = null
    info.value = null
    localStorage.removeItem('admin_token')
    localStorage.removeItem('admin_info')
  }

  return { token, info, isAuthenticated, login, fetchInfo, logout }
})
