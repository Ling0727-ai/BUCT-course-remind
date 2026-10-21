<template>
  <div
    :class="['card', cardStatus]"
    :data-type="assignment.type"
    role="link"
    tabindex="0"
    :aria-label="cardAriaLabel"
    @click="$emit('open-url', assignment)"
    @keydown.enter.self.prevent="$emit('open-url', assignment)"
    @keydown.space.self.prevent="$emit('open-url', assignment)"
  >
    <div v-if="assignment.type === '待办' && assignment.priority"
         :class="['priority-dot', assignment.priority]"
         :title="`优先级：${priorityText}`"></div>

    <button
      type="button"
      class="preview-btn"
      :aria-label="`预览 ${assignment.title}`"
      title="预览"
      @click.stop="$emit('show-preview', assignment)"
    >
      <i class="fas fa-eye" aria-hidden="true"></i>
    </button>

    <div class="card-head">
      <span class="tag-subject">{{ assignment.subject }}</span>
      <span class="tag-type">{{ assignment.type }}</span>
    </div>

    <h3 class="card-title">{{ assignment.title }}</h3>
    <p class="card-desc">{{ assignment.description }}</p>

    <div class="card-foot">
      <div v-if="assignment.dueDate" class="due-info">
        <span :class="['due-date', cardStatus]">
          <i :class="statusIcon" aria-hidden="true"></i>
          <span class="sr-only">{{ statusLabel }}</span>
          {{ formattedDateOnly }}
        </span>
        <span v-if="remainingTimeText" class="due-remain">{{ remainingTimeText }}</span>
      </div>

      <div class="actions" @click.stop>
        <template v-if="!assignment.completed">
          <button type="button"
                  class="act-btn remind" :aria-label="`为 ${assignment.title} 设置提醒`" title="设置提醒"
                  @click="$emit('set-reminder', assignment)">
            <i class="fas fa-bell" aria-hidden="true"></i>
          </button>
          <button type="button"
                  class="act-btn done" :aria-label="`标记 ${assignment.title} 为完成`" title="标记完成"
                  :disabled="assignment.completing"
                  @click="$emit('mark-completed', assignment)">
            <i v-if="!assignment.completing" class="fas fa-check" aria-hidden="true"></i>
            <i v-else class="fas fa-spinner fa-spin" aria-hidden="true"></i>
          </button>
        </template>
        <template v-else>
          <span class="done-badge">
            <i class="fas fa-check-circle" aria-hidden="true"></i> 已完成
          </span>
          <button type="button"
                  class="act-btn undo" :aria-label="`撤销 ${assignment.title} 的完成状态`" title="撤销"
                  :disabled="assignment.undoing"
                  @click="$emit('undo-completed', assignment)">
            <i v-if="!assignment.undoing" class="fas fa-undo" aria-hidden="true"></i>
            <i v-else class="fas fa-spinner fa-spin" aria-hidden="true"></i>
          </button>
        </template>
        <details ref="menu" class="card-menu" @keydown.esc.stop.prevent="closeMenu(true)">
          <summary class="act-btn" :aria-label="`${assignment.title} 的更多操作`" title="更多操作"><i class="fas fa-ellipsis-h" aria-hidden="true"></i></summary>
          <div class="card-menu-panel">
            <button v-if="assignment.type !== '待办'" type="button" :aria-label="`拉黑科目 ${assignment.subject}`" @click="closeMenu(true); $emit('blacklist-subject', assignment)"><i class="fas fa-ban" aria-hidden="true"></i> 拉黑此科目</button>
            <button type="button" class="danger" :aria-label="`删除 ${assignment.title}`" @click="closeMenu(true); $emit('delete-assignment', assignment)"><i class="fas fa-trash" aria-hidden="true"></i> 移至回收站</button>
          </div>
        </details>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, ref, onMounted, onUnmounted } from 'vue'
import type { PropType } from 'vue'
import type { Assignment } from '../home.data'
import { parseDueDate } from '@/utils/datetime'
import { getAssignmentStatus } from '@/utils/assignment-status'
import { useTaskClock } from '@/composables/useTaskClock'

type CardStatus = 'urgent' | 'warning' | 'normal' | 'completed'

const STATUS_ICONS: Record<CardStatus, string> = {
  urgent: 'fas fa-exclamation-triangle',
  warning: 'fas fa-clock',
  normal: 'fas fa-calendar-alt',
  completed: 'fas fa-check-circle'
}

const STATUS_LABELS: Record<CardStatus, string> = {
  urgent: '紧急',
  warning: '即将到期',
  normal: '正常',
  completed: '已完成'
}

const PRIORITY_LABELS: Record<string, string> = {
  high: '高',
  medium: '中',
  low: '低'
}

