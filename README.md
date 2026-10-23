# Календарь звонков

[![hexlet-check](https://github.com/madatsci/ai-for-developers-project-386/actions/workflows/hexlet-check.yml/badge.svg)](https://github.com/madatsci/ai-for-developers-project-386/actions)

Учебный сервис для бронирования звонков по мотивам Cal.com. Сейчас реализован
каркас: стартовая страница, HTTP API, подключение PostgreSQL и CI.

Учебный проект Хекслета: https://ru.hexlet.io/programs/ai-for-developers
Как это должно работать: https://files.hexlet.app/a/2ipc5m

## Стек

Go 1.26, `net/http`, `chi`, PostgreSQL 18 и pgx; Vue 3, TypeScript, Vite, Vue
Router, shadcn-vue и Tailwind CSS; TypeSpec как Design First источник
HTTP-контракта, Hey API для клиентского SDK и OpenAPI Generator для серверных
Go-артефактов.

Для разработки нужны Go 1.26+, **Node.js 26** с npm, Docker с Compose v2,
GNU Make и curl. Версия Node закреплена в `.node-version` и проверяется при
установке frontend-зависимостей. CI использует ту же версию.

## Установка

```bash
git clone https://github.com/madatsci/ai-for-developers-project-386.git
cd ai-for-developers-project-386
make setup
```

`make setup` устанавливает зависимости из lock-файлов, Chromium для Playwright и
golangci-lint фиксированной версии в `.bin/` проекта. Системная установка линтера
не меняется.

## Локальный запуск

```bash
make dev
```

Команда запускает PostgreSQL, ждёт его готовности и поднимает бэкенд с фронтендом.

| Сервис | Адрес |
| --- | --- |
| Фронтенд | <http://127.0.0.1:5175> |
| Проверка API и соединения с БД | <http://127.0.0.1:8080/api/health> |
| API через прокси фронтенда | <http://127.0.0.1:5175/api/health> |
| PostgreSQL | `127.0.0.1:5433` |

Успешный ответ `/api/health`: HTTP 200 и `{"status":"ok"}`. Если БД становится
недоступна, эндпоинт возвращает HTTP 503. При недоступной БД бэкенд не запускается.
Порт PostgreSQL внутри контейнера — `5432`, на машине — **`5433`**.

Ctrl+C останавливает серверы приложения. `make db-down` останавливает PostgreSQL,
сохраняя данные в Docker volume. Для запуска частей в отдельных терминалах:

```bash
make db-up
make dev-backend   # первый терминал
make dev-frontend  # второй терминал
```

Значения для разработки уже заданы в `Makefile`, копировать `.env` для первого
запуска не требуется. При необходимости скопируйте `.env.example` в `.env`:
`make` загрузит оттуда `DATABASE_URL`, `HTTP_ADDR`, `API_PROXY_TARGET` и
`FRONTEND_PORT`. При изменении адреса бэкенда также измените `API_PROXY_TARGET`.
Учётные данные `calendar/calendar` предназначены только для локальной БД.

### Контейнерный production-запуск

```bash
docker compose up --build --wait app
```

Приложение доступно на <http://127.0.0.1:8080>. Один Go-процесс обслуживает
собранный frontend и same-origin `/api`; перед его запуском контейнер применяет
миграции и идемпотентный seed. PostgreSQL запускается отдельным сервисом Compose.
Остановка с сохранением данных: `docker compose down`.

## Проверки

| Команда | Что проверяет |
| --- | --- |
| `make test` | Go-тесты с race detector и Vitest без watch-режима |
| `make lint` | Линтеры, форматирование, TypeSpec и отсутствие дрейфа generated-кода |
| `make format` | Форматирует Go, frontend и TypeSpec-код |
| `make build` | Go-бинарник, frontend-сборка, OpenAPI и клиентский SDK |
| `make generate-server` | Обновляет сгенерированный Go-пакет из TypeSpec |
| `make lint-server` | Проверяет сгенерированный Go-пакет на дрейф |
| `make container-build` | Собирает production-образ приложения |
| `make smoke` | На чистой БД проверяет контейнерный frontend/API и browser E2E |
| `make check` | Все проверки: lint, test, build и smoke |

По отдельности доступны `make test-backend`, `make test-frontend`,
`make lint-backend`, `make lint-frontend`, `make lint-typespec`,
`make lint-client`, `make lint-server` и `make build-typespec`.

### Генерация API-кода

Единственный редактируемый источник HTTP-контракта находится в `api/*.tsp`.
Серверная цепочка выглядит так:

```text
TypeSpec -> api/tsp-output/openapi.yaml -> backend/internal/httpapi/generated/*.go
```

`make generate-server` сначала пересобирает промежуточный OpenAPI, затем запускает
`go-server` из закреплённого Docker-образа OpenAPI Generator `v7.26.0`, форматирует
Go-файлы и целиком заменяет generated-каталог. Файлы в
`backend/internal/httpapi/generated/` коммитятся, но вручную не редактируются.
`make lint-server` повторяет генерацию во временный каталог и сообщает пути с
дрейфом. Docker нужен этим двум командам и полной проверке `make lint`; обычные
Go-сборка и тесты используют закоммиченный пакет и не запускают генератор.

`make generate-client` также сначала генерирует OpenAPI из TypeSpec, затем
обновляет Fetch SDK в `frontend/src/api/generated/`. CI проверяет соответствие
контракту обоих generated-пакетов. Промежуточный
`api/tsp-output/openapi.yaml` не коммитится и не копируется в backend.

Сгенерированный роутер использует `chi` как `http.Handler` внутри стандартного
`http.Server`. Реализованные controllers подключены к рукописным сервисам; сам
generated-код не содержит доменную логику.

Проект намеренно не использует `kin-openapi` и не загружает OpenAPI-схему во
время работы. Generated controller выполняет разбор и часть проверок, но не
полную runtime-валидацию OpenAPI. В частности, он не гарантирует обязательность
`Idempotency-Key` и `If-Match`, `Content-Type`, лимит тела 64 KiB, единственность
JSON-значения, запрет неизвестных query-параметров, enum, `pattern`, ограничения
длины и размера массивов и `multipleOf`. При подключении каждого продуктового
endpoint эти требования нужно реализовать и протестировать в рукописном
transport/service-адаптере вне generated-каталога.

`make smoke` создаёт изолированный Compose-проект, собирает production-образ,
запускает приложение с чистой PostgreSQL и проверяет frontend assets, same-origin
API, миграции, seed и фактическую запись/чтение данных. Затем Playwright проходит
публичное бронирование и административное изменение режима работы, расписания и
типа звонка. По завершении тестовые контейнеры и volume удаляются. Порты можно
изменить переменными `SMOKE_APP_PORT` и `SMOKE_POSTGRES_PORT`.

## Структура

```text
api/                      TypeSpec-контракт и генерация OpenAPI
backend/cmd/api/          запуск HTTP-сервера и подключение БД
backend/internal/httpapi/ маршруты и тесты HTTP API
backend/internal/httpapi/generated/  сгенерированные Go-модели и controllers
frontend/src/            Vue-приложение и тесты
frontend/src/api/generated/  сгенерированный Fetch SDK
frontend/e2e/            browser E2E production-контура
scripts/smoke.mjs         проверка запуска приложения
Dockerfile               multi-stage production-образ
compose.yaml             приложение и PostgreSQL
.github/workflows/ci.yml  автоматические проверки
```

Workflow `CI` на каждый push и pull request устанавливает зависимости, запускает
линтеры, тесты, сборку и дымовую проверку с PostgreSQL. Исходный workflow Хекслета
сохранён отдельно.

## Релизы

Отдельный workflow [Release Please](.github/workflows/release-please.yml)
запускается при push в `main`, в том числе после мержа PR. Для всего приложения
используется одна версия в `version.txt` и стратегия `simple`.

Все коммиты в проекте должны соответствовать
[Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/): например,
`feat: add application skeleton` или `fix: handle database timeout`.
При squash merge задайте такой заголовок итоговому коммиту. Изменения `feat:`,
`fix:` и breaking changes создают или обновляют общий release-PR с новой версией
и `CHANGELOG.md`. Обычные `chore:`, `ci:` и `docs:` сами по себе релиз не запускают.
После мержа release-PR workflow создаёт тег и GitHub Release.

Для работы со стандартным `GITHUB_TOKEN` в настройках репозитория
**Settings → Actions → General → Workflow permissions** включите
**Allow GitHub Actions to create and approve pull requests**. Права на запись
контента, PR и меток заданы в workflow.

Чтобы CI автоматически запускался и на release-PR, добавьте Actions secret
`RELEASE_PLEASE_TOKEN` с токеном отдельного бота или fine-grained PAT для этого
репозитория с правами **Contents**, **Issues** и **Pull requests: Read and write**.
Без этого секрета workflow использует `GITHUB_TOKEN`: release-PR будет создан,
но его события не запустят другие workflows. Это
[ограничение GitHub, описанное в release-please-action](https://github.com/googleapis/release-please-action#other-actions-on-release-please-prs).

## Официальные руководства

Основа frontend создана официальным `create-vue`. При настройке использованы
[Vue Quick Start](https://vuejs.org/guide/quick-start.html),
[shadcn-vue для Vite](https://www.shadcn-vue.com/docs/installation/vite),
[TypeSpec](https://typespec.io/docs/),
[Hey API](https://heyapi.dev/docs/openapi/typescript/get-started),
[OpenAPI Generator](https://openapi-generator.tech/docs/generators/go-server/),
[chi](https://github.com/go-chi/chi),
[pgxpool](https://pkg.go.dev/github.com/jackc/pgx/v5/pgxpool),
[Docker Compose](https://docs.docker.com/compose/gettingstarted/) и
[golangci-lint](https://golangci-lint.run/docs/welcome/install/).

---

<details>
<summary>Автоматические тесты Хекслета</summary>

Тесты запускаются на каждый коммит. За запуск отвечает файл `.github/workflows/hexlet-check.yml` — не удаляйте и не переименовывайте ни его, ни репозиторий.

</details>

## О Хекслете

[Хекслет](https://ru.hexlet.io/) — школа программирования: авторские программы обучения с практикой, поддержкой наставников и реальными проектами, которые остаются в резюме. Этот репозиторий — один из таких проектов.
