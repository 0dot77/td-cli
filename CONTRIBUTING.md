# Contributing to td-cli

Thank you for considering contributing to td-cli!

## Getting Started

1. Fork and clone the repository
2. Build: `go build -o td-cli ./cmd/td-cli/`
3. Run tests: `go test ./...`
4. Make your changes on a feature branch

## Project Layout

```
cmd/td-cli/      Go CLI entry point and command routing
internal/        Go packages (client, commands, discovery, docs, protocol)
td/              Python scripts that run inside TouchDesigner
tox/             Pre-built TDCliServer.tox component
tests/           Python tests for TD-side code
```

## Development Requirements

- Go 1.26+ (no external dependencies)
- TouchDesigner 099 for integration testing
- Python 3 for TD-side script testing

## Code Style

- Go: standard `gofmt` formatting
- Python: PEP 8 with 4-space indentation
- Keep error messages lowercase, no trailing periods
- Wrap errors with `%w` for context

## Before Submitting

- [ ] `go build ./cmd/td-cli/` compiles without errors
- [ ] `go test ./...` passes
- [ ] New commands include help text in `printUsage()`
- [ ] New TD endpoints include a route entry in `td_cli_handler.py`
- [ ] Commit messages are concise and in English

## Pull Requests

- Keep PRs focused on a single change
- Include a brief description of what changed and why
- If adding a new command, include example usage in the PR description
- Breaking changes to the HTTP protocol require a protocol version bump

## Reporting Issues

Open an issue on GitHub with:
- `td-cli version` and `td-cli doctor` output
- TouchDesigner version
- Steps to reproduce
- Expected vs actual behavior
