<script setup lang="ts">
  import { CheckCircle2, X, AlertCircle } from '@lucide/vue'
  import { useNotification } from '../composables/useNotification'

  const { successMessage, errorMessage } = useNotification()
</script>

<template>
  <div class="toast-region" aria-live="polite">
    <div v-if="successMessage" class="toast toast--success" role="status">
      <CheckCircle2 class="toast-icon" aria-hidden="true" />
      <span class="toast-text">{{ successMessage }}</span>
    </div>
    <div v-if="errorMessage" class="toast toast--error" role="alert">
      <AlertCircle class="toast-icon" aria-hidden="true" />
      <span class="toast-text">{{ errorMessage }}</span>
      <button
        type="button"
        class="toast-dismiss"
        aria-label="Dismiss message"
        @click="errorMessage = ''"
      >
        <X class="toast-icon" aria-hidden="true" />
      </button>
    </div>
  </div>
</template>

<style scoped>
  @keyframes toast-in {
    from {
      opacity: 0;
      transform: translateY(12px) scale(0.96);
    }
    to {
      opacity: 1;
      transform: translateY(0) scale(1);
    }
  }

  .toast-region {
    position: fixed;
    right: 2rem;
    bottom: 2rem;
    z-index: 100;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    pointer-events: none;
  }

  .toast {
    display: flex;
    align-items: center;
    gap: 0.625rem;
    padding: 0.875rem 1.25rem;
    border-radius: 0.5rem;
    font-family: Inter, sans-serif;
    font-size: 0.875rem;
    color: #fff;
    box-shadow: var(--shadow-elevated);
    pointer-events: auto;
    animation: toast-in 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
  }

  .toast--success {
    background: var(--accent-sage);
  }

  .toast--error {
    background: var(--accent-coral);
  }

  .toast-icon {
    width: 1.125rem;
    height: 1.125rem;
    flex-shrink: 0;
  }

  .toast-text {
    min-width: 0;
  }

  .toast-dismiss {
    display: flex;
    align-items: center;
    justify-content: center;
    margin-left: 0.5rem;
    padding: 0;
    border: none;
    background: transparent;
    color: rgba(255, 255, 255, 0.8);
    cursor: pointer;
  }

  .toast-dismiss:hover {
    color: #fff;
  }

  @media (max-width: 640px) {
    .toast-region {
      right: 1rem;
      bottom: 1rem;
      left: 1rem;
    }
  }
</style>
