# medincident-zitadel-actions

HTTP-шлюз между Zitadel Actions v2 и доменными микросервисами через NATS JetStream.

Каждый Zitadel webhook пересылается 1-to-1 в JetStream, обёрнутый в
`zitadel.events.v1.Envelope`. Никакого splitting, никакой трансформации
значений — событие публикуется на своём subject'е со всеми полями,
которые прислал Zitadel.

## Actions v2 Targets

| Endpoint | Zitadel Event | NATS Subject |
|----------|---------------|--------------|
| `POST /events/user/human/added` | `user.human.added` | `zitadel.users.v1.human.added` |
| `POST /events/user/human/profile/changed` | `user.human.profile.changed` | `zitadel.users.v1.human.profile.changed` |
| `POST /events/user/human/email/changed` | `user.human.email.changed` | `zitadel.users.v1.human.email.changed` |
| `POST /events/user/human/email/verified` | `user.human.email.verified` | `zitadel.users.v1.human.email.verified` |
| `POST /events/session/added` | `session.added` | `zitadel.sessions.v1.added` |
| `POST /events/session/user/checked` | `session.user.checked` | `zitadel.sessions.v1.user.checked` |
| `POST /debug` | любой | — (логирование raw body) |
| `GET /health` | — | — (проверка здоровья) |

`user.human.profile.changed` публикует одно сообщение с `FieldMask`
внутри payload'а (`UserHumanProfileChanged.updated_fields`) — маска
перечисляет только те поля, которые реально пришли в webhook от Zitadel.

## JetStream stream (операторская зона)

Сервис **не создаёт** JetStream-стрим — операторы провижинят его
заранее. Рекомендованная конфигурация:

| Параметр | Значение |
|---|---|
| Name | `zitadel` |
| Subjects | `zitadel.>` |
| Duplicates window | `24h` |

Publisher выставляет `Nats-Msg-Id` заголовок формата
`{instance_id}:{aggregate_type}:{aggregate_id}:{sequence}`, и
JetStream использует его для дедупликации ретраев в пределах окна
`duplicates`.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for setup instructions, pre-commit hooks, and available tasks.
