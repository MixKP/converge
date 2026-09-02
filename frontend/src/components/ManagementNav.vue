<script setup lang="ts">
  import { useRoute } from 'vue-router'
  import { computed } from 'vue'

  const route = useRoute()
  const currentPath = computed(() => route.path)

  const tabs = [
    { label: 'Teachers', path: '/manage' },
    { label: 'Branches', path: '/manage/branches' },
    { label: 'Commute', path: '/manage/commute' },
    { label: 'Accounts', path: '/manage/accounts' },
    { label: 'Schedule', path: '/manage/schedule' },
  ]
</script>

<template>
  <!-- Section tabs only. Booking lives on the dashboard, and the way back out
       is the navbar's back link, which every page shares. -->
  <nav class="manage-nav" aria-label="Management navigation">
    <router-link
      v-for="tab in tabs"
      :key="tab.path"
      :to="tab.path"
      class="manage-nav-tab"
      :class="{ 'manage-nav-tab--active': currentPath === tab.path }"
    >
      {{ tab.label }}
    </router-link>
  </nav>
</template>

<style scoped>
  .manage-nav {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.25rem;
    padding: 1rem 1.5rem 0;
    max-width: 64rem;
    margin: 0 auto;
  }

  .manage-nav-tab {
    padding: 0.5rem 1.25rem;
    font-family: Inter, sans-serif;
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text-secondary);
    text-decoration: none;
    border-radius: 9999px;
    transition: all 0.15s;
  }

  .manage-nav-tab:hover {
    color: var(--text-primary);
    background: var(--bg-subtle);
  }

  .manage-nav-tab--active {
    color: #fff;
    background: var(--primary-indigo);
  }

  .manage-nav-tab--active:hover {
    color: #fff;
    background: var(--primary-indigo);
  }

  @media (max-width: 767px) {
    .manage-nav {
      padding: 1rem 1rem 0;
    }
  }
</style>
