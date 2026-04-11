import td


ROOT_PATH = __ROOT_PATH__
BASE_NAME = __BASE_NAME__
PREVIEW_NAME = __PREVIEW_NAME__


def pos(op_ref, x, y):
    op_ref.nodeCenterX = x
    op_ref.nodeCenterY = y


def recreate(parent, op_type, name, x, y):
    existing = parent.op(name)
    if existing is not None:
        existing.destroy()
    created = parent.create(op_type, name)
    pos(created, x, y)
    return created


root = op(ROOT_PATH)
if root is None:
    raise Exception("Root operator not found: {}".format(ROOT_PATH))

scene = recreate(root, td.baseCOMP, BASE_NAME, 0, 0)
preview = recreate(root, td.containerCOMP, PREVIEW_NAME, 800, 0)

audio = scene.create(td.audiodeviceinCHOP, "audio_in")
pos(audio, -1800, 700)

select = scene.create(td.selectCHOP, "mono")
select.par.channames = "chan1"
select.inputConnectors[0].connect(audio.outputConnectors[0])
pos(select, -1500, 700)

lowpass = scene.create(td.audiofilterCHOP, "low_band")
lowpass.par.filter = "lowpass"
lowpass.par.cutofffrequency = 220
lowpass.inputConnectors[0].connect(select.outputConnectors[0])
pos(lowpass, -1200, 850)

hipass = scene.create(td.audiofilterCHOP, "high_band")
hipass.par.filter = "hipass"
hipass.par.cutofffrequency = 2400
hipass.inputConnectors[0].connect(select.outputConnectors[0])
pos(hipass, -1200, 550)

env_low = scene.create(td.analyzeCHOP, "env_low")
env_low.par.function = "rms"
env_low.inputConnectors[0].connect(lowpass.outputConnectors[0])
pos(env_low, -900, 850)

env_high = scene.create(td.analyzeCHOP, "env_high")
env_high.par.function = "rms"
env_high.inputConnectors[0].connect(hipass.outputConnectors[0])
pos(env_high, -900, 550)

env_all = scene.create(td.analyzeCHOP, "env_all")
env_all.par.function = "rms"
env_all.inputConnectors[0].connect(select.outputConnectors[0])
pos(env_all, -900, 700)

scale_low = scene.create(td.mathCHOP, "scale_low")
scale_low.par.gain = 6.0
scale_low.par.fromrange1 = 0
scale_low.par.fromrange2 = 0.35
scale_low.par.torange1 = 0
scale_low.par.torange2 = 1
scale_low.inputConnectors[0].connect(env_low.outputConnectors[0])
pos(scale_low, -600, 850)

scale_high = scene.create(td.mathCHOP, "scale_high")
scale_high.par.gain = 6.0
scale_high.par.fromrange1 = 0
scale_high.par.fromrange2 = 0.2
scale_high.par.torange1 = 0
scale_high.par.torange2 = 1
scale_high.inputConnectors[0].connect(env_high.outputConnectors[0])
pos(scale_high, -600, 550)

scale_all = scene.create(td.mathCHOP, "scale_all")
scale_all.par.gain = 4.0
scale_all.par.fromrange1 = 0
scale_all.par.fromrange2 = 0.4
scale_all.par.torange1 = 0
scale_all.par.torange2 = 1
scale_all.inputConnectors[0].connect(env_all.outputConnectors[0])
pos(scale_all, -600, 700)

grid = scene.create(td.gridPOP, "grid")
grid.par.surftype = "triangles"
grid.par.cols = 220
grid.par.rows = 124
grid.par.sizex = 3.6
grid.par.sizey = 2.0
grid.par.normal = "pointNormals"
pos(grid, -180, 760)

