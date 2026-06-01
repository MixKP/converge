<script setup lang="ts">
  import { computed, onMounted } from 'vue'
  import { useRouter } from 'vue-router'
  import { useBookingCart } from '../composables/useBookingCart'

  interface Props {
    title: string
    showCart?: boolean
  }
  const props = withDefaults(defineProps<Props>(), {
    showCart: true,
  })
  const router = useRouter()
  const { cartItems, fetchCartItems } = useBookingCart()
  const cartCount = computed(() => cartItems.value.length)
  const goToCart = () => {
    router.push('/booking/confirm')
  }
  onMounted(() => {
    fetchCartItems()
  })
</script>

<template>
  <div
    class="bg-(--paper-cream) items-center justify-center p-3 sm:p-6 h-screen flex overflow-hidden"
  >
    <div
      class="w-full bg-(--paper-white) rounded-sm shadow-elevated border border-(--border-subtle) p-4 sm:p-6 lg:p-8 transition-all h-full max-h-full flex flex-col"
    >
      <div class="flex items-center justify-between mb-4 sm:mb-6 border-b border-(--border-subtle) pb-3 sm:pb-4 shrink-0">
        <h2
          class="text-2xl sm:text-3xl lg:text-4xl font-semibold text-(--ink-primary) tracking-tight leading-tight"
          style="font-family: 'Playfair Display', Georgia, serif"
        >
          {{ title }}
        </h2>
        <button
          v-if="showCart"
          type="button"
          class="relative p-2 rounded-lg hover:bg-(--paper-cream) transition-all group"
          @click="goToCart"
        >
          <svg
            class="w-6 h-6 text-(--ink-primary) group-hover:text-(--accent-terracotta) transition-colors"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M16 11V7a4 4 0 00-8 0v4M5 9h14l1 12H4L5 9z"
            />
          </svg>
          <span
            v-if="cartCount > 0"
            class="absolute -top-1 -right-1 w-5 h-5 bg-(--accent-terracotta) text-white text-xs font-semibold rounded-full flex items-center justify-center"
          >
            {{ cartCount > 9 ? '9+' : cartCount }}
          </span>
        </button>
      </div>
      <slot />
    </div>
  </div>
</template>
