"""
td-cli Heartbeat — CHOP Execute DAT

Place this code in a CHOP Execute DAT inside TDCliServer COMP.
Reference a Timer CHOP (1 second cycle, On Done = Re-Start).
Enable the 'Off to On' and 'On to Off' toggles.

me - this DAT
"""

import os
import json
import hashlib
import time


def _get_state():
    """Determine current TouchDesigner state.
    Returns one of: ready, cooking, error, initializing, playing, paused."""
    try:
        # Check for cook errors on the project root
        root = op('/')
        if root and hasattr(root, 'errors'):
            errs = root.errors(recurse=False)
            if errs:
                return 'error'

        # Check timeline state
        if hasattr(op, 'timeline'):
            tl = op.timeline
            if tl and not tl.play:
                return 'paused'

        # Check if the project is still loading
        if project.name == '':
            return 'initializing'

        # Check cook rate — if realTime is on and actual FPS is very low, still cooking
        if hasattr(project, 'realTime') and project.realTime:
            if hasattr(absTime, 'stepSeconds') and absTime.stepSeconds > 1.0:
                return 'cooking'

        return 'ready'
    except Exception:
        return 'ready'


def _write_heartbeat():
    """Write instance heartbeat file for CLI discovery."""
    home = os.path.expanduser('~')
    instances_dir = os.path.join(home, '.td-cli', 'instances')
    os.makedirs(instances_dir, exist_ok=True)

    project_path = project.folder
    hash_id = hashlib.md5(project_path.encode()).hexdigest()[:12]

    server = op('webserver1')
    port = int(server.par.port) if server else 9500

    instance_data = {
        'projectPath': project_path,
        'projectName': project.name,
        'port': port,
        'pid': os.getpid(),
        'timestamp': time.time(),
        'tdVersion': app.version,
        'tdBuild': app.build,
        'state': _get_state(),
    }

    filepath = os.path.join(instances_dir, f'{hash_id}.json')
    with open(filepath, 'w') as f:
        json.dump(instance_data, f, indent=2)


def _cleanup_heartbeat():
    """Remove heartbeat file on shutdown."""
    home = os.path.expanduser('~')
    instances_dir = os.path.join(home, '.td-cli', 'instances')
    project_path = project.folder
    hash_id = hashlib.md5(project_path.encode()).hexdigest()[:12]
    filepath = os.path.join(instances_dir, f'{hash_id}.json')
    if os.path.exists(filepath):
        os.remove(filepath)


def onOffToOn(channel: 'Channel', sampleIndex: int, val: float,
              prev: float):
    """Timer fires every 1s cycle — write heartbeat."""
    try:
        _write_heartbeat()
    except Exception as e:
        debug(f'td-cli heartbeat error: {e}')
    return


def whileOn(channel: 'Channel', sampleIndex: int, val: float,
            prev: float):
    return


def onOnToOff(channel: 'Channel', sampleIndex: int, val: float,
              prev: float):
    """Timer cycle ended — clean up if needed."""
    return


def whileOff(channel: 'Channel', sampleIndex: int, val: float,
             prev: float):
    return


def onValueChange(channel: 'Channel', sampleIndex: int, val: float,
                  prev: float):
    return
