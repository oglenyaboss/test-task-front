# 1) ТЗ для GPT-5 Кодекса — WMS API (Go + Postgres + Docker) с OpenAPI

## Цель

Собрать рабочий backend-сервис WMS для тестового задания фронтендера:

- Стек: **Go 1.22**, **PostgreSQL 16**, **Docker Compose**.
- Аутентификация: **JWT Bearer**. `access_token` (TTL 15м), `refresh_token` (TTL 7д) — реализовать.
- Эндпоинты: `POST /auth/login`, `POST /auth/refresh`, `GET /items`, `GET /items/{id}`, `PATCH /items/{id}` (qtyDelta).
- **OpenAPI 3.1**: файл `web/swagger/openapi.yaml` и Swagger UI на `/swagger`.
- Стабильные сид-данные, CORS `*`, искусственная задержка на `/items` 300–500 мс (env).
- Запуск одной командой: `docker compose up` — сервис доступен на `http://localhost:8080`.

## Функциональные требования

### Эндпоинты и поведение

- `POST /auth/login` → `{access_token, refresh_token}`; 401 с `{code:"unauthorized", message:"invalid credentials"}`.
- `POST /auth/refresh` → `{access_token}`; 401 при неверном/просроченном refresh.
- `GET /items` (Bearer): query `q, page(>=1, default 1), limit(<=100, default 20), sort(name|sku|qty|updated_at), dir(asc|desc, default asc)`.  
  Ответ: `{ items: Item[], page, page_size, total }`. Добавить задержку (см. env).
- `GET /items/{id}` (Bearer): Item или 404 `{code:"not_found"}`.
- `PATCH /items/{id}` (Bearer): тело `{ qtyDelta: number }` (может быть отрицательным).  
  Возвращает обновлённый Item; 400 если итоговый `qty < 0`.

### Модель ошибок (везде)

```json
{
  "code": "unauthorized|not_found|bad_request|conflict|internal",
  "message": "..."
}
```

### Безопасность

- JWT HS256; секрет в `JWT_SECRET`.
- Пароли хранить через **bcrypt** (cost 12).
- Таблица `refresh_tokens` с отзывом/экспирацией.

### Схема БД (Postgres)

- `users(id uuid pk, email citext unique not null, password_hash text not null, created_at timestamptz default now())`
- `items(id uuid pk, sku text unique not null, name text not null, qty int not null check(qty>=0), location text not null, updated_at timestamptz not null default now())`
- `refresh_tokens(id uuid pk, user_id uuid not null, token text not null, expires_at timestamptz not null, revoked_at timestamptz null)`
- Триггер/логика авто-обновления `updated_at` при изменении `items`.

### Сид-данные

- Пользователь: `test@example.com` / `Password123!`
- 120 товаров:
  - `sku`: `SKU-0001` … `SKU-0120`
  - `name`: `Widget 1` … `Widget 120`
  - `qty`: псевдослучайно 0–500 (фиксированный seed)
  - `location`: по кругу `A-01-01`…`A-10-12`

### CORS и задержка

- `Access-Control-Allow-Origin: *`; методы `GET, POST, PATCH, OPTIONS`; заголовки `Authorization, Content-Type`.
- Задержка на `/items`: `ARTIFICIAL_DELAY_MS` (default 400).

## Проект и инфраструктура

### Структура

```
/cmd/api/main.go
/internal/http/handlers/*.go
/internal/http/middleware/*.go
/internal/core/models.go
/internal/core/repo/*.go
/internal/core/services/*.go
/internal/auth/jwt.go
/internal/config/config.go
/migrations/*.sql
/seed/seed.go   (или сиды в миграции)
/web/swagger/openapi.yaml
Dockerfile
docker-compose.yml
.env.example
Makefile
README.md
```

### Технологии/пакеты

- Роутер: **chi** или **gin** (любой).
- DB: **sqlx** или **gorm** (любой).
- JWT: `github.com/golang-jwt/jwt/v5`
- bcrypt: `golang.org/x/crypto/bcrypt`
- Миграции: **golang-migrate** или **pressly/goose** (на твой выбор; CLI в контейнере).

### Конфиг (env)

```
APP_PORT=8080
APP_ENV=dev
JWT_SECRET=devsecret_change_me
ACCESS_TOKEN_TTL=15m
REFRESH_TOKEN_TTL=168h
DATABASE_URL=postgres://postgres:postgres@db:5432/wms?sslmode=disable
CORS_ALLOWED_ORIGINS=*
ARTIFICIAL_DELAY_MS=400
```

### Docker Compose — ожидания по сервисам

- `db`: `postgres:16`, volume для данных, `POSTGRES_PASSWORD=postgres`, БД `wms`.
- `api`: билд из `Dockerfile`, зависит от `db`, ждёт healthcheck БД, прогоняет миграции, сиды и стартует HTTP.
- (Опц.) `swagger-ui`: можно встроить в `api` как статик `/swagger`.

### README: что описать

- Запуск: `cp .env.example .env && docker compose up`.
- Swagger: `http://localhost:8080/swagger` и `http://localhost:8080/openapi.yaml`.
- Логин: `test@example.com / Password123!`
- Примеры `curl` (см. ниже).

## OpenAPI 3.1 (положить в `web/swagger/openapi.yaml`)

