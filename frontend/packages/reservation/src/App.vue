<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
auth.init()
</script>

<template>
  <router-view />
</template>
