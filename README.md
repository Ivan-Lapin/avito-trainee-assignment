# PR Reviewer Assignment Service (Avito Backend Trainee, Autumn 2025)

Сервис автоматически назначает ревьюверов на Pull Request’ы, поддерживает переназначение по правилам из ТЗ, управляет активностью пользователей и обеспечивает идемпотентный merge. Реализованы rate limiting, дедупликация повторов, кэш запросов по Idempotency-Key, метрики и docker-compose окружение.

## Стек

- Go 1.22, chi router, sqlx/pgx, Redis go-redis v9, zap, Prometheus client.
- PostgreSQL 16 (docker-compose), Redis 7 (docker-compose).
- OpenAPI (openapi/openapi.yml) как контракт, oapi-codegen (опционально).

## Архитектура

- Слои:
  - transport/http (handlers): реализация OpenAPI эндпоинтов, маппинг ошибок.
  - service: бизнес‑правила автоназначения/переназначения/merge/статистика/массовая деактивация.
  - repository: Postgres (users, teams, team_members, pull_requests, pr_reviewers), миграции/сиды через go:embed.
  - cache & infra: Redis для rate limit, идемпотентности, дедупликации и кэша статистики.
- Ограничения/инварианты:
  - До двух ревьюверов на PR; после MERGED список ревьюверов менять нельзя.
  - Идемпотентность merge: повтор возвращает текущее состояние PR; поддерживается заголовок Idempotency-Key.
  - Автоназначение: активные участники команды автора, исключая автора; при нехватке — 0/1 ревьювер.

## Запуск

Локально:
- Убедитесь, что Postgres и Redis запущены и доступны:  
  - DB_DSN=postgres://postgres:@localhost:8080/avito_task?sslmode=disable  
  - REDIS_ADDR=localhost:6379  
  - PORT=8095.
- Запуск:
  - go build -o bin/pr-reviewer ./cmd/server  
  - ./bin/pr-reviewer.
- Health: curl http://localhost:8095/api/health -> ok.

Docker Compose:
- В корне: docker-compose up --build.
- Сервис: http://localhost:8095, Postgres для psql: localhost:8080, Redis: localhost:6379.
- При старте автоматически применяются миграции и сиды (go:embed).

## Основные эндпоинты

- POST /api/pullRequest/create — создать PR, автоназначение до двух ревьюверов из команды автора (активные, кроме автора).
- POST /api/pullRequest/reassign — заменить одного ревьювера на активного из команды заменяемого ревьювера.
- POST /api/pullRequest/merge — идемпотентный merge; поддерживает Idempotency-Key.
- POST /api/user/activity — смена активности пользователя (true/false).
- POST /api/team/add — создание команды.
- GET /api/stats/assignments — простая статистика: количество назначений по ревьюверам/PR (доп. задание).
- POST /api/team/deactivateAndReassign — массовая деактивация команды и безопасная переназначаемость в открытых PR (доп. задание).
- GET /metrics — Prometheus метрики процесса и бизнес‑метрики.

OpenAPI: файл openapi/openapi.yml — источник правды по контракту; генерация типов: make generate (oapi-codegen).

## Примеры запросов

Создать PR:
```
POST /api/pullRequest/create
Content-Type: application/json

{
  "id": "00000000-0000-0000-0000-00000000ab01",
  "title": "feat: auto assign",
  "author_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
  "team_name": "core"
}
```
Ожидание: 201 Created, status=OPEN, reviewers 0..2.

Идемпотентный merge:
```
POST /api/pullRequest/merge
Content-Type: application/json
Idempotency-Key: abc-1

{"id": "00000000-0000-0000-0000-00000000ab01"}
```
Ожидание: 200 OK, status=MERGED; повтор с тем же ключом — 200 и X-Idempotency: hit.

Статистика:
```
GET /api/stats/assignments
```
Возвращает агрегаты по ревьюверам/PR, кэш 30с в Redis.

## Метрики

Экспортируются в /metrics:
- assignments_total, reassignments_total, merges_total, rate_limited_total, idempotency_hits_total.
- http_request_duration_ms гистограмма (если подключён middleware времени).

Смотреть:
- Браузер: http://localhost:8095/metrics  
- Prometheus scrape target: localhost:8095.

## Rate limiting, идемпотентность, дедупликация

- Rate limiting: Redis‑базированный фиксированное окно (пример), 429 при превышении; Retry-After=1.
- Идемпотентность: header Idempotency-Key, кеширование успешных 2xx/3xx ответов на TTL (обычно 1 час).
- Дедупликация: короткое окно (5с) на идентичные write‑запросы по хешу метода/пути/тела; X-Dedupe: hit при возврате кеша.

## Дополнительные задания

- Статистика: GET /api/stats/assignments — реализовано (service + Redis cache).
- Нагрузочное тестирование: k6/load.js, пороги p95<300мс, ошибки<0.1%; запуск make k6, результаты ниже.
- Массовая деактивация: POST /api/team/deactivateAndReassign — транзакционно деактивирует пользователей и переназначает, либо снимает неактивных при отсутствии кандидатов; рассчитано на укладку ~100мс при объёмах ТЗ.
- Интеграционные тесты: test/integration/pr_lifecycle_test.go; запуск make e2e (сервис должен быть запущен).
- Линтер: .golangci.yml; запуск make lint.

## Запуск и команды

- make up — собрать и запустить docker-compose.
- make down — остановить и удалить тома.
- make build / make run — локальный билд/запуск.
- make test — прогон всех тестов Go.
- make e2e — интеграционные тесты против запущенного сервиса.
- make k6 — запуск сценария нагрузки.
- make lint — проверка кодстайла.
- make generate — генерация типов/серверных ручек из OpenAPI.

## Допущения и решения спорных моментов

- Создание пользователей и добавление их в команду не реализованы как эндпоинты — данные задаются сид‑скриптом для упрощения и воспроизводимости проверки; при желании легко расширить API.
- Переназначение выбирает одного случайного активного кандидата из команды заменяемого ревьювера; при отсутствии кандидатов возвращается NO_CANDIDATE, или слот освобождается в массовой операции.
- Лимит двух ревьюверов поддерживается бизнес‑логикой и уникальным ключом (pr_id, reviewer_id); при необходимости можно усилить триггером в БД.
- Идемпотентность merge обеспечивается состоянием PR и кэшированием по Idempotency-Key; повтор без ключа также безопасен (операция детерминирована).
- Производительность: индексы на status, author_id, pr_id; Redis используется для горячих путей (rate limit, idem, dedupe, stats cache) для соблюдения SLI 300 мс при RPS≈5.

## Результаты нагрузки (пример)

- k6 (10 vus, 1m):  
  - http_req_duration p(95): < 300 ms  
  - http_req_failed rate: < 0.1%  
  Логи запуска и сводка приложены в раздел «benchmarks» README или артефактом CI (опционально).

## Структура репозитория

- cmd/server — точка входа.
- internal/{transport,httpapi}/handlers — HTTP‑хендлеры.
- internal/service — бизнес‑логика.
- internal/repository — Postgres, миграции/сиды (go:embed).
- internal/cache — Redis клиент.
- internal/metrics — метрики Prometheus.
- openapi/openapi.yml — спецификация.
- test/integration — интеграционные тесты.
- k6/load.js — нагрузочный сценарий.
- Makefile, docker-compose.yml, .golangci.yml, README.md.
