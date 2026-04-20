# Task Service

Небольшой backend-сервис на Go для управления задачами.

Сервис поддерживает:

- обычные задачи
- шаблоны периодических задач
- автоматическую генерацию задач по шаблонам

По смыслу это API модуля трекера задач внутри медицинской информационной системы: сотрудники могут создавать себе рабочие задачи вручную или настраивать их повторение по расписанию.

## Что умеет сервис

### Обычные задачи

CRUD для задач:

- `POST /api/v1/tasks`
- `GET /api/v1/tasks`
- `GET /api/v1/tasks/{id}`
- `PUT /api/v1/tasks/{id}`
- `DELETE /api/v1/tasks/{id}`

### Шаблоны периодических задач

CRUD для recurring task templates:

- `POST /api/v1/task-templates`
- `GET /api/v1/task-templates`
- `GET /api/v1/task-templates/{id}`
- `PUT /api/v1/task-templates/{id}`
- `DELETE /api/v1/task-templates/{id}`

Поддерживаемые типы периодичности:

- `daily` — каждый `n`-й день
- `monthly` — указанное число месяца
- `specific_dates` — конкретные даты
- `even_odd` — чётные или нечётные дни месяца
- `weekdays` — повторяющиеся дни недели

## Как это работает

В системе есть две разные сущности:

- `task instance` — конкретная рабочая задача
- `recurring task template` — правило, по которому автоматически создаются задачи

При создании или обновлении шаблона сервис:

1. валидирует шаблон;
2. сохраняет его в БД;
3. генерирует задачи на окно вперёд.

Дополнительно работает фоновый sync, который догенерирует недостающие задачи по активным шаблонам.

## Важные архитектурные решения

- Шаблоны вынесены отдельно от `tasks`, потому что правило генерации и конкретная задача — это разные сущности с разной жизнью.
- `timezone` хранится на шаблоне, потому что периодичность считается по календарной дате шаблона.
- При update шаблона пересоздаются только будущие `new` generated task instances, чтобы не терять уже начатую или завершённую работу.
- `DELETE /api/v1/task-templates/{id}` — это deactivate, а не hard delete:
  - новые generated task instances больше не создаются;
  - будущие generated task instances в статусе `new` удаляются;
  - прошлые, текущие и уже начатые/завершённые экземпляры остаются.
- `scheduled_for` не обязателен для ручной задачи; это поле используется в первую очередь для generated task instances.

## Как запустить

### Требования

- Go `1.23+`
- Docker
- Docker Compose

### Запуск через Docker Compose

```bash
docker compose down -v
docker compose up --build
```

После запуска сервис будет доступен по адресу:

```text
http://localhost:8080
```

Важно: каталог `migrations/` монтируется в `docker-entrypoint-initdb.d`, поэтому при изменении схемы лучше пересоздавать volume.

## Swagger

Swagger UI:

```text
http://localhost:8080/swagger/
```

OpenAPI JSON:

```text
http://localhost:8080/swagger/openapi.json
```

## Пример запроса

Пример создания recurring task template:

```http
POST /api/v1/task-templates
Content-Type: application/json
```

```json
{
  "title": "Обход пациентов",
  "description": "Утренний обход отделения",
  "timezone": "Europe/Moscow",
  "start_date": "2026-04-20",
  "recurrence_type": "weekdays",
  "recurrence_settings": {
    "weekdays": ["monday", "wednesday", "friday"]
  }
}
```

## Коротко про хранение данных

- generated tasks получают `template_id` и `scheduled_for`
- ручные задачи могут существовать без `scheduled_for`
- update шаблона пересобирает только будущие `new` экземпляры
- delete шаблона деактивирует правило и очищает только будущие `new` экземпляры
