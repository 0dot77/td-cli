# td-cli Development Guidelines

## Git Workflow
- Commit and push after every meaningful change (new feature, bug fix, refactor)
- Write concise commit messages in English
- Always push to origin/main after committing

## CLI Command Reference (for Claude)

### Workflow — typical agent loop
```bash
td-cli status                              # 1. Check connection
td-cli context --depth 2                   # 2. Get full project summary (tree, families, harness history)
td-cli exec -f scene.py --verify /project1 --screenshot /project1/render1
                                           # 3. Execute + verify + capture preview to .tmp/preview.png
td-cli harness observe /project1 --depth 2 # 4. Deep inspect (graph, data flow, issues)
td-cli harness verify /project1 --assert '[{"kind":"parameterValue","path":"/project1/noise1","name":"roughness","min":0.1}]'
                                           # 5. Assert expected state
td-cli harness rollback <id>               # 6. Undo if needed
```

### Connection & Discovery
| Command | Description |
|---------|-------------|
| `td-cli status` | Check TD connection |
| `td-cli context [--depth N]` | Project summary: tree, families, activity, harness |
| `td-cli instances` | List running TD instances |
| `td-cli describe [path]` | AI-friendly network description |

### Operators
| Command | Description |
|---------|-------------|
| `td-cli ops list [path] [--depth N] [--family TYPE]` | List operators |
| `td-cli ops create <type> <parent> [--name N] [--x X] [--y Y]` | Create operator |
| `td-cli ops delete <path>` | Delete operator |
| `td-cli ops info <path>` | Operator details |
| `td-cli ops rename <path> <new-name>` | Rename |
| `td-cli ops copy <src> <parent>` | Copy |
| `td-cli ops move <src> <parent>` | Move |
| `td-cli ops clone <src> <parent>` | Clone |
| `td-cli ops search <parent> <pattern> [--family TYPE]` | Search |

### Parameters
| Command | Description |
|---------|-------------|
| `td-cli par get <op> [names...]` | Read parameters |
| `td-cli par set <op> <name> <val> [...]` | Set parameters (key-value pairs) |
| `td-cli par pulse <op> <name>` | Pulse button parameter |
| `td-cli par reset <op> [names...]` | Reset to default |
| `td-cli par expr <op> <name> [expression]` | Get/set expression |
| `td-cli par export <op>` | Export all as JSON |
| `td-cli par import <op> <json>` | Import from JSON |

### Connections
| Command | Description |
|---------|-------------|
| `td-cli connect <src> <dst> [--src-index N] [--dst-index N]` | Wire operators |
| `td-cli disconnect <src> <dst>` | Unwire |

### Execution
| Command | Description |
|---------|-------------|
| `td-cli exec "<code>"` | Execute Python in TD |
| `td-cli exec -f <file>` | Execute from file |
| `td-cli exec ... --verify <path>` | + verify node graph after exec |
| `td-cli exec ... --screenshot <path>` | + capture TOP to `.tmp/preview.png` |

### Data Access
| Command | Description |
|---------|-------------|
| `td-cli dat read <path>` | Read DAT content |
| `td-cli dat write <path> <content> [-f file]` | Write DAT |
| `td-cli chop info <path>` | Channel info |
| `td-cli chop channels <path>` | List channels |
| `td-cli chop sample <path> [--channel NAME]` | Sample value |
| `td-cli sop info <path>` | Geometry info |
| `td-cli sop points <path>` | Point data |
| `td-cli pop info <path>` | POP info |
| `td-cli pop points <path> [--attr P]` | POP point data |
| `td-cli pop bounds <path>` | Bounding box |
| `td-cli table rows <path>` | Read rows |
| `td-cli table cell <path> <row> <col> [--value V]` | Read/write cell |

### Visual & Media
| Command | Description |
|---------|-------------|
| `td-cli screenshot [path] [-o file]` | Capture TOP as PNG |
| `td-cli media info <path>` | TOP metadata |
| `td-cli media export <path> <file>` | Export media |
| `td-cli watch [path] [--interval ms]` | Real-time monitor |

### Harness (Agent Loop)
| Command | Description |
|---------|-------------|
| `td-cli harness capabilities` | List supported features |
| `td-cli harness observe [path] [--depth N]` | Capture state snapshot |
| `td-cli harness verify [path] [--assert JSON]` | Run assertions |
| `td-cli harness apply <path> [--goal TEXT] [--op JSON]` | Apply operations with rollback |
| `td-cli harness rollback <id>` | Restore prior state |
| `td-cli harness history [--limit N]` | List iterations |

