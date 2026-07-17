<script setup lang="ts">
  import { onMounted, onUnmounted, ref } from 'vue'
  import { useRouter } from 'vue-router'
  import { useBookingCart } from '../composables/useBookingCart.ts'
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

  const showClearConfirm = ref(false)

  onMounted(() => {
    fetchCartItems()
  })

  const handleRemove = (id: number) => {
    removeItem(id)
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
    await submitBookings()
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

  const handleKeydown = (e: KeyboardEvent) => {
    if (e.key === 'Escape') {
      showClearConfirm.value = false
    }
  }

  onMounted(() => {
    window.addEventListener('keydown', handleKeydown)
  })

  onUnmounted(() => {
    window.removeEventListener('keydown', handleKeydown)
  })
</script>

<template>
  <PageLayout title="Confirm Bookings">
    <div class="confirm-container">
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
        <h2 class="empty-title">Your cart is empty</h2>
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
        <!-- Header with Clear All -->
        <div class="content-header">
          <div class="header-info">
            <span class="session-count"
              >{{ cartItems.length }} session{{ cartItems.length !== 1 ? 's' : '' }}</span
            >
          </div>
          <button
            v-if="cartItems.length > 0"
            type="button"
            class="clear-btn"
            @click="handleClearAll"
          >
            Clear All
          </button>
        </div>

        <!-- Booking Cards -->
        <div class="booking-cards">
          <div
            v-for="(item, index) in cartItems"
            :key="item.id"
            class="booking-card"
            :style="{ animationDelay: `${index * 50}ms` }"
          >
            <button type="button" class="card-remove" @click="handleRemove(item.id)">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor">
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="2"
                  d="M6 18L18 6M6 6l12 12"
                />
              </svg>
            </button>

            <div class="card-content">
              <span class="card-subject">{{ item.subject_name }}</span>
              <span class="card-time"
                >Date: {{ formatDate(item.start_time) }} · {{ formatTime(item.start_time) }}-{{
                  formatTime(item.end_time)
                }}</span
              >
              <span class="card-teacher">Teacher: {{ item.teacher_name }}</span>
              <span class="card-branch">Branch: {{ item.branch_name }}</span>
            </div>
          </div>
        </div>

        <!-- Action Buttons -->
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
            class="action-btn action-btn--primary"
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

      <!-- Clear All Confirmation Dialog -->
      <div
        v-if="showClearConfirm"
        class="confirm-dialog-overlay"
        @click.self="showClearConfirm = false"
      >
        <div
          class="confirm-dialog"
          role="dialog"
          aria-modal="true"
          aria-labelledby="clear-confirm-title"
        >
          <h3 id="clear-confirm-title" class="confirm-title">Clear all sessions?</h3>
          <p class="confirm-desc">
            This will remove all sessions from your cart. This action cannot be undone.
          </p>
          <div class="confirm-actions">
            <button
              type="button"
              class="confirm-btn confirm-btn--cancel"
              @click="showClearConfirm = false"
            >
              Cancel
            </button>
            <button type="button" class="confirm-btn confirm-btn--confirm" @click="confirmClearAll">
              Clear All
            </button>
          </div>
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
    max-width: 700px;
    margin: 0 auto;
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
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
    animation: fade-in-up 0.4s ease-out;
  }

  /* Content Header */
  .content-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding-bottom: 1rem;
    border-bottom: 1px solid var(--border-medium);
  }

  .header-info {
    display: flex;
    flex-direction: column;
  }

  .session-count {
    font-size: 1rem;
    font-weight: 500;
    color: var(--text-primary);
  }

  .clear-btn {
    padding: 0.5rem 1rem;
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--accent-coral);
    background: transparent;
    border: 1px solid var(--border-medium);
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.2s ease;
    font-family: 'Inter', sans-serif;
  }

  .clear-btn:hover {
    background: rgba(201, 109, 93, 0.08);
    border-color: var(--accent-coral);
  }

  /* Booking Cards */
  .booking-cards {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .booking-card {
    position: relative;
    background: var(--bg-card);
    border: 1px solid var(--border-medium);
    border-radius: 8px;
    padding: 1rem;
    animation: card-in 0.3s cubic-bezier(0.34, 1.56, 0.64, 1) backwards;
    transition: background 0.2s ease;
  }

  .booking-card:hover {
    background: var(--bg-subtle);
  }

  @keyframes card-in {
    from {
      opacity: 0;
      transform: translateY(8px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  .card-remove {
    position: absolute;
    top: 1rem;
    right: 1rem;
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

  .card-remove:hover {
    color: var(--accent-coral);
    background: rgba(201, 109, 93, 0.08);
  }

  .card-remove svg {
    width: 0.875rem;
    height: 0.875rem;
  }

  .card-content {
    display: flex;
    flex-direction: column;
    gap: 0.375rem;
    padding-right: 2rem;
  }

  .card-teacher {
    font-size: 0.875rem;
    color: var(--text-secondary);
  }

  .card-subject {
    font-size: 1rem;
    font-weight: 600;
    color: var(--text-primary);
    font-family: 'Instrument Sans', sans-serif;
  }

  .card-time,
  .card-branch {
    font-size: 0.875rem;
    color: var(--text-secondary);
  }

  .card-time {
    font-family: 'JetBrains Mono', monospace;
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
    font-family: 'Inter', sans-serif;
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
    font-family: 'Inter', sans-serif;
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

  /* Confirmation Dialog */
  .confirm-dialog-overlay {
    position: fixed;
    inset: 0;
    background: rgba(26, 28, 35, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 200;
    animation: fade-in 0.2s ease-out;
  }

  .confirm-dialog {
    background: var(--bg-card);
    border: 1px solid var(--border-medium);
    border-top: 2px solid var(--accent-coral);
    border-radius: 8px;
    padding: 1.5rem;
    max-width: 400px;
    margin: 0 1rem;
    animation: dialog-in 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
  }

  .confirm-title {
    font-size: 1rem;
    font-weight: 600;
    font-family: 'Instrument Sans', sans-serif;
    color: var(--text-primary);
    margin: 0 0 0.5rem 0;
  }

  .confirm-desc {
    font-size: 0.875rem;
    color: var(--text-secondary);
    margin: 0 0 1.25rem 0;
    line-height: 1.5;
  }

  .confirm-actions {
    display: flex;
    gap: 0.75rem;
    justify-content: flex-end;
  }

  .confirm-btn {
    padding: 0.625rem 1rem;
    font-size: 0.875rem;
    font-weight: 500;
    font-family: 'Inter', sans-serif;
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .confirm-btn--cancel {
    color: var(--text-primary);
    background: var(--bg-subtle);
    border: 1px solid var(--border-medium);
  }

  .confirm-btn--cancel:hover {
    background: var(--border-subtle);
  }

  .confirm-btn--confirm {
    color: white;
    background: var(--accent-coral);
    border: none;
  }

  .confirm-btn--confirm:hover {
    background: #b85c4e;
  }

  /* Animations */
  @keyframes dialog-in {
    from {
      opacity: 0;
      transform: scale(0.95);
    }
    to {
      opacity: 1;
      transform: scale(1);
    }
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

  /* Mobile Responsive */
  @media (max-width: 640px) {
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
