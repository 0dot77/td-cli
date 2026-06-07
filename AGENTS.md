# td-cli Development Guidelines

## Git Workflow
- Commit and push to origin/main after every meaningful change
- Concise commit messages in English

## Project Structure
- `cmd/td-cli/` — Go CLI entry | `internal/` — Go packages | `td/` — Python scripts for TD
- Build: `go build -o td-cli ./cmd/td-cli/`
- Handler recovery: if DAT errors break POST routes, paste `td/td_cli_handler.py` into `/project1/TDCliServer/handler`

## CLI Quick Reference
```bash
td-cli status                              # Check connection
td-cli context --depth 2                   # Project summary
td-cli exec -f scene.py --verify /project1 --screenshot /project1/render1
td-cli harness observe /project1 --depth 2 # Deep inspect
td-cli harness verify /project1 --assert '<json>'
td-cli harness rollback <id>               # Undo
td-cli tox export /project1/base1 -o base1.tox
```
**Core commands:** `ops list|create|delete|info|rename|copy|search`, `par get|set|pulse|expr|export`, `connect|disconnect`, `exec "<code>"|exec -f <file>`, `dat read|write`, `chop info|channels|sample`, `sop info|points`, `pop info|points|bounds`, `table rows|cell`, `screenshot`, `tox export|import`, `network export|import`, `docs <op>|search|api`
**Flags:** `--port N`, `--project <path>`, `--json`, `--timeout <ms>`

## TD Exec Guidelines (CRITICAL)

### Operator Types
- Types live in `td` module: `op.create(td.nullTOP, 'name')` — lowercase prefix always (`td.audiodeviceinCHOP`)
- Helper: `_T('nullTOP')` = `getattr(td, 'nullTOP')`
- No `popnet` in TD 099 — POPs are standalone, connect like regular ops
- Custom pars: `page.appendFloat('Name', label='Label')` returns ParGroup — set `.default` and `.val` separately

### POP Network
- Generator: gridPOP, pointgeneratorPOP, circlePOP, spherePOP
- Modifier: noisePOP, transformPOP, particlePOP, mathPOP, randomPOP
- Converter: soptoPOP, poptoSOP (`par.pop = pop_op`, NOT wire), choptoPOP, toptoPOP
- `geometryCOMP`: use `par.pathsop` (NOT `par.sop`)

### Render Pipeline
- `renderTOP` uses parameter refs: `par.camera`, `par.geometry`, `par.lights` (space-separated paths) — NOT wire
- Rotate at `geometryCOMP` level (stable), NOT at POP/SOP level
- `cam.par.lookat = geo` keeps rotating geometry centered
- Wireframe: `constantMAT` with `par.wireframe = 'on'`

### noisePOP
- `par.tx/ty/tz` = spatial translation (pushes points away!) — DON'T use for animation
- `par.t4d` = 4D noise dimension — use for smooth temporal animation
- `par.gain` = displacement amplitude (keep 0.1–1.5) | `par.spread` = harmonic (keep 0.1–0.8)

### Parameter Name Gotchas
| Operator | Correct | Wrong |
|----------|---------|-------|
| selectCHOP | `channames` | chans |
| mathCHOP | `gain`, `fromrange1/2`, `torange1/2` | clamp |
| analyzeCHOP | `function='rmspower'` | 'average' (cancels ±audio→0) |
| noiseCHOP | `rough` | roughness |
| levelTOP | `brightness1` | brightness |
| compositeTOP | `operand='add'` (string!) | blend / int index |
| lightCOMP | `dimmer`, `cr/cg/cb` | intensity, colorr |
| blurTOP | `size` | — |
| pointgeneratorPOP | `numpoints` | rate |
| spherePOP | `radx/rady/radz` | radius |
| noisePOP | `spread`, `gain`, `t4d` | tx/ty/tz |
| gridPOP | `sizex/sizey`, `cols/rows` | — |
| poptoSOP | `pop` (par ref, NOT wire) | — |
| geometryCOMP | `pathsop` | sop |
| glslmultiTOP | `pixeldat`, `vec0name`, `vec0valuex/y/z/w` | — |
| constantMAT | `wireframe='on'/'off'`, `wirewidth` | — |

