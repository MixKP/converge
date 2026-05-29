<script setup lang="ts">
  import { onMounted } from 'vue'
  import { useTeacherStore } from './stores/teacherStore'
  const teacherStore = useTeacherStore()
  onMounted(async () => {
    try {
      await teacherStore.fetchTeachers()
    } catch (e) {
      console.error('Failed to fetch teachers:', e)
    }
  })
</script>
<template>
  <router-view />
</template>
