# Go Testing Guide

This project uses both unit and integration tests. The goal is confidence, not only coverage numbers.

## Test Types in This Project

### 1) Unit Tests

- Scope: One function or one file behavior.
- Dependencies: Minimal or mocked.
- Speed: Fast.
- Command: `make test-unit`

Current examples:

- `cmd/api/main_test.go` tests environment-driven log-level resolution.
- `internal/tools/mockdb_test.go` tests mock DB behavior.
- `internal/middleware/authorization_test.go` tests authorization middleware decisions.
- `internal/handlers/get_coin_balance_test.go` tests request validation.

### 2) Integration Tests

- Scope: Multiple layers working together (router + middleware + handlers + data layer).
- Dependencies: Real in-process wiring with `httptest.Server`.
- Speed: Slower than unit tests.
- Command: `make test-integration`

Current example:

- `internal/handlers/handlers_integration_test.go` validates URL path, auth middleware, JSON payload, and update flow.

## How to Run Tests

- Unit + integration: `make test`
- Unit only: `make test-unit`
- Integration only: `make test-integration`

## How to Think About "Every Line"

Trying to assert every line directly is usually brittle. Instead:

1. Assert every branch that matters (success, validation failure, downstream error).
2. Assert contract outputs (HTTP status, JSON body, side effects).
3. Cover critical paths end-to-end with integration tests.

This gives practical confidence even when internal implementation changes.

## Test Scenarios You Should Add Next

### Handler Scenarios

- `GET /account/coins` without auth header -> `400`.
- `GET /account/coins` with unknown user -> `500` or expected error code decision.
- `PUT /account/coins` with malformed JSON -> `400`.
- `PUT /account/coins` with negative balance -> `400`.

### Middleware Scenarios

- Missing username query -> request rejected.
- Mismatched token -> request rejected.
- Valid token -> next handler called.

### Data Layer Scenarios

- Update unknown user -> error.
- Read after update returns updated value.
- Concurrent updates (goroutines) keep data valid.

### Configuration Scenarios

- `APP_ENV=development` defaults to debug.
- `APP_ENV=production` defaults to info.
- `LOG_LEVEL` overrides environment default.

## Integration Test Pattern (URL Path Testing)

1. Create a Chi router with your production route wiring.
2. Wrap it with `httptest.NewServer`.
3. Call real URLs like `/account/coins?username=alex`.
4. Assert status code and JSON payload.

This validates real path matching and middleware chain, not only direct function calls.

## Coverage Command

Use this when you want a coverage report:

```bash
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

## When to Prefer Unit vs Integration

- Prefer unit tests for logic-heavy functions and validation branches.
- Prefer integration tests for routing, middleware ordering, and handler contracts.
- Keep a healthy mix: many unit tests + a smaller set of high-value integration tests.
