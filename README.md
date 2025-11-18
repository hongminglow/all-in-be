# ALL-IN Backend

Production-ready skeleton for deploying a Go API on Render with Neon Postgres + Neon Stack Auth (Stack Auth) for identity.

## Project layout

```
cmd/server              # app entrypoint
internal/config         # env loading + validation
internal/http/handlers  # health + auth HTTP handlers
internal/neonauth       # JWKS-backed token verification
internal/server         # http.Server wiring + middleware
internal/storage        # storage interfaces
internal/storage/postgres # pgx-based implementation
```

## Environment variables

| Key                                 | Description                                                                                                                 |
| ----------------------------------- | --------------------------------------------------------------------------------------------------------------------------- |
| `PORT`                              | HTTP port (Render sets this automatically).                                                                                 |
| `DATABASE_URL`                      | Neon Postgres connection string (required).                                                                                 |
| `NEON_PROJECT_ID`                   | Optional metadata for downstream tooling.                                                                                   |
| `NEON_STACK_AUTH_PROJECT_ID`        | Stack Auth project identifier.                                                                                              |
| `NEON_STACK_PUBLISHABLE_CLIENT_KEY` | Public key for clients calling Stack Auth.                                                                                  |
| `NEON_STACK_SECRET_SERVER_KEY`      | Server-side API key if you later need to call Stack Auth admin endpoints.                                                   |
| `NEON_JWKS_URL`                     | JWKS endpoint used to verify Stack Auth JWTs (required for `/register` + `/login`).                                         |
| `ALLOW_DEV_AUTH`                    | Optional flag (`true`/`false`). When `true`, you can send `X-Dev-Auth-Subject` instead of a bearer token for local testing. |

> ⚠️ Your `.env` currently truncates `DATABASE_URL` (the string ends after `sslmode=`). Copy the full connection string from Neon to avoid startup failures.

## Endpoints

| Method | Path        | Auth?              | Description                                                                                     |
| ------ | ----------- | ------------------ | ----------------------------------------------------------------------------------------------- |
| GET    | `/health`   | No                 | Returns uptime + status.                                                                        |
| POST   | `/register` | Yes (Bearer token) | Verifies the Stack Auth token, then stores username/email/phone + Stack Auth `sub` in Postgres. |
| POST   | `/login`    | Yes (Bearer token) | Verifies the token and fetches the user profile tied to the `sub`.                              |

### Sample requests

```bash
# Replace $TOKEN with the Stack Auth JWT issued after signup/signin.

curl -X GET http://localhost:8080/health

curl -X POST http://localhost:8080/register \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"username":"alex","email":"alex@example.com","phone":"+155555501"}'

curl -X POST http://localhost:8080/login \
  -H "Authorization: Bearer $TOKEN"
```

When `ALLOW_DEV_AUTH=true`, replace the `Authorization` header with `X-Dev-Auth-Subject: local-user-123` (or add `?dev_subject=local-user-123` to the URL) so you can exercise the APIs without minting Stack Auth tokens. Keep this disabled in production.

## Local development

1. Export required env vars (or use an `.env` file + direnv). During local testing you can set `ALLOW_DEV_AUTH=true`.
2. Run the server:

```bash
go run ./cmd/server
```

## Render deployment

1. Push to GitHub and create a **Render Web Service**.
2. Build command: `go build -o app ./cmd/server`.
3. Start command: `./app`.
4. Add the environment variables from the table above (especially `DATABASE_URL` + `NEON_JWKS_URL`).
5. Deploy. Render injects `PORT`, so no extra config is required.

## How Neon Auth fits in

- Stack Auth manages credential flows and issues JWTs.
- This service only trusts requests that include a valid Stack Auth bearer token (validated against `NEON_JWKS_URL`).
- `/register` expects that the caller already completed the Stack Auth signup. It stores profile metadata (username/email/phone + Stack Auth `sub`) inside Neon Postgres.
- `/login` simply validates the token and returns the stored profile, making it trivial to extend with session issuance or downstream services.

### Extending further

1. Use the `NEON_STACK_SECRET_SERVER_KEY` to call Stack Auth management APIs (e.g., revoke, force reset).
2. Add additional tables (roles, audit logs) in `internal/storage/postgres` and expose them via new handlers.
3. Introduce structured logging/metrics (Zap, OpenTelemetry) and automated tests with `httptest`.
4. Add a migration tool (Atlas, golang-migrate) once schemas grow.
