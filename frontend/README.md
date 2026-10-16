# Фронтенд календаря звонков

Vue 3 + TypeScript + Vite. Требуется Node.js 26.
Запуск всего приложения и CI описаны в [корневом README](../README.md).

Команды из этой директории:

```bash
npm ci
npm run dev
npm test
npm run test:watch
npm run lint
npm run format:check
npm run build
```

Прокси `/api` направлен на `http://127.0.0.1:8080`. При запуске через `make dev`
его адрес можно изменить через `.env` в корне проекта.

shadcn-vue настроен через `components.json`; компоненты добавляются по мере
появления задач командой `npx shadcn-vue@latest add <component>`.

Известное ограничение зависимостей (16.10.2026): `npm audit` сообщает о
[braces](https://github.com/advisories/GHSA-vfj7-8cjw-p6xm) и
[decode-uri-component](https://github.com/advisories/GHSA-vcc3-ghjq-m6fr)
в цепочках зависимостей ESLint и CLI shadcn-vue. Совместимый `npm audit fix`
их не устраняет; `--force` предлагает откат инструментов. Эти пакеты
используются инструментами разработки и не входят в клиентский JS-бандл.
