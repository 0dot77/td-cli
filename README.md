# td-cli

[English](#english) | [한국어](#한국어)

## English

TouchDesigner CLI for LLM agents and terminal-driven workflows.

`td-cli` lets you control a live TouchDesigner project from the terminal with commands such as `status`, `ops`, `par`, `dat`, and `exec`.

### What This Does

- Connect to a running TouchDesigner project over HTTP
- Discover running TD instances automatically
- Inspect and edit operators, parameters, DATs, and networks
- Generate starter guidance for agent-assisted workflows

### How It Works

```text
LLM Agent / Terminal
        |
        v
td-cli (Go binary)
        |
        v
HTTP on port 9500
        |
        v
TouchDesigner Web Server DAT + Python handler
```

The TouchDesigner side writes heartbeat files to `~/.td-cli/instances/`, and `td-cli` uses those files to auto-discover active projects.

In an agent workflow, the LLM is the reasoning layer and `td-cli` is the execution layer. The model decides what to inspect or change, and `td-cli` performs the concrete operations against a live TouchDesigner session.

Security note: if you set `TD_CLI_TOKEN` in both the shell running `td-cli` and the TouchDesigner process environment before launch, the server will require that shared token on every HTTP request.

### Beginner Install Guide

#### 1. Prerequisites

- TouchDesigner installed and able to open a project
- Windows terminal such as PowerShell
- One of:
  - the prebuilt `td-cli.exe` from GitHub Releases
  - Go `1.26.1` or newer if you want to build/install from source

#### 2. Install the CLI

Option A: Download a release binary

1. Download `td-cli.exe` from [Releases](https://github.com/td-cli/td-cli/releases).
2. Put the file somewhere easy to find, for example `C:\Tools\td-cli\td-cli.exe`.
3. Either run it with the full path, or add that folder to your `PATH`.

Example:

```powershell
C:\Tools\td-cli\td-cli.exe version
```

Option B: Install with Go

```powershell
go install github.com/td-cli/td-cli/cmd/td-cli@latest
```

If Go is configured correctly, the binary will usually be installed into your Go bin directory.

If you want to build from this repository directly:

```powershell
go build -o td-cli.exe ./cmd/td-cli/
```

#### 3. Set Up TouchDesigner Server

You must add the `TDCliServer` component to your TouchDesigner project before `td-cli` can connect.

Recommended setup:

1. Open your project in TouchDesigner.
2. Import [`tox/TDCliServer.tox`](tox/TDCliServer.tox) into the root network, or drag it into the project.
3. Make sure the imported component is named `TDCliServer`.
4. Open the component and verify that `webserver1` is active on port `9500`.

That `.tox` already contains the required operators and scripts, so this is the fastest way to get started.

Manual setup is still possible if you want to build it yourself. The Python source files used by the component are:

- [`td/webserver_callbacks.py`](td/webserver_callbacks.py)
- [`td/td_cli_handler.py`](td/td_cli_handler.py)
- [`td/heartbeat.py`](td/heartbeat.py)

Detailed setup notes are also available in [`td/setup_instructions.md`](td/setup_instructions.md).

#### 4. Verify the Connection

```powershell
td-cli status
```

Expected result:

```text
Connected to TouchDesigner
  Project:    ...
  TD Version: ...
  Server:     td-cli v...
```

If you have more than one TouchDesigner project open:

```powershell
td-cli instances
td-cli --port 9500 status
td-cli --project "C:\path\to\your\project.toe" status
```

#### 5. Bootstrap Agent Guidance

```powershell
td-cli init
```

This currently creates a `CLAUDE.md` file in the current folder with command examples and usage notes. The CLI itself is not Claude-specific: Codex and other agents can use the same commands directly, or adapt the generated guidance into their own project-instruction format such as `AGENTS.md`.

### First Commands to Try

```powershell
td-cli status
td-cli instances
td-cli ops list /project1
td-cli ops create noiseTOP /project1 --name myNoise
td-cli par get /project1/myNoise
td-cli par set /project1/myNoise period 4
td-cli dat read /project1/text1
td-cli exec "print(op('/project1').children)"
```

### Main Commands

| Command | Description |
|---------|-------------|
| `td-cli status` | Check TD connection |
| `td-cli instances` | List running TD instances |
| `td-cli exec "<code>"` | Execute Python in TD |
| `td-cli exec -f script.py` | Execute a local Python file in TD |
| `td-cli ops list [path]` | List operators |
| `td-cli ops create <type> <parent>` | Create an operator |
| `td-cli ops delete <path>` | Delete an operator |
| `td-cli ops info <path>` | Show operator details |
| `td-cli par get <op> [names]` | Get parameter values |
| `td-cli par set <op> <name> <value>` | Set one or more parameters |
| `td-cli connect <src> <dst>` | Connect operators |
| `td-cli disconnect <src> <dst>` | Disconnect operators |
| `td-cli dat read <path>` | Read DAT contents |
| `td-cli dat write <path> <content>` | Write DAT contents |
| `td-cli screenshot [path] -o file.png` | Save TOP output as PNG |
| `td-cli project info` | Show project metadata |
| `td-cli project save [path]` | Save the current project |
| `td-cli backup list [--limit N]` | List recent backup artifacts |
| `td-cli backup restore <backup-id>` | Restore a previous backup artifact |
| `td-cli logs list [--limit N]` | List recent audit log events |
| `td-cli logs tail [--limit N]` | Read recent audit log events |
| `td-cli tox export <comp> -o file.tox` | Export a COMP as `.tox` |
| `td-cli tox import <file.tox> [parent]` | Import a `.tox` file |
| `td-cli network export [path] [-o file]` | Export a network snapshot |
| `td-cli network import <file> [path]` | Import a network snapshot |
| `td-cli describe [path]` | Produce an AI-friendly network summary |
| `td-cli diff <file1> <file2>` | Compare two snapshots |
| `td-cli diff --live <file> [path]` | Compare a snapshot against live TD |
| `td-cli watch [path] [--interval ms]` | Monitor a network in real time |
| `td-cli docs` | Browse built-in offline docs |
| `td-cli docs <operator>` | Lookup an operator |
| `td-cli docs api [class]` | Lookup Python API docs |
| `td-cli shaders list` | List shader templates |
| `td-cli shaders apply <name> <glsl_top_path>` | Apply a shader template |
| `td-cli update` | Self-update from GitHub Releases |
| `td-cli version` | Show version |

### Global Flags

- `--port <N>`: connect to a specific port
- `--project <path>`: target a specific `.toe` project path
- `--json`: print raw JSON output
- `--timeout <ms>`: change request timeout, default `30000`

### Troubleshooting

`td-cli status` says no running TouchDesigner instances:

- Make sure the TouchDesigner project is open
- Make sure `webserver1` is active on port `9500`
- Make sure the heartbeat callback is running every second
- Check whether `~/.td-cli/instances/` is being updated

More than one project is running:

- Use `td-cli instances`
- Then choose one with `--port` or `--project`

Command not found:

- If you downloaded `td-cli.exe`, run it with the full path first
- If that works, add its folder to `PATH`

### Development

Build locally:

```powershell
go build -o td-cli.exe ./cmd/td-cli/
```

Show help:

```powershell
td-cli help
```

## 한국어

LLM 에이전트와 터미널 기반 워크플로를 위한 TouchDesigner CLI입니다.

`td-cli`는 TouchDesigner 프로젝트를 터미널에서 제어할 수 있게 해주는 도구입니다. `status`, `ops`, `par`, `dat`, `exec` 같은 명령으로 TouchDesigner를 다룰 수 있습니다.

### 이 프로젝트로 할 수 있는 것

- 실행 중인 TouchDesigner 프로젝트에 HTTP로 연결합니다
- 실행 중인 TD 인스턴스를 자동으로 찾습니다
- 오퍼레이터, 파라미터, DAT, 네트워크를 조회하고 수정할 수 있습니다
- 에이전트 보조 워크플로를 위한 시작 가이드를 생성합니다

### 동작 방식

```text
LLM Agent / Terminal
        |
        v
td-cli (Go binary)
        |
        v
HTTP on port 9500
        |
        v
TouchDesigner Web Server DAT + Python handler
```

TouchDesigner 쪽은 `~/.td-cli/instances/` 경로에 heartbeat 파일을 기록하고, `td-cli`는 그 파일을 읽어서 현재 실행 중인 프로젝트를 자동 탐지합니다.

에이전트 워크플로에서 LLM은 추론 레이어이고 `td-cli`는 실행 레이어입니다. 모델이 무엇을 조회하거나 수정할지 판단하고, `td-cli`가 그 결정을 실제 TouchDesigner 세션에 반영합니다.

보안 참고: `td-cli`를 실행하는 셸과 TouchDesigner 프로세스를 실행하기 전 환경에 동일한 `TD_CLI_TOKEN` 값을 설정하면, 서버가 모든 HTTP 요청에 대해 해당 공유 토큰을 요구합니다.

### 초심자용 설치 가이드

#### 1. 준비물

- TouchDesigner가 설치되어 있고 프로젝트를 열 수 있어야 합니다
- PowerShell 같은 Windows 터미널이 필요합니다
- 아래 둘 중 하나가 필요합니다
  - GitHub Releases에서 받은 미리 빌드된 `td-cli.exe`
  - 소스에서 직접 설치할 경우 Go `1.26.1` 이상

#### 2. CLI 설치

방법 A: 릴리스 바이너리 다운로드

1. [Releases](https://github.com/td-cli/td-cli/releases)에서 `td-cli.exe`를 내려받습니다.
2. 예를 들어 `C:\Tools\td-cli\td-cli.exe` 같이 찾기 쉬운 위치에 둡니다.
3. 아래 둘 중 하나로 사용합니다.
   - 전체 경로로 실행
   - 해당 폴더를 `PATH`에 추가

예시:

```powershell
C:\Tools\td-cli\td-cli.exe version
```

방법 B: Go로 설치

```powershell
go install github.com/td-cli/td-cli/cmd/td-cli@latest
```

Go 환경 설정이 정상이라면 보통 Go bin 디렉터리에 실행 파일이 설치됩니다.

이 저장소를 직접 빌드하려면:

```powershell
go build -o td-cli.exe ./cmd/td-cli/
```

#### 3. TouchDesigner 서버 설정

`td-cli`가 연결되려면 먼저 TouchDesigner 프로젝트 안에 `TDCliServer` 컴포넌트를 넣어야 합니다.

권장 방법:

1. TouchDesigner에서 프로젝트를 엽니다.
2. [`tox/TDCliServer.tox`](tox/TDCliServer.tox)를 루트 네트워크로 import 하거나 프로젝트에 드래그해서 넣습니다.
3. 가져온 컴포넌트 이름이 `TDCliServer`인지 확인합니다.
4. 컴포넌트 안으로 들어가서 `webserver1`이 포트 `9500`에서 활성화되어 있는지 확인합니다.

이 `.tox` 파일 안에는 필요한 오퍼레이터와 스크립트가 이미 들어 있으므로, 가장 빠른 셋업 방법입니다.

직접 수동 구성하고 싶다면 아래 Python 소스 파일을 참고하면 됩니다.

- [`td/webserver_callbacks.py`](td/webserver_callbacks.py)
- [`td/td_cli_handler.py`](td/td_cli_handler.py)
- [`td/heartbeat.py`](td/heartbeat.py)

더 자세한 설정 설명은 [`td/setup_instructions.md`](td/setup_instructions.md)에도 있습니다.

#### 4. 연결 확인

```powershell
td-cli status
```

기대 결과:

```text
Connected to TouchDesigner
  Project:    ...
  TD Version: ...
  Server:     td-cli v...
```

TouchDesigner 프로젝트를 여러 개 열어 둔 경우에는 명시적으로 지정해야 합니다:

```powershell
td-cli instances
td-cli --port 9500 status
td-cli --project "C:\path\to\your\project.toe" status
```

#### 5. 에이전트 가이드 시작하기

```powershell
td-cli init
```

이 명령은 현재 폴더에 `CLAUDE.md`를 만들고, 명령 예시와 사용법을 기록합니다. 현재 출력 파일 이름은 Claude 스타일이지만, CLI 자체는 Claude 전용이 아닙니다. Codex 같은 다른 에이전트도 같은 명령을 직접 사용할 수 있고, 필요하면 내용을 `AGENTS.md` 같은 형식으로 옮겨서 사용할 수 있습니다.

### 처음 해볼 명령

```powershell
td-cli status
td-cli instances
td-cli ops list /project1
td-cli ops create noiseTOP /project1 --name myNoise
td-cli par get /project1/myNoise
td-cli par set /project1/myNoise period 4
td-cli dat read /project1/text1
td-cli exec "print(op('/project1').children)"
```

### 주요 명령어

| 명령 | 설명 |
|------|------|
| `td-cli status` | TD 연결 상태 확인 |
| `td-cli instances` | 실행 중인 TD 인스턴스 목록 |
| `td-cli exec "<code>"` | TD 내부에서 Python 실행 |
| `td-cli exec -f script.py` | 로컬 Python 파일 실행 |
| `td-cli ops list [path]` | 오퍼레이터 목록 조회 |
| `td-cli ops create <type> <parent>` | 오퍼레이터 생성 |
| `td-cli ops delete <path>` | 오퍼레이터 삭제 |
| `td-cli ops info <path>` | 오퍼레이터 상세 정보 |
| `td-cli par get <op> [names]` | 파라미터 값 조회 |
| `td-cli par set <op> <name> <value>` | 파라미터 값 설정 |
| `td-cli connect <src> <dst>` | 오퍼레이터 연결 |
| `td-cli disconnect <src> <dst>` | 오퍼레이터 연결 해제 |
| `td-cli dat read <path>` | DAT 내용 읽기 |
| `td-cli dat write <path> <content>` | DAT 내용 쓰기 |
| `td-cli screenshot [path] -o file.png` | TOP 출력 PNG 저장 |
| `td-cli project info` | 프로젝트 정보 조회 |
| `td-cli project save [path]` | 프로젝트 저장 |
| `td-cli backup list [--limit N]` | 최근 백업 목록 보기 |
| `td-cli backup restore <backup-id>` | 이전 백업 복구 |
| `td-cli logs list [--limit N]` | 최근 감사 로그 보기 |
| `td-cli logs tail [--limit N]` | 최근 감사 로그 읽기 |
| `td-cli tox export <comp> -o file.tox` | COMP를 `.tox`로 내보내기 |
| `td-cli tox import <file.tox> [parent]` | `.tox` 파일 가져오기 |
| `td-cli network export [path] [-o file]` | 네트워크 스냅샷 내보내기 |
| `td-cli network import <file> [path]` | 네트워크 스냅샷 가져오기 |
| `td-cli describe [path]` | AI 친화적인 네트워크 요약 생성 |
| `td-cli diff <file1> <file2>` | 두 스냅샷 비교 |
| `td-cli diff --live <file> [path]` | 스냅샷과 현재 TD 상태 비교 |
| `td-cli watch [path] [--interval ms]` | 실시간 모니터링 |
| `td-cli docs` | 내장 오프라인 문서 보기 |
| `td-cli docs <operator>` | 오퍼레이터 문서 조회 |
| `td-cli docs api [class]` | Python API 문서 조회 |
| `td-cli shaders list` | 셰이더 템플릿 목록 |
| `td-cli shaders apply <name> <glsl_top_path>` | 셰이더 템플릿 적용 |
| `td-cli update` | GitHub Releases에서 자체 업데이트 |
| `td-cli version` | 버전 표시 |

### 전역 플래그

- `--port <N>`: 특정 포트로 직접 연결합니다
- `--project <path>`: 특정 `.toe` 프로젝트를 대상으로 지정합니다
- `--json`: 결과를 JSON으로 출력합니다
- `--timeout <ms>`: 요청 타임아웃을 변경합니다. 기본값은 `30000`

### 문제 해결

`td-cli status`에서 실행 중인 TouchDesigner 인스턴스가 없다고 나오는 경우:

- TouchDesigner 프로젝트가 실제로 열려 있는지 확인하세요
- `webserver1`이 포트 `9500`에서 활성화되어 있는지 확인하세요
- heartbeat 콜백이 1초마다 실행되는지 확인하세요
- `~/.td-cli/instances/` 경로가 실제로 갱신되는지 확인하세요

프로젝트가 여러 개 실행 중인 경우:

- `td-cli instances`로 목록을 확인하세요
- 그 다음 `--port` 또는 `--project`로 원하는 프로젝트를 지정하세요

명령을 찾을 수 없다고 나오는 경우:

- `td-cli.exe`를 직접 내려받았다면 우선 전체 경로로 실행해 보세요
- 그 방식이 된다면 해당 폴더를 `PATH`에 추가하세요

### 개발

로컬 빌드:

```powershell
go build -o td-cli.exe ./cmd/td-cli/
```

도움말 보기:

```powershell
td-cli help
```

## License

MIT
