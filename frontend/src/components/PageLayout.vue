<script setup lang="ts">
  import AppNavbar from './AppNavbar.vue'
  import AppToasts from './AppToasts.vue'

  interface Props {
    title: string
    showCart?: boolean
    backTo?: string
    backLabel?: string
  }
  withDefaults(defineProps<Props>(), {
    showCart: true,
    backTo: undefined,
    backLabel: 'Back',
  })
</script>

<template>
  <div class="page-layout">
    <AppNavbar :title="title" :show-cart="showCart" :back-to="backTo" :back-label="backLabel" />
    <main class="page-main">
      <slot />
    </main>
    <AppToasts />
  </div>
</template>

<style scoped>
  .page-layout {
    min-height: 100vh;
    background: #fafafa;
    display: flex;
    flex-direction: column;
  }

  .page-main {
    flex: 1;
    width: 100%;
  }

  @media (max-width: 767px) {
    .page-layout {
      background: #ffffff;
    }
  }
</style>
