# Календарь звонков

[![hexlet-check](https://github.com/madatsci/ai-for-developers-project-386/actions/workflows/hexlet-check.yml/badge.svg)](https://github.com/madatsci/ai-for-developers-project-386/actions)

Учебный сервис для публичного бронирования звонков и управления расписанием
организатора. Приложение включает Vue-интерфейс, Go API и PostgreSQL.

- [Публичное приложение](https://calendar-of-calls.onrender.com)
- [Административный интерфейс](https://calendar-of-calls.onrender.com/admin/calls)
- [Программа Хекслета](https://ru.hexlet.io/programs/ai-for-developers)
- [Описание проекта](https://files.hexlet.app/a/2ipc5m)

Стек: Go 1.26, `net/http`, `chi`, PostgreSQL 18, Vue 3, TypeScript, Vite,
Tailwind CSS и TypeSpec.

## Запуск и настройка

Нужны Go 1.26+, Node.js 26 с npm, Docker Compose v2, GNU Make и curl.

```bash
git clone https://github.com/madatsci/ai-for-developers-project-386.git
cd ai-for-developers-project-386
make setup
make dev
```

`make dev` запускает PostgreSQL, применяет миграции и seed, затем поднимает API и
frontend.

| Сервис           | Адрес                              |
| ---------------- | ---------------------------------- |
| Frontend         | <http://127.0.0.1:5175>            |
| API health check | <http://127.0.0.1:8080/api/health> |
| PostgreSQL       | `127.0.0.1:5433`                   |

Для запуска компонентов в отдельных терминалах:

```bash
make db-up && make seed
make dev-backend   # терминал 1
make dev-frontend  # терминал 2
```

### Конфигурация

Значения для локальной разработки заданы в `Makefile`. Для переопределения
скопируйте `.env.example` в `.env`.

| Переменная         | Назначение                                    |
| ------------------ | --------------------------------------------- |
| `DATABASE_URL`     | Подключение API, миграций и seed к PostgreSQL |
| `HTTP_ADDR`        | Адрес локального Go API                       |
| `API_PROXY_TARGET` | Адрес API для Vite proxy                      |
| `FRONTEND_PORT`    | Порт локального Vite-сервера                  |
| `PORT`             | Внутренний HTTP-порт production-контейнера    |

Учётные данные `calendar/calendar` предназначены только для локальной БД.

### Контейнерный запуск

```bash
docker compose up --build --wait app
```

Приложение будет доступно на <http://127.0.0.1:8080>. Остановка с сохранением
данных:

```bash
docker compose down
```

## Деплой в Render

[`render.yaml`](render.yaml) описывает Render Web Service и PostgreSQL в регионе
Frankfurt. Render собирает корневой `Dockerfile`, передаёт приложению
`DATABASE_URL` и `PORT`, проверяет `/api/health` и публикует сервис по HTTPS.

Для первого деплоя создайте в Render Blueprint из этого GitHub-репозитория и
выберите ветку `main`. Последующие изменения деплоятся автоматически после
успешного завершения GitHub CI.

Проверка опубликованного контура:

```bash
make smoke-remote REMOTE_SMOKE_BASE_URL=https://calendar-of-calls.onrender.com
```

## Проверки

```bash
make test   # Go и frontend-тесты
make check  # полная CI-эквивалентная проверка
```

## Подводные камни

- Production-контейнер слушает `0.0.0.0:$PORT`; локальный API использует
  `HTTP_ADDR`.
- Контейнер применяет миграции и идемпотентный seed перед каждым запуском API.
- `make smoke-remote` создаёт реальный звонок и оставляет его в Render PostgreSQL.
- Источник HTTP-контракта — `api/*.tsp`. Generated OpenAPI, frontend SDK и Go
  server package нельзя редактировать вручную; для их обновления используйте
  `make generate`.
- Бесплатный Render Web Service засыпает без трафика, поэтому первый запрос после
  простоя может выполняться дольше.
- Бесплатная Render PostgreSQL имеет ограниченный срок жизни. Отключение Blueprint
  не удаляет Web Service и базу данных — эти ресурсы удаляются отдельно.

---

<details>
<summary>Автоматические тесты Хекслета</summary>

Тесты запускаются на каждый коммит. За запуск отвечает файл `.github/workflows/hexlet-check.yml` — не удаляйте и не переименовывайте ни его, ни репозиторий.

</details>

## О Хекслете

[Хекслет](https://ru.hexlet.io/) — школа программирования: авторские программы обучения с практикой, поддержкой наставников и реальными проектами, которые остаются в резюме. Этот репозиторий — один из таких проектов.
