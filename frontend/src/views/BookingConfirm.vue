<script setup lang="ts">
  import { computed, onMounted } from 'vue'
  import { useRouter } from 'vue-router'
  import { useBookingCart } from '../composables/useBookingCart'
  import PageLayout from '../components/PageLayout.vue'

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

  const totalAmount = computed(() => cartItems.value.length * 50)

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

  const formatDate = (date: string | Date) => {
    return new Date(date).toLocaleDateString('en-US', {
      weekday: 'short',
      month: 'short',
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
    <div class="confirm-container">
      <!-- Technical Background -->
      <div class="technical-bg" aria-hidden="true">
        <div class="bg-grid"></div>
      </div>

      <div v-if="cartItems.length === 0 && !isLoading" class="empty-state">
        <div class="empty-icon">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="1.5"
              d="M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5m8.25 3v6.75m0 0l-3-3m3 3l3-3M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z"
            />
          </svg>
        </div>
        <h2 class="empty-title">Cart Empty</h2>
        <p class="empty-desc">Add sessions to begin booking</p>
        <button type="button" class="empty-action" @click="goBack">
          Browse Sessions
          <svg class="action-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M13.5 4.5L21 12m0 0l-7.5 7.5M21 12H3"
            />
          </svg>
        </button>
      </div>

      <div v-else class="cart-content">
        <!-- Invoice Header -->
        <div class="invoice-header">
          <div class="header-left">
            <div class="header-label">INVOICE SUMMARY</div>
            <div class="header-meta">
              <span class="meta-item">
                <span class="meta-label">DATE</span>
                <span class="meta-value">{{
                  new Date().toLocaleDateString('en-US', {
                    month: 'short',
                    day: 'numeric',
                    year: 'numeric',
                  })
                }}</span>
              </span>
              <span class="meta-item">
                <span class="meta-label">ITEMS</span>
                <span class="meta-value">{{ cartItems.length }}</span>
              </span>
            </div>
          </div>
          <button v-if="cartItems.length > 0" type="button" class="clear-btn" @click="clearCart">
            Clear All
          </button>
        </div>

        <!-- Line Items -->
        <div class="line-items">
          <div class="line-items-header">
            <span class="header-cell cell-desc">DESCRIPTION</span>
            <span class="header-cell cell-qty">QTY</span>
            <span class="header-cell cell-price">PRICE</span>
            <span class="header-cell cell-total">TOTAL</span>
            <span class="header-cell cell-action"></span>
          </div>

          <div class="line-items-list">
            <div
              v-for="(item, index) in cartItems"
              :key="item.id"
              class="line-item"
              :style="{ animationDelay: `${index * 50}ms` }"
            >
              <div class="item-desc">
                <span class="item-index">{{ String(index + 1).padStart(3, '0') }}</span>
                <div class="item-info">
                  <span class="item-teacher">{{ item.teacher_name }}</span>
                  <span class="item-time"
                    >{{ formatDate(item.start_time) }} · {{ formatTime(item.start_time) }}-{{
                      formatTime(item.end_time)
                    }}</span
                  >
                </div>
              </div>
              <span class="item-qty">1</span>
              <span class="item-price">$50.00</span>
              <span class="item-total">$50.00</span>
              <button type="button" class="item-remove" @click="handleRemove(item.id)">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 01-2.244 2.077H8.084a2.25 2.25 0 01-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 00-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 013.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 00-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 00-7.5 0"
                  />
                </svg>
              </button>
            </div>
          </div>
        </div>

        <!-- Invoice Summary -->
        <div class="invoice-summary">
          <div class="summary-row">
            <span class="summary-label">SUBTOTAL</span>
            <span class="summary-value">${{ (cartItems.length * 50).toFixed(2) }}</span>
          </div>
          <div class="summary-row">
            <span class="summary-label">SESSIONS</span>
            <span class="summary-value">{{ cartItems.length }} × $50.00</span>
          </div>
          <div class="summary-row summary-row--total">
            <span class="total-label">TOTAL</span>
            <span class="total-value">${{ totalAmount.toFixed(2) }}</span>
          </div>
        </div>

        <!-- Actions -->
        <div class="cart-actions">
          <button type="button" class="action-btn action-btn--secondary" @click="goBack">
            <svg class="btn-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
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
            :disabled="isLoading || cartItems.length === 0"
            class="action-btn action-btn--primary spring-press"
            @click="handleSubmit"
          >
            <span v-if="!isLoading">Confirm Booking</span>
            <span v-else>Processing...</span>
            <svg
              v-if="!isLoading"
              class="btn-icon"
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

      <!-- Toast Notifications -->
      <div v-if="successMessage" class="toast toast--success">
        <svg class="toast-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
          />
        </svg>
        {{ successMessage }}
      </div>

      <div v-if="errorMessage" class="toast toast--error">
        <svg class="toast-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z"
          />
        </svg>
        {{ errorMessage }}
        <button class="toast-close" @click="errorMessage = ''">×</button>
      </div>
    </div>
  </PageLayout>
</template>

<style scoped>
  .confirm-container {
    position: relative;
    max-width: 900px;
    margin: 0 auto;
  }

  /* Technical Background */
  .technical-bg {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    pointer-events: none;
    z-index: 0;
    overflow: hidden;
  }

  .bg-grid {
    position: absolute;
    inset: 0;
    background-image:
      linear-gradient(var(--border-subtle) 1px, transparent 1px),
      linear-gradient(90deg, var(--border-subtle) 1px, transparent 1px);
    background-size: 32px 32px;
    opacity: 0.2;
  }

  .bg-label {
    position: absolute;
    bottom: 10%;
    left: 50%;
    transform: translateX(-50%);
    font-family: 'JetBrains Mono', monospace;
    font-size: 3rem;
    font-weight: 600;
    color: var(--primary-indigo);
    opacity: 0.03;
    letter-spacing: 0.5em;
  }

  /* Empty State */
  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 4rem 2rem;
    text-align: center;
    animation: fade-in 0.4s ease-out;
    position: relative;
    z-index: 1;
  }

  .empty-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 4rem;
    height: 4rem;
    margin-bottom: 1.5rem;
    color: var(--text-muted);
    background: var(--bg-subtle);
    border-radius: 8px;
  }

  .empty-icon svg {
    width: 1.75rem;
    height: 1.75rem;
  }

  .empty-title {
    font-size: 1.25rem;
    font-weight: 600;
    color: var(--text-primary);
    font-family: 'Instrument Sans', sans-serif;
    margin-bottom: 0.375rem;
  }

  .empty-desc {
    font-size: 0.875rem;
    color: var(--text-secondary);
    margin-bottom: 1.5rem;
  }

  .empty-action {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.75rem 1.5rem;
    font-size: 0.875rem;
    font-weight: 500;
    color: white;
    background: var(--primary-indigo);
    border: none;
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
    box-shadow: 0 2px 8px rgba(62, 76, 122, 0.2);
  }

  .empty-action:hover {
    background: var(--primary-indigo-deep);
    transform: translateY(-2px);
  }

  .action-icon {
    width: 1rem;
    height: 1rem;
  }

  /* Cart Content */
  .cart-content {
    position: relative;
    z-index: 1;
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
    animation: fade-in-up 0.4s ease-out;
  }

  /* Invoice Header */
  .invoice-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    padding-bottom: 1rem;
    border-bottom: 2px solid var(--border-technical);
  }

  .header-left {
    display: flex;
    flex-direction: column;
    gap: 0.375rem;
  }

  .header-label {
    font-family: 'JetBrains Mono', monospace;
    font-size: 0.75rem;
    font-weight: 600;
    letter-spacing: 0.15em;
    color: var(--text-technical);
  }

  .header-meta {
    display: flex;
    gap: 1.5rem;
  }

  .meta-item {
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
  }

  .meta-label {
    font-size: 0.625rem;
    font-weight: 500;
    letter-spacing: 0.1em;
    color: var(--text-muted);
    text-transform: uppercase;
  }

  .meta-value {
    font-family: 'JetBrains Mono', monospace;
    font-size: 0.75rem;
    color: var(--text-primary);
  }

  .clear-btn {
    padding: 0.5rem 1rem;
    font-size: 0.75rem;
    font-weight: 500;
    color: var(--accent-coral);
    background: transparent;
    border: 1px solid var(--border-medium);
    border-radius: 4px;
    cursor: pointer;
    transition: all 0.2s ease;
    font-family: 'IBM Plex Sans', sans-serif;
  }

  .clear-btn:hover {
    background: rgba(201, 109, 93, 0.08);
    border-color: var(--accent-coral);
  }

  /* Line Items */
  .line-items {
    background: var(--bg-card);
    border: 1px solid var(--border-medium);
    border-radius: 8px;
    overflow: hidden;
  }

  .line-items-header {
    display: grid;
    grid-template-columns: 1fr auto auto auto auto;
    gap: 1rem;
    padding: 0.75rem 1rem;
    background: var(--bg-subtle);
    border-bottom: 1px solid var(--border-medium);
  }

  .header-cell {
    font-family: 'JetBrains Mono', monospace;
    font-size: 0.625rem;
    font-weight: 600;
    letter-spacing: 0.1em;
    color: var(--text-secondary);
    text-transform: uppercase;
  }

  .cell-desc {
    grid-column: 1 / 2;
  }

  .cell-qty {
    justify-self: center;
    width: 2rem;
    text-align: center;
  }

  .cell-price {
    justify-self: end;
    width: 3.5rem;
    text-align: right;
  }

  .cell-total {
    justify-self: end;
    width: 3.5rem;
    text-align: right;
  }

  .cell-action {
    width: 2.5rem;
  }

  .line-items-list {
    max-height: 400px;
    overflow-y: auto;
  }

  .line-item {
    display: grid;
    grid-template-columns: 1fr auto auto auto auto;
    gap: 1rem;
    padding: 0.875rem 1rem;
    border-bottom: 1px solid var(--border-subtle);
    align-items: center;
    animation: line-in 0.3s cubic-bezier(0.34, 1.56, 0.64, 1) backwards;
  }

  .line-item:last-child {
    border-bottom: none;
  }

  .line-item:hover {
    background: var(--bg-subtle);
  }

  @keyframes line-in {
    from {
      opacity: 0;
      transform: translateX(-8px);
    }
    to {
      opacity: 1;
      transform: translateX(0);
    }
  }

  .item-desc {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .item-index {
    font-family: 'JetBrains Mono', monospace;
    font-size: 0.625rem;
    font-weight: 600;
    color: var(--text-technical);
  }

  .item-info {
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
  }

  .item-teacher {
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text-primary);
  }

  .item-time {
    font-size: 0.75rem;
    color: var(--text-secondary);
    font-family: 'JetBrains Mono', monospace;
  }

  .item-qty {
    font-family: 'JetBrains Mono', monospace;
    font-size: 0.875rem;
    color: var(--text-primary);
    justify-self: center;
  }

  .item-price {
    font-family: 'JetBrains Mono', monospace;
    font-size: 0.875rem;
    color: var(--text-secondary);
    justify-self: end;
  }

  .item-total {
    font-family: 'JetBrains Mono', monospace;
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text-primary);
    justify-self: end;
  }

  .item-remove {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 1.75rem;
    height: 1.75rem;
    color: var(--text-muted);
    background: transparent;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .item-remove:hover {
    color: var(--accent-coral);
    background: rgba(201, 109, 93, 0.08);
  }

  .item-remove svg {
    width: 0.875rem;
    height: 0.875rem;
  }

  /* Summary */
  .invoice-summary {
    background: var(--bg-card);
    border: 1px solid var(--border-medium);
    border-radius: 8px;
    padding: 1rem 1.25rem;
  }

  .summary-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.5rem 0;
  }

  .summary-label {
    font-size: 0.8125rem;
    color: var(--text-secondary);
  }

  .summary-value {
    font-family: 'JetBrains Mono', monospace;
    font-size: 0.8125rem;
    color: var(--text-primary);
  }

  .summary-row--total {
    margin-top: 0.5rem;
    padding-top: 0.75rem;
    border-top: 2px solid var(--border-technical);
  }

  .total-label {
    font-family: 'JetBrains Mono', monospace;
    font-size: 0.75rem;
    font-weight: 600;
    letter-spacing: 0.1em;
    color: var(--text-primary);
  }

  .total-value {
    font-family: 'JetBrains Mono', monospace;
    font-size: 1.25rem;
    font-weight: 600;
    color: var(--primary-indigo);
  }

  /* Actions */
  .cart-actions {
    display: flex;
    gap: 0.75rem;
  }

  .action-btn {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    padding: 0.875rem 1.25rem;
    font-size: 0.875rem;
    font-weight: 500;
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
    font-family: 'IBM Plex Sans', sans-serif;
  }

  .action-btn--secondary {
    color: var(--text-primary);
    background: var(--bg-card);
    border: 1px solid var(--border-medium);
  }

  .action-btn--secondary:hover {
    background: var(--bg-subtle);
  }

  .action-btn--primary {
    color: white;
    background: var(--accent-sage);
    border: none;
    box-shadow: 0 2px 8px rgba(122, 139, 109, 0.2);
  }

  .action-btn--primary:hover:not(:disabled) {
    filter: brightness(0.9);
    transform: translateY(-2px);
    box-shadow: 0 4px 16px rgba(122, 139, 109, 0.3);
  }

  .action-btn--primary:disabled {
    opacity: 0.4;
    cursor: not-allowed;
    transform: none;
    box-shadow: none;
  }

  .btn-icon {
    width: 1rem;
    height: 1rem;
  }

  /* Toast */
  .toast {
    position: fixed;
    bottom: 2rem;
    right: 2rem;
    display: flex;
    align-items: center;
    gap: 0.625rem;
    padding: 0.875rem 1.25rem;
    border-radius: 6px;
    box-shadow: var(--shadow-elevated);
    animation: toast-in 0.4s cubic-bezier(0.34, 1.56, 0.64, 1);
    z-index: 100;
    font-size: 0.875rem;
    font-family: 'IBM Plex Sans', sans-serif;
  }

  .toast--success {
    background: var(--accent-sage);
    color: white;
  }

  .toast--error {
    background: var(--accent-coral);
    color: white;
  }

  .toast-icon {
    width: 1.125rem;
    height: 1.125rem;
    flex-shrink: 0;
  }

  .toast-close {
    margin-left: 0.625rem;
    color: rgba(255, 255, 255, 0.8);
    background: transparent;
    border: none;
    font-size: 1.25rem;
    cursor: pointer;
    width: 1.25rem;
    height: 1.25rem;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  @keyframes fade-in {
    from {
      opacity: 0;
    }
    to {
      opacity: 1;
    }
  }

  @keyframes fade-in-up {
    from {
      opacity: 0;
      transform: translateY(12px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  @keyframes toast-in {
    from {
      opacity: 0;
      transform: translateY(16px) scale(0.95);
    }
    to {
      opacity: 1;
      transform: translateY(0) scale(1);
    }
  }

  @media (max-width: 640px) {
    .line-items-header,
    .line-item {
      grid-template-columns: 1fr auto;
      gap: 0.5rem;
    }

    .cell-price,
    .cell-total,
    .item-price,
    .item-total {
      display: none;
    }

    .invoice-header {
      flex-direction: column;
      gap: 0.75rem;
      align-items: stretch;
    }

    .clear-btn {
      align-self: flex-start;
    }

    .toast {
      right: 1rem;
      left: 1rem;
    }

    .cart-actions {
      flex-direction: column;
    }

    .action-btn {
      width: 100%;
    }
  }
</style>
