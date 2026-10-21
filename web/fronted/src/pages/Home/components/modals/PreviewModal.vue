<template>
  <BaseModal
    :model-value="show"
    title="详情预览"
    icon="fas fa-eye"
    width="600px"
    panel-class="preview-modal"
    @update:model-value="onUpdate"
  >
    <div v-if="item">
      <div class="preview-field">
        <label>类型</label>
        <div class="preview-value">
          <span :class="['type-badge', item.type]">{{ item.type }}</span>
        </div>
      </div>
      <div class="preview-field">
        <label>科目</label>
        <div class="preview-value">{{ item.subject }}</div>
      </div>
      <div class="preview-field">
        <label>标题</label>
        <div class="preview-value preview-title">{{ item.title }}</div>
      </div>
      <div class="preview-field">
        <label>内容</label>
        <div class="preview-value preview-content">
          <div v-if="item.description && item.description.trim()">
            {{ item.description }}
          </div>
          <div v-else>
            暂无内容
          </div>
        </div>
      </div>
      <div v-if="item.dueDate" class="preview-field">
        <label>截止时间</label>
        <div class="preview-value preview-date">
          <i :class="statusIcon" aria-hidden="true"></i>
          <span>
            <template v-for="(line, index) in formattedDateLines" :key="index">
              <br v-if="index > 0" />
              {{ line }}
            </template>
          </span>
        </div>
      </div>
      <div v-if="item.priority && item.type === '待办'" class="preview-field">
        <label>优先级</label>
        <div class="preview-value">
          <span :class="['priority-badge', item.priority]">
            <i class="fas fa-flag" aria-hidden="true"></i>
            {{ getPriorityText(item.priority) }}
          </span>
        </div>
      </div>
      <div v-if="item.url" class="preview-field">
        <label>链接</label>
        <div class="preview-value">
          <a :href="item.url" target="_blank" rel="noopener" class="preview-link">
            <i class="fas fa-external-link-alt" aria-hidden="true"></i>
            打开链接
          </a>
        </div>
      </div>
      <div class="preview-field">
        <label>状态</label>
        <div class="preview-value">
          <span :class="['status-badge', assignmentStatus]">
            <i :class="statusIcon" aria-hidden="true"></i>
            {{ getStatusText(assignmentStatus) }}
          </span>
        </div>
      </div>
    </div>

    <template #footer>
      <button type="button" class="btn btn-secondary" @click="$emit('close')">关闭</button>
      <button
        v-if="item && item.url"
        type="button"
        class="btn btn-primary"
        @click="$emit('open-url', item)"
      >
        <i class="fas fa-external-link-alt" aria-hidden="true"></i> 打开链接
      </button>
    </template>
  </BaseModal>
</template>

<script lang="ts">
import { getAssignmentStatus } from '@/utils/assignment-status'
import { useTaskClock } from '@/composables/useTaskClock'
import { computed, defineComponent } from 'vue'
import type { PropType } from 'vue'
import BaseModal from '@/components/BaseModal.vue'
import type { Assignment } from '../../home.data'
import { daysUntil, parseDueDate } from '@/utils/datetime'

type AssignmentStatus = 'urgent' | 'warning' | 'normal' | 'completed'

const STATUS_ICONS: Record<AssignmentStatus, string> = {
  urgent: 'fas fa-exclamation-triangle',
  warning: 'fas fa-clock',
  normal: 'fas fa-calendar-alt',
  completed: 'fas fa-check-circle'
}

const PRIORITY_TEXT: Record<string, string> = {
  high: '高',
  medium: '中',
  low: '低'
}

const STATUS_TEXT: Record<string, string> = {
  urgent: '紧急',
  warning: '即将到期',
  normal: '正常',
  completed: '已完成'
}

/** 无时区信息的 ISO 字符串按北京时间解析。 */
function getDaysUntilDue(dueDate?: string): number {
  const days = daysUntil(dueDate)
  return days === null ? (dueDate ? 7 : Infinity) : days
}

