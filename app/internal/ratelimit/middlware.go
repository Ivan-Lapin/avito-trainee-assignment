package ratelimit

import (
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// simpleLimiter использует готовый алгоритм "фиксированное окно" для простоты.
// Для продакшн лучше применить Lua-скрипт token bucket с временем истечения.
type simpleLimiter struct {
	rdb       *redis.Client
	rps       int
	burst     int
	windowSec int
}

func Middleware(rdb *redis.Client, rps, burst int) func(http.Handler) http.Handler {
	l := &simpleLimiter{rdb: rdb, rps: rps, burst: burst, windowSec: 1}
	if l.rps <= 0 {
		l.rps = 5
	}
	if l.burst < l.rps {
		l.burst = l.rps * 2
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			route := r.URL.Path
			principal := clientIP(r)
			key := "rl:" + route + ":" + principal + ":" + time.Now().Format("2006-01-02T15:04:05")

			// Увеличиваем счётчик в текущем секундном окне и ставим срок жизни.
			pipe := l.rdb.TxPipeline()
			incr := pipe.Incr(r.Context(), key)
			pipe.Expire(r.Context(), key, 2*time.Second)
			_, _ = pipe.Exec(r.Context())

			count := int(incr.Val())
			limit := l.burst
			if count > limit {
				w.Header().Set("Retry-After", "1")
				http.Error(w, "rate limited", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
