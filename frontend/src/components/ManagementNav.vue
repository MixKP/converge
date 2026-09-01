<script setup lang="ts">
  import { useRoute } from 'vue-router'
  import { computed } from 'vue'
  import { CalendarPlus } from '@lucide/vue'

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

    <!-- Booking is a workflow rather than a management section, so it sits
         apart from the tabs instead of becoming one that never activates. -->
    <router-link to="/booking" class="manage-nav-action">
      <CalendarPlus class="manage-nav-action-icon" aria-hidden="true" />
      Book a session
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

  .manage-nav-action {
    display: inline-flex;
    align-items: center;
    gap: 0.375rem;
    margin-left: auto;
    padding: 0.5rem 1rem;
    font-family: Inter, sans-serif;
    font-size: 0.875rem;
    font-weight: 500;
    color: #fff;
    text-decoration: none;
    /* Deeper than the active tab's --primary-indigo so the two dark pills stay
       distinguishable. --accent-sage was too light to carry white text — it
       read as a disabled control. */
    background: var(--primary-navy-deep);
    border-radius: 9999px;
    transition: all 0.15s;
  }

  .manage-nav-action:hover {
    background: var(--primary-navy);
  }

  .manage-nav-action:focus-visible {
    outline: 2px solid var(--primary-indigo);
    outline-offset: 2px;
  }

  .manage-nav-action-icon {
    width: 1rem;
    height: 1rem;
  }

  @media (max-width: 767px) {
    .manage-nav {
      padding: 1rem 1rem 0;
    }

    /* Wrapping to its own row beats being squeezed against the last tab. */
    .manage-nav-action {
      margin-left: 0;
    }
  }
</style>
