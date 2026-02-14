
### Blog API

```text
REST API для блога на Go с PostgreSQL и JWT авторизацией.
```

## О проекте

```text
Blog API — это RESTful API для управления блогом с пользователями, постами и комментариями. 
- Пользователи могут регистрироваться, логиниться и получать JWT-токен.
- Авторизованные пользователи могут создавать, редактировать и удалять посты и комментарии.
- Все данные хранятся в PostgreSQL.
- JWT обеспечивает безопасный доступ к защищённым маршрутам.
```

## Структура проекта

```text
final_project/
├── cmd/api/ # Точка входа приложения
├── internal/ # Внутренние пакеты (handler, service, repository, middleware)
├── pkg/ # Общие пакеты (auth, database)
├── docker-compose.yml
├── .env
├── go.mod
└── README.md
```

## Требования


- Go 1.21+
- Docker & Docker Compose
- PostgreSQL (через Docker)
- curl или Postman


## Миграции и начальные данные

```text
Docker автоматически создаёт таблицы и начальные данные.
Можно добавить пример SQL-запроса, который используется в 001_init_schema.sql.
```

## Установка

```bash
git clone <your-repo-url>

cd final_project
```
```text
Создайте файл .env (необязательно, значения по умолчанию будут использованы):
```
```bash
SERVER_HOST=localhost # Хост сервера  
SERVER_PORT=8080 # Порт сервера

DB_HOST=localhost # Хост PostgreSQL 
DB_PORT=5432 # Порт PostgreSQL 
DB_USER=bloguser # Пользователь базы
DB_PASSWORD=blogpassword # Пароль
DB_NAME=blogdb # Название базы
DB_SSLMODE=disable

JWT_SECRET=secret # Секрет для подписи JWT
JWT_EXPIRY_HOURS=24 # Время жизни токена (часы)
CACHE_TTL_MINUTES=5
```

## Запуск через Docker

```text
Поднимаем PostgreSQL и Adminer:
```

```bash
docker-compose up -d
```

```text
Проверяем статус:
```

```bash
docker ps
docker logs blog_postgres
```

```text
База данных создаст схему и таблицы автоматически.

Для полной очистки данных:
```

```bash
docker-compose down -v
```

## Запуск Go сервера

```text
Установить зависимости:
```

```bash
go mod tidy
```

```text
Запуск сервера:
```

```bash
go run ./cmd/api
```

```text
Сервер стартует на localhost:8080.

Проверка health:
```
```bash
curl http://localhost:8080/api/health
# {"status":"ok","service":"blog-api"}
```

## Работа с API через curl

```text
1️⃣ Регистрация пользователя
```
```bash
curl -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@test.com",
    "username": "test",
    "password": "123456"
  }'
```
```text
Возвращается JSON с JWT токеном:
```
```bash
{
  "token": "<JWT_TOKEN>",
  "expires_at": "2026-02-15T20:15:34.2281859+03:00",
  "user": {
    "id": 1,
    "username": "test",
    "email": "test@test.com",
    "created_at": "2026-02-14T20:15:34.2251854+03:00"
  }
}
```

```text
⚠️ Если пользователь уже существует, вернётся ошибка {"error":"user already exists"}
```

```text
2️⃣ Логин
```
```bash
curl -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@test.com",
    "password": "123456"
  }'
```
```text
Ответ такой же, как при регистрации: JWT токен + информация о пользователе.
```
```text
3️⃣ Создание поста (требуется JWT)
```
```bash
JWT="<JWT_TOKEN>"

curl -X POST http://localhost:8080/api/posts \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $JWT" \
  -d '{
    "title": "Мой первый пост",
    "content": "Привет, это тестовый пост"
  }'
```

```text
Возвращается JSON с созданным постом.
```
```text
4️⃣ Получение всех постов (публично)
```
```bash
curl http://localhost:8080/api/posts
```

```text
5️⃣ Получение поста по ID (публично)
```
```bash
curl http://localhost:8080/api/posts/1
```
```text
6️⃣ Добавление комментария к посту (требуется JWT)
```
```bash
JWT="<JWT_TOKEN>"

curl -X POST http://localhost:8080/api/posts/1/comments \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $JWT" \
  -d '{
    "content": "Отличный пост!"
  }'
```
```text
7️⃣ Редактирование и удаление поста (требуется JWT)
```
```text
Редактировать:
```
```bash
JWT="<JWT_TOKEN>"

curl -X PUT http://localhost:8080/api/posts/1 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $JWT" \
  -d '{
    "title": "Обновлённый заголовок",
    "content": "Обновлённый текст поста"
  }'
```
```text
Удалить:
```
```bash
JWT="<JWT_TOKEN>"

curl -X DELETE http://localhost:8080/api/posts/1 \
  -H "Authorization: Bearer $JWT"
```

## Ошибки и обработка
```text
Примеры ошибок и их коды HTTP:

- 400 Bad Request — неверный формат данных
- 401 Unauthorized — отсутствует или недействителен токен
- 404 Not Found — ресурс не найден
- 409 Conflict — пользователь уже существует
```

## Получение JWT токена для использования в API

# PowerShell
```bash
$response = curl -Method POST http://localhost:8080/api/login `
  -Headers @{ "Content-Type" = "application/json" } `
  -Body '{ "email": "test@test.com", "password": "123456" }'

$token = ($response.Content | ConvertFrom-Json).token
```

# Bash
```bash
response=$(curl -s -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","password":"123456"}')

token=$(echo $response | jq -r '.token')
```

## Примеры структуры JSON для запросов

# Создание поста
```json
{
  "title": "Заголовок",
  "content": "Содержимое поста"
}
```
# Добавление комментария
```json
{
  "content": "Текст комментария"
}
```
## Пример использования Adminer для просмотра БД

```text
Админка Adminer доступна на http://localhost:8081.  
- Сервер: blog_postgres  
- Пользователь: bloguser  
- Пароль: blogpassword  
- База: blogdb
```

## Примечания

```text
Все защищённые маршруты требуют заголовок Authorization: Bearer <JWT_TOKEN>.
Пароли хранятся в базе в виде хеша.
```