# td-cli Request Handler
# Place this code in a Text DAT named 'handler' inside TDCliServer COMP
# This module routes requests to the appropriate tool handlers

import json
import sys
import io
import os
import time
import tempfile
import hashlib
import traceback

CONNECTOR_NAME = "TDCliServer"
CONNECTOR_VERSION = "0.1.0"
PROTOCOL_VERSION = 1
CONNECTOR_INSTALL_MODE = "tox"


def handle_request(uri, body):
    """Route request to appropriate handler based on URI."""
    routes = {
        "/exec": handle_exec,
        "/ops/list": handle_ops_list,
        "/ops/create": handle_ops_create,
        "/ops/delete": handle_ops_delete,
        "/ops/info": handle_ops_info,
        "/ops/rename": handle_ops_rename,
        "/ops/copy": handle_ops_copy,
        "/ops/move": handle_ops_move,
        "/ops/clone": handle_ops_clone,
        "/ops/search": handle_ops_search,
        "/par/get": handle_par_get,
        "/par/set": handle_par_set,
        "/par/pulse": handle_par_pulse,
        "/par/reset": handle_par_reset,
        "/par/expr": handle_par_expr,
        "/par/export": handle_par_export,
        "/par/import": handle_par_import,
        "/connect": handle_connect,
        "/disconnect": handle_disconnect,
        "/dat/read": handle_dat_read,
        "/dat/write": handle_dat_write,
        "/project/info": handle_project_info,
        "/project/save": handle_project_save,
        "/screenshot": handle_screenshot,
        "/network/export": handle_network_export,
        "/network/import": handle_network_import,
        "/backup/list": handle_backup_list,
        "/backup/restore": handle_backup_restore,
        "/logs/list": handle_logs_list,
        "/logs/tail": handle_logs_tail,
        "/network/describe": handle_network_describe,
        "/monitor": handle_monitor,
        "/shaders/apply": handle_shaders_apply,
        "/tox/export": handle_tox_export,
        "/tox/import": handle_tox_import,
        "/tools/list": handle_tools_list,
    }

    handler = routes.get(uri)
    if handler is None:
        return {"success": False, "message": f"Unknown route: {uri}", "data": None}

    request_id = _new_request_id()
    started_at = time.time()

    try:
        result = handler(body)
    except Exception as e:
        result = {
            "success": False,
            "message": str(e),
            "data": {"traceback": traceback.format_exc()},
        }

    result = _attach_request_id(result, request_id)
    _log_request_event(uri, body, result, request_id, started_at)
    return result


def _success(message, data=None):
    return {"success": True, "message": message, "data": data}


def _error(message, data=None):
    return {"success": False, "message": message, "data": data}


_OP_REFERENCE_STYLES = {
    "OP",
    "COMP",
    "TOP",
    "CHOP",
    "SOP",
    "DAT",
    "MAT",
    "OBJECT",
}


def _resolve_reference_target(owner, value):
    if not isinstance(value, str):
        return None

    text = value.strip()
    if not text:
        return None

    candidates = [text]
    owner_path = getattr(owner, "path", "")
    if owner_path:
        candidates.append(owner_path.rstrip("/") + "/" + text)
    parent = owner.parent() if hasattr(owner, "parent") else None
    parent_path = getattr(parent, "path", "")
    if parent_path:
        candidates.append(parent_path.rstrip("/") + "/" + text)

    seen = set()
    for candidate in candidates:
        if candidate in seen:
            continue
        seen.add(candidate)
        target = op(candidate)
        if target is not None:
            return target
    return None


def _normalize_op_reference_value(owner, par, value):
    if not isinstance(value, str):
        return value
    if value.startswith("./") or value.startswith("../"):
        return value

    style = str(getattr(par, "style", "")).upper()
    if style not in _OP_REFERENCE_STYLES:
        return value

    target = _resolve_reference_target(owner, value)
    if target is None or not hasattr(owner, "relativePath"):
        return value

    try:
        return owner.relativePath(target)
    except Exception:
        return value


def _validate_connector_index(connectors, index, label):
    """Validate connector index access before mutating the network."""
    if not isinstance(index, int):
        return _error(f"{label} index must be an integer")
    if index < 0 or index >= len(connectors):
        return _error(f"{label} index out of range: {index}")
    return None


def _snapshot_value(value):
    """Convert a parameter value into a JSON-safe representation plus its type."""
    if value is None:
        return None, "none"
    if isinstance(value, bool):
        return value, "bool"
    if isinstance(value, int) and not isinstance(value, bool):
        return value, "int"
    if isinstance(value, float):
        return value, "float"
    if isinstance(value, str):
        return value, "str"
    if isinstance(value, tuple):
        return [_snapshot_value(v)[0] for v in value], "tuple"
    if isinstance(value, list):
        return [_snapshot_value(v)[0] for v in value], "list"
    return str(value), type(value).__name__


def _operator_create_type(op_obj):
    """Return the type token required by parent.create(...)."""
    create_type = getattr(op_obj, "OPType", None)
    if create_type:
        return str(create_type)
    return str(op_obj.type)


def _snapshot_values_equal(value, default):
    """Compare values after JSON-safe normalization."""
    return _snapshot_value(value)[0] == _snapshot_value(default)[0]


def _restore_snapshot_value(pdata):
    """Restore a snapshot value with backward compatibility for v1 snapshots."""
    value_type = pdata.get("valueType")
    value = pdata.get("value")

    if not value_type:
        return value
    if value_type == "tuple":
        return tuple(value) if isinstance(value, list) else value
    if value_type == "list":
        return list(value) if isinstance(value, (list, tuple)) else value
    if value_type == "bool":
        return bool(value)
    if value_type == "int":
        return int(value)
    if value_type == "float":
        return float(value)
    if value_type == "none":
        return None
    return value


def _project_path():
    return (
        getattr(project, "folder", "")
        or getattr(project, "name", "")
        or "unknown-project"
    )


def _backup_root_dir():
    home = os.path.expanduser("~")
    project_hash = hashlib.sha256(_project_path().encode("utf-8")).hexdigest()[:16]
    return os.path.join(home, ".td-cli", "backups", project_hash)


def _logs_root_dir():
    home = os.path.expanduser("~")
    project_hash = hashlib.sha256(_project_path().encode("utf-8")).hexdigest()[:16]
    return os.path.join(home, ".td-cli", "logs", project_hash)


def _events_log_path():
    return os.path.join(_logs_root_dir(), "events.jsonl")


def _new_request_id():
    return f"{time.time_ns()}"


def _attach_request_id(result, request_id):
    data = result.get("data")
    if isinstance(data, dict):
        data.setdefault("requestId", request_id)
    elif data is None:
        result["data"] = {"requestId": request_id}
    else:
        result["data"] = {"requestId": request_id, "value": data}
    return result


def _target_path_from_body(body, result):
    if isinstance(body, dict):
        for key in ("path", "targetPath", "parentPath", "toxPath"):
            value = body.get(key)
            if value:
                return value
        if body.get("src") and body.get("dst"):
            return f"{body.get('src')} -> {body.get('dst')}"
        if body.get("id"):
            return body.get("id")

    data = result.get("data", {})
    if isinstance(data, dict):
        for key in ("restoredPath", "targetPath", "backupId"):
            value = data.get(key)
            if value:
                return value
    return ""


def _append_log_event(event):
    logs_dir = _logs_root_dir()
    os.makedirs(logs_dir, exist_ok=True)
    path = _events_log_path()
    with open(path, "a", encoding="utf-8") as handle:
        handle.write(json.dumps(event, ensure_ascii=True) + "\n")


