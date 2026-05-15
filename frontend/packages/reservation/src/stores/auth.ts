import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(null)
  const error = ref<string | null>(null)

  const isAuthenticated = computed(() => !!token.value)

  function init() {
    const params = new URLSearchParams(window.location.search)
    const urlToken = params.get('token')
    const storedToken = localStorage.getItem('auth_token')

    if (urlToken) {
      token.value = urlToken
      localStorage.setItem('auth_token', urlToken)
      const url = new URL(window.location.href)
      url.searchParams.delete('token')
      window.history.replaceState({}, '', url.toString())
    } else if (storedToken) {
      token.value = storedToken
    } else {
      error.value = '未授权访问'
    }
  }

  function logout() {
    token.value = null
    error.value = '登录已过期，请重新进入'
    localStorage.removeItem('auth_token')
  }

  return { token, error, isAuthenticated, init, logout }
})
