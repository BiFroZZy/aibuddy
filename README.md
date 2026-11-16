# AI-buddy: ИИ-наставник для стажёров

Прототип для кейса Avito Start (Hack&Change 2026). Помощник отвечает на вопросы стажёра о процессах и
инструментах, показывает этапы адаптации и оформляет организационные действия (отгул, доступ) через
имитацию корпоративных сервисов, **не заменяя наставника**.

## Быстрый старт

```bash
cp backend/.env.example .env        # по желанию: впишите LLM_API_KEY и LLM_MODEL
docker compose up --build
```

API: <http://localhost:8080>. Миграции и тестовые данные применяются при старте.
Проверка основного сценария: `./scripts/smoke.sh` (нужны `curl` и `python3`).

Без ключа LLM запускается режим **fake**: детерминированный помощник на правилах. Он использует те же
инструменты, права и подтверждения, поэтому все сценарии работают и проверяются без внешних вызовов.

### Подключение настоящей LLM

Любой OpenAI-совместимый API (`/chat/completions` с `tools`):

| Переменная | Значение |
|---|---|
| `LLM_MODE` | `fake`, `openai` или `ml`; пусто: `openai`, если задан `LLM_API_KEY`, иначе `fake` |
| `ML_URL` | адрес ML-сервиса команды (обязателен для `ml`), контракт: [`specs/ml-contract.md`](specs/ml-contract.md) |
| `LLM_BASE_URL` | по умолчанию `https://api.openai.com/v1` (подойдёт и локальный сервер, например Ollama: `http://host:11434/v1`) |
| `LLM_MODEL` | имя модели (обязательно для `openai`) |
| `LLM_API_KEY` | ключ (обязателен для `openai`), в репозитории не хранится |
| `LLM_TIMEOUT` | таймаут запроса, по умолчанию `60s` |

Остальные переменные описаны в `.env.example`.

Режим `ml` подключает собственный ML-сервис: бэкенд сам ищет по базе знаний (с правами пользователя) и передаёт
фрагменты в запросе, сервис отвечает текстом или вызовом инструмента, а проверка прав, подтверждение и выполнение остаются
на бэкенде. Попробовать без настоящего сервиса: `python3 scripts/fake-ml.py` (референсная реализация контракта).

### Корпоративные сервисы (`mock-hr`)

Отгулы и доступы оформляются во внешнем сервисе. В прототипе его имитирует отдельный HTTP-сервис `mock-hr`
(`cmd/mockhr`, порт 8081, поднимается в compose), бэкенд знает только `HR_URL`. Запросы идемпотентны
(`Idempotency-Key` = id действия), сервис недоступен → действие получает статус `failed` без технических деталей для
пользователя. Если `HR_URL` пуст, имитация работает внутри процесса бэкенда (удобно для разработки без compose).
Для боевого подключения заменяется только URL и аутентификация клиента `internal/clients/hr`.

## Тестовые учётные данные

Авторизация имитирует корпоративную (`Authorization: Bearer <token>`). Все данные синтетические.

| Роль | Токен | Кто |
|---|---|---|
| intern | `dev-intern-1` | Анна Стажёр (intern1) |
| intern | `dev-intern-2` | Борис Стажёр (intern2) |
| mentor | `dev-mentor` | Михаил Наставник |
| manager | `dev-manager` | Мария Руководитель |
| hr_bp | `dev-hr` | Ольга HR |

## Порядок проверки (сценарии)

1. **Отгул.** `POST /api/chat` «Как оформить отгул?» → ответ из базы знаний. «Оформи отгул на 2030-01-10» →
   карточка подтверждения (`pending_actions`). `POST /api/actions/{id}/confirm` → заявка в имитации сервиса
   отсутствий, результат сохранён. Руководитель одобряет: `POST /api/manager/requests/{id}/decision`.
   «Какой статус моей заявки на отгул?» → «одобрена».
2. **Доступ.** «Нужен доступ к billing-service на запись» → карточка с согласующим по уровню прав и напоминанием
   сверить с наставником.
3. **Этапы адаптации.** `GET /api/progress`; `POST /api/progress/steps/{id}/status`. Шаги, которые подтверждает
   наставник, стажёр переводит в `waiting`, наставник решает через
   `POST /api/mentor/interns/{internId}/steps/{stepId}/decision`.