def _read_log_events(limit=50):
    path = _events_log_path()
    if not os.path.exists(path):
        return []

    events = []
    with open(path, "r", encoding="utf-8") as handle:
        for line in handle:
            line = line.strip()
            if not line:
                continue
            try:
                events.append(json.loads(line))
            except Exception:
                continue

    return events[-limit:] if limit > 0 else events


def _log_request_event(uri, body, result, request_id, started_at):
    if uri in ("/logs/list", "/logs/tail"):
        return

    data = result.get("data", {})
    if not isinstance(data, dict):
        data = {}

    event = {
        "timestamp": time.time(),
        "requestId": request_id,
        "route": uri,
        "projectName": getattr(project, "name", ""),
        "projectPath": _project_path(),
        "targetPath": _target_path_from_body(body, result),
        "success": bool(result.get("success", False)),
        "durationMs": round((time.time() - started_at) * 1000, 3),
        "backupId": data.get("backupId"),
        "warningCount": data.get("warningCount", 0),
        "message": result.get("message", ""),
    }
    if not result.get("success", False):
        event["error"] = result.get("message", "")

    _append_log_event(event)


def _write_backup(kind, payload):
    """Persist a JSON backup before mutating the live project."""
    backup_dir = _backup_root_dir()
    os.makedirs(backup_dir, exist_ok=True)

    backup_id = f"{time.time_ns()}-{kind}"
    backup_path = os.path.join(backup_dir, backup_id + ".json")
    backup_record = {
        "id": backup_id,
        "kind": kind,
        "createdAt": time.time(),
        "projectName": getattr(project, "name", ""),
        "projectPath": _project_path(),
        "payload": payload,
    }

    fd, tmp_path = tempfile.mkstemp(dir=backup_dir, suffix=".tmp")
    try:
        with os.fdopen(fd, "w") as handle:
            json.dump(backup_record, handle, indent=2)
        os.replace(tmp_path, backup_path)
    except Exception:
        try:
            os.remove(tmp_path)
        except OSError:
            pass
        raise

    return {
        "backupId": backup_id,
        "backupPath": backup_path,
    }


def _snapshot_dat(target):
    is_table = target.isTable if hasattr(target, "isTable") else False
    if is_table:
        rows = []
        for row_idx in range(target.numRows):
            row = []
            for col_idx in range(target.numCols):
                row.append(str(target[row_idx, col_idx]))
            rows.append(row)
        return {
            "path": target.path,
            "family": target.family,
            "isTable": True,
            "table": rows,
            "numRows": target.numRows,
            "numCols": target.numCols,
        }

    return {
        "path": target.path,
        "family": target.family,
        "isTable": False,
        "content": target.text,
        "numRows": target.numRows,
        "numCols": target.numCols,
    }


def _apply_dat_snapshot(target, snapshot):
    if snapshot.get("isTable"):
        target.clear()
        for row in snapshot.get("table", []):
            target.appendRow(row)
    else:
        target.text = snapshot.get("content", "")


def _read_backup_record(backup_id):
    backup_path = os.path.join(_backup_root_dir(), backup_id + ".json")
    if not os.path.exists(backup_path):
        return None, backup_path
    with open(backup_path, "r") as handle:
        return json.load(handle), backup_path


def _list_backup_records(limit=20):
    backup_dir = _backup_root_dir()
    if not os.path.isdir(backup_dir):
        return []

    records = []
    for entry in os.listdir(backup_dir):
        if not entry.endswith(".json"):
            continue
        path = os.path.join(backup_dir, entry)
        try:
            with open(path, "r") as handle:
                record = json.load(handle)
            payload = record.get("payload", {})
            records.append(
                {
                    "id": record.get("id", entry[:-5]),
                    "kind": record.get("kind", ""),
                    "createdAt": record.get("createdAt", 0),
                    "projectName": record.get("projectName", ""),
                    "projectPath": record.get("projectPath", ""),
                    "path": path,
                    "targetPath": payload.get("targetPath", ""),
                }
            )
        except Exception:
            continue

    records.sort(key=lambda item: item.get("createdAt", 0), reverse=True)
    return records[:limit]


# --- exec ---


def handle_exec(body):
    """Execute arbitrary Python code in TouchDesigner."""
    code = body.get("code", "")
    if not code:
        return _error("No code provided")

    # Capture stdout
    old_stdout = sys.stdout
    old_stderr = sys.stderr
    captured_out = io.StringIO()
    captured_err = io.StringIO()
    sys.stdout = captured_out
    sys.stderr = captured_err

    result_value = None
    try:
        # If code starts with 'return', wrap in a function
        if code.strip().startswith("return"):
            wrapped = f"def __td_cli_exec__():\n"
            for line in code.split("\n"):
                wrapped += f"    {line}\n"
            namespace = {}
            exec(wrapped, globals(), namespace)
            result_value = namespace["__td_cli_exec__"]()
        else:
            exec(code)
    except Exception as e:
        sys.stdout = old_stdout
        sys.stderr = old_stderr
        return _error(
            f"Execution error: {e}",
            {
                "stdout": captured_out.getvalue(),
                "stderr": captured_err.getvalue(),
                "traceback": traceback.format_exc(),
            },
        )
    finally:
        sys.stdout = old_stdout
        sys.stderr = old_stderr

    return _success(
        "Script executed",
        {
            "result": str(result_value) if result_value is not None else None,
            "stdout": captured_out.getvalue(),
            "stderr": captured_err.getvalue(),
        },
    )


# --- ops ---


def handle_ops_list(body):
    """List operators at a given path."""
    path = body.get("path", "/")
    depth = body.get("depth", 1)
    family = body.get("family", None)  # TOP, CHOP, SOP, DAT, COMP, MAT

    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")

    # TD's findChildren(depth=N) returns children at exactly depth N,
    # not up to depth N. Collect all levels up to the requested depth.
    children = []
    for d in range(1, depth + 1):
        children.extend(target.findChildren(depth=d))

    if family:
        family_upper = family.upper()
        children = [c for c in children if c.family == family_upper]

    ops_list = []
    for c in children:
        ops_list.append(
            {
                "path": c.path,
                "name": c.name,
                "type": c.type,
                "family": c.family,
                "nodeX": c.nodeX,
                "nodeY": c.nodeY,
            }
        )

    return _success(f"Found {len(ops_list)} operators", {"operators": ops_list})


def _find_open_position(parent_op, node_x=None, node_y=None):
    """Find a non-overlapping position for a new operator.
    If nodeX/nodeY are given, use them. Otherwise auto-place
    to the right of the rightmost existing child."""
    if node_x is not None and node_y is not None:
        return int(node_x), int(node_y)

    children = parent_op.findChildren(depth=1)
    if not children:
        return 0, 0

    # Place to the right of the rightmost child, same row
    max_x = max(c.nodeX for c in children)
    # Find the Y of the rightmost child for alignment
    rightmost = [c for c in children if c.nodeX == max_x]
    ref_y = rightmost[0].nodeY

    return max_x + 200, ref_y


def handle_ops_create(body):
    """Create a new operator."""
    op_type = body.get("type", "")
    parent_path = body.get("parent", "/")
    name = body.get("name", None)
    node_x = body.get("nodeX", None)
    node_y = body.get("nodeY", None)

    if not op_type:
        return _error("No operator type provided")

    parent_op = op(parent_path)
    if parent_op is None:
        return _error(f"Parent not found: {parent_path}")

    new_op = parent_op.create(op_type, name)

    # Position the new operator
    px, py = _find_open_position(parent_op, node_x, node_y)
    new_op.nodeX = px
    new_op.nodeY = py

    return _success(
        f"Created {op_type}",
        {
            "path": new_op.path,
            "name": new_op.name,
            "type": new_op.type,
            "family": new_op.family,
            "nodeX": new_op.nodeX,
            "nodeY": new_op.nodeY,
        },
    )


