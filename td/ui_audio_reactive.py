"""
UI + Audio-Reactive Scene
=========================
Builds a complete audio-reactive wireframe mesh with interactive UI controls.

Structure:
  ctrl (baseCOMP)     — custom parameters: Audio / Visual / Post pages
  ui (parameterCOMP)  — auto-generated interactive panel from ctrl
  ui_window           — floating interactive window
  Audio chain         — audiodevicein → filter(bass/mid/high) → analyze → lag → math
  Geo (SOP-in-geo)    — gridSOP → noiseSOP → nullSOP (display+render flags)
  Render              — renderTOP with camera lookat, wireframe material
  Post                — level → glow → feedback trail → composite → out

Key patterns applied:
  - NO pathsop on geometryCOMP (causes cook dependency self-loop)
  - Display/render flags explicitly set on output SOP
  - Absolute paths in expressions for SOPs inside COMPs
  - parameterCOMP for UI (most reliable, auto-interactive)
  - windowCOMP for guaranteed mouse interaction
"""
import td


def pos(o, x, y):
    o.nodeCenterX = x
    o.nodeCenterY = y


def slider(page, name, label, val, lo, hi):
    pa = page.appendFloat(name, label=label)
    pa[0].default = val
    pa[0].val = val
    pa[0].min = lo
    pa[0].max = hi
    pa[0].clampMin = True
    pa[0].clampMax = True
    return pa


# ── Clean slate ──────────────────────────────────────────────
p = op("/project1")
for c in p.findChildren(depth=1):
    if c.name != "TDCliServer":
        c.destroy()


# ═════════════════════════════════════════════════════════════
# 1. CONTROL PANEL
# ═════════════════════════════════════════════════════════════
ctrl = p.create(td.baseCOMP, "ctrl")
pos(ctrl, -2400, 0)

pg_audio = ctrl.appendCustomPage("Audio")
slider(pg_audio, "Bassgain",  "Bass Gain",  5.0,  0.0, 30.0)
slider(pg_audio, "Midgain",   "Mid Gain",   10.0, 0.0, 30.0)
slider(pg_audio, "Highgain",  "High Gain",  20.0, 0.0, 50.0)
slider(pg_audio, "Lagtime",   "Smoothing",  0.15, 0.01, 1.0)

pg_vis = ctrl.appendCustomPage("Visual")
slider(pg_vis, "Noiseamp",   "Noise Amplitude",  1.5, 0.0, 5.0)
slider(pg_vis, "Noisespeed", "Noise Speed",      0.5, 0.0, 3.0)
slider(pg_vis, "Rotspeed",   "Rotation Speed",   5.0, 0.0, 30.0)
slider(pg_vis, "Meshscale",  "Mesh Scale",       1.0, 0.1, 3.0)
slider(pg_vis, "Colorr",     "Base Red",         0.9, 0.0, 1.0)
slider(pg_vis, "Colorg",     "Base Green",       0.2, 0.0, 1.0)
slider(pg_vis, "Colorb",     "Base Blue",        0.05, 0.0, 1.0)

pg_post = ctrl.appendCustomPage("Post")
slider(pg_post, "Glowsize",   "Glow Size",   6.0,  0.0, 30.0)
slider(pg_post, "Glowmix",    "Glow Mix",    0.3,  0.0, 1.0)
slider(pg_post, "Trailfade",  "Trail Fade",  0.7,  0.0, 0.99)
slider(pg_post, "Brightness", "Brightness",  1.0,  0.5, 2.0)
slider(pg_post, "Contrast",   "Contrast",    0.5,  0.0, 1.5)


# ═════════════════════════════════════════════════════════════
# 2. UI PANEL (parameterCOMP — auto-interactive)
# ═════════════════════════════════════════════════════════════
ui = p.create(td.parameterCOMP, "ui")
pos(ui, -2400, 500)
ui.par.op = ctrl.path
ui.par.header = False
ui.par.pagenames = True
ui.par.labels = True
ui.par.builtin = False
ui.par.custom = True
ui.par.compress = 0.85
ui.par.w = 350
ui.par.h = 550
ui.par.bgcolorr = 0.1
ui.par.bgcolorg = 0.1
ui.par.bgcolorb = 0.12
ui.par.bgalpha = 1.0

win = p.create(td.windowCOMP, "ui_window")
pos(win, -2100, 500)
win.par.winop = ui.path
win.par.winw = 350
win.par.winh = 550
win.par.borders = True
win.par.title = "Audio Reactive Controls"
win.par.winopen = True


# ═════════════════════════════════════════════════════════════
# 3. AUDIO PIPELINE
# ═════════════════════════════════════════════════════════════
audio = p.create(td.audiodeviceinCHOP, "audio_in")
pos(audio, -1800, 600)

