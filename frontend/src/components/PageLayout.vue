<script setup lang="ts">
  interface Props {
    title: string
  }

  defineProps<Props>()
</script>

<template>
  <div class="bg-slate-50 items-center justify-center p-3 font-sans h-screen flex overflow-hidden">
    <div
      class="w-full max-w-5xl bg-white rounded-xl shadow-lg border border-slate-100 p-4 transition-all h-full max-h-full flex flex-col"
    >
      <div class="mb-3 border-b border-slate-100 pb-2 shrink-0">
        <h2 class="text-xl font-bold text-slate-800">{{ title }}</h2>
      </div>
      <slot />
    </div>
  </div>
</template>