def handle_ops_delete(body):
    """Delete an operator."""
    path = body.get("path", "")
    if not path:
        return _error("No operator path provided")

    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")

    backup_meta = _write_backup(
        "ops-delete",
        {
            "targetPath": path,
            "parentPath": target.parent().path
            if hasattr(target, "parent") and target.parent()
            else "",
            "snapshot": _serialize_network_snapshot(
                target, 20, include_defaults=True, include_root=True
            ),
        },
    )

    name = target.name
    target.destroy()

    return _success(f"Deleted {name}", backup_meta)


def handle_ops_info(body):
    """Get detailed info about an operator."""
    path = body.get("path", "")
    if not path:
        return _error("No operator path provided")

    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")

    # Get input/output connections
    inputs = []
    for i, conn in enumerate(target.inputConnectors):
        for c in conn.connections:
            inputs.append({"index": i, "path": c.owner.path, "name": c.owner.name})

    outputs = []
    for i, conn in enumerate(target.outputConnectors):
        for c in conn.connections:
            outputs.append({"index": i, "path": c.owner.path, "name": c.owner.name})

    # Get parameters summary (first 50)
    params = []
    for p in target.pars()[:50]:
        params.append(
            {
                "name": p.name,
                "label": p.label,
                "value": str(p.val),
                "default": str(p.default),
                "mode": str(p.mode),
                "page": p.page.name if p.page else "",
            }
        )

    info = {
        "path": target.path,
        "name": target.name,
        "type": target.type,
        "family": target.family,
        "nodeX": target.nodeX,
        "nodeY": target.nodeY,
        "inputs": inputs,
        "outputs": outputs,
        "parameters": params,
        "errors": target.errors(recurse=False) if hasattr(target, "errors") else "",
        "warnings": target.warnings(recurse=False)
        if hasattr(target, "warnings")
        else "",
        "comment": target.comment,
    }

    return _success(f"Info for {target.name}", info)


def handle_ops_rename(body):
    path = body.get("path", "")
    new_name = body.get("name", "")
    if not path or not new_name:
        return _error("path and name required")
    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")
    old_name = target.name
    target.name = new_name
    return _success(
        f"Renamed {old_name} to {new_name}",
        {
            "path": target.path,
            "name": target.name,
        },
    )


def handle_ops_copy(body):
    src_path = body.get("src", "")
    parent_path = body.get("parent", "")
    name = body.get("name", None)
    node_x = body.get("nodeX", None)
    node_y = body.get("nodeY", None)
    if not src_path or not parent_path:
        return _error("src and parent required")
    src = op(src_path)
    parent_op = op(parent_path)
    if src is None:
        return _error(f"Source not found: {src_path}")
    if parent_op is None:
        return _error(f"Parent not found: {parent_path}")
    create_type = _operator_create_type(src)
    new_op = parent_op.copy(src, name=name)
    px, py = _find_open_position(parent_op, node_x, node_y)
    new_op.nodeX = px
    new_op.nodeY = py
    return _success(
        f"Copied {src.name}",
        {
            "path": new_op.path,
            "name": new_op.name,
            "type": new_op.type,
            "family": new_op.family,
            "nodeX": new_op.nodeX,
            "nodeY": new_op.nodeY,
        },
    )


def handle_ops_move(body):
    src_path = body.get("src", "")
    parent_path = body.get("parent", "")
    if not src_path or not parent_path:
        return _error("src and parent required")
    src = op(src_path)
    parent_op = op(parent_path)
    if src is None:
        return _error(f"Source not found: {src_path}")
    if parent_op is None:
        return _error(f"Parent not found: {parent_path}")
    src.currentParOpen = parent_op
    return _success(
        f"Moved {src.name} to {parent_op.path}",
        {
            "path": src.path,
            "name": src.name,
        },
    )


def handle_ops_clone(body):
    src_path = body.get("src", "")
    parent_path = body.get("parent", "")
    name = body.get("name", None)
    node_x = body.get("nodeX", None)
    node_y = body.get("nodeY", None)
    if not src_path or not parent_path:
        return _error("src and parent required")
    src = op(src_path)
    parent_op = op(parent_path)
    if src is None:
        return _error(f"Source not found: {src_path}")
    if parent_op is None:
        return _error(f"Parent not found: {parent_path}")
    new_op = parent_op.copy(src, name=name)
    px, py = _find_open_position(parent_op, node_x, node_y)
    new_op.nodeX = px
    new_op.nodeY = py
    return _success(
        f"Cloned {src.name}",
        {
            "path": new_op.path,
            "name": new_op.name,
            "type": new_op.type,
            "family": new_op.family,
            "nodeX": new_op.nodeX,
            "nodeY": new_op.nodeY,
        },
    )


def handle_ops_search(body):
    parent_path = body.get("parent", "/")
    pattern = body.get("pattern", "")
    family = body.get("family", None)
    depth = body.get("depth", 10)
    parent_op = op(parent_path)
    if parent_op is None:
        return _error(f"Parent not found: {parent_path}")
    children = []
    for d in range(1, depth + 1):
        children.extend(parent_op.findChildren(depth=d))
    results = []
    import re

    try:
        regex = re.compile(pattern, re.IGNORECASE)
    except re.error:
        regex = None
    for c in children:
        if family and c.family != family.upper():
            continue
        if pattern:
            match = False
            if regex:
                if regex.search(c.name) or regex.search(c.path) or regex.search(c.type):
                    match = True
            else:
                if (
                    pattern.lower() in c.name.lower()
                    or pattern.lower() in c.type.lower()
                ):
                    match = True
            if not match:
                continue
        results.append(
            {
                "path": c.path,
                "name": c.name,
                "type": c.type,
                "family": c.family,
                "nodeX": c.nodeX,
                "nodeY": c.nodeY,
            }
        )
    return _success(f"Found {len(results)} operators", {"operators": results})


# --- par ---


def handle_par_get(body):
    """Get parameters of an operator."""
    path = body.get("path", "")
    names = body.get("names", None)  # Optional: specific parameter names

    if not path:
        return _error("No operator path provided")

    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")

    params = []
    if names:
        for name in names:
            p = getattr(target.par, name, None)
            if p is not None:
                params.append(
                    {
                        "name": p.name,
                        "label": p.label,
                        "value": str(p.val),
                        "default": str(p.default),
                        "min": str(p.min) if hasattr(p, "min") else None,
                        "max": str(p.max) if hasattr(p, "max") else None,
                        "type": str(type(p.val).__name__),
                        "mode": str(p.mode),
                    }
                )
    else:
        for p in target.pars()[:100]:
            params.append(
                {
                    "name": p.name,
                    "label": p.label,
                    "value": str(p.val),
                    "default": str(p.default),
                    "type": str(type(p.val).__name__),
                    "mode": str(p.mode),
                }
            )

    return _success(f"{len(params)} parameters", {"parameters": params})


def handle_par_set(body):
    """Set parameters on an operator."""
    path = body.get("path", "")
    params = body.get("params", {})

    if not path:
        return _error("No operator path provided")
    if not params:
        return _error("No parameters provided")

    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")

    updated = []
    for name, value in params.items():
        p = getattr(target.par, name, None)
        if p is None:
            return _error(f"Parameter not found: {name}")
        value = _normalize_op_reference_value(target, p, value)
        p.val = value
        updated.append({"name": name, "value": str(p.val)})

    return _success(f"Updated {len(updated)} parameters", {"updated": updated})


