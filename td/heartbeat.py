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
import tempfile


def _get_connector_metadata():
    handler = op('handler')
    module = getattr(handler, 'module', None) if handler else None
    return {
        'connectorName': getattr(module, 'CONNECTOR_NAME', 'TDCliServer'),
        'connectorVersion': getattr(module, 'CONNECTOR_VERSION', '0.1.0'),
        'protocolVersion': getattr(module, 'PROTOCOL_VERSION', 1),
        'connectorInstallMode': getattr(module, 'CONNECTOR_INSTALL_MODE', 'tox'),
    }


def _get_state():
    """Determine current TouchDesigner state.
    Returns one of: ready, cooking, error, initializing, playing, paused, unknown."""
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
    except Exception as e:
        debug(f'td-cli state detection error: {e}')
        return 'unknown'


def _hash_project(project_path):
    """Generate a filename-safe hash from the project path."""
    return hashlib.sha256(project_path.encode()).hexdigest()[:16]


def _write_heartbeat():
    """Write instance heartbeat file for CLI discovery.
    Uses atomic write (temp file + rename) to prevent partial reads."""
    home = os.path.expanduser('~')
    instances_dir = os.path.join(home, '.td-cli', 'instances')
    os.makedirs(instances_dir, exist_ok=True)

    project_path = project.folder
    hash_id = _hash_project(project_path)

    server = op('webserver1')
    port = int(server.par.port) if server else 9500
    metadata = _get_connector_metadata()

    instance_data = {
        'projectPath': project_path,
        'projectName': project.name,
        'port': port,
        'pid': os.getpid(),
        'timestamp': time.time(),
        'tdVersion': app.version,
        'tdBuild': app.build,
        'state': _get_state(),
        'connectorName': metadata['connectorName'],
        'connectorVersion': metadata['connectorVersion'],
        'protocolVersion': metadata['protocolVersion'],
        'connectorInstallMode': metadata['connectorInstallMode'],
    }

    filepath = os.path.join(instances_dir, f'{hash_id}.json')

    # Atomic write: write to temp file then rename
    fd, tmp_path = tempfile.mkstemp(dir=instances_dir, suffix='.tmp')
    try:
        with os.fdopen(fd, 'w') as f:
            json.dump(instance_data, f, indent=2)
        # On Windows, target must not exist for rename
        if os.path.exists(filepath):
            os.replace(tmp_path, filepath)
        else:
            os.rename(tmp_path, filepath)
    except Exception:
        # Clean up temp file on failure
        try:
            os.remove(tmp_path)
        except OSError:
            pass
        raise


def _cleanup_heartbeat():
    """Remove heartbeat file on shutdown."""
    home = os.path.expanduser('~')
    instances_dir = os.path.join(home, '.td-cli', 'instances')
    project_path = project.folder
    hash_id = _hash_project(project_path)
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
