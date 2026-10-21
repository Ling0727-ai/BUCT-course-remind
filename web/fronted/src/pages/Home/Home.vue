<template>
  <div class="page workspace">
    <Header
      :current-time="currentTime"
      :user="user"
      @show-recycle-bin="openRecycleBin"
      @logout="handleLogout"
      @refresh="() => fetchAssignments(true)"
      @show-blacklist="showBlacklistModal = true"
    />

    <Controls
      v-model:search-term="searchTerm"
      v-model:subject-filter="subjectFilter"
      v-model:status-filter="statusFilter"
      :subjects="subjects"
      :urgent-count="urgentCount"
      :soon-count="soonCount"
      :total-count="totalCount"
      :completed-count="completedCount"
      :todo-count="todoCount"
      :compact="compactView"
      @update:compact="setCompactView"
      @show-stat-modal="openStatModal"
    />

    <div class="section-heading">
      <div><span class="section-eyebrow">YOUR STUDY SPACE</span><h2>学习任务</h2></div>
      <span class="result-count">{{ filteredAssignments.length }} 项任务<span v-if="filtersActive"> · 已筛选</span></span>
    </div>

    <AssignmentGrid
      :assignments="filteredAssignments"
      :loading="loading"
      :error="error"
      :compact="compactView"
      :filters-active="filtersActive"
      :status-filter="statusFilter"
      @clear-filters="clearFilters"
      @refresh="handleGridRefresh"
      @open-url="openAssignmentUrl"
      @show-preview="openPreview"
      @set-reminder="openReminderModal"
      @delete-assignment="confirmDelete"
      @mark-completed="handleComplete"
      @undo-completed="handleUndoComplete"
      @blacklist-subject="confirmBlacklistSubject"
    />

    <FloatingAddButton @show-add-todo="showAddTodoModal = true" />

    <!-- Modals -->
    <AddTodoModal
      :show="showAddTodoModal"
      v-model:todo="newTodo"
      :adding="addingTodo"
      @close="closeAddTodoModal"
      @add="onAddTodo"
    />
    <PreviewModal
      :show="showPreviewModal" :item="previewItem"
      @close="closePreviewModal" @open-url="openAssignmentUrl"
    />
    <ConfirmModal
      :show="showConfirmModal" :message="confirmMessage" :busy="confirmBusy"
      @confirm="onConfirmAction" @cancel="onCancelConfirm"
    />
    <StatisticsModal
      :show="showStatisticsModal" :type="currentStatType"
      :items="filteredStatItems" @close="closeStatModal"
      @open-url="openAssignmentUrl" @show-preview="openPreview"
      @set-reminder="openReminderModal" @delete-assignment="confirmDelete"
      @mark-completed="handleComplete" @undo-completed="handleUndoComplete"
      @blacklist-subject="confirmBlacklistSubject"
    />
    <RecycleBinModal
      :show="showRecycleBinModal" :items="deletedItems"
      :loading="recycleBinLoading" :restoring-all="restoringAllRecycle"
      @close="showRecycleBinModal = false"
      @restore-item="handleRestoreItem" @permanent-delete="onPermanentDelete"
      @restore-all="onRestoreAll"
    />
    <ReminderModal
      :assignment="reminderAssignment" :show="showReminderModal"
      @close="closeReminderModal" @confirm="onConfirmReminder"
    />
    <BlacklistModal
      :show="showBlacklistModal"
      @close="showBlacklistModal = false"
      @updated="fetchAssignments(false)"
    />

    <div class="footer">
      <span>联系邮箱：<a href="mailto:lingxin@lingxin.org">lingxin@lingxin.org</a></span>
      <span class="sep">|</span>
      <span>GitHub：<a href="https://github.com/ling0727-ai" target="_blank" rel="noopener">lingxin</a></span>
    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent, computed, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useHomeService } from './home';
import { useToast } from '@/composables/useToast';
import * as api from './home.api';
import Header from './components/Header.vue';
import Controls from './components/Controls.vue';
import AssignmentGrid from './components/AssignmentGrid.vue';
import FloatingAddButton from './components/FloatingAddButton.vue';
import AddTodoModal from './components/modals/AddTodoModal.vue';
import PreviewModal from './components/modals/PreviewModal.vue';
import ConfirmModal from './components/modals/ConfirmModal.vue';
import StatisticsModal from './components/modals/StatisticsModal.vue';
import RecycleBinModal from './components/modals/RecycleBinModal.vue';
import ReminderModal from './components/modals/ReminderModal.vue';
import BlacklistModal from './components/modals/BlacklistModal.vue';
import type { Assignment, Todo, ReminderConfig } from './home.data';