def handle_par_pulse(body):
    path = body.get("path", "")
    name = body.get("name", "")
    if not path or not name:
        return _error("path and name required")
    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")
    p = getattr(target.par, name, None)
    if p is None:
        return _error(f"Parameter not found: {name}")
    p.pulse()
    return _success(f"Pulsed {name}")


def handle_par_reset(body):
    path = body.get("path", "")
    names = body.get("names", [])
    if not path:
        return _error("path required")
    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")
    reset = []
    if names:
        for name in names:
            p = getattr(target.par, name, None)
            if p is not None:
                p.val = p.default
                reset.append(name)
    else:
        for p in target.pars():
            try:
                p.val = p.default
                reset.append(p.name)
            except Exception:
                pass
    return _success(f"Reset {len(reset)} parameters", {"reset": reset})


def handle_par_expr(body):
    path = body.get("path", "")
    name = body.get("name", "")
    expression = body.get("expression", None)
    if not path or not name:
        return _error("path and name required")
    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")
    p = getattr(target.par, name, None)
    if p is None:
        return _error(f"Parameter not found: {name}")
    if expression is not None:
        p.expr = expression
    return _success(
        f"Expression for {name}",
        {
            "name": name,
            "expression": getattr(p, "expr", "") or "",
            "value": str(p.val),
            "mode": str(p.mode),
        },
    )


def handle_par_export(body):
    path = body.get("path", "")
    if not path:
        return _error("path required")
    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")
    params = []
    for p in target.pars():
        entry = {
            "name": p.name,
            "label": p.label,
            "value": str(p.val),
            "default": str(p.default),
            "mode": str(p.mode),
            "type": str(type(p.val).__name__),
        }
        expr = getattr(p, "expr", "") or ""
        if expr:
            entry["expression"] = expr
        entry["page"] = p.page.name if p.page else ""
        params.append(entry)
    return _success(
        f"Exported {len(params)} parameters",
        {
            "path": path,
            "parameters": params,
        },
    )


def handle_par_import(body):
    path = body.get("path", "")
    params = body.get("params", [])
    if not path or not params:
        return _error("path and params required")
    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")
    applied = 0
    for pdata in params:
        name = pdata.get("name", "")
        p = getattr(target.par, name, None)
        if p is None:
            continue
        if "expression" in pdata and pdata["expression"]:
            p.expr = pdata["expression"]
        elif "value" in pdata:
            p.val = pdata["value"]
        applied += 1
    return _success(f"Imported {applied} parameters", {"applied": applied})


# --- connect / disconnect ---


def handle_connect(body):
    """Connect two operators."""
    src_path = body.get("src", "")
    dst_path = body.get("dst", "")
    src_index = body.get("srcIndex", 0)
    dst_index = body.get("dstIndex", 0)

    if not src_path or not dst_path:
        return _error("Both src and dst paths required")

    src_op = op(src_path)
    dst_op = op(dst_path)

    if src_op is None:
        return _error(f"Source not found: {src_path}")
    if dst_op is None:
        return _error(f"Destination not found: {dst_path}")

    err = _validate_connector_index(
        src_op.outputConnectors, src_index, "Source connector"
    )
    if err:
        return err

    err = _validate_connector_index(
        dst_op.inputConnectors, dst_index, "Destination connector"
    )
    if err:
        return err

    src_op.outputConnectors[src_index].connect(dst_op.inputConnectors[dst_index])

    return _success(f"Connected {src_op.name} -> {dst_op.name}")


def handle_disconnect(body):
    """Disconnect two operators."""
    src_path = body.get("src", "")
    dst_path = body.get("dst", "")

    if not src_path or not dst_path:
        return _error("Both src and dst paths required")

    src_op = op(src_path)
    dst_op = op(dst_path)

    if src_op is None:
        return _error(f"Source not found: {src_path}")
    if dst_op is None:
        return _error(f"Destination not found: {dst_path}")

    # Find and disconnect the connection
    for conn in src_op.outputConnectors:
        for c in conn.connections:
            if c.owner == dst_op:
                conn.disconnect(c)
                return _success(f"Disconnected {src_op.name} -> {dst_op.name}")

    return _error(f"No connection found between {src_path} and {dst_path}")


# --- dat ---


def handle_dat_read(body):
    """Read DAT content."""
    path = body.get("path", "")
    if not path:
        return _error("No DAT path provided")

    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")

    if target.family != "DAT":
        return _error(f"{path} is not a DAT (family: {target.family})")

    snapshot = _snapshot_dat(target)

    if snapshot["isTable"]:
        return _success(
            "DAT content read",
            {
                "content": None,
                "table": snapshot["table"],
                "numRows": snapshot["numRows"],
                "numCols": snapshot["numCols"],
                "isTable": True,
            },
        )
    else:
        return _success(
            "DAT content read",
            {
                "content": snapshot["content"],
                "table": None,
                "numRows": snapshot["numRows"],
                "numCols": snapshot["numCols"],
                "isTable": False,
            },
        )


def handle_dat_write(body):
    """Write DAT content."""
    path = body.get("path", "")
    content = body.get("content", None)
    table = body.get("table", None)

    if not path:
        return _error("No DAT path provided")

    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")

    if target.family != "DAT":
        return _error(f"{path} is not a DAT (family: {target.family})")

    backup_meta = _write_backup(
        "dat-write",
        {
            "targetPath": path,
            "before": _snapshot_dat(target),
        },
    )

    if table is not None:
        target.clear()
        for row in table:
            target.appendRow(row)
        return _success(f"Wrote {len(table)} rows to table DAT", backup_meta)
    elif content is not None:
        target.text = content
        return _success("Wrote content to DAT", backup_meta)
    else:
        return _error("No content or table data provided")


# --- project ---


def handle_project_info(body):
    """Get project metadata."""
    return _success(
        "Project info",
        {
            "name": project.name,
            "folder": project.folder,
            "saveVersion": project.saveVersion,
            "tdVersion": app.version,
            "tdBuild": app.build,
            "fps": project.cookRate,
            "realTime": project.realTime,
            "timelineFrame": absTime.frame,
            "timelineSeconds": absTime.seconds,
        },
    )


def handle_project_save(body):
    """Save the project."""
    path = body.get("path", None)

    if path:
        project.save(path)
        return _success(f"Project saved to {path}")
    else:
        project.save()
        return _success("Project saved")


# --- screenshot ---


def handle_screenshot(body):
    """Capture a TOP's output as a base64-encoded PNG image."""
    import base64
    import tempfile
    import os

    path = body.get("path", "")

    if not path:
        # Auto-detect: find first null/out TOP in project root
        root = op("/project1")
        if root:
            tops = root.findChildren(family="TOP", depth=1)
            nulls = [
                t for t in tops if "null" in t.type.lower() or "out" in t.name.lower()
            ]
            if nulls:
                target = nulls[0]
            elif tops:
                target = tops[0]
            else:
                return _error("No TOP found and no path specified")
        else:
            return _error("No path specified and /project1 not found")
    else:
        target = op(path)

    if target is None:
        return _error(f"Operator not found: {path}")
    if target.family != "TOP":
        return _error(f"{path} is not a TOP (family: {target.family})")

    try:
        tmp = tempfile.mktemp(suffix=".png")
        target.save(tmp)
        with open(tmp, "rb") as f:
            img_bytes = f.read()
        os.remove(tmp)

        b64 = base64.b64encode(img_bytes).decode("ascii")
        return _success(
            "Screenshot captured",
            {
                "image": b64,
                "width": target.width,
                "height": target.height,
            },
        )
    except Exception as e:
        return _error(f"Screenshot failed: {e}")


