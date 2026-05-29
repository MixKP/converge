<script setup lang="ts">
  import { onMounted } from 'vue'
  import { useTeacherStore } from './stores/teacherStore'
  const teacherStore = useTeacherStore()
  onMounted(() => {
    teacherStore.fetchTeachers()
  })
</script>
<template>
  <router-view />
</template>
