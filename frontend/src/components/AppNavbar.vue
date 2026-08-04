<script setup lang="ts">
  import { computed, onMounted } from 'vue'
  import { useRouter } from 'vue-router'
  import { useCart } from '../composables/useCart'
  import { useAuthStore } from '../stores/authStore'

  interface Props {
    title: string
    showCart?: boolean
  }

  withDefaults(defineProps<Props>(), {
    showCart: true,
  })

  const router = useRouter()
  const { cartItems, loadCart } = useCart()
  const auth = useAuthStore()
  const cartCount = computed(() => cartItems.value.length)

  const goToCart = () => {
    router.push('/booking/confirm')
  }

  const handleLogout = () => {
    auth.logout()
    // loadCart re-reads the now-cleared storage so the badge does not keep
    // showing the previous operator's item count.
    loadCart()
    router.replace('/login')
  }

  onMounted(() => {
    loadCart()
  })
</script>

<template>
  <header class="app-navbar">
    <div class="navbar-container">
      <div class="navbar-left">
        <div class="navbar-indicator"></div>
        <h1 class="navbar-title">{{ title }}</h1>
      </div>
      <div class="navbar-right">
        <button
          v-if="showCart"
          type="button"
          class="navbar-cart"
          :class="{ 'navbar-cart--active': cartCount > 0 }"
          aria-label="View cart"
          @click="goToCart"
        >
          <svg class="navbar-cart-icon" viewBox="0 0 24 24" fill="currentColor">
            <path
              d="m21,5H3c-.55,0-1,.45-1,1v3.55c0,.48.33.89.8.98.69.14,1.2.76,1.2,1.47s-.5,1.33-1.2,1.47c-.47.09-.8.5-.8.98v3.55c0,.55.45,1,1,1h18c.55,0,1-.45,1-1v-3.55c0-.48-.33-.89-.8-.98-.69-.14-1.2-.76-1.2-1.47s.5-1.33,1.2-1.47c.47-.09.8-.5.8-.98v-3.55c0-.55-.45-1-1-1Zm-1,3.84c-1.2.57-2,1.79-2,3.16s.8,2.59,2,3.16v1.84h-4v-2h-1v2H4v-1.84c1.2-.57,2-1.79,2-3.16s-.8-2.59-2-3.16v-1.84h11v1h1v-1h4v1.84Z"
            />
            <path d="M15 9H16V11H15z" />
            <path d="M15 12H16V14H15z" />
          </svg>
          <span v-if="cartCount > 0" class="navbar-cart-badge">{{
            cartCount > 9 ? '9+' : cartCount
          }}</span>
        </button>

        <template v-if="auth.isAuthenticated">
          <span class="navbar-user">
            {{ auth.user?.name }}
            <span class="navbar-role">{{ auth.user?.role }}</span>
          </span>
          <button type="button" class="navbar-logout" @click="handleLogout">Sign out</button>
        </template>
        <router-link v-else to="/login" class="navbar-logout">Sign in</router-link>
      </div>
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
    height: 60px;
    padding: 0 1.5rem;
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
    height: 1.25rem;
    background: #3e4c7a;
    border-radius: 2px;
    flex-shrink: 0;
  }

  .navbar-title {
    font-size: 1.25rem;
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

  .navbar-right {
    display: flex;
    align-items: center;
    gap: 0.625rem;
    flex-shrink: 0;
  }

  .navbar-user {
    display: flex;
    align-items: center;
    gap: 0.375rem;
    font-family: Inter, sans-serif;
    font-size: 0.8125rem;
    font-weight: 500;
    color: #374151;
    white-space: nowrap;
  }

  .navbar-role {
    padding: 0.125rem 0.4375rem;
    font-size: 0.6875rem;
    font-weight: 600;
    color: #6b7280;
    background: #f3f4f6;
    border-radius: 9999px;
    text-transform: capitalize;
  }

  .navbar-logout {
    padding: 0.375rem 0.75rem;
    font-family: Inter, sans-serif;
    font-size: 0.8125rem;
    font-weight: 500;
    color: #6b7280;
    background: transparent;
    border: 1px solid #e5e7eb;
    border-radius: 8px;
    cursor: pointer;
    text-decoration: none;
    white-space: nowrap;
    transition: all 0.15s ease;
  }

  .navbar-logout:hover {
    color: #111827;
    background: #f9fafb;
  }

  @media (max-width: 767px) {
    .navbar-user {
      display: none;
    }
  }

  .navbar-cart {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 2.25rem;
    height: 2.25rem;
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
    width: 1.125rem;
    height: 1.125rem;
    stroke-width: 1.5;
  }

  .navbar-cart-badge {
    position: absolute;
    top: -3px;
    right: -3px;
    display: flex;
    align-items: center;
    justify-content: center;
    min-width: 1rem;
    height: 1rem;
    padding: 0 0.25rem;
    font-size: 0.5625rem;
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
      height: 60px;
      padding: 0 1.25rem;
    }

    .navbar-indicator {
      width: 3px;
      height: 1.125rem;
    }

    .navbar-title {
      font-size: 1.125rem;
    }

    .navbar-cart {
      width: 2.125rem;
      height: 2.125rem;
    }

    .navbar-cart-icon {
      width: 1.0625rem;
      height: 1.0625rem;
    }

    .navbar-cart-badge {
      top: -3px;
      right: -3px;
      min-width: 0.9375rem;
      height: 0.9375rem;
      font-size: 0.53125rem;
      padding: 0 0.21875rem;
    }
  }

  /* Mobile (< 768px) */
  @media (max-width: 767px) {
    .navbar-container {
      height: 60px;
      padding: 0 1rem;
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
      height: 60px;
      padding: 0 0.875rem;
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
