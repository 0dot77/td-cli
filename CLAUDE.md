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
- `poptoSOP` uses `par.pop = <pop_op>` (parameter reference, NOT wire connections — it's a SOP)
- `geometryCOMP` uses `par.pathsop` (NOT `par.sop`)

### POP Rendering Pipeline (Verified Pattern)
```python
# POP → SOP → Geo → Render (wire + parameter references)
p2s = container.create(_T('poptoSOP'), 'pop2sop')
p2s.par.pop = noise_pop          # par reference, NOT wire!

geo = container.create(_T('geometryCOMP'), 'geo')
geo.par.pathsop = p2s            # pathsop, NOT sop!
geo.par.material = mat
geo.par['ry'].expr = 'absTime.seconds * 5'  # rotate at geo level, NOT POP level

cam = container.create(_T('cameraCOMP'), 'cam')
cam.par.lookat = geo             # lookAt keeps mesh centered

render = container.create(_T('renderTOP'), 'render')
render.par.camera = cam
render.par.geometry = geo
render.par.lights = light.path   # string path for lights
```
- Rotate at geometryCOMP level (stable), NOT at transformPOP level (pushes mesh out of view)
- Use `cam.par.lookat = geo` to keep rotating geometry centered in frame
- For wireframe: `constantMAT` with `par.wireframe = 'on'` (self-lit, no lighting needed)

### noisePOP — Animation & Calibration (CRITICAL)
- `par.tx/ty/tz`: translates noise field **spatially** — pushes points far away! DON'T use for animation
- `par.t4d`: translates 4th dimension of 4D noise — use THIS for smooth temporal animation
- `par.gain`: controls displacement amplitude — keep small (0.1–1.5), calibrate against audio range
- `par.spread`: controls harmonic spread — keep 0.1–0.8
```python
noise.par.type = 'simplex4d'
noise.par['t4d'].expr = 'absTime.seconds * 0.5'  # smooth morph over time
noise.par['gain'].expr = "op('math_bass')['chan1'] * 0.8 + 0.15"  # NOT * 10!
```

### renderTOP Parameters (TD 099)
- Uses PARAMETER REFERENCES, not wire connections
- `render.par.camera = cam_op` (not `render.inputConnectors[0].connect(...)`)
- `render.par.geometry = geo_op`
- `render.par.lights = '/project1/light1 /project1/light2'` (space-separated paths)

### Common Parameter Name Gotchas
- `selectCHOP`: `channames` (not `chans`)
- `mathCHOP`: `gain`, `fromrange1/2`, `torange1/2` (no `clamp`/`clampmax`)
- `analyzeCHOP`: `function` — use `'rmspower'` for audio (NOT `'average'` — average cancels +/- audio signals to ~0)
- `noiseCHOP`: `rough` (not `roughness`)
- `levelTOP`: `brightness1` (not `brightness`), `contrast`
- `compositeTOP`: `operand` (not `blend`) — use STRING values: `'add'`, `'multiply'`, `'over'`, `'screen'`, etc. (NOT integer indices)
- `constantMAT`: `wireframe` (`'on'`/`'off'`), `wirewidth` — default is OFF
- `lightCOMP`: `dimmer` (not `intensity`), `cr/cg/cb` (not `colorr/colorg/colorb`)
- `blurTOP`: `size`
- `pointgeneratorPOP`: `numpoints` (not `rate`)
- `spherePOP`: `radx/rady/radz` (not `radius`)
- `noisePOP`: `spread`, `gain`, `t4d` for time animation (NOT `tx/ty/tz`)
- `gridPOP`: `sizex/sizey`, `cols/rows`, `randomx/randomy`
- `poptoSOP`: `pop` (reference to source POP, NOT wire connection)
- `geometryCOMP`: `pathsop` (not `sop`), `lookat` on cameraCOMP
- `glslmultiTOP`: `pixeldat`, `vec0name`, `vec0valuex/y/z/w` for uniforms

### Audio Signal Calibration (CRITICAL)
Raw audio from `audiodeviceinCHOP` is typically -60 to -20 dB (peak ~0.01–0.05).
After `audiofilterCHOP` + `analyzeCHOP(rmspower)`, values are ~0.001–0.01.
The `mathCHOP` gain must amplify to 0–1 range for shader uniforms.
```
Signal chain: audiodevicein → select(chan1) → audiofilter → analyze(rmspower) → lag → math(gain)
Typical gains: bass=5, mid=10, high=20 (adjust to mic/line level)
```
- Too low gain → no visible reaction
- Too high gain → shader saturates to white
- Always clamp in shader: `float bass = clamp(uAudio.x, 0.0, 2.0);`

### GLSL TOP Gotchas (TD 099 / macOS)
- `uTDOutputInfo.res` does NOT contain resolution — hardcode aspect ratio: `vec2(1.78, 1.0)`
- macOS does NOT support geometry shaders (`glslMAT.gdat`) — use raymarching in GLSL TOP instead
- Use `vUV.st` for texture coordinates (confirmed working)
- Use `TDOutputSwizzle()` on final fragColor
- `root` is a `baseCOMP` object, NOT a function — use `root.time` directly (not `root().time`)

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

### geometryCOMP — pathsop Causes Cook Loop (CRITICAL)
- DO NOT set `geo.par.pathsop` — causes cook dependency self-loop: `geo_main → geo_main`
- Instead, rely on **display/render flags** on the output SOP inside the geo
- Programmatically created SOPs have `display=False, render=False` by default — you MUST set them:
```python
geo = p.create(td.geometryCOMP, 'geo')
for child in list(geo.findChildren(depth=1)):
    child.destroy()
grid = geo.create(td.gridSOP, 'grid')
noise = geo.create(td.noiseSOP, 'noise')
noise.inputConnectors[0].connect(grid.outputConnectors[0])
null_out = geo.create(td.nullSOP, 'out')
null_out.inputConnectors[0].connect(noise.outputConnectors[0])
# CRITICAL — without this, render shows nothing:
null_out.display = True
null_out.render = True
# DO NOT: geo.par.pathsop = 'out'  ← cook loop!
```

### Expression Paths — Always Use Absolute Paths (CRITICAL)
- Expressions on parameters resolve relative to the **operator that owns the parameter**
- `op('math_bass')` inside a child SOP of `geo_main` resolves to `/project1/geo_main/math_bass` (WRONG)
- Always use absolute paths in expressions, especially for SOPs inside geometryCOMP:
```python
# WRONG — resolves relative to geo_main
noise.par.amp.expr = "op('math_bass')['chan1'] * 2.0"

# CORRECT — absolute path
noise.par.amp.expr = "op('/project1/math_bass')['chan1'] * 2.0"
```
- For top-level operators referencing each other, relative paths work fine
- For operators INSIDE a COMP referencing OUTSIDE operators, MUST use absolute paths

### Node Naming — Collision Suffix (IMPORTANT)
- When TD creates a node with a name that already exists (even if previously deleted), it appends a numeric suffix: `sl_bass` → `sl_bass1`
- After bulk `destroy()` + `create()`, names may not match expectations
- Always verify node names after creation, or use `op.name` attribute to get actual name
- Use absolute paths (`op.path`) in expressions, not assumed names

### UI Panel Building (TD 099)

#### Recommended: `parameterCOMP` (simplest, most reliable)
Auto-generates an interactive panel from any operator's custom parameters:
```python
ctrl = p.create(td.baseCOMP, 'ctrl')
# ... add custom parameter pages with appendCustomPage / appendFloat ...

ui = p.create(td.parameterCOMP, 'ui')
ui.par.op = ctrl.path        # point at ctrl
ui.par.builtin = False        # hide built-in pars
ui.par.custom = True          # show custom pars only
ui.par.pagenames = True       # show Audio/Visual/Post tabs
ui.par.labels = True
ui.par.compress = 0.85        # compact layout

# Open as interactive window (no A-key needed)
win = p.create(td.windowCOMP, 'ui_window')
win.par.winop = ui.path
win.par.winw = 350
win.par.winh = 550
win.par.borders = True
win.par.winopen = True
```
- Edits ctrl parameters **directly** — no expression wiring needed
- Page tabs, labels, sliders all auto-generated
- Best for: quick control panels, prototyping, parameter tuning

#### Custom: `containerCOMP` + `sliderCOMP` children
For fully custom-styled UI panels:
```python
ui = p.create(td.containerCOMP, 'ui')
ui.par.align = 'column'       # auto vertical stacking
ui.par.spacing = 4             # gap between children
ui.par.marginl = 10
ui.par.marginr = 10

sl = ui.create(td.sliderCOMP, 'sl_gain')
sl.par.label = 'Gain'
sl.par.valuerange0l = 0.0
sl.par.valuerange0h = 30.0
sl.par.value0 = 5.0
sl.par.hmode = 'fill'          # auto-fill parent width
sl.par.h = 40
```
- Use `hmode='fill'` on children for responsive width
- `align='column'` eliminates manual x/y positioning
- `windowCOMP` for guaranteed interaction; container viewer needs `A` key
- Nested containerCOMPs intercept mouse events — keep structure flat

#### Available widget types:
- `sliderCOMP` — slider with `value0`, `valuerange0l/h`, `label`, `colorr/g/b`
- `buttonCOMP` — toggle/momentary button
- `fieldCOMP` — text input field
- `listCOMP` — scrollable list
- `parameterCOMP` — auto-generated parameter panel
- `widgetCOMP` — generic widget container

### Creating Networks — Checklist
1. Always `import td` and use `td.lowercaseTypeCHOP` (not uppercase globals)
2. Always set positions with `pos(op, x, y)` immediately after creation
3. Connect inputs: `child.inputConnectors[0].connect(parent.outputConnectors[0])`
4. For renderTOP: use `par.camera`, `par.geometry`, `par.lights` (not wire connections)
5. For feedbackTOP: wire + par.top to SAME independent upstream node (see feedbackTOP section)
6. For poptoSOP: use `par.pop = pop_op` (parameter, NOT wire — SOP has no POP connectors)
7. For audio reactivity: use `par.expr = "op('math_bass')['chan1'] * 2.0"` (NOT `par.val`)
8. For 3D rotation: rotate at geometryCOMP level, NOT at POP/SOP level (avoids mesh leaving camera view)
9. For noisePOP time animation: use `par.t4d` (NOT `par.tx/ty/tz` which moves spatially)
10. Verify parameter names exist before setting — TD 099 has many gotchas (see table above)
11. Set `display=True, render=True` on output SOP inside geometryCOMP (defaults are False)
12. DO NOT set `geo.par.pathsop` — causes cook dependency self-loop
13. Use absolute paths in expressions for operators inside COMPs referencing outside operators
14. After bulk create, verify actual node names (TD may add numeric suffix)
