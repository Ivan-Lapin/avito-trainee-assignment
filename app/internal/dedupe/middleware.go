package dedupe

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

// responseRecorder перехватывает ответ, чтобы сохранить тело/статус.
type responseRecorder struct {
	http.ResponseWriter
	status int
	buf    bytes.Buffer
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.status = code
	rr.ResponseWriter.WriteHeader(code)
}

func (rr *responseRecorder) Write(p []byte) (int, error) {
	_, _ = rr.buf.Write(p)
	return rr.ResponseWriter.Write(p)
}

// keyFromRequest строит ключ дедупликации из метода, пути и тела.
// В production стоит добавить user-id/tenant-id/критичные заголовки.
func keyFromRequest(r *http.Request, body []byte) string {
	h := sha256.New()
	h.Write([]byte(r.Method))
	h.Write([]byte{0})
	h.Write([]byte(r.URL.Path))
	h.Write([]byte{0})
	h.Write(body)
	return "dedupe:" + hex.EncodeToString(h.Sum(nil))
}

// Middleware делает дедупликацию write-запросов в коротком окне TTL.
func Middleware(rdb *redis.Client, ttl time.Duration) func(http.Handler) http.Handler {
	if ttl <= 0 {
		ttl = 5 * time.Second // дефолтное короткое окно
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Только для write-методов.
			if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodPatch && r.Method != http.MethodDelete {
				next.ServeHTTP(w, r)
				return
			}
			// Считываем тело для хеша; восстанавливаем body для хендлера.
			var body []byte
			if r.Body != nil {
				body, _ = io.ReadAll(r.Body)
				r.Body.Close()
			}
			r.Body = io.NopCloser(bytes.NewReader(body))

			key := keyFromRequest(r, body)
			ctx := context.Background()

			// Попытка прочитать сохранённый ответ.
			cached, err := rdb.HGetAll(ctx, key).Result()
			if err == nil && len(cached) > 0 && cached["status"] != "" && cached["body"] != "" {
				// Нашли ответ — отдаем его и завершаем.
				w.Header().Set("X-Dedupe", "hit")
				w.WriteHeader(atoiSafe(cached["status"]))
				_, _ = w.Write([]byte(cached["body"]))
				return
			}

			// Проставим «замок» с коротким TTL, чтобы конкурентные запросы ждали/получили кеш.
			ok, _ := rdb.SetNX(ctx, key+":lock", "1", ttl).Result()
			if !ok {
				// Уже есть обработка; можно подождать небольшой период и повторить чтение кеша.
				time.Sleep(50 * time.Millisecond)
				cached, _ = rdb.HGetAll(ctx, key).Result()
				if len(cached) > 0 {
					w.Header().Set("X-Dedupe", "wait-hit")
					w.WriteHeader(atoiSafe(cached["status"]))
					_, _ = w.Write([]byte(cached["body"]))
					return
				}
			}

			// Перехват ответа хендлера для кеширования.
			rr := &responseRecorder{ResponseWriter: w, status: 200}
			next.ServeHTTP(rr, r)

			// Сохраняем ответ на короткий TTL и снимаем замок.
			_ = rdb.HSet(ctx, key, "status", itoaSafe(rr.status), "body", rr.buf.String()).Err()
			_ = rdb.Expire(ctx, key, ttl).Err()
			_ = rdb.Del(ctx, key+":lock").Err()
		})
	}
}

func atoiSafe(s string) int {
	var n int
	for i := 0; i < len(s); i++ {
		n = n*10 + int(s[i]-'0')
	}
	return n
}

func itoaSafe(n int) string {
	if n == 0 {
		return "0"
	}
	var b [4]byte
	i := len(b)
	for n > 0 && i > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
