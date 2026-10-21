<template>
  <BaseModal
    :model-value="show"
    :title="getStatTitle(type)"
    :icon="getStatIcon(type)"
    width="1200px"
    panel-class="statistics-modal"
    @update:model-value="onUpdate"
  >
    <div v-if="items.length === 0" class="empty-stat">
      <i class="fas fa-inbox" aria-hidden="true"></i>
      <p>暂无{{ getStatTitle(type) }}</p>
    </div>
    <div v-else class="stat-items-grid">
      <AssignmentCard
        v-for="item in items" 
        :key="item.id"
        :assignment="item"
        @open-url="$emit('open-url', $event)"
        @show-preview="$emit('show-preview', $event)"
        @set-reminder="$emit('set-reminder', $event)"
        @delete-assignment="$emit('delete-assignment', $event)"
        @mark-completed="$emit('mark-completed', $event)"
        @undo-completed="$emit('undo-completed', $event)"
        @blacklist-subject="$emit('blacklist-subject', $event)"
      />
    </div>

    <template #footer>
      <button type="button" class="btn btn-secondary" @click="$emit('close')">关闭</button>
    </template>
  </BaseModal>
</template>

<script lang="ts">
import { defineComponent } from 'vue'
import type { PropType } from 'vue'
import BaseModal from '@/components/BaseModal.vue'
import AssignmentCard from '../AssignmentCard.vue'
import type { Assignment } from '../../home.data'

const STAT_TITLES: Record<string, string> = {
  urgent: '紧急作业（1天内，含逾期）',
  warning: '即将到期（1～3天内）',
  completed: '已完成',
  todos: '待办事项',
  all: '全部作业'
}

const STAT_ICONS: Record<string, string> = {
  urgent: 'fas fa-exclamation-triangle',
  warning: 'fas fa-clock',
  completed: 'fas fa-check-circle',
  todos: 'fas fa-list-check',
  all: 'fas fa-tasks'
}

export default defineComponent({
  name: 'StatisticsModal',
  components: {
    BaseModal,
    AssignmentCard
  },
  props: {
    show: {
      type: Boolean,
      default: false
    },
    type: {
      type: String,
      default: ''
    },
    items: {
      type: Array as PropType<Assignment[]>,
      default: () => []
    }
  },
  emits: ['close', 'open-url', 'show-preview', 'set-reminder', 'delete-assignment', 'mark-completed', 'undo-completed', 'blacklist-subject'],
  setup(_props, { emit }) {
    const onUpdate = (value: boolean) => {
      if (!value) {
        emit('close')
      }
    }

    const getStatTitle = (type: string): string => STAT_TITLES[type] ?? '作业列表'
    const getStatIcon = (type: string): string => STAT_ICONS[type] ?? 'fas fa-list'

    return { onUpdate, getStatTitle, getStatIcon }
  }
})
</script>

<style scoped>
.empty-stat {
  text-align: center;
  padding: 40px 20px;
  color: var(--text-tertiary);
}

.empty-stat i {
  font-size: 3em;
  margin-bottom: 15px;
  opacity: 0.5;
}

.stat-items-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(450px, 100%), 1fr));
  gap: 20px;
  padding: 10px 0;
}

.btn {
  padding: 10px 20px;
  border: none;
  border-radius: var(--radius-md);
  cursor: pointer;
  font-size: var(--font-sm);
  transition: all var(--transition);
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  font-weight: var(--font-medium);
  min-width: 80px;
}

.btn-secondary {
  background: var(--gray-100);
  color: var(--text-secondary);
  border: 1px solid var(--border);
}

.btn-secondary:hover {
  background: var(--gray-200);
}
</style>
