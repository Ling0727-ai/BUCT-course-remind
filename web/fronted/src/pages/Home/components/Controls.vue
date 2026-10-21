<template>
  <div class="controls">
    <div class="filters">
      <div class="search-box">
        <i class="fas fa-search" aria-hidden="true"></i>
        <label class="sr-only" for="assignmentSearch">搜索作业</label>
        <input
          id="assignmentSearch"
          type="search"
          :value="searchTerm"
          @input="$emit('update:searchTerm', $event.target.value)"
          placeholder="搜索作业标题或科目..."
        >
        <button v-if="searchTerm" type="button" class="clear-search" aria-label="清空搜索" @click="$emit('update:searchTerm', '')">
          <i class="fas fa-times" aria-hidden="true"></i>
        </button>
      </div>

      <button type="button" class="filter-toggle" :aria-expanded="filtersOpen" aria-controls="advanced-filters" @click="filtersOpen = !filtersOpen">
        <i class="fas fa-sliders" aria-hidden="true"></i> 筛选<span v-if="activeFilterCount"> · {{ activeFilterCount }}</span>
      </button>
      <div id="advanced-filters" class="advanced-filters" :class="{ 'is-open': filtersOpen }">
      <label class="sr-only" for="subjectFilter">按科目筛选</label>
      <select id="subjectFilter" class="select" :value="subjectFilter" @change="$emit('update:subjectFilter', $event.target.value)">
        <option value="">全部科目</option>
        <option v-for="s in subjects" :key="s" :value="s">{{ s }}</option>
      </select>

      <label class="sr-only" for="statusFilter">按状态筛选</label>
      <select id="statusFilter" class="select" :value="statusFilter" @change="$emit('update:statusFilter', $event.target.value)">
        <option value="">全部状态</option>
        <option value="urgent">紧急（1天内，含逾期）</option>
        <option value="warning">即将到期（1～3天内）</option>
        <option value="normal">正常（超过3天或未设截止）</option>
        <option value="completed">已完成</option>
      </select>

      <button v-if="searchTerm || subjectFilter || statusFilter" type="button" class="reset-btn" @click="resetFilters">
        重置
      </button>

      <!-- 文字显示"当前"视图状态（配合 aria-pressed 的高亮），
           避免用户读到目标状态却看到相反效果 -->
      <button
        type="button"
        class="view-toggle"
        :aria-pressed="compact"
        :aria-label="compact ? '当前为紧凑视图，点击切换到宽松视图' : '当前为宽松视图，点击切换到紧凑视图'"
        :title="compact ? '当前：紧凑视图（点击切换）' : '当前：宽松视图（点击切换）'"
        @click="$emit('update:compact', !compact)"
      >
        <i :class="compact ? 'fas fa-list' : 'fas fa-th-large'" aria-hidden="true"></i>
        <span>{{ compact ? '紧凑' : '宽松' }}</span>
      </button>
      </div>
    </div>

    <!-- 统计卡同时作为快捷筛选入口：点击既看数也筛列表 -->
    <div class="stats">
      <button type="button" class="stat-card urgent" @click="$emit('show-stat-modal', 'urgent')">
        <span class="stat-num">{{ urgentCount }}</span>
        <span class="stat-label">紧急 · 1天内</span>
      </button>
      <button type="button" class="stat-card warning" @click="$emit('show-stat-modal', 'warning')">
        <span class="stat-num">{{ soonCount }}</span>
        <span class="stat-label">即将到期 · 3天内</span>
      </button>
      <button type="button" class="stat-card total" @click="$emit('show-stat-modal', 'all')">
        <span class="stat-num">{{ totalCount }}</span>
        <span class="stat-label">全部作业</span>
      </button>
      <button type="button" class="stat-card completed" @click="$emit('show-stat-modal', 'completed')">
        <span class="stat-num">{{ completedCount }}</span>
        <span class="stat-label">已完成</span>
      </button>
      <button type="button" class="stat-card todos" @click="$emit('show-stat-modal', 'todos')">
        <span class="stat-num">{{ todoCount }}</span>
        <span class="stat-label">待办</span>
      </button>
    </div>
  </div>
</template>

<script lang="ts">
import { computed, defineComponent, ref } from 'vue'
import type { PropType } from 'vue'

export default defineComponent({
  name: 'Controls',
  props: {
    searchTerm: String,
    subjectFilter: String,
    statusFilter: String,
    subjects: { type: Array as PropType<string[]>, default: () => [] },
    urgentCount: Number,
    soonCount: Number,
    totalCount: Number,
    completedCount: Number,
    todoCount: Number,
    compact: Boolean
  },
  emits: ['update:searchTerm', 'update:subjectFilter', 'update:statusFilter', 'update:compact', 'show-stat-modal'],
  setup(props, { emit }) {
    const filtersOpen = ref(false)
    const activeFilterCount = computed(() => Number(Boolean(props.subjectFilter)) + Number(Boolean(props.statusFilter)))
    const resetFilters = () => {
      emit('update:searchTerm', '')
      emit('update:subjectFilter', '')
      emit('update:statusFilter', '')
    }
    return { resetFilters, filtersOpen, activeFilterCount }
  }
})
</script>

