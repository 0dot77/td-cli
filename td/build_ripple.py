"""Build wave-equation ripple effect using user's Wave_ripples component.

Pipeline:
  geo2 (video plane) -> render1 (video only)
  geo1 (circle instances) -> render_circles (mask, same cam, 1280x720)
                          -> mask_blur -> mask_merge (overlapping circles unified)
                          -> Wave_ripples (Stas Sumarokov / Hugo Elias water sim)
                          -> wave_apply (UV displacement on render1)
"""
import td

PARENT = op('/project1')
P = PARENT


def pos(o, x, y):
    o.nodeCenterX = x
    o.nodeCenterY = y


# ---- 0. cleanup if rerunning ----
for name in ('render_circles', 'mask_blur', 'mask_merge',
             'wave_sim', 'wave_fb', 'wave_apply',
             'wave_sim_pixeldat', 'wave_apply_pixeldat',
             'mat_white_ripple', 'ripple_out'):
    o = P.op(name)
    if o:
        o.destroy()

# ---- 1. render1 shows only the video plane ----
r1 = P.op('render1')
r1.par.geometry = 'geo2'

# ---- 2. white material for circle mask ----
mat_w = P.create(td.constantMAT, 'mat_white_ripple')
mat_w.par.colorr = 1.0
mat_w.par.colorg = 1.0
mat_w.par.colorb = 1.0
mat_w.par.alpha = 1.0
pos(mat_w, -1100, -300)

# ---- 3. render_circles: same camera, only circles, white on black ----
# Match Wave_ripples uRes (1280x720) so coords align with displacement target.
rc = P.create(td.renderTOP, 'render_circles')
rc.par.camera = 'cam1'
rc.par.geometry = 'geo1'
rc.par.lights = ''
rc.par.bgcolorr = 0.0
rc.par.bgcolorg = 0.0
rc.par.bgcolorb = 0.0
rc.par.bgcolora = 1.0
rc.par.overridemat = mat_w.path
rc.par.resolutionw = 640      # half-res for sim chain (perf: 30fps budget)
rc.par.resolutionh = 360
rc.par.outputresolution = 'custom'
rc.par.antialias = 'aa4'
rc.par.rendermode = 'render2d'
pos(rc, -800, -300)

# ---- 3b. blur + soft threshold: merge overlapping circles into one source ----
mask_blur = P.create(td.blurTOP, 'mask_blur')
mask_blur.par.size = 40.0
mask_blur.inputConnectors[0].connect(rc)
pos(mask_blur, -600, -300)

mask_merge = P.create(td.levelTOP, 'mask_merge')
mask_merge.par.blacklevel = 0.08
mask_merge.par.brightness1 = 0.85   # softer source impulse -> gentler waves
mask_merge.inputConnectors[0].connect(mask_blur)
pos(mask_merge, -440, -300)

# ---- 4. feed the merged mask into the existing Wave_ripples component ----
wr = P.op('Wave_ripples')
if wr is None:
    raise RuntimeError('Wave_ripples component not found at /project1/Wave_ripples')
# Connect external input 0 (the in TOP "MovementInput") to mask_merge
wr.inputConnectors[0].connect(mask_merge)
# Tune the comp's params for our scale
wr.par.Damping = 0.995   # waves linger longer
wr.par.Npasses = 1       # slower propagation (calmer, less dizzying)

# ---- 5. wave_apply: displace render1 UV by Wave_ripples gradient ----
wave_apply = P.create(td.glslmultiTOP, 'wave_apply')
wave_apply.par.outputresolution = 'useinput'
wave_apply.par.format = 'rgba8fixed'
pos(wave_apply, 300, -300)

wave_apply.inputConnectors[0].connect(r1)
# Container COMP output connects via its output side
wr.outputConnectors[0].connect(wave_apply.inputConnectors[1])

# params:  x=displaceStrength, y=chromaSplit, z=specStrength, w=causticsStrength
wave_apply.par.vec0name = 'params'
wave_apply.par.vec0valuex = 0.04    # less UV displacement -> easier to read video while walking
wave_apply.par.vec0valuey = 0.35
wave_apply.par.vec0valuez = 1.0
wave_apply.par.vec0valuew = 0.6

# params2: x=tintStrength, y=lightDirX, z=lightDirY, w=normalScale
wave_apply.par.vec1name = 'params2'
wave_apply.par.vec1valuex = 0.55
wave_apply.par.vec1valuey = -0.5
wave_apply.par.vec1valuez = 0.6
wave_apply.par.vec1valuew = 80.0

