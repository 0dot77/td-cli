# td-cli

面向 LLM 智能体、艺术家和终端驱动工作流的 TouchDesigner 命令行界面（CLI）。

`td-cli` 将实时运行的 TouchDesigner 项目连接到终端。它可同时在两种模式下发挥作用：

- 作为 Codex 或 Claude 等 LLM 智能体的命令界面
- 作为艺术家的实用实时控制工具，使其无需手动点击每个网络节点即可检查、构建和微调 TouchDesigner 项目

## 简体中文

`td-cli` 是一个运行中的 TouchDesigner 会话的执行层。它可以检查算子（operators）、修改参数、读写 DAT、导出快照、应用着色器模板，并在 TouchDesigner 内部执行 Python 代码。

### 适用场景

- 从终端检查实时 TD 场景
- 构建或连接算子，无需在网络编辑器中四处寻找
- 基于本地文件快速迭代 Python DAT 和 GLSL 着色器
- 使用 LLM 或 Shell 脚本自动化重复性的设置任务
- 为艺术家提供带有备份和审计日志的可恢复工作流

### 工作原理

```text
艺术家 / LLM / 终端
          |
          v
td-cli (Go 二进制文件)
          |
          v
HTTP 端口 9500
          |
          v
TouchDesigner Web Server DAT + Python 处理程序
```

TouchDesigner 端会将心跳文件写入 `~/.td-cli/instances/`，`td-cli` 利用这些文件自动发现运行中的项目。

在智能体工作流中，模型是推理层，而 `td-cli` 是执行层。

安全提示：如果您在运行 `td-cli` 的 Shell 和启动前的 TouchDesigner 进程环境中都设置了 `TD_CLI_TOKEN`，则服务器将在每次 HTTP 请求时要求该共享令牌。

### 艺术家工作流

可以将 `td-cli` 视为 TouchDesigner 的实时工作室助手：

1. 找到运行中的项目。
2. 检查当前网络或参数。
3. 创建或连接算子。
4. 将结果在容器、窗口中可见，或拍摄屏幕截图。
5. 快速迭代，必要时通过备份回滚。

典型的实时工作流：

```powershell
td-cli status
td-cli ops list /project1 --depth 2
td-cli ops create noiseTOP /project1 --name myNoise
td-cli par get /project1/myNoise
td-cli par set /project1/myNoise period 4 amp 0.35
td-cli screenshot /project1/myNoise -o noise.png
```

### 视觉输出工作流

创建 TOP 或 GLSL 网络只是工作的一部分。您仍然需要将其路由到可见的位置。

常用方案：

- 将结果分配给容器的 `Background TOP`
- 将查看器（viewer）或窗口（window）指向某个 COMP
- 使用 `td-cli screenshot` 保存结果

示例：

```powershell
td-cli par set /project1/myContainer top ./out1
td-cli screenshot /project1/myContainer/out1 -o frame.png
```

重要提示：对于 `top`、`opviewer`、`pixeldat`、`component` 或 `winop` 等 OP 引用参数，请优先使用 `./out1` 这样的本地相对路径。处理程序会将可解析的本地目标正规化为相对引用。

### 着色器工作流

对于艺术家，着色器工作流通常遵循此循环：

1. 检查可用模板
2. 使用前阅读模板
3. 将其应用于 GLSL TOP
4. 实时微调 DAT 内容或参数
5. 将输出路由到可见的 TOP 或 COMP

```powershell
td-cli shaders list
td-cli shaders get plasma
td-cli shaders apply plasma /project1/glsl1
td-cli dat read /project1/glsl1_pixel
td-cli screenshot /project1/glsl1 -o glsl.png
```

### POP 音频视觉工作流

如果您需要一个用于实时音频的现成 POP 场景，`td-cli` 可以在一个安全的容器下直接构建，而无需重写整个项目根目录。

```powershell
td-cli pop av --root /project1 --name popAudioVisual
td-cli screenshot /project1/popAudioVisual/out -o pop-av.png
```

这将创建：

- `/project1/popAudioVisual`：包含音频 CHOP 链、POP 网络和 TOP 后处理
- `/project1/popAudioVisual_preview`：连接到输出 TOP 的预览容器

### Harness 循环

Harness 界面是智能体执行 TouchDesigner 工作的结构化循环：观察、应用、验证、检查历史记录并回滚。

```powershell
td-cli harness capabilities
td-cli harness observe /project1 --depth 2
td-cli harness apply /project1 --file patch.json
td-cli harness verify /project1 --assert '{"kind":"family","equals":"COMP"}'
td-cli harness history
td-cli harness rollback 1712900000-harness
```

