# medincident-zitadel-actions

HTTP-шлюз между Zitadel Actions v2 и доменными микросервисами через NATS JetStream.

## Actions v2 Targets

| Endpoint | Zitadel Event | NATS Subject | Описание |
|----------|---------------|--------------|----------|
| `POST /events/user/human/added` | `user.human.added` | `medincident.users.v1.created` | Создание пользователя |
| `POST /events/user/human/profile/changed` | `user.human.profile.changed` | `medincident.users.v1.name_changed` | Изменение имени (firstName, lastName, displayName, nickName) |
| `POST /events/user/human/profile/changed` | `user.human.profile.changed` | `medincident.users.v1.language_changed` | Изменение языка |
| `POST /events/user/human/profile/changed` | `user.human.profile.changed` | `medincident.users.v1.gender_changed` | Изменение пола |
| `POST /debug` | любой | — | Логирование raw body (отладка) |
| `GET /health` | — | — | Проверка здоровья сервиса |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for setup instructions, pre-commit hooks, and available tasks.
