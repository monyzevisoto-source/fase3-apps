# Repository Guidelines

## Project Structure & Module Organization

This repository contains the `flag-service`, a Flask CRUD API for ToggleMaster feature flag definitions.

- `app.py`: main Flask application, authentication middleware, PostgreSQL pool, and `/flags` endpoints.
- `db/init.sql`: database schema and trigger setup for the `flags` table.
- `k8s/deployment.yaml`: Kubernetes deployment manifest for the service container.
- `Dockerfile`: production image build using Python 3.12 slim.
- `requirements.txt`: pinned Python runtime dependencies.

There is currently no dedicated `tests/` directory or static asset directory.

## Build, Test, and Development Commands

- `pip install -r requirements.txt`: install Flask, Gunicorn, PostgreSQL, dotenv, and HTTP client dependencies.
- `psql -U <user> -d <db> -f db/init.sql`: initialize or refresh the local PostgreSQL schema.
- `gunicorn --bind 0.0.0.0:8002 app:app`: run the service locally in the same style documented in `README.md`.
- `python app.py`: run the Flask development entrypoint using `PORT` or default port `8002`.
- `docker build -t flag-service .`: build the container image.

Before running locally, define `DATABASE_URL`, `AUTH_SERVICE_URL`, and optionally `PORT` in `.env`. The `auth-service` must be reachable for all routes except `/health`.

## Coding Style & Naming Conventions

Use Python 3.9+ compatible code, four-space indentation, and clear snake_case names for functions, variables, and database fields. Keep route handlers small and explicit. Use parameterized SQL with psycopg2 placeholders (`%s`) for user-provided values. Preserve the current JSON response style, including Portuguese error messages where the surrounding endpoint already uses them.

## Testing Guidelines

No automated tests are currently checked in. When adding tests, prefer `pytest` and place files under `tests/` with names like `test_flags.py`. Cover `/health`, authentication failures, CRUD success paths, duplicate flag handling, and database error handling. For manual checks, use the `curl` examples in `README.md` with a valid API key from `auth-service`.

## Commit & Pull Request Guidelines

The current Git history uses Conventional Commit style, for example `feat: first deploy`. Continue with short, imperative messages such as `fix: handle auth timeout` or `docs: update local setup`.

Pull requests should include a concise description, any required environment or database changes, manual or automated test results, and linked issues when applicable. Include screenshots only for changes that affect deployed dashboards or visible operational tooling.

## Security & Configuration Tips

Do not commit `.env` files, database credentials, API keys, or ECR credentials. Keep Kubernetes manifests free of secrets; use cluster secret management for sensitive values.
