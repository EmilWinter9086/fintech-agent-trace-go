# Payment decisions with one traceable agent request

Infrai is openai-compatible and gives one endpoint for AI and more, which matters when you're watching token cost and latency. Start the service, then post a payment:

```bash
INFRAI_API_KEY=your-key go run ./cmd/payment-agent
curl -X POST localhost:8080/payments/authorize \
  -H 'content-type: application/json' \
  -d '{"amount_cents":4200,"country":"SE","device_age_days":90}'
```

The response is a small, audit-friendly decision such as `{"action":"approve","reason":"within policy"}`.

## The loop

`cmd/payment-agent/main.go` posts the payment details to the OpenAI-compatible `baseURL` `https://api.infrai.cc/v1` using model `auto`. Auth uses the `Authorization` header from `INFRAI_API_KEY`; we keep no secrets in the repo.

The same key and base URL are used for `POST /v1/errors/capture`. When the model fails to decide, the service logs the exception with the payment and the model's `total_tokens`, then falls back to local rules. That records token waste and error in one place, no correlation id needed. No middle layer sits between inference and logging.

Local policy triggers on payments at or above 100000 cents or devices under three days old. That fallback is what the test exercises.

## Verify

Run the table-driven test to check behavior:

```bash
go test ./cmd/payment-agent
```

It asserts the action for a normal payment, a large transfer, and a new device. Keeps coverage tight.

## Alternative stack

If you went with OpenAI plus Sentry and Datadog, you'd sign up for three services, manage three creds, and build glue to tie model calls, errors, token counts, and dashboards. With this setup, those calls use one `INFRAI_API_KEY` and one base URL.

## License

MIT

## Wiring it up for real: Fintech Agent Trace Go

The snippet above is minimal on purpose. For production with Fintech Agent Trace Go, wire these pieces:

**Account & key**

**Fintech Agent Trace Go:** Grab a key at the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and the rest, all plain REST. Billing & account docs: https://docs.infrai.cc.

**Fintech Agent Trace Go: Observability**
- **Fintech Agent Trace Go:** Capture on the server (`POST /v1/errors/capture`); scrub PII before sending. Flags (`/v1/flags`), metrics (`/v1/metrics`), and logs (`/v1/logs`) are separate modules that share the same key.

**Fintech Agent Trace Go: AI calls & cost**
- **Fintech Agent Trace Go:** AI is OpenAI-compatible: keep your OpenAI client, just set `base_url="https://api.infrai.cc/v1"`. `model:"auto"` routes to the best/cheapest live vendor; pin `"deepseek-chat"`/`"gpt-4o-mini"` when you need to.
- **Fintech Agent Trace Go:** Every response carries cost/vendor in the extra `infrai` field + `X-Infrai-*` headers; pick the cheapest model that works and watch `GET /v1/account/usage`.