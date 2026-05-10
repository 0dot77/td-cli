"""Install the velocity-gated OSC callback for the ripple installation.

Brief (Inter-Embodiment, calm-water reactive):
- |v| < VEL_FLOOR  -> emit OFF (stationary people make no ripple)
- emission strength is flat regardless of speed
- faster walkers emit a longer trail (intermediate samples between frames)
- emit slightly behind the foot (latency absorbed as form)
- cap concurrent sources to keep the displacement field readable

Run via td-cli exec -f td/install_ripple_osc.py /project1
Idempotent: rewrites /project1/oscin1_callbacks .text and clears _prev_pos.
"""
import td

P = op('/project1')
cb = P.op('oscin1_callbacks')
if cb is None:
    raise RuntimeError('oscin1_callbacks DAT not found at /project1/oscin1_callbacks')

CALLBACK_SOURCE = '''from typing import List, Any

# === Brief: velocity-gated, flat-strength, trailing wake emission ===
# Tune these from the live-tuning panel (or here, then reload).

SCREEN_W = 9600.0
SCREEN_H = 1080.0
ASPECT = SCREEN_W / SCREEN_H

VEL_FLOOR = 0.06       # UV/s; below this, no emission (stationary gate)
TAIL_VEL = 0.15        # UV/s; above this, start dropping tail samples
MAX_TAIL = 4           # max intermediate impulses per frame per gid
EMIT_OFFSET = 0.012    # UV; emit this far behind the foot along velocity
MAX_SOURCES = 10       # hard cap on rows written per OSC frame

# module-scope state: gid -> (tx, tz, t)
_prev_pos = {}


def onReceiveOSC(dat: oscinDAT, rowIndex: int, message: str,
                byteData: bytes, timeStamp: float, address: str,
                args: List[Any], peer: Peer):

    if address != '/proj/corridor/xy':
        return

    table = op('person_table')
    table.clear()
    table.appendRow(['gid', 'tx', 'ty', 'tz', 'active'])

    now = absTime.seconds
    seen = set()
    written = 0

    for i in range(0, len(args), 3):
        if i + 2 >= len(args):
            break
        if written >= MAX_SOURCES:
            break

        gid = int(args[i])
        x = float(args[i + 1])
        y = float(args[i + 2])

        tx = ((x / SCREEN_W) - 0.5) * ASPECT
        tz = 0.5 - (y / SCREEN_H)
        seen.add(gid)

        prev = _prev_pos.get(gid)
        _prev_pos[gid] = (tx, tz, now)

        if prev is None:
            continue

        dt = max(now - prev[2], 1e-3)
        dx = tx - prev[0]
        dz = tz - prev[1]
        dist = (dx * dx + dz * dz) ** 0.5
        vel = dist / dt

        if vel < VEL_FLOOR:
            continue

        ux = dx / dist if dist > 1e-6 else 0.0
        uz = dz / dist if dist > 1e-6 else 0.0

        head_tx = tx - ux * EMIT_OFFSET
        head_tz = tz - uz * EMIT_OFFSET
        table.appendRow([gid, head_tx, 0.0, head_tz, 1])
        written += 1

        if vel > TAIL_VEL and written < MAX_SOURCES:
            n_tail = min(int(vel / TAIL_VEL), MAX_TAIL)
            for k in range(1, n_tail + 1):
                if written >= MAX_SOURCES:
                    break
                a = k / (n_tail + 1)
                t_tx = tx - dx * a - ux * EMIT_OFFSET
                t_tz = tz - dz * a - uz * EMIT_OFFSET
                table.appendRow([gid * 100 + k, t_tx, 0.0, t_tz, 1])
                written += 1

    stale = [g for g in _prev_pos if g not in seen]
    for g in stale:
        _prev_pos.pop(g, None)

    return
'''

cb.text = CALLBACK_SOURCE

mod = cb.module
if hasattr(mod, '_prev_pos'):
    mod._prev_pos.clear()

pt = P.op('person_table')
if pt:
    pt.clear()
    pt.appendRow(['gid', 'tx', 'ty', 'tz', 'active'])

print('OSC callback installed; _prev_pos & person_table cleared.')
