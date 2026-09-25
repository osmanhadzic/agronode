# Contributing to AgroNode

Thank you for contributing to AgroNode.

AgroNode is an IoT platform for agricultural monitoring and automation, built around ESP32 devices, MQTT, Go, PostgreSQL, and a React/TypeScript dashboard.

## Development Workflow

1. Create an issue describing the problem or feature.
2. Create a branch from `main`.
3. Implement the change.
4. Add or update tests where applicable.
5. Run formatting, tests, and validation locally.
6. Open a Pull Request against `main`.

## Branch Naming

Use descriptive branch names:

```text
feature/device-registration
feature/irrigation-rules
feature/mqtt-authentication
fix/device-telemetry
fix/mqtt-connection
refactor/device-service
docs/device-api
```

## Commit Messages

Use clear and concise commit messages.

Preferred format:

```text
feat: add device registration
fix: handle MQTT reconnect
refactor: extract device service
docs: update MQTT configuration
test: add device registry tests
```

Keep commits focused on one logical change.

## Pull Requests

A Pull Request should:

* Have a clear title.
* Explain what was changed and why.
* Reference the related issue.
* Include tests for new backend functionality where appropriate.
* Avoid unrelated changes.
* Include migration information when the database schema changes.

Example:

```text
## What changed

Added device registration using MQTT credentials and device metadata.

## Why

Devices need to be registered before they can publish telemetry.

## Testing

- Added device registration tests
- Tested MQTT authentication locally
- Tested PostgreSQL migration

## Related Issue

Closes #123
```

## Go Backend

Follow standard Go conventions.

Before opening a PR:

```bash
gofmt -w .
go test ./...
go vet ./...
```

Keep business logic separated from HTTP handlers, database models, and infrastructure code.

Prefer small, testable services over large handlers.

## MQTT

MQTT-related changes require additional care because MQTT is part of the device communication and security layer.

When modifying MQTT functionality:

* Do not hard-code credentials.
* Do not commit private keys or certificates.
* Validate device identity before accepting protected operations.
* Keep MQTT topics consistent with the established AgroNode topic structure.
* Handle reconnects and connection failures explicitly.
* Consider MQTT QoS and retained-message behavior when changing telemetry flows.

Example topic:

```text
agronode/{deviceId}/telemetry
```

Security-sensitive MQTT changes should include tests covering unauthorized devices.

## Device Registration

Device registration should maintain a clear separation between:

1. Device identity
2. Device credentials
3. Device metadata
4. Device authorization
5. Device telemetry

Do not expose private credentials or private keys through API responses or logs.

## Database Changes

Database schema changes must include a migration.

Do not modify an existing production migration after it has been applied.

Example:

```bash
go run ./cmd/migrate create add_device_credentials
```

The exact migration command may differ depending on the migration tooling used by the project.

Database changes should include:

* Migration
* Updated models
* Required indexes
* Foreign keys where appropriate
* Tests for important constraints

## Frontend

The frontend uses React and TypeScript.

Before opening a PR:

```bash
npm run lint
npm run build
```

Keep API communication separate from UI components where practical.

Avoid putting business logic directly into large React components.

## Testing

New functionality should include appropriate tests.

For backend changes, prefer:

* Unit tests for business logic
* Integration tests for database functionality
* MQTT integration tests for device communication
* API tests for HTTP endpoints

For bug fixes, add a regression test when practical.

## Configuration and Secrets

Never commit:

* Passwords
* API keys
* MQTT credentials
* TLS private keys
* Certificates containing private material
* Production database credentials
* `.env` files containing secrets

Use environment variables or local configuration files that are excluded from Git.

Example:

```text
.env
.env.local
*.key
*.pem
```

Check the repository's `.gitignore` before committing configuration files.

## Code Quality

Prefer:

* Small functions
* Clear names
* Explicit error handling
* Strong typing
* Simple abstractions
* Minimal dependencies

Avoid:

* Unnecessary abstractions
* Large unrelated refactors
* Copy-pasted logic
* Ignoring errors
* Logging sensitive information

## Issue Guidelines

A useful issue should contain:

### Problem

What is wrong or missing?

### Expected Behavior

What should happen?

### Current Behavior

What currently happens?

### Reproduction

How can the problem be reproduced?

### Environment

Include relevant versions such as:

```text
Go:
PostgreSQL:
MQTT broker:
Node.js:
Docker:
OS:
```

## Security Issues

Do not report security vulnerabilities through public GitHub issues.

Use the repository's private security reporting mechanism when available.

Security issues involving:

* Device authentication
* MQTT authorization
* TLS certificates
* Credentials
* API authentication
* Tenant isolation
* Production infrastructure

should be treated as security-sensitive changes.

## Review Expectations

Reviewers should focus on:

* Correctness
* Security
* Reliability
* Maintainability
* Test coverage
* Database integrity
* MQTT behavior
* API compatibility

PRs should be small enough to review effectively.

## Before Opening a PR

Run the relevant checks:

```bash
gofmt -w .
go test ./...
go vet ./...
```

For frontend changes:

```bash
npm run lint
npm run build
```

Also verify that:

* No secrets are committed.
* Database migrations are included when required.
* Tests pass.
* Documentation is updated when behavior changes.
* The PR contains only related changes.
