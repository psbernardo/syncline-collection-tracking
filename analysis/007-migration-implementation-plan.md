# Migration Implementation Plan

## 1. Requested Direction

The migration executable should use one configuration source and require no command-line arguments:

- Load `.env` with `github.com/joho/godotenv`.
- Use hardcoded development defaults when variables are absent.
- Run the default migration action without arguments.
- Use the same `CTS_DEV` database for development and testing for now.
- Use `.env` for both development and the current shared `CTS_DEV` test database.

Proposed configuration defaults:

```go
type Config struct {
	Host     string `env:"HOST" envDefault:"localhost"`
	Port     int    `env:"PORT" envDefault:"1433"`
	Username string `env:"USER" envDefault:"sa"`
	Password string `env:"PASSWORD" envDefault:"Admin@12345"`
	Database string `env:"DATABASE" envDefault:"CTS_DEV"`
}
```

The database default should use the agreed local database name `CTS_DEV`, not the earlier example name `freshketdev`.

## 2. Important Compatibility Detail

`godotenv.Load()` only loads `.env` values into the process environment. It does not parse `env` struct tags or apply their defaults.

To use the requested environment mapping, add an environment parser such as:

```text
github.com/caarlos0/env/v11
```

The planned load sequence is:

1. Call `godotenv.Load()`.
2. Parse the `Config` struct using the environment-tag parser.
3. Apply the tag defaults for missing values.
4. Validate the resulting configuration.
5. Build the SQL Server connection.

The selected `caarlos0/env/v11` syntax uses separate `env` and `envDefault` tags. The combined `env:"HOST,default=localhost"` syntax is not supported by that parser. Do not assume `godotenv` interprets the tags.

## 3. No-Argument Migration Flow

The default command should be:

```text
go run ./cmd/migrate
```

Recommended behavior:

1. Load `.env` and defaults.
2. Connect to SQL Server.
3. Acquire the migration lock.
4. Ensure the migration registry exists.
5. Apply all pending migrations in version order.
6. Record successful versions.
7. Release the transaction-owned lock.
8. Exit with a non-zero status on failure.

No command-line argument is needed for the normal migration path. Status and rollback can remain internal/test capabilities or be exposed later through separate tooling if required.

## 4. Configuration Contract

The `.env` file should use the names expected by the struct tags:

```text
HOST=localhost
PORT=1433
USER=sa
PASSWORD=Admin@12345
DATABASE=CTS_DEV
```

Rules:

- `.env` is ignored by Git.
- `.env.example` contains safe placeholder values only.
- Existing process environment variables take precedence over `.env` values under normal `godotenv.Load()` behavior.
- The migration executable must not print the password or full connection string.
- A missing `.env` may fall back to defaults for local development.
- Malformed `.env` values must fail configuration parsing clearly.
- Production use of hardcoded credentials is prohibited.

## 5. Database and Testing Consequences

Development and testing currently share `CTS_DEV`:

- Migration verification runs against `CTS_DEV` using `.env`.
- Do not run destructive `down` migrations against `CTS_DEV` while development data exists.
- Database integration tests must not truncate, drop, or reset shared data.
- Unit tests remain database-independent.
- A separate test database is required before destructive migration tests, automated integration cleanup, or parallel development/testing.

## 6. Security Risk Requiring Explicit Acceptance

The requested default password `Admin@12345` is a credential embedded in source code. It is acceptable only as a local development fallback and must never be used for production or a network-accessible database.

Before the application is exposed beyond the local machine:

- Remove the password default from source code.
- Require `PASSWORD` from the environment or a secret manager.
- Use a least-privilege SQL login instead of `sa`.
- Rotate the local development password if it has been shared or committed.

## 7. Implementation Tasks

1. Use `github.com/caarlos0/env/v11` for the requested struct tags.
2. Load only `.env` with `godotenv.Load()`.
3. Use `CTS_DEV` as the default database name.
4. Keep `cmd/migrate` argument-free; its default action is to apply pending migrations.
5. Keep the existing versioned migration registry, transaction-owned lock, rollback, constraints, and indexes.
6. Run `go run ./cmd/migrate` after SQL Server is listening on the configured endpoint.
7. Record the migration result in `analysis/028-progress.md`.

## 8. Approval Questions

1. Is the hardcoded `Admin@12345` password explicitly limited to local development?