# --- network ---


def _serialize_op(o, include_defaults=False):
    """Serialize a single operator to dict."""
    node = {
        "path": o.path,
        "name": o.name,
        "type": o.type,
        "createType": _operator_create_type(o),
        "family": o.family,
        "nodeX": o.nodeX,
        "nodeY": o.nodeY,
        "comment": o.comment or "",
        "inputs": [],
        "parameters": {},
        "parameterErrors": [],
    }

    # Input connections
    for i, conn in enumerate(o.inputConnectors):
        for c in conn.connections:
            src_index = 0
            for si, sc in enumerate(c.owner.outputConnectors):
                for scc in sc.connections:
                    if scc.owner == o:
                        src_index = si
                        break
            node["inputs"].append(
                {
                    "index": i,
                    "sourcePath": c.owner.path,
                    "sourceIndex": src_index,
                }
            )

    # Parameters (non-default only unless include_defaults)
    for p in o.pars():
        try:
            has_expression = hasattr(p, "expr") and bool(p.expr)
            if (
                include_defaults
                or has_expression
                or not _snapshot_values_equal(p.val, p.default)
            ):
                value, value_type = _snapshot_value(p.val)
                default, default_type = _snapshot_value(p.default)
                par_data = {
                    "value": value,
                    "valueType": value_type,
                    "default": default,
                    "defaultType": default_type,
                    "mode": str(p.mode),
                }
                if hasattr(p, "expr") and p.expr:
                    par_data["expression"] = p.expr
                node["parameters"][p.name] = par_data
        except Exception as e:
            node["parameterErrors"].append(f"{p.name}: {e}")

    return node


def _collect_network_nodes(
    parent, remaining_depth, include_defaults=False, include_root=False
):
    """Recursively walk network and serialize operators."""
    nodes = []
    if remaining_depth < 0:
        return nodes

    if include_root:
        nodes.append(_serialize_op(parent, include_defaults))
        if remaining_depth == 0:
            return nodes

    if remaining_depth == 0:
        return nodes

    for child in parent.findChildren(depth=1):
        nodes.append(_serialize_op(child, include_defaults))
        if child.isCOMP and child.name != "TDCliServer":
            nodes.extend(
                _collect_network_nodes(
                    child, remaining_depth - 1, include_defaults, include_root=False
                )
            )
    return nodes


def _serialize_network_snapshot(
    target, depth, include_defaults=False, include_root=False
):
    nodes = _collect_network_nodes(
        target, depth, include_defaults, include_root=include_root
    )
    return {
        "version": 2,
        "rootPath": target.path,
        "exportTime": absTime.seconds,
        "tdVersion": app.version,
        "tdBuild": app.build,
        "nodeCount": len(nodes),
        "nodes": nodes,
        "warningCount": sum(len(n.get("parameterErrors", [])) for n in nodes),
    }


def _clear_children(target):
    for child in list(target.findChildren(depth=1)):
        child.destroy()


def _import_network_snapshot(
    snapshot, target_path, create_backup=True, clear_existing=False
):
    nodes = snapshot.get("nodes", [])
    parent = op(target_path)
    if parent is None:
        return _error(f"Target not found: {target_path}")

    backup_meta = {}
    if create_backup:
        backup_meta = _write_backup(
            "network-import",
            {
                "targetPath": target_path,
                "before": _serialize_network_snapshot(
                    parent, 10, include_defaults=True, include_root=False
                ),
                "incomingSnapshotVersion": snapshot.get("version", 1),
            },
        )

    if clear_existing:
        _clear_children(parent)

    created = []
    create_failures = []
    parameter_failures = []
    connection_failures = []
    path_map = {}
    root_path = snapshot.get("rootPath", "/")

    sorted_nodes = sorted(nodes, key=lambda n: n["path"].count("/"))

    for node in sorted_nodes:
        try:
            old_path = node["path"]
            old_parent_path = "/".join(old_path.rsplit("/", 1)[:-1]) or "/"

            if old_parent_path in path_map:
                actual_parent = path_map[old_parent_path]
            elif old_parent_path == root_path or old_parent_path == "/":
                actual_parent = parent
            else:
                actual_parent = parent

            create_type = node.get("createType") or node.get("type")
            new_op = actual_parent.create(create_type, node["name"])
            new_op.nodeX = node.get("nodeX", 0)
            new_op.nodeY = node.get("nodeY", 0)
            if node.get("comment"):
                new_op.comment = node["comment"]

            for pname, pdata in node.get("parameters", {}).items():
                p = getattr(new_op.par, pname, None)
                if p is None:
                    parameter_failures.append(
                        {
                            "path": new_op.path,
                            "parameter": pname,
                            "error": "Parameter not found",
                        }
                    )
                    continue
                try:
                    if pdata.get("expression"):
                        p.expr = pdata["expression"]
                    else:
                        p.val = _restore_snapshot_value(pdata)
                except Exception as e:
                    parameter_failures.append(
                        {
                            "path": new_op.path,
                            "parameter": pname,
                            "error": str(e),
                        }
                    )

            created.append(new_op.path)
            path_map[node["path"]] = new_op
        except Exception as e:
            create_failures.append(
                {
                    "path": node.get("path", ""),
                    "name": node.get("name", "?"),
                    "error": str(e),
                }
            )

    connections_made = 0
    for node in nodes:
        new_dst = path_map.get(node["path"])
        if not new_dst:
            continue
        for inp in node.get("inputs", []):
            src_op = path_map.get(inp["sourcePath"])
            if src_op:
                try:
                    src_idx = inp.get("sourceIndex", 0)
                    dst_idx = inp.get("index", 0)
                    src_op.outputConnectors[src_idx].connect(
                        new_dst.inputConnectors[dst_idx]
                    )
                    connections_made += 1
                except Exception as e:
                    connection_failures.append(
                        {
                            "sourcePath": inp.get("sourcePath", ""),
                            "targetPath": new_dst.path,
                            "sourceIndex": src_idx,
                            "targetIndex": dst_idx,
                            "error": str(e),
                        }
                    )

    warning_count = (
        len(create_failures) + len(parameter_failures) + len(connection_failures)
    )
    return _success(
        f"Imported {len(created)} nodes, {connections_made} connections ({warning_count} warning(s))",
        {
            "created": created,
            "connections": connections_made,
            "createFailures": create_failures,
            "parameterFailures": parameter_failures,
            "connectionFailures": connection_failures,
            "warningCount": warning_count,
            **backup_meta,
        },
    )


def handle_network_export(body):
    """Export network structure as JSON snapshot."""
    path = body.get("path", "/")
    depth = body.get("depth", 10)
    include_defaults = body.get("includeDefaults", False)

    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")

    snapshot = _serialize_network_snapshot(
        target, depth, include_defaults, include_root=False
    )
    return _success(f"Exported {snapshot['nodeCount']} nodes", snapshot)


def handle_network_import(body):
    """Recreate network from snapshot JSON."""
    snapshot = body.get("snapshot", {})
    target_path = body.get("targetPath", snapshot.get("rootPath", "/"))
    return _import_network_snapshot(
        snapshot, target_path, create_backup=True, clear_existing=False
    )


# --- backup ---


