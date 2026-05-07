<script setup lang="ts">
  import { ref } from 'vue'

  const selectedTeacher = ref()
  const teachers = ref([
    { name: 'Teacher A' },
    { name: 'Teacher B' },
    { name: 'Teacher C' },
    { name: 'Teacher D' },
  ])

  // Logic to handle the submission
  const handleSubmit = () => {
    console.log('Selected Teacher:', selectedTeacher.value)
  }
</script>

<template>
  <div class="min-h-screen bg-gray-50 flex items-center justify-center p-6">
    <div class="w-full max-w-md bg-white rounded-2xl shadow-lg p-8">
      <h2 class="text-2xl font-bold mb-6 text-center text-gray-800">Submit Availability</h2>

      <form @submit.prevent="handleSubmit" class="flex flex-col gap-5">
        <div class="flex flex-col gap-2">
          <select
            v-model="selectedTeacher"
            class="w-full border border-gray-300 text-gray-700 rounded-lg px-4 py-3 focus:outline-none focus:border-blue-500 focus:ring-2 focus:ring-blue-200 transition-colors cursor-pointer bg-white"
          >
            <option :value="undefined" disabled>Select a Teacher</option>

            <option v-for="teacher in teachers" :key="teacher.name" :value="teacher">
              {{ teacher.name }}
            </option>
          </select>
        </div>

        <button
          type="submit"
          class="mt-4 w-full bg-gray-600 hover:bg-gray-900 text-white font-bold py-3 rounded-xl shadow-sm transition-all active:scale-95"
        >
          Submit Data
        </button>
      </form>
    </div>
  </div>
</template>
