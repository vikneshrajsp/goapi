# API Reference

## Base URL

- Local: `http://localhost:8080`

## Authentication

- Header: `Authorization`
- Query parameter: `username`
- Valid mock users:
  - `alex` -> `123AL100`
  - `kevin` -> `456KV100`
  - `max` -> `789MX100`

## Endpoints

### `GET /account/coins`

- Purpose: Fetch coin balance for a user.
- Query params:
  - `username` (required)
- Headers:
  - `Authorization` (required)
- Example:

```bash
curl -s -H "Authorization: 123AL100" \
  "http://localhost:8080/account/coins?username=alex"
```

- Success response:

```json
{"code":200,"balance":100}
```

### `PUT /account/coins`

- Purpose: Update coin balance for a user.
- Query params:
  - `username` (required)
- Headers:
  - `Authorization` (required)
  - `Content-Type: application/json`
- Request body:

```json
{"balance":150}
```

- Example:

```bash
curl -s -X PUT \
  -H "Authorization: 123AL100" \
  -H "Content-Type: application/json" \
  -d '{"balance":150}' \
  "http://localhost:8080/account/coins?username=alex"
```

- Success response:

```json
{"code":200,"username":"alex","balance":150}
```

## Error Response Shape

```json
{"code":400,"message":"error details"}
```
