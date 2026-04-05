# td-cli Development Guidelines

## Git Workflow
- Commit and push after every meaningful change (new feature, bug fix, refactor)
- Write concise commit messages in English
- Always push to origin/main after committing

## Project Structure
- `cmd/td-cli/` — Go CLI entry point
- `internal/` — Go packages (client, commands, discovery, docs, protocol)
- `td/` — Python scripts that run inside TouchDesigner
- `docs/` — Raw documentation data (not tracked in git)
- `internal/docs/data/` — Slim embedded JSON for offline docs

## TD-Side Code
- Python scripts in `td/` can be pushed to live TD via `td-cli dat write <path> -f <file>`
- Web Server DAT callbacks: `td/webserver_callbacks.py`
- Request handler: `td/td_cli_handler.py`
- Heartbeat: `td/heartbeat.py`

## Build
```bash
export PATH="/c/Program Files/Go/bin:$PATH"
go build -o td-cli.exe ./cmd/td-cli/
```

## Test
```bash
td-cli status
td-cli exec "return 1+1"
td-cli ops list /project1
```
