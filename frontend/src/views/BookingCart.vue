<script setup lang="ts">
  import { onMounted, ref } from 'vue'
  import { useRouter } from 'vue-router'
  import { useCart } from '../composables/useCart'
  import { useNotification } from '../composables/useNotification'
  import PageLayout from '../components/PageLayout.vue'

  const router = useRouter()
  const { cartItems, isConfirming, loadCart, removeCartItem, clearCart, confirmBookings } =
    useCart()
  const { successMessage, errorMessage } = useNotification()

  const showClearConfirm = ref(false)

  onMounted(() => {
    loadCart()
  })

  const handleRemove = (id: number) => {
    removeCartItem(id)
  }

  const handleClearAll = () => {
    if (cartItems.value.length === 0) return
    showClearConfirm.value = true
  }

  const confirmClearAll = () => {
    clearCart()
    showClearConfirm.value = false
  }

  const handleSubmit = async () => {
    await confirmBookings()
    if (!errorMessage.value) {
      setTimeout(() => {
        router.push('/booking')
      }, 2000)
    }
  }

  const goBack = () => {
    router.push('/booking')
  }

  const formatDate = (date: string | Date) => {
    return new Date(date).toLocaleDateString('en-US', {
      weekday: 'long',
      month: 'long',
      day: 'numeric',
    })
  }

  const formatTime = (date: string | Date) => {
    return new Date(date).toLocaleTimeString('en-US', {
      hour: '2-digit',
      minute: '2-digit',
      hour12: false,
    })
  }
</script>

