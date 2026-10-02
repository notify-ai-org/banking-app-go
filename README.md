<p align="center">
  <img src="assets/notify-ai-logo.svg" alt="Notify.ai" width="96" />
</p>

<h1 align="center">Notify.ai</h1>

<p align="center"><b>Banking Example Application (Go)</b> — Go SDK integration demo</p>

---

## 📖 Overview

The Go port of the Java [banking-app](https://github.com/notify-ai-org/examples/tree/main/banking-app) example. It shows how to integrate the [Notify.ai Go SDK](https://github.com/notify-ai-org/client-go) into a banking service. All four banking events, their subject suppliers, the vocabulary supplier, both rules and both callbacks are ported one-to-one.

| Event | Endpoint | Subjects | Hooks |
|---|---|---|---|
| `SUSPICIOUS_LOGIN` (priority 5, immediate) | `POST /api/banking/suspicious-login` | SMS + email to the account holder | `BEFORE` callback |
| `OTP_REQUESTED` (priority 5, immediate) | `POST /api/banking/request-otp` | SMS | — |
| `LARGE_TRANSFER` (priority 4, 09:00–17:00) | `POST /api/banking/transfer` | Email to the sender | `transfer-fraud-check` (≤ $50,000), `velocity-check` (< 5 recent transfers), vocabulary supplier, `AFTER` callback |
| `DAILY_BALANCE_SUMMARY` (priority 2, scheduled) | `POST /api/banking/daily-summary` | Email | — |

## ⚙️ How it is wired

Each Java annotation maps to a Go SDK registration call:
- **Models** (`internal/model`): payload structs carry `notify:"name" notifyDesc:"…"` tags, the Go form of `@Vocabulary`. `Account` is registered with `notify.RegisterModel`.
- **Service** (`internal/service`): `service.New` wraps each event method with `notify.Event`. Handlers call the wrapped versions (`svc.ProcessLargeTransfer`, …), the Go form of calling through a Spring proxy. It also registers the subject suppliers, vocabulary supplier, rules and callbacks.
- **HTTP** (`internal/api`): the four endpoints from the Java `BankingController`.

## 🚀 Running locally

Prerequisite: the Notify.ai control plane (`access`) running on `http://localhost:8080`.

```bash
NOTIFY_AI_CLIENT_TOKEN=client-... go run .
```

| Env var | Default |
|---|---|
| `NOTIFY_AI_ACP_SERVER_URL` | `http://localhost:8080` |
| `NOTIFY_AI_CLIENT_TOKEN` | — |
| `NOTIFY_AI_APPLICATION_NAME` | `banking-app-go` |
| `PORT` | `8091` |

Other `NOTIFY_AI_*` variables from the SDK are honored too.

```bash
curl -X POST localhost:8091/api/banking/suspicious-login \
  -d '{"userId":"ACC-1","ipAddress":"203.0.113.9","deviceId":"dev-42","location":"Lagos, NG"}'

curl -X POST localhost:8091/api/banking/transfer \
  -d '{"transactionId":"TX-1","fromAccountId":"ACC-1","toAccountId":"ACC-2","amount":75000,"currency":"USD","type":"WIRE"}'
```

Seeded accounts are `ACC-1`, `ACC-2` and `ACC-3`.

## 🧪 Tests

```bash
go test ./...
```

`internal/api/api_test.go` drives every endpoint against a fake acp-server. It checks subjects, rule results, payload flattening and model registration.

Inside the [notify-ai](https://github.com/notify-ai-org/notify-ai) superproject, the root `go.work` builds this app against the local `client-go` checkout.

---

## 👥 Developer Contact & Contributing

- **Lead Developer**: Rohan Naik ([rohan.naik07@github](https://github.com/rohan-naik07))
- **Email**: dev-support@notify.ai

Contributions are welcome: fork, branch from `main`, make sure `go vet ./...` and `go test ./...` pass, and open a pull request.
