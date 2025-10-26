# WMS Test API

Backend сервис для тестового задания фронтенд-разработчика. Реализован на Go 1.22 с PostgreSQL 16 и запускается одной командой через Docker Compose.

## Запуск

```bash
cp .env.example .env
docker compose up --build
```

После старта:

- API: http://localhost:8080
- Swagger UI: http://localhost:8080/swagger
- OpenAPI: http://localhost:8080/openapi.yaml

В сид-данных создаётся учётная запись `test@example.com / Password123!`. При первом запуске автоматически применяются миграции и загружаются 120 товаров.

## Локальная разработка

```bash
# запустить сервис локально (Postgres должен быть доступен)
go run ./cmd/api

# собрать бинарник
go build -o bin/wms ./cmd/api
```

Полезные команды доступны в `Makefile`.

## Переменные окружения

| Переменная | Описание | Значение по умолчанию |
|-----------|----------|------------------------|
| `APP_PORT` | HTTP порт приложения | `8080` |
| `APP_ENV` | окружение (`dev`, `prod`, …) | `dev` |
| `JWT_SECRET` | секрет для подписания JWT | обязательна |
| `ACCESS_TOKEN_TTL` | время жизни access token | `15m` |
| `REFRESH_TOKEN_TTL` | время жизни refresh token | `168h` |
| `DATABASE_URL` | строка подключения к Postgres | обязательна |
| `CORS_ALLOWED_ORIGINS` | список разрешённых origin (через запятую) | `*` |
| `ARTIFICIAL_DELAY_MS` | искусственная задержка для `/items` | `400` |

## Эндпоинты

| Метод | Путь | Описание |
|-------|------|----------|
| `POST` | `/auth/login` | Логин и получение пары токенов |
| `POST` | `/auth/refresh` | Обновление access token по refresh token |
| `GET` | `/items` | Список товаров с фильтрацией/сортировкой |
| `GET` | `/items/{id}` | Получение товара по ID |
| `PATCH` | `/items/{id}` | Изменение количества на дельту |

Все эндпоинты, кроме аутентификации, требуют заголовок `Authorization: Bearer <ACCESS_TOKEN>`.

## Типичный сценарий работы фронтенда

- `POST /auth/login` — пользователь вводит креды, сохраняем `access_token` и `refresh_token`.
- `GET /items` — показываем таблицу с фильтрами, сортировкой и пагинацией.
- `GET /items/{id}` — при открытии карточки товара загружаем актуальные данные.
- `PATCH /items/{id}` — изменения количества (плюс/минус) отправляем дельтой; ответ содержит обновлённый товар.
- `POST /auth/refresh` — по истечении `access_token` запрашиваем новый, используя сохранённый refresh-токен.

## Обработка ошибок и edge-case’ы

- Все ошибки возвращают JSON `{ "code": "...", "message": "..." }`.
- `401 unauthorized` — неверные креды, просроченные/битые токены, отсутствующий заголовок `Authorization`.
- `404 not_found` — запрошенный товар не существует.
- `400 bad_request` — проблемы с валидацией (например, `qtyDelta` отсутствует или конечное количество ушло ниже нуля).
- `409 conflict` пока не используется, но зарезервирован под возможные конфликты склада.
- `500 internal` — непредвиденная ошибка; текст сообщения не предназначен для отображения в UI.

## Примеры cURL

```bash
# Логин
curl -s http://localhost:8080/auth/login -X POST \
  -H 'Content-Type: application/json' \
  -d '{"email":"test@example.com","password":"Password123!"}'

# Список товаров
curl -s http://localhost:8080/items?q=widget&page=1&limit=20&sort=name&dir=asc \
  -H "Authorization: Bearer <ACCESS_TOKEN>"

# Получение товара по ID
curl -s http://localhost:8080/items/<UUID> \
  -H "Authorization: Bearer <ACCESS_TOKEN>"

# Изменение количества
curl -s http://localhost:8080/items/<UUID> -X PATCH \
  -H "Authorization: Bearer <ACCESS_TOKEN>" \
  -H 'Content-Type: application/json' \
  -d '{"qtyDelta": -2}'
```

## Архитектура и стек

- Go 1.22, chi router, sqlx
- PostgreSQL 16, миграции через goose (встраиваются в бинарник)
- JWT (HS256) для аутентификации, refresh-токены в базе
- Слои: handlers → services → repositories
- Swagger UI и OpenAPI 3.1 доступны в `/swagger` и `/openapi.yaml`
- CORS `*`, искусственная задержка на `/items` (300–500 мс по переменной окружения)