sel = p.create(td.selectCHOP, "sel_mono")
sel.par.channames = "chan1"
sel.inputConnectors[0].connect(audio.outputConnectors[0])
pos(sel, -1500, 600)

# Bass
filt_lo = p.create(td.audiofilterCHOP, "filt_bass")
filt_lo.par.filter = "lowpass"
filt_lo.par.cutofffrequency = 200
filt_lo.inputConnectors[0].connect(sel.outputConnectors[0])
pos(filt_lo, -1200, 750)

ana_lo = p.create(td.analyzeCHOP, "ana_bass")
ana_lo.par.function = "rmspower"
ana_lo.inputConnectors[0].connect(filt_lo.outputConnectors[0])
pos(ana_lo, -900, 750)

lag_lo = p.create(td.lagCHOP, "lag_bass")
lag_lo.par.lag1.expr = "op('ctrl').par.Lagtime"
lag_lo.par.lag2.expr = "op('ctrl').par.Lagtime"
lag_lo.inputConnectors[0].connect(ana_lo.outputConnectors[0])
pos(lag_lo, -600, 750)

math_lo = p.create(td.mathCHOP, "math_bass")
math_lo.par.gain.expr = "op('ctrl').par.Bassgain"
math_lo.inputConnectors[0].connect(lag_lo.outputConnectors[0])
pos(math_lo, -300, 750)

# Mid
filt_mid = p.create(td.audiofilterCHOP, "filt_mid")
filt_mid.par.filter = "bandpass"
filt_mid.par.cutofffrequency = 1000
filt_mid.inputConnectors[0].connect(sel.outputConnectors[0])
pos(filt_mid, -1200, 600)

ana_mid = p.create(td.analyzeCHOP, "ana_mid")
ana_mid.par.function = "rmspower"
ana_mid.inputConnectors[0].connect(filt_mid.outputConnectors[0])
pos(ana_mid, -900, 600)

lag_mid = p.create(td.lagCHOP, "lag_mid")
lag_mid.par.lag1.expr = "op('ctrl').par.Lagtime"
lag_mid.par.lag2.expr = "op('ctrl').par.Lagtime"
lag_mid.inputConnectors[0].connect(ana_mid.outputConnectors[0])
pos(lag_mid, -600, 600)

math_mid = p.create(td.mathCHOP, "math_mid")
math_mid.par.gain.expr = "op('ctrl').par.Midgain"
math_mid.inputConnectors[0].connect(lag_mid.outputConnectors[0])
pos(math_mid, -300, 600)

# High
filt_hi = p.create(td.audiofilterCHOP, "filt_high")
filt_hi.par.filter = "hipass"
filt_hi.par.cutofffrequency = 3000
filt_hi.inputConnectors[0].connect(sel.outputConnectors[0])
pos(filt_hi, -1200, 450)

ana_hi = p.create(td.analyzeCHOP, "ana_high")
ana_hi.par.function = "rmspower"
ana_hi.inputConnectors[0].connect(filt_hi.outputConnectors[0])
pos(ana_hi, -900, 450)

lag_hi = p.create(td.lagCHOP, "lag_high")
lag_hi.par.lag1.expr = "op('ctrl').par.Lagtime"
lag_hi.par.lag2.expr = "op('ctrl').par.Lagtime"
lag_hi.inputConnectors[0].connect(ana_hi.outputConnectors[0])
pos(lag_hi, -600, 450)

math_hi = p.create(td.mathCHOP, "math_high")
math_hi.par.gain.expr = "op('ctrl').par.Highgain"
math_hi.inputConnectors[0].connect(lag_hi.outputConnectors[0])
pos(math_hi, -300, 450)


# ═════════════════════════════════════════════════════════════
# 4. GEOMETRY (SOP-inside-geo, NO pathsop)
# ═════════════════════════════════════════════════════════════
mat = p.create(td.constantMAT, "mat_main")
mat.par.colorr.expr = "op('/project1/ctrl').par.Colorr + op('/project1/math_bass')['chan1'] * 0.3"
mat.par.colorg.expr = "op('/project1/ctrl').par.Colorg + op('/project1/math_mid')['chan1'] * 0.4"
mat.par.colorb.expr = "op('/project1/ctrl').par.Colorb + op('/project1/math_high')['chan1'] * 0.5"
mat.par.wireframe = "on"
mat.par.wirewidth = 2.0
pos(mat, 200, 200)

geo = p.create(td.geometryCOMP, "geo_main")
pos(geo, 600, 500)
for child in list(geo.findChildren(depth=1)):
    child.destroy()

grid = geo.create(td.gridSOP, "grid")
grid.par.sizex = 5.0
grid.par.sizey = 5.0
grid.par.rows = 80
grid.par.cols = 80

