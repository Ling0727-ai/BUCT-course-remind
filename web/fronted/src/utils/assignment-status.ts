import type { Assignment, AssignmentStatus } from '@/pages/Home/home.data'
import { parseDueDate } from './datetime'

/** Remaining duration, including overdue tasks in urgent; completed always wins. */
export function getAssignmentStatus(item: Pick<Assignment, 'completed' | 'dueDate'>, now = Date.now()): AssignmentStatus {
  if (item.completed) return 'completed'
  const due = parseDueDate(item.dueDate)
  if (!due) return 'normal'
  const remaining = due.getTime() - now
  if (remaining <= 86400000) return 'urgent'
  if (remaining <= 3 * 86400000) return 'warning'
  return 'normal'
}