`apply` 期待的 JSON 格式如下：

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

重要提示：请不要将目标范围设置为包含 `TDCliServer` 的范围。对于 Harness 的变更和回滚，请使用子 COMP 范围（如 `/project1/myScene`），而非 `/project1`。

### 初学者安装指南

#### 1. 前置条件

- 已安装 TouchDesigner 且能够打开项目
- Windows 上的 PowerShell 等终端
- 以下之一：
  - 从 GitHub Releases 下载的预编译 `td-cli.exe`
  - 如果从源码构建，需要 Go `1.26.1` 或更新版本

#### 2. 安装 CLI

方案 A：下载发布二进制文件

1. 从 [Releases](https://github.com/0dot77/td-cli/releases) 下载 `td-cli.exe`。
2. 将其放在易于查找的位置，例如 `C:\Tools\td-cli\td-cli.exe`。
3. 通过完整路径运行，或将该文件夹添加到 `PATH` 环境变量中。

示例：

```powershell
C:\Tools\td-cli\td-cli.exe version
```

方案 B：使用 Go 安装

```powershell
go install github.com/0dot77/td-cli/cmd/td-cli@latest
```

直接构建本仓库：

```powershell
go build -o td-cli.exe ./cmd/td-cli/
```

#### 3. 安装 TouchDesigner 连接器

在 `td-cli` 能够连接之前，您必须将 `TDCliServer` 连接器添加到 TouchDesigner 项目中。

推荐设置：

1. 打开您的 TouchDesigner 项目。
2. 将 [`tox/TDCliServer.tox`](tox/TDCliServer.tox) 拖放到根网络中，或将其导入 TouchDesigner。
3. 确保导入的组件名为 `TDCliServer`。
4. 打开它并验证 `webserver1` 在端口 `9500` 上处于激活状态。

正常使用边界：

- 将 `TDCliServer` 视为已安装的运行时连接器
- 使用 `td-cli` 命令来检查和修改项目的其余部分
- 在正常的 AI 或艺术家工作流中，避免编辑 `/project1/TDCliServer/*`

连接器开发参考文件：

- [`td/webserver_callbacks.py`](td/webserver_callbacks.py)
- [`td/td_cli_handler.py`](td/td_cli_handler.py)
- [`td/heartbeat.py`](td/heartbeat.py)

详细的设置说明也可在 [`td/setup_instructions.md`](td/setup_instructions.md) 中找到。

#### 4. 验证连接

```powershell
td-cli status
```

预期结果：

```text
Connected to TouchDesigner
  Project:    ...
  TD Version: ...
  Server:     td-cli v...
  Connector:  TDCliServer v...
```

如果打开了多个 TouchDesigner 项目：

```powershell
td-cli instances
td-cli --port 9500 status
td-cli --project "C:\path\to\your\project.toe" status
```

#### 5. 引导智能体指导

```powershell
td-cli init
```

这将创建一个 `CLAUDE.md` 文件，包含命令示例和使用说明。CLI 本身并非 Claude 专用；Codex 和其他智能体可以直接使用相同的命令，或将生成的指导适配到 `AGENTS.md` 或其他指令格式中。

生成的指导会告知智能体将 `TDCliServer` 视为已安装的连接器边界，并将 `td-cli` 作为主要的执行界面。

### 首次尝试的命令

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

### 主要命令

| 命令 | 描述 |
|------|------|
| `td-cli status` | 检查 TD 连接状态 |
| `td-cli instances` | 列出运行中的 TD 实例 |
| `td-cli exec "<code>"` | 在 TD 中执行 Python 代码 |
| `td-cli exec -f script.py` | 在 TD 中执行本地 Python 文件 |
| `td-cli ops list [path]` | 列出算子 |
| `td-cli ops create <type> <parent>` | 创建算子 |
| `td-cli ops delete <path>` | 删除算子 |
| `td-cli ops info <path>` | 显示算子详情 |
| `td-cli par get <op> [names]` | 读取参数值 |
| `td-cli par set <op> <name> <value>` | 设置一个或多个参数 |
| `td-cli connect <src> <dst>` | 连接算子 |
| `td-cli disconnect <src> <dst>` | 断开算子连接 |
| `td-cli dat read <path>` | 读取 DAT 内容 |
| `td-cli dat write <path> <content>` | 写入 DAT 内容 |
| `td-cli screenshot [path] -o file.png` | 将 TOP 输出保存为 PNG |
| `td-cli project info` | 显示项目元数据 |
| `td-cli project save [path]` | 保存项目 |
| `td-cli backup list [--limit N]` | 列出最近的备份文件 |
| `td-cli backup restore <backup-id>` | 恢复之前的备份 |
| `td-cli logs list [--limit N]` | 列出最近的审计日志事件 |
| `td-cli logs tail [--limit N]` | 读取最近的审计日志事件 |
| `td-cli tox export <comp> -o file.tox` | 将 COMP 导出为 `.tox` |
| `td-cli tox import <file.tox> [parent]` | 导入 `.tox` 文件 |
| `td-cli network export [path] [-o file]` | 导出网络快照 |
| `td-cli network import <file> [path]` | 导入网络快照 |
| `td-cli describe [path]` | 生成 AI 友好的网络摘要 |
| `td-cli diff <file1> <file2>` | 比较两个快照 |
| `td-cli diff --live <file> [path]` | 将快照与实时 TD 状态进行比较 |
| `td-cli watch [path] [--interval ms]` | 监控实时性能 |
| `td-cli tools list` | 列出可用于智能体发现的工具路由 |
| `td-cli shaders list` | 列出着色器模板 |
| `td-cli shaders get <name>` | 显示着色器模板详情 |
| `td-cli shaders apply <name> <glsl_top_path>` | 应用着色器模板 |
| `td-cli pop av [audio-reactive] [--root /project1] [--name popAudioVisual]` | 构建 POP 音频响应场景 |
| `td-cli docs` | 浏览离线文档 |
| `td-cli docs <operator>` | 查找算子文档 |
| `td-cli docs api [class]` | 阅读 Python API 文档 |
| `td-cli init` | 为智能体集成生成 CLAUDE.md + AGENTS.md |
| `td-cli doctor` | 诊断设置和连接问题 |
| `td-cli update` | 从 GitHub Releases 自行更新 |
| `td-cli version` | 显示版本 |

### 全局标志

- `--port <N>`: 连接到特定端口
- `--project <path>`: 目标特定的 `.toe` 项目
- `--json`: 输出原始 JSON
- `--debug`: 将 HTTP 请求和响应记录到 stderr
- `--timeout <ms>`: 更改请求超时时间，默认为 `30000`

### 故障排除

首先运行 `td-cli doctor` —— 它会一次性检查主目录、心跳文件、端口可达性、服务器健康状况和协议版本。

如果 `td-cli status` 报告没有运行中的 TouchDesigner 实例：

- 确认 TouchDesigner 项目确实已打开
- 确认 `webserver1` 在端口 `9500` 上处于激活状态
- 确认心跳回调（heartbeat callback）正在运行
- 确认 `~/.td-cli/instances/` 正在被更新
- 如果 `status` 显示连接器协议警告，请将项目连接器替换为最新的 `TDCliServer.tox`

如果运行了多个项目：

- 使用 `td-cli instances` 检查列表
- 然后使用 `--port` 或 `--project` 指定目标项目

如果视觉结果存在但您仍然看不到：

- 将输出路由到可见的 `Background TOP`、查看器或窗口
- 使用 `td-cli screenshot` 验证 TOP 是否实际上在渲染
- 检查 OP 引用参数并优先使用 `./out1` 这样的相对路径

如果找不到命令：

- 尝试使用 `td-cli.exe` 的完整路径
- 如果有效，请将其文件夹添加到 `PATH` 中

### 安全性

`td-cli` 通过 `127.0.0.1`（仅限本地）上的 HTTP 与 TouchDesigner 通信。它专为本地单用户工作流设计。

**代码执行：** `td-cli exec` 命令在 TouchDesigner 进程内运行任意 Python 代码。这是设计使然 —— 它赋予了智能体和艺术家完整的脚本访问权限。任何能够访问该 HTTP 端口的人都可以执行与 TouchDesigner 进程具有相同权限的代码。

**身份验证：** 在 Shell 和 TouchDesigner 进程环境中同时设置 `TD_CLI_TOKEN`，以便在每次请求时要求 HMAC 令牌验证。如果没有令牌，同一台机器上的任何进程都可以使用该 API。

**何时启用令牌：**
- 多个用户登录的共享工作站
- 与 TouchDesigner 并行运行不受信任代码的环境
- 通过 SSH 隧道进行的远程访问

**CORS：** 服务器仅接受来自 `localhost` 和 `127.0.0.1` 源的请求。来自其他主机的跨源请求将被拒绝。

对于典型的本地使用（单用户、单机），在没有令牌的情况下运行是安全的。

### 开发

本地构建：

```powershell
go build -o td-cli.exe ./cmd/td-cli/
```

显示帮助：

```powershell
td-cli help
```

## License

MIT
