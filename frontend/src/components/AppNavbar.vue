<script setup lang="ts">
  import { computed, onMounted } from 'vue'
  import { useRouter } from 'vue-router'
  import { useBookingCart } from '../composables/useBookingCart'

  interface Props {
    title: string
    showCart?: boolean
  }

  withDefaults(defineProps<Props>(), {
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
  <header class="app-navbar">
    <div class="navbar-container">
      <div class="navbar-left">
        <div class="navbar-indicator"></div>
        <h1 class="navbar-title">{{ title }}</h1>
      </div>
      <button
        v-if="showCart"
        type="button"
        class="navbar-cart"
        :class="{ 'navbar-cart--active': cartCount > 0 }"
        aria-label="View cart"
        @click="goToCart"
      >
        <svg class="navbar-cart-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="1.5"
            d="M15.75 10.5V6a3.75 3.75 0 10-7.5 0v4.5m11.356-1.993l1.263 12c.07.665-.45 1.243-1.119 1.243H4.25a1.125 1.125 0 01-1.12-1.243l1.264-12A1.125 1.125 0 015.513 7.5h12.974c.576 0 1.059.435 1.119 1.007z"
          />
        </svg>
        <span v-if="cartCount > 0" class="navbar-cart-badge">{{
          cartCount > 9 ? '9+' : cartCount
        }}</span>
      </button>
    </div>
  </header>
</template>

<style scoped>
  .app-navbar {
    position: sticky;
    top: 0;
    z-index: 1000;
    background: #ffffff;
    border-bottom: 1px solid #e5e7eb;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
  }

  .navbar-container {
    width: 100%;
    padding: 1rem 1.5rem;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
  }

  .navbar-left {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    min-width: 0;
    flex: 1;
  }

  .navbar-indicator {
    width: 4px;
    height: 1.5rem;
    background: #3e4c7a;
    border-radius: 2px;
    flex-shrink: 0;
  }

  .navbar-title {
    font-size: 1.375rem;
    font-weight: 600;
    color: #111827;
    font-family:
      'Instrument Sans',
      'DM Sans',
      -apple-system,
      sans-serif;
    letter-spacing: -0.02em;
    line-height: 1.2;
  }

  .navbar-cart {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 2.5rem;
    height: 2.5rem;
    color: #6b7280;
    background: #f3f4f6;
    border: 1px solid transparent;
    border-radius: 8px;
    cursor: pointer;
    transition: all 0.15s ease;
    flex-shrink: 0;
  }

  .navbar-cart:hover {
    color: #3e4c7a;
    background: #e5e7eb;
  }

  .navbar-cart--active {
    color: #3e4c7a;
    background: rgba(62, 76, 122, 0.1);
    border-color: rgba(62, 76, 122, 0.2);
  }

  .navbar-cart--active:hover {
    background: rgba(62, 76, 122, 0.15);
  }

  .navbar-cart-icon {
    width: 1.25rem;
    height: 1.25rem;
    stroke-width: 1.5;
  }

  .navbar-cart-badge {
    position: absolute;
    top: -4px;
    right: -4px;
    display: flex;
    align-items: center;
    justify-content: center;
    min-width: 1.125rem;
    height: 1.125rem;
    padding: 0 0.375rem;
    font-size: 0.625rem;
    font-weight: 600;
    font-family: 'JetBrains Mono', 'SF Mono', monospace;
    color: #ffffff;
    background: #ef4444;
    border-radius: 9999px;
    border: 2px solid #ffffff;
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
  }

  /* Laptop/Tablet (768px - 1439px) */
  @media (min-width: 768px) and (max-width: 1439px) {
    .navbar-container {
      padding: 0.875rem 1.25rem;
    }

    .navbar-indicator {
      width: 3px;
      height: 1.25rem;
    }

    .navbar-title {
      font-size: 1.125rem;
    }

    .navbar-cart {
      width: 2.25rem;
      height: 2.25rem;
    }

    .navbar-cart-icon {
      width: 1.125rem;
      height: 1.125rem;
    }

    .navbar-cart-badge {
      top: -3px;
      right: -3px;
      min-width: 1rem;
      height: 1rem;
      font-size: 0.5625rem;
      padding: 0 0.25rem;
    }
  }

  /* Mobile (< 768px) */
  @media (max-width: 767px) {
    .navbar-container {
      padding: 0.75rem 1rem;
    }

    .navbar-indicator {
      width: 2px;
      height: 1rem;
      background: #111827;
    }

    .navbar-title {
      font-size: 1rem;
    }

    .navbar-cart {
      width: 2rem;
      height: 2rem;
      background: transparent;
      border-color: transparent;
      color: #111827;
    }

    .navbar-cart:hover {
      background: #f3f4f6;
    }

    .navbar-cart--active {
      background: transparent;
      border-color: transparent;
    }

    .navbar-cart--active:hover {
      background: #f3f4f6;
    }

    .navbar-cart-icon {
      width: 1.125rem;
      height: 1.125rem;
    }

    .navbar-cart-badge {
      top: -2px;
      right: -2px;
      min-width: 0.875rem;
      height: 0.875rem;
      font-size: 0.5rem;
      border-width: 1.5px;
    }
  }

  /* Small mobile (< 425px) */
  @media (max-width: 424px) {
    .navbar-container {
      padding: 0.625rem 0.875rem;
    }

    .navbar-title {
      font-size: 0.9375rem;
    }

    .navbar-cart {
      width: 1.875rem;
      height: 1.875rem;
    }

    .navbar-cart-icon {
      width: 1rem;
      height: 1rem;
    }
  }
</style>
