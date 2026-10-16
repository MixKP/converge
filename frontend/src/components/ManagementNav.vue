<script setup lang="ts">
  import { useRoute } from 'vue-router'
  import { Building2, Car, DatabaseBackup, GraduationCap, Users } from '@lucide/vue'

  const route = useRoute()

  const tabs = [
    { label: 'Teachers', path: '/manage', icon: GraduationCap },
    { label: 'Branches', path: '/manage/branches', icon: Building2 },
    { label: 'Commute', path: '/manage/commute', icon: Car },
    { label: 'Accounts', path: '/manage/accounts', icon: Users },
    { label: 'Backups', path: '/manage/backups', icon: DatabaseBackup },
  ]
</script>

<template>
  <nav aria-label="Management sections">
    <div class="mx-auto flex w-full max-w-6xl flex-col gap-2 px-4 pt-4 sm:px-6 lg:px-8">
      <p class="text-muted-foreground text-2xs font-semibold tracking-wider uppercase">Manage</p>

      <div class="flex flex-wrap gap-2">
        <RouterLink
          v-for="tab in tabs"
          :key="tab.path"
          :to="tab.path"
          :aria-current="route.path === tab.path ? 'page' : undefined"
          class="focus-visible:ring-ring/50 inline-flex items-center gap-2 rounded-md border px-3 py-1.5 text-sm font-medium transition-colors outline-none focus-visible:ring-3"
          :class="
            route.path === tab.path
              ? 'bg-primary text-primary-foreground border-transparent shadow-xs'
              : 'border-border text-muted-foreground hover:bg-accent hover:text-foreground'
          "
        >
          <component :is="tab.icon" class="size-4" />
          {{ tab.label }}
        </RouterLink>
      </div>
    </div>
  </nav>
</template>
