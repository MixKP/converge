<script setup lang="ts">
  interface Props {
    title: string
  }

  defineProps<Props>()
</script>

<template>
  <div class="bg-(--paper-cream) items-center justify-center p-4 h-screen flex overflow-hidden">
    <div
      class="w-full max-w-5xl bg-(--paper-white) rounded-sm shadow-lg border border-(--border-subtle) p-5 transition-all h-full max-h-full flex flex-col"
    >
      <div class="mb-4 border-b-2 border-(--border-subtle) pb-3 shrink-0">
        <h2
          class="text-2xl font-semibold text-(--ink-primary) tracking-tight"
          style="font-family: 'Playfair Display', Georgia, serif"
        >
          {{ title }}
        </h2>
      </div>
      <slot />
    </div>
  </div>
</template>