4. **Права.** HR не получает инструмент отгула; стажёр не видит памятку руководителя и документы другой команды;
   чужие действия и диалоги возвращают 404; метрики доступны только manager и hr_bp.
5. **Метрики.** `GET /api/metrics?days=30` (manager, hr_bp).

## API

| Метод и путь | Роли | Назначение |
|---|---|---|
| `GET /healthz`, `GET /readyz` | все | проверки живости и готовности |
| `GET /api/me` | любая | текущий пользователь |
| `POST /api/chat` `{message, conversation_id?, request_id?}` | любая | сообщение помощнику |
| `GET /api/conversations/{id}/messages` | владелец | история диалога |
| `GET /api/progress` | intern | этапы, статусы, следующий шаг |
| `POST /api/progress/steps/{id}/status` `{status}` | intern | сменить статус своего шага |
| `GET /api/actions`, `POST /api/actions/{id}/confirm`, `.../cancel` | владелец | действия и подтверждение |
| `GET /api/requests` | любая | мои заявки в корпоративных сервисах (имитация) |
| `GET /api/mentor/confirmations`, `POST /api/mentor/interns/{i}/steps/{s}/decision` | mentor | подтверждение шагов |
| `GET /api/manager/requests`, `POST /api/manager/requests/{id}/decision` | manager | согласование заявок |
| `GET /api/metrics?days=` | manager, hr_bp | метрики пользы |

Ошибки: `{"error": "..."}` с кодами 400, 401, 403, 404, 409, 502. Внутренние детали клиенту не отдаются.

## Архитектура

```
HTTP (net/http; middleware: CORS, logging, recover, auth, RBAC)
  └─ chat.Service ── llm.Client (OpenAI-compatible | ML-сервис команды | fake)
       └─ tools.Registry ── Policy (RBAC) ── ActionStore (action_log: подтверждение, идемпотентность)
            ├─ KB (PostgreSQL FTS, права в SQL)  ├─ People  ├─ Steps/progress
            └─ RequestService ── clients/hr ──HTTP──▶ mock-hr (имитация сервисов отсутствий и доступов)
PostgreSQL: users/teams, чек-лист, диалоги, action_log, kb_*, events, mock_* (mock-hr использует те же таблицы)
```

- Бэкенд на Go + PostgreSQL, как в стеке Авито.
- Имитируется то, что в Авито уже есть: авторизация (`mock_auth_tokens`), ролевая модель (`auth.Policy`),
  сервисы заявок (`mock-hr`). Для боевого подключения достаточно реализовать интерфейсы
  `auth.Authenticator` и `tools.RequestService` (RPC или MCP), остальной код не меняется.
- Действия оформляются инструментами агента с подтверждением; их можно вынести в MCP-сервер без изменения контракта.
- События (`events`) пригодны для выгрузки в Clickstream и Grafana.
- Модель не получает данных, недоступных роли: фильтрация выполняется до обращения к LLM.

Спецификация поведения агента: [`specs/agent-tools/spec.md`](specs/agent-tools/spec.md).

## Тесты и покрытие

Интеграционным тестам нужен Postgres (для каждого теста создаётся отдельная база):

```bash
docker compose up -d db
cd backend
TEST_DATABASE_URL='postgres://app:app@localhost:5432/postgres?sslmode=disable' \
  go test -race -p 1 -count=1 -coverpkg=./internal/... -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | tail -1      # итоговое покрытие
go tool cover -html=coverage.out -o coverage.html
```

Без `TEST_DATABASE_URL` тесты с БД пропускаются. CI (`.github/workflows/ci.yaml`) поднимает Postgres, запускает
тесты с `-race`, публикует отчёт и **падает при покрытии ниже 80%**. Покрытие считается по `internal/` (код команды);
`cmd/` содержит только запуск и остановку сервера, `internal/testdb` это тестовая инфраструктура.

Требуется Go 1.26 (см. `backend/go.mod`); зависимости: pgx v5, google/uuid, godotenv.

## Ограничения прототипа

Веб-интерфейс находится в `frontend/`: запуск и проверки описаны в [`frontend/README.md`](frontend/README.md).
Режим `openai` проверен тестами с
имитацией сервера, но не на живой модели; токены демо-авторизации хранятся открыто; нет ограничения частоты запросов.
