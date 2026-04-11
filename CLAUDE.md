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
- `compositeTOP`: `operand` (not `blend`) — values: 0=multiply, 4=add, etc.
- `lightCOMP`: `dimmer` (not `intensity`), `cr/cg/cb` (not `colorr/colorg/colorb`)
- `blurTOP`: `size`
- `pointgeneratorPOP`: `numpoints` (not `rate`)
- `gridPOP`: `sizex/sizey`, `cols/rows`, `randomx/randomy`

### Making Parameters Audio Reactive
Use expression references on parameters:
```python
par.expr = "op('math_bass')['chan1'] * 2.0"
```
NOT Python assignments — `par.val = X` sets a static value.

### Exec Handler Scoping
- `-f` file mode works identically to inline mode (both go through same handler)
- `td` module is pre-imported in exec scope
- `_T(name)` helper is available for type lookup
- Variables persist within single exec call only

### Handler Recovery
If handler DAT has compilation errors, ALL POST routes fail (including `dat write`).
Recovery: in TD UI, open `/project1/TDCliServer/handler` DAT and paste content from `td/td_cli_handler.py`.
Alternative: use `td-cli exec` BEFORE the bad handler is pushed to verify syntax with `python3 -c "import py_compile; py_compile.compile('td/td_cli_handler.py', doraise=True)"`