export default defineComponent({
  name: 'AssignmentCard',
  props: {
    assignment: {
      type: Object as PropType<Assignment>,
      required: true
    }
  },
  emits: ['open-url', 'show-preview', 'set-reminder', 'delete-assignment', 'mark-completed', 'undo-completed', 'blacklist-subject'],
  setup(props) {
    const now = useTaskClock()
    const menu = ref<HTMLDetailsElement | null>(null)
    const closeMenu = (restoreFocus = false) => {
      if (!menu.value?.open) return
      menu.value.open = false
      if (restoreFocus) menu.value.querySelector('summary')?.focus()
    }
    const outside = (event: MouseEvent) => {
      if (!menu.value?.contains(event.target as Node)) closeMenu()
    }
    onMounted(() => document.addEventListener('click', outside, true))
    onUnmounted(() => document.removeEventListener('click', outside, true))
    const cardStatus = computed<CardStatus>(() => {
      if (props.assignment.completed) return 'completed'
      return getAssignmentStatus(props.assignment, now.value)
    })

    const statusIcon = computed(() => STATUS_ICONS[cardStatus.value])
    const statusLabel = computed(() => STATUS_LABELS[cardStatus.value])

    const priorityText = computed(
      () => PRIORITY_LABELS[props.assignment.priority ?? ''] ?? ''
    )

    const formattedDateOnly = computed(() => {
      const dueDate = props.assignment.dueDate
      if (!dueDate) return ''
      const date = parseDueDate(dueDate)
      if (!date || Number.isNaN(date.getTime())) return ''
      return date.toLocaleString('zh-CN', {
        year: 'numeric', month: '2-digit', day: '2-digit',
        hour: '2-digit', minute: '2-digit'
      }).replace(/\//g, '-')
    })

    const remainingTimeText = computed(() => {
      const dueDate = props.assignment.dueDate
      if (!dueDate) return ''
      const date = parseDueDate(dueDate)
      if (!date || Number.isNaN(date.getTime())) return ''
      const diff = date.getTime() - now.value
      if (diff < 0) {
        const d = Math.floor(Math.abs(diff) / 86400000)
        const h = Math.floor(Math.abs(diff) / 3600000) % 24
        return `已超时 ${d}天${h ? h + '小时' : ''}`
      }
      const d = Math.floor(diff / 86400000)
      const h = Math.floor(diff / 3600000) % 24
      const m = Math.floor(diff / 60000) % 60
      if (d > 0) return `剩余 ${d}天${h ? h + '小时' : ''}`
      if (h > 0) return `剩余 ${h}小时${m ? m + '分钟' : ''}`
      return `剩余 ${m}分钟`
    })

    const cardAriaLabel = computed(() => {
      const item = props.assignment
      const parts = [`${item.subject} ${item.type}：${item.title}`]
      if (item.dueDate) {
        parts.push(`截止 ${formattedDateOnly.value}`)
      }
      if (remainingTimeText.value) {
        parts.push(remainingTimeText.value)
      }
      parts.push(`状态 ${statusLabel.value}`)
      if (item.url?.trim()) {
        parts.push('按回车打开链接')
      }
      return parts.join('，')
    })

    return {
      menu, closeMenu,
      cardStatus,
      statusIcon,
      statusLabel,
      priorityText,
      formattedDateOnly,
      remainingTimeText,
      cardAriaLabel
    }
  }
})
</script>

<style scoped>
.card:has(.card-menu[open]) { z-index: 5; }
.card-menu { position: relative; }
.card-menu summary { list-style: none; }
.card-menu summary::-webkit-details-marker { display: none; }
.card-menu[open] summary { color: var(--primary); background: var(--primary-subtle); }
.card-menu-panel { position: absolute; right: 0; bottom: calc(100% + 8px); width: 168px; padding: 6px; border: 1px solid var(--border); border-radius: 12px; background: rgba(var(--surface-rgb), 1); box-shadow: var(--shadow-lg); z-index: 6; }
.card-menu-panel button { display: flex; align-items: center; gap: 10px; width: 100%; min-height: 44px; padding: 8px 12px; border: 0; border-radius: 8px; background: transparent; color: var(--text-secondary); font-size: 13px; cursor: pointer; text-align: left; }
.card-menu-panel button:hover { background: var(--gray-100); }
.card-menu-panel button.danger { color: var(--red); }
/* ── Card Base ── */
.card {
  position: relative;
  display: flex;
  flex-direction: column;
  min-height: 0;
  padding: 24px;
  background: var(--bg);
  border: 1px solid var(--border);
  border-left: 3px solid var(--primary);
  border-radius: var(--radius-lg);
  box-shadow: 0 3px 16px rgba(30, 60, 80, 0.025);
  transition: transform 240ms ease, box-shadow 240ms ease, border-color 240ms ease;
}

.card:hover {
  box-shadow: var(--shadow-lg);
  border-color: var(--gray-200);
}

@media (hover: hover) { .card:hover { transform: translateY(-4px); } }

.card:focus-visible {
  outline: 2px solid var(--primary);
  outline-offset: 2px;
}

/* Status colors */
.card.urgent { border-left-color: var(--red); }
.card.warning { border-left-color: var(--orange); }
.card.normal { border-left-color: var(--primary); }
.card.completed { border-left-color: var(--green); background: color-mix(in srgb, var(--green-subtle) calc(var(--card-alpha) * 100%), transparent); }
.card[data-type="待办"] { border-left-color: var(--purple); }

/* ── Priority dot (todo only) ── */
.priority-dot {
  position: absolute;
  top: var(--space-4);
  right: 50px;
  width: 8px;
  height: 8px;
  border-radius: 50%;
}
.priority-dot.high { background: var(--red); }
.priority-dot.medium { background: var(--orange); }
.priority-dot.low { background: var(--green); }

/* ── Preview button ── */
.preview-btn {
  position: absolute;
  top: var(--space-3);
  right: var(--space-3);
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  padding: 0;
  font-size: 13px;
  color: var(--text-tertiary);
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all var(--transition);
}
.preview-btn:hover {
  border-color: var(--primary);
  color: var(--primary);
  background: var(--primary-subtle);
}

/* ── Card Head ── */
.card-head {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin-bottom: var(--space-3);
  padding-right: 44px;
}

.tag-subject {
  padding: 3px 10px;
  font-size: var(--font-xs);
  font-weight: var(--font-semibold);
  color: var(--primary-dark);
  background: var(--primary-subtle);
  border-radius: var(--radius-full);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 60%;
}

.card[data-type="待办"] .tag-subject {
  color: var(--purple);
  background: var(--purple-subtle);
}

.tag-type {
  padding: 3px 10px;
  font-size: var(--font-xs);
  color: var(--text-tertiary);
  background: var(--gray-100);
  border-radius: var(--radius-full);
}

/* ── Title & Desc ──
   用 CSS 行数截断代替 JS 字符截断：中文不会被切到半个词，且随容器宽度自适应。
   宽松视图下列宽更大，描述多给一行。 */
.card-title {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  margin-bottom: var(--space-2);
  font-size: var(--font-base);
  font-weight: var(--font-semibold);
  line-height: 1.4;
  color: var(--text-primary);
  overflow-wrap: anywhere;
}

.card-desc {
  display: -webkit-box;
  -webkit-line-clamp: 3;
  line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
  flex: 1;
  margin-bottom: var(--space-4);
  font-size: var(--font-sm);
  line-height: 1.5;
  color: var(--text-secondary);
  overflow-wrap: anywhere;
}

.card.completed .card-title {
  color: var(--text-tertiary);
  text-decoration: line-through;
  text-decoration-color: var(--gray-300);
}

.card.completed .card-desc {
  color: var(--text-tertiary);
}

/* ── Card Footer ── */
.card-foot {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: var(--space-3);
  padding-top: var(--space-3);
  border-top: 1px solid var(--gray-100);
}

.due-info {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.due-date {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: var(--font-xs);
  font-weight: var(--font-medium);
  /* 日期与时间是一个整体，不允许在"2026-09-30"中间断行 */
  white-space: nowrap;
}

.due-date i {
  font-size: 12px;
  width: 14px;
}

.due-date.urgent { color: var(--red); }
.due-date.warning { color: var(--orange); }
.due-date.normal { color: var(--text-secondary); }
.due-date.completed { color: var(--green); }

.due-remain {
  padding-left: 18px;
  font-size: var(--font-xs);
  color: var(--text-tertiary);
  white-space: nowrap;
}

/* ── Actions ── */
.actions {
  display: flex;
  gap: var(--space-2);
  flex-shrink: 0;
}

.act-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  padding: 0;
  font-size: 13px;
  color: var(--text-tertiary);
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all var(--transition);
}

.act-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.act-btn.done { color: var(--green); border-color: transparent; background: var(--green-subtle); }
.act-btn.done:hover { background: var(--green); color: white; }
.act-btn.delete { color: var(--red); border-color: transparent; background: var(--red-subtle); }
.act-btn.delete:hover { background: var(--red); color: white; }
.act-btn.remind { color: var(--orange); border-color: transparent; background: var(--orange-subtle); }
.act-btn.remind:hover { background: var(--orange); color: white; }
.act-btn.blacklist { color: var(--red); border-color: transparent; background: var(--red-subtle); }
.act-btn.blacklist:hover { background: var(--red); color: white; }
.act-btn.undo { color: var(--gray-500); border-color: transparent; background: var(--gray-50); }
.act-btn.undo:hover { background: var(--gray-200); color: var(--text-primary); }

.done-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 36px;
  padding: 0 10px;
  font-size: var(--font-xs);
  font-weight: var(--font-medium);
  color: var(--green);
  background: var(--green-subtle);
  border-radius: var(--radius-sm);
  white-space: nowrap;
}

/* ── Responsive ── */
@media (max-width: 768px) {
  .card { padding: var(--space-4); }
  .card-foot { flex-direction: column; align-items: flex-start; gap: var(--space-3); }
  .actions { align-self: stretch; justify-content: flex-end; }
}

@media (max-width: 480px) {
  .card { padding: var(--space-3); }
  .card-title { font-size: var(--font-sm); }
}
</style>
