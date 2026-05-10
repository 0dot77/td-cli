"""Plan B (ink-trail variant): callback-zero-touch directional warp +
visible ink composite, driven by a fading "ink" presence field.

No per-walker velocity is needed. The walker mask (mask_merge) is fed into
a feedback loop that retains and decays walker presence over time. Each
frame the previous field is blurred slightly (diffusion) and noise-warped
(organic bleed). The spatial gradient of the field becomes the UV warp;
the magnitude of the field is also used to subtly desaturate/darken the
base video where ink is dense, so the trail itself is visible.

Pipeline:
  mask_merge ─────────────────────> ink_accum (glslmultiTOP) ── ink_field
                                          ↑                          │
                                          │                          ▼
                                       (input1)               ink_fb (feedbackTOP)
                                          │                          │
                                          └──────────────────────────┘
  ink_accum ─> ink_grad (gradient in RG, ink magnitude in B)
  ink_grad  -> wave_apply.input[2]

wave_apply shader: combines simGrad + ambGrad + warpGrad, and composites a
cool desaturated tint over `col` proportional to ink magnitude.
Sim displace strength (params.x) lowered 0.035 -> 0.022 to balance.

Idempotent.
"""
import td

P = op('/project1')


def pos(o, x, y):
    o.nodeCenterX = x
    o.nodeCenterY = y


# --- 0. cleanup --------------------------------------------------------------
NAMES = ('ink_source',
         'ink_accum', 'ink_accum_pixeldat',
         'ink_fb',
         'ink_grad', 'ink_grad_pixeldat')
for n in NAMES:
    o = P.op(n)
    if o:
        o.destroy()

mask_merge = P.op('mask_merge')
if mask_merge is None:
    raise RuntimeError('mask_merge missing - run td/build_ripple.py first')

# --- 0b. ink_source: independently-blurred ink input -----------------------
# Decoupled from Wave_ripples source so we can dissolve the circle outline
# without softening the wave-sim impulse.
ink_source = P.create(td.blurTOP, 'ink_source')
ink_source.par.size = 50.0
ink_source.inputConnectors[0].connect(mask_merge)
pos(ink_source, -350, -300)

# --- 1. ink_accum: max(current_mask, diffused-and-noise-warped prev * decay) -
ink_accum = P.create(td.glslmultiTOP, 'ink_accum')
ink_accum.par.outputresolution = 'useinput'
ink_accum.par.format = 'rgba16float'
ink_accum.inputConnectors[0].connect(ink_source)

ink_accum.par.vec0name = 'inkparams'
ink_accum.par.vec0valuex = 0.985    # decay per frame (~0.7s halflife)
ink_accum.par.vec0valuey = 0.6      # diffusion (mix sharp vs 5-tap-blurred prev)
ink_accum.par.vec0valuez = 0.005    # noise UV warp magnitude
ink_accum.par.vec0valuew = 0.0
ink_accum.par.vec0valuew.expr = 'absTime.seconds'
ink_accum.par.vec0valuew.mode = ParMode.EXPRESSION
pos(ink_accum, -200, -300)

ink_accum_dat = P.create(td.textDAT, 'ink_accum_pixeldat')
ink_accum_dat.text = '''// Ink presence accumulator with diffusion + curl-noise warp.
// input0 = current walker mask (mask_merge)
// input1 = previous frame of ink_accum (via feedbackTOP)
// output = max(current, diffused-and-noise-warped prev * decay)
out vec4 fragColor;
uniform vec4 inkparams; // x=decay y=diffuse z=noise_warp w=time

float hash21(vec2 p) {
    return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453);
}
float vnoise(vec2 p) {
    vec2 i = floor(p), f = fract(p);
    f = f * f * (3.0 - 2.0 * f);
    return mix(mix(hash21(i),               hash21(i + vec2(1, 0)), f.x),
               mix(hash21(i + vec2(0, 1)),    hash21(i + vec2(1, 1)), f.x), f.y);
}

void main() {
    vec2 uv = vUV.st;
    vec2 ts = 1.0 / vec2(textureSize(sTD2DInputs[1], 0));

    float src = texture(sTD2DInputs[0], uv).r;

    float t = inkparams.w * 0.25;
    vec2 nuv = vec2(
        vnoise(uv * 6.0 + vec2(t, 0.0))         - 0.5,
        vnoise(uv * 6.0 + vec2(0.0, t) + 17.3)  - 0.5
    ) * inkparams.z;
    vec2 sampleUV = uv + nuv;

    float pc = texture(sTD2DInputs[1], sampleUV).r;
    float pl = texture(sTD2DInputs[1], sampleUV + vec2(-ts.x, 0.0)).r;
    float pr = texture(sTD2DInputs[1], sampleUV + vec2( ts.x, 0.0)).r;
    float pu = texture(sTD2DInputs[1], sampleUV + vec2(0.0,  ts.y)).r;
    float pd = texture(sTD2DInputs[1], sampleUV + vec2(0.0, -ts.y)).r;
    float pblur = (pc + pl + pr + pu + pd) * 0.2;

    float prev = mix(pc, pblur, inkparams.y) * inkparams.x;

    float ink = max(src, prev);
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

ink_accum.inputConnectors[1].connect(ink_fb)

# --- 3. ink_grad: gradient (RG) + ink magnitude (B) ----------------------
ink_grad = P.create(td.glslmultiTOP, 'ink_grad')
ink_grad.par.outputresolution = 'useinput'
ink_grad.par.format = 'rgba16float'
ink_grad.inputConnectors[0].connect(ink_accum)

ink_grad.par.vec0name = 'gparams'
ink_grad.par.vec0valuex = 25.0      # gradient gain (max useful range ~ -1..+1)
ink_grad.par.vec0valuey = 0.0
ink_grad.par.vec0valuez = 0.0
ink_grad.par.vec0valuew = 0.0
pos(ink_grad, 100, -300)

ink_grad_dat = P.create(td.textDAT, 'ink_grad_pixeldat')
ink_grad_dat.text = '''// Spatial gradient of the ink field (RG) + magnitude (B).
// wave_apply uses RG as warpGrad and B for the visible ink composite.
out vec4 fragColor;
uniform vec4 gparams; // x=gain

