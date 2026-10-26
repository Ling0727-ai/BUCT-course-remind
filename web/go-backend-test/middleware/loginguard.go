package middleware

import (
	"sync"
	"time"
)

// 按账号的登录失败锁定。
//
// 为什么不能只按 IP 限流：校园网里大量用户共享同一个出口 IP，
// 按 IP 计数会把正常用户一起挡在门外；而攻击者换 IP 成本极低。
// 因此这里按「账号」维度记录连续失败次数，成功登录即清零。

type failureEntry struct {
	count       int
	firstFailAt time.Time
	lockedUntil time.Time
}

type loginGuard struct {
	mu      sync.Mutex
	entries map[string]*failureEntry

	maxFailures int
	window      time.Duration
	lockFor     time.Duration
	lastGC      time.Time
}

var guard = &loginGuard{
	entries:     make(map[string]*failureEntry),
	maxFailures: 5,
	window:      15 * time.Minute,
	lockFor:     15 * time.Minute,
}

// LoginLocked 返回账号是否处于锁定期，以及剩余锁定秒数
func LoginLocked(account string) (bool, int) {
	if account == "" {
		return false, 0
	}

	guard.mu.Lock()
	defer guard.mu.Unlock()

	guard.gcLocked()

	e, ok := guard.entries[account]
	if !ok {
		if len(guard.entries) >= 10000 {
			return true, 60
		}
		return false, 0
	}

	now := time.Now()
	if e.lockedUntil.After(now) {
		return true, int(e.lockedUntil.Sub(now).Seconds()) + 1
	}

	// 锁定已过期且窗口也过了，清理
	if now.Sub(e.firstFailAt) > guard.window {
		delete(guard.entries, account)
	}
	return false, 0
}

// RecordLoginFailure 记录一次登录失败，返回是否已触发锁定
func RecordLoginFailure(account string) bool {
	if account == "" {
		return false
	}

	guard.mu.Lock()
	defer guard.mu.Unlock()

	now := time.Now()
	e, ok := guard.entries[account]
	if !ok && len(guard.entries) >= 10000 {
		return true
	}
	if !ok || now.Sub(e.firstFailAt) > guard.window {
		guard.entries[account] = &failureEntry{count: 1, firstFailAt: now}
		return false
	}

	e.count++
	if e.count >= guard.maxFailures {
		e.lockedUntil = now.Add(guard.lockFor)
		// 重置窗口，避免解锁后立刻又因旧计数被锁
		e.count = 0
		e.firstFailAt = now
		return true
	}
	return false
}

// ResetLoginFailures 登录成功后清除该账号的失败记录
func ResetLoginFailures(account string) {
	if account == "" {
		return
	}

	guard.mu.Lock()
	defer guard.mu.Unlock()
	delete(guard.entries, account)
}

// gcLocked 清理长期无活动的条目，调用方须持有锁
func (g *loginGuard) gcLocked() {
	now := time.Now()
	if now.Sub(g.lastGC) < time.Minute {
		return
	}
	cutoff := now.Add(-g.window - g.lockFor)
	for k, e := range g.entries {
		if e.firstFailAt.Before(cutoff) && e.lockedUntil.Before(now) {
			delete(g.entries, k)
		}
	}
	g.lastGC = now
}
