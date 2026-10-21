import { nextTick, onBeforeUnmount, onMounted, ref, watch, type Ref } from 'vue'

const FOCUSABLE_SELECTOR = [
  'a[href]',
  'button:not([disabled])',
  'input:not([disabled]):not([type="hidden"])',
  'select:not([disabled])',
  'textarea:not([disabled])',
  '[tabindex]:not([tabindex="-1"])'
].join(',')

/** 打开中的弹窗栈，用于分配 z-index 并只在最外层关闭时解锁滚动。 */
const openStack: symbol[] = []

let scrollLockCount = 0
let savedBodyOverflow = ''
let appInertApplied = false

function lockPageScroll(): void {
  if (scrollLockCount === 0) {
    savedBodyOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    // 只把应用根节点设为 inert：弹窗与 Toast 都在 body 下，不受影响。
    const app = document.getElementById('app')
    if (app && !app.hasAttribute('inert')) {
      app.setAttribute('inert', '')
      appInertApplied = true
    }
  }
  scrollLockCount += 1
}

function unlockPageScroll(): void {
  scrollLockCount = Math.max(0, scrollLockCount - 1)
  if (scrollLockCount > 0) {
    return
  }
  document.body.style.overflow = savedBodyOverflow
  if (appInertApplied) {
    document.getElementById('app')?.removeAttribute('inert')
    appInertApplied = false
  }
}

let uid = 0

export interface UseModalOptions {
  /** 弹窗是否打开。 */
  open: Ref<boolean>
  /** 弹窗外层元素，用于判断点击是否落在遮罩上。 */
  overlay: Ref<HTMLElement | null>
  /** 弹窗内容元素，作为焦点陷阱的边界。 */
  content: Ref<HTMLElement | null>
  /** 请求关闭（Esc / 关闭按钮 / 点击遮罩）。 */
  onClose: () => void
  /** 为 false 时忽略 Esc 与点击遮罩。 */
  closable: Ref<boolean>
}

export interface UseModalReturn {
  titleId: string
  zIndex: Ref<number>
}

/**
 * 弹窗的键盘与焦点行为：Esc 关闭、Tab 焦点陷阱、打开时聚焦、关闭后归还焦点、锁定页面滚动。
 */
export function useModal(options: UseModalOptions): UseModalReturn {
  const token = Symbol('modal')
  const titleId = `modal-title-${(uid += 1)}`
  const zIndex = ref(1000)

  let previouslyFocused: HTMLElement | null = null
  let active = false

  function focusableElements(): HTMLElement[] {
    const root = options.content.value
    if (!root) {
      return []
    }
    return Array.from(root.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR)).filter((el) => {
      if (el.hasAttribute('disabled') || el.getAttribute('aria-hidden') === 'true') {
        return false
      }
      return el.offsetWidth > 0 || el.offsetHeight > 0 || el === document.activeElement
    })
  }

  function focusInitial(): void {
    const root = options.content.value
    if (!root) {
      return
    }
    const preferred = root.querySelector<HTMLElement>('[data-autofocus]')
    const target = preferred ?? focusableElements()[0] ?? root
    target.focus({ preventScroll: true })
  }

  function handleKeydown(event: KeyboardEvent): void {
    if (openStack[openStack.length - 1] !== token) return
    if (event.key === 'Escape' && options.closable.value) {
      event.preventDefault()
      event.stopPropagation()
      options.onClose()
      return
    }

    if (event.key !== 'Tab') {
      return
    }

    const root = options.content.value
    if (!root) {
      return
    }

    const items = focusableElements()
    if (items.length === 0) {
      event.preventDefault()
      root.focus({ preventScroll: true })
      return
    }

    const first = items[0]
    const last = items[items.length - 1]
    const current = document.activeElement as HTMLElement | null
    const inside = current !== null && root.contains(current)

    if (event.shiftKey) {
      if (!inside || current === first) {
        event.preventDefault()
        last.focus({ preventScroll: true })
      }
      return
    }

    if (!inside || current === last) {
      event.preventDefault()
      first.focus({ preventScroll: true })
    }
  }

  function activate(): void {
    if (active) {
      return
    }
    active = true
    previouslyFocused = document.activeElement as HTMLElement | null

    openStack.push(token)
    zIndex.value = 1000 + openStack.indexOf(token) * 10

    lockPageScroll()
    document.addEventListener('keydown', handleKeydown, true)
    void nextTick(focusInitial)
  }

  function deactivate(): void {
    if (!active) {
      return
    }
    active = false

    document.removeEventListener('keydown', handleKeydown, true)

    const index = openStack.indexOf(token)
    if (index !== -1) {
      openStack.splice(index, 1)
    }

    unlockPageScroll()

    const restore = previouslyFocused
    previouslyFocused = null
    if (restore && document.contains(restore)) {
      restore.focus({ preventScroll: true })
    }
  }

  watch(
    options.open,
    (isOpen) => {
      if (isOpen) {
        activate()
      } else {
        deactivate()
      }
    },
    { flush: 'post' }
  )

  onMounted(() => {
    if (options.open.value) {
      activate()
    }
  })

  onBeforeUnmount(deactivate)

  return { titleId, zIndex }
}