const COMPACT_KEY = 'buct-compact-view';

export default defineComponent({
  name: 'Home',
  components: {
    Header, Controls, AssignmentGrid, FloatingAddButton,
    AddTodoModal, PreviewModal, ConfirmModal, StatisticsModal,
    RecycleBinModal, ReminderModal, BlacklistModal
  },
  setup() {
    const service = useHomeService();
    const { showToast } = useToast();
    const route = useRoute();
    const router = useRouter();

    // ─── 筛选状态同步到 URL，便于刷新保留和分享链接 ───
    const searchTerm = service.searchTerm;
    const subjectFilter = service.subjectFilter;
    const statusFilter = service.statusFilter;

    const applyQuery = (query: Record<string, unknown>) => {
      const q = (key: string) => (typeof query[key] === 'string' ? (query[key] as string) : '');
      searchTerm.value = q('q');
      subjectFilter.value = q('subject');
      statusFilter.value = q('status');
    };

    applyQuery(route.query);

    let syncingFromUrl = false;
    watch(
      () => route.query,
      (query) => {
        syncingFromUrl = true;
        applyQuery(query);
        syncingFromUrl = false;
      }
    );

    watch([searchTerm, subjectFilter, statusFilter], () => {
      if (syncingFromUrl) return;
      const query: Record<string, string> = {};
      if (searchTerm.value) query.q = searchTerm.value;
      if (subjectFilter.value) query.subject = subjectFilter.value;
      if (statusFilter.value) query.status = statusFilter.value;
      // 使用 replace 避免每个字符都产生一条历史记录
      router.replace({ query });
    });

    const filtersActive = computed(
      () => Boolean(searchTerm.value || subjectFilter.value || statusFilter.value)
    );

    const clearFilters = () => {
      searchTerm.value = '';
      subjectFilter.value = '';
      statusFilter.value = '';
    };

    // ─── 紧凑视图偏好 ───
    const compactView = ref(false);
    try {
      compactView.value = localStorage.getItem(COMPACT_KEY) === '1';
    } catch {
      // localStorage 不可用时使用默认值
    }

    const setCompactView = (value: boolean) => {
      compactView.value = value;
      try {
        localStorage.setItem(COMPACT_KEY, value ? '1' : '0');
      } catch {
        // 忽略写入失败
      }
    };

    const showAddTodoModal = ref(false);
    const addingTodo = ref(false);
    const newTodo = ref<Partial<Todo>>({
      title: '', description: '', hours: 24, timeInput: '24:00:00', priority: 'medium'
    });
    const closeAddTodoModal = () => {
      showAddTodoModal.value = false;
      newTodo.value = { title: '', description: '', hours: 24, timeInput: '24:00:00', priority: 'medium' };
    };
    const onAddTodo = async () => {
      if (!newTodo.value.title?.trim()) { showToast('warning', '提示', '请输入待办事项名称'); return; }
      addingTodo.value = true;
      const ok = await service.handleAddTodo(newTodo.value);
      addingTodo.value = false;
      if (ok) closeAddTodoModal();
    };

    const showPreviewModal = ref(false);
    const previewItem = ref<Assignment | null>(null);
    const openPreview = (item: Assignment) => { previewItem.value = item; showPreviewModal.value = true; };
    const closePreviewModal = () => { showPreviewModal.value = false; previewItem.value = null; };
    const openAssignmentUrl = (item: Assignment) => { if (item.url?.trim()) window.open(item.url, '_blank'); };

    const showConfirmModal = ref(false);
    const confirmMessage = ref('');
    const confirmBusy = ref(false);
    let confirmCallback: (() => void | Promise<unknown>) | null = null;
    const onConfirmAction = async () => {
      if (confirmBusy.value) return;
      confirmBusy.value = true;
      try {
        await confirmCallback?.();
        showConfirmModal.value = false;
        confirmCallback = null;
      } catch (error) {
        showToast('error', '操作失败', error instanceof Error ? error.message : '请稍后重试');
      } finally {
        confirmBusy.value = false;
      }
    };
    const onCancelConfirm = () => { if (!confirmBusy.value) { showConfirmModal.value = false; confirmCallback = null; } };
    const showCustomConfirm = (msg: string, cb: () => void | Promise<unknown>) => { confirmMessage.value = msg; confirmCallback = cb; showConfirmModal.value = true; };

    const showStatisticsModal = ref(false);
    const currentStatType = ref('');
    const filteredStatItems = computed(() => service.allItems.value.filter(item => {
      if (currentStatType.value === 'urgent') return service.assignmentStatus(item) === 'urgent';
      if (currentStatType.value === 'warning') return service.assignmentStatus(item) === 'warning';
      if (currentStatType.value === 'completed') return item.completed;
      if (currentStatType.value === 'todos') return item.type === '待办' && !item.completed;
      return true;
    }));
    const openStatModal = (type: string) => {
      currentStatType.value = type;
      showStatisticsModal.value = true;
    };
    const closeStatModal = () => { showStatisticsModal.value = false; };

    const handleGridRefresh = (notify = false) => service.fetchAssignments(notify);

    const showRecycleBinModal = ref(false);
    const openRecycleBin = async () => { await service.loadRecycleBin(); showRecycleBinModal.value = true; };
    const onPermanentDelete = async (item: any) => {
      showCustomConfirm(`确定要永久删除"${item.title}"吗？`, () => service.handlePermanentDelete(item));
    };

    // 「全部恢复」没有后端批量接口，是前端逐条调用，
    // 项数可能较多，用状态位防止重复点击并给出进度反馈。
    const restoringAllRecycle = ref(false);
    const onRestoreAll = async () => {
      if (restoringAllRecycle.value) return;
      restoringAllRecycle.value = true;
      try {
        await service.handleRestoreAll();
      } finally {
        restoringAllRecycle.value = false;
      }
    };

    const showReminderModal = ref(false);
    const reminderAssignment = ref<Assignment | null>(null);
    const openReminderModal = (item: Assignment) => { reminderAssignment.value = item; showReminderModal.value = true; };
    const closeReminderModal = () => { showReminderModal.value = false; reminderAssignment.value = null; };
    const onConfirmReminder = async (config: ReminderConfig) => {
      if (reminderAssignment.value) {
        await service.handleSetReminder(reminderAssignment.value, config);
        closeReminderModal();
      }
    };

    const showBlacklistModal = ref(false);
    const extractCourseID = (url?: string) => { const m = url?.match(/[?&]courseId=(\d+)/i); return m ? m[1] : null; };
    const confirmBlacklistSubject = (a: Assignment) => {
      showCustomConfirm(`确定要拉黑科目"${a.subject}"吗？`, async () => {
        const cid = extractCourseID(a.url);
        if (!cid) throw new Error('该作业缺少课程链接，无法识别科目');
        await api.addToBlacklist(cid);
        showToast('success', 'Success', `"${a.subject}"已加入黑名单`);
        await service.fetchAssignments(false);
      });
    };
    const confirmDelete = (item: Assignment) => {
      showCustomConfirm(`确定要删除"${item.title}"吗？`, () => service.handleDelete(item));
    };

    return {
      ...service,
      filtersActive, clearFilters, compactView, setCompactView,
      showAddTodoModal, addingTodo, newTodo, closeAddTodoModal, onAddTodo,
      showPreviewModal, previewItem, openPreview, closePreviewModal, openAssignmentUrl,
      showConfirmModal, confirmMessage, confirmBusy, onConfirmAction, onCancelConfirm,
      showStatisticsModal, currentStatType, filteredStatItems, openStatModal, closeStatModal,
      handleGridRefresh, showRecycleBinModal, openRecycleBin, onPermanentDelete,
      restoringAllRecycle, onRestoreAll,
      showReminderModal, reminderAssignment, openReminderModal, closeReminderModal, onConfirmReminder,
      showBlacklistModal, confirmBlacklistSubject, confirmDelete
    };
  }
});
</script>

<style scoped>
.page {
  max-width: 1200px;
  margin: 0 auto;
  padding-bottom: 80px;
  min-height: 100vh;
}

.section-heading { display: flex; align-items: flex-end; justify-content: space-between; gap: 16px; padding: 12px 24px 20px; }
.section-eyebrow { font-size: 10px; letter-spacing: 0.18em; color: var(--text-tertiary); font-weight: 600; }
.section-heading h2 { font-size: 22px; font-weight: 600; margin-top: 5px; letter-spacing: -0.03em; }
.result-count { font-size: 12px; color: var(--text-tertiary); padding-bottom: 3px; }
@media (max-width: 768px) { .section-heading { padding: 12px 16px 16px; } }

.footer {
  text-align: center;
  padding: var(--space-6) var(--space-4);
  font-size: var(--font-sm);
  color: var(--text-tertiary);
  border-top: 1px solid var(--border);
  margin: var(--space-4) var(--space-6) 0;
}

.footer a { color: var(--text-tertiary); }
.footer a:hover { color: var(--primary); }

.sep { margin: 0 var(--space-2); color: var(--border); }

@media (max-width: 768px) {
  .footer { margin: var(--space-4); }
}
</style>