```yaml
openapi: 3.1.0
info:
  title: WMS Test API
  version: 1.0.0
servers:
  - url: http://localhost:8080
components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
  schemas:
    Error:
      type: object
      required: [code, message]
      properties:
        code: { type: string }
        message: { type: string }
    LoginRequest:
      type: object
      required: [email, password]
      properties:
        email: { type: string, format: email }
        password: { type: string, minLength: 6 }
    LoginResponse:
      type: object
      required: [access_token, refresh_token]
      properties:
        access_token: { type: string }
        refresh_token: { type: string }
    RefreshRequest:
      type: object
      required: [refresh_token]
      properties:
        refresh_token: { type: string }
    RefreshResponse:
      type: object
      required: [access_token]
      properties:
        access_token: { type: string }
    Item:
      type: object
      required: [id, sku, name, qty, location, updated_at]
      properties:
        id: { type: string, format: uuid }
        sku: { type: string }
        name: { type: string }
        qty: { type: integer, minimum: 0 }
        location: { type: string }
        updated_at: { type: string, format: date-time }
    ItemsResponse:
      type: object
      required: [items, page, page_size, total]
      properties:
        items:
          type: array
          items: { $ref: "#/components/schemas/Item" }
        page: { type: integer, minimum: 1 }
        page_size: { type: integer, minimum: 1 }
        total: { type: integer, minimum: 0 }
security:
  - bearerAuth: []
paths:
  /auth/login:
    post:
      summary: Login
      requestBody:
        required: true
        content:
          application/json:
            schema: { $ref: "#/components/schemas/LoginRequest" }
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema: { $ref: "#/components/schemas/LoginResponse" }
        "401":
          description: Unauthorized
          content:
            application/json:
              schema: { $ref: "#/components/schemas/Error" }
  /auth/refresh:
    post:
      summary: Refresh access token
      requestBody:
        required: true
        content:
          application/json:
            schema: { $ref: "#/components/schemas/RefreshRequest" }
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema: { $ref: "#/components/schemas/RefreshResponse" }
        "401":
          description: Unauthorized
          content:
            application/json:
              schema: { $ref: "#/components/schemas/Error" }
  /items:
    get:
      summary: List items
      parameters:
        - in: query
          name: q
          schema: { type: string }
        - in: query
          name: page
          schema: { type: integer, minimum: 1, default: 1 }
        - in: query
          name: limit
          schema: { type: integer, minimum: 1, maximum: 100, default: 20 }
        - in: query
          name: sort
          schema:
            { type: string, enum: [name, sku, qty, updated_at], default: name }
        - in: query
          name: dir
          schema: { type: string, enum: [asc, desc], default: asc }
      security:
        - bearerAuth: []
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema: { $ref: "#/components/schemas/ItemsResponse" }
        "401":
          description: Unauthorized
          content:
            application/json:
              schema: { $ref: "#/components/schemas/Error" }
  /items/{id}:
    get:
      summary: Get item by id
      parameters:
        - in: path
          name: id
          required: true
          schema: { type: string, format: uuid }
      security:
        - bearerAuth: []
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema: { $ref: "#/components/schemas/Item" }
        "404":
          description: Not found
          content:
            application/json:
              schema: { $ref: "#/components/schemas/Error" }
        "401":
          description: Unauthorized
          content:
            application/json:
              schema: { $ref: "#/components/schemas/Error" }
    patch:
      summary: Adjust quantity by delta
      parameters:
        - in: path
          name: id
          required: true
          schema: { type: string, format: uuid }
      security:
        - bearerAuth: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [qtyDelta]
              properties:
                qtyDelta: { type: number }
      responses:
        "200":
          description: Updated
          content:
            application/json:
              schema: { $ref: "#/components/schemas/Item" }
        "400":
          description: Bad request
          content:
            application/json:
              schema: { $ref: "#/components/schemas/Error" }
        "404":
          description: Not found
          content:
            application/json:
              schema: { $ref: "#/components/schemas/Error" }
        "401":
          description: Unauthorized
          content:
            application/json:
              schema: { $ref: "#/components/schemas/Error" }
```

## Примеры cURL (для README)

```bash
# Login
curl -s http://localhost:8080/auth/login -X POST \
  -H 'Content-Type: application/json' \
  -d '{"email":"test@example.com","password":"Password123!"}'

# List
curl -s http://localhost:8080/items?q=widget&page=1&limit=20&sort=name&dir=asc \
  -H "Authorization: Bearer <ACCESS_TOKEN>"

# Get by id
curl -s http://localhost:8080/items/<UUID> -H "Authorization: Bearer <ACCESS_TOKEN>"

# Patch qty
curl -s http://localhost:8080/items/<UUID> -X PATCH \
  -H "Authorization: Bearer <ACCESS_TOKEN>" -H 'Content-Type: application/json' \
  -d '{"qtyDelta": -2}'
```

## Критерии приёмки для этого backend

- Проект поднимается `docker compose up`, Swagger доступен, сиды загружены.
- Все эндпоинты работают, JWT-поток логина/рефреша корректен.
- Фильтрация/сортировка/пагинация `/items` корректны, `total` возвращается.
- CORS `*`, задержка на `/items` учитывается из env.
- Код разложен по слоям, есть миграции, README понятный.
