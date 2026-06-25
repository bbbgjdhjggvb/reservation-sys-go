<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
auth.init()
</script>

<template>
  <div class="min-h-screen flex flex-col">
    <div class="flex-1">
      <router-view />
    </div>
    <footer class="text-center text-xs text-gray-400 py-3">
      <a href="https://beian.miit.gov.cn/" target="_blank" rel="noopener noreferrer" class="hover:text-gray-500 transition-colors">
        ICP备案号：粤ICP备2026072431号
      </a>
    </footer>
  </div>
</template>