def handle_backup_list(body):
    """List recent backup artifacts for the current project."""
    limit = body.get("limit", 20)
    try:
        limit = int(limit)
    except Exception:
        limit = 20
    if limit <= 0:
        limit = 20

    backups = _list_backup_records(limit=limit)
    return _success(f"Found {len(backups)} backup(s)", {"backups": backups})


def handle_backup_restore(body):
    """Restore a previously recorded backup artifact."""
    backup_id = body.get("id", "")
    if not backup_id:
        return _error("No backup id provided")

    record, backup_path = _read_backup_record(backup_id)
    if record is None:
        return _error(f"Backup not found: {backup_id}", {"backupPath": backup_path})

    kind = record.get("kind", "")
    payload = record.get("payload", {})

    if kind == "dat-write":
        target = op(payload.get("targetPath", ""))
        if target is None:
            return _error(f"Target not found: {payload.get('targetPath', '')}")
        _apply_dat_snapshot(target, payload.get("before", {}))
        return _success(
            f"Restored backup {backup_id}",
            {
                "restoredKind": kind,
                "restoredPath": payload.get("targetPath", ""),
                "backupId": backup_id,
                "backupPath": backup_path,
            },
        )

    if kind == "ops-delete":
        snapshot = payload.get("snapshot", {})
        target_path = payload.get("parentPath", snapshot.get("rootPath", "/"))
        result = _import_network_snapshot(
            snapshot, target_path, create_backup=False, clear_existing=False
        )
        if result.get("success"):
            data = result.get("data", {})
            data.update(
                {
                    "restoredKind": kind,
                    "backupId": backup_id,
                    "backupPath": backup_path,
                }
            )
            result["message"] = f"Restored backup {backup_id}"
        return result

    if kind == "network-import":
        before_snapshot = payload.get("before", {})
        target_path = payload.get("targetPath", before_snapshot.get("rootPath", "/"))
        result = _import_network_snapshot(
            before_snapshot, target_path, create_backup=False, clear_existing=True
        )
        if result.get("success"):
            data = result.get("data", {})
            data.update(
                {
                    "restoredKind": kind,
                    "backupId": backup_id,
                    "backupPath": backup_path,
                }
            )
            result["message"] = f"Restored backup {backup_id}"
        return result

    return _error(
        f"Unsupported backup kind: {kind}",
        {
            "backupId": backup_id,
            "backupPath": backup_path,
        },
    )


# --- logs ---


def handle_logs_list(body):
    """List recent audit log events (newest first)."""
    limit = body.get("limit", 20)
    try:
        limit = int(limit)
    except Exception:
        limit = 20
    if limit <= 0:
        limit = 20

    events = list(reversed(_read_log_events(limit=limit)))
    return _success(f"Found {len(events)} log event(s)", {"events": events})


def handle_logs_tail(body):
    """Return recent audit log events in chronological order."""
    limit = body.get("limit", 20)
    try:
        limit = int(limit)
    except Exception:
        limit = 20
    if limit <= 0:
        limit = 20

    events = _read_log_events(limit=limit)
    return _success(f"Found {len(events)} log event(s)", {"events": events})


# --- describe ---


def handle_network_describe(body):
    """Generate AI-friendly description of a network."""
    path = body.get("path", "/")

    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")

    children = target.findChildren(depth=1)

    nodes = []
    edges = []
    families = {}

    for child in children:
        if child.name == "TDCliServer":
            continue

        node_info = {
            "name": child.name,
            "type": child.type,
            "family": child.family,
            "keyParams": {},
        }

        # Non-default parameters
        for p in child.pars():
            try:
                if str(p.val) != str(p.default):
                    node_info["keyParams"][p.name] = str(p.val)
            except Exception:
                pass

        nodes.append(node_info)
        families[child.family] = families.get(child.family, 0) + 1

        # Output connections
        for i, conn in enumerate(child.outputConnectors):
            for c in conn.connections:
                edges.append(
                    {
                        "from": child.name,
                        "to": c.owner.name,
                        "fromIndex": i,
                    }
                )

    # Find data flow chains (source -> ... -> sink)
    chains = []
    # Find roots (nodes with no inputs from siblings)
    sibling_names = {n["name"] for n in nodes}
    roots = []
    for n in nodes:
        has_input = any(
            e["to"] == n["name"] and e["from"] in sibling_names for e in edges
        )
        if not has_input:
            roots.append(n["name"])

    def trace_chain(name, visited=None):
        if visited is None:
            visited = set()
        if name in visited:
            return [name + "(loop)"]
        visited.add(name)
        chain = [name]
        outputs = [
            e["to"] for e in edges if e["from"] == name and e["to"] in sibling_names
        ]
        if outputs:
            for out in outputs:
                chain.extend(trace_chain(out, visited.copy()))
        return chain

    for root in roots:
        chain = trace_chain(root)
        if len(chain) > 1:
            chains.append(" -> ".join(chain))

    description = {
        "path": path,
        "nodeCount": len(nodes),
        "families": families,
        "nodes": nodes,
        "connections": edges,
        "dataFlow": chains,
    }
    return _success(
        f"Network: {len(nodes)} nodes, {len(edges)} connections", description
    )


# --- monitor ---


def handle_monitor(body):
    """Collect performance metrics for monitoring."""
    path = body.get("path", "/")

    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")

    step = (
        absTime.stepSeconds
        if hasattr(absTime, "stepSeconds") and absTime.stepSeconds > 0
        else 0.0167
    )

    metrics = {
        "fps": project.cookRate,
        "actualFps": round(1.0 / step, 1) if step > 0 else 0,
        "frame": absTime.frame,
        "seconds": round(absTime.seconds, 2),
        "realTime": project.realTime,
    }

    # Per-child performance metrics
    children = target.findChildren(depth=1)
    child_metrics = []
    for child in children:
        if child.name == "TDCliServer":
            continue
        cm = {
            "name": child.name,
            "type": child.type,
            "family": child.family,
        }
        if hasattr(child, "cookTime"):
            cm["cookTime"] = round(child.cookTime() * 1000, 3)  # ms
        if hasattr(child, "cpuCookTime"):
            cm["cpuCookTime"] = round(child.cpuCookTime() * 1000, 3)
        errs = child.errors(recurse=False) if hasattr(child, "errors") else ""
        warns = child.warnings(recurse=False) if hasattr(child, "warnings") else ""
        if errs:
            cm["errors"] = errs
        if warns:
            cm["warnings"] = warns
        child_metrics.append(cm)

    # Sort by cook time descending
    child_metrics.sort(key=lambda x: x.get("cookTime", 0), reverse=True)
    metrics["children"] = child_metrics[:20]

    return _success("Monitor data", metrics)


# --- shaders apply ---