# noiseP: x=strength y=scale z=speed w=time(absTime.seconds)
wave_apply.par.vec2name = 'noiseP'
wave_apply.par.vec2valuex = 0.006   # very subtle shimmer
wave_apply.par.vec2valuey = 28.0
wave_apply.par.vec2valuez = 0.7
wave_apply.par.vec2valuew.expr = 'absTime.seconds'
try:
    wave_apply.par.vec2valuew.mode = ParMode.EXPRESSION  # noqa: F821
except NameError:
    import tdu
    wave_apply.par.vec2valuew.mode = tdu.ParMode.EXPRESSION

# ambient: x=amplitude y=scale z=speed w=fresnelStrength
wave_apply.par.vec3name = 'ambient'
wave_apply.par.vec3valuex = 0.05    # ambient FBM height amplitude
wave_apply.par.vec3valuey = 4.0     # spatial scale (low freq -> wide swells)
wave_apply.par.vec3valuez = 0.12    # very slow drift
wave_apply.par.vec3valuew = 0.55    # fresnel reflection strength

# sky: xyz=skyTint w=bloomStrength
wave_apply.par.vec4name = 'sky'
wave_apply.par.vec4valuex = 0.35    # cool moonlit blue
wave_apply.par.vec4valuey = 0.50
wave_apply.par.vec4valuez = 0.78
wave_apply.par.vec4valuew = 0.85    # soft bloom intensity

