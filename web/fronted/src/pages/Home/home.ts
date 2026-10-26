import { ref, computed, onMounted, onUnmounted } from 'vue';
import { useRouter } from 'vue-router';
import { useToast } from '@/composables/useToast';
import * as api from './home.api';
import { getAssignmentStatus } from '@/utils/assignment-status';
import { useTaskClock } from '@/composables/useTaskClock';
import {
    Assignment,
    AssignmentRaw,
    AssignmentStatus,
    AssignmentWithStatus,
    AssignmentsResponse,
    ReminderConfig,
    RecycleBinItem,
    Todo,
    TodosResponse,
    User
} from './home.data';

function getErrorMessage(error: unknown, fallback: string): string {
    if (error instanceof Error && error.message) {
        return error.message;
    }
    return fallback;
}

function normalizeAssignmentsPayload(payload: AssignmentsResponse | AssignmentRaw[]): AssignmentRaw[] {
    if (Array.isArray(payload)) {
        return payload;
    }
    if (Array.isArray(payload.tasks)) {
        return payload.tasks;
    }
    return [];
}

function normalizeTodosPayload(payload: Todo[] | TodosResponse): Todo[] {
    if (Array.isArray(payload)) {
        return payload;
    }
    if (payload.success && Array.isArray(payload.data)) {
        return payload.data;
    }
    return [];
}

export function useHomeService() {
    const taskNow = useTaskClock();
    const router = useRouter();
    const { showToast } = useToast();

    // --- State ---
    const user = ref<User | null>(null);
    const currentTime = ref('');
    const assignments = ref<Assignment[]>([]);
    const todos = ref<Todo[]>([]);
    const loading = ref(false);
    const error = ref('');

    // Filters
    const searchTerm = ref('');
    const subjectFilter = ref('');
    const statusFilter = ref('');

    // --- Helpers ---
    const parseDate = (s: string | undefined | null): Date | null => {
        if (!s) return null;
        if (s.includes('+') || s.endsWith('Z')) return new Date(s);
        if (/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/.test(s)) return new Date(s.replace(' ', 'T') + '+08:00');
        if (s.includes('T')) return new Date(s + '+08:00');
        // Chinese format: 2025年9月23日 23:59:00
        const cn = s.match(/^(\d{4})年(\d{1,2})月(\d{1,2})日\s+(\d{2}:\d{2}:\d{2})$/);
        if (cn) {
            const [, y, mo, d, t] = cn;
            return new Date(`${y}-${mo.padStart(2, '0')}-${d.padStart(2, '0')}T${t}+08:00`);
        }
        return new Date(s);
    };

    const assignmentStatus = (item: Assignment): AssignmentStatus => {
        if (item.completed) return 'completed';
        return getAssignmentStatus(item, taskNow.value);
    };

    // --- Computed ---
    const allItems = computed(() => {
        const items: Assignment[] = [...assignments.value];
        // Merge todos
        const todoItems: Assignment[] = todos.value.map(todo => ({
            id: `todo_${todo._id}`,
            _todoId: todo._id,
            subject: '待办事项',
            title: todo.title,
            description: todo.description || '无备注', // map description to content/description
            dueDate: todo.due_date,
            type: '待办',
            completed: todo.completed,
            priority: todo.priority,
            estimatedHours: todo.estimated_hours,
            publisher: '用户'
        } as Assignment));
        items.push(...todoItems);
        return items;
    });

    const classifiedItems = computed<AssignmentWithStatus[]>(() => allItems.value.map(item => ({
            ...item,
            publisher: item.publisher || (item.type === '待办' ? '用户' : '系统'),
            url: item.url || '',
            status: assignmentStatus(item)
        })));
    const filteredAssignments = computed<AssignmentWithStatus[]>(() => {
        const search = searchTerm.value.toLowerCase();
        return classifiedItems.value.filter(item => {
            const title = item.title || '';
            const subject = item.subject || '';

            const matchesSearch = title.toLowerCase().includes(search) || subject.toLowerCase().includes(search);
            const matchesSubject = !subjectFilter.value || subject === subjectFilter.value;
            const matchesStatus = !statusFilter.value || item.status === statusFilter.value;

            return matchesSearch && matchesSubject && matchesStatus;
        });
    });

    const subjects = computed(() => Array.from(new Set(allItems.value.map(a => a.subject))));
    const itemCounts = computed(() => {
        const counts = { urgent: 0, warning: 0, completed: 0 };
        for (const item of classifiedItems.value) {
            if (item.status === 'urgent') counts.urgent++;
            if (item.status === 'warning') counts.warning++;
            if (item.completed) counts.completed++;
        }
        return counts;
    });
    const urgentCount = computed(() => itemCounts.value.urgent);
    const soonCount = computed(() => itemCounts.value.warning);
    const completedCount = computed(() => itemCounts.value.completed);
    const todoCount = computed(() => todos.value.filter(t => !t.completed).length);
    const totalCount = computed(() => allItems.value.length);

    // --- Actions ---
    const fetchAssignments = async (showNotification = false) => {
        loading.value = true;
        error.value = '';
        if (showNotification) showToast('info', '刷新数据', '正在刷新作业数据...');

        try {
            const [assignmentsRes, todosRes] = await Promise.allSettled([
                api.getAssignments(),
                api.getTodos()
            ]);

            // Handle Assignments
            if (assignmentsRes.status === 'fulfilled') {
                const dataArray = normalizeAssignmentsPayload(assignmentsRes.value);

                assignments.value = dataArray.map((item) => {
                    const rawDeadline = item.details?.deadline || item.deadline;
                    const parsed = parseDate(rawDeadline);
                    return {
                        id: item.id,
                        title: item.details?.task || item.title || '未知任务',
                        subject: item.subject || '未知科目',
                        description: item.details?.details_content || '',
                        dueDate: parsed ? parsed.toISOString() : undefined,
                        type: item.type === 'homework' ? '作业' : '测试',
                        completed: Boolean(item.completed),
                        url: item.details?.url || item.url || '',
                        publisher: '系统'
                    };
                });
            }

            // Handle Todos
            if (todosRes.status === 'fulfilled') {
                 todos.value = normalizeTodosPayload(todosRes.value);
            }

        } catch (err: unknown) {
            console.error('Fetch error:', err);
            error.value = '加载失败';
            if (showNotification) showToast('error', 'Error', '数据加载失败');
        } finally {
            loading.value = false;
        }
    };

    const handleLogout = async () => {
        try {
            await api.logout();
            user.value = null;
            localStorage.removeItem('user');
            sessionStorage.removeItem('user');
            router.push('/login');
            showToast('success', 'Goodbye', '已退出登录');
        } catch (e) {
            router.push('/login');
        }
    };

    const updateTime = () => {
        const now = new Date();
        currentTime.value = now.toLocaleTimeString('zh-CN', { hour12: false });
    };

    let timer: number;

    const init = async () => {
        timer = setInterval(updateTime, 1000);
        updateTime();

        // Load User
        try {
            user.value = await api.getUserInfo();
        } catch (e) {
            console.warn('Failed to load user info');
        }

        await fetchAssignments();
    };

    onMounted(init);
    onUnmounted(() => clearInterval(timer));

    // --- CRUD Operations ---

    // Example: Delete Assignment/Todo
    const handleDelete = async (item: Assignment) => {
        try {
            if (item.type === '待办') {
                if(!item._todoId) return;
                await api.deleteTodo(item._todoId);
                // Refresh todos locally
                todos.value = todos.value.filter(t => t._id !== item._todoId);
            } else {
                // 后端用这个 body 存删除快照（标题/科目/类型）。
                // 若课表后续不再包含该作业，回收站就只能靠快照显示，
                // 所以这里必须把信息传过去，不能发空 body。
                await api.deleteAssignment(item.id, {
                    title: item.title,
                    subject: item.subject,
                    type: item.type === '作业' ? 'homework' : 'test'
                });
                assignments.value = assignments.value.filter(a => a.id !== item.id);
            }
            showToast('success', 'Deleted', `${item.title} 已移至回收站`);
        } catch (e: unknown) {
            showToast('error', 'Error', getErrorMessage(e, '删除失败'));
            throw e;
        }
    };

    const handleComplete = async (item: Assignment) => {
        try {
            if (item.type === '待办') {
                if(!item._todoId) return;
                await api.markTodoCompleted(item._todoId);
                const todo = todos.value.find(t => t._id === item._todoId);
                if (todo) todo.completed = true;
            } else {
                await api.markAssignmentCompleted(item.id);
                const assign = assignments.value.find(a => a.id === item.id);
                if (assign) assign.completed = true;
            }
            showToast('success', 'Completed', `${item.title} 已完成`);
        } catch (e: unknown) {
            showToast('error', 'Error', '操作失败');
        }
    };

    const handleUndoComplete = async (item: Assignment) => {
         try {
            if (item.type === '待办') {
                if(!item._todoId) return;
                await api.undoTodoCompleted(item._todoId);
                const todo = todos.value.find(t => t._id === item._todoId);
                if (todo) todo.completed = false;
            } else {
                await api.undoAssignmentCompleted(item.id);
                const assign = assignments.value.find(a => a.id === item.id);
                if (assign) assign.completed = false;
            }
            showToast('info', 'Undo', `${item.title} 已撤销完成`);
        } catch (e: unknown) {
            showToast('error', 'Error', '操作失败');
        }
    };

     const handleSetReminder = async (item: Assignment, config: ReminderConfig) => {
        try {
            if (item.type === '待办') {
                 if(!item._todoId) return;
                 await api.setTodoReminder(item._todoId, config);
            } else {
                 await api.setAssignmentReminder(item.id, config);
            }
            showToast('success', 'Reminder', '提醒设置成功');
        } catch (e: unknown) {
            showToast('error', 'Error', '提醒设置失败');
        }
    };

    const handleAddTodo = async (todoData: Partial<Todo>) => {
        try {
            await api.addTodo(todoData);
            await fetchAssignments(false);
            showToast('success', 'Added', '待办事项已添加');
            return true;
        } catch (e: unknown) {
            showToast('error', 'Error', '添加失败');
            return false;
        }
    };

    // Recycle Bin
    //
    // 后端把「已删除的作业」和「已删除的待办」分开存放，接口也是两组：
    //   GET  /assignments/deleted          POST /assignments/:id/restore
    //   GET  /todos/deleted                POST /todos/:id/restore
    // 所以这里合并两个列表，并用 source 字段记住每条该走哪组接口。
    const deletedItems = ref<RecycleBinItem[]>([]);
    const recycleBinLoading = ref(false);

    const TYPE_LABELS: Record<string, string> = {
        homework: '作业',
        test: '测试',
        todo: '待办'
    };

    const loadRecycleBin = async () => {
        recycleBinLoading.value = true;
        const items: RecycleBinItem[] = [];

        // 两个接口相互独立，其中一个失败不应让整个回收站空白
        const [assignRes, todoRes] = await Promise.allSettled([
            api.getDeletedAssignments(),
            api.getDeletedTodos()
        ]);

        if (assignRes.status === 'fulfilled') {
            const list = assignRes.value?.deleted_assignments || [];
            list.forEach((a) => {
                items.push({
                    id: a.task_id,
                    title: a.title || '未知任务',
                    subject: a.subject,
                    content: a.details,
                    dueDate: a.deadline,
                    deletedAt: a.delete_time,
                    type: TYPE_LABELS[a.type || ''] || '作业',
                    source: 'assignment'
                });
            });
        } else {
            console.error('Failed to load deleted assignments', assignRes.reason);
        }

        if (todoRes.status === 'fulfilled') {
            const list = todoRes.value?.deleted_todos || [];
            list.forEach((t) => {
                items.push({
                    id: t.todo_id || t._id,
                    title: t.title || '未知待办',
                    subject: '用户待办',
                    content: t.description,
                    dueDate: t.due_date,
                    deletedAt: t.delete_time,
                    type: '待办',
                    source: 'todo'
                });
            });
        } else {
            console.error('Failed to load deleted todos', todoRes.reason);
        }

        // 按删除时间倒序，最近删除的排在最前
        items.sort((a, b) => {
            const ta = a.deletedAt ? Date.parse(a.deletedAt) : 0;
            const tb = b.deletedAt ? Date.parse(b.deletedAt) : 0;
            return tb - ta;
        });

        deletedItems.value = items;
        recycleBinLoading.value = false;

        // 两个都失败说明是系统性问题，需要让用户知道
        if (assignRes.status === 'rejected' && todoRes.status === 'rejected') {
            showToast('error', 'Error', '回收站加载失败');
        }
    };

    const handleRestoreItem = async (item: RecycleBinItem) => {
        try {
            if (item.source === 'todo') {
                await api.restoreTodo(item.id);
            } else {
                await api.restoreAssignment(item.id);
            }
            await loadRecycleBin();
            await fetchAssignments(false);
            showToast('success', 'Restored', `"${item.title}" 已恢复`);
        } catch (e: unknown) {
            showToast('error', 'Error', getErrorMessage(e, '恢复失败'));
        }
    };

    const handlePermanentDelete = async (item: RecycleBinItem) => {
        try {
            if (item.source === 'todo') {
                await api.permanentDeleteTodo(item.id);
            } else {
                await api.permanentDeleteAssignment(item.id);
            }
            await loadRecycleBin();
            showToast('success', 'Deleted', '永久删除成功');
        } catch (e: unknown) {
            showToast('error', 'Error', getErrorMessage(e, '删除失败'));
        }
    };

    // 逐条恢复。后端没有「一键恢复」接口，
    // 原先调用的 /recycle-bin/restore-all 并不存在。
    const handleRestoreAll = async () => {
        const items = [...deletedItems.value];
        if (items.length === 0) return;

        const results = await Promise.allSettled(
            items.map((item) =>
                item.source === 'todo'
                    ? api.restoreTodo(item.id)
                    : api.restoreAssignment(item.id)
            )
        );

        const failed = results.filter((r) => r.status === 'rejected').length;
        await loadRecycleBin();
        await fetchAssignments(false);

        if (failed === 0) {
            showToast('success', 'Restored', `已恢复 ${items.length} 项`);
        } else if (failed === items.length) {
            showToast('error', 'Error', '全部恢复失败');
        } else {
            showToast('warning', 'Partial', `已恢复 ${items.length - failed} 项，${failed} 项失败`);
        }
    };

    return {
        allItems,
        assignmentStatus,
        user,
        currentTime,
        loading,
        error,
        assignments,
        todos,
        searchTerm,
        subjectFilter,
        statusFilter,
        filteredAssignments,
        subjects,
        urgentCount,
        soonCount,
        totalCount,
        completedCount,
        todoCount,
        fetchAssignments,
        handleLogout,
        handleDelete,
        handleComplete,
        handleUndoComplete,
        handleSetReminder,
        handleAddTodo,
        deletedItems,
        recycleBinLoading,
        loadRecycleBin,
        handleRestoreItem,
        handlePermanentDelete,
        handleRestoreAll
    };
}
