# Task Service

Сервис для управления задачами с HTTP API на Go.

Теперь сервис поддерживает два уровня сущностей:

- обычные задачи в `/api/v1/tasks`
- шаблоны периодических задач в `/api/v1/task-templates`

Шаблон периодической задачи хранит один тип периодичности и автоматически создаёт обычные задачи-экземпляры на 30 дней вперёд. Генерация выполняется фоновым процессом внутри API и также запускается сразу после создания или изменения шаблона.

Текущая архитектура сознательно поддерживает только один тип периодичности на шаблон, но модель сервиса и API можно расширить до composite rules без ломки основного контракта.

## Архитектурные решения

- Шаблоны вынесены отдельно от `tasks`, потому что recurring task template и task instance — это разные сущности: шаблон описывает правило генерации, а задача — конкретный рабочий экземпляр.
- `timezone` хранится на шаблоне, потому что периодичность считается по календарной дате шаблона, и разные шаблоны могут жить в разных часовых поясах.
- При update шаблона пересоздаются только будущие generated task instances в статусе `new`, чтобы не терять уже начатую или завершённую работу.
- `DELETE /api/v1/task-templates/{id}` — это deactivate, а не hard delete: новые generated task instances больше не создаются, будущие generated task instances в статусе `new` удаляются, а прошлые, текущие и уже начатые/завершённые task instances остаются в системе.

## Поддерживаемые типы периодичности

- `daily` — каждый `n`-й день
- `monthly` — указанное число месяца с явной стратегией для коротких месяцев
- `specific_dates` — только на указанные даты
- `even_odd` — только на чётные или нечётные дни месяца
- `weekdays` — на повторяющиеся дни недели*

## Требования

- Go `1.23+`
- Docker и Docker Compose

## Быстрый запуск через Docker Compose

```bash
docker compose down -v
docker compose up --build
```

После запуска сервис будет доступен по адресу `http://localhost:8080`.

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

## API

Базовый префикс API:

```text
/api/v1
```

Основные маршруты задач:

- `POST /api/v1/tasks`
- `GET /api/v1/tasks`
- `GET /api/v1/tasks/{id}`
- `PUT /api/v1/tasks/{id}`
- `DELETE /api/v1/tasks/{id}`

Маршруты шаблонов периодических задач:

- `POST /api/v1/task-templates`
- `GET /api/v1/task-templates`
- `GET /api/v1/task-templates/{id}`
- `PUT /api/v1/task-templates/{id}`
- `DELETE /api/v1/task-templates/{id}`

## Пример шаблона

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

## Хранение и генерация

- сгенерированные задачи получают `template_id` и `scheduled_for`
- `scheduled_for` не обязателен для ручных task instances: у задач, созданных через обычный CRUD без шаблона, это поле отсутствует
- существующие вручную созданные задачи продолжают работать как раньше
- при изменении шаблона будущие экземпляры удаляются и строятся заново
- при удалении шаблона происходит deactivation: новые generated task instances больше не создаются, будущие generated task instances в статусе `new` удаляются, а прошлые, текущие и уже начатые/завершённые экземпляры остаются в системе
# test-task-for-junior-backend-developer