apply_dat = P.create(td.textDAT, 'wave_apply_pixeldat')
apply_dat.text = '''// Poetic water shader: ambient FBM swell + Fresnel sky + soft bloom
// On top of: chromatic aberration, specular (sharp+halo), caustics, depth tint, shimmer noise
out vec4 fragColor;
uniform vec4 params;   // x=displace y=chromaSplit z=specStrength w=causticsStrength
uniform vec4 params2;  // x=tintStrength y=lightDirX z=lightDirY w=normalScale
uniform vec4 noiseP;   // x=strength y=scale z=speed w=time
uniform vec4 ambient;  // x=amplitude y=scale z=speed w=fresnelStrength
uniform vec4 sky;      // xyz=skyTint w=bloomStrength

float hash21(vec2 p) {
    return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453);
}
float vnoise(vec2 p) {
    vec2 i = floor(p), f = fract(p);
    f = f * f * (3.0 - 2.0 * f);
    float a = hash21(i);
    float b = hash21(i + vec2(1.0, 0.0));
    float c = hash21(i + vec2(0.0, 1.0));
    float d = hash21(i + vec2(1.0, 1.0));
    return mix(mix(a, b, f.x), mix(c, d, f.x), f.y);
}
float fbm(vec2 p, float t) {
    // 2 octaves for perf (from 3) - keeps the swell character at much lower cost
    float v = 0.5 * vnoise(p + vec2(t, t * 0.7));
    p *= 2.07;
    v += 0.275 * vnoise(p + vec2(t * 1.6, t * 1.1));
    return v;
}

void main() {
    vec2 uv = vUV.st;
    vec2 ts = 1.0 / vec2(textureSize(sTD2DInputs[1], 0));

    // --- Sim wave height & gradient & Laplacian ---
    float c  = texture(sTD2DInputs[1], uv).r - 0.5;
    float hl = texture(sTD2DInputs[1], uv - vec2(ts.x, 0.0)).r - 0.5;
    float hr = texture(sTD2DInputs[1], uv + vec2(ts.x, 0.0)).r - 0.5;
    float hu = texture(sTD2DInputs[1], uv + vec2(0.0, ts.y)).r - 0.5;
    float hd = texture(sTD2DInputs[1], uv - vec2(0.0, ts.y)).r - 0.5;
    vec2  simGrad = vec2(hr - hl, hu - hd);
    float lap     = (hl + hr + hu + hd) - 4.0 * c;

    // --- Ambient FBM swell (no feedback into sim, computed analytically) ---
    float at = noiseP.w * ambient.z;
    vec2  ap = uv * ambient.y;
    float dh = 0.5 / max(ambient.y, 0.001);
    float a_l = fbm(vec2(ap.x - dh, ap.y), at);
    float a_r = fbm(vec2(ap.x + dh, ap.y), at);
    float a_u = fbm(vec2(ap.x, ap.y + dh), at);
    float a_d = fbm(vec2(ap.x, ap.y - dh), at);
    vec2 ambGrad = vec2(a_r - a_l, a_u - a_d) * ambient.x;

    // Combined surface gradient
    vec2 grad = simGrad + ambGrad;

    // Surface normal
    vec3 N = normalize(vec3(-grad * params2.w, 1.0));

    // Video presence mask
    vec4 orig = texture(sTD2DInputs[0], uv);
    float lum = dot(orig.rgb, vec3(0.299, 0.587, 0.114));
    float videoMask = smoothstep(0.02, 0.15, lum);

    // --- High-freq shimmer (active where waves exist) ---
    vec2 nuv = uv * noiseP.y;
    float t  = noiseP.w * noiseP.z;
    vec2 nDisp = vec2(
        vnoise(nuv + vec2(t,        0.0))         - 0.5,
        vnoise(nuv + vec2(0.0, t * 0.83) + 17.3)  - 0.5
    ) * noiseP.x;
    float waveActivity = clamp(length(simGrad) * 80.0, 0.0, 1.0);
    nDisp *= mix(0.15, 1.0, waveActivity);

    // --- Chromatic aberration on displacement ---
    vec2 baseDisp   = grad * params.x + nDisp;
    float chromaOff = params.y * length(grad) * 2.0;
    vec2 uvR = clamp(uv + baseDisp * (1.0 + chromaOff), 0.0, 1.0);
    vec2 uvG = clamp(uv + baseDisp,                       0.0, 1.0);
    vec2 uvB = clamp(uv + baseDisp * (1.0 - chromaOff), 0.0, 1.0);
    vec3 col = vec3(
        texture(sTD2DInputs[0], uvR).r,
        texture(sTD2DInputs[0], uvG).g,
        texture(sTD2DInputs[0], uvB).b
    );

    // --- Fresnel sky reflection (Schlick) ---
    vec3 V = vec3(0.0, 0.0, 1.0);
    float NdotV   = max(dot(N, V), 0.0);
    float fresnel = mix(0.02, 1.0, pow(1.0 - NdotV, 4.0));
    // Simple gradient sky: deeper at "horizon" (where N tilts), brighter overhead
    vec3 skyCol = mix(sky.xyz * 0.45, sky.xyz, smoothstep(-1.0, 1.0, N.y));
    col = mix(col, col * 0.7 + skyCol * 0.6, fresnel * ambient.w * videoMask);

    // --- Specular: sharp highlight + soft halo (cheap fake bloom) ---
    vec3 lightDir   = normalize(vec3(params2.y, params2.z, 0.6));
    float NdotL     = max(dot(N, lightDir), 0.0);
    float spec_sharp = pow(NdotL, 36.0);
    float spec_soft  = pow(NdotL, 6.0);
    vec3 specCol = vec3(0.85, 0.95, 1.0) * (spec_sharp + spec_soft * 0.18) * params.z * videoMask;
    col += specCol;

    // --- Caustics from Laplacian (positive = focal pinch) ---
    float caustics = clamp(lap * 220.0, 0.0, 1.0);
    vec3 caustCol = vec3(0.55, 0.78, 1.0) * caustics * params.w * videoMask;
    col += caustCol;

    // --- Soft bloom: highlight^2 with cool tint (cheap, no extra sampling) ---
    float highlight = max(spec_soft * 0.35, caustics * 0.6);
    col += vec3(0.7, 0.85, 1.0) * highlight * highlight * sky.w * videoMask;

    // --- Depth tint by displacement magnitude ---
    float depthMix = clamp(length(grad) * 30.0, 0.0, 1.0) * params2.x;
    col = mix(col, col * vec3(0.82, 0.95, 1.12), depthMix * videoMask);

    fragColor = TDOutputSwizzle(vec4(col, 1.0));
}
'''
wave_apply.par.pixeldat = apply_dat.path
pos(apply_dat, 300, -150)

# ---- 6. make wave_apply the displayed TOP ----
try:
    r1.display = False
except Exception:
    pass
wave_apply.display = True
wave_apply.viewer = True

# Comments
wave_apply.comment = ('Apply Wave_ripples height field as UV displacement on render1. '
                      'uniform params: x=strength y=rimBoost')
rc.comment = 'Mask render of circle instances (same cam) - feeds Wave_ripples source'

print('ripple chain (with Wave_ripples component) built')
print('  mask: render_circles -> mask_blur -> mask_merge')
print('  sim : Wave_ripples (', wr.path, ')')
print('  out : wave_apply ->', wave_apply.path)