### Project & Timeline
| Command | Description |
|---------|-------------|
| `td-cli project info` | Project metadata |
| `td-cli project save [path]` | Save project |
| `td-cli timeline [info\|play\|pause]` | Timeline control |
| `td-cli timeline seek <time>` | Jump to frame |
| `td-cli cook node <path>` | Force cook operator |

### Templates & Docs
| Command | Description |
|---------|-------------|
| `td-cli pop av [--root path] [--name NAME]` | Build audio-reactive POP scene |
| `td-cli shaders list [--cat CAT]` | List shader templates |
| `td-cli shaders apply <name> <top>` | Apply shader to GLSL TOP |
| `td-cli docs <operator>` | Offline operator docs |
| `td-cli docs search <keyword>` | Search operators |
| `td-cli docs api [class]` | Python API reference |

### Batch & Network
| Command | Description |
|---------|-------------|
| `td-cli batch exec <file.json>` | Batch execute commands |
| `td-cli batch parset <file.json>` | Batch set parameters |
| `td-cli network export [path] [-o file]` | Export snapshot |
| `td-cli network import <file> [target]` | Import snapshot |
| `td-cli tox export <comp> -o <file>` | Export as .tox |
| `td-cli tox import <file> [parent]` | Import .tox |

### Global Flags
- `--port N` — connect to specific port
- `--project <path>` — target specific TD project
- `--json` — raw JSON output (pipe-friendly)
- `--timeout <ms>` — request timeout (default 30000)

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
go build -o td-cli ./cmd/td-cli/       # Mac
go build -o td-cli.exe ./cmd/td-cli/   # Windows
```

## Test
```bash
td-cli status
td-cli exec "return 1+1"
td-cli ops list /project1
```

## TD Exec Guidelines (CRITICAL)

### Operator Type Access
- TD operator types (e.g. `nullTOP`, `noisePOP`) live in the `td` module, NOT as globals
- Access: `import td; op.create(td.nullTOP, 'name')`
- Helper available in exec: `_T('nullTOP')` is a shortcut for `getattr(td, 'nullTOP')`
- DO NOT use uppercase `audioDeviceInCHOP` — use `td.audiodeviceinCHOP` (lowercase prefix)
- There is NO `popnet` in TD 099 — POPs are standalone operators (gridPOP, noisePOP, etc.)

### POP Network (TD 099)
- POPs connect like regular operators: `noisePOP.inputConnectors[0].connect(gridPOP.outputConnectors[0])`
- Generator POPs: gridPOP, pointgeneratorPOP, circlePOP, spherePOP
- Modifier POPs: noisePOP, transformPOP, particlePOP, mathPOP, randomPOP
- Converter: soptoPOP (SOP→POP), poptoSOP (POP→SOP), choptoPOP, toptoPOP

### renderTOP Parameters (TD 099)
- Uses PARAMETER REFERENCES, not wire connections
- `render.par.camera = cam_op` (not `render.inputConnectors[0].connect(...)`)
- `render.par.geometry = geo_op`
- `render.par.lights = '/project1/light1 /project1/light2'` (space-separated paths)

### Common Parameter Name Gotchas
- `selectCHOP`: `channames` (not `chans`)
- `mathCHOP`: `gain`, `fromrange1/2`, `torange1/2` (no `clamp`/`clampmax`)
- `levelTOP`: `brightness1` (not `brightness`), `contrast`
- `compositeTOP`: `operand` (not `blend`) — use STRING values: `'add'`, `'multiply'`, `'over'`, `'screen'`, etc. (NOT integer indices)
- `lightCOMP`: `dimmer` (not `intensity`), `cr/cg/cb` (not `colorr/colorg/colorb`)
- `blurTOP`: `size`
- `pointgeneratorPOP`: `numpoints` (not `rate`)
- `noisePOP`: `spread`, `gain` (not `turbx`/`turby`/`turbz` — use `transformPOP` for per-axis)
- `gridPOP`: `sizex/sizey`, `cols/rows`, `randomx/randomy`

### Making Parameters Audio Reactive
Use expression references on parameters:
```python
par.expr = "op('math_bass')['chan1'] * 2.0"
```
NOT Python assignments — `par.val = X` sets a static value.
IMPORTANT: `audioDeviceInCHOP` typically outputs only 1 channel (`chan1`).
Do NOT select `chan1-chan8`, `chan7-chan24`, etc. — use `sel.par.channames = 'chan1'` and split with `analyzeCHOP` or `audiofilterCHOP` for frequency bands.

### Exec Handler Scoping
- `-f` file mode works identically to inline mode (both go through same handler)
- `td` module is pre-imported in exec scope
- `_T(name)` helper is available for type lookup
- Variables persist within single exec call only

### Handler Recovery
If handler DAT has compilation errors, ALL POST routes fail (including `dat write`).
Recovery: in TD UI, open `/project1/TDCliServer/handler` DAT and paste content from `td/td_cli_handler.py`.
Alternative: use `td-cli exec` BEFORE the bad handler is pushed to verify syntax with `python3 -c "import py_compile; py_compile.compile('td/td_cli_handler.py', doraise=True)"`

### Node Layout (ALWAYS position nodes)
When creating multiple operators, ALWAYS set node positions to avoid overlap.
Use a helper function and arrange nodes in logical flow:

```python
def pos(op_ref, x, y):
    op_ref.nodeCenterX = x
    op_ref.nodeCenterY = y
