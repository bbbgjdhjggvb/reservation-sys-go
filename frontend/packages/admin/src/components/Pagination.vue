<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  currentPage: number
  total: number
  pageSize: number
}>()

const emit = defineEmits<{
  'page-change': [page: number]
}>()

const totalPages = computed(() => Math.max(1, Math.ceil(props.total / props.pageSize)))
</script>

<template>
  <div v-if="totalPages > 1" class="flex justify-center gap-2 mt-6">
    <button
      :disabled="currentPage <= 1"
      class="px-3 py-1.5 text-sm rounded border border-gray-200 text-gray-600 hover:bg-gray-50 disabled:opacity-40 disabled:cursor-not-allowed transition"
      @click="$emit('page-change', currentPage - 1)"
    >
      上一页
    </button>
    <span class="px-3 py-1.5 text-sm text-gray-500">
      {{ currentPage }} / {{ totalPages }}
    </span>
    <button
      :disabled="currentPage >= totalPages"
      class="px-3 py-1.5 text-sm rounded border border-gray-200 text-gray-600 hover:bg-gray-50 disabled:opacity-40 disabled:cursor-not-allowed transition"
      @click="$emit('page-change', currentPage + 1)"
    >
      下一页
    </button>
  </div>
</template>
