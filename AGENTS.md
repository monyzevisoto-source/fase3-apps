# Repository Guidelines

## Project Structure & Module Organization

This is the ToggleMaster microservices monorepo. Each service is independently buildable and deployable:

- `auth-service/` (Go): API-key creation and validation; PostgreSQL schema in `db/init.sql`.
- `flag-service/` and `targeting-service/` (Python): protected Flask APIs; each has `db/init.sql`.
- `evaluation-service/` (Go): hot-path flag evaluation using Redis and AWS SQS.
- `analytics-service/` (Python): SQS consumer that writes analytics to DynamoDB.

Every service owns its `app.py` or Go source files, `requirements.txt` or `go.mod`, `Dockerfile`, `README.md`, and `k8s/` manifests. Keep changes scoped to the relevant service; its README documents required environment variables and dependencies. `flag-service/AGENTS.md` contains additional instructions for that service.

## Build, Test, and Development Commands

Run commands from the service directory. For Go services:

```sh
go mod download && go build ./...
go test ./...
go run .
```

For Python services, use Python 3.9+ locally (CI targets 3.12):

```sh
python -m pip install -r requirements.txt
python -m compileall .
gunicorn --bind 0.0.0.0:<port> app:app
```

Use `docker build -t <service-name> .` to validate a service image. Initialize PostgreSQL-backed services with `psql -U <user> -d <db> -f db/init.sql`.

## Coding Style & Naming Conventions

Use `gofmt` for all Go changes; use standard Go naming (`ExportedName`, `localName`) and keep packages buildable with `go build ./...`. Python uses four-space indentation, `snake_case` functions and variables, and explicit, small route handlers. Prefer parameterized SQL and never embed user input in queries. No repository-wide formatter or linter configuration is checked in; do not introduce style-only churn.

## Testing Guidelines

There are currently no checked-in test suites. Add Go tests as `*_test.go` next to the package they cover. Add Python tests under `<service>/tests/` as `test_*.py`, using `pytest`; cover health checks, authentication failures, success paths, and dependency failures. Always run the applicable build/test or compile check before submitting.

## Commit & Pull Request Guidelines

History currently has short, imperative messages (for example, `adicionando servicos`); keep commits focused and descriptive, optionally using Conventional Commit prefixes such as `fix: handle SQS retry`. PRs should state affected services, configuration or schema changes, validation performed, and linked issues. Include screenshots only for user-visible operational changes.

## Security & Configuration

Do not commit `.env` files, API keys, database URLs, AWS credentials, or generated secrets. Use Kubernetes Secrets and environment configuration for sensitive values; review `k8s/` changes carefully for exposed configuration.