```

**Layout convention (left → right = data flow, top → bottom = parallel branches):**
- Column spacing: ~300px between stages
- Row spacing: ~150px between parallel branches
- Source nodes: x starts at -1800
- Processing: -400 to 500
- Render: 800 to 1400
- Post-processing: 1700 to 2600

Example layout pattern:
```
Audio CHOPs (x: -1800 to -900)  |  POP chain (x: -400 to 500)  |  Render (x: 800+)  |  Post (x: 1700+)
  audio_in (-1800, 500)         |  gridp (-400, 500)            |  geo1 (800, 300)   |  level1 (1700, 500)
  sel_bass (-1500, 600)         |  noise1 (-100, 500)           |  cam1 (800, 600)   |  glow1 (1700, 300)
  sel_mid (-1500, 450)          |  xform1 (200, 500)            |  light1 (1100,600) |  comp1 (2000, 400)
  sel_high (-1500, 300)         |  pop_out (500, 500)           |  render (1400,450) |  out1 (2600, 450)
```

### feedbackTOP — Correct Wiring Pattern (CRITICAL)
feedbackTOP needs BOTH `par.top` AND wire input from the SAME independent upstream node.
- Wire input provides resolution and data source
- `par.top` tells feedback which TOP's previous frame to capture
- Target must NOT depend on feedback — otherwise cook dependency loop
- Stale feedback nodes get stuck — always `destroy()` and create fresh

```python
# WRONG — circular: fb targets a node that depends on fb
#   fb.par.top = comp; comp depends on fb  ← COOK LOOP!

# WRONG — no wire input
#   fb.par.top = glsl (only)  ← "Not enough sources" error!

# CORRECT — wire + par.top to independent upstream node
fb = container.create(_T('feedbackTOP'), 'fb')
fb.inputConnectors[0].connect(glsl.outputConnectors[0])  # wire first
fb.par.top = glsl                                          # then par.top

fade = container.create(_T('levelTOP'), 'fb_fade')
fade.par.opacity = 0.85
fade.inputConnectors[0].connect(fb.outputConnectors[0])

comp = container.create(_T('compositeTOP'), 'comp_trail')
comp.par.operand = 'over'
comp.inputConnectors[0].connect(glsl.outputConnectors[0])   # current frame
comp.inputConnectors[1].connect(fade.outputConnectors[0])   # faded prev frame
```
Pattern: `glsl → fb(wire+par.top) → fade → comp[1]`, `glsl → comp[0]` — zero errors, zero warnings.

### Creating Networks — Checklist
1. Always `import td` and use `td.lowercaseTypeCHOP` (not uppercase globals)
2. Always set positions with `pos(op, x, y)` immediately after creation
3. Connect inputs: `child.inputConnectors[0].connect(parent.outputConnectors[0])`
4. For renderTOP: use `par.camera`, `par.geometry`, `par.lights` (not wire connections)
5. For feedbackTOP: use `par.top = target_op` (NOT wire connections — causes cook loop)
6. For audio reactivity: use `par.expr = "op('math_bass')['chan1'] * 2.0"` (NOT `par.val`)
7. Verify parameter names exist before setting — TD 099 has many gotchas (see table above)
