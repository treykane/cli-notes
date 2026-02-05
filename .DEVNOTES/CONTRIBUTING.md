# Contributing

Thanks for contributing to CLI Notes. This project is early-stage and moving fast; the notes below help keep changes consistent and easy to review.

## Quick Start

See `DEVELOPMENT.md` for setup requirements and run commands.

## Development Practices

- Keep behavior changes small and focused.
- Favor clear, readable Go over clever code.
- Prefer standard library solutions when practical.
- Avoid hidden filesystem side effects; all notes live in `~/notes`.
- Keep the README, `docs/DEVELOPMENT.md`, and in-app help aligned.
- Preserve note file normalization rules (notes end with exactly one trailing newline).

## Code Style

- Follow Go conventions (`gofmt`-compatible formatting).
- Use clear, descriptive variable names.
- Keep functions cohesive; refactor if a function grows too large.
- Add succinct comments only where intent is non-obvious.

## Testing

See `DEVELOPMENT.md` for the current testing guidance.

## Submitting Changes

- Open a PR with a clear description of what changed and why.
- Include any screenshots or short recordings for UI changes.
- Call out any user-facing behavior changes explicitly.

## Issue Reporting

When filing a bug:
- Provide steps to reproduce.
- Include your OS, terminal, and Go version.
- Attach any relevant logs or screenshots.