<template>
  <PageLayout title="Confirm Bookings">
    <div class="relative mx-auto mt-3 max-w-175">
      <div
        v-if="cartItems.length === 0 && !isConfirming"
        class="anim-fade-in flex flex-col items-center justify-center px-8 py-16 text-center"
      >
        <div
          class="mb-6 flex h-16 w-16 items-center justify-center rounded-lg bg-(--bg-subtle) text-(--text-muted)"
        >
          <svg class="h-7 w-7" viewBox="0 0 24 24" fill="none" stroke="currentColor">
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="1.5"
              d="M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5m8.25 3v6.75m0 0l-3-3m3 3l3-3M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z"
            />
          </svg>
        </div>
        <h2
          class="mb-1.5 font-['Instrument_Sans',sans-serif] text-xl font-semibold text-(--text-primary)"
        >
          Your cart is empty
        </h2>
        <p class="mb-6 text-sm text-(--text-secondary)">Add sessions to begin booking</p>
        <button
          type="button"
          class="inline-flex cursor-pointer items-center gap-2 rounded-md bg-(--primary-indigo) px-6 py-3 text-sm font-medium text-white shadow-[0_2px_8px_rgba(62,76,122,0.2)] transition-all duration-300 ease-[cubic-bezier(0.34,1.56,0.64,1)] hover:-translate-y-0.5 hover:bg-(--primary-indigo-deep)"
          @click="goBack"
        >
          Browse Sessions
          <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor">
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M13.5 4.5L21 12m0 0l-7.5 7.5M21 12H3"
            />
          </svg>
        </button>
      </div>

      <div v-else class="anim-fade-up flex flex-col gap-6">
        <!-- Header with Clear All -->
        <div class="flex items-center justify-between border-b border-(--border-medium) pb-4">
          <div class="flex flex-col">
            <span class="text-base font-medium text-(--text-primary)"
              >{{ cartItems.length }} session{{ cartItems.length !== 1 ? 's' : '' }}</span
            >
          </div>
          <button
            v-if="cartItems.length > 0"
            type="button"
            class="cursor-pointer rounded-md border border-(--border-medium) bg-transparent px-4 py-2 font-[Inter,sans-serif] text-sm font-medium text-(--accent-coral) transition-all duration-200 hover:border-(--accent-coral) hover:bg-[rgba(201,109,93,0.08)]"
            @click="handleClearAll"
          >
            Clear All
          </button>
        </div>

        <!-- Booking Cards -->
        <div class="flex flex-col gap-3">
          <div
            v-for="(item, index) in cartItems"
            :key="item.id"
            class="anim-card-in relative rounded-lg border border-(--border-medium) bg-(--bg-card) p-4 transition-[background] duration-200 hover:bg-(--bg-subtle)"
            :style="{ animationDelay: `${index * 50}ms` }"
          >
            <button
              type="button"
              class="absolute top-4 right-4 flex h-7 w-7 cursor-pointer items-center justify-center rounded bg-transparent text-(--text-muted) transition-all duration-200 hover:bg-[rgba(201,109,93,0.08)] hover:text-(--accent-coral)"
              aria-label="Remove session"
              @click="handleRemove(item.id)"
            >
              <svg class="h-3.5 w-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M6 18L18 6M6 6l12 12"
                />
              </svg>
            </button>

            <div class="flex flex-col gap-1.5 pr-8">
              <span
                class="font-['Instrument_Sans',sans-serif] text-base font-semibold text-(--text-primary)"
                >{{ item.subject_name }}</span
              >
              <span class="font-['JetBrains_Mono',monospace] text-sm text-(--text-secondary)"
                >Date: {{ formatDate(item.start_time) }} · {{ formatTime(item.start_time) }}-{{
                  formatTime(item.end_time)
                }}</span
              >
              <span class="text-sm text-(--text-secondary)">Student: {{ item.student_name }}</span>
              <span class="text-sm text-(--text-secondary)">Teacher: {{ item.teacher_name }}</span>
              <span class="text-sm text-(--text-secondary)">Branch: {{ item.branch_name }}</span>
            </div>
          </div>
        </div>

        <!-- Action Buttons -->
        <div class="flex gap-3 max-sm:flex-col">
          <button
            type="button"
            class="flex flex-1 cursor-pointer items-center justify-center gap-2 rounded-md border border-(--border-medium) bg-(--bg-card) px-5 py-3.5 font-[Inter,sans-serif] text-sm font-medium text-(--text-primary) transition-all duration-300 ease-[cubic-bezier(0.34,1.56,0.64,1)] hover:bg-(--bg-subtle) max-sm:w-full"
            @click="goBack"
          >
            <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor">
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M10.5 19.5L3 12m0 0l7.5-7.5M3 12h18"
              />
            </svg>
            Add More
          </button>
          <button
            type="button"
            :disabled="isConfirming || cartItems.length === 0"
            class="flex flex-1 cursor-pointer items-center justify-center gap-2 rounded-md border-none bg-(--accent-sage) px-5 py-3.5 font-[Inter,sans-serif] text-sm font-medium text-white shadow-[0_2px_8px_rgba(122,139,109,0.2)] transition-all duration-300 ease-[cubic-bezier(0.34,1.56,0.64,1)] hover:enabled:-translate-y-0.5 hover:enabled:shadow-[0_4px_16px_rgba(122,139,109,0.3)] hover:enabled:brightness-90 disabled:translate-y-0 disabled:cursor-not-allowed disabled:opacity-40 disabled:shadow-none max-sm:w-full"
            @click="handleSubmit"
          >
            <span v-if="!isConfirming">Confirm Booking</span>
            <span v-else>Processing...</span>
            <svg
              v-if="!isConfirming"
              class="h-4 w-4"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M13.5 4.5L21 12m0 0l-7.5 7.5M21 12H3"
              />
            </svg>
          </button>
        </div>
      </div>

      <!-- Clear All Confirmation Dialog -->
      <div
        v-if="showClearConfirm"
        class="anim-overlay-in fixed inset-0 z-200 flex items-center justify-center bg-[rgba(26,28,35,0.5)]"
        @click.self="showClearConfirm = false"
      >
        <div
          class="anim-dialog-in mx-4 max-w-100 rounded-lg border border-t-2 border-(--border-medium) border-t-(--accent-coral) bg-(--bg-card) p-6"
          role="dialog"
          aria-modal="true"
          aria-labelledby="clear-confirm-title"
        >
          <h3
            id="clear-confirm-title"
            class="m-0 mb-2 font-['Instrument_Sans',sans-serif] text-base font-semibold text-(--text-primary)"
          >
            Clear all sessions?
          </h3>
          <p class="m-0 mb-5 text-sm leading-normal text-(--text-secondary)">
            This will remove all sessions from your cart. This action cannot be undone.
          </p>
          <div class="flex justify-end gap-3">
            <button
              type="button"
              class="cursor-pointer rounded-md border border-(--border-medium) bg-(--bg-subtle) px-4 py-2.5 font-[Inter,sans-serif] text-sm font-medium text-(--text-primary) transition-all duration-200 hover:bg-(--border-subtle)"
              @click="showClearConfirm = false"
            >
              Cancel
            </button>
            <button
              type="button"
              class="cursor-pointer rounded-md border-none bg-(--accent-coral) px-4 py-2.5 font-[Inter,sans-serif] text-sm font-medium text-white transition-all duration-200 hover:bg-[#b85c4e]"
              @click="confirmClearAll"
            >
              Clear All
            </button>
          </div>
        </div>
      </div>

      <!-- Toast Notifications -->
      <div
        v-if="successMessage"
        class="anim-toast-in fixed right-8 bottom-8 z-100 flex items-center gap-2.5 rounded-md bg-(--accent-sage) px-5 py-3.5 font-[Inter,sans-serif] text-sm text-white shadow-(--shadow-elevated) max-sm:right-4 max-sm:left-4"
      >
        <svg class="h-4.5 w-4.5 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
          />
        </svg>
        {{ successMessage }}
      </div>

      <div
        v-if="errorMessage"
        class="anim-toast-in fixed right-8 bottom-8 z-100 flex items-center gap-2.5 rounded-md bg-(--accent-coral) px-5 py-3.5 font-[Inter,sans-serif] text-sm text-white shadow-(--shadow-elevated) max-sm:right-4 max-sm:left-4"
      >
        <svg class="h-4.5 w-4.5 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z"
          />
        </svg>
        {{ errorMessage }}
        <button
          class="ml-2.5 flex h-5 w-5 cursor-pointer items-center justify-center border-none bg-transparent text-xl text-[rgba(255,255,255,0.8)]"
          aria-label="Dismiss message"
          @click="errorMessage = ''"
        >
          ×
        </button>
      </div>
    </div>
  </PageLayout>
