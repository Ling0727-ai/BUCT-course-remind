<template>
  <div class="content">
    <!-- 首屏骨架屏：避免整页闪成"加载中"再闪回 -->
    <div v-if="loading && assignments.length === 0" class="grid" aria-busy="true" aria-live="polite">
      <span class="sr-only">正在加载作业数据</span>
      <div v-for="n in skeletonCount" :key="n" class="skeleton-card" aria-hidden="true">
        <div class="sk-line sk-tags"></div>
        <div class="sk-line sk-title"></div>
        <div class="sk-line sk-desc"></div>
        <div class="sk-line sk-foot"></div>
      </div>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="state error" role="alert">
      <i class="fas fa-exclamation-triangle" aria-hidden="true"></i>
      <h3>加载失败</h3>
      <p>{{ error }}</p>
      <button type="button" class="btn" @click="$emit('refresh', true)">重新加载</button>
    </div>

    <!-- Grid -->
    <div v-else class="task-board" :style="{ '--task-columns': Math.max(1, groups.filter(group => group.key !== 'completed').length) }">
      <div v-if="assignments.length === 0" key="empty" class="state empty">
        <i class="fas fa-inbox" aria-hidden="true"></i>
        <h3>{{ emptyTitle }}</h3>
        <p>{{ emptyHint }}</p>
        <button v-if="filtersActive" type="button" class="btn" @click="$emit('clear-filters')">
          清除筛选条件
        </button>
      </div>
      <section v-for="group in groups" :key="group.key" class="task-group" :class="group.key">
        <h3 class="group-heading">
          <button v-if="group.key === 'completed'" type="button" class="group-toggle" :aria-expanded="completedOpen" aria-controls="completed-tasks" @click="completedOpen = !completedOpen">
            <i :class="completedOpen ? 'fas fa-chevron-down' : 'fas fa-chevron-right'" aria-hidden="true"></i>
            {{ group.label }} <span>{{ group.items.length }}</span>
          </button>
          <template v-else>{{ group.label }} <span>{{ group.items.length }}</span><small>{{ group.hint }}</small></template>
        </h3>
        <TransitionGroup v-if="group.key !== 'completed' || completedOpen" :id="group.key === 'completed' ? 'completed-tasks' : undefined" name="assignment" tag="div" class="grid" :class="{ 'grid-compact': compact }">
        <AssignmentCard
        v-for="a in group.items" :key="a.id"
        :assignment="a"
        @open-url="$emit('open-url', $event)"
        @show-preview="$emit('show-preview', $event)"
        @set-reminder="$emit('set-reminder', $event)"
        @delete-assignment="$emit('delete-assignment', $event)"
        @mark-completed="$emit('mark-completed', $event)"
        @undo-completed="$emit('undo-completed', $event)"
        @blacklist-subject="$emit('blacklist-subject', $event)"
      />
        </TransitionGroup>
      </section>
    </div>

    <!-- 后台刷新指示：不遮挡已有列表 -->
    <div v-if="loading && assignments.length > 0" class="refresh-bar" role="status">
      <i class="fas fa-spinner fa-spin" aria-hidden="true"></i>
      正在刷新…
    </div>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, ref, watch } from 'vue'
import { parseDueDate } from '@/utils/datetime'
import { getAssignmentStatus } from '@/utils/assignment-status'
import { useTaskClock } from '@/composables/useTaskClock'
import type { PropType } from 'vue'
import AssignmentCard from './AssignmentCard.vue'
import type { Assignment } from '../home.data'

export default defineComponent({
  name: 'AssignmentGrid',
  components: { AssignmentCard },
  props: {
    assignments: { type: Array as PropType<Assignment[]>, required: true },
    loading: Boolean,
    error: { type: String, default: '' },
    /** 紧凑视图：单列窄卡片，一屏能看到更多条目。 */
    compact: { type: Boolean, default: false },
    /** 是否有筛选条件生效，用于给出"清除筛选"出口。 */
    filtersActive: { type: Boolean, default: false },
    statusFilter: { type: String, default: '' },
    emptyTitle: { type: String, default: '暂无作业' },
    emptyHint: { type: String, default: '没有找到符合条件的作业' },
    skeletonCount: { type: Number, default: 6 }
  },
  emits: ['refresh', 'clear-filters', 'open-url', 'show-preview', 'set-reminder', 'delete-assignment', 'mark-completed', 'undo-completed', 'blacklist-subject'],
  setup(props) {
    const completedOpen = ref(false)
    watch(() => props.statusFilter, value => { if (value === 'completed') completedOpen.value = true }, { immediate: true })
    const now = useTaskClock()
    const deadline = (a: Assignment) => {
      const value = a.dueDate ? parseDueDate(a.dueDate)?.getTime() : undefined
      return value !== undefined && Number.isFinite(value) ? value : Infinity
    }
    const groups = computed(() => {
      if (!props.assignments.length) return []
      const buckets = [
        { key: 'urgent', label: '紧急', hint: '1天内 · 含逾期', items: [] as Assignment[] },
        { key: 'warning', label: '即将到期', hint: '1～3天内', items: [] as Assignment[] },
        { key: 'normal', label: '正常', hint: '超过3天 · 或未设截止', items: [] as Assignment[] },
        { key: 'completed', label: '已完成', hint: '', items: [] as Assignment[] }
      ]
      for (const item of [...props.assignments].sort((a, b) => deadline(a) - deadline(b))) {
        buckets.find(group => group.key === getAssignmentStatus(item, now.value))?.items.push(item)
      }
      return buckets.filter(group => group.items.length > 0)
    })
    return { groups, completedOpen }
  }
})
</script>