def handle_shaders_apply(body):
    """Apply a shader template to a GLSL TOP."""
    target_path = body.get("path", "")
    glsl_code = body.get("glsl", "")
    uniforms = body.get("uniforms", [])

    if not target_path:
        return _error("No GLSL TOP path specified")
    if not glsl_code:
        return _error("No GLSL code provided")

    target = op(target_path)
    if target is None:
        return _error(f"Operator not found: {target_path}")

    # Find the pixel shader DAT
    pixel_dat_name = (
        target.par.pixeldat.val if hasattr(target.par, "pixeldat") else None
    )
    if not pixel_dat_name:
        return _error(f"Cannot find pixel shader DAT for {target_path}")

    pixel_dat = op(f"{target.parent().path}/{pixel_dat_name}")
    if pixel_dat is None:
        return _error(f"Pixel shader DAT not found: {pixel_dat_name}")

    # Write the GLSL code
    pixel_dat.text = glsl_code
    if hasattr(target.par, "pixeldat"):
        try:
            target.par.pixeldat.val = _normalize_op_reference_value(
                target, target.par.pixeldat, pixel_dat.path
            )
        except Exception:
            pass

    # Configure uniforms
    for i, u in enumerate(uniforms):
        if i >= 8:
            break  # GLSL TOP supports up to 8 vec uniforms
        name_par = f"vec{i}name"
        val_par = f"vec{i}valuex"
        if hasattr(target.par, name_par):
            setattr(target.par, name_par, u.get("name", ""))
        if hasattr(target.par, val_par) and u.get("default") is not None:
            try:
                getattr(target.par, val_par).val = float(u["default"])
            except (ValueError, TypeError):
                pass
        # Set expression if provided
        if u.get("expression") and hasattr(target.par, val_par):
            try:
                getattr(target.par, val_par).expr = u["expression"]
            except Exception:
                pass

    # Check compile status
    warnings = target.warnings(recurse=False) if hasattr(target, "warnings") else ""

    return _success(
        "Shader applied",
        {
            "path": target_path,
            "pixelDat": pixel_dat.path,
            "uniformsSet": len(uniforms),
            "compileWarnings": warnings,
        },
    )


# --- tox ---


def handle_tox_export(body):
    """Export a COMP as a .tox file."""
    import os

    comp_path = body.get("path", "")
    output = body.get("output", "")

    if not comp_path:
        return _error("No COMP path specified")
    if not output:
        return _error("No output path specified")

    target = op(comp_path)
    if target is None:
        return _error(f"Operator not found: {comp_path}")
    if not target.isCOMP:
        return _error(f"{comp_path} is not a COMP")

    try:
        # Ensure output directory exists
        os.makedirs(os.path.dirname(os.path.abspath(output)), exist_ok=True)
        target.save(output)
        size = os.path.getsize(output)
        return _success(
            f"Exported {target.name} to {output}",
            {
                "path": comp_path,
                "output": output,
                "size": size,
            },
        )
    except Exception as e:
        return _error(f"Export failed: {e}")


def handle_tox_import(body):
    """Import a .tox file into a parent COMP."""
    import os

    tox_path = body.get("toxPath", "")
    parent_path = body.get("parentPath", "/project1")
    name = body.get("name", "")

    if not tox_path:
        return _error("No .tox file path specified")
    if not os.path.exists(tox_path):
        return _error(f"File not found: {tox_path}")

    parent = op(parent_path)
    if parent is None:
        return _error(f"Parent not found: {parent_path}")

    try:
        # loadTox returns the created COMP
        new_op = parent.loadTox(tox_path)
        if name and new_op:
            new_op.name = name
        return _success(
            f"Imported {tox_path}",
            {
                "path": new_op.path if new_op else "",
                "name": new_op.name if new_op else "",
            },
        )
    except Exception as e:
        return _error(f"Import failed: {e}")


# --- tools ---

