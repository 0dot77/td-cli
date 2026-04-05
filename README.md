# td-cli

TouchDesigner CLI for Claude Code. Control TouchDesigner with natural language via Claude Code.

## Architecture

```
Claude Code → td-cli (Go binary) → HTTP → TD Web Server DAT (Python) → td module
```

## Quick Start

### 1. Install CLI

```bash
go install github.com/td-cli/td-cli/cmd/td-cli@latest
```

Or download from [Releases](https://github.com/td-cli/td-cli/releases).

### 2. Install TD Server

In your TouchDesigner project, set up the `TDCliServer` component. See [td/setup_instructions.md](td/setup_instructions.md) for detailed steps.

### 3. Verify

```bash
td-cli status
```

### 4. Claude Code Integration

```bash
td-cli init    # Creates CLAUDE.md in current directory
```

Now Claude Code can control TouchDesigner via natural language.

## Commands

| Command | Description |
|---------|-------------|
| `td-cli status` | Check TD connection |
| `td-cli instances` | List running TD instances |
| `td-cli exec "<code>"` | Execute Python in TD |
| `td-cli ops list [path]` | List operators |
| `td-cli ops create <type> <parent>` | Create operator |
| `td-cli ops delete <path>` | Delete operator |
| `td-cli ops info <path>` | Operator details |
| `td-cli par get <op> [names]` | Get parameters |
| `td-cli par set <op> <name> <val>` | Set parameters |
| `td-cli connect <src> <dst>` | Connect operators |
| `td-cli disconnect <src> <dst>` | Disconnect operators |
| `td-cli dat read <path>` | Read DAT content |
| `td-cli dat write <path> <content>` | Write DAT content |
| `td-cli screenshot [path] -o file` | Capture TOP as PNG |
| `td-cli project info` | Project metadata |
| `td-cli project save` | Save project |

## Global Flags

- `--port <N>` — Connect to specific port (default: auto-discover)
- `--project <path>` — Target specific TD project
- `--json` — Output raw JSON
- `--timeout <ms>` — Request timeout (default: 30000)

## How It Works

1. **TD-side:** A Web Server DAT inside TouchDesigner listens on port 9500
2. **Heartbeat:** A Timer CHOP writes instance info to `~/.td-cli/instances/` every 1s
3. **CLI-side:** The Go binary auto-discovers running instances and sends HTTP requests
4. **Claude Code:** Reads `CLAUDE.md` and uses `td-cli` commands via shell

## Development

```bash
go build -o td-cli.exe ./cmd/td-cli/
```

## License

MIT