noise = scene.create(td.noisePOP, "noise")
noise.inputConnectors[0].connect(grid.outputConnectors[0])
noise.par.type = "simplex4d"
noise.par.period = 0.34
noise.par.harmon = 3
noise.par.spread = 1.85
noise.par.gain = 0.42
noise.par.amp.expr = "0.14 + op('scale_low')['chan1'] * 1.6"
noise.par.t4d.expr = "absTime.seconds * (0.05 + op('scale_high')['chan1'] * 0.24)"
noise.par.ty.expr = "op('scale_all')['chan1'] * 0.05"
noise.par.computenormals = True
pos(noise, 120, 760)

detail = scene.create(td.noisePOP, "detail")
detail.inputConnectors[0].connect(noise.outputConnectors[0])
detail.par.type = "simplex3d"
detail.par.period = 0.12
detail.par.harmon = 2
detail.par.spread = 2.0
detail.par.gain = 0.3
detail.par.amp.expr = "0.03 + op('scale_high')['chan1'] * 0.45"
detail.par.tx.expr = "absTime.seconds * 0.08"
detail.par.rz.expr = "op('scale_high')['chan1'] * 70"
detail.par.computenormals = True
pos(detail, 420, 760)

out_pop = scene.create(td.nullPOP, "out_pop")
out_pop.inputConnectors[0].connect(detail.outputConnectors[0])
pos(out_pop, 720, 760)

render = scene.create(td.rendersimpleTOP, "render")
render.par.pop = out_pop.path
render.par.normalizegeo = True
render.par.bgcolorr = 0.01
render.par.bgcolorg = 0.015
render.par.bgcolorb = 0.03
render.par.bgcolora = 1.0
render.par.wireframe = True
render.par.constantr.expr = "0.06 + op('scale_high')['chan1'] * 0.30"
render.par.constantg.expr = "0.08 + op('scale_all')['chan1'] * 0.24"
render.par.constantb.expr = "0.16 + op('scale_low')['chan1'] * 0.40"
render.par.diffuser.expr = "0.55 + op('scale_low')['chan1'] * 0.25"
render.par.diffuseg.expr = "0.22 + op('scale_all')['chan1'] * 0.55"
render.par.diffuseb.expr = "0.30 + op('scale_high')['chan1'] * 0.55"
render.par.camdistance.expr = "3.6 - op('scale_all')['chan1'] * 0.7"
render.par.georotatex.expr = "62 + op('scale_high')['chan1'] * 18"
render.par.georotatey.expr = "12 + absTime.seconds * 11"
render.par.georotatez.expr = "absTime.seconds * 7"
render.par.geoscale.expr = "1.0 + op('scale_low')['chan1'] * 0.18"
render.par.geotranslatey.expr = "op('scale_all')['chan1'] * 0.12"
pos(render, 1080, 760)

level = scene.create(td.levelTOP, "level")
level.par.brightness1.expr = "0.72 + op('scale_all')['chan1'] * 0.95"
level.par.contrast.expr = "0.18 + op('scale_low')['chan1'] * 0.55"
level.inputConnectors[0].connect(render.outputConnectors[0])
pos(level, 1380, 760)

glow = scene.create(td.blurTOP, "glow")
glow.par.size.expr = "2.0 + op('scale_low')['chan1'] * 11.0"
glow.inputConnectors[0].connect(level.outputConnectors[0])
pos(glow, 1680, 620)

comp = scene.create(td.compositeTOP, "comp")
comp.par.operand = "screen"
comp.inputConnectors[0].connect(level.outputConnectors[0])
comp.inputConnectors[1].connect(glow.outputConnectors[0])
pos(comp, 1680, 820)

out_top = scene.create(td.nullTOP, "out")
out_top.inputConnectors[0].connect(comp.outputConnectors[0])
pos(out_top, 1980, 820)

preview.par.top = scene.path + "/out"
preview.color = (0.18, 0.24, 0.38)

print("Created POP audio visual:", scene.path)
print("Preview container:", preview.path)
print("Output TOP:", out_top.path)
