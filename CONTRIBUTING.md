# Contributing

## Требования

- [Go 1.26+](https://go.dev/dl/)
- [Task](https://taskfile.dev/installation/) — task runner
- [pre-commit](https://pre-commit.com/#install) — git hook manager
- [Docker](https://docs.docker.com/get-docker/) — для интеграционных тестов

## Старт

1. Склонируй репу и поставь pre-commit хуки:

```bash
git clone https://github.com/medincident/medincident-zitadel-actions.git
cd medincident-zitadel-actions
pre-commit install --hook-type pre-commit --hook-type pre-push
```

2. Проверь, что всё работает:

```bash
task test
```

## Pre-commit хуки

Проект использует [pre-commit](https://pre-commit.com/) для прогона проверок перед каждым коммитом. После установки pre-commit выполни:

```bash
pre-commit install --hook-type pre-commit --hook-type pre-push
```

Запуск всех хуков руками по всем файлам:

```bash
pre-commit run --all-files
```

Настроенные хуки:

**На коммит:**
- **Whitespace и формат** — trailing whitespace, EOF fixer, YAML validation, large files, merge conflicts
- **Форматирование protobuf** — `task fmt:check` (только при изменении `.proto` файлов)
- **Lint** — `task lint` (protobuf + Go)

**На push:**
- **Юнит-тесты** — `task test:unit`
- **Проверка уязвимостей** — `task vuln`

## Частые команды

```bash
task gen               # генерация proto и Go enum кода
task gen:check         # перегенерация и проверка, что gen/ не разъехался (гейт в CI)
task fmt               # форматирование protobuf
task fmt:check         # проверка форматирования (dry-run, без модификаций)
task lint              # lint protobuf и Go
task lint:proto        # lint только protobuf
task lint:go           # lint только Go
task vuln              # проверка уязвимостей Go
task test:unit         # юнит-тесты
task test:integration  # интеграционные тесты (требует Docker)
task test              # полный набор тестов (unit + integration)
```
