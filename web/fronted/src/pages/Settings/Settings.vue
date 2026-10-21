<template>
  <div class="settings-page">
    <div class="settings-header">
      <button class="back-btn" @click="goBack">&larr; 返回</button>
      <h1>系统设置</h1>
    </div>

    <div class="settings-body">
      <!-- Student Info -->
      <section class="card">
        <h2 class="card-title">学生信息配置</h2>
        <div class="card-row">
          <div class="field">
            <label class="label">学号</label>
            <input type="text" v-model="studentInfo.studentId" placeholder="请输入学号" class="input">
          </div>
          <div class="field">
            <label class="label">外部系统密码</label>
            <input type="password" v-model="studentInfo.sPassword" placeholder="用于访问教务系统的密码" class="input">
          </div>
        </div>
        <p class="tip">此信息用于自动登录教务系统获取作业和考试信息，请确保信息准确</p>
        <button class="btn primary" @click="handleSaveStudentInfo">保存学生信息</button>
      </section>

      <!-- Email -->
      <section class="card">
        <h2 class="card-title">邮箱设置</h2>
        <div class="field">
          <label class="label">账号邮箱</label>
          <input type="email" v-model="emailSettings.email" placeholder="请输入新的邮箱地址" class="input">
        </div>
        <p class="tip">修改邮箱将同时更新账号恢复邮箱和作业提醒接收邮箱。系统默认使用注册邮箱发送提醒。</p>
        <button class="btn primary" @click="handleSaveEmailSettings">修改邮箱</button>
      </section>

      <!-- Data -->
      <section class="card">
        <h2 class="card-title">数据管理</h2>
        <div class="data-card">
          <div class="data-grid">
            <div class="data-item">
              <span class="data-label">数据状态</span>
              <span :class="['data-value', dataStatus.hasData ? 'active' : 'inactive']">
                {{ dataStatus.hasData ? '已同步' : '未同步' }}
              </span>
            </div>
            <div class="data-item" v-if="dataStatus.lastUpdate">
              <span class="data-label">最后更新</span>
              <span class="data-value">{{ formatDateTime(dataStatus.lastUpdate) }}</span>
            </div>
            <div class="data-item" v-if="dataStatus.hoursUntilRefresh !== null">
              <span class="data-label">下次自动刷新</span>
              <span class="data-value">{{ formatNextRefresh(dataStatus.hoursUntilRefresh) }}</span>
            </div>
          </div>
          <div class="data-actions">
            <button class="btn" @click="handleRefreshCourseData" :disabled="refreshing">
              <i v-if="refreshing" class="fas fa-spinner fa-spin"></i>
              {{ refreshing ? '刷新中...' : '手动刷新数据' }}
            </button>
            <span class="tip">数据每12小时自动刷新一次</span>
          </div>
        </div>
      </section>

      <!-- Background -->
      <section class="card">
        <h2 class="card-title">背景设置</h2>
        <BackgroundSettings />
      </section>

      <!-- Font -->
      <section class="card">
        <h2 class="card-title">字体与字号</h2>
        <FontSettings />
      </section>

    </div>
  </div>
</template>

<script lang="ts">
import { defineComponent } from 'vue';
import { useSettingsService } from './settings';
import BackgroundSettings from './components/BackgroundSettings.vue';
import FontSettings from './components/FontSettings.vue';

export default defineComponent({
  name: 'Settings',
  components: { BackgroundSettings, FontSettings },
  setup() { return useSettingsService(); }
});
</script>

<style scoped>
.settings-page {
  max-width: 720px;
  margin: 0 auto;
  padding: var(--space-6);
  min-height: 100vh;
}

.settings-header {
  display: flex;
  align-items: center;
  margin-bottom: var(--space-6);
  position: relative;
}

.back-btn {
  background: none;
  border: none;
  font-size: var(--font-sm);
  color: var(--text-tertiary);
  cursor: pointer;
  padding: 8px 12px;
  border-radius: var(--radius-md);
  transition: all var(--transition);
  position: absolute;
  left: -12px;
}

.back-btn:hover {
  background: var(--gray-100);
  color: var(--text-primary);
}

.settings-header h1 {
  flex: 1;
  text-align: center;
  font-size: var(--font-xl);
  font-weight: var(--font-bold);
  color: var(--text-primary);
}

.settings-body {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

/* Card */
.card {
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: var(--radius-xl);
  padding: var(--space-6);
}

.card-title {
  font-size: var(--font-lg);
  font-weight: var(--font-semibold);
  color: var(--text-primary);
  margin-bottom: var(--space-5);
  display: flex;
  align-items: center;
  gap: 8px;
}

/* Fields */
.card-row {
  display: flex;
  gap: var(--space-4);
  margin-bottom: var(--space-3);
}

.field {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: var(--space-3);
}

.label {
  font-size: var(--font-sm);
  font-weight: var(--font-medium);
  color: var(--text-secondary);
}

.input {
  width: 100%;
  padding: 10px 14px;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  font-size: var(--font-sm);
  color: var(--text-primary);
  background: var(--bg-secondary);
  transition: all var(--transition);
}

.input:focus {
  outline: none;
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--primary-subtle);
  background: var(--bg);
}

.tip {
  font-size: var(--font-xs);
  color: var(--text-tertiary);
  line-height: 1.5;
  padding: var(--space-3);
  background: var(--gray-50);
  border-radius: var(--radius-md);
  border-left: 3px solid var(--primary-light);
  margin-bottom: var(--space-4);
}

/* Buttons */
.btn {
  padding: 10px 20px;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg);
  color: var(--text-secondary);
  font-weight: var(--font-medium);
  font-size: var(--font-sm);
  cursor: pointer;
  transition: all var(--transition);
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.btn:hover:not(:disabled) {
  border-color: var(--gray-300);
  background: var(--gray-50);
}

.btn.primary {
  background: var(--primary);
  border-color: var(--primary);
  color: white;
}

.btn.primary:hover:not(:disabled) {
  background: var(--primary-dark);
  border-color: var(--primary-dark);
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.3);
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

/* Data Card */
.data-card {
  background: var(--gray-50);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: var(--space-5);
}

.data-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: var(--space-4);
  margin-bottom: var(--space-4);
}

.data-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.data-label {
  font-size: var(--font-xs);
  color: var(--text-tertiary);
  font-weight: var(--font-medium);
}

.data-value {
  font-size: var(--font-sm);
  color: var(--text-primary);
  font-weight: var(--font-semibold);
}

.data-value.active { color: var(--green); }
.data-value.inactive { color: var(--red); }

.data-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: var(--space-4);
  border-top: 1px solid var(--border);
}

.data-actions .tip { margin-bottom: 0; border: none; padding: 0; background: none; }

@media (max-width: 640px) {
  .card-row { flex-direction: column; gap: 0; }
  .data-actions { flex-direction: column; gap: var(--space-3); align-items: stretch; }
  .data-actions .btn { width: 100%; justify-content: center; }
  .settings-page { padding: var(--space-4); }
  .card { padding: var(--space-4); }
}
</style>
