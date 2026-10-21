<template>
  <BaseModal
    :model-value="show"
    :closable="!busy"
    title="确认操作"
    icon="fas fa-exclamation-triangle"
    tone="warning"
    width="450px"
    aria-label="确认操作"
    @update:model-value="onUpdate"
  >
    <p class="confirm-message">{{ message }}</p>

    <template #footer>
      <button type="button" class="btn btn-secondary" :disabled="busy" @click="$emit('cancel')">取消</button>
      <button type="button" class="btn btn-danger" :disabled="busy" @click="$emit('confirm')" data-autofocus>{{ busy ? '处理中…' : '确定' }}</button>
    </template>
  </BaseModal>
</template>

<script lang="ts">
import { defineComponent } from 'vue'
import BaseModal from '@/components/BaseModal.vue'

export default defineComponent({
  name: 'ConfirmModal',
  components: { BaseModal },
  props: {
    show: { type: Boolean, default: false },
    busy: { type: Boolean, default: false },
    message: { type: String, default: '' }
  },
  emits: ['confirm', 'cancel'],
  setup(_props, { emit }) {
    const onUpdate = (value: boolean) => {
      if (!value) {
        emit('cancel')
      }
    }
    return { onUpdate }
  }
})
</script>

<style scoped>
.confirm-message {
  margin: 0;
  font-size: var(--font-base);
  line-height: 1.6;
  color: var(--text-secondary);
  text-align: center;
  white-space: pre-line;
}

.btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-width: 80px;
  padding: 10px 20px;
  border: none;
  border-radius: var(--radius-md);
  font-size: var(--font-sm);
  font-weight: var(--font-medium);
  cursor: pointer;
  transition: all var(--transition);
}

.btn-secondary {
  background: var(--gray-100);
  color: var(--text-secondary);
  border: 1px solid var(--border);
}

.btn-secondary:hover {
  background: var(--gray-200);
}

.btn-danger {
  background: var(--red);
  color: white;
}

.btn-danger:hover {
  background: #dc2626;
  box-shadow: 0 4px 12px rgba(239, 68, 68, 0.3);
}
</style>
