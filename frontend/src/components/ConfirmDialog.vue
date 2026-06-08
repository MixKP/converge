<script setup lang="ts">
  interface Props {
    show?: boolean
    title?: string
    items?: Array<{ label: string; value: string }>
    confirmText?: string
    cancelText?: string
  }

  withDefaults(defineProps<Props>(), {
    show: false,
    title: 'Confirm',
    items: () => [],
    confirmText: 'Confirm',
    cancelText: 'Cancel',
  })

  const emit = defineEmits<{
    confirm: []
    cancel: []
  }>()
</script>

<template>
  <div v-if="show" class="dialog-overlay" @click.self="emit('cancel')">
    <div class="dialog">
      <h3 class="dialog-title">{{ title }}</h3>
      <div class="dialog-content">
        <div v-for="item in items" :key="item.label" class="dialog-row">
          <span class="dialog-label">{{ item.label }}</span>
          <span class="dialog-value">{{ item.value }}</span>
        </div>
        <slot />
      </div>
      <div class="dialog-actions">
        <button type="button" class="dialog-btn dialog-btn--cancel" @click="emit('cancel')">
          {{ cancelText }}
        </button>
        <button type="button" class="dialog-btn dialog-btn--confirm" @click="emit('confirm')">
          {{ confirmText }}
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
  .dialog-overlay {
    position: fixed;
    inset: 0;
    background: rgba(26, 28, 35, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 200;
    animation: fade-in 0.2s ease-out;
  }

  .dialog {
    background: var(--bg-card);
    border: 1px solid var(--border-medium);
    border-top: 2px solid var(--accent-sage);
    border-radius: 8px;
    padding: 1.5rem;
    max-width: 400px;
    margin: 0 1rem;
    animation: dialog-in 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
  }

  .dialog-title {
    font-size: 1rem;
    font-weight: 600;
    font-family: 'Instrument Sans', sans-serif;
    color: var(--text-primary);
    margin: 0 0 1rem 0;
  }

  .dialog-content {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    margin-bottom: 1.25rem;
  }

  .dialog-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .dialog-label {
    font-family: 'JetBrains Mono', monospace;
    font-size: 0.6875rem;
    font-weight: 600;
    letter-spacing: 0.1em;
    color: var(--text-secondary);
    text-transform: uppercase;
  }

  .dialog-value {
    font-family: 'Inter', sans-serif;
    font-size: 0.875rem;
    color: var(--text-primary);
  }

  .dialog-content :deep(.confirm-slots) {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    padding: 0.75rem;
    background: var(--bg-subtle);
    border-radius: 4px;
  }

  .dialog-content :deep(.confirm-slot) {
    font-family: 'JetBrains Mono', monospace;
    font-size: 0.8125rem;
    color: var(--text-primary);
  }

  .dialog-actions {
    display: flex;
    gap: 0.75rem;
    justify-content: flex-end;
  }

  .dialog-btn {
    padding: 0.625rem 1rem;
    font-size: 0.875rem;
    font-weight: 500;
    font-family: 'Inter', sans-serif;
    border-radius: 6px;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .dialog-btn--cancel {
    color: var(--text-primary);
    background: var(--bg-subtle);
    border: 1px solid var(--border-medium);
  }

  .dialog-btn--cancel:hover {
    background: var(--border-subtle);
  }

  .dialog-btn--confirm {
    color: white;
    background: var(--accent-sage);
    border: none;
  }

  .dialog-btn--confirm:hover {
    filter: brightness(0.9);
  }

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
</style>
