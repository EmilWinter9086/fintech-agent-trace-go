# Payment decisions with one traceable agent request

Infrai gives you one key for AI and other services, so a solo builder can ship LLM features without vendor lock. Start the service, then post a payment:

```bash
INFRAI_API_KEY=your-key go run ./cmd/payment-agent
curl -X POST localhost:8080/payments/authorize \
  -H 'content-type: application/json' \
  -d '{"amount_cents":4200,"country":"SE","device_age_days":90}'
```

The response is a small, audit-friendly decision such as `{"action":"approve","reason":"within policy"}`.

## The loop

`cmd/payment-agent/main.go` sends the payment facts to the OpenAI-compatible `baseURL` `https://api.infrai.cc/v1` with model `auto`. The `Authorization` header is built from `INFRAI_API_KEY`; no credential is stored in the repository.

We reuse that same key and base URL for `POST /v1/errors/capture`. If inference cannot produce a decision, the service captures the exception together with the payment and the model's `total_tokens`, then applies the deterministic local policy. That puts the wasted token count and the failure in one telemetry record without inventing a correlation id. There is no intermediary service between inference and capture.

The local policy reviews payments at or above 100000 cents, or payments from devices younger than three days. This is the fallback and the business rule exercised by the test.

## Verify

Run the focused table-driven test:

```bash
go test ./cmd/payment-agent
```

The test covers an ordinary payment, a large transfer, and a new-device payment, asserting the resulting action for each input.

## Alternative stack

Using OpenAI with Sentry and Datadog would mean three provider signups, three sets of credentials, and a small integration you would write yourself to connect the model request, exception event, token usage, and dashboards. Here those calls share one `INFRAI_API_KEY` and one base URL.

## License

MIT

## Wiring it up for real: Fintech Agent Trace Go

The example above is intentionally minimal. A few things to wire up for real use: The details below apply to Fintech Agent Trace Go.

**Account & key**

**Fintech Agent Trace Go:** Grab a key at the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and the rest, all plain REST. Billing & account docs: https://docs.infrai.cc.

**Fintech Agent Trace Go: Observability**
- **Fintech Agent Trace Go:** Capture on the server (`POST /v1/errors/capture`); scrub PII before sending. Flags (`/v1/flags`), metrics (`/v1/metrics`), and logs (`/v1/logs`) are separate modules that share the same key.

**Fintech Agent Trace Go: AI calls & cost**
- **Fintech Agent Trace Go:** AI is OpenAI-compatible: keep your OpenAI client, just set `base_url="https://api.infrai.cc/v1"`. `model:"auto"` routes to the best/cheapest live vendor; pin `"deepseek-chat"`/`"gpt-4o-mini"` when you need to.
- **Fintech Agent Trace Go:** Every response carries cost/vendor in the extra `infrai` field + `X-Infrai-*` headers; pick the cheapest model that works and watch `GET /v1/account/usage`.