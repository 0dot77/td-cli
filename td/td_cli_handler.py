# td-cli Request Handler
# Place this code in a Text DAT named 'handler' inside TDCliServer COMP
# This module routes requests to the appropriate tool handlers
#
# === Section Index ===
# Core & Routing ............ line ~20   (handle_request, _success, _error, helpers)
# Exec ...................... line ~1230  (handle_exec)
# Operators ................. line ~1287  (handle_ops_*)
# Parameters ................ line ~1615  (handle_par_*)
# Connections ............... line ~1807  (handle_connect, handle_disconnect)
# DAT ....................... line ~1871  (handle_dat_*)
# Project ................... line ~1949  (handle_project_*)
# Screenshot & Media ........ line ~1982  (handle_screenshot, handle_media_*)
# Network Serialization ..... line ~2280  (handle_network_*, _serialize_*, _deserialize_*)
# Backup & Logs ............. line ~2308  (handle_backup_*, handle_logs_*)
# Describe .................. line ~2429  (handle_network_describe)
# Harness ................... line ~2539  (handle_harness_*)
# Monitor & Shaders ......... line ~2824  (handle_monitor, handle_shaders_apply)
# TOX ....................... line ~2912  (handle_tox_*)
# CHOP ...................... line ~2983  (handle_chop_*)
# SOP ....................... line ~3063  (handle_sop_*)
# POP ....................... line ~3158  (handle_pop_*)
# Table ..................... line ~3345  (handle_table_*)
# Timeline .................. line ~3450  (handle_timeline_*)
# Cook & UI ................. line ~3516  (handle_cook_*, handle_ui_*)
# Batch ..................... line ~3596  (handle_batch_*)
# Tools & Route Table ....... line ~5154  (handle_tools_list, ROUTE_TABLE)

import json
import sys
import io
import os
import re
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
    handler = ROUTE_TABLE.get(uri)
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


def _project_slug():
    raw_name = getattr(project, "name", "") or os.path.splitext(
        os.path.basename(_project_path())
    )[0]
    slug = re.sub(r"[^A-Za-z0-9._-]+", "-", str(raw_name)).strip("-._")
    if not slug:
        slug = "unknown-project"
    project_hash = hashlib.sha256(_project_path().encode("utf-8")).hexdigest()[:8]
    return f"{slug}-{project_hash}"


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


def _harness_root_dir():
    return os.path.join(os.path.expanduser("~"), ".td-cli", "harness", _project_slug())


def _harness_iterations_dir():
    return os.path.join(_harness_root_dir(), "iterations")


def _harness_events_log_path():
    return os.path.join(_harness_root_dir(), "events.jsonl")


def _write_json_file(path, payload):
    directory = os.path.dirname(path)
    os.makedirs(directory, exist_ok=True)
    fd, tmp_path = tempfile.mkstemp(dir=directory, suffix=".tmp")
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as handle:
            json.dump(payload, handle, indent=2, ensure_ascii=True)
        os.replace(tmp_path, path)
    except Exception:
        try:
            os.remove(tmp_path)
        except OSError:
            pass
        raise


def _append_harness_event(event):
    root_dir = _harness_root_dir()
    os.makedirs(root_dir, exist_ok=True)
    with open(_harness_events_log_path(), "a", encoding="utf-8") as handle:
        handle.write(json.dumps(event, ensure_ascii=True) + "\n")


def _read_json_file(path):
    with open(path, "r", encoding="utf-8") as handle:
        return json.load(handle)


def _read_harness_record(rollback_id):
    record_path = os.path.join(_harness_iterations_dir(), rollback_id + ".json")
    if not os.path.exists(record_path):
        return None, record_path
    return _read_json_file(record_path), record_path


def _write_harness_record(record):
    record_id = record.get("id") or f"{time.time_ns()}-harness"
    record["id"] = record_id
    record_path = os.path.join(_harness_iterations_dir(), record_id + ".json")
    _write_json_file(record_path, record)
    return record_id, record_path


def _stringify_messages(value):
    if value is None or value == "":
        return []
    if isinstance(value, (list, tuple)):
        return [str(item) for item in value if str(item)]
    text = str(value)
    return [line for line in text.splitlines() if line.strip()]


def _get_operator_messages(target, attr_name):
    if target is None or not hasattr(target, attr_name):
        return []
    member = getattr(target, attr_name)
    try:
        value = member(recurse=False) if callable(member) else member
    except TypeError:
        try:
            value = member() if callable(member) else member
        except Exception:
            return []
    except Exception:
        return []
    return _stringify_messages(value)


def _path_label(path):
    if path in ("", "/"):
        return path or "/"
    return path.rsplit("/", 1)[-1]


def _snapshot_root_node(snapshot):
    root_path = snapshot.get("rootPath", "")
    for node in snapshot.get("nodes", []):
        if node.get("path") == root_path:
            return node
    nodes = snapshot.get("nodes", [])
    return nodes[0] if nodes else {}


def _contains_connector_boundary(target):
    if target is None:
        return False
    if getattr(target, "name", "") == CONNECTOR_NAME:
        return True
    if getattr(target, "path", "") == "/" + CONNECTOR_NAME:
        return True
    if not getattr(target, "isCOMP", False):
        return False
    try:
        children = target.findChildren(depth=1)
    except Exception:
        return False
    return any(getattr(child, "name", "") == CONNECTOR_NAME for child in children)


def _snapshot_target_state(target, depth=20):
    state = {
        "version": 1,
        "capturedAt": time.time(),
        "path": target.path,
        "family": getattr(target, "family", ""),
        "network": _serialize_network_snapshot(
            target, depth, include_defaults=True, include_root=True
        ),
    }
    if getattr(target, "family", "") == "DAT":
        state["dat"] = _snapshot_dat(target)
    return state


def _summarize_target_state(state):
    network = state.get("network", {})
    root_node = _snapshot_root_node(network)
    return {
        "path": state.get("path", network.get("rootPath", "")),
        "family": state.get("family", root_node.get("family", "")),
        "type": root_node.get("type", ""),
        "name": root_node.get("name", _path_label(state.get("path", ""))),
        "nodeCount": network.get("nodeCount", len(network.get("nodes", []))),
        "warningCount": network.get("warningCount", 0),
        "hasDatSnapshot": bool(state.get("dat")),
    }


def _disconnect_input_connector(connector):
    failures = []
    for connection in list(getattr(connector, "connections", [])):
        source = getattr(connection, "owner", None)
        disconnected = False
        if source is not None:
            for output_connector in getattr(source, "outputConnectors", []):
                disconnect_fn = getattr(output_connector, "disconnect", None)
                if not callable(disconnect_fn):
                    continue
                try:
                    disconnect_fn(connection)
                    disconnected = True
                    break
                except Exception:
                    continue
        if disconnected:
            continue
        try:
            connector.connections.remove(connection)
            disconnected = True
        except Exception:
            pass
        if not disconnected:
            failures.append(str(source.path) if source is not None else "unknown")
    return failures


def _apply_operator_snapshot(target, node):
    failures = []
    if "nodeX" in node:
        target.nodeX = node.get("nodeX", target.nodeX)
    if "nodeY" in node:
        target.nodeY = node.get("nodeY", target.nodeY)
    if "comment" in node:
        target.comment = node.get("comment", "") or ""

    for pname, pdata in node.get("parameters", {}).items():
        p = getattr(target.par, pname, None)
        if p is None:
            failures.append(
                {
                    "path": target.path,
                    "parameter": pname,
                    "error": "Parameter not found",
                }
            )
            continue
        try:
            expression = pdata.get("expression")
            if expression:
                p.expr = expression
            else:
                if hasattr(p, "expr") and getattr(p, "expr", ""):
                    p.expr = ""
                p.val = _restore_snapshot_value(pdata)
        except Exception as e:
            failures.append({"path": target.path, "parameter": pname, "error": str(e)})
    return failures


def _restore_input_connections(target, node):
    failures = []
    desired_inputs = {}
    for entry in node.get("inputs", []):
        desired_inputs.setdefault(entry.get("index", 0), []).append(entry)

    for index, connector in enumerate(getattr(target, "inputConnectors", [])):
        for orphan in _disconnect_input_connector(connector):
            failures.append(
                {
                    "targetPath": target.path,
                    "targetIndex": index,
                    "error": f"Could not fully disconnect existing input from {orphan}",
                }
            )
        for entry in desired_inputs.get(index, []):
            source = op(entry.get("sourcePath", ""))
            if source is None:
                failures.append(
                    {
                        "targetPath": target.path,
                        "targetIndex": index,
                        "sourcePath": entry.get("sourcePath", ""),
                        "error": "Source not found",
                    }
                )
                continue
            source_index = entry.get("sourceIndex", 0)
            try:
                source.outputConnectors[source_index].connect(target.inputConnectors[index])
            except Exception as e:
                failures.append(
                    {
                        "targetPath": target.path,
                        "targetIndex": index,
                        "sourcePath": source.path,
                        "sourceIndex": source_index,
                        "error": str(e),
                    }
                )
    return failures


def _restore_target_state(state):
    snapshot = state.get("network", {})
    root_path = state.get("path", snapshot.get("rootPath", ""))
    if not root_path:
        return _error("Rollback snapshot is missing target path")

    root_node = _snapshot_root_node(snapshot)
    current = op(root_path)
    recreate = False
    if current is not None and root_node:
        recreate = (
            getattr(current, "type", None) != root_node.get("type")
            or getattr(current, "family", None) != root_node.get("family")
        )
        if recreate:
            current.destroy()
            current = None

    parameter_failures = []
    connection_failures = []

    if current is None:
        parent_path = "/".join(root_path.rsplit("/", 1)[:-1]) or "/"
        result = _import_network_snapshot(
            snapshot, parent_path, create_backup=False, clear_existing=False
        )
        if not result.get("success", False):
            return result
        data = result.get("data", {})
        parameter_failures.extend(data.get("parameterFailures", []))
        connection_failures.extend(data.get("connectionFailures", []))
        current = op(root_path)
    else:
        parameter_failures.extend(_apply_operator_snapshot(current, root_node))
        descendants = [
            node for node in snapshot.get("nodes", []) if node.get("path") != root_path
        ]
        if descendants:
            if not getattr(current, "isCOMP", False):
                return _error(
                    f"Cannot restore descendant network into non-COMP target: {root_path}"
                )
            _clear_children(current)
            child_snapshot = {
                "version": snapshot.get("version", 2),
                "rootPath": root_path,
                "exportTime": snapshot.get("exportTime"),
                "tdVersion": snapshot.get("tdVersion"),
                "tdBuild": snapshot.get("tdBuild"),
                "nodes": descendants,
                "nodeCount": len(descendants),
                "warningCount": sum(
                    len(node.get("parameterErrors", [])) for node in descendants
                ),
            }
            result = _import_network_snapshot(
                child_snapshot, root_path, create_backup=False, clear_existing=False
            )
            if not result.get("success", False):
                return result
            data = result.get("data", {})
            parameter_failures.extend(data.get("parameterFailures", []))
            connection_failures.extend(data.get("connectionFailures", []))
        current = op(root_path)

    if current is None:
        return _error(f"Failed to restore target: {root_path}")

    connection_failures.extend(_restore_input_connections(current, root_node))

    dat_failures = []
    if state.get("dat") and getattr(current, "family", "") == "DAT":
        try:
            _apply_dat_snapshot(current, state.get("dat", {}))
        except Exception as e:
            dat_failures.append(str(e))

    warning_count = len(parameter_failures) + len(connection_failures) + len(dat_failures)
    return _success(
        f"Restored harness snapshot for {root_path}",
        {
            "rollbackTarget": root_path,
            "restoredPath": root_path,
            "recreated": recreate,
            "parameterFailures": parameter_failures,
            "connectionFailures": connection_failures,
            "datFailures": dat_failures,
            "warningCount": warning_count,
        },
    )


def _build_monitor_metrics(target, limit=20):
    step = (
        getattr(absTime, "stepSeconds", 0)
        if hasattr(absTime, "stepSeconds")
        else 0
    )
    if not step:
        step = 0.0167

    metrics = {
        "fps": getattr(project, "cookRate", 0),
        "actualFps": round(1.0 / step, 1) if step > 0 else 0,
        "frame": getattr(absTime, "frame", None),
        "seconds": round(getattr(absTime, "seconds", 0), 3),
        "realTime": getattr(project, "realTime", None),
    }

    child_metrics = []
    children = target.findChildren(depth=1) if hasattr(target, "findChildren") else []
    for child in children:
        if child.name == "TDCliServer":
            continue
        entry = {
            "path": child.path,
            "name": child.name,
            "type": child.type,
            "family": child.family,
        }
        if hasattr(child, "cookTime"):
            try:
                entry["cookTime"] = round(child.cookTime() * 1000, 3)
            except Exception:
                pass
        if hasattr(child, "cpuCookTime"):
            try:
                entry["cpuCookTime"] = round(child.cpuCookTime() * 1000, 3)
            except Exception:
                pass
        errors = _get_operator_messages(child, "errors")
        warnings = _get_operator_messages(child, "warnings")
        if errors:
            entry["errors"] = errors
        if warnings:
            entry["warnings"] = warnings
        child_metrics.append(entry)

    child_metrics.sort(key=lambda item: item.get("cookTime", 0), reverse=True)
    metrics["children"] = child_metrics[:limit]
    return metrics


def _build_observation_payload(target, depth=2, include_snapshot=False):
    snapshot = _serialize_network_snapshot(
        target, depth, include_defaults=False, include_root=True
    )
    nodes = snapshot.get("nodes", [])
    path_to_name = {node.get("path", ""): node.get("name", "") for node in nodes}
    incoming = {}
    outgoing = {}
    edges = []
    families = {}
    semantic_nodes = []

    for node in nodes:
        node_path = node.get("path", "")
        families[node.get("family", "")] = families.get(node.get("family", ""), 0) + 1
        incoming.setdefault(node_path, 0)
        outgoing.setdefault(node_path, 0)

    for node in nodes:
        target_path = node.get("path", "")
        for input_entry in node.get("inputs", []):
            source_path = input_entry.get("sourcePath", "")
            edges.append(
                {
                    "sourcePath": source_path,
                    "sourceName": path_to_name.get(source_path, _path_label(source_path)),
                    "sourceIndex": input_entry.get("sourceIndex", 0),
                    "targetPath": target_path,
                    "targetName": node.get("name", _path_label(target_path)),
                    "targetIndex": input_entry.get("index", 0),
                }
            )
            outgoing[source_path] = outgoing.get(source_path, 0) + 1
            incoming[target_path] = incoming.get(target_path, 0) + 1

    issue_nodes = []
    for node in nodes:
        node_path = node.get("path", "")
        target_op = op(node_path)
        errors = _get_operator_messages(target_op, "errors")
        warnings = _get_operator_messages(target_op, "warnings")
        modified_params = sorted(node.get("parameters", {}).keys())
        expression_params = [
            name
            for name, pdata in node.get("parameters", {}).items()
            if pdata.get("expression")
        ]
        semantic = {
            "path": node_path,
            "name": node.get("name", _path_label(node_path)),
            "type": node.get("type", ""),
            "family": node.get("family", ""),
            "comment": node.get("comment", ""),
            "modifiedParameterCount": len(modified_params),
            "modifiedParameters": modified_params[:25],
            "expressionParameters": expression_params[:25],
            "inputCount": incoming.get(node_path, 0),
            "outputCount": outgoing.get(node_path, 0),
        }
        if target_op is not None and getattr(target_op, "family", "") == "TOP":
            semantic["width"] = getattr(target_op, "width", 0)
            semantic["height"] = getattr(target_op, "height", 0)
        if errors:
            semantic["errors"] = errors
        if warnings:
            semantic["warnings"] = warnings
        if errors or warnings:
            issue_nodes.append(
                {
                    "path": node_path,
                    "errors": errors,
                    "warnings": warnings,
                }
            )
        semantic_nodes.append(semantic)

    graph_paths = set(path_to_name.keys())
    root_paths = [path for path in graph_paths if incoming.get(path, 0) == 0]
    leaf_paths = [path for path in graph_paths if outgoing.get(path, 0) == 0]
    isolated_paths = [
        path
        for path in graph_paths
        if incoming.get(path, 0) == 0 and outgoing.get(path, 0) == 0
    ]

    def _trace_chain(path, visited=None, max_depth=16):
        if visited is None:
            visited = set()
        if path in visited:
            return [_path_label(path) + "(loop)"]
        visited.add(path)
        chain = [_path_label(path)]
        if max_depth <= 0:
            return chain
        next_paths = [
            edge["targetPath"] for edge in edges if edge.get("sourcePath") == path
        ]
        if next_paths:
            chain.extend(_trace_chain(next_paths[0], visited, max_depth - 1))
        return chain

    data_flow = []
    for path in root_paths[:10]:
        chain = _trace_chain(path)
        if len(chain) > 1:
            data_flow.append(" -> ".join(chain))

    output_candidates = []
    for node in semantic_nodes:
        score = 0
        reasons = []
        name_lower = node.get("name", "").lower()
        if node.get("family") == "TOP":
            score += 4
            reasons.append("top")
        if node.get("outputCount", 0) == 0:
            score += 2
            reasons.append("leaf")
        if any(
            token in name_lower
            for token in ("out", "null", "render", "display", "window", "viewer", "final")
        ):
            score += 2
            reasons.append("output-like-name")
        if score <= 0:
            continue
        candidate = {
            "path": node.get("path", ""),
            "name": node.get("name", ""),
            "type": node.get("type", ""),
            "family": node.get("family", ""),
            "score": score,
            "reasons": reasons,
        }
        if "width" in node:
            candidate["width"] = node.get("width")
            candidate["height"] = node.get("height")
        output_candidates.append(candidate)

    output_candidates.sort(key=lambda item: (-item.get("score", 0), item.get("path", "")))
    target_errors = _get_operator_messages(target, "errors")
    target_warnings = _get_operator_messages(target, "warnings")

    activity_prefix = (
        target.path.rstrip("/") + "/" if target.path not in ("", "/") else "/"
    )
    payload = {
        "path": target.path,
        "timestamp": time.time(),
        "project": {
            "name": getattr(project, "name", ""),
            "folder": getattr(project, "folder", ""),
            "tdVersion": getattr(app, "version", ""),
            "tdBuild": getattr(app, "build", ""),
            "timelineFrame": getattr(absTime, "frame", None),
            "timelineSeconds": getattr(absTime, "seconds", 0),
            "fps": getattr(project, "cookRate", 0),
            "realTime": getattr(project, "realTime", None),
        },
        "target": {
            "path": target.path,
            "name": target.name,
            "type": target.type,
            "family": target.family,
            "isCOMP": bool(getattr(target, "isCOMP", False)),
            "errors": target_errors,
            "warnings": target_warnings,
        },
        "graph": {
            "nodeCount": snapshot.get("nodeCount", len(nodes)),
            "connectionCount": len(edges),
            "families": families,
            "roots": root_paths,
            "leaves": leaf_paths,
            "isolated": isolated_paths,
            "dataFlow": data_flow,
        },
        "nodes": semantic_nodes,
        "connections": edges,
        "issues": {
            "targetErrors": target_errors,
            "targetWarnings": target_warnings,
            "issueCount": len(issue_nodes),
            "nodes": issue_nodes[:20],
        },
        "outputs": output_candidates[:10],
        "performance": _build_monitor_metrics(target, limit=10),
        "recentActivity": [
            event
            for event in _read_log_events(limit=100)
            if event.get("targetPath", "").startswith(activity_prefix)
            or event.get("targetPath", "") == target.path
        ][-10:],
        "snapshotSummary": {
            "version": snapshot.get("version", 2),
            "warningCount": snapshot.get("warningCount", 0),
        },
    }
    if include_snapshot:
        payload["snapshot"] = snapshot
    return payload


def _build_verification_evidence(target, depth=2, include_observation=False):
    evidence = {
        "path": target.path,
        "name": target.name,
        "type": target.type,
        "family": target.family,
        "exists": True,
        "parentPath": target.parent().path
        if hasattr(target, "parent") and target.parent() is not None
        else "",
        "errors": _get_operator_messages(target, "errors"),
        "warnings": _get_operator_messages(target, "warnings"),
        "inputCount": sum(
            len(getattr(connector, "connections", []))
            for connector in getattr(target, "inputConnectors", [])
        ),
        "outputCount": sum(
            len(getattr(connector, "connections", []))
            for connector in getattr(target, "outputConnectors", [])
        ),
    }

    modified_params = []
    expression_params = []
    for p in target.pars() if hasattr(target, "pars") else []:
        try:
            if hasattr(p, "expr") and p.expr:
                expression_params.append(p.name)
            if hasattr(p, "default") and not _snapshot_values_equal(p.val, p.default):
                modified_params.append(p.name)
        except Exception:
            continue

    evidence["modifiedParameters"] = modified_params
    evidence["expressionParameters"] = expression_params
    evidence["childCount"] = (
        len(target.findChildren(depth=1)) if getattr(target, "isCOMP", False) else 0
    )

    if target.family == "TOP":
        evidence["width"] = getattr(target, "width", 0)
        evidence["height"] = getattr(target, "height", 0)
    elif target.family == "CHOP":
        evidence["numChannels"] = getattr(target, "numChans", 0)
        evidence["numSamples"] = getattr(target, "numSamples", 0)
        evidence["sampleRate"] = getattr(target, "rate", 0)
    elif target.family == "SOP":
        evidence["numPoints"] = getattr(target, "numPoints", 0)
        evidence["numPrims"] = getattr(target, "numPrims", 0)
        evidence["numVerts"] = getattr(target, "numVertices", 0)
    elif target.family == "DAT":
        dat_snapshot = _snapshot_dat(target)
        evidence["isTable"] = dat_snapshot.get("isTable", False)
        evidence["numRows"] = dat_snapshot.get("numRows", 0)
        evidence["numCols"] = dat_snapshot.get("numCols", 0)
        if dat_snapshot.get("isTable"):
            evidence["tablePreview"] = dat_snapshot.get("table", [])[:5]
        else:
            evidence["contentPreview"] = (dat_snapshot.get("content", "") or "")[:500]
    else:
        try:
            evidence["numPoints"] = target.numPoints()
            evidence["numPrims"] = target.numPrims()
            evidence["numVerts"] = target.numVerts()
        except Exception:
            pass

    if include_observation or getattr(target, "isCOMP", False):
        observation = _build_observation_payload(target, depth=depth, include_snapshot=False)
        evidence["observation"] = {
            "graph": observation.get("graph", {}),
            "issues": observation.get("issues", {}),
            "outputs": observation.get("outputs", []),
            "performance": observation.get("performance", {}),
        }
    return evidence


def _evaluate_assertion(target, evidence, assertion):
    kind = assertion.get("kind", "")
    passed = True
    actual = None
    details = []

    def _normalized(value):
        return _snapshot_value(value)[0]

    def _compare(value):
        reasons = []
        normalized_value = _normalized(value)
        if "equals" in assertion and normalized_value != _normalized(assertion.get("equals")):
            reasons.append(f"expected {assertion.get('equals')}, got {value}")
        if "min" in assertion:
            try:
                if value < assertion.get("min"):
                    reasons.append(f"expected >= {assertion.get('min')}, got {value}")
            except Exception:
                reasons.append("min comparison unsupported")
        if "max" in assertion:
            try:
                if value > assertion.get("max"):
                    reasons.append(f"expected <= {assertion.get('max')}, got {value}")
            except Exception:
                reasons.append("max comparison unsupported")
        if "contains" in assertion:
            container = value if isinstance(value, (list, tuple, str)) else str(value)
            if assertion.get("contains") not in container:
                reasons.append(f"missing {assertion.get('contains')}")
        if "oneOf" in assertion and value not in assertion.get("oneOf", []):
            reasons.append(f"{value} not in {assertion.get('oneOf', [])}")
        return reasons

    if kind == "exists":
        actual = True
        details = _compare(True)
    elif kind == "family":
        actual = evidence.get("family")
        details = _compare(actual)
    elif kind == "type":
        actual = evidence.get("type")
        details = _compare(actual)
    elif kind == "name":
        actual = evidence.get("name")
        details = _compare(actual)
    elif kind == "param":
        param_name = assertion.get("name", "")
        param = getattr(target.par, param_name, None)
        if param is None:
            actual = None
            details = [f"parameter not found: {param_name}"]
        else:
            actual = param.val
            details = _compare(actual)
    elif kind == "expression":
        param_name = assertion.get("name", "")
        param = getattr(target.par, param_name, None)
        if param is None:
            actual = None
            details = [f"parameter not found: {param_name}"]
        else:
            actual = getattr(param, "expr", "") or ""
            details = _compare(actual)
    elif kind == "childCount":
        actual = evidence.get("childCount", 0)
        details = _compare(actual)
    elif kind == "nodeCount":
        actual = evidence.get("observation", {}).get("graph", {}).get("nodeCount", 0)
        details = _compare(actual)
    elif kind == "errorCount":
        actual = len(evidence.get("errors", []))
        details = _compare(actual)
    elif kind == "warningCount":
        actual = len(evidence.get("warnings", []))
        details = _compare(actual)
    elif kind == "inputCount":
        actual = evidence.get("inputCount", 0)
        details = _compare(actual)
    elif kind == "outputCount":
        actual = evidence.get("outputCount", 0)
        details = _compare(actual)
    elif kind == "hasChild":
        name = assertion.get("name", "")
        path = assertion.get("path", "")
        children = target.findChildren(depth=1) if getattr(target, "isCOMP", False) else []
        actual = [
            child.path
            for child in children
            if (name and child.name == name) or (path and child.path == path)
        ]
        details = [] if actual else [f"child not found: {path or name}"]
    else:
        actual = None
        details = [f"unsupported assertion kind: {kind}"]

    passed = len(details) == 0
    return {
        "kind": kind,
        "passed": passed,
        "actual": actual,
        "expected": {
            key: assertion[key]
            for key in ("equals", "min", "max", "contains", "oneOf", "name", "path")
            if key in assertion
        },
        "details": details,
    }


def _list_harness_history(limit=20, target_path=""):
    iterations_dir = _harness_iterations_dir()
    if not os.path.isdir(iterations_dir):
        return []

    records = []
    for entry in os.listdir(iterations_dir):
        if not entry.endswith(".json"):
            continue
        path = os.path.join(iterations_dir, entry)
        try:
            record = _read_json_file(path)
        except Exception:
            continue
        if target_path and record.get("targetPath", "") != target_path:
            continue
        records.append(
            {
                "id": record.get("id", entry[:-5]),
                "createdAt": record.get("createdAt", 0),
                "updatedAt": record.get("updatedAt", record.get("createdAt", 0)),
                "status": record.get("status", ""),
                "targetPath": record.get("targetPath", ""),
                "goal": record.get("goal", ""),
                "iteration": record.get("iteration"),
                "operationCount": len(record.get("operations", [])),
                "failureCount": len(
                    [result for result in record.get("results", []) if not result.get("success", False)]
                ),
                "beforeSummary": record.get("beforeSummary", {}),
                "afterSummary": record.get("afterSummary", {}),
                "recordPath": path,
                "rolledBackAt": record.get("rolledBackAt"),
            }
        )
    records.sort(key=lambda item: item.get("createdAt", 0), reverse=True)
    return records[:limit]


# --- exec ---


def handle_exec(body):
    """Execute arbitrary Python code in TouchDesigner."""
    code = body.get("code", "")
    if not code:
        return _error("No code provided")

    old_stdout = sys.stdout
    old_stderr = sys.stderr
    captured_out = io.StringIO()
    captured_err = io.StringIO()
    sys.stdout = captured_out
    sys.stderr = captured_err

    result_value = None
    try:
        import td as _td

        exec_globals = globals().copy()
        exec_globals["td"] = _td
        exec_globals["_T"] = lambda n: getattr(_td, n)
        if code.strip().startswith("return"):
            wrapped = "def __td_cli_exec__():\n"
            for line in code.split("\n"):
                wrapped += "    " + line + "\n"
            namespace = {}
            exec(wrapped, exec_globals, namespace)
            result_value = namespace["__td_cli_exec__"]()
        else:
            exec(code, exec_globals)
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
    for i, inp in enumerate(dst_op.inputConnectors):
        for c in inp.connections:
            if c.owner == src_op:
                inp.disconnect()
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

    if not hasattr(parent, 'findChildren'):
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
    depth = body.get("depth", 1)
    include_params = body.get("includeParams", True)

    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")

    children = target.findChildren(depth=depth)

    nodes = []
    edges = []
    families = {}

    for child in children:
        if child.name == "TDCliServer":
            continue

        node_info = {
            "name": child.name,
            "path": child.path,
            "type": child.type,
            "family": child.family,
            "keyParams": {},
        }

        if include_params:
            for p in child.pars():
                try:
                    if str(p.val) != str(p.default):
                        val_str = str(p.val)
                        if len(val_str) > 100:
                            val_str = val_str[:100] + "..."
                        node_info["keyParams"][p.name] = val_str
                except Exception:
                    pass

        nodes.append(node_info)
        families[child.family] = families.get(child.family, 0) + 1

        for i, conn in enumerate(child.outputConnectors):
            for c in conn.connections:
                edges.append(
                    {
                        "from": child.name,
                        "to": c.owner.name,
                        "fromIndex": i,
                    }
                )

    sibling_names = {n["name"] for n in nodes}
    roots = []
    for n in nodes:
        has_input = any(
            e["to"] == n["name"] and e["from"] in sibling_names for e in edges
        )
        if not has_input:
            roots.append(n["name"])

    chains = []

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

    isolated = [
        n["name"]
        for n in nodes
        if not any(e["from"] == n["name"] or e["to"] == n["name"] for e in edges)
    ]

    summary_parts = [f"{len(nodes)} nodes", f"{len(edges)} connections"]
    if isolated:
        summary_parts.append(f"{len(isolated)} isolated")
    summary = ", ".join(summary_parts)

    description = {
        "path": path,
        "nodeCount": len(nodes),
        "families": families,
        "nodes": nodes,
        "connections": edges,
        "dataFlow": chains,
        "isolatedNodes": isolated,
        "summary": summary,
    }
    return _success(f"Network: {summary}", description)


# --- harness ---


def handle_harness_capabilities(body):
    """Report connector and harness capabilities for agent orchestration."""
    tool_routes = sorted(
        {
            schema.get("route", "")
            for schema in TOOL_SCHEMAS
            if schema.get("route", "")
        }
    )
    route_namespaces = {}
    for route in tool_routes:
        namespace = route.strip("/").split("/", 1)[0] if route.strip("/") else ""
        route_namespaces[namespace] = route_namespaces.get(namespace, 0) + 1

    family_support = {
        "COMP": ["ops/*", "network/*", "backup/*", "harness/*", "tox/*"],
        "TOP": ["screenshot", "media/*", "shaders/apply", "cook/*"],
        "CHOP": ["chop/*", "cook/*"],
        "SOP": ["sop/*", "cook/*"],
        "DAT": ["dat/*", "table/*"],
        "POP": ["pop/*"],
        "TIMELINE": ["timeline/*"],
        "UI": ["ui/*"],
    }

    data = {
        "connector": {
            "name": CONNECTOR_NAME,
            "version": CONNECTOR_VERSION,
            "protocolVersion": PROTOCOL_VERSION,
            "installMode": CONNECTOR_INSTALL_MODE,
        },
        "runtime": {
            "projectName": getattr(project, "name", ""),
            "projectPath": _project_path(),
            "tdVersion": getattr(app, "version", ""),
            "tdBuild": getattr(app, "build", ""),
            "timelineFrame": getattr(absTime, "frame", None),
            "timelineSeconds": getattr(absTime, "seconds", 0),
            "harnessRoot": _harness_root_dir(),
        },
        "tools": {
            "count": len(tool_routes),
            "routes": tool_routes,
            "namespaces": route_namespaces,
        },
        "support": {
            "families": family_support,
            "rollback": True,
            "history": True,
            "observe": True,
            "verify": True,
            "batchRoutes": ["/batch/exec", "/batch/parset"],
        },
    }
    return _success("Harness capabilities", data)


def handle_harness_observe(body):
    """Return semantic network state for an agent loop."""
    path = body.get("path", "/")
    depth = body.get("depth", 2)
    include_snapshot = body.get("includeSnapshot", False)

    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")

    payload = _build_observation_payload(
        target, depth=depth, include_snapshot=include_snapshot
    )
    return _success(
        f"Observed {payload['graph']['nodeCount']} node(s) at {path}", payload
    )


def handle_harness_verify(body):
    """Return evidence and assertion results for a target path."""
    path = body.get("path", "")
    depth = body.get("depth", 2)
    assertions = body.get("assertions", [])
    include_observation = body.get("includeObservation", False)

    if not path:
        return _error("path required")

    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")

    evidence = _build_verification_evidence(
        target, depth=depth, include_observation=include_observation
    )
    assertion_results = [
        _evaluate_assertion(target, evidence, assertion) for assertion in assertions
    ]
    passed = all(result.get("passed", False) for result in assertion_results)

    return _success(
        f"Verification {'passed' if passed else 'failed'} for {path}",
        {
            "path": path,
            "passed": passed,
            "assertionCount": len(assertion_results),
            "passedCount": len(
                [result for result in assertion_results if result.get("passed", False)]
            ),
            "evidence": evidence,
            "assertions": assertion_results,
        },
    )


def handle_harness_apply(body):
    """Snapshot a target path, execute operations, and persist rollback state."""
    target_path = body.get("targetPath", "")
    commands_list = body.get("operations", [])
    snapshot_depth = body.get("snapshotDepth", 20)
    stop_on_error = body.get("stopOnError", True)

    if not target_path:
        return _error("targetPath required")
    if not commands_list:
        return _error("operations list required")

    target = op(target_path)
    if target is None:
        return _error(f"Target not found: {target_path}")
    if _contains_connector_boundary(target):
        return _error(
            "Harness apply cannot target a scope that contains TDCliServer. Use a child COMP scope instead."
        )

    blocked_routes = {"/harness/apply", "/harness/rollback"}
    for command in commands_list:
        route = command.get("route", "")
        if route not in ROUTE_TABLE:
            return _error(f"Unknown route in harness apply: {route}")
        if route in blocked_routes:
            return _error(f"Nested harness mutation is not allowed: {route}")

    before_state = _snapshot_target_state(target, depth=snapshot_depth)
    before_summary = _summarize_target_state(before_state)
    rollback_id = f"{time.time_ns()}-harness"
    record = {
        "id": rollback_id,
        "kind": "harness-iteration",
        "createdAt": time.time(),
        "updatedAt": time.time(),
        "projectName": getattr(project, "name", ""),
        "projectPath": _project_path(),
        "targetPath": target_path,
        "goal": body.get("goal", ""),
        "note": body.get("note", ""),
        "iteration": body.get("iteration"),
        "status": "running",
        "operations": commands_list,
        "snapshotBefore": before_state,
        "beforeSummary": before_summary,
        "results": [],
    }
    rollback_id, record_path = _write_harness_record(record)

    results = []
    failed = False
    for index, command in enumerate(commands_list):
        route = command.get("route", "")
        command_body = command.get("body", {})
        result = handle_request(route, command_body)
        success = bool(result.get("success", False))
        results.append(
            {
                "index": index,
                "route": route,
                "body": command_body,
                "success": success,
                "message": result.get("message", ""),
                "data": result.get("data"),
            }
        )
        if not success:
            failed = True
            if stop_on_error:
                break

    current_target = op(target_path)
    after_summary = (
        _summarize_target_state(_snapshot_target_state(current_target, depth=snapshot_depth))
        if current_target is not None
        else {"path": target_path, "missing": True}
    )

    record.update(
        {
            "updatedAt": time.time(),
            "status": "failed" if failed else "applied",
            "results": results,
            "afterSummary": after_summary,
        }
    )
    _write_json_file(record_path, record)
    _append_harness_event(
        {
            "timestamp": time.time(),
            "id": rollback_id,
            "event": "apply",
            "status": record["status"],
            "targetPath": target_path,
            "operationCount": len(results),
            "projectName": getattr(project, "name", ""),
        }
    )

    response_data = {
        "rollbackId": rollback_id,
        "recordPath": record_path,
        "targetPath": target_path,
        "status": record["status"],
        "beforeSummary": before_summary,
        "afterSummary": after_summary,
        "results": results,
    }
    if failed:
        return _error(f"Harness apply failed for {target_path}", response_data)
    return _success(f"Harness apply completed for {target_path}", response_data)


def handle_harness_rollback(body):
    """Restore a previous harness snapshot by rollback id."""
    rollback_id = body.get("id", "")
    if not rollback_id:
        return _error("id required")

    record, record_path = _read_harness_record(rollback_id)
    if record is None:
        return _error(f"Harness rollback not found: {rollback_id}", {"recordPath": record_path})

    state = record.get("snapshotBefore", {})
    result = _restore_target_state(state)
    record["updatedAt"] = time.time()
    record["rolledBackAt"] = record["updatedAt"]
    record["status"] = "rolled_back" if result.get("success", False) else "rollback_failed"
    record["rollbackResult"] = {
        "success": result.get("success", False),
        "message": result.get("message", ""),
        "data": result.get("data"),
    }
    _write_json_file(record_path, record)
    _append_harness_event(
        {
            "timestamp": time.time(),
            "id": rollback_id,
            "event": "rollback",
            "status": record["status"],
            "targetPath": record.get("targetPath", ""),
            "projectName": getattr(project, "name", ""),
        }
    )

    data = result.get("data", {})
    if not isinstance(data, dict):
        data = {"value": data}
    data.update({"rollbackId": rollback_id, "recordPath": record_path})
    result["data"] = data
    return result


def handle_harness_history(body):
    """List recent harness iterations for the current project."""
    limit = body.get("limit", 20)
    try:
        limit = int(limit)
    except Exception:
        limit = 20
    if limit <= 0:
        limit = 20

    target_path = body.get("targetPath", "")
    history = _list_harness_history(limit=limit, target_path=target_path)
    return _success(f"Found {len(history)} harness iteration(s)", {"iterations": history})


# --- monitor ---


def handle_monitor(body):
    """Collect performance metrics for monitoring."""
    path = body.get("path", "/")

    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")

    metrics = _build_monitor_metrics(target, limit=20)
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


# --- CHOP ---


def handle_chop_info(body):
    path = body.get("path", "")
    if not path:
        return _error("path required")
    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")
    if target.family != "CHOP":
        return _error(f"{path} is not a CHOP")
    chans = []
    for i in range(target.numChans):
        c = target[i]
        chans.append({"name": c.name, "length": len(c.vals)})
    return _success(
        "CHOP info",
        {
            "path": path,
            "name": target.name,
            "type": target.type,
            "numChannels": target.numChans,
            "numSamples": target.numSamples,
            "sampleRate": target.rate,
            "channels": chans,
        },
    )


def handle_chop_channels(body):
    path = body.get("path", "")
    start = body.get("start", 0)
    count = body.get("count", -1)
    if not path:
        return _error("path required")
    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")
    if target.family != "CHOP":
        return _error(f"{path} is not a CHOP")
    channels = []
    for i in range(target.numChans):
        c = target[i]
        vals = list(c.vals)
        if count > 0:
            vals = vals[start : start + count]
        elif start > 0:
            vals = vals[start:]
        channels.append({"name": c.name, "values": vals})
    return _success(f"{len(channels)} channels", {"channels": channels})


def handle_chop_sample(body):
    path = body.get("path", "")
    channel_name = body.get("channel", "")
    index = body.get("index", 0)
    if not path:
        return _error("path required")
    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")
    if target.family != "CHOP":
        return _error(f"{path} is not a CHOP")
    chan = target[channel_name] if channel_name else target[0]
    if chan is None:
        return _error(f"Channel not found: {channel_name}")
    vals = list(chan.vals)
    if index < 0 or index >= len(vals):
        return _error(f"Index {index} out of range (0-{len(vals) - 1})")
    return _success(
        f"Sample {index}",
        {
            "channel": chan.name,
            "index": index,
            "value": vals[index],
        },
    )


# --- SOP ---


def handle_sop_info(body):
    path = body.get("path", "")
    if not path:
        return _error("path required")
    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")
    if target.family != "SOP":
        return _error(f"{path} is not a SOP")
    return _success(
        "SOP info",
        {
            "path": path,
            "name": target.name,
            "type": target.type,
            "numPoints": target.numPoints,
            "numPrims": target.numPrims,
            "numVerts": target.numVertices,
        },
    )


def handle_sop_points(body):
    path = body.get("path", "")
    start = body.get("start", 0)
    limit = body.get("limit", 100)
    if not path:
        return _error("path required")
    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")
    if target.family != "SOP":
        return _error(f"{path} is not a SOP")
    points = []
    end = min(start + limit, target.numPoints)
    for i in range(start, end):
        p = target.points[i]
        points.append(
            {
                "index": i,
                "x": p.x,
                "y": p.y,
                "z": p.z,
            }
        )
    return _success(
        f"{len(points)} points",
        {
            "totalPoints": target.numPoints,
            "start": start,
            "count": len(points),
            "points": points,
        },
    )


def handle_sop_attribs(body):
    path = body.get("path", "")
    if not path:
        return _error("path required")
    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")
    if target.family != "SOP":
        return _error(f"{path} is not a SOP")
    point_attrs = (
        [a.name for a in target.pointAttribs] if hasattr(target, "pointAttribs") else []
    )
    prim_attrs = (
        [a.name for a in target.primAttribs] if hasattr(target, "primAttribs") else []
    )
    vert_attrs = (
        [a.name for a in target.vertAttribs] if hasattr(target, "vertAttribs") else []
    )
    return _success(
        "SOP attributes",
        {
            "pointAttributes": point_attrs,
            "primitiveAttributes": prim_attrs,
            "vertexAttributes": vert_attrs,
        },
    )


# --- POP ---


def _get_pop(target):
    if hasattr(target, "numPoints") and hasattr(target, "points"):
        pop_attr = getattr(target, "pointAttributes", None)
        if pop_attr is not None:
            return True
    return False


def handle_pop_info(body):
    path = body.get("path", "")
    if not path:
        return _error("path required")
    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")
    try:
        num_pts = target.numPoints()
    except Exception:
        return _error(f"{path} is not a POP or does not support POP methods")
    try:
        num_prims = target.numPrims()
    except Exception:
        num_prims = 0
    try:
        num_verts = target.numVerts()
    except Exception:
        num_verts = 0
    dim = str(target.dimension) if hasattr(target, "dimension") else ""
    pt_attrs = (
        list(target.pointAttributes) if hasattr(target, "pointAttributes") else []
    )
    pr_attrs = list(target.primAttributes) if hasattr(target, "primAttributes") else []
    vt_attrs = list(target.vertAttributes) if hasattr(target, "vertAttributes") else []
    return _success(
        "POP info",
        {
            "path": path,
            "name": target.name,
            "type": target.type,
            "numPoints": num_pts,
            "numPrims": num_prims,
            "numVerts": num_verts,
            "dimension": dim,
            "pointAttributes": [str(a) for a in pt_attrs],
            "primAttributes": [str(a) for a in pr_attrs],
            "vertAttributes": [str(a) for a in vt_attrs],
        },
    )


def handle_pop_points(body):
    path = body.get("path", "")
    attr = body.get("attribute", "P")
    start = body.get("start", 0)
    count = body.get("count", -1)
    if not path:
        return _error("path required")
    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")
    try:
        vals = target.points(attr, startIndex=start, count=count)
    except Exception as e:
        return _error(f"Failed to read POP points: {e}")
    return _success(
        f"POP points ({attr})",
        {
            "attribute": attr,
            "start": start,
            "count": len(vals) if isinstance(vals, list) else -1,
            "values": list(vals) if isinstance(vals, (list, tuple)) else str(vals),
        },
    )


def handle_pop_prims(body):
    path = body.get("path", "")
    attr = body.get("attribute", "N")
    start = body.get("start", 0)
    count = body.get("count", -1)
    if not path:
        return _error("path required")
    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")
    try:
        vals = target.prims(attr, startIndex=start, count=count)
    except Exception as e:
        return _error(f"Failed to read POP prims: {e}")
    return _success(
        f"POP prims ({attr})",
        {
            "attribute": attr,
            "start": start,
            "count": len(vals) if isinstance(vals, list) else -1,
            "values": list(vals) if isinstance(vals, (list, tuple)) else str(vals),
        },
    )


def handle_pop_verts(body):
    path = body.get("path", "")
    attr = body.get("attribute", "uv")
    start = body.get("start", 0)
    count = body.get("count", -1)
    if not path:
        return _error("path required")
    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")
    try:
        vals = target.verts(attr, startIndex=start, count=count)
    except Exception as e:
        return _error(f"Failed to read POP verts: {e}")
    return _success(
        f"POP verts ({attr})",
        {
            "attribute": attr,
            "start": start,
            "count": len(vals) if isinstance(vals, list) else -1,
            "values": list(vals) if isinstance(vals, (list, tuple)) else str(vals),
        },
    )


def handle_pop_bounds(body):
    path = body.get("path", "")
    if not path:
        return _error("path required")
    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")
    try:
        b = target.bounds()
    except Exception as e:
        return _error(f"Failed to get POP bounds: {e}")
    return _success(
        "POP bounds",
        {
            "minX": b.min.x,
            "minY": b.min.y,
            "minZ": b.min.z,
            "maxX": b.max.x,
            "maxY": b.max.y,
            "maxZ": b.max.z,
            "centerX": b.center.x,
            "centerY": b.center.y,
            "centerZ": b.center.z,
            "sizeX": b.size.x,
            "sizeY": b.size.y,
            "sizeZ": b.size.z,
        },
    )


def handle_pop_attributes(body):
    path = body.get("path", "")
    if not path:
        return _error("path required")
    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")
    pt_attrs = (
        list(target.pointAttributes) if hasattr(target, "pointAttributes") else []
    )
    pr_attrs = list(target.primAttributes) if hasattr(target, "primAttributes") else []
    vt_attrs = list(target.vertAttributes) if hasattr(target, "vertAttributes") else []
    return _success(
        "POP attributes",
        {
            "pointAttributes": [str(a) for a in pt_attrs],
            "primAttributes": [str(a) for a in pr_attrs],
            "vertAttributes": [str(a) for a in vt_attrs],
        },
    )


def handle_pop_save(body):
    path = body.get("path", "")
    filepath = body.get("filepath", "")
    if not path:
        return _error("path required")
    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")
    try:
        saved = target.save(filepath) if filepath else target.save()
    except Exception as e:
        return _error(f"Failed to save POP: {e}")
    return _success("POP saved", {"filepath": str(saved)})


# --- Table DAT ---


def handle_table_rows(body):
    path = body.get("path", "")
    start = body.get("start", 0)
    end = body.get("end", -1)
    if not path:
        return _error("path required")
    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")
    if not target.isTable:
        return _error(f"{path} is not a Table DAT")
    last = target.numRows if end < 0 else min(end, target.numRows)
    rows = []
    for r in range(start, last):
        row = []
        for c in range(target.numCols):
            row.append(str(target[r, c]))
        rows.append(row)
    return _success(
        f"{len(rows)} rows",
        {
            "numRows": target.numRows,
            "numCols": target.numCols,
            "rows": rows,
        },
    )


def handle_table_cell(body):
    path = body.get("path", "")
    row = body.get("row", 0)
    col = body.get("col", 0)
    value = body.get("value", None)
    if not path:
        return _error("path required")
    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")
    if not target.isTable:
        return _error(f"{path} is not a Table DAT")
    if value is not None:
        target[row, col] = value
    return _success(
        f"Cell [{row},{col}]",
        {
            "row": row,
            "col": col,
            "value": str(target[row, col]),
        },
    )


def handle_table_append(body):
    path = body.get("path", "")
    mode = body.get("mode", "row")
    values = body.get("values", [])
    if not path:
        return _error("path required")
    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")
    if not target.isTable:
        return _error(f"{path} is not a Table DAT")
    if mode == "row":
        if values:
            target.appendRow(values)
        else:
            target.appendRow([""] * target.numCols)
        return _success("Row appended", {"numRows": target.numRows})
    elif mode == "col":
        if values:
            target.appendCol(values)
        else:
            target.appendCol([""] * target.numRows)
        return _success("Col appended", {"numCols": target.numCols})
    else:
        return _error(f"Unknown mode: {mode} (use row or col)")


def handle_table_delete(body):
    path = body.get("path", "")
    mode = body.get("mode", "row")
    index = body.get("index", -1)
    if not path:
        return _error("path required")
    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")
    if not target.isTable:
        return _error(f"{path} is not a Table DAT")
    if index < 0:
        index = (target.numRows if mode == "row" else target.numCols) - 1
    if mode == "row":
        target.deleteRow(index)
        return _success(f"Row {index} deleted", {"numRows": target.numRows})
    elif mode == "col":
        target.deleteCol(index)
        return _success(f"Col {index} deleted", {"numCols": target.numCols})
    else:
        return _error(f"Unknown mode: {mode} (use row or col)")


# --- timeline ---


def handle_timeline_info(body):
    tl = root.time
    return _success(
        "Timeline info",
        {
            "currentTime": tl.frame / tl.rate if tl.rate > 0 else 0,
            "start": tl.start,
            "end": tl.end,
            "rangeStart": tl.rangeStart,
            "rangeEnd": tl.rangeEnd,
            "rate": tl.rate,
            "isPlaying": tl.play,
            "isLooping": tl.loop,
            "currentFrame": tl.frame,
        },
    )


def handle_timeline_play(body):
    root.time.play = True
    return _success("Timeline playing")


def handle_timeline_pause(body):
    root.time.play = False
    return _success("Timeline paused")


def handle_timeline_seek(body):
    t = body.get("time")
    frame = body.get("frame")
    tl = root.time
    if frame is not None:
        tl.frame = int(frame)
    elif t is not None:
        tl.frame = int(float(t) * tl.rate)
    else:
        return _error("time or frame required")
    return _success("Seeked", {"frame": tl.frame})


def handle_timeline_range(body):
    start = body.get("start")
    end = body.get("end")
    tl = root.time
    if start is not None:
        tl.rangeStart = int(float(start) * tl.rate)
    if end is not None:
        tl.rangeEnd = int(float(end) * tl.rate)
    return _success(
        "Range set",
        {"rangeStart": tl.rangeStart, "rangeEnd": tl.rangeEnd},
    )


def handle_timeline_rate(body):
    rate = body.get("rate")
    if rate is None:
        return _error("rate required")
    root.time.rate = float(rate)
    return _success("Rate set", {"rate": root.time.rate})


# --- cook ---


def handle_cook_node(body):
    path = body.get("path", "")
    if not path:
        return _error("path required")
    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")
    t0 = time.time()
    target.cook(force=True)
    elapsed = (time.time() - t0) * 1000
    return _success(
        "Cooked", {"path": path, "cookTime": round(elapsed, 2), "cooked": True}
    )


def handle_cook_network(body):
    path = body.get("path", "/")
    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")
    children = target.findChildren(depth=1)
    t0 = time.time()
    count = 0
    for child in children:
        try:
            child.cook(force=True)
            count += 1
        except Exception:
            pass
    elapsed = (time.time() - t0) * 1000
    return _success(
        "Network cooked",
        {"path": path, "nodesCooked": count, "totalTime": round(elapsed, 2)},
    )


# --- ui ---


def _first_network_pane():
    panes = getattr(ui, "panes", [])
    if panes:
        return panes[0]
    return None


def _navigate_to_operator(target):
    """Best-effort navigation that works across TD builds."""
    pane = _first_network_pane()
    if pane is None:
        return False

    try:
        if getattr(target, "isCOMP", False):
            pane.owner = target
        else:
            parent = target.parent() if hasattr(target, "parent") else None
            pane.owner = parent or target
        return True
    except Exception:
        return False


def _select_operator(target):
    parent = target.parent() if hasattr(target, "parent") else None
    if parent is not None:
        for child in getattr(parent, "children", []):
            try:
                child.selected = False
            except Exception:
                pass
    try:
        target.selected = True
        return True
    except Exception:
        return False


def handle_ui_navigate(body):
    path = body.get("path", "")
    if not path:
        return _error("path required")
    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")
    if not _navigate_to_operator(target):
        return _error("Could not navigate: no usable network pane found")
    return _success(f"Navigated to {path}")


def handle_ui_select(body):
    path = body.get("path", "")
    if not path:
        return _error("path required")
    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")
    _navigate_to_operator(target)
    if not _select_operator(target):
        return _error(f"Could not select {path}")
    return _success(f"Selected {path}")


def handle_ui_pulse(body):
    path = body.get("path", "")
    name = body.get("name", "")
    if not path or not name:
        return _error("path and name required")
    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")
    p = target.par[name]
    if p is None:
        return _error(f"Parameter not found: {name}")
    p.pulse()
    return _success(f"Pulsed {path}.{name}")


# --- batch ---


def handle_batch_exec(body):
    commands_list = body.get("commands", [])
    if not commands_list:
        return _error("commands list required")
    t0 = time.time()
    results = []
    success_count = 0
    for cmd in commands_list:
        route = cmd.get("route", "")
        cmd_body = cmd.get("body", {})
        result = handle_request(route, cmd_body)
        ok = result.get("success", False)
        if ok:
            success_count += 1
        results.append(
            {"route": route, "success": ok, "message": result.get("message", "")}
        )
    elapsed = (time.time() - t0) * 1000
    return _success(
        f"Batch: {success_count}/{len(commands_list)} succeeded",
        {
            "total": len(commands_list),
            "success": success_count,
            "failed": len(commands_list) - success_count,
            "duration": round(elapsed, 2),
            "results": results,
        },
    )


def handle_batch_parset(body):
    sets = body.get("sets", [])
    if not sets:
        return _error("sets list required")
    success_count = 0
    for item in sets:
        path = item.get("path", "")
        params = item.get("params", {})
        result = handle_par_set({"path": path, "params": params})
        if result.get("success", False):
            success_count += 1
    return _success(
        f"Batch par set: {success_count}/{len(sets)}",
        {
            "total": len(sets),
            "success": success_count,
            "failed": len(sets) - success_count,
        },
    )


# --- media ---


def handle_media_info(body):
    path = body.get("path", "")
    if not path:
        return _error("path required")
    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")
    info = {"path": path, "type": target.type}
    if hasattr(target, "par"):
        fp = getattr(target.par, "file", None)
        if fp is not None:
            info["filePath"] = fp.eval()
    if target.isTOP:
        info["width"] = target.width
        info["height"] = target.height
    elif target.isCHOP:
        info["numChannels"] = target.numChans
        info["numSamples"] = target.numSamples
        info["sampleRate"] = target.rate
    elif target.isSOP:
        info["numPoints"] = target.numPoints
        info["numPrims"] = target.numPrims
    return _success("Media info", info)


def handle_media_export(body):
    path = body.get("path", "")
    output_file = body.get("outputFile", "")
    if not path:
        return _error("path required")
    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")
    if not output_file:
        output_file = tempfile.mktemp(suffix=".png")
    if target.isTOP:
        target.save(output_file)
    else:
        return _error(f"Export not supported for {target.family}")
    return _success("Exported", {"outputPath": output_file})


def handle_media_record(body):
    path = body.get("path", "")
    start = body.get("start", 0)
    end = body.get("end", 0)
    if not path:
        return _error("path required")
    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")
    rec = op("/project1").create(recordCHOP)
    if rec is None:
        return _error("Could not create Record CHOP")
    if start > 0:
        rec.par.start.pulse()
    rec.par.record.pulse()
    return _success("Recording started", {"path": rec.path})


def handle_media_snapshot(body):
    path = body.get("path", "")
    output_file = body.get("outputFile", "")
    if not path:
        return _error("path required")
    target = op(path)
    if target is None:
        return _error(f"Operator not found: {path}")
    if not output_file:
        output_file = tempfile.mktemp(suffix=".png")
    if target.isTOP:
        target.save(output_file)
        return _success("Snapshot saved", {"outputPath": output_file})
    return _error(f"Snapshot not supported for {target.family}")


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
    {
        "name": "chop/info",
        "route": "/chop/info",
        "description": "Get CHOP info (channel count, sample rate, sample count)",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "CHOP operator path",
            },
        ],
    },
    {
        "name": "chop/channels",
        "route": "/chop/channels",
        "description": "Read all CHOP channel values",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "CHOP operator path",
            },
            {
                "name": "start",
                "type": "integer",
                "required": False,
                "description": "Start sample index",
            },
            {
                "name": "count",
                "type": "integer",
                "required": False,
                "description": "Number of samples (-1 for all)",
            },
        ],
    },
    {
        "name": "chop/sample",
        "route": "/chop/sample",
        "description": "Read a specific sample from a CHOP channel",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "CHOP operator path",
            },
            {
                "name": "channel",
                "type": "string",
                "required": False,
                "description": "Channel name",
            },
            {
                "name": "index",
                "type": "integer",
                "required": False,
                "description": "Sample index",
            },
        ],
    },
    {
        "name": "sop/info",
        "route": "/sop/info",
        "description": "Get SOP info (point/prim/vert counts)",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "SOP operator path",
            },
        ],
    },
    {
        "name": "sop/points",
        "route": "/sop/points",
        "description": "Read SOP point positions (paginated)",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "SOP operator path",
            },
            {
                "name": "start",
                "type": "integer",
                "required": False,
                "description": "Start index (default: 0)",
            },
            {
                "name": "limit",
                "type": "integer",
                "required": False,
                "description": "Max points (default: 100)",
            },
        ],
    },
    {
        "name": "sop/attribs",
        "route": "/sop/attribs",
        "description": "List SOP point/prim/vert attributes",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "SOP operator path",
            },
        ],
    },
    {
        "name": "pop/info",
        "route": "/pop/info",
        "description": "Get POP info (GPU geometry: point/prim/vert counts, dimension, attributes)",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "POP operator path",
            },
        ],
    },
    {
        "name": "pop/points",
        "route": "/pop/points",
        "description": "Read POP point attribute values (paginated)",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "POP operator path",
            },
            {
                "name": "attribute",
                "type": "string",
                "required": False,
                "description": "Attribute name (default: P)",
            },
            {
                "name": "start",
                "type": "integer",
                "required": False,
                "description": "Start index (default: 0)",
            },
            {
                "name": "count",
                "type": "integer",
                "required": False,
                "description": "Number of points (-1 for all)",
            },
        ],
    },
    {
        "name": "pop/prims",
        "route": "/pop/prims",
        "description": "Read POP primitive attribute values",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "POP operator path",
            },
            {
                "name": "attribute",
                "type": "string",
                "required": False,
                "description": "Attribute name (default: N)",
            },
            {
                "name": "start",
                "type": "integer",
                "required": False,
                "description": "Start index",
            },
            {
                "name": "count",
                "type": "integer",
                "required": False,
                "description": "Number of prims (-1 for all)",
            },
        ],
    },
    {
        "name": "pop/verts",
        "route": "/pop/verts",
        "description": "Read POP vertex attribute values",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "POP operator path",
            },
            {
                "name": "attribute",
                "type": "string",
                "required": False,
                "description": "Attribute name (default: uv)",
            },
            {
                "name": "start",
                "type": "integer",
                "required": False,
                "description": "Start index",
            },
            {
                "name": "count",
                "type": "integer",
                "required": False,
                "description": "Number of verts (-1 for all)",
            },
        ],
    },
    {
        "name": "pop/bounds",
        "route": "/pop/bounds",
        "description": "Get POP bounding box (min, max, center, size)",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "POP operator path",
            },
        ],
    },
    {
        "name": "pop/attributes",
        "route": "/pop/attributes",
        "description": "List POP point/prim/vert attribute names",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "POP operator path",
            },
        ],
    },
    {
        "name": "pop/save",
        "route": "/pop/save",
        "description": "Save POP geometry to file",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "POP operator path",
            },
            {
                "name": "filepath",
                "type": "string",
                "required": False,
                "description": "Output file path",
            },
        ],
    },
    {
        "name": "table/rows",
        "route": "/table/rows",
        "description": "Read Table DAT rows",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "Table DAT path",
            },
            {
                "name": "start",
                "type": "integer",
                "required": False,
                "description": "Start row (default: 0)",
            },
            {
                "name": "end",
                "type": "integer",
                "required": False,
                "description": "End row (-1 for all)",
            },
        ],
    },
    {
        "name": "table/cell",
        "route": "/table/cell",
        "description": "Get or set a Table DAT cell value",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "Table DAT path",
            },
            {
                "name": "row",
                "type": "integer",
                "required": False,
                "description": "Row index",
            },
            {
                "name": "col",
                "type": "integer",
                "required": False,
                "description": "Column index",
            },
            {
                "name": "value",
                "type": "string",
                "required": False,
                "description": "Set value (get if omitted)",
            },
        ],
    },
    {
        "name": "table/append",
        "route": "/table/append",
        "description": "Append a row or column to a Table DAT",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "Table DAT path",
            },
            {
                "name": "mode",
                "type": "string",
                "required": False,
                "description": "row or col (default: row)",
            },
            {
                "name": "values",
                "type": "array",
                "required": False,
                "description": "Cell values for new row/col",
            },
        ],
    },
    {
        "name": "table/delete",
        "route": "/table/delete",
        "description": "Delete a row or column from a Table DAT",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "Table DAT path",
            },
            {
                "name": "mode",
                "type": "string",
                "required": False,
                "description": "row or col (default: row)",
            },
            {
                "name": "index",
                "type": "integer",
                "required": False,
                "description": "Index (last if omitted)",
            },
        ],
    },
    {
        "name": "timeline/info",
        "route": "/timeline/info",
        "description": "Get timeline state (time, range, rate, playing status)",
        "parameters": [],
    },
    {
        "name": "timeline/play",
        "route": "/timeline/play",
        "description": "Start timeline playback",
        "parameters": [],
    },
    {
        "name": "timeline/pause",
        "route": "/timeline/pause",
        "description": "Pause timeline playback",
        "parameters": [],
    },
    {
        "name": "timeline/seek",
        "route": "/timeline/seek",
        "description": "Seek timeline to a specific time or frame",
        "parameters": [
            {
                "name": "time",
                "type": "number",
                "required": False,
                "description": "Time in seconds",
            },
            {
                "name": "frame",
                "type": "integer",
                "required": False,
                "description": "Frame number",
            },
        ],
    },
    {
        "name": "timeline/range",
        "route": "/timeline/range",
        "description": "Set timeline start/end range",
        "parameters": [
            {
                "name": "start",
                "type": "number",
                "required": False,
                "description": "Start time in seconds",
            },
            {
                "name": "end",
                "type": "number",
                "required": False,
                "description": "End time in seconds",
            },
        ],
    },
    {
        "name": "timeline/rate",
        "route": "/timeline/rate",
        "description": "Set timeline playback rate",
        "parameters": [
            {
                "name": "rate",
                "type": "number",
                "required": True,
                "description": "Playback rate (fps)",
            },
        ],
    },
    {
        "name": "cook/node",
        "route": "/cook/node",
        "description": "Force cook a single operator",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "Operator path to cook",
            },
        ],
    },
    {
        "name": "cook/network",
        "route": "/cook/network",
        "description": "Force cook all operators in a network",
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
        "name": "ui/navigate",
        "route": "/ui/navigate",
        "description": "Navigate the network editor to a path",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "Operator path to navigate to",
            },
        ],
    },
    {
        "name": "ui/select",
        "route": "/ui/select",
        "description": "Select an operator in the network editor",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "Operator path to select",
            },
        ],
    },
    {
        "name": "ui/pulse",
        "route": "/ui/pulse",
        "description": "Pulse a parameter via UI",
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
        ],
    },
    {
        "name": "batch/exec",
        "route": "/batch/exec",
        "description": "Execute multiple commands in sequence",
        "parameters": [
            {
                "name": "commands",
                "type": "array",
                "required": True,
                "description": "List of {route, body} objects",
            },
        ],
    },
    {
        "name": "batch/parset",
        "route": "/batch/parset",
        "description": "Set parameters on multiple operators in one call",
        "parameters": [
            {
                "name": "sets",
                "type": "array",
                "required": True,
                "description": "List of {path, params} objects",
            },
        ],
    },
    {
        "name": "media/info",
        "route": "/media/info",
        "description": "Get media info (resolution, duration, codec) for TOP/CHOP/SOP",
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
        "name": "media/export",
        "route": "/media/export",
        "description": "Export TOP as image file",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "TOP operator path",
            },
            {
                "name": "outputFile",
                "type": "string",
                "required": True,
                "description": "Output file path",
            },
        ],
    },
    {
        "name": "media/record",
        "route": "/media/record",
        "description": "Start recording",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "Operator path",
            },
            {
                "name": "start",
                "type": "number",
                "required": False,
                "description": "Start time",
            },
            {
                "name": "end",
                "type": "number",
                "required": False,
                "description": "End time",
            },
        ],
    },
    {
        "name": "media/snapshot",
        "route": "/media/snapshot",
        "description": "Capture a snapshot of a TOP",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "TOP operator path",
            },
            {
                "name": "outputFile",
                "type": "string",
                "required": False,
                "description": "Output file path",
            },
        ],
    },
    {
        "name": "harness/capabilities",
        "route": "/harness/capabilities",
        "description": "Report connector, protocol, tool, and family support for agent harnesses",
        "parameters": [],
    },
    {
        "name": "harness/observe",
        "route": "/harness/observe",
        "description": "Return semantic network state for an agent loop",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": False,
                "description": "Target path to observe (default: /)",
            },
            {
                "name": "depth",
                "type": "integer",
                "required": False,
                "description": "Observation depth (default: 2)",
            },
            {
                "name": "includeSnapshot",
                "type": "boolean",
                "required": False,
                "description": "Include structural snapshot payload",
            },
        ],
    },
    {
        "name": "harness/verify",
        "route": "/harness/verify",
        "description": "Return assertion-friendly evidence for a target path",
        "parameters": [
            {
                "name": "path",
                "type": "string",
                "required": True,
                "description": "Target path to verify",
            },
            {
                "name": "depth",
                "type": "integer",
                "required": False,
                "description": "Observation depth for COMP targets (default: 2)",
            },
            {
                "name": "assertions",
                "type": "array",
                "required": False,
                "description": "Assertions such as exists/family/type/param/childCount/nodeCount/errorCount",
            },
            {
                "name": "includeObservation",
                "type": "boolean",
                "required": False,
                "description": "Include observation summary in evidence",
            },
        ],
    },
    {
        "name": "harness/apply",
        "route": "/harness/apply",
        "description": "Snapshot a target, execute existing route operations, and persist rollback state",
        "parameters": [
            {
                "name": "targetPath",
                "type": "string",
                "required": True,
                "description": "Target path whose pre-state should be snapshotted",
            },
            {
                "name": "operations",
                "type": "array",
                "required": True,
                "description": "Array of {route, body} operations using existing routes",
            },
            {
                "name": "snapshotDepth",
                "type": "integer",
                "required": False,
                "description": "Snapshot depth used for rollback capture (default: 20)",
            },
            {
                "name": "stopOnError",
                "type": "boolean",
                "required": False,
                "description": "Stop execution after the first failed operation (default: true)",
            },
            {
                "name": "goal",
                "type": "string",
                "required": False,
                "description": "Optional goal label stored with the harness iteration",
            },
        ],
    },
    {
        "name": "harness/rollback",
        "route": "/harness/rollback",
        "description": "Restore a prior harness snapshot by rollback id",
        "parameters": [
            {
                "name": "id",
                "type": "string",
                "required": True,
                "description": "Rollback id returned from harness/apply",
            },
        ],
    },
    {
        "name": "harness/history",
        "route": "/harness/history",
        "description": "List recent harness iterations for the current project",
        "parameters": [
            {
                "name": "limit",
                "type": "integer",
                "required": False,
                "description": "Maximum number of iterations to return (default: 20)",
            },
            {
                "name": "targetPath",
                "type": "string",
                "required": False,
                "description": "Optional exact target path filter",
            },
        ],
    },
]


def handle_tools_list(body):
    """Return schemas for all registered tools, enabling AI agent discovery."""
    return _success(f"{len(TOOL_SCHEMAS)} tools available", {"tools": TOOL_SCHEMAS})


ROUTE_TABLE = {
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
    "/harness/capabilities": handle_harness_capabilities,
    "/harness/observe": handle_harness_observe,
    "/harness/verify": handle_harness_verify,
    "/harness/apply": handle_harness_apply,
    "/harness/rollback": handle_harness_rollback,
    "/harness/history": handle_harness_history,
    "/monitor": handle_monitor,
    "/shaders/apply": handle_shaders_apply,
    "/tox/export": handle_tox_export,
    "/tox/import": handle_tox_import,
    "/tools/list": handle_tools_list,
    "/chop/info": handle_chop_info,
    "/chop/channels": handle_chop_channels,
    "/chop/sample": handle_chop_sample,
    "/sop/info": handle_sop_info,
    "/sop/points": handle_sop_points,
    "/sop/attribs": handle_sop_attribs,
    "/pop/info": handle_pop_info,
    "/pop/points": handle_pop_points,
    "/pop/prims": handle_pop_prims,
    "/pop/verts": handle_pop_verts,
    "/pop/bounds": handle_pop_bounds,
    "/pop/attributes": handle_pop_attributes,
    "/pop/save": handle_pop_save,
    "/table/rows": handle_table_rows,
    "/table/cell": handle_table_cell,
    "/table/append": handle_table_append,
    "/table/delete": handle_table_delete,
    "/timeline/info": handle_timeline_info,
    "/timeline/play": handle_timeline_play,
    "/timeline/pause": handle_timeline_pause,
    "/timeline/seek": handle_timeline_seek,
    "/timeline/range": handle_timeline_range,
    "/timeline/rate": handle_timeline_rate,
    "/cook/node": handle_cook_node,
    "/cook/network": handle_cook_network,
    "/ui/navigate": handle_ui_navigate,
    "/ui/select": handle_ui_select,
    "/ui/pulse": handle_ui_pulse,
    "/batch/exec": handle_batch_exec,
    "/batch/parset": handle_batch_parset,
    "/media/info": handle_media_info,
    "/media/export": handle_media_export,
    "/media/record": handle_media_record,
    "/media/snapshot": handle_media_snapshot,
}