<style scoped>
.content {
  padding: 0 var(--space-6) var(--space-8);
}
.task-board { display: grid; grid-template-columns: repeat(var(--task-columns, 3), minmax(0, 1fr)); gap: 16px; align-items: start; }
.task-group { --group-bg: var(--primary-subtle); min-width: 0; padding: 14px 10px; border-top: 3px solid var(--border); border-radius: 12px; background: color-mix(in srgb, var(--group-bg) calc(var(--card-alpha) * 100%), transparent); }
.task-group.urgent { --group-bg: var(--red-subtle); border-top-color: var(--red); }
.task-group.warning { --group-bg: var(--orange-subtle); border-top-color: var(--orange); }
.task-group.normal { border-top-color: var(--primary); }
.task-group.completed { grid-column: 1 / -1; background: transparent; }
.task-group:not(.completed) .grid { grid-template-columns: minmax(0, 1fr); gap: 12px; }
.task-group:not(.completed) :deep(.card-foot) { flex-wrap: wrap; }
.task-group:not(.completed) :deep(.actions) { margin-left: auto; }
.task-group :deep(.grid-compact .card) { padding: 16px; }
.task-board > .state { grid-column: 1 / -1; }
.group-heading { flex-wrap: wrap; }
.group-heading small { width: 100%; font-weight: 400; font-size: 11px; color: var(--text-tertiary); }
.group-heading { display: flex; align-items: center; gap: 8px; margin-bottom: 12px; font-size: 14px; font-weight: 600; color: var(--text-secondary); }
.group-heading span { font-size: 12px; font-weight: 400; color: var(--text-tertiary); }
.group-toggle { display: flex; align-items: center; gap: 8px; min-height: 44px; border: 0; background: transparent; color: inherit; font: inherit; cursor: pointer; }

/* 宽松视图（默认）：卡片更宽、留白更大、描述多显示一行。 */
.grid {
  position: relative;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(440px, 100%), 1fr));
  gap: var(--space-6);
  align-items: start;
}

/* 紧凑视图：沿用原来的默认密度（340px 列 + 16px 间距），一屏看到更多条目。
   列宽下限保持在能容纳"2026-09-30 23:59"单行显示的宽度。 */
.grid-compact {
  grid-template-columns: repeat(auto-fill, minmax(min(340px, 100%), 1fr));
  gap: var(--space-4);
}

.state {
  grid-column: 1 / -1;
  padding: 60px 20px;
  text-align: center;
  color: var(--text-tertiary);
}

.state i {
  display: block;
  margin-bottom: var(--space-4);
  font-size: 2.5rem;
}

.state h3 {
  margin-bottom: var(--space-2);
  font-size: var(--font-lg);
  font-weight: var(--font-semibold);
  color: var(--text-secondary);
}

.state p {
  margin-bottom: var(--space-4);
  font-size: var(--font-sm);
}

.state.error i { color: var(--red); }

.btn {
  min-height: 44px;
  padding: 8px 20px;
  font-size: var(--font-sm);
  color: var(--text-secondary);
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all var(--transition);
}

.btn:hover {
  border-color: var(--primary);
  color: var(--primary);
}

/* ── 骨架屏 ── */
.skeleton-card {
  padding: var(--space-5);
  background: var(--bg);
  border: 1px solid var(--border);
  border-left: 4px solid var(--gray-200);
  border-radius: var(--radius-lg);
}

.sk-line {
  height: 12px;
  margin-bottom: var(--space-3);
  background: linear-gradient(90deg, var(--gray-100) 25%, var(--gray-200) 37%, var(--gray-100) 63%);
  background-size: 400% 100%;
  border-radius: var(--radius-sm);
  animation: sk-shimmer 1.4s ease infinite;
}

.sk-tags { width: 45%; height: 18px; }
.sk-title { width: 85%; }
.sk-desc { width: 65%; margin-bottom: var(--space-5); }
.sk-foot { width: 40%; margin-bottom: 0; }

@keyframes sk-shimmer {
  0% { background-position: 100% 50%; }
  100% { background-position: 0 50%; }
}

@media (prefers-reduced-motion: reduce) {
  .sk-line { animation: none; }
}

/* ── 后台刷新条 ── */
.refresh-bar {
  position: sticky;
  bottom: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
  margin-top: var(--space-4);
  padding: var(--space-2);
  font-size: var(--font-xs);
  color: var(--text-secondary);
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: var(--radius-full);
  box-shadow: var(--shadow-sm);
  width: fit-content;
  margin-left: auto;
  margin-right: auto;
}

@media (max-width: 768px) {
  .task-board { grid-template-columns: minmax(0, 1fr); }
  .content { padding: 0 var(--space-4) var(--space-6); }
  .grid { grid-template-columns: 1fr; gap: var(--space-3); }
  .grid-compact { grid-template-columns: 1fr; }
}
</style>