void main() {
    vec2 uv = vUV.st;
    vec2 ts = 1.0 / vec2(textureSize(sTD2DInputs[0], 0));
    float c = texture(sTD2DInputs[0], uv).r;
    float l = texture(sTD2DInputs[0], uv - vec2(ts.x, 0.0)).r;
    float r = texture(sTD2DInputs[0], uv + vec2(ts.x, 0.0)).r;
    float u = texture(sTD2DInputs[0], uv + vec2(0.0, ts.y)).r;
    float d = texture(sTD2DInputs[0], uv - vec2(0.0, ts.y)).r;
    vec2 grad = vec2(r - l, u - d) * 0.5 * gparams.x;
    fragColor = TDOutputSwizzle(vec4(grad, c, 1.0));
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

wa.par.vec0valuex = 0.022     # lowered sim displace strength

# --- 5. wave_apply shader: warpGrad + visible ink composite --------------
apply_dat = P.op('wave_apply_pixeldat')
shader = apply_dat.text

# 5a. read warpGrad alongside (but separate from) grad
# Critical: warpGrad is NOT folded into `grad` because that would inflate the
# length(grad) used by chroma split / depth tint / normal calc, which produces
# extreme RGB-fringing artifacts across the whole frame. Instead we apply
# warpGrad directly to baseDisp below, so the chroma/depth/normal logic stays
# tuned for the small sim+ambient gradient magnitudes.
old_grad = "    // Combined surface gradient\n    vec2 grad = simGrad + ambGrad;"
new_grad = (
    "    // Ink-trail directional warp field (input[2]) - kept separate from grad\n"
    "    // so chroma/depth/normal calculations don't blow up.\n"
    "    vec2 warpGrad = texture(sTD2DInputs[2], uv).rg;\n"
    "    // Combined surface gradient (sim + ambient only - drives chroma/depth/normal)\n"
    "    vec2 grad = simGrad + ambGrad;"
)
if "vec2 warpGrad = texture(sTD2DInputs[2], uv).rg;" not in shader:
    if old_grad in shader:
        shader = shader.replace(old_grad, new_grad, 1)
        print('shader: warpGrad read added (separate from grad)')
    else:
        print('WARNING: grad-combine marker not found')

# 5a-2. apply warpGrad directly to baseDisp
old_disp = "    vec2 baseDisp   = grad * params.x + nDisp;"
new_disp = "    vec2 baseDisp   = grad * params.x + warpGrad * 0.018 + nDisp;"
if "warpGrad * 0.018" in shader:
    pass
elif old_disp in shader:
    shader = shader.replace(old_disp, new_disp, 1)
    print('shader: warpGrad applied to baseDisp at 0.018 scale')
else:
    print('WARNING: baseDisp marker not found')

# 5b. inject visible ink composite right before the final TDOutputSwizzle
old_tail = '''    // --- Depth tint by displacement magnitude ---
    float depthMix = clamp(length(grad) * 30.0, 0.0, 1.0) * params2.x;
    col = mix(col, col * vec3(0.82, 0.95, 1.12), depthMix * videoMask);

    fragColor = TDOutputSwizzle(vec4(col, 1.0));'''
new_tail = '''    // --- Depth tint by displacement magnitude ---
    float depthMix = clamp(length(grad) * 30.0, 0.0, 1.0) * params2.x;
    col = mix(col, col * vec3(0.82, 0.95, 1.12), depthMix * videoMask);

    // --- Visible ink: cool desaturated tint, only where the trail is dense ---
    // smoothstep clips off the soft halo so the circular tracking source
    // never reveals itself - only the saturated core gets tinted.
    float inkRaw = clamp(texture(sTD2DInputs[2], uv).b, 0.0, 1.0);
    float inkAmt = smoothstep(0.18, 0.65, inkRaw);
    float lumIn  = dot(col, vec3(0.299, 0.587, 0.114));
    vec3  inkCol = mix(col, vec3(lumIn) * vec3(0.55, 0.62, 0.78), 0.55);
    col = mix(col, inkCol, inkAmt * 0.4 * videoMask);

    fragColor = TDOutputSwizzle(vec4(col, 1.0));'''
if 'inkAmt' not in shader:
    if old_tail in shader:
        shader = shader.replace(old_tail, new_tail, 1)
        print('shader: visible-ink composite added')
    else:
        print('WARNING: tail marker not found for ink composite')

apply_dat.text = shader

print('Ink-trail layer installed:')
print(f'  decay={ink_accum.par.vec0valuex.eval()} '
      f'diffuse={ink_accum.par.vec0valuey.eval()} '
      f'noise={ink_accum.par.vec0valuez.eval()}')
print(f'  gradient_gain={ink_grad.par.vec0valuex.eval()}')
print(f'  wave_apply.params.x -> {wa.par.vec0valuex.eval()}')
