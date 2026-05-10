"""Plan B (ink-trail variant): callback-zero-touch directional warp via
a fading "ink" presence field driven by feedbackTOP.

No per-walker velocity is needed. The walker mask (mask_merge, already used
to drive Wave_ripples) is fed into a feedback loop that retains and decays
walker presence over time. The spatial gradient of that decaying field is
used as the UV displacement for wave_apply: the gradient naturally points
along the trail (from old/faded toward fresh/walker), giving an organic
ink-bleed direction without any per-frame velocity tracking.

Pipeline:
  mask_merge ─────────────────────> ink_accum (glslmultiTOP) ── ink_field
                                          ↑                          │
                                          │                          ▼
                                       (input1)               ink_fb (feedbackTOP)
                                          │                          │
                                          └──────────────────────────┘
  ink_accum ─> ink_grad (glslmultiTOP, central-difference gradient)
  ink_grad  -> wave_apply.input[2]

Shader change in wave_apply: grad = simGrad + ambGrad + warpGrad.
Sim displace strength (params.x) lowered 0.035 -> 0.022 to balance.

Idempotent. Safe to re-run.
"""
import td

P = op('/project1')


def pos(o, x, y):
    o.nodeCenterX = x
    o.nodeCenterY = y


# --- 0. cleanup --------------------------------------------------------------
NAMES = ('ink_accum', 'ink_accum_pixeldat',
         'ink_fb',
         'ink_grad', 'ink_grad_pixeldat')
for n in NAMES:
    o = P.op(n)
    if o:
        o.destroy()

mask_merge = P.op('mask_merge')
if mask_merge is None:
    raise RuntimeError('mask_merge missing - run td/build_ripple.py first')

# --- 1. ink_accum: max(current_mask, prev_field * decay) -------------------
ink_accum = P.create(td.glslmultiTOP, 'ink_accum')
ink_accum.par.outputresolution = 'useinput'
ink_accum.par.format = 'rgba16float'
ink_accum.inputConnectors[0].connect(mask_merge)

ink_accum.par.vec0name = 'inkparams'
ink_accum.par.vec0valuex = 0.992    # decay per frame (~1.5s halflife at 60fps)
ink_accum.par.vec0valuey = 0.0
ink_accum.par.vec0valuez = 0.0
ink_accum.par.vec0valuew = 0.0
pos(ink_accum, -200, -300)

ink_accum_dat = P.create(td.textDAT, 'ink_accum_pixeldat')
ink_accum_dat.text = '''// Ink presence accumulator.
// input0 = current walker mask (mask_merge)
// input1 = previous frame of ink_accum output (via feedbackTOP)
// output = max(input0, input1 * decay)
out vec4 fragColor;
uniform vec4 inkparams; // x=decay

void main() {
    vec2 uv = vUV.st;
    float src  = texture(sTD2DInputs[0], uv).r;
    float prev = texture(sTD2DInputs[1], uv).r * inkparams.x;
    float ink  = max(src, prev);
    fragColor = TDOutputSwizzle(vec4(ink, ink, ink, 1.0));
}
'''
ink_accum.par.pixeldat = ink_accum_dat.path
pos(ink_accum_dat, -200, -150)

# --- 2. ink_fb: feedbackTOP holding previous frame of ink_accum -----------
ink_fb = P.create(td.feedbackTOP, 'ink_fb')
ink_fb.par.top = ink_accum.path
ink_fb.inputConnectors[0].connect(ink_accum)
pos(ink_fb, -50, -300)

# Wire feedback back into ink_accum.input[1]
ink_accum.inputConnectors[1].connect(ink_fb)

# --- 3. ink_grad: spatial gradient of accumulator (central difference) ----
ink_grad = P.create(td.glslmultiTOP, 'ink_grad')
ink_grad.par.outputresolution = 'useinput'
ink_grad.par.format = 'rgba16float'
ink_grad.inputConnectors[0].connect(ink_accum)

ink_grad.par.vec0name = 'gparams'
ink_grad.par.vec0valuex = 4.0       # gain
ink_grad.par.vec0valuey = 0.0
ink_grad.par.vec0valuez = 0.0
ink_grad.par.vec0valuew = 0.0
pos(ink_grad, 100, -300)

ink_grad_dat = P.create(td.textDAT, 'ink_grad_pixeldat')
ink_grad_dat.text = '''// Spatial gradient of the ink presence field.
// Output (RG) is a UV displacement vector pointing from low-ink (fading
// trail) toward high-ink (current walker). wave_apply will add this to
// its grad accumulator.
out vec4 fragColor;
uniform vec4 gparams; // x=gain

void main() {
    vec2 uv = vUV.st;
    vec2 ts = 1.0 / vec2(textureSize(sTD2DInputs[0], 0));
    float l = texture(sTD2DInputs[0], uv - vec2(ts.x, 0.0)).r;
    float r = texture(sTD2DInputs[0], uv + vec2(ts.x, 0.0)).r;
    float u = texture(sTD2DInputs[0], uv + vec2(0.0, ts.y)).r;
    float d = texture(sTD2DInputs[0], uv - vec2(0.0, ts.y)).r;
    vec2 grad = vec2(r - l, u - d) * 0.5 * gparams.x;
    fragColor = TDOutputSwizzle(vec4(grad, 0.0, 1.0));
}
'''
ink_grad.par.pixeldat = ink_grad_dat.path
pos(ink_grad_dat, 100, -150)

# --- 4. wire ink_grad into wave_apply.input[2] ----------------------------
wa = P.op('wave_apply')
if wa is None:
    raise RuntimeError('wave_apply missing - run td/build_ripple.py first')

wa.inputConnectors[2].disconnect()
ink_grad.outputConnectors[0].connect(wa.inputConnectors[2])

# lower sim strength to balance with the new warp layer
wa.par.vec0valuex = 0.022

# --- 5. update wave_apply shader: add warpGrad ----------------------------
apply_dat = P.op('wave_apply_pixeldat')
shader = apply_dat.text
old_marker = "    // Combined surface gradient\n    vec2 grad = simGrad + ambGrad;"
new_block = (
    "    // Ink-trail directional warp field (input[2])\n"
    "    vec2 warpGrad = texture(sTD2DInputs[2], uv).rg;\n"
    "    // Combined surface gradient\n"
    "    vec2 grad = simGrad + ambGrad + warpGrad;"
)
already = "vec2 warpGrad = texture(sTD2DInputs[2], uv).rg;"
if already in shader:
    print('shader already has warpGrad; skipping rewrite')
elif old_marker in shader:
    apply_dat.text = shader.replace(old_marker, new_block, 1)
    print('shader updated: warpGrad added to grad combine')
else:
    print('WARNING: grad-combine marker not found; wave_apply shader unchanged')

print('Ink-trail layer installed:')
print(f'  decay={ink_accum.par.vec0valuex.eval()} '
      f'gradient_gain={ink_grad.par.vec0valuex.eval()}')
print(f'  wave_apply.params.x -> {wa.par.vec0valuex.eval()}')
