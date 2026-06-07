<script setup lang="ts">
  interface Toast {
    message: string
    type?: 'success' | 'error'
    duration?: number
  }

  const props = withDefaults(defineProps<Toast>(), {
    type: 'success',
    duration: 3000,
  })

  const emit = defineEmits<{
    close: []
  }>()

  setTimeout(() => emit('close'), props.duration)
</script>

<template>
  <div :class="['toast', `toast--${type}`]">
    <svg
      v-if="type === 'success'"
      class="toast-icon"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
    >
      <path
        stroke-linecap="round"
        stroke-linejoin="round"
        stroke-width="2"
        d="M9 12.75L11.25 15 15 9.75M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
      />
    </svg>
    <svg v-else class="toast-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor">
      <path
        stroke-linecap="round"
        stroke-linejoin="round"
        stroke-width="2"
        d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z"
      />
    </svg>
    {{ message }}
    <button class="toast-close" @click="emit('close')">×</button>
  </div>
</template>

<style scoped>
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
    .toast {
      right: 1rem;
      left: 1rem;
    }
  }
</style>
