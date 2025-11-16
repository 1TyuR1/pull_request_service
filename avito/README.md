
# PR Reviewer Assignment Service

Сервис автоматического назначения ревьюеров на Pull Request'ы по тестовому заданию Backend (осень 2025). 

## Описание

Микросервис предоставляет HTTP API для:
- управления командами и участниками; 
- создания PR с автоматическим назначением до двух активных ревьюверов из команды автора (автор исключается); 
- переназначения ревьюверов по правилам ТЗ (в том числе массово при деактивации команды); 
- merge PR (идемпотентная операция); 
- получения списка PR, назначенных конкретному пользователю; 
- получения простой статистики по назначениям и статусам PR.

API следует спецификации `openapi.yml`, лежащей в корне репозитория. 

## Запуск

Требуются Docker и Docker Compose. 

Из корня проекта: 

```
docker-compose up --build
```

Команда: [file:169]
- поднимет PostgreSQL (user: `app`, password: `app`, db: `app`); 
- соберёт и запустит сервис на Go 1.25.1; 
- автоматически применит SQL‑миграцию `001_init.sql` (создание таблиц и enum `pr_status`). 

После старта сервис доступен по адресу `http://localhost:8080`. 

Проверка доступности: 

```
curl http://localhost:8080/health
```

Ожидаемый ответ: 

```
{"status":"ok"}
```

Также в корне есть `Makefile` с удобными командами: 

```
make build        # локальная сборка бинарника
make run          # запуск без Docker (нужен DATABASE_URL)
make test         # запуск тестов, если добавлены
make docker-up    # docker-compose up --build
make docker-down  # docker-compose down
make docker-down-v # docker-compose down -v (снос volume с БД)
make docker-logs  # хвост логов app и db
make lint         # запуск линтера (при наличии golangci-lint)
```

## Основные эндпоинты

Ниже краткие примеры запросов; полный контракт описан в `openapi.yml`.

### Команды

Создать команду: 

```
curl -i -X POST http://localhost:8080/team/add \
  -H "Content-Type: application/json" \
  -d '{
        "team_name": "backend",
        "members": [
          {"user_id": "u1", "username": "alice",   "is_active": true},
          {"user_id": "u2", "username": "bob",     "is_active": true},
          {"user_id": "u3", "username": "charlie", "is_active": true}
        ]
      }'
```

Получить команду: 

```
curl -i "http://localhost:8080/team/get?team_name=backend"
```

Массовая деактивация пользователей команды с безопасным переназначением их открытых PR: 

```
curl -i -X POST http://localhost:8080/team/deactivateUsers \
  -H "Content-Type: application/json" \
  -d '{ "team_name": "backend" }'
```

Ожидаемый ответ:

```
{
  "team_name": "backend",
  "deactivated_users": 3,
  "reassigned_reviewers": 2
}
```

### Пользователи

Смена активности пользователя: 

```
curl -i -X POST http://localhost:8080/users/setIsActive \
  -H "Content-Type: application/json" \
  -d '{ "user_id": "u2", "is_active": false }'
```

Получить PR, назначенные пользователю: 

```
curl -i "http://localhost:8080/users/getReview?user_id=u2"
```

Ответ: 

```
{
  "user_id": "u2",
  "pull_requests": [
    {
      "pull_request_id": "pr-1",
      "pull_request_name": "Add feature",
      "author_id": "u1",
      "status": "OPEN"
    }
  ]
}
```

### Pull Request'ы

Создать PR (автоназначение ревьюверов): 

```
curl -i -X POST http://localhost:8080/pullRequest/create \
  -H "Content-Type: application/json" \
  -d '{
        "pull_request_id": "pr-1",
        "pull_request_name": "Add feature",
        "author_id": "u1"
      }'
```

Переназначить ревьювера: 

```
curl -i -X POST http://localhost:8080/pullRequest/reassign \
  -H "Content-Type: application/json" \
  -d '{
        "pull_request_id": "pr-1",
        "old_user_id": "u2"
      }'
```

Ответ содержит обновлённый PR и `replaced_by` с `user_id` нового ревьювера. 

Merge PR (идемпотентно): 

```
curl -i -X POST http://localhost:8080/pullRequest/merge \
  -H "Content-Type: application/json" \
  -d '{ "pull_request_id": "pr-1" }'
```

Повторный вызов возвращает актуальное состояние PR со статусом `MERGED` без ошибки. 

### Статистика

Простой эндпоинт статистики: 

```
curl -i http://localhost:8080/stats
```

Пример ответа: 

```
{
  "per_reviewer": {
    "u2": 5,
    "u3": 8
  },
  "per_status": {
    "OPEN": 3,
    "MERGED": 10
  }
}
```

## Архитектура

Проект разбит на слои:
- `cmd/app` — точка входа, инициализация подключения к БД, миграций и HTTP‑роутера. 
- `internal/domain` — доменные модели (`Team`, `User`, `PullRequest` и т.д.).
- `internal/service` — бизнес-логика (назначение и переназначение ревьюверов, merge, управление командами и пользователями, массовая деактивация). 
- `internal/repository/postgres` — репозитории поверх PostgreSQL (`teams`, `users`, `pull_requests`, `pull_request_reviewers`). 
- `internal/http` — HTTP‑хендлеры и роутер на базе `chi`. 
- `internal/db` — подключение к БД и применение миграций через `go:embed`.
- `internal/errs` — доменные ошибки и маппинг в формат `ErrorResponse` из OpenAPI. 

Сервис использует интерфейсы репозиториев, поэтому можно при необходимости включить in‑memory реализацию для локальных тестов, не меняя сервисный слой. 

## Дополнительные задания

Реализованы следующие дополнительные фичи из ТЗ: 
- эндпоинт статистики `/stats` (количество назначений по ревьюверам и PR по статусам).
- метод массовой деактивации пользователей команды `/team/deactivateUsers` с безопасным переназначением открытых PR.
- Makefile с командами сборки, запуска и обслуживания стенда.