<style scoped>
.advanced-filters { display: contents; }
.filter-toggle { display: none; }
.controls {
  display: flex;
  flex-direction: column;
  gap: 24px;
  padding: 0 var(--space-6) var(--space-4);
}

.filters {
  order: 2;
  display: flex;
  gap: var(--space-2);
  align-items: center;
  flex-wrap: wrap;
  margin-bottom: 0;
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: 18px;
  background: var(--bg);
}

.search-box {
  flex: 1;
  min-width: 220px;
  position: relative;
}

.search-box > i {
  position: absolute;
  left: 12px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-tertiary);
  font-size: 14px;
  pointer-events: none;
}

.search-box input {
  width: 100%;
  min-height: 40px;
  padding: 9px 36px 9px 36px;
  font-size: var(--font-sm);
  color: var(--text-primary);
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  transition: all var(--transition);
}

.search-box input:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--primary-subtle);
}

.clear-search {
  position: absolute;
  right: 6px;
  top: 50%;
  transform: translateY(-50%);
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  font-size: 11px;
  color: var(--text-tertiary);
  background: var(--gray-100);
  border: none;
  border-radius: 50%;
  cursor: pointer;
}

.clear-search:hover {
  background: var(--gray-200);
}

.select {
  min-height: 40px;
  padding: 9px 32px 9px 12px;
  font-size: var(--font-sm);
  color: var(--text-primary);
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all var(--transition);
  appearance: none;
  -webkit-appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 12 12'%3E%3Cpath fill='%239ca3af' d='M6 8L1 3h10z'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 10px center;
}

.select:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--primary-subtle);
}

.reset-btn,
.view-toggle {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 40px;
  padding: 9px 16px;
  font-size: var(--font-sm);
  color: var(--text-secondary);
  white-space: nowrap;
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all var(--transition);
}

.reset-btn:hover,
.view-toggle:hover {
  border-color: var(--primary);
  color: var(--primary);
  background: var(--primary-subtle);
}

.view-toggle[aria-pressed="true"] {
  border-color: var(--primary);
  color: var(--primary);
  background: var(--primary-subtle);
}

/* ── Stats ── */
.stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: var(--space-3);
}

.stat-card {
  position: relative;
  overflow: hidden;
  padding: var(--space-4) var(--space-3);
  font-family: inherit;
  text-align: left;
  padding: 22px;
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  cursor: pointer;
  transition: all var(--transition);
}

.stat-card::before {
  content: '';
  position: absolute;
  top: 24px;
  right: 20px;
  width: 7px;
  height: 7px;
  border-radius: 50%;
}

.stat-card.urgent::before { background: var(--red); }
.stat-card.warning::before { background: var(--orange); }
.stat-card.total::before { background: var(--primary); }
.stat-card.completed::before { background: var(--green); }
.stat-card.todos::before { background: var(--purple); }

.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

.stat-num {
  display: block;
  margin-bottom: 2px;
  font-size: 32px;
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.04em;
  font-weight: var(--font-bold);
  line-height: 1.2;
}

.stat-card.urgent .stat-num { color: var(--red); }
.stat-card.warning .stat-num { color: var(--orange); }
.stat-card.total .stat-num { color: var(--primary); }
.stat-card.completed .stat-num { color: var(--green); }
.stat-card.todos .stat-num { color: var(--purple); }

.stat-label {
  font-size: var(--font-sm);
  color: var(--text-secondary);
}

@media (max-width: 768px) {
  .controls { padding: 0 var(--space-4) var(--space-3); }
  .controls { gap: 14px; }
  .filters { flex-direction: row; align-items: center; padding: 8px; }
  .search-box { min-width: 0; }
  .filter-toggle { display: flex; align-items: center; justify-content: center; gap: 6px; min-height: 44px; padding: 0 12px; border: 1px solid var(--border); border-radius: var(--radius-md); background: var(--bg); color: var(--text-secondary); cursor: pointer; }
  .filter-toggle[aria-expanded="true"] { color: var(--primary); border-color: var(--primary); }
  .advanced-filters { display: none; }
  .advanced-filters.is-open { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; width: 100%; padding-top: 8px; border-top: 1px solid var(--border); }
  .select,
  .reset-btn,
  .view-toggle { width: 100%; justify-content: center; }
  .stats { grid-template-columns: repeat(6, minmax(0, 1fr)); gap: 8px; }
  .stat-card { grid-column: span 3; }
  .stat-card.total, .stat-card.completed, .stat-card.todos { grid-column: span 2; display: flex; flex-direction: row-reverse; justify-content: center; align-items: baseline; gap: 6px; padding: 10px 4px; background: transparent; border-color: transparent; box-shadow: none; }
  .stat-card.total::before, .stat-card.completed::before, .stat-card.todos::before { display: none; }
  .stat-card.total .stat-num, .stat-card.completed .stat-num, .stat-card.todos .stat-num { font-size: 16px; }
  .stat-card { padding: 16px 12px; }
  .stat-card::before { top: 18px; right: 12px; }
  .stat-num { font-size: 26px; }
  .stat-label { font-size: 12px; }
}
</style>
