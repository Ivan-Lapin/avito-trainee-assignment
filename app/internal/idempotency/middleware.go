package idempotency

import (
	"bytes"
	"context"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

// Middleware реализует идемпотентность по заголовку Idempotency-Key.
// Ключ должен быть уникален для операции на стороне клиента.
func Middleware(rdb *redis.Client, ttl time.Duration) func(http.Handler) http.Handler {
	if ttl <= 0 {
		ttl = time.Hour // типичное окно идемпотентности
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get("Idempotency-Key")
			if key == "" {
				// Нет ключа — обычная обработка.
				next.ServeHTTP(w, r)
				return
			}
			cacheKey := "idem:" + r.Method + ":" + r.URL.Path + ":" + key
			ctx := context.Background()

			// Если уже есть сохраненный ответ — отдаём его.
			cached, err := rdb.HGetAll(ctx, cacheKey).Result()
			if err == nil && len(cached) > 0 && cached["status"] != "" && cached["body"] != "" {
				w.Header().Set("X-Idempotency", "hit")
				w.WriteHeader(atoi(cached["status"]))
				_, _ = w.Write([]byte(cached["body"]))
				return
			}

			// Перехватываем ответ для сохранения.
			rec := newRecorder(w)
			next.ServeHTTP(rec, r)

			// Сохраняем только успешные ответы 2xx/3xx (настраивается при желании).
			if rec.status >= 200 && rec.status < 400 {
				_ = rdb.HSet(ctx, cacheKey, "status", itoa(rec.status), "body", rec.buf.String()).Err()
				_ = rdb.Expire(ctx, cacheKey, ttl).Err()
			}
		})
	}
}

type recorder struct {
	http.ResponseWriter
	status int
	buf    bytes.Buffer
}

func newRecorder(w http.ResponseWriter) *recorder {
	return &recorder{ResponseWriter: w, status: http.StatusOK}
}

func (r *recorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *recorder) Write(p []byte) (int, error) {
	_, _ = r.buf.Write(p)
	return r.ResponseWriter.Write(p)
}

func atoi(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		n = n*10 + int(s[i]-'0')
	}
	return n
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var out [4]byte
	i := len(out)
	for n > 0 && i > 0 {
		i--
		out[i] = byte('0' + n%10)
		n /= 10
	}
	return string(out[i:])
}
