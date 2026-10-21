import { ref, onMounted, onUnmounted } from 'vue'

const now = ref(Date.now())
let consumers = 0
let timer: ReturnType<typeof setInterval> | undefined

/** One clock keeps counts, filters, cards and columns in sync as deadlines pass. */
export function useTaskClock() {
  onMounted(() => {
    if (consumers++ === 0) {
      now.value = Date.now()
      timer = setInterval(() => { now.value = Date.now() }, 1000)
    }
  })
  onUnmounted(() => {
    if (--consumers === 0) {
      clearInterval(timer)
      timer = undefined
    }
  })
  return now
}
