<script setup lang="ts">
import { ref, computed, nextTick } from 'vue'
import { ALUMNI_OPTIONS } from '@reservation/shared'

const props = defineProps<{
  modelValue: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const inputValue = ref('')
const showDropdown = ref(false)
const activeIdx = ref(-1)
const dropdownRef = ref<HTMLElement | null>()

const filteredOptions = ref<string[]>([])

function escapeHtml(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
}

function fuzzyScore(query: string, target: string): number {
  if (!query) return 0
  const q = query.toLowerCase()
  const t = target.toLowerCase()
  let score = 0

  // Prefix match of any segment
  const segments = target.split(/[\/\-、，\s]+/)
  for (const seg of segments) {
    if (seg.toLowerCase().startsWith(q)) {
      score = Math.max(score, 100)
    }
  }

  // Consecutive char match
  let consecCount = 0
  let maxConsec = 0
  let ti = 0
  for (let qi = 0; qi < q.length && ti < t.length; ti++) {
    if (t[ti] === q[qi]) {
      consecCount++
      qi++
      maxConsec = Math.max(maxConsec, consecCount)
    } else {
      consecCount = 0
      qi = 0
    }
  }
  score += maxConsec * 10

  // Any char found
  for (const ch of q) {
    if (t.includes(ch)) score += 1
  }

  // Substring match
  if (t.includes(q)) {
    score = Math.max(score, 80)
  }

  // Word match
  const qWords = q.split(/\s+/)
  const tWords = t.split(/[\/\-、，\s]+/)
  const matchedWords = qWords.filter(qw => tWords.some(tw => tw.includes(qw)))
  if (matchedWords.length > 0) {
    score = Math.max(score, 60 + matchedWords.length * 3)
  }

  // Common suffix bonus
  const suffixes = ['校友会', '学院', '学部']
  for (const suffix of suffixes) {
    if (q.endsWith(suffix) && t.endsWith(suffix)) score += 5
  }

  return score
}

function filterAlumni(query: string) {
  if (!query.trim()) {
    filteredOptions.value = ALUMNI_OPTIONS
    return
  }
  const scored = ALUMNI_OPTIONS.map(opt => ({ opt, score: fuzzyScore(query, opt) }))
    .filter(x => x.score > 0)
    .sort((a, b) => b.score - a.score)
    .slice(0, 10)
  filteredOptions.value = scored.map(x => x.opt)
  activeIdx.value = -1
}

function handleInput(e: Event) {
  const val = (e.target as HTMLInputElement).value
  inputValue.value = val
  emit('update:modelValue', '')
  filterAlumni(val)
  showDropdown.value = true
}

function handleFocus() {
  filterAlumni(inputValue.value)
  showDropdown.value = true
}

function handleKeydown(e: KeyboardEvent) {
  if (!showDropdown.value) return
  const len = filteredOptions.value.length

  if (e.key === 'ArrowDown') {
    e.preventDefault()
    activeIdx.value = (activeIdx.value + 1) % len
    scrollActiveIntoView()
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    activeIdx.value = (activeIdx.value - 1 + len) % len
    scrollActiveIntoView()
  } else if (e.key === 'Enter') {
    e.preventDefault()
    if (activeIdx.value >= 0 && activeIdx.value < len) {
      selectItem(filteredOptions.value[activeIdx.value])
    }
  } else if (e.key === 'Escape') {
    showDropdown.value = false
  }
}

function scrollActiveIntoView() {
  nextTick(() => {
    const el = dropdownRef.value?.querySelector('.ac-active') as HTMLElement
    el?.scrollIntoView({ block: 'nearest' })
  })
}

function selectItem(value: string) {
  inputValue.value = value.replace(/校友会$/, '')
  emit('update:modelValue', value)
  showDropdown.value = false
}

function handleBlur() {
  setTimeout(() => {
    showDropdown.value = false
  }, 150)
}

function highlightMatch(text: string, query: string): string {
  if (!query) return escapeHtml(text)
  let result = ''
  let remaining = text
  const q = query.toLowerCase()
  const t = text.toLowerCase()
  let idx = 0
  for (const ch of q) {
    const pos = t.indexOf(ch, idx)
    if (pos === -1) break
    result += escapeHtml(remaining.slice(0, pos - idx))
    result += `<span class="text-primary-600 font-semibold">${escapeHtml(text[pos])}</span>`
    remaining = text.slice(pos + 1)
    idx = pos + 1
  }
  result += escapeHtml(remaining)
  return result
}
</script>

<template>
  <div class="relative">
    <input
      type="text"
      name="alumni"
      :value="inputValue"
      placeholder="请输入校友会名称…"
      autocomplete="off"
      class="alumni-input w-full px-4 py-3 rounded-lg border border-gray-200 text-base focus:border-red-400 focus:ring-2 focus:ring-red-100 outline-none transition"
      style="font-size: 16px;"
      @input="handleInput"
      @focus="handleFocus"
      @keydown="handleKeydown"
      @blur="handleBlur"
    />
    <div
      v-show="showDropdown && filteredOptions.length > 0"
      ref="dropdownRef"
      class="absolute left-0 right-0 mt-1 bg-white border border-gray-200 rounded-lg shadow-lg z-50 max-h-56 overflow-y-auto"
    >
      <div
        v-for="(opt, i) in filteredOptions"
        :key="opt"
        class="px-4 py-2.5 text-sm cursor-pointer hover:bg-primary-50 transition-colors"
        :class="{ 'bg-primary-50 ac-active': i === activeIdx }"
        v-html="highlightMatch(opt, inputValue)"
        @mousedown.prevent="selectItem(opt)"
      />
    </div>
  </div>
</template>

<style scoped>
.alumni-input {
  -webkit-appearance: none;
}
</style>
