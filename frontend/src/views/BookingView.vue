<script setup lang="ts">
  import { ref, watch, onMounted } from 'vue'
  import { CalendarDays, Sparkles } from '@lucide/vue'
  import { useBooking } from '../composables/useBooking'
  import { useBookingContext } from '../composables/useBookingContext'
  import { useCart } from '../composables/useCart'
  import { useNotification } from '../composables/useNotification'
  import PageLayout from '../components/PageLayout.vue'
  import ContextBar from '../components/booking/ContextBar.vue'
  import ManualPath from '../components/booking/ManualPath.vue'
  import SmartPath from '../components/booking/SmartPath.vue'
  import { Button } from '@/components/ui/button'
  import { Card, CardContent } from '@/components/ui/card'

  const { initialize, fetchSubjects, fetchBranches, fetchStudents, resetBookingState } =
    useBooking()
  const { contextComplete } = useBookingContext()
  const { loadCart } = useCart()
  const { successMessage, errorMessage } = useNotification()

  type Path = 'manual' | 'smart'
  const path = ref<Path | null>(null)

  /* Both paths share the context bar, so switching keeps student, subject and
   * branch. Only the mode-specific results are dropped. */
  const selectPath = (next: Path) => {
    if (path.value === next) return
    path.value = next
    resetBookingState()
  }

  // Re-opening the context bar to change the student or branch invalidates any
  // suggestions already on screen, so the choice is asked again.
  watch(contextComplete, (complete) => {
    if (!complete) path.value = null
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
    <!-- Desktop-first: the week grid is the primary surface, so it gets the
         room. The forms above it are grids and simply breathe wider. -->
    <div class="mx-auto flex w-full max-w-7xl flex-col gap-5 px-4 py-6 sm:px-6 lg:px-8">
      <ContextBar />

      <template v-if="contextComplete">
        <div class="flex gap-2">
          <Button
            :variant="path === 'manual' ? 'default' : 'outline'"
            class="flex-1 justify-start"
            @click="selectPath('manual')"
          >
            <CalendarDays class="size-4" />
            Pick a time myself
          </Button>
          <Button
            :variant="path === 'smart' ? 'default' : 'outline'"
            class="flex-1 justify-start"
            @click="selectPath('smart')"
          >
            <Sparkles class="size-4" />
            Find me a time
          </Button>
        </div>

        <ManualPath v-if="path === 'manual'" />
        <SmartPath v-else-if="path === 'smart'" />
        <Card v-else>
          <CardContent class="text-muted-foreground py-10 text-center text-sm">
            Choose how you want to book. Pick a time yourself to browse the calendar, or let the
            engine match a teacher to the windows you give it.
          </CardContent>
        </Card>
      </template>

      <div class="flex justify-end">
        <RouterLink to="/booking/confirm">
          <Button variant="ghost">Review cart</Button>
        </RouterLink>
      </div>

      <div
        v-if="successMessage"
        class="bg-primary text-primary-foreground fixed right-6 bottom-6 z-50 rounded-lg px-4 py-3 text-sm shadow-lg"
        role="status"
        aria-live="polite"
      >
        {{ successMessage }}
      </div>
      <div
        v-if="errorMessage"
        class="bg-destructive fixed right-6 bottom-6 z-50 rounded-lg px-4 py-3 text-sm text-white shadow-lg"
        role="alert"
        aria-live="assertive"
      >
        {{ errorMessage }}
      </div>
    </div>
  </PageLayout>
</template>