# Tool schema registry: each entry describes a route's purpose and parameters.
# This enables AI agents to discover available commands via `td-cli tools list`.
TOOL_SCHEMAS = [
    {
        "name": "exec",
        "route": "/exec",
        "description": "Execute arbitrary Python code in TouchDesigner",
        "parameters": [
            {
                "name": "code",
                "type": "string",
                "required": True,
                "description": 'Python code to execute. Prefix with "return" to get a value back.',
            },
        ],
    },
    {
        "name": "ops/list",
        "route": "/ops/list",
        "description": "List operators at a given path",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": False,
                "description": "Operator path (default: /)",
            },
            {
                "name": "depth",
                "type": "integer",
                "required": False,
                "description": "Search depth (default: 1)",
            },
            {
                "name": "family",
                "type": "string",
                "required": False,
                "description": "Filter by family: TOP, CHOP, SOP, DAT, COMP, MAT",
            },
        ],
    },
    {
        "name": "ops/create",
        "route": "/ops/create",
        "description": "Create a new operator",
        "parameters": [
            {
                "name": "type",
                "type": "string",
                "required": True,
                "description": "Operator type (e.g., noiseTOP, waveCHOP)",
            },
            {
                "name": "parent",
                "type": "string",
                "required": True,
                "description": "Parent operator path",
            },
            {
                "name": "name",
                "type": "string",
                "required": False,
                "description": "Operator name (auto-generated if omitted)",
            },
            {
                "name": "nodeX",
                "type": "integer",
                "required": False,
                "description": "X position (auto-placed if omitted)",
            },
            {
                "name": "nodeY",
                "type": "integer",
                "required": False,
                "description": "Y position (auto-placed if omitted)",
            },
        ],
    },
    {
        "name": "ops/delete",
        "route": "/ops/delete",
        "description": "Delete an operator",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "Operator path to delete",
            },
        ],
    },
    {
        "name": "ops/info",
        "route": "/ops/info",
        "description": "Get detailed info about an operator (connections, parameters, errors)",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "Operator path",
            },
        ],
    },
    {
        "name": "par/set",
        "route": "/par/set",
        "description": "Set parameters on an operator",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "Operator path",
            },
            {
                "name": "params",
                "type": "object",
                "required": True,
                "description": "Key-value pairs of parameter names and values",
            },
        ],
    },
    {
        "name": "par/pulse",
        "route": "/par/pulse",
        "description": "Pulse (momentary activate) a parameter",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "Operator path",
            },
            {
                "name": "name",
                "type": "string",
                "required": True,
                "description": "Parameter name to pulse",
            },
        ],
    },
    {
        "name": "par/reset",
        "route": "/par/reset",
        "description": "Reset parameters to their default values",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "Operator path",
            },
            {
                "name": "names",
                "type": "array",
                "required": False,
                "description": "Parameter names to reset (all if omitted)",
            },
        ],
    },
    {
        "name": "par/expr",
        "route": "/par/expr",
        "description": "Get or set a parameter expression",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "Operator path",
            },
            {
                "name": "name",
                "type": "string",
                "required": True,
                "description": "Parameter name",
            },
            {
                "name": "expression",
                "type": "string",
                "required": False,
                "description": "Expression to set (get if omitted)",
            },
        ],
    },
    {
        "name": "par/export",
        "route": "/par/export",
        "description": "Export all parameters of an operator as JSON",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "Operator path",
            },
        ],
    },
    {
        "name": "par/import",
        "route": "/par/import",
        "description": "Import parameters from JSON array",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "Operator path",
            },
            {
                "name": "params",
                "type": "array",
                "required": True,
                "description": "Array of {name, value/expression} objects",
            },
        ],
    },
    {
        "name": "ops/copy",
        "route": "/ops/copy",
        "description": "Copy (duplicate) an operator into a parent COMP",
        "parameters": [
            {
                "name": "src",
                "type": "string",
                "required": True,
                "description": "Source operator path",
            },
            {
                "name": "parent",
                "type": "string",
                "required": True,
                "description": "Parent COMP path",
            },
            {
                "name": "name",
                "type": "string",
                "required": False,
                "description": "New operator name",
            },
            {
                "name": "nodeX",
                "type": "integer",
                "required": False,
                "description": "X position",
            },
            {
                "name": "nodeY",
                "type": "integer",
                "required": False,
                "description": "Y position",
            },
        ],
    },
    {
        "name": "ops/move",
        "route": "/ops/move",
        "description": "Move an operator into a different parent COMP",
        "parameters": [
            {
                "name": "src",
                "type": "string",
                "required": True,
                "description": "Source operator path",
            },
            {
                "name": "parent",
                "type": "string",
                "required": True,
                "description": "Destination parent COMP path",
            },
        ],
    },
    {
        "name": "ops/clone",
        "route": "/ops/clone",
        "description": "Clone an operator (creates a linked clone)",
        "parameters": [
            {
                "name": "src",
                "type": "string",
                "required": True,
                "description": "Source operator path",
            },
            {
                "name": "parent",
                "type": "string",
                "required": True,
                "description": "Parent COMP path",
            },
            {
                "name": "name",
                "type": "string",
                "required": False,
                "description": "Clone name",
            },
        ],
    },
    {
        "name": "ops/search",
        "route": "/ops/search",
        "description": "Search operators by pattern (regex supported)",
        "parameters": [
            {
                "name": "parent",
                "type": "string",
                "required": False,
                "description": "Parent path (default: /)",
            },
            {
                "name": "pattern",
                "type": "string",
                "required": False,
                "description": "Search pattern (regex)",
            },
            {
                "name": "family",
                "type": "string",
                "required": False,
                "description": "Filter by family",
            },
            {
                "name": "depth",
                "type": "integer",
                "required": False,
                "description": "Search depth (default: 10)",
            },
        ],
    },
    {
        "name": "par/get",
        "route": "/par/get",
        "description": "Get parameters of an operator",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "Operator path",
            },
            {
                "name": "names",
                "type": "array",
                "required": False,
                "description": "Specific parameter names (all if omitted)",
            },
        ],
    },
    {
        "name": "par/set",
        "route": "/par/set",
        "description": "Set parameters on an operator",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "Operator path",
            },
            {
                "name": "params",
                "type": "object",
                "required": True,
                "description": "Key-value pairs of parameter names and values",
            },
        ],
    },
    {
        "name": "connect",
        "route": "/connect",
        "description": "Connect two operators (wire output to input)",
        "parameters": [
            {
                "name": "src",
                "type": "string",
                "required": True,
                "description": "Source operator path",
            },
            {
                "name": "dst",
                "type": "string",
                "required": True,
                "description": "Destination operator path",
            },
            {
                "name": "srcIndex",
                "type": "integer",
                "required": False,
                "description": "Source output index (default: 0)",
            },
            {
                "name": "dstIndex",
                "type": "integer",
                "required": False,
                "description": "Destination input index (default: 0)",
            },
        ],
    },
    {
        "name": "disconnect",
        "route": "/disconnect",
        "description": "Disconnect two operators",
        "parameters": [
            {
                "name": "src",
                "type": "string",
                "required": True,
                "description": "Source operator path",
            },
            {
                "name": "dst",
                "type": "string",
                "required": True,
                "description": "Destination operator path",
            },
        ],
    },
    {
        "name": "dat/read",
        "route": "/dat/read",
        "description": "Read DAT content (text or table)",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "DAT operator path",
            },
        ],
    },
    {
        "name": "dat/write",
        "route": "/dat/write",
        "description": "Write content to a DAT",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "DAT operator path",
            },
            {
                "name": "content",
                "type": "string",
                "required": False,
                "description": "Text content to write",
            },
            {
                "name": "table",
                "type": "array",
                "required": False,
                "description": "Table data as array of rows",
            },
        ],
    },
    {
        "name": "project/info",
        "route": "/project/info",
        "description": "Get project metadata (name, folder, TD version, FPS, timeline)",
        "parameters": [],
    },
    {
        "name": "project/save",
        "route": "/project/save",
        "description": "Save the project",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": False,
                "description": "Save path (current location if omitted)",
            },
        ],
    },
    {
        "name": "screenshot",
        "route": "/screenshot",
        "description": "Capture a TOP output as base64-encoded PNG image",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": False,
                "description": "TOP operator path (auto-detects if omitted)",
            },
        ],
    },
    {
        "name": "network/export",
        "route": "/network/export",
        "description": "Export network structure as JSON snapshot (operators, connections, parameters)",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": False,
                "description": "Root path to export (default: /)",
            },
            {
                "name": "depth",
                "type": "integer",
                "required": False,
                "description": "Recursion depth (default: 10)",
            },
            {
                "name": "includeDefaults",
                "type": "boolean",
                "required": False,
                "description": "Include default parameter values (default: false)",
            },
        ],
    },
    {
        "name": "network/import",
        "route": "/network/import",
        "description": "Recreate network from a JSON snapshot",
        "parameters": [
            {
                "name": "snapshot",
                "type": "object",
                "required": True,
                "description": "Network snapshot JSON object",
            },
            {
                "name": "targetPath",
                "type": "string",
                "required": False,
                "description": "Target parent path (default: from snapshot)",
            },
        ],
    },
    {
        "name": "backup/list",
        "route": "/backup/list",
        "description": "List recent backup artifacts for the current project",
        "parameters": [
            {
                "name": "limit",
                "type": "integer",
                "required": False,
                "description": "Maximum backups to return (default: 20)",
            },
        ],
    },
    {
        "name": "backup/restore",
        "route": "/backup/restore",
        "description": "Restore a previous backup artifact by id",
        "parameters": [
            {
                "name": "id",
                "type": "string",
                "required": True,
                "description": "Backup id returned by a mutating command or backup list",
            },
        ],
    },
    {
        "name": "logs/list",
        "route": "/logs/list",
        "description": "List recent audit log events (newest first)",
        "parameters": [
            {
                "name": "limit",
                "type": "integer",
                "required": False,
                "description": "Maximum log events to return (default: 20)",
            },
        ],
    },
    {
        "name": "logs/tail",
        "route": "/logs/tail",
        "description": "Read recent audit log events in chronological order",
        "parameters": [
            {
                "name": "limit",
                "type": "integer",
                "required": False,
                "description": "Maximum log events to return (default: 20)",
            },
        ],
    },
    {
        "name": "network/describe",
        "route": "/network/describe",
        "description": "Generate AI-friendly description of a network (nodes, connections, data flow)",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": False,
                "description": "Network path (default: /)",
            },
        ],
    },
    {
        "name": "monitor",
        "route": "/monitor",
        "description": "Collect performance metrics (FPS, cook time, errors)",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": False,
                "description": "Root path to monitor (default: /)",
            },
        ],
    },
    {
        "name": "shaders/apply",
        "route": "/shaders/apply",
        "description": "Apply a GLSL shader to a GLSL TOP (write pixel DAT + configure uniforms)",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "GLSL TOP operator path",
            },
            {
                "name": "glsl",
                "type": "string",
                "required": True,
                "description": "GLSL pixel shader code",
            },
            {
                "name": "uniforms",
                "type": "array",
                "required": False,
                "description": "Uniform definitions [{name, type, default, expression}]",
            },
        ],
    },
    {
        "name": "tox/export",
        "route": "/tox/export",
        "description": "Export a COMP as a .tox file",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "COMP operator path to export",
            },
            {
                "name": "output",
                "type": "string",
                "required": True,
                "description": "Output .tox file path",
            },
        ],
    },
    {
        "name": "tox/import",
        "route": "/tox/import",
        "description": "Import a .tox file into a parent COMP",
        "parameters": [
            {
                "name": "toxPath",
                "type": "string",
                "required": True,
                "description": ".tox file path to import",
            },
            {
                "name": "parentPath",
                "type": "string",
                "required": False,
                "description": "Parent COMP path (default: /project1)",
            },
            {
                "name": "name",
                "type": "string",
                "required": False,
                "description": "Rename the imported COMP",
            },
        ],
    },
]


def handle_tools_list(body):
    """Return schemas for all registered tools, enabling AI agent discovery."""
    return _success(f"{len(TOOL_SCHEMAS)} tools available", {"tools": TOOL_SCHEMAS})