noise = geo.create(td.noiseSOP, "noise")
noise.par.amp.expr = "op('/project1/ctrl').par.Noiseamp * (0.3 + op('/project1/math_bass')['chan1'] * 1.0)"
noise.par.spread.expr = "0.3 + op('/project1/math_mid')['chan1'] * 0.3"
noise.par.period.expr = "2.0 + op('/project1/math_high')['chan1'] * 3.0"
noise.inputConnectors[0].connect(grid.outputConnectors[0])

null_out = geo.create(td.nullSOP, "out")
null_out.inputConnectors[0].connect(noise.outputConnectors[0])
null_out.display = True
null_out.render = True

geo.par.material = mat
geo.par.ry.expr = "absTime.seconds * op('/project1/ctrl').par.Rotspeed"
geo.par.sx.expr = "op('/project1/ctrl').par.Meshscale * (1.0 + op('/project1/math_bass')['chan1'] * 0.2)"
geo.par.sy.expr = "op('/project1/ctrl').par.Meshscale * (1.0 + op('/project1/math_bass')['chan1'] * 0.2)"
geo.par.sz.expr = "op('/project1/ctrl').par.Meshscale * (1.0 + op('/project1/math_bass')['chan1'] * 0.2)"

cam = p.create(td.cameraCOMP, "cam")
cam.par.tx = 0
cam.par.ty = 2
cam.par.tz = 8
cam.par.lookat = geo
pos(cam, 600, 750)

light1 = p.create(td.lightCOMP, "light1")
light1.par.tx = 5
light1.par.ty = 5
light1.par.tz = 5
light1.par.dimmer = 1.0
pos(light1, 600, 300)

light2 = p.create(td.lightCOMP, "light2")
light2.par.tx = -4
light2.par.ty = -2
light2.par.tz = 4
light2.par.dimmer = 0.5
light2.par.cr = 0.3
light2.par.cg = 0.2
light2.par.cb = 1.0
pos(light2, 600, 150)

render = p.create(td.renderTOP, "render")
render.par.camera = cam
render.par.geometry = geo
render.par.lights = "{} {}".format(light1.path, light2.path)
render.par.bgcolora = 1.0
pos(render, 900, 500)


# ═════════════════════════════════════════════════════════════
# 5. POST-PROCESSING
# ═════════════════════════════════════════════════════════════
level = p.create(td.levelTOP, "level")
level.par.brightness1.expr = "op('ctrl').par.Brightness"
level.par.contrast.expr = "op('ctrl').par.Contrast"
level.inputConnectors[0].connect(render.outputConnectors[0])
pos(level, 1200, 500)

glow_blur = p.create(td.blurTOP, "glow_blur")
glow_blur.par.size.expr = "op('ctrl').par.Glowsize + op('math_bass')['chan1'] * 5.0"
glow_blur.inputConnectors[0].connect(level.outputConnectors[0])
pos(glow_blur, 1500, 350)

glow_level = p.create(td.levelTOP, "glow_level")
glow_level.par.opacity.expr = "op('ctrl').par.Glowmix"
glow_level.inputConnectors[0].connect(glow_blur.outputConnectors[0])
pos(glow_level, 1800, 350)

glow_comp = p.create(td.compositeTOP, "glow_comp")
glow_comp.par.operand = "add"
glow_comp.inputConnectors[0].connect(level.outputConnectors[0])
glow_comp.inputConnectors[1].connect(glow_level.outputConnectors[0])
pos(glow_comp, 2100, 500)

fb = p.create(td.feedbackTOP, "fb")
fb.inputConnectors[0].connect(glow_comp.outputConnectors[0])
fb.par.top = glow_comp
pos(fb, 2400, 350)

fb_fade = p.create(td.levelTOP, "fb_fade")
fb_fade.par.opacity.expr = "op('ctrl').par.Trailfade"
fb_fade.inputConnectors[0].connect(fb.outputConnectors[0])
pos(fb_fade, 2700, 350)

trail_comp = p.create(td.compositeTOP, "trail_comp")
trail_comp.par.operand = "over"
trail_comp.inputConnectors[0].connect(glow_comp.outputConnectors[0])
trail_comp.inputConnectors[1].connect(fb_fade.outputConnectors[0])
pos(trail_comp, 2700, 500)

out = p.create(td.nullTOP, "out")
out.inputConnectors[0].connect(trail_comp.outputConnectors[0])
pos(out, 3000, 500)


# ═════════════════════════════════════════════════════════════
# VERIFY
# ═════════════════════════════════════════════════════════════
s = render.sample(x=640, y=360)
loop = "loop" in (geo.warnings() or "").lower()
nodes = len(p.findChildren(depth=1))
print("Render: {} | Loop: {} | Nodes: {}".format(
    "OK" if s[3] > 0 else "empty", loop, nodes))
