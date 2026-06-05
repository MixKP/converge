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
  <div class="page-layout">
    <div class="page-container">
      <div class="page-header">
        <div class="header-left">
          <div class="header-indicator"></div>
          <h1 class="header-title">{{ title }}</h1>
        </div>
        <button
          v-if="showCart"
          type="button"
          class="cart-button"
          :class="{ 'cart-button--has-items': cartCount > 0 }"
          @click="goToCart"
        >
          <svg class="cart-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="1.5"
              d="M15.75 10.5V6a3.75 3.75 0 10-7.5 0v4.5m11.356-1.993l1.263 12c.07.665-.45 1.243-1.119 1.243H4.25a1.125 1.125 0 01-1.12-1.243l1.264-12A1.125 1.125 0 015.513 7.5h12.974c.576 0 1.059.435 1.119 1.007z"
            />
          </svg>
          <span v-if="cartCount > 0" class="cart-badge">{{
            cartCount > 9 ? '9+' : cartCount
          }}</span>
        </button>
      </div>
      <div class="page-content">
        <slot />
      </div>
    </div>
  </div>
</template>

<style scoped>
  .page-layout {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 100vh;
    padding: 1rem;
    background: var(--bg-cream);
  }

  .page-container {
    width: 100%;
    max-width: 1400px;
    height: calc(100vh - 2rem);
    max-height: 900px;
    background: var(--bg-card);
    border-radius: 8px;
    border: 1px solid var(--border-medium);
    box-shadow: var(--shadow-elevated);
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .page-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1.25rem 1.5rem;
    border-bottom: 1px solid var(--border-medium);
    flex-shrink: 0;
    position: relative;
  }

  .page-header::after {
    content: '';
    position: absolute;
    bottom: 0;
    left: 1.5rem;
    right: 1.5rem;
    height: 1px;
    background: linear-gradient(
      90deg,
      transparent,
      var(--border-technical) 30%,
      var(--border-technical) 70%,
      transparent
    );
  }

  .header-left {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .header-indicator {
    width: 3px;
    height: 1.5rem;
    background: var(--primary-indigo);
    border-radius: 1px;
  }

  .header-title {
    font-size: 1.5rem;
    font-weight: 600;
    color: var(--text-primary);
    font-family: 'Instrument Sans', sans-serif;
    letter-spacing: -0.02em;
  }

  .cart-button {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 2.5rem;
    height: 2.5rem;
    color: var(--text-secondary);
    background: var(--bg-subtle);
    border: 1px solid var(--border-medium);
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
  }

  .cart-button:hover {
    color: var(--primary-indigo);
    border-color: var(--primary-indigo);
    transform: translateY(-2px);
  }

  .cart-button--has-items {
    color: var(--primary-indigo);
    background: rgba(62, 76, 122, 0.08);
    border-color: var(--primary-indigo);
  }

  .cart-icon {
    width: 1.25rem;
    height: 1.25rem;
  }

  .cart-badge {
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
    font-family: 'JetBrains Mono', monospace;
    color: white;
    background: var(--accent-coral);
    border-radius: 6px;
    border: 2px solid var(--bg-card);
  }

  .page-content {
    flex: 1;
    padding: 1.25rem 1.5rem;
    overflow-y: auto;
    overflow-x: hidden;
  }

  /* Custom scrollbar */
  .page-content::-webkit-scrollbar {
    width: 6px;
  }

  .page-content::-webkit-scrollbar-track {
    background: transparent;
  }

  .page-content::-webkit-scrollbar-thumb {
    background: var(--border-strong);
    border-radius: 2px;
  }

  .page-content::-webkit-scrollbar-thumb:hover {
    background: var(--text-secondary);
  }

  @media (max-width: 640px) {
    .page-layout {
      padding: 0;
      background: var(--bg-cream);
    }

    .page-container {
      height: 100vh;
      max-height: none;
      border-radius: 0;
      border: none;
      box-shadow: none;
      background: var(--bg-cream);
    }

    .page-header {
      padding: 0.75rem 1rem;
      background: var(--bg-card);
      border-bottom: 1px solid var(--border-subtle);
      position: sticky;
      top: 0;
      z-index: 10;
    }

    .page-header::after {
      display: none;
    }

    .header-left {
      gap: 0.5rem;
    }

    .header-indicator {
      width: 2px;
      height: 1rem;
      border-radius: 1px;
      background: var(--text-primary);
    }

    .header-title {
      font-size: 0.9375rem;
      font-weight: 600;
    }

    .cart-button {
      width: 1.75rem;
      height: 1.75rem;
      border-radius: 6px;
      background: transparent;
      border-color: transparent;
      color: var(--text-primary);
    }

    .cart-button:hover {
      background: var(--bg-subtle);
      border-color: transparent;
      transform: none;
    }

    .cart-badge {
      top: -2px;
      right: -2px;
      min-width: 0.75rem;
      height: 0.75rem;
      font-size: 0.4375rem;
      border-width: 1px;
    }

    .cart-icon {
      width: 1rem;
      height: 1rem;
    }

    .page-content {
      padding: 0;
      overflow-y: auto;
      overflow-x: hidden;
      position: relative;
      z-index: 1;
      -webkit-overflow-scrolling: touch;
    }

    /* Custom scrollbar for mobile */
    .page-content::-webkit-scrollbar {
      width: 0;
    }
  }

  @media (max-width: 425px) {
    .page-header {
      padding: 0.625rem 1rem;
    }

    .header-title {
      font-size: 0.875rem;
    }

    .cart-button {
      width: 1.625rem;
      height: 1.625rem;
    }

    .cart-icon {
      width: 0.9375rem;
      height: 0.9375rem;
    }
  }
</style>
