package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// 登录/注册等敏感接口的固定窗口限流。
// 之前只有验证码发送有频率限制，Login 完全没有任何限制，
// 攻击者可以对单个账号做无限次密码爆破。

type rateEntry struct {
	count       int
	windowStart time.Time
}

type rateLimiter struct {
	shards [32]rateShard
	limit  int
	window time.Duration
}

type rateShard struct {
	mu      sync.Mutex
	entries map[string]*rateEntry
}

func (r *rateLimiter) shard(key string) *rateShard {
	var hash uint64 = 14695981039346656037
	for i := 0; i < len(key); i++ {
		hash = (hash ^ uint64(key[i])) * 1099511628211
	}
	return &r.shards[hash%uint64(len(r.shards))]
}

// allow 返回是否放行，以及建议的 Retry-After 秒数
func (r *rateLimiter) allow(key string) (bool, int) {
	shard := r.shard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	if shard.entries == nil {
		shard.entries = make(map[string]*rateEntry)
	}

	now := time.Now()
	e, ok := shard.entries[key]
	if !ok || now.Sub(e.windowStart) >= r.window {
		if !ok && len(shard.entries) >= 2048 {
			return false, max(1, int(r.window.Seconds()))
		}
		shard.entries[key] = &rateEntry{count: 1, windowStart: now}
		return true, 0
	}

	if e.count >= r.limit {
		retry := int((r.window - now.Sub(e.windowStart)).Seconds())
		if retry < 1 {
			retry = 1
		}
		return false, retry
	}

	e.count++
	return true, 0
}

// gc 清理过期条目，避免 map 无限增长
func (r *rateLimiter) gc() {
	for i := range r.shards {
		r.gcShard(i)
	}
}

func (r *rateLimiter) gcShard(index int) {
	shard := &r.shards[index]
	shard.mu.Lock()
	defer shard.mu.Unlock()
	now := time.Now()
	for k, e := range shard.entries {
		if now.Sub(e.windowStart) >= r.window {
			delete(shard.entries, k)
		}
	}
}

// RateLimit 返回一个按客户端 IP 计数的固定窗口限流中间件。
// limit 为窗口内允许的请求数，window 为窗口长度。
func RateLimit(limit int, window time.Duration) gin.HandlerFunc {
	rl := &rateLimiter{
		limit:  limit,
		window: window,
	}

	// 后台定期清理，防止被大量不同 IP 撑爆内存
	go func() {
		ticker := time.NewTicker(max(time.Second, window/time.Duration(len(rl.shards))))
		defer ticker.Stop()
		index := 0
		for range ticker.C {
			rl.gcShard(index)
			index = (index + 1) % len(rl.shards)
		}
	}()

	return func(c *gin.Context) {
		key := c.ClientIP()
		if key == "" {
			key = "unknown"
		}

		ok, retry := rl.allow(key)
		if !ok {
			c.Header("Retry-After", itoa(retry))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "请求过于频繁，请稍后再试",
				"retry_after": retry,
			})
			return
		}

		c.Next()
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
