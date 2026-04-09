root = op('/project1')
existing = op('/project1/mouseParticlesSOP')
if existing is not None:
    existing.destroy()

comp = root.create('containerCOMP', 'mouseParticlesSOP')
comp.nodeX = 300
comp.nodeY = -150
comp.par.w = 900
comp.par.h = 700
comp.par.display = 1
comp.par.bgcolorr = 0.02
comp.par.bgcolorg = 0.02
comp.par.bgcolorb = 0.03
comp.par.bgalpha = 1
comp.color = (0.18, 0.24, 0.34)
comp.comment = 'Mouse-tracked SOP particle trail. Move the mouse over the container viewer.'


def add(parent, op_type, name, x, y):
    node = parent.create(op_type, name)
    node.nodeX = x
    node.nodeY = y
    return node


panel1 = add(comp, 'panelCHOP', 'panel1', -800, -100)
math1 = add(comp, 'mathCHOP', 'math1', -600, -100)
constant1 = add(comp, 'constantCHOP', 'constant1', -600, 60)
merge1 = add(comp, 'mergeCHOP', 'merge1', -400, -20)
trail1 = add(comp, 'trailCHOP', 'trail1', -200, -20)
phong1 = add(comp, 'phongMAT', 'phong1', 140, 220)
geo1 = add(comp, 'geometryCOMP', 'geo1', 160, -40)
cam1 = add(comp, 'cameraCOMP', 'cam1', 380, -180)
light1 = add(comp, 'lightCOMP', 'light1', 360, 60)
render1 = add(comp, 'renderTOP', 'render1', 560, -40)
out1 = add(comp, 'nullTOP', 'out1', 760, -40)

default_torus = op(f'{geo1.path}/torus1')
if default_torus is not None:
    default_torus.destroy()

chopto1 = add(geo1, 'choptoSOP', 'chopto1', -420, -60)
sphere1 = add(geo1, 'sphereSOP', 'sphere1', -620, -180)
copy1 = add(geo1, 'copySOP', 'copy1', -220, -100)

panel1.par.component = comp.path
panel1.par.select = 'u v'
panel1.par.rename = 'tx ty'

math1.inputConnectors[0].connect(panel1.outputConnectors[0])
math1.par.preoff = -0.5
math1.par.gain = 2.0
math1.par.postoff = 0.0

constant1.par.const = 1
constant1.par.const0name = 'tz'
constant1.par.const0value = 0.0

math1.outputConnectors[0].connect(merge1.inputConnectors[0])
constant1.outputConnectors[0].connect(merge1.inputConnectors[1])

merge1.outputConnectors[0].connect(trail1.inputConnectors[0])
trail1.par.active = 1
trail1.par.wlength = 160
trail1.par.capture = 1
trail1.par.resetpulse.pulse()

chopto1.par.chop = trail1.path
chopto1.par.mapping = 'onetoone'

sphere1.par.radx = 0.03
sphere1.par.rady = 0.03
sphere1.par.radz = 0.03
sphere1.par.rows = 6
sphere1.par.cols = 8

sphere1.outputConnectors[0].connect(copy1.inputConnectors[0])
chopto1.outputConnectors[0].connect(copy1.inputConnectors[1])
copy1.display = True
copy1.render = True

geo1.par.material = phong1.path
geo1.par.tx = 0
geo1.par.ty = 0
geo1.par.tz = 0

phong1.par.diffr = 0.16
phong1.par.diffg = 0.82
phong1.par.diffb = 1.0
phong1.par.emitr = 0.04
phong1.par.emitg = 0.18
phong1.par.emitb = 0.28
phong1.par.specr = 0.85
phong1.par.specg = 0.95
phong1.par.specb = 1.0
phong1.par.shininess = 24

cam1.par.ty = 0.1
cam1.par.tz = 6.5
light1.par.ty = 1.5
light1.par.tz = 4.5
light1.par.rx = -18

render1.par.camera = cam1.name
render1.par.geometry = geo1.name
render1.par.lights = light1.name
render1.par.bgcolorr = 0.015
render1.par.bgcolorg = 0.015
render1.par.bgcolorb = 0.02
render1.par.bgcolora = 1

render1.outputConnectors[0].connect(out1.inputConnectors[0])
comp.par.top = out1.path

print({
    'component': comp.path,
    'panel': panel1.path,
    'trail': trail1.path,
    'render': render1.path,
    'output': out1.path,
})