### Audio Signal Chain
```
audiodevicein → select(chan1) → audiofilter → analyze(rmspower) → lag → math(gain)
Gains: bass=5, mid=10, high=20 | Clamp in shader: clamp(val, 0.0, 2.0)
```
- `audiodeviceinCHOP` outputs 1 channel (`chan1`) — don't select `chan1-chan8`
- Audio reactivity: `par.expr = "op('math_bass')['chan1'] * 2.0"` (NOT `par.val = X`)

### GLSL TOP (TD 099 / macOS)
- `uTDOutputInfo.res` broken — hardcode aspect: `vec2(1.78, 1.0)`
- No geometry shaders on macOS — use raymarching in GLSL TOP
- Use `vUV.st` for UVs, `TDOutputSwizzle()` on fragColor
- `root` is baseCOMP object, not function — `root.time` (not `root().time`)

### feedbackTOP (CRITICAL)
Wire + `par.top` to SAME independent upstream node. Target must NOT depend on feedback.
```
glsl → fb(wire+par.top) → levelTOP(opacity=0.85) → compositeTOP[1]
glsl → compositeTOP[0]
```

### geometryCOMP (CRITICAL)
- DO NOT set `geo.par.pathsop` — causes cook self-loop
- Use display/render flags: `null_out.display = True; null_out.render = True`
- Programmatic SOPs default to `display=False, render=False` — MUST set explicitly

### Expressions & Naming
- Inside COMPs referencing outside ops: MUST use absolute paths (`op('/project1/math_bass')`)
- Top-level ops referencing each other: relative paths OK
- Name collisions: TD appends suffix (`sl_bass` → `sl_bass1`) — verify with `op.name`

### UI Panels
- Quick: `parameterCOMP` — `par.op=ctrl.path`, `par.builtin=False`, `par.custom=True`
- Custom: `containerCOMP(align='column')` + `sliderCOMP(hmode='fill')`
- `windowCOMP` for guaranteed interaction (container viewer needs A key)

### webclientDAT (API Fetching)
- Defaults to POST — always set `par.reqmethod = 'get'` for REST APIs
- `par.includeheader = False` to get clean JSON
- `onResponse(dat, statusCode, headerDict, data)`: statusCode is dict `{'code': 200, 'message': 'OK'}` — use `statusCode.get('code', 0)`

### Network Creation Checklist
1. `import td` — use `td.lowercaseTypeFAMILY` (not uppercase globals)
2. Position every node: `op.nodeCenterX = x; op.nodeCenterY = y`
3. renderTOP: `par.camera/geometry/lights` (not wires)
4. feedbackTOP: wire + par.top to same upstream node
5. poptoSOP: `par.pop = pop_op` (par ref, not wire)
6. Audio: `par.expr = "..."` (not `par.val = X`)
7. noisePOP animation: `par.t4d` (not `par.tx/ty/tz`)
8. geometryCOMP: set display/render flags on output SOP, don't set `par.pathsop`
9. Expressions inside COMPs: absolute paths to outside ops
10. COMP outputs: use `outCHOP`/`outTOP`/`outSOP`/`outDAT` (real container outputs, not null named 'out')
11. Set `comp.comment` on baseCOMP: brief description, output type, key params
12. Verify node names after bulk create (TD may add numeric suffix)
13. `parameterexecuteDAT` inside a COMP cannot watch its own parent COMP's custom params — place it OUTSIDE
14. Extension cleanup: `comp.par.extension1` is a Python expression, NOT a path — never assign an operator to it. If stale, clear with `par.extension1=''` + `reinitextensions.pulse()` + `comp.clearScriptErrors()`
15. Stale COMP script errors persist even after fixing the cause — always call `comp.clearScriptErrors()` after extension/script fixes
16. Custom par UI triggers: `parameterexecuteDAT` cannot watch baseCOMP custom params — use Toggle par + `constantCHOP(expr)` + `chopexecuteDAT` instead of Pulse par
17. timerCHOP periodic trigger: `done` channel off→on may not fire reliably in cycle mode — use `timer_fraction` off→on instead (fires at each cycle start). Don't filter by `channel.name` in `onOffToOn`
18. timerCHOP start: `par.start` is a pulse — use `par.start.pulse()`, not `par.start = True`

### Node Layout
```python
def pos(op_ref, x, y):
    op_ref.nodeCenterX = x
    op_ref.nodeCenterY = y
```
Left→right = data flow, top→bottom = parallel. Column spacing ~300px, row ~150px.