export default defineComponent({
  name: 'PreviewModal',
  components: { BaseModal },
  props: {
    show: {
      type: Boolean,
      default: false
    },
    item: {
      type: Object as PropType<Assignment | null>,
      default: null
    }
  },
  emits: ['close', 'open-url'],
  setup(props, { emit }) {
    const now = useTaskClock()
    const onUpdate = (value: boolean) => {
      if (!value) {
        emit('close')
      }
    }

    const assignmentStatus = computed<AssignmentStatus>(() => {
      const item = props.item
      if (!item) return 'normal'
      return getAssignmentStatus(item, now.value)
    })

    const statusIcon = computed(() => STATUS_ICONS[assignmentStatus.value])

    /** 返回多行文本，由模板渲染 <br>，不再使用 v-html。 */
    const formattedDateLines = computed<string[]>(() => {
      const item = props.item
      if (!item?.dueDate) return []

      const date = parseDueDate(item.dueDate)
      if (!date) return []

      const diffMs = date.getTime() - Date.now()

      if (item.type === '待办') {
        const lines: string[] = []
        const estimatedHours = item.estimatedHours

        if (estimatedHours) {
          if (estimatedHours >= 24) {
            const days = Math.floor(estimatedHours / 24)
            const hours = estimatedHours % 24
            lines.push(`预计 ${days}天${hours > 0 ? hours + '小时' : ''}`)
          } else {
            lines.push(`预计 ${estimatedHours}小时`)
          }
        }

        if (diffMs < 0) {
          const overdueDays = Math.floor(Math.abs(diffMs) / (1000 * 60 * 60 * 24))
          const overdueHours = Math.floor(Math.abs(diffMs) / (1000 * 60 * 60)) % 24
          lines.push(
            overdueDays > 0
              ? `已超时 ${overdueDays}天${overdueHours > 0 ? overdueHours + '小时' : ''}`
              : `已超时 ${overdueHours}小时`
          )
        } else {
          const remainingDays = Math.floor(diffMs / (1000 * 60 * 60 * 24))
          const remainingHours = Math.floor(diffMs / (1000 * 60 * 60)) % 24
          const remainingMinutes = Math.floor(diffMs / (1000 * 60)) % 60

          if (remainingDays > 0) {
            lines.push(`剩余 ${remainingDays}天${remainingHours > 0 ? remainingHours + '小时' : ''}`)
          } else if (remainingHours > 0) {
            lines.push(`剩余 ${remainingHours}小时${remainingMinutes > 0 ? remainingMinutes + '分钟' : ''}`)
          } else {
            lines.push(`剩余 ${remainingMinutes}分钟`)
          }
        }

        return lines
      }

      const days = getDaysUntilDue(item.dueDate)
      let formatted = date.toLocaleDateString('zh-CN', {
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
      })

      if (days < 0) {
        formatted += ` (已过期${Math.abs(days)}天)`
      } else if (days === 0) {
        formatted += ' (今天截止)'
      } else if (days === 1) {
        formatted += ' (明天截止)'
      } else if (days <= 7) {
        formatted += ` (还有${days}天)`
      }

      return [formatted]
    })

    const getPriorityText = (priority: string): string => PRIORITY_TEXT[priority] ?? priority
    const getStatusText = (status: string): string => STATUS_TEXT[status] ?? status

    return {
      onUpdate,
      assignmentStatus,
      statusIcon,
      formattedDateLines,
      getPriorityText,
      getStatusText
    }
  }
})
</script>

<style scoped>
.preview-field {
  margin-bottom: 20px;
}

.preview-field label {
  display: block;
  font-weight: var(--font-semibold);
  color: var(--text-secondary);
  margin-bottom: 6px;
  font-size: var(--font-xs);
}

.preview-value {
  background: var(--gray-50);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  padding: 12px 16px;
  font-size: var(--font-sm);
  line-height: 1.6;
  color: var(--text-primary);
  min-height: 20px;
}

.preview-title {
  font-weight: 600;
  font-size: 16px;
  color: #111827;
}

.preview-content {
  white-space: pre-wrap;
  word-break: break-word;
}

.preview-date {
  display: flex;
  align-items: center;
  gap: 8px;
}

.preview-link {
  color: var(--primary);
  text-decoration: none;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-weight: 500;
  transition: all 0.3s ease;
}

.preview-link:hover {
  color: #5a6fd8;
  text-decoration: underline;
}

/* 类型标签 */
.type-badge {
  display: inline-flex;
  align-items: center;
  padding: 4px 12px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.type-badge.作业 {
  background: var(--primary);
  color: white;
}

.type-badge.测试 {
  background: var(--orange);
  color: white;
}

.type-badge.待办 {
  background: var(--purple);
  color: white;
}

/* 优先级标签 */
.priority-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 20px;
  font-size: 13px;
  font-weight: 600;
}

.priority-badge.high {
  background: rgba(239, 68, 68, 0.1);
  color: #dc2626;
  border: 1px solid rgba(239, 68, 68, 0.2);
}

.priority-badge.medium {
  background: rgba(245, 158, 11, 0.1);
  color: #d97706;
  border: 1px solid rgba(245, 158, 11, 0.2);
}

.priority-badge.low {
  background: rgba(34, 197, 94, 0.1);
  color: #16a34a;
  border: 1px solid rgba(34, 197, 94, 0.2);
}

/* 状态标签 */
.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 20px;
  font-size: 13px;
  font-weight: 600;
}

.status-badge.urgent {
  background: rgba(239, 68, 68, 0.1);
  color: #dc2626;
  border: 1px solid rgba(239, 68, 68, 0.2);
}

.status-badge.warning {
  background: rgba(245, 158, 11, 0.1);
  color: #d97706;
  border: 1px solid rgba(245, 158, 11, 0.2);
}

.status-badge.normal {
  background: rgba(34, 197, 94, 0.1);
  color: #16a34a;
  border: 1px solid rgba(34, 197, 94, 0.2);
}

.status-badge.completed {
  background: rgba(34, 197, 94, 0.1);
  color: #16a34a;
  border: 1px solid rgba(34, 197, 94, 0.2);
}

.btn {
  padding: 12px 24px;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
  transition: all 0.3s ease;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  text-align: center;
  font-weight: 500;
  min-width: 100px;
  -webkit-tap-highlight-color: transparent;
}

.btn-secondary {
  padding: 10px 18px;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--gray-100);
  color: var(--text-secondary);
  font-weight: var(--font-medium);
  font-size: var(--font-sm);
  cursor: pointer;
  transition: all var(--transition);
}

.btn-secondary:hover {
  background: var(--gray-200);
}

.btn-primary {
  padding: 10px 18px;
  border: none;
  border-radius: var(--radius-md);
  background: var(--primary);
  color: white;
  font-weight: var(--font-medium);
  font-size: var(--font-sm);
  cursor: pointer;
  transition: all var(--transition);
}

.btn-primary:hover {
  background: var(--primary-dark);
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.3);
}
</style>
