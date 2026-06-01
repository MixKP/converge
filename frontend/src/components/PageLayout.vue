<script setup lang="ts">
  interface Props {
    title: string
  }
  defineProps<Props>()
</script>
<template>
  <div
    class="bg-(--paper-cream) items-center justify-center p-3 sm:p-6 h-screen flex overflow-hidden"
  >
    <div
      class="w-full bg-(--paper-white) rounded-sm shadow-elevated border border-(--border-subtle) p-4 sm:p-6 lg:p-8 transition-all h-full max-h-full flex flex-col"
    >
      <div class="mb-4 sm:mb-6 border-b border-(--border-subtle) pb-3 sm:pb-4 shrink-0">
        <h2
          class="text-2xl sm:text-3xl lg:text-4xl font-semibold text-(--ink-primary) tracking-tight leading-tight"
          style="font-family: 'Playfair Display', Georgia, serif"
        >
          {{ title }}
        </h2>
      </div>
      <slot />
    </div>
  </div>
</template>
