<script setup lang="ts">
  defineProps<{
    message: string
  }>()
</script>

<template>
  <div class="absolute inset-0 z-10 flex items-center justify-center bg-(--paper-white)/80">
    <div
      class="bg-(--paper-white) px-5 py-2.5 rounded-sm shadow-md border border-(--border-strong) text-sm font-medium text-(--ink-primary) flex items-center gap-2.5"
    >
      <svg
        class="w-4 h-4 text-(--accent-indigo)"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
      >
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          stroke-width="2"
          d="M15 15l-2 5L9 9l11 4-5 2zm0 0l5 5M7.188 2.239l.777 2.897M5.136 7.965l-2.898-.777M13.95 4.05l-2.122 2.122m-5.657 5.656l-2.12 2.122"
        ></path>
      </svg>
      {{ message }}
    </div>
  </div>
</template>
