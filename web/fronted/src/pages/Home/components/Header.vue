<template>
  <header class="header">
    <div class="header-left">
      <span class="brand-note"><span class="brand-dot"></span> BUCT · 学习日常</span>
      <h1>让学习，有条不紊。</h1>
      <p>把任务安排好，给自己多一点从容。</p>
    </div>

    <div class="header-right">
      <button type="button" class="icon-btn" aria-label="刷新数据" title="刷新数据" @click="$emit('refresh')">
        <i class="fas fa-sync-alt" aria-hidden="true"></i>
      </button>

      <!-- 桌面端：全部动作直接平铺 -->
      <template v-if="!isNarrow">
        <button type="button" class="icon-btn" aria-label="打开回收站" title="回收站" @click="$emit('show-recycle-bin')">
          <i class="fas fa-trash-alt" aria-hidden="true"></i>
        </button>
        <button type="button" class="icon-btn blacklist" aria-label="管理黑名单科目" title="管理黑名单科目" @click="$emit('show-blacklist')">
          <i class="fas fa-ban" aria-hidden="true"></i>
        </button>
      </template>

      <!-- 窄屏：收进"更多"菜单，避免按钮换行挤压 -->
      <div v-else class="more-menu">
        <button
          type="button"
          class="icon-btn"
          aria-haspopup="true"
          :aria-expanded="menuOpen"
          aria-label="更多操作"
          title="更多操作"
          @click="menuOpen = !menuOpen"
        >
          <i class="fas fa-ellipsis-vertical" aria-hidden="true"></i>
        </button>
        <div v-if="menuOpen" class="more-panel" role="menu">
          <button type="button" role="menuitem" class="more-item" @click="runAction('show-recycle-bin')">
            <i class="fas fa-trash-alt" aria-hidden="true"></i> 回收站
          </button>
          <button type="button" role="menuitem" class="more-item" @click="runAction('show-blacklist')">
            <i class="fas fa-ban" aria-hidden="true"></i> 黑名单管理
          </button>
        </div>
      </div>

      <span class="time">{{ currentTime }}</span>

      <div v-if="user" class="user-info">
        <span class="username">{{ user.username }}</span>
        <button type="button" class="logout-btn" @click="$emit('logout')">退出</button>
      </div>

      <router-link class="settings-link" to="/settings" aria-label="打开设置" title="设置">
        <i class="fas fa-cog" aria-hidden="true"></i>
      </router-link>
    </div>
  </header>
</template>

<script lang="ts">
import { defineComponent, onBeforeUnmount, onMounted, ref } from 'vue'
import type { PropType } from 'vue'
import type { User } from '../home.data'

export default defineComponent({
  name: 'Header',
  props: {
    currentTime: { type: String, required: true },
    user: { type: Object as PropType<User | null>, default: null }
  },
  emits: ['show-recycle-bin', 'logout', 'refresh', 'show-blacklist'],
  setup(_props, { emit }) {
    const isNarrow = ref(false)
    const menuOpen = ref(false)
    let media: MediaQueryList | null = null

    const syncNarrow = (event: MediaQueryList | MediaQueryListEvent) => {
      isNarrow.value = event.matches
      if (!event.matches) {
        menuOpen.value = false
      }
    }

    const closeMenu = (event: MouseEvent) => {
      const target = event.target as HTMLElement | null
      if (menuOpen.value && !target?.closest('.more-menu')) {
        menuOpen.value = false
      }
    }

    const runAction = (event: 'show-recycle-bin' | 'show-blacklist') => {
      menuOpen.value = false
      emit(event)
    }

    onMounted(() => {
      media = window.matchMedia('(max-width: 768px)')
      syncNarrow(media)
      media.addEventListener('change', syncNarrow)
      document.addEventListener('click', closeMenu)
    })

    onBeforeUnmount(() => {
      media?.removeEventListener('change', syncNarrow)
      document.removeEventListener('click', closeMenu)
    })

    return { isNarrow, menuOpen, runAction }
  }
})
</script>

<style scoped>
.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-4);
  margin: 0 0 var(--space-4);
  padding: 38px var(--space-6) 28px;
  background: transparent;
}

.brand-note { display: inline-flex; align-items: center; gap: 7px; font-size: 11px; letter-spacing: 0.1em; color: var(--text-tertiary); margin-bottom: 14px; }
.brand-dot { width: 7px; height: 7px; border-radius: 50%; background: #1b9b8b; box-shadow: 0 0 0 4px rgba(27,155,139,.1); }

.header-left h1 {
  margin-bottom: 2px;
  font-size: clamp(24px, 3vw, 32px);
  letter-spacing: -0.04em;
  font-weight: var(--font-bold);
  color: var(--text-primary);
}

.header-left p {
  font-size: var(--font-sm);
  color: var(--text-tertiary);
}

.header-right {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.icon-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  font-size: 14px;
  color: var(--text-tertiary);
  background: var(--gray-50);
  border: none;
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all var(--transition);
}

.icon-btn:hover {
  background: var(--gray-100);
  color: var(--text-primary);
}

.icon-btn.blacklist {
  color: var(--red);
  background: var(--red-subtle);
  opacity: 0.7;
}

.icon-btn.blacklist:hover {
  opacity: 1;
  color: var(--red);
  background: var(--red-subtle);
}

/* ── 更多菜单 ── */
.more-menu { position: relative; }

.more-panel {
  position: absolute;
  top: calc(100% + 6px);
  right: 0;
  z-index: 200;
  min-width: 160px;
  padding: var(--space-1);
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-lg);
}

.more-item {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  width: 100%;
  min-height: 44px;
  padding: 0 var(--space-3);
  font-size: var(--font-sm);
  color: var(--text-secondary);
  text-align: left;
  background: none;
  border: none;
  border-radius: var(--radius-sm);
  cursor: pointer;
}

.more-item:hover {
  background: var(--gray-100);
  color: var(--text-primary);
}

.time {
  padding: 0 var(--space-2);
  font-size: var(--font-sm);
  color: var(--text-tertiary);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.user-info {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding-left: var(--space-3);
  border-left: 1px solid var(--border);
}

.username {
  font-size: var(--font-sm);
  font-weight: var(--font-semibold);
  color: var(--text-primary);
}

.logout-btn {
  min-height: 36px;
  padding: 4px 12px;
  font-size: var(--font-sm);
  color: var(--text-tertiary);
  background: none;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all var(--transition);
}

.logout-btn:hover {
  color: var(--red);
  border-color: var(--red);
  background: var(--red-subtle);
}

.settings-link {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  font-size: 16px;
  color: var(--text-tertiary);
  background: var(--gray-50);
  border-radius: var(--radius-md);
  transition: all var(--transition);
}

.settings-link:hover {
  background: var(--gray-100);
  color: var(--text-primary);
}

@media (max-width: 768px) {
  .header {
    flex-direction: column;
    gap: var(--space-3);
    padding: var(--space-4);
    text-align: center;
  }

  .header-left h1 { font-size: var(--font-lg); }

  .header-right {
    flex-wrap: wrap;
    justify-content: center;
    width: 100%;
  }
}
</style>
