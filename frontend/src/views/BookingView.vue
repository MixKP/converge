<script setup lang="ts">
  import { ref, watch, onMounted } from 'vue'
  import { CalendarDays, Pencil, Sparkles } from '@lucide/vue'
  import { useBooking } from '../composables/useBooking'
  import { useBookingContext } from '../composables/useBookingContext'
  import { useCart } from '../composables/useCart'
  import PageLayout from '../components/PageLayout.vue'
  import ContextBar from '../components/booking/ContextBar.vue'
  import ManualPath from '../components/booking/ManualPath.vue'
  import SmartPath from '../components/booking/SmartPath.vue'
  import EditPath from '../components/booking/EditPath.vue'
  import { Button } from '@/components/ui/button'
  import { Card, CardContent } from '@/components/ui/card'

  const {
    initialize,
    fetchSubjects,
    fetchBranches,
    fetchStudents,
    resetBookingState,
    selectedStudentId,
  } = useBooking()
  const { contextBlocker, contextComplete } = useBookingContext()
  const { loadCart } = useCart()

  type Path = 'manual' | 'smart' | 'edit'
  const path = ref<Path | null>(null)

  const selectPath = (next: Path) => {
    if (path.value === next) return
    path.value = next
    resetBookingState()
  }

  // Editing needs only the student; the subject and branch are for new bookings.
  watch(contextComplete, (complete) => {
    if (!complete && path.value !== 'edit') path.value = null
  })

  watch(selectedStudentId, () => {
    if (path.value === 'edit') path.value = null
  })

  onMounted(() => {
    loadCart()
    fetchSubjects()
    fetchBranches()
    fetchStudents()
  })

  initialize()
</script>

<template>
  <PageLayout title="Book a Session" back-to="/dashboard" back-label="Dashboard">
    <div class="mx-auto flex w-full max-w-7xl flex-col gap-5 px-4 py-6 sm:px-6 lg:px-8">
      <ContextBar />

      <template v-if="selectedStudentId !== null">
        <div class="flex gap-2">
          <Button
            :variant="path === 'manual' ? 'default' : 'outline'"
            class="flex-1 justify-start"
            :disabled="!contextComplete"
            @click="selectPath('manual')"
          >
            <CalendarDays class="size-4" />
            Pick a time myself
          </Button>
          <Button
            :variant="path === 'smart' ? 'default' : 'outline'"
            class="flex-1 justify-start"
            :disabled="!contextComplete"
            @click="selectPath('smart')"
          >
            <Sparkles class="size-4" />
            Find me a time
          </Button>
          <Button
            :variant="path === 'edit' ? 'default' : 'outline'"
            class="flex-1 justify-start"
            @click="selectPath('edit')"
          >
            <Pencil class="size-4" />
            Edit bookings
          </Button>
        </div>

        <ManualPath v-if="path === 'manual'" />
        <SmartPath v-else-if="path === 'smart'" />
        <EditPath v-else-if="path === 'edit'" />
        <Card v-else>
          <CardContent class="text-muted-foreground py-10 text-center text-sm">
            <template v-if="contextComplete">
              Choose how you want to book. Pick a time yourself to browse the calendar, let the
              engine match a teacher to the windows you give it, or edit classes already booked.
            </template>
            <template v-else>
              {{ contextBlocker }} Or edit the classes this student already has.
            </template>
          </CardContent>
        </Card>
      </template>
    </div>
  </PageLayout>
</template>
