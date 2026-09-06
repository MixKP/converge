import { computed } from 'vue'
import { storeToRefs } from 'pinia'
import { useBookingStore } from '../stores/bookingStore'
import { useCartStore } from '../stores/cartStore'
import { fullCapacitySpans, overlapsFullCapacity } from '../utils/branchCapacity'
import type { EventInput } from '@fullcalendar/core'

export function useBranchCapacity() {
  const bookingStore = useBookingStore()
  const cartStore = useCartStore()
  const { branches, selectedBranchId, confirmedBookings } = storeToRefs(bookingStore)
  const { cartItems } = storeToRefs(cartStore)

  const selectedBranch = computed(
    () => branches.value.find((branch) => branch.id === selectedBranchId.value) ?? null
  )

  const capacitySpans = computed(() => {
    const branch = selectedBranch.value
    if (!branch) return []
    return fullCapacitySpans(branch.id, branch.capacity, [
      ...confirmedBookings.value,
      ...cartItems.value.filter((item) => item.status === 'pending'),
    ])
  })

  const capacityEvents = computed<EventInput[]>(() =>
    capacitySpans.value.map((span, index) => ({
      id: `capacity-${span.start}-${span.end}-${index}`,
      title: `At capacity (${span.occupancy}/${selectedBranch.value?.capacity})`,
      start: new Date(span.start).toISOString(),
      end: new Date(span.end).toISOString(),
      display: 'background',
      editable: false,
      backgroundColor: 'rgba(232, 165, 152, 0.28)',
      classNames: ['capacity-event'],
      extendedProps: { isCapacityWarning: true },
    }))
  )

  const isAtCapacity = (startTime: string, endTime: string): boolean =>
    overlapsFullCapacity(capacitySpans.value, startTime, endTime)

  return {
    capacityEvents,
    hasCapacityWarnings: computed(() => capacitySpans.value.length > 0),
    isAtCapacity,
  }
}
