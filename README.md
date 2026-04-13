# td-cli

TouchDesigner CLI for LLM agents, artists, and terminal-driven workflows.

`td-cli` connects a live TouchDesigner project to the terminal. It is useful in two modes at the same time:

- as a command surface for LLM agents such as Codex or Claude
- as a practical live-control tool for artists who want to inspect, build, and tweak TouchDesigner projects without clicking through every network manually

## English

`td-cli` is an execution layer for a running TouchDesigner session. It can inspect operators, change parameters, read and write DATs, export snapshots, apply shader templates, and execute Python inside TouchDesigner.

### What This Is Good For

- inspect a live TD scene from the terminal
- build or patch operators without hunting through the network editor
- iterate on Python DATs and GLSL shaders from local files
- automate repetitive setup tasks with an LLM or shell scripts
- give artists a recoverable workflow with backups and audit logs

### How It Works

```text
Artist / LLM / Terminal
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

The TouchDesigner side writes heartbeat files to `~/.td-cli/instances/`, and `td-cli` uses those files to auto-discover running projects.

In an agent workflow, the model is the reasoning layer and `td-cli` is the execution layer.

Security note: if you set `TD_CLI_TOKEN` in both the shell running `td-cli` and the TouchDesigner process environment before launch, the server will require that shared token on every HTTP request.

### Artist Workflow

Think of `td-cli` as a live studio assistant for TouchDesigner:

1. Find the running project.
2. Inspect the current network or parameters.
3. Create or patch operators.
4. Make the result visible in a container, window, or screenshot.
5. Iterate quickly, and fall back with backups if needed.

Typical live workflow:

```powershell
td-cli status
td-cli ops list /project1 --depth 2
td-cli ops create noiseTOP /project1 --name myNoise
td-cli par get /project1/myNoise
td-cli par set /project1/myNoise period 4 amp 0.35
td-cli screenshot /project1/myNoise -o noise.png
```

### Visual Output Workflow

Creating a TOP or GLSL network is only part of the job. You still need to route it somewhere visible.

Common options:

- assign the result to a container's `Background TOP`
- point a viewer or window at a COMP
- save the result with `td-cli screenshot`

Example:

```powershell
td-cli par set /project1/myContainer top ./out1
td-cli screenshot /project1/myContainer/out1 -o frame.png
```

Important: for OP-reference parameters such as `top`, `opviewer`, `pixeldat`, `component`, or `winop`, prefer local relative paths like `./out1`. The handler will normalize resolvable local targets to relative references.

### Shader Workflow

For artists, shaders usually follow this loop:

1. inspect available templates
2. read a template before using it
3. apply it to a GLSL TOP
4. tweak DAT content or parameters live
5. route the output to a visible TOP or COMP

```powershell
td-cli shaders list
td-cli shaders get plasma
td-cli shaders apply plasma /project1/glsl1
td-cli dat read /project1/glsl1_pixel
td-cli screenshot /project1/glsl1 -o glsl.png
```

### POP Audio Visual Workflow

If you want a ready-made POP scene for live audio, `td-cli` can build one directly under a safe container instead of rewriting the whole project root.

```powershell
td-cli pop av --root /project1 --name popAudioVisual
td-cli screenshot /project1/popAudioVisual/out -o pop-av.png
```

This creates:

- `/project1/popAudioVisual` with the audio CHOP chain, POP network, and TOP post-processing
- `/project1/popAudioVisual_preview` as a preview container wired to the output TOP

### Harness Loop

The harness surface is the structured loop for agentic TouchDesigner work: observe, apply, verify, inspect history, and roll back.

```powershell
td-cli harness capabilities
td-cli harness observe /project1 --depth 2
td-cli harness apply /project1 --file patch.json
td-cli harness verify /project1 --assert '{"kind":"family","equals":"COMP"}'
td-cli harness history
td-cli harness rollback 1712900000-harness
```

`apply` expects JSON shaped like:

```json
{
  "targetPath": "/project1",
  "goal": "add preview chain",
  "operations": [
    {
      "route": "/ops/create",
      "body": { "type": "nullTOP", "parent": "/project1", "name": "out1" }
    }
  ]
}
```

Important: do not target a scope that contains `TDCliServer`. Use a child COMP scope such as `/project1/myScene`, not `/project1`, for harness mutation and rollback.

### Beginner Install Guide

#### 1. Prerequisites

- TouchDesigner installed and able to open projects
- a terminal such as PowerShell on Windows
- one of the following:
  - prebuilt `td-cli.exe` from GitHub Releases
  - Go `1.26.1` or newer if building from source

#### 2. Install the CLI

Option A: download a release binary

1. Download `td-cli.exe` from [Releases](https://github.com/0dot77/td-cli/releases).
2. Put it somewhere easy to find, for example `C:\Tools\td-cli\td-cli.exe`.
3. Either run it by full path or add that folder to `PATH`.

Example:

```powershell
C:\Tools\td-cli\td-cli.exe version
```

Option B: install with Go

```powershell
go install github.com/0dot77/td-cli/cmd/td-cli@latest
```

To build this repository directly:

```powershell
go build -o td-cli.exe ./cmd/td-cli/
```

#### 3. Install the TouchDesigner Connector

You must add the `TDCliServer` connector to your TouchDesigner project before `td-cli` can connect.

Recommended setup:

1. Open your TouchDesigner project.
2. Drag-and-drop [`tox/TDCliServer.tox`](tox/TDCliServer.tox) into the root network, or import it from TouchDesigner.
3. Make sure the imported component is named `TDCliServer`.
4. Open it and verify that `webserver1` is active on port `9500`.

Normal usage boundary:

- treat `TDCliServer` as an installed runtime connector
- use `td-cli` commands to inspect and modify the rest of the project
- avoid editing `/project1/TDCliServer/*` during normal AI or artist workflows

Developer-only reference files for connector work:

- [`td/webserver_callbacks.py`](td/webserver_callbacks.py)
- [`td/td_cli_handler.py`](td/td_cli_handler.py)
- [`td/heartbeat.py`](td/heartbeat.py)

Detailed setup notes are also in [`td/setup_instructions.md`](td/setup_instructions.md).

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
  Connector:  TDCliServer v...
```

If multiple TouchDesigner projects are open:

```powershell
td-cli instances
td-cli --port 9500 status
td-cli --project "C:\path\to\your\project.toe" status
```

#### 5. Bootstrap Agent Guidance

```powershell
td-cli init
```

This creates a `CLAUDE.md` file with command examples and usage notes. The CLI itself is not Claude-specific; Codex and other agents can use the same commands directly, or adapt the generated guidance into `AGENTS.md` or another instruction format.

The generated guidance tells agents to treat `TDCliServer` as the installed connector boundary and to use `td-cli` as the main execution surface.

### First Commands To Try

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
|------|------|
| `td-cli status` | Check TD connection |
| `td-cli instances` | List running TD instances |
| `td-cli exec "<code>"` | Execute Python in TD |
| `td-cli exec -f script.py` | Execute a local Python file in TD |
| `td-cli ops list [path]` | List operators |
| `td-cli ops create <type> <parent>` | Create an operator |
| `td-cli ops delete <path>` | Delete an operator |
| `td-cli ops info <path>` | Show operator details |
| `td-cli par get <op> [names]` | Read parameter values |
| `td-cli par set <op> <name> <value>` | Set one or more parameters |
| `td-cli connect <src> <dst>` | Connect operators |
| `td-cli disconnect <src> <dst>` | Disconnect operators |
| `td-cli dat read <path>` | Read DAT content |
| `td-cli dat write <path> <content>` | Write DAT content |
| `td-cli screenshot [path] -o file.png` | Save TOP output as PNG |
| `td-cli project info` | Show project metadata |
| `td-cli project save [path]` | Save the project |
| `td-cli backup list [--limit N]` | List recent backup artifacts |
| `td-cli backup restore <backup-id>` | Restore a previous backup |
| `td-cli logs list [--limit N]` | List recent audit log events |
| `td-cli logs tail [--limit N]` | Read recent audit log events |
| `td-cli tox export <comp> -o file.tox` | Export a COMP as `.tox` |
| `td-cli tox import <file.tox> [parent]` | Import a `.tox` file |
| `td-cli network export [path] [-o file]` | Export a network snapshot |
| `td-cli network import <file> [path]` | Import a network snapshot |
| `td-cli describe [path]` | Generate an AI-friendly network summary |
| `td-cli diff <file1> <file2>` | Compare two snapshots |
| `td-cli diff --live <file> [path]` | Compare a snapshot against live TD state |
| `td-cli watch [path] [--interval ms]` | Monitor live performance |
| `td-cli tools list` | List available tool routes for agent discovery |
| `td-cli shaders list` | List shader templates |
| `td-cli shaders get <name>` | Show shader template details |
| `td-cli shaders apply <name> <glsl_top_path>` | Apply a shader template |
| `td-cli pop av [audio-reactive] [--root /project1] [--name popAudioVisual]` | Build a POP audio reactive scene |
| `td-cli docs` | Browse offline docs |
| `td-cli docs <operator>` | Look up an operator |
| `td-cli docs api [class]` | Read Python API docs |
| `td-cli init` | Generate CLAUDE.md + AGENTS.md for agent integration |
| `td-cli doctor` | Diagnose setup and connection issues |
| `td-cli update` | Self-update from GitHub Releases |
| `td-cli version` | Show version |

### Global Flags

- `--port <N>`: connect to a specific port
- `--project <path>`: target a specific `.toe` project
- `--json`: output raw JSON
- `--debug`: log HTTP requests and responses to stderr
- `--timeout <ms>`: change request timeout, default `30000`

### Troubleshooting

Run `td-cli doctor` first — it checks the home directory, heartbeat files, port reachability, server health, and protocol version in one pass.

If `td-cli status` reports no running TouchDesigner instances:

- confirm the TouchDesigner project is actually open
- confirm `webserver1` is active on port `9500`
- confirm the heartbeat callback is running
- confirm `~/.td-cli/instances/` is being updated
- if `status` shows a connector protocol warning, replace the project connector with the current `TDCliServer.tox`

If multiple projects are running:

- check the list with `td-cli instances`
- then target the right one with `--port` or `--project`

If a visual result exists but you still do not see it:

- route the output to a visible `Background TOP`, viewer, or window
- use `td-cli screenshot` to verify that the TOP is actually rendering
- check OP-reference parameters and prefer relative paths like `./out1`

If the command is not found:

- try the full path to `td-cli.exe`
- if that works, add its folder to `PATH`

### Security

td-cli communicates with TouchDesigner over HTTP on `127.0.0.1` (localhost only). It is designed for local, single-user workflows.

**Code execution:** The `td-cli exec` command runs arbitrary Python code inside the TouchDesigner process. This is by design — it gives agents and artists full scripting access. Anyone who can reach the HTTP port can execute code with the same permissions as the TouchDesigner process.

**Authentication:** Set `TD_CLI_TOKEN` in both the shell and the TouchDesigner process environment to require HMAC token verification on every request. Without the token, any process on the same machine can use the API.

**When to enable the token:**
- Shared workstations where multiple users are logged in
- Environments where untrusted code runs alongside TouchDesigner
- Remote access via SSH tunnels

**CORS:** The server only accepts requests from `localhost` and `127.0.0.1` origins. Cross-origin requests from other hosts are rejected.

For typical local use (single user, single machine), running without a token is fine.

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

`td-cli`는 실행 중인 TouchDesigner 세션을 터미널에서 제어하는 실행 레이어입니다. 오퍼레이터 조회, 파라미터 수정, DAT 읽기/쓰기, 네트워크 스냅샷, 셰이더 템플릿 적용, TouchDesigner 내부 Python 실행까지 다룰 수 있습니다.

### 이 프로젝트로 할 수 있는 것

- 실행 중인 TouchDesigner 프로젝트를 터미널에서 직접 다루기
- 네트워크를 뒤지지 않고 필요한 오퍼레이터와 파라미터를 바로 조회하기
- 로컬 파일에서 Python DAT나 GLSL 셰이더를 빠르게 반복 수정하기
- LLM이나 스크립트로 반복 작업 자동화하기
- 백업과 감사 로그를 기반으로 더 안전하게 라이브 수정하기

### 동작 방식

```text
작가 / LLM / Terminal
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

에이전트 워크플로에서 LLM은 추론 레이어이고 `td-cli`는 실행 레이어입니다.

보안 참고: `td-cli`를 실행하는 셸과 TouchDesigner 프로세스를 실행하기 전 환경에 동일한 `TD_CLI_TOKEN` 값을 설정하면, 서버가 모든 HTTP 요청에 대해 해당 공유 토큰을 요구합니다.

### 작가 워크플로

작가 입장에서는 `td-cli`를 라이브 스튜디오 어시스턴트처럼 생각하면 됩니다.

1. 지금 열려 있는 프로젝트를 찾고
2. 현재 네트워크나 파라미터를 읽고
3. 필요한 오퍼레이터를 만들거나 수정하고
4. 결과를 보이게 연결하고
5. 백업과 로그를 남기면서 반복합니다

기본적인 라이브 작업 흐름:

```powershell
td-cli status
td-cli ops list /project1 --depth 2
td-cli ops create noiseTOP /project1 --name myNoise
td-cli par get /project1/myNoise
td-cli par set /project1/myNoise period 4 amp 0.35
td-cli screenshot /project1/myNoise -o noise.png
```

### 화면에 보이게 만드는 워크플로

TOP이나 GLSL 네트워크를 만들었다고 끝이 아닙니다. 실제로는 그 결과를 어디엔가 보이게 연결해야 합니다.

보통은 아래 셋 중 하나입니다.

- 컨테이너의 `Background TOP`으로 연결
- viewer나 window가 볼 COMP를 지정
- `td-cli screenshot`으로 렌더 결과를 바로 확인

예시:

```powershell
td-cli par set /project1/myContainer top ./out1
td-cli screenshot /project1/myContainer/out1 -o frame.png
```

중요: `top`, `opviewer`, `pixeldat`, `component`, `winop` 같은 OP reference 파라미터에는 `./out1` 같은 상대 경로를 권장합니다. 현재 핸들러는 로컬에서 해석 가능한 대상이면 상대 경로로 정규화합니다.

### 셰이더 워크플로

작가용 셰이더 작업은 보통 아래 순서로 갑니다.

1. 어떤 템플릿이 있는지 보고
2. 적용 전에 내용을 읽고
3. GLSL TOP에 올리고
4. DAT나 파라미터를 라이브로 수정하고
5. 결과를 화면이나 스크린샷으로 확인합니다

```powershell
td-cli shaders list
td-cli shaders get plasma
td-cli shaders apply plasma /project1/glsl1
td-cli dat read /project1/glsl1_pixel
td-cli screenshot /project1/glsl1 -o glsl.png
```

### POP 오디오 비주얼 워크플로

라이브 오디오에 반응하는 POP 장면이 필요하면, 프로젝트 루트를 통째로 건드리지 않고 안전한 전용 컨테이너 아래에 바로 생성할 수 있습니다.

```powershell
td-cli pop av --root /project1 --name popAudioVisual
td-cli screenshot /project1/popAudioVisual/out -o pop-av.png
```

이 명령은 아래 둘을 만듭니다.

- `/project1/popAudioVisual`: 오디오 CHOP 체인, POP 네트워크, TOP 후처리
- `/project1/popAudioVisual_preview`: 출력 TOP이 연결된 preview 컨테이너

### 예정된 Harness 루프

### Harness 루프

Harness 표면은 에이전트가 TouchDesigner에서 관측, 적용, 검증, 히스토리 조회, 롤백 루프를 구조적으로 돌릴 수 있게 하는 상위 명령입니다.

```powershell
td-cli harness capabilities
td-cli harness observe /project1 --depth 2
td-cli harness apply /project1 --file patch.json
td-cli harness verify /project1 --assert '{"kind":"family","equals":"COMP"}'
td-cli harness history
td-cli harness rollback 1712900000-harness
```

`apply`가 읽는 JSON 기본 형태는 아래와 같습니다.

```json
{
  "targetPath": "/project1",
  "goal": "add preview chain",
  "operations": [
    {
      "route": "/ops/create",
      "body": { "type": "nullTOP", "parent": "/project1", "name": "out1" }
    }
  ]
}
```

중요: `TDCliServer`를 포함하는 스코프에는 `apply`를 직접 걸지 마세요. 하네스 수정/롤백 범위는 `/project1` 전체보다 `/project1/myScene` 같은 하위 COMP로 잡아야 합니다.

### 초심자용 설치 가이드

#### 1. 준비물

- TouchDesigner가 설치되어 있고 프로젝트를 열 수 있어야 합니다
- PowerShell 같은 Windows 터미널이 필요합니다
- 아래 둘 중 하나가 필요합니다
  - GitHub Releases에서 받은 미리 빌드된 `td-cli.exe`
  - 소스에서 직접 설치할 경우 Go `1.26.1` 이상

#### 2. CLI 설치

방법 A: 릴리스 바이너리 다운로드

1. [Releases](https://github.com/0dot77/td-cli/releases)에서 `td-cli.exe`를 내려받습니다.
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
go install github.com/0dot77/td-cli/cmd/td-cli@latest
```

이 저장소를 직접 빌드하려면:

```powershell
go build -o td-cli.exe ./cmd/td-cli/
```

#### 3. TouchDesigner 커넥터 설치

`td-cli`가 연결되려면 먼저 TouchDesigner 프로젝트 안에 `TDCliServer` 커넥터를 넣어야 합니다.

권장 방법:

1. TouchDesigner에서 프로젝트를 엽니다.
2. [`tox/TDCliServer.tox`](tox/TDCliServer.tox)를 루트 네트워크에 드래그앤드롭하거나 import 합니다.
3. 가져온 컴포넌트 이름이 `TDCliServer`인지 확인합니다.
4. 컴포넌트 안으로 들어가서 `webserver1`이 포트 `9500`에서 활성화되어 있는지 확인합니다.

일반 사용 경계:

- `TDCliServer`는 설치된 런타임 커넥터로 취급합니다
- 실제 작업은 `td-cli` 명령으로 프로젝트 나머지 부분에 대해 수행합니다
- 일반적인 AI/작업 워크플로에서는 `/project1/TDCliServer/*`를 직접 수정하지 않습니다

커넥터 개발용 참고 파일:

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
  Connector:  TDCliServer v...
```

TouchDesigner 프로젝트를 여러 개 열어 둔 경우:

```powershell
td-cli instances
td-cli --port 9500 status
td-cli --project "C:\path\to\your\project.toe" status
```

#### 5. 에이전트 가이드 시작하기

```powershell
td-cli init
```

이 명령은 현재 폴더에 `CLAUDE.md`를 만들고, 명령 예시와 사용법을 기록합니다. CLI 자체는 Claude 전용이 아니고, Codex 같은 다른 에이전트도 같은 명령을 그대로 사용할 수 있습니다. 필요하면 생성된 내용을 `AGENTS.md` 같은 형식으로 옮겨도 됩니다.

생성되는 가이드는 `TDCliServer`를 설치된 커넥터 경계로 보고, 실제 작업 표면은 `td-cli` 명령으로 유지하도록 안내합니다.

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
| `td-cli par set <op> <name> <value>` | 하나 이상의 파라미터 값 설정 |
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
| `td-cli tools list` | 에이전트 검색용 tool route 목록 |
| `td-cli shaders list` | 셰이더 템플릿 목록 |
| `td-cli shaders get <name>` | 셰이더 템플릿 상세 보기 |
| `td-cli shaders apply <name> <glsl_top_path>` | 셰이더 템플릿 적용 |
| `td-cli pop av [audio-reactive] [--root /project1] [--name popAudioVisual]` | POP 오디오 리액티브 장면 생성 |
| `td-cli docs` | 내장 오프라인 문서 보기 |
| `td-cli docs <operator>` | 오퍼레이터 문서 조회 |
| `td-cli docs api [class]` | Python API 문서 조회 |
| `td-cli init` | 에이전트 연동용 CLAUDE.md + AGENTS.md 생성 |
| `td-cli doctor` | 셋업 및 연결 문제 진단 |
| `td-cli update` | GitHub Releases에서 자체 업데이트 |
| `td-cli version` | 버전 표시 |

### 전역 플래그

- `--port <N>`: 특정 포트로 직접 연결합니다
- `--project <path>`: 특정 `.toe` 프로젝트를 대상으로 지정합니다
- `--json`: 결과를 JSON으로 출력합니다
- `--debug`: HTTP 요청/응답을 stderr에 출력합니다
- `--timeout <ms>`: 요청 타임아웃을 변경합니다. 기본값은 `30000`

### 문제 해결

먼저 `td-cli doctor`를 실행하세요 — 홈 디렉토리, heartbeat 파일, 포트 접근성, 서버 상태, 프로토콜 버전을 한번에 점검합니다.

`td-cli status`에서 실행 중인 TouchDesigner 인스턴스가 없다고 나오는 경우:

- TouchDesigner 프로젝트가 실제로 열려 있는지 확인하세요
- `webserver1`이 포트 `9500`에서 활성화되어 있는지 확인하세요
- heartbeat 콜백이 실제로 실행되는지 확인하세요
- `~/.td-cli/instances/` 경로가 실제로 갱신되는지 확인하세요
- `status`에 커넥터 프로토콜 경고가 보이면 프로젝트의 `TDCliServer.tox`를 현재 버전으로 교체하세요

프로젝트가 여러 개 실행 중인 경우:

- `td-cli instances`로 목록을 확인하세요
- 그 다음 `--port` 또는 `--project`로 원하는 프로젝트를 지정하세요

결과가 존재하는데 화면에 안 보이는 경우:

- 출력 TOP을 `Background TOP`, viewer, window 중 하나에 연결했는지 확인하세요
- `td-cli screenshot`으로 실제 렌더 여부를 먼저 확인하세요
- OP reference 파라미터는 `./out1` 같은 상대 경로를 우선 사용하세요

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
