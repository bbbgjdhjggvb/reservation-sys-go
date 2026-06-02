<script setup lang="ts">
defineProps<{
  name: string
  capacity: string
  motto: string
  logoSrc: string
}>()
</script>

<template>
  <section class="p-6 bg-gradient-to-br from-szu-red-light/30 to-white border-b border-gray-100">
    <div class="flex items-center space-x-4">
      <!-- Circular logo -->
      <div class="w-12 h-12 rounded-full flex-shrink-0 flex items-center justify-center bg-white border border-white shadow-sm overflow-hidden">
        <img :src="logoSrc" :alt="name" class="w-full h-full object-cover" />
      </div>

      <!-- Venue info -->
      <div class="flex-1 min-w-0">
        <h2 class="text-sm md:text-base font-bold text-gray-900 truncate">{{ name }}</h2>
        <div class="flex items-center space-x-2 mt-1">
          <span class="text-[10px] md:text-xs text-gray-400">{{ capacity }}</span>
          <span class="text-gray-200">|</span>
          <span class="text-[10px] md:text-xs text-szu-red font-bold tracking-widest">{{ motto }}</span>
        </div>
      </div>
    </div>
  </section>
</template>
