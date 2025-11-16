package main

import (
	"avito/train-assignment/app/internal/cache"
	"avito/train-assignment/app/internal/config"
	"avito/train-assignment/app/internal/logger"
	"avito/train-assignment/app/internal/metrics"
	"avito/train-assignment/app/internal/repository"
	"avito/train-assignment/app/internal/transport/httpapi"
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

func main() {
	// Логи в production‑формате для удобства анализа и трейсинга.
	log := logger.New()
	defer log.Sync()

	// Конфигурация из окружения c валидацией.
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("config", zap.Error(err))
	}

	// Инициализация Postgres и Redis, применение миграций/сидов.
	db, err := repository.OpenPostgres(cfg.DBDSN)
	if err != nil {
		log.Fatal("db open", zap.Error(err))
	}
	defer db.Close()

	if err := repository.Migrate(db); err != nil {
		log.Fatal("db migrate", zap.Error(err))
	}
	if err := repository.Seed(db); err != nil {
		log.Fatal("db seed", zap.Error(err))
	}

	rdb, err := cache.NewRedis(cfg.RedisAddr)
	if err != nil {
		log.Fatal("redis", zap.Error(err))
	}
	defer rdb.Close()

	// Сборка HTTP роутера и middleware цепочки.
	r := chi.NewRouter()
	metrics.Register(r)

	// Роуты API и системные эндпоинты.
	httpapi.Mount(r, db, rdb, cfg)

	r.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Graceful shutdown.
	go func() {
		log.Info("http listen", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("listen", zap.Error(err))
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	log.Info("shutdown complete")
}
