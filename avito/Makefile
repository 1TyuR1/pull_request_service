BINARY_NAME := pr-reviewer-service

.PHONY: build run test docker-up docker-down docker-logs lint

## Сборка бинарника (локально)
build:
	go build -o bin/$(BINARY_NAME) ./cmd/app

## Запуск сервиса локально (без Docker, нужен запущенный Postgres и DATABASE_URL)
run:
	go run ./cmd/app


## Поднять сервис и Postgres через docker-compose
docker-up:
	docker-compose up --build

## Остановить сервис и БД, сохранить данные в volume
docker-down:
	docker-compose down

## Остановить сервис и БД и удалить volume с данными БД
docker-down-v:
	docker-compose down -v

## Логи приложения и БД
docker-logs:
	docker-compose logs -f

## Проверка кода линтером (если настроен golangci-lint)
lint:
	golangci-lint run ./...
