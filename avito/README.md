# PR Reviewer Assignment Service

Сервис автоматического назначения ревьюеров на Pull Request'ы

## Описание

Микросервис предоставляет HTTP API для:
- управления командами и участниками;
- создания PR с автоматическим назначением до двух активных ревьюверов из команды автора (автор исключается);
- переназначения ревьюверов в соответствии с правилами ТЗ;
- merge PR (идемпотентная операция); [file:169]
- получения списка PR, назначенных на конкретного пользователя. [file:169]

API строго соответствует спецификации `openapi.yml` в корне репозитория. [file:1]

## Запуск

Требуются Docker и Docker Compose. [file:169]

Из корня проекта: [file:169]

```
docker-compose up --build
```

Команда: [file:169]
- поднимет PostgreSQL (user: `app`, password: `app`, db: `app`); [file:169]
- соберёт и запустит сервис на Go 1.25.1; [file:169]
- автоматически применит SQL-миграцию `001_init.sql` (создание таблиц и enum `pr_status`). [file:169]

После старта сервис доступен по адресу `http://localhost:8080`. [file:169]

Проверка доступности: [file:1]

```
curl http://localhost:8080/health
```

Ожидаемый ответ: [file:1]

```
{"status":"ok"}
```

## Основные эндпоинты

Ниже краткие примеры запросов, полный контракт описан в `openapi.yml`. [file:1]

### Команды

Создать команду: [file:1]

```
curl -X POST http://localhost:8080/team/add \
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

Получить команду: [file:1]

```
curl "http://localhost:8080/team/get?team_name=backend"
```

### Пользователи

Смена активности пользователя: [file:1]

```
curl -X POST http://localhost:8080/users/setIsActive \
  -H "Content-Type: application/json" \
  -d '{ "user_id": "u2", "is_active": false }'
```

Получить PR, назначенные пользователю: [file:1]

```
curl "http://localhost:8080/users/getReview?user_id=u2"
```

### Pull Request'ы

Создать PR (автоназначение ревьюверов): [file:1][file:169]

```
curl -X POST http://localhost:8080/pullRequest/create \
  -H "Content-Type: application/json" \
  -d '{
        "pull_request_id": "pr-1",
        "pull_request_name": "Add feature",
        "author_id": "u1"
      }'
```

Переназначить ревьювера: [file:1][file:169]

```
curl -X POST http://localhost:8080/pullRequest/reassign \
  -H "Content-Type: application/json" \
  -d '{
        "pull_request_id": "pr-1",
        "old_user_id": "u2"
      }'
```

Merge PR (идемпотентно): [file:1][file:169]

```
curl -X POST http://localhost:8080/pullRequest/merge \
  -H "Content-Type: application/json" \
  -d '{ "pull_request_id": "pr-1" }'
```

Повторный вызов возвращает актуальное состояние PR в статусе `MERGED` без ошибки. [file:1][file:169]

## Архитектура

Проект разбит на слои: [file:169]
- `cmd/app` — точка входа, инициализация подключения к БД, миграций и HTTP‑роутера. [file:169]
- `internal/domain` — доменные модели (`Team`, `User`, `PullRequest` и т.п.). [file:169]
- `internal/service` — бизнес-логика (назначение/переназначение ревьюверов, merge, работа с командами и пользователями). [file:169]
- `internal/repository/postgres` — репозитории поверх PostgreSQL (`teams`, `users`, `pull_requests`, `pull_request_reviewers`). [file:169]
- `internal/http` — HTTP‑хендлеры и роутер на основе `chi`. [file:169]
- `internal/db` — подключение к базе и применение миграций через `go:embed`. [file:169]
- `internal/errs` — доменные ошибки и маппинг в формат `ErrorResponse` из OpenAPI. [file:1][file:169]

Сервис использует интерфейсы репозиториев, поэтому при необходимости in-memory реализацию можно включить, не меняя сервисный слой. [file:169]

## Ключевые решения и допущения

1. **PostgreSQL как основное хранилище.**
   Несмотря на допустимую in-memory реализацию в ТЗ, используется PostgreSQL 16, чтобы показать работу с реальной БД и поддержать персистентность данных между перезапусками. [file:169]

2. **Миграции через `go:embed`.**
   Одна SQL‑миграция `001_init.sql` встраивается в бинарник и выполняется при старте, тип `pr_status` создаётся в DO‑блоке с проверкой существования, чтобы миграции были идемпотентны. [file:169]

3. **Случайный выбор ревьюверов.**
   Для назначения и переназначения используется `math/rand` с перетасовкой списка кандидатов, что удовлетворяет требованию случайности в рамках тестового задания. [file:169]

4. **Ожидание готовности БД.**
   При старте сервис делает несколько попыток `Ping` к PostgreSQL перед запуском HTTP‑сервера, чтобы избежать падения из‑за гонки старта контейнеров. [file:169]

5. **Доменные ошибки и HTTP‑коды.**
   Все коды ошибок (`TEAM_EXISTS`, `PR_EXISTS`, `PR_MERGED`, `NOT_ASSIGNED`, `NO_CANDIDATE`, `NOT_FOUND`) маппятся в формат `ErrorResponse` с корректными HTTP‑статусами (`400`, `404`, `409`) согласно `openapi.yml`. [file:1]

## Дальнейшее развитие

Потенциальные улучшения, перекрывающие дополнительные пункты из ТЗ: [file:169]
- Эндпоинт статистики (количество PR по статусам и/или по ревьюверам). [file:169]
- Интеграционные тесты поверх HTTP API с использованием тестовой БД. [file:169]
- Конфигурация линтера (например, `golangci-lint`) и её описание в README. [file:169]
```

[1](https://habr.com/ru/companies/otus/articles/531624/)
[2](https://www.reddit.com/r/golang/comments/hcvm6j/simple_microservice_boilerplate/)
[3](https://www.youtube.com/watch?v=VQAX_W2cXQc)
[4](https://gitlab.mai.ru/online-store-pet-project/backend/product-service/-/blob/main/README.md)
[5](https://praxiscode.io/knowledge-base/golang-project-structure-guide)
[6](https://www.reddit.com/r/Python/comments/13kpoti/readmeai_autogenerate_readmemd_files/)
[7](https://olezhek28.courses/microservices)
[8](https://dev-gang.ru/article/micro-v-deistvii-czast--polnoe-rukovodstvo-po-bootstrap-9oyo5h33wi/)
[9](https://gitlab.itcomgk.ru/crm-core-clients/certificates/-/blob/1.0.10/README.md)
[10](https://ppl-ai-file-upload.s3.amazonaws.com/web/direct-files/attachments/88172437/c16ab72b-4dd3-4c35-adb2-8ba88dfd8e3f/Backend-trainee-assignment-autumn-2025.md)
