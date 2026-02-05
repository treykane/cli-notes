# Contributing

Thanks for contributing to CLI Notes. This project is early-stage and moving fast; the notes below help keep changes consistent and easy to review.

## Quick Start

Requirements:
- Go 1.21+
- A terminal with ANSI color support

Build and run:

```bash
go build -o notes ./cmd/notes
./notes
```

Run without building a binary:

```bash
go run ./cmd/notes
```

## Development Practices

- Keep behavior changes small and focused.
- Favor clear, readable Go over clever code.
- Prefer standard library solutions when practical.
- Avoid hidden filesystem side effects; all notes live in `~/notes`.
- Keep the README, `docs/DEVELOPMENT.md`, and in-app help aligned.

## Code Style

- Follow Go conventions (`gofmt`-compatible formatting).
- Use clear, descriptive variable names.
- Keep functions cohesive; refactor if a function grows too large.
- Add succinct comments only where intent is non-obvious.

## Project Structure

- `cmd/notes/main.go`: Entry point; keep minimal and wiring-focused.
- `internal/app/`: UI model, tree logic, rendering, and helpers.

## Testing

There are no formal tests yet. If you add tests, keep them fast and deterministic:

```bash
go test ./...
```

## Submitting Changes

- Open a PR with a clear description of what changed and why.
- Include any screenshots or short recordings for UI changes.
- Call out any user-facing behavior changes explicitly.

## Issue Reporting

When filing a bug:
- Provide steps to reproduce.
- Include your OS, terminal, and Go version.
- Attach any relevant logs or screenshots.
