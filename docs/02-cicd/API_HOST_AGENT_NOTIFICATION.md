# CI/CD API Addendum

## Git and Logs

| Method | Path | Authentication | Purpose |
| --- | --- | --- | --- |
| POST | `/api/v1/webhooks/git?source_id={id}` | GitLab HMAC secret | Branch filtering and delivery-ledger idempotency |
| GET | `/api/v1/builds/{id}/logs/stream` | JWT + `build_log:view` | SSE stream with `Last-Event-ID` cursor |

The webhook secret is a dedicated encrypted `git_webhook_secret`; it is never
the repository access token. A duplicate delivery reuses the original build.
The SSE endpoint emits a terminal `complete` event and then closes.

## Notification Subscriptions

| Method | Path | Permission |
| --- | --- | --- |
| GET | `/api/v1/notification-subscriptions?project_id={id}` | `notification:view` |
| POST | `/api/v1/projects/{id}/notification-subscriptions` | `notification:create` + `credential:view` |
| DELETE | `/api/v1/notification-subscriptions/{id}?project_id={id}` | `notification:delete` |
| GET | `/api/v1/notification-deliveries?project_id={id}&status={status}&page=1&page_size=20` | `notification:view` |

投递记录接口按项目授权范围过滤，`status` 仅接受 `pending`、`processing`、`succeeded`、`dead`，单页最多 100 条。响应包含订阅名称、渠道、事件、尝试次数、下次重试时间和失败原因，不返回消息 payload、Webhook 地址、SMTP 密码或任何凭据数据。

Channels are `feishu`, `dingtalk`, `wecom`, `email`, and generic `webhook`.
Terminal workflow/build events enqueue durable outbox rows in the same DB
transaction as the state transition. Delivery retries with exponential delay
and becomes `dead` after the configured attempt limit.

Webhook credential JSON:

```json
{"url":"https://hooks.example.com/axiops"}
```

SMTP credential JSON:

```json
{"host":"smtp.example.com","port":465,"username":"axiops","password":"<secret>","from":"axiops@example.com"}
```

Credentials are AES-GCM ciphertext in the credential table and are decrypted
only inside the delivery worker. Plaintext credentials never enter payloads or
logs.