</template>

<style scoped>
  @keyframes cart-fade-in {
    from {
      opacity: 0;
    }
    to {
      opacity: 1;
    }
  }

  @keyframes cart-fade-up {
    from {
      opacity: 0;
      transform: translateY(12px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  @keyframes cart-card-in {
    from {
      opacity: 0;
      transform: translateY(8px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  @keyframes cart-dialog-in {
    from {
      opacity: 0;
      transform: scale(0.95);
    }
    to {
      opacity: 1;
      transform: scale(1);
    }
  }

  @keyframes cart-toast-in {
    from {
      opacity: 0;
      transform: translateY(16px) scale(0.95);
    }
    to {
      opacity: 1;
      transform: translateY(0) scale(1);
    }
  }

  .anim-fade-in {
    animation: cart-fade-in 0.4s ease-out;
  }

  .anim-fade-up {
    animation: cart-fade-up 0.4s ease-out;
  }

  .anim-card-in {
    animation: cart-card-in 0.3s cubic-bezier(0.34, 1.56, 0.64, 1) backwards;
  }

  .anim-overlay-in {
    animation: cart-fade-in 0.2s ease-out;
  }

  .anim-dialog-in {
    animation: cart-dialog-in 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
  }

  .anim-toast-in {
    animation: cart-toast-in 0.4s cubic-bezier(0.34, 1.56, 0.64, 1);
  }
</style>
