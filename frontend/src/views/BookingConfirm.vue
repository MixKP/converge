<script setup lang="ts">
  import { computed, onMounted } from 'vue'
  import { useRouter } from 'vue-router'
  import { useBookingCart } from '../composables/useBookingCart'
  import PageLayout from '../components/PageLayout.vue'
  import BookingCartItem from '../components/BookingCartItem.vue'

  const router = useRouter()
  const {
    cartItems,
    isLoading,
    errorMessage,
    successMessage,
    fetchCartItems,
    removeItem,
    clearCart,
    submitBookings,
  } = useBookingCart()

  onMounted(() => {
    fetchCartItems()
  })

  const totalAmount = computed(() => {
    return cartItems.value.length * 500
  })

  const handleRemove = (id: number) => {
    removeItem(id)
  }

  const handleSubmit = async () => {
    await submitBookings()
    if (!errorMessage.value) {
      setTimeout(() => {
        router.push('/')
      }, 2000)
    }
  }

  const goBack = () => {
    router.push('/booking')
  }
</script>

<template>
  <PageLayout title="Confirm Bookings">
    <div class="max-w-4xl mx-auto">
      <div v-if="cartItems.length === 0 && !isLoading" class="text-center py-16">
        <svg
          class="w-24 h-24 mx-auto mb-6 text-(--border-strong)"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="1.5"
            d="M16 11V7a4 4 0 00-8 0v4M5 9h14l1 12H4L5 9z"
          />
        </svg>
        <h3 class="text-xl font-semibold text-(--ink-primary) mb-2">Your cart is empty</h3>
        <p class="text-(--text-secondary) mb-6">Add bookings to your cart to get started</p>
        <button
          type="button"
          class="px-6 py-2 bg-(--accent-terracotta) text-white rounded-lg hover:bg-(--accent-terracotta-dark) transition-all font-medium"
          @click="goBack"
        >
          Browse Available Sessions
        </button>
      </div>

      <div v-else class="space-y-6">
        <div class="flex items-center justify-between">
          <h2 class="text-lg font-semibold text-(--ink-primary)">
            {{ cartItems.length }} Booking{{ cartItems.length > 1 ? 's' : '' }}
          </h2>
          <button
            v-if="cartItems.length > 0"
            type="button"
            class="text-sm text-red-600 hover:text-red-700 font-medium"
            @click="clearCart"
          >
            Clear All
          </button>
        </div>

        <div class="space-y-4">
          <BookingCartItem
            v-for="item in cartItems"
            :key="item.id"
            :item="item"
            @remove="handleRemove"
          />
        </div>

        <div class="bg-(--paper-cream) p-4 rounded-lg border border-(--border-subtle)">
          <div class="flex items-center justify-between mb-4">
            <span class="text-(--ink-primary) font-medium">Total Sessions</span>
            <span class="text-(--ink-primary) font-semibold">{{ cartItems.length }}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-(--ink-primary) font-medium">Estimated Total</span>
            <span class="text-xl font-bold text-(--accent-terracotta)">${{ totalAmount }}</span>
          </div>
        </div>

        <div class="flex gap-3">
          <button
            type="button"
            class="flex-1 px-4 py-3 text-(--ink-primary) bg-white border border-(--border-strong) rounded-lg hover:bg-(--paper-cream) transition-all font-medium"
            @click="goBack"
          >
            Add More Bookings
          </button>
          <button
            type="button"
            :disabled="isLoading || cartItems.length === 0"
            class="flex-1 px-4 py-3 text-white bg-(--accent-terracotta) rounded-lg hover:bg-(--accent-terracotta-dark) disabled:opacity-50 disabled:cursor-not-allowed transition-all font-semibold"
            @click="handleSubmit"
          >
            {{ isLoading ? 'Confirming...' : 'Confirm All Bookings' }}
          </button>
        </div>
      </div>

      <div
        v-if="successMessage"
        class="fixed bottom-4 right-4 bg-green-600 text-white px-4 py-3 rounded-lg shadow-lg flex items-center gap-2 animate-slide-up"
      >
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M5 13l4 4L19 7"
          />
        </svg>
        {{ successMessage }}
      </div>

      <div
        v-if="errorMessage"
        class="fixed bottom-4 right-4 bg-red-600 text-white px-4 py-3 rounded-lg shadow-lg flex items-center gap-2 animate-slide-up"
      >
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M6 18L18 6M6 6l12 12"
          />
        </svg>
        {{ errorMessage }}
        <button class="ml-2 text-white/80 hover:text-white" @click="errorMessage = ''">×</button>
      </div>
    </div>
  </PageLayout>
</template>

<style scoped>
  @keyframes slide-up {
    from {
      opacity: 0;
      transform: translateY(20px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .animate-slide-up {
    animation: slide-up 0.3s ease-out;
  }
</style>
