import {http} from '@/api/http';
import {
    AssignmentRaw,
    AssignmentsResponse,
    CompletedAssignmentsResponse,
    DeletedAssignmentsResponse,
    DeletedTodosResponse,
    ReminderConfig,
    Todo,
    TodosResponse,
    User
} from './home.data';

// Auth
export const getUserInfo = () => http.get<User>('/auth/user-info');
export const logout = () => http.post('/auth/logout', {});

// Assignments
export const getAssignments = () => http.get<AssignmentsResponse | AssignmentRaw[]>('/assignments/standard?summary=true');
export const getAssignmentDetail = (id: string) => http.get<{ success: boolean; description: string }>(`/assignments/${encodeURIComponent(id)}/detail`);
export const getCompletedAssignments = () => http.get<CompletedAssignmentsResponse>('/assignments/completed');
export const markAssignmentCompleted = (id: string) => http.post(`/assignments/${id}/complete`, {});
export const undoAssignmentCompleted = (id: string) => http.post(`/assignments/${id}/uncomplete`, {});
export const deleteAssignment = (id: string, meta?: { title?: string; subject?: string; type?: string }) =>
    http.post(`/assignments/${id}/delete`, meta || {});
export const restoreAssignment = (id: string) => http.post(`/assignments/${id}/restore`, {});
export const permanentDeleteAssignment = (id: string) => http.delete(`/assignments/${id}/permanent-delete`);
export const clearDeletedAssignments = () => http.delete('/assignments/clear-deleted');
export const setAssignmentReminder = (id: string, config: ReminderConfig) => http.post(`/assignments/${id}/remind`, {reminderConfig: config});

// Todos
export const getTodos = () => http.get<Todo[] | TodosResponse>('/todos/');
export const addTodo = (todo: Partial<Todo>) => http.post('/todos/', todo);
export const deleteTodo = (id: string) => http.post(`/todos/${id}/delete`, {});
export const markTodoCompleted = (id: string) => http.post(`/todos/${id}/complete`, {});
export const undoTodoCompleted = (id: string) => http.post(`/todos/${id}/uncomplete`, {});
export const setTodoReminder = (id: string, config: ReminderConfig) => http.post(`/todos/${id}/remind`, {reminderConfig: config});
export const restoreTodo = (id: string) => http.post(`/todos/${id}/restore`, {});
export const permanentDeleteTodo = (id: string) => http.delete(`/todos/${id}/permanent-delete`);

// Recycle Bin
//
// 注意：后端从来没有 /recycle-bin/* 这组路由（Python 原版也没有），
// 原先这里调用它导致回收站永远 404、始终显示为空。
// 实际可用的接口是下面这两个按类型分开的列表接口。
export const getDeletedAssignments = () => http.get<DeletedAssignmentsResponse>('/assignments/deleted');
export const getDeletedTodos = () => http.get<DeletedTodosResponse>('/todos/deleted');

// Blacklist
export const addToBlacklist = (subjectId: string) => http.post('/blacklist', {subject_id: subjectId});

