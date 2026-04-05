# td-cli Request Handler
# Place this code in a Text DAT named 'handler' inside TDCliServer COMP
# This module routes requests to the appropriate tool handlers

import json
import sys
import io
import traceback


def handle_request(uri, body):
    """Route request to appropriate handler based on URI."""
    routes = {
        '/exec': handle_exec,
        '/ops/list': handle_ops_list,
        '/ops/create': handle_ops_create,
        '/ops/delete': handle_ops_delete,
        '/ops/info': handle_ops_info,
        '/par/get': handle_par_get,
        '/par/set': handle_par_set,
        '/connect': handle_connect,
        '/disconnect': handle_disconnect,
        '/dat/read': handle_dat_read,
        '/dat/write': handle_dat_write,
        '/project/info': handle_project_info,
        '/project/save': handle_project_save,
        '/screenshot': handle_screenshot,
        '/network/export': handle_network_export,
        '/network/import': handle_network_import,
        '/network/describe': handle_network_describe,
        '/monitor': handle_monitor,
        '/shaders/apply': handle_shaders_apply,
        '/tools/list': handle_tools_list,
    }

    handler = routes.get(uri)
    if handler is None:
        return {
            'success': False,
            'message': f'Unknown route: {uri}',
            'data': None
        }

    try:
        return handler(body)
    except Exception as e:
        return {
            'success': False,
            'message': str(e),
            'data': {'traceback': traceback.format_exc()}
        }


def _success(message, data=None):
    return {'success': True, 'message': message, 'data': data}


def _error(message, data=None):
    return {'success': False, 'message': message, 'data': data}


# --- exec ---

def handle_exec(body):
    """Execute arbitrary Python code in TouchDesigner."""
    code = body.get('code', '')
    if not code:
        return _error('No code provided')

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
        if code.strip().startswith('return'):
            wrapped = f"def __td_cli_exec__():\n"
            for line in code.split('\n'):
                wrapped += f"    {line}\n"
            exec(wrapped)
            result_value = locals()['__td_cli_exec__']()
        else:
            exec(code)
    except Exception as e:
        sys.stdout = old_stdout
        sys.stderr = old_stderr
        return _error(f'Execution error: {e}', {
            'stdout': captured_out.getvalue(),
            'stderr': captured_err.getvalue(),
            'traceback': traceback.format_exc()
        })
    finally:
        sys.stdout = old_stdout
        sys.stderr = old_stderr

    return _success('Script executed', {
        'result': str(result_value) if result_value is not None else None,
        'stdout': captured_out.getvalue(),
        'stderr': captured_err.getvalue(),
    })


# --- ops ---

def handle_ops_list(body):
    """List operators at a given path."""
    path = body.get('path', '/')
    depth = body.get('depth', 1)
    family = body.get('family', None)  # TOP, CHOP, SOP, DAT, COMP, MAT

    target = op(path)
    if target is None:
        return _error(f'Operator not found: {path}')

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
        ops_list.append({
            'path': c.path,
            'name': c.name,
            'type': c.type,
            'family': c.family,
            'nodeX': c.nodeX,
            'nodeY': c.nodeY,
        })

    return _success(f'Found {len(ops_list)} operators', {'operators': ops_list})


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
    op_type = body.get('type', '')
    parent_path = body.get('parent', '/')
    name = body.get('name', None)
    node_x = body.get('nodeX', None)
    node_y = body.get('nodeY', None)

    if not op_type:
        return _error('No operator type provided')

    parent_op = op(parent_path)
    if parent_op is None:
        return _error(f'Parent not found: {parent_path}')

    new_op = parent_op.create(op_type, name)

    # Position the new operator
    px, py = _find_open_position(parent_op, node_x, node_y)
    new_op.nodeX = px
    new_op.nodeY = py

    return _success(f'Created {op_type}', {
        'path': new_op.path,
        'name': new_op.name,
        'type': new_op.type,
        'family': new_op.family,
        'nodeX': new_op.nodeX,
        'nodeY': new_op.nodeY,
    })


def handle_ops_delete(body):
    """Delete an operator."""
    path = body.get('path', '')
    if not path:
        return _error('No operator path provided')

    target = op(path)
    if target is None:
        return _error(f'Operator not found: {path}')

    name = target.name
    target.destroy()

    return _success(f'Deleted {name}')


def handle_ops_info(body):
    """Get detailed info about an operator."""
    path = body.get('path', '')
    if not path:
        return _error('No operator path provided')

    target = op(path)
    if target is None:
        return _error(f'Operator not found: {path}')

    # Get input/output connections
    inputs = []
    for i, conn in enumerate(target.inputConnectors):
        for c in conn.connections:
            inputs.append({'index': i, 'path': c.owner.path, 'name': c.owner.name})

    outputs = []
    for i, conn in enumerate(target.outputConnectors):
        for c in conn.connections:
            outputs.append({'index': i, 'path': c.owner.path, 'name': c.owner.name})

    # Get parameters summary (first 50)
    params = []
    for p in target.pars()[:50]:
        params.append({
            'name': p.name,
            'label': p.label,
            'value': str(p.val),
            'default': str(p.default),
            'mode': str(p.mode),
            'page': p.page.name if p.page else '',
        })

    info = {
        'path': target.path,
        'name': target.name,
        'type': target.type,
        'family': target.family,
        'nodeX': target.nodeX,
        'nodeY': target.nodeY,
        'inputs': inputs,
        'outputs': outputs,
        'parameters': params,
        'errors': target.errors(recurse=False) if hasattr(target, 'errors') else '',
        'warnings': target.warnings(recurse=False) if hasattr(target, 'warnings') else '',
        'comment': target.comment,
    }

    return _success(f'Info for {target.name}', info)


# --- par ---

def handle_par_get(body):
    """Get parameters of an operator."""
    path = body.get('path', '')
    names = body.get('names', None)  # Optional: specific parameter names

    if not path:
        return _error('No operator path provided')

    target = op(path)
    if target is None:
        return _error(f'Operator not found: {path}')

    params = []
    if names:
        for name in names:
            p = getattr(target.par, name, None)
            if p is not None:
                params.append({
                    'name': p.name,
                    'label': p.label,
                    'value': str(p.val),
                    'default': str(p.default),
                    'min': str(p.min) if hasattr(p, 'min') else None,
                    'max': str(p.max) if hasattr(p, 'max') else None,
                    'type': str(type(p.val).__name__),
                    'mode': str(p.mode),
                })
    else:
        for p in target.pars()[:100]:
            params.append({
                'name': p.name,
                'label': p.label,
                'value': str(p.val),
                'default': str(p.default),
                'type': str(type(p.val).__name__),
                'mode': str(p.mode),
            })

    return _success(f'{len(params)} parameters', {'parameters': params})


def handle_par_set(body):
    """Set parameters on an operator."""
    path = body.get('path', '')
    params = body.get('params', {})

    if not path:
        return _error('No operator path provided')
    if not params:
        return _error('No parameters provided')

    target = op(path)
    if target is None:
        return _error(f'Operator not found: {path}')

    updated = []
    for name, value in params.items():
        p = getattr(target.par, name, None)
        if p is None:
            return _error(f'Parameter not found: {name}')
        p.val = value
        updated.append({'name': name, 'value': str(p.val)})

    return _success(f'Updated {len(updated)} parameters', {'updated': updated})


# --- connect / disconnect ---

def handle_connect(body):
    """Connect two operators."""
    src_path = body.get('src', '')
    dst_path = body.get('dst', '')
    src_index = body.get('srcIndex', 0)
    dst_index = body.get('dstIndex', 0)

    if not src_path or not dst_path:
        return _error('Both src and dst paths required')

    src_op = op(src_path)
    dst_op = op(dst_path)

    if src_op is None:
        return _error(f'Source not found: {src_path}')
    if dst_op is None:
        return _error(f'Destination not found: {dst_path}')

    src_op.outputConnectors[src_index].connect(dst_op.inputConnectors[dst_index])

    return _success(f'Connected {src_op.name} -> {dst_op.name}')


def handle_disconnect(body):
    """Disconnect two operators."""
    src_path = body.get('src', '')
    dst_path = body.get('dst', '')

    if not src_path or not dst_path:
        return _error('Both src and dst paths required')

    src_op = op(src_path)
    dst_op = op(dst_path)

    if src_op is None:
        return _error(f'Source not found: {src_path}')
    if dst_op is None:
        return _error(f'Destination not found: {dst_path}')

    # Find and disconnect the connection
    for conn in src_op.outputConnectors:
        for c in conn.connections:
            if c.owner == dst_op:
                conn.disconnect(c)
                return _success(f'Disconnected {src_op.name} -> {dst_op.name}')

    return _error(f'No connection found between {src_path} and {dst_path}')


# --- dat ---

def handle_dat_read(body):
    """Read DAT content."""
    path = body.get('path', '')
    if not path:
        return _error('No DAT path provided')

    target = op(path)
    if target is None:
        return _error(f'Operator not found: {path}')

    if target.family != 'DAT':
        return _error(f'{path} is not a DAT (family: {target.family})')

    # Check if it's a table DAT
    is_table = target.isTable if hasattr(target, 'isTable') else False

    if is_table:
        rows = []
        for row_idx in range(target.numRows):
            row = []
            for col_idx in range(target.numCols):
                row.append(str(target[row_idx, col_idx]))
            rows.append(row)
        return _success('DAT content read', {
            'content': None,
            'table': rows,
            'numRows': target.numRows,
            'numCols': target.numCols,
            'isTable': True,
        })
    else:
        return _success('DAT content read', {
            'content': target.text,
            'table': None,
            'numRows': target.numRows,
            'numCols': target.numCols,
            'isTable': False,
        })


def handle_dat_write(body):
    """Write DAT content."""
    path = body.get('path', '')
    content = body.get('content', None)
    table = body.get('table', None)

    if not path:
        return _error('No DAT path provided')

    target = op(path)
    if target is None:
        return _error(f'Operator not found: {path}')

    if target.family != 'DAT':
        return _error(f'{path} is not a DAT (family: {target.family})')

    if table is not None:
        target.clear()
        for row in table:
            target.appendRow(row)
        return _success(f'Wrote {len(table)} rows to table DAT')
    elif content is not None:
        target.text = content
        return _success('Wrote content to DAT')
    else:
        return _error('No content or table data provided')


# --- project ---

def handle_project_info(body):
    """Get project metadata."""
    return _success('Project info', {
        'name': project.name,
        'folder': project.folder,
        'saveVersion': project.saveVersion,
        'tdVersion': app.version,
        'tdBuild': app.build,
        'fps': project.cookRate,
        'realTime': project.realTime,
        'timelineFrame': absTime.frame,
        'timelineSeconds': absTime.seconds,
    })


def handle_project_save(body):
    """Save the project."""
    path = body.get('path', None)

    if path:
        project.save(path)
        return _success(f'Project saved to {path}')
    else:
        project.save()
        return _success('Project saved')


# --- screenshot ---

def handle_screenshot(body):
    """Capture a TOP's output as a base64-encoded PNG image."""
    import base64
    import tempfile
    import os

    path = body.get('path', '')

    if not path:
        # Auto-detect: find first null/out TOP in project root
        root = op('/project1')
        if root:
            tops = root.findChildren(family='TOP', depth=1)
            nulls = [t for t in tops if 'null' in t.type.lower() or 'out' in t.name.lower()]
            if nulls:
                target = nulls[0]
            elif tops:
                target = tops[0]
            else:
                return _error('No TOP found and no path specified')
        else:
            return _error('No path specified and /project1 not found')
    else:
        target = op(path)

    if target is None:
        return _error(f'Operator not found: {path}')
    if target.family != 'TOP':
        return _error(f'{path} is not a TOP (family: {target.family})')

    try:
        tmp = tempfile.mktemp(suffix='.png')
        target.save(tmp)
        with open(tmp, 'rb') as f:
            img_bytes = f.read()
        os.remove(tmp)

        b64 = base64.b64encode(img_bytes).decode('ascii')
        return _success('Screenshot captured', {
            'image': b64,
            'width': target.width,
            'height': target.height,
        })
    except Exception as e:
        return _error(f'Screenshot failed: {e}')


# --- network ---

def _serialize_op(o, include_defaults=False):
    """Serialize a single operator to dict."""
    node = {
        'path': o.path,
        'name': o.name,
        'type': o.type,
        'family': o.family,
        'nodeX': o.nodeX,
        'nodeY': o.nodeY,
        'comment': o.comment or '',
        'inputs': [],
        'parameters': {},
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
            node['inputs'].append({
                'index': i,
                'sourcePath': c.owner.path,
                'sourceIndex': src_index,
            })

    # Parameters (non-default only unless include_defaults)
    for p in o.pars():
        try:
            if include_defaults or str(p.val) != str(p.default):
                par_data = {
                    'value': str(p.val),
                    'default': str(p.default),
                    'mode': str(p.mode),
                }
                if hasattr(p, 'expr') and p.expr:
                    par_data['expression'] = p.expr
                node['parameters'][p.name] = par_data
        except Exception:
            pass

    return node


def _walk_network(parent, remaining_depth, include_defaults=False):
    """Recursively walk network and serialize all operators."""
    nodes = []
    if remaining_depth <= 0:
        return nodes
    for child in parent.findChildren(depth=1):
        nodes.append(_serialize_op(child, include_defaults))
        if child.isCOMP and child.name != 'TDCliServer':
            nodes.extend(_walk_network(child, remaining_depth - 1, include_defaults))
    return nodes


def handle_network_export(body):
    """Export network structure as JSON snapshot."""
    path = body.get('path', '/')
    depth = body.get('depth', 10)
    include_defaults = body.get('includeDefaults', False)

    target = op(path)
    if target is None:
        return _error(f'Operator not found: {path}')

    nodes = _walk_network(target, depth, include_defaults)

    snapshot = {
        'version': 1,
        'rootPath': path,
        'exportTime': absTime.seconds,
        'tdVersion': app.version,
        'tdBuild': app.build,
        'nodeCount': len(nodes),
        'nodes': nodes,
    }
    return _success(f'Exported {len(nodes)} nodes', snapshot)


def handle_network_import(body):
    """Recreate network from snapshot JSON."""
    snapshot = body.get('snapshot', {})
    target_path = body.get('targetPath', snapshot.get('rootPath', '/'))

    nodes = snapshot.get('nodes', [])
    parent = op(target_path)
    if parent is None:
        return _error(f'Target not found: {target_path}')

    created = []
    path_map = {}  # old path -> new op

    # Phase 1: Create all operators
    for node in nodes:
        try:
            new_op = parent.create(node['type'], node['name'])
            new_op.nodeX = node.get('nodeX', 0)
            new_op.nodeY = node.get('nodeY', 0)
            if node.get('comment'):
                new_op.comment = node['comment']

            # Set non-default parameters
            for pname, pdata in node.get('parameters', {}).items():
                p = getattr(new_op.par, pname, None)
                if p is not None:
                    try:
                        if pdata.get('expression'):
                            p.expr = pdata['expression']
                        else:
                            p.val = pdata['value']
                    except Exception:
                        pass

            created.append(new_op.path)
            path_map[node['path']] = new_op
        except Exception as e:
            created.append(f'FAILED: {node.get("name", "?")} - {e}')

    # Phase 2: Reconnect wires
    connections_made = 0
    for node in nodes:
        new_dst = path_map.get(node['path'])
        if not new_dst:
            continue
        for inp in node.get('inputs', []):
            src_op = path_map.get(inp['sourcePath'])
            if src_op:
                try:
                    src_idx = inp.get('sourceIndex', 0)
                    dst_idx = inp.get('index', 0)
                    src_op.outputConnectors[src_idx].connect(new_dst.inputConnectors[dst_idx])
                    connections_made += 1
                except Exception:
                    pass

    return _success(f'Imported {len(created)} nodes, {connections_made} connections', {
        'created': created,
        'connections': connections_made,
    })


# --- describe ---

def handle_network_describe(body):
    """Generate AI-friendly description of a network."""
    path = body.get('path', '/')

    target = op(path)
    if target is None:
        return _error(f'Operator not found: {path}')

    children = target.findChildren(depth=1)

    nodes = []
    edges = []
    families = {}

    for child in children:
        if child.name == 'TDCliServer':
            continue

        node_info = {
            'name': child.name,
            'type': child.type,
            'family': child.family,
            'keyParams': {},
        }

        # Non-default parameters
        for p in child.pars():
            try:
                if str(p.val) != str(p.default):
                    node_info['keyParams'][p.name] = str(p.val)
            except Exception:
                pass

        nodes.append(node_info)
        families[child.family] = families.get(child.family, 0) + 1

        # Output connections
        for i, conn in enumerate(child.outputConnectors):
            for c in conn.connections:
                edges.append({
                    'from': child.name,
                    'to': c.owner.name,
                    'fromIndex': i,
                })

    # Find data flow chains (source -> ... -> sink)
    chains = []
    # Find roots (nodes with no inputs from siblings)
    sibling_names = {n['name'] for n in nodes}
    roots = []
    for n in nodes:
        has_input = any(e['to'] == n['name'] and e['from'] in sibling_names for e in edges)
        if not has_input:
            roots.append(n['name'])

    def trace_chain(name, visited=None):
        if visited is None:
            visited = set()
        if name in visited:
            return [name + '(loop)']
        visited.add(name)
        chain = [name]
        outputs = [e['to'] for e in edges if e['from'] == name and e['to'] in sibling_names]
        if outputs:
            for out in outputs:
                chain.extend(trace_chain(out, visited.copy()))
        return chain

    for root in roots:
        chain = trace_chain(root)
        if len(chain) > 1:
            chains.append(' -> '.join(chain))

    description = {
        'path': path,
        'nodeCount': len(nodes),
        'families': families,
        'nodes': nodes,
        'connections': edges,
        'dataFlow': chains,
    }
    return _success(f'Network: {len(nodes)} nodes, {len(edges)} connections', description)


# --- monitor ---

def handle_monitor(body):
    """Collect performance metrics for monitoring."""
    path = body.get('path', '/')

    target = op(path)
    if target is None:
        return _error(f'Operator not found: {path}')

    step = absTime.stepSeconds if hasattr(absTime, 'stepSeconds') and absTime.stepSeconds > 0 else 0.0167

    metrics = {
        'fps': project.cookRate,
        'actualFps': round(1.0 / step, 1) if step > 0 else 0,
        'frame': absTime.frame,
        'seconds': round(absTime.seconds, 2),
        'realTime': project.realTime,
    }

    # Per-child performance metrics
    children = target.findChildren(depth=1)
    child_metrics = []
    for child in children:
        if child.name == 'TDCliServer':
            continue
        cm = {
            'name': child.name,
            'type': child.type,
            'family': child.family,
        }
        if hasattr(child, 'cookTime'):
            cm['cookTime'] = round(child.cookTime() * 1000, 3)  # ms
        if hasattr(child, 'cpuCookTime'):
            cm['cpuCookTime'] = round(child.cpuCookTime() * 1000, 3)
        errs = child.errors(recurse=False) if hasattr(child, 'errors') else ''
        warns = child.warnings(recurse=False) if hasattr(child, 'warnings') else ''
        if errs:
            cm['errors'] = errs
        if warns:
            cm['warnings'] = warns
        child_metrics.append(cm)

    # Sort by cook time descending
    child_metrics.sort(key=lambda x: x.get('cookTime', 0), reverse=True)
    metrics['children'] = child_metrics[:20]

    return _success('Monitor data', metrics)


# --- shaders apply ---

def handle_shaders_apply(body):
    """Apply a shader template to a GLSL TOP."""
    target_path = body.get('path', '')
    glsl_code = body.get('glsl', '')
    uniforms = body.get('uniforms', [])

    if not target_path:
        return _error('No GLSL TOP path specified')
    if not glsl_code:
        return _error('No GLSL code provided')

    target = op(target_path)
    if target is None:
        return _error(f'Operator not found: {target_path}')

    # Find the pixel shader DAT
    pixel_dat_name = target.par.pixeldat.val if hasattr(target.par, 'pixeldat') else None
    if not pixel_dat_name:
        return _error(f'Cannot find pixel shader DAT for {target_path}')

    pixel_dat = op(f'{target.parent().path}/{pixel_dat_name}')
    if pixel_dat is None:
        return _error(f'Pixel shader DAT not found: {pixel_dat_name}')

    # Write the GLSL code
    pixel_dat.text = glsl_code

    # Configure uniforms
    for i, u in enumerate(uniforms):
        if i >= 8:
            break  # GLSL TOP supports up to 8 vec uniforms
        name_par = f'vec{i}name'
        val_par = f'vec{i}valuex'
        if hasattr(target.par, name_par):
            setattr(target.par, name_par, u.get('name', ''))
        if hasattr(target.par, val_par) and u.get('default') is not None:
            try:
                getattr(target.par, val_par).val = float(u['default'])
            except (ValueError, TypeError):
                pass
        # Set expression if provided
        if u.get('expression') and hasattr(target.par, val_par):
            try:
                getattr(target.par, val_par).expr = u['expression']
            except Exception:
                pass

    # Check compile status
    warnings = target.warnings(recurse=False) if hasattr(target, 'warnings') else ''

    return _success('Shader applied', {
        'path': target_path,
        'pixelDat': pixel_dat.path,
        'uniformsSet': len(uniforms),
        'compileWarnings': warnings,
    })


# --- tools ---

# Tool schema registry: each entry describes a route's purpose and parameters.
# This enables AI agents to discover available commands via `td-cli tools list`.
TOOL_SCHEMAS = [
    {
        'name': 'exec',
        'route': '/exec',
        'description': 'Execute arbitrary Python code in TouchDesigner',
        'parameters': [
            {'name': 'code', 'type': 'string', 'required': True, 'description': 'Python code to execute. Prefix with "return" to get a value back.'},
        ],
    },
    {
        'name': 'ops/list',
        'route': '/ops/list',
        'description': 'List operators at a given path',
        'parameters': [
            {'name': 'path', 'type': 'string', 'required': False, 'description': 'Operator path (default: /)'},
            {'name': 'depth', 'type': 'integer', 'required': False, 'description': 'Search depth (default: 1)'},
            {'name': 'family', 'type': 'string', 'required': False, 'description': 'Filter by family: TOP, CHOP, SOP, DAT, COMP, MAT'},
        ],
    },
    {
        'name': 'ops/create',
        'route': '/ops/create',
        'description': 'Create a new operator',
        'parameters': [
            {'name': 'type', 'type': 'string', 'required': True, 'description': 'Operator type (e.g., noiseTOP, waveCHOP)'},
            {'name': 'parent', 'type': 'string', 'required': True, 'description': 'Parent operator path'},
            {'name': 'name', 'type': 'string', 'required': False, 'description': 'Operator name (auto-generated if omitted)'},
            {'name': 'nodeX', 'type': 'integer', 'required': False, 'description': 'X position (auto-placed if omitted)'},
            {'name': 'nodeY', 'type': 'integer', 'required': False, 'description': 'Y position (auto-placed if omitted)'},
        ],
    },
    {
        'name': 'ops/delete',
        'route': '/ops/delete',
        'description': 'Delete an operator',
        'parameters': [
            {'name': 'path', 'type': 'string', 'required': True, 'description': 'Operator path to delete'},
        ],
    },
    {
        'name': 'ops/info',
        'route': '/ops/info',
        'description': 'Get detailed info about an operator (connections, parameters, errors)',
        'parameters': [
            {'name': 'path', 'type': 'string', 'required': True, 'description': 'Operator path'},
        ],
    },
    {
        'name': 'par/get',
        'route': '/par/get',
        'description': 'Get parameters of an operator',
        'parameters': [
            {'name': 'path', 'type': 'string', 'required': True, 'description': 'Operator path'},
            {'name': 'names', 'type': 'array', 'required': False, 'description': 'Specific parameter names (all if omitted)'},
        ],
    },
    {
        'name': 'par/set',
        'route': '/par/set',
        'description': 'Set parameters on an operator',
        'parameters': [
            {'name': 'path', 'type': 'string', 'required': True, 'description': 'Operator path'},
            {'name': 'params', 'type': 'object', 'required': True, 'description': 'Key-value pairs of parameter names and values'},
        ],
    },
    {
        'name': 'connect',
        'route': '/connect',
        'description': 'Connect two operators (wire output to input)',
        'parameters': [
            {'name': 'src', 'type': 'string', 'required': True, 'description': 'Source operator path'},
            {'name': 'dst', 'type': 'string', 'required': True, 'description': 'Destination operator path'},
            {'name': 'srcIndex', 'type': 'integer', 'required': False, 'description': 'Source output index (default: 0)'},
            {'name': 'dstIndex', 'type': 'integer', 'required': False, 'description': 'Destination input index (default: 0)'},
        ],
    },
    {
        'name': 'disconnect',
        'route': '/disconnect',
        'description': 'Disconnect two operators',
        'parameters': [
            {'name': 'src', 'type': 'string', 'required': True, 'description': 'Source operator path'},
            {'name': 'dst', 'type': 'string', 'required': True, 'description': 'Destination operator path'},
        ],
    },
    {
        'name': 'dat/read',
        'route': '/dat/read',
        'description': 'Read DAT content (text or table)',
        'parameters': [
            {'name': 'path', 'type': 'string', 'required': True, 'description': 'DAT operator path'},
        ],
    },
    {
        'name': 'dat/write',
        'route': '/dat/write',
        'description': 'Write content to a DAT',
        'parameters': [
            {'name': 'path', 'type': 'string', 'required': True, 'description': 'DAT operator path'},
            {'name': 'content', 'type': 'string', 'required': False, 'description': 'Text content to write'},
            {'name': 'table', 'type': 'array', 'required': False, 'description': 'Table data as array of rows'},
        ],
    },
    {
        'name': 'project/info',
        'route': '/project/info',
        'description': 'Get project metadata (name, folder, TD version, FPS, timeline)',
        'parameters': [],
    },
    {
        'name': 'project/save',
        'route': '/project/save',
        'description': 'Save the project',
        'parameters': [
            {'name': 'path', 'type': 'string', 'required': False, 'description': 'Save path (current location if omitted)'},
        ],
    },
    {
        'name': 'screenshot',
        'route': '/screenshot',
        'description': 'Capture a TOP output as base64-encoded PNG image',
        'parameters': [
            {'name': 'path', 'type': 'string', 'required': False, 'description': 'TOP operator path (auto-detects if omitted)'},
        ],
    },
    {
        'name': 'network/export',
        'route': '/network/export',
        'description': 'Export network structure as JSON snapshot (operators, connections, parameters)',
        'parameters': [
            {'name': 'path', 'type': 'string', 'required': False, 'description': 'Root path to export (default: /)'},
            {'name': 'depth', 'type': 'integer', 'required': False, 'description': 'Recursion depth (default: 10)'},
            {'name': 'includeDefaults', 'type': 'boolean', 'required': False, 'description': 'Include default parameter values (default: false)'},
        ],
    },
    {
        'name': 'network/import',
        'route': '/network/import',
        'description': 'Recreate network from a JSON snapshot',
        'parameters': [
            {'name': 'snapshot', 'type': 'object', 'required': True, 'description': 'Network snapshot JSON object'},
            {'name': 'targetPath', 'type': 'string', 'required': False, 'description': 'Target parent path (default: from snapshot)'},
        ],
    },
    {
        'name': 'network/describe',
        'route': '/network/describe',
        'description': 'Generate AI-friendly description of a network (nodes, connections, data flow)',
        'parameters': [
            {'name': 'path', 'type': 'string', 'required': False, 'description': 'Network path (default: /)'},
        ],
    },
    {
        'name': 'monitor',
        'route': '/monitor',
        'description': 'Collect performance metrics (FPS, cook time, errors)',
        'parameters': [
            {'name': 'path', 'type': 'string', 'required': False, 'description': 'Root path to monitor (default: /)'},
        ],
    },
    {
        'name': 'shaders/apply',
        'route': '/shaders/apply',
        'description': 'Apply a GLSL shader to a GLSL TOP (write pixel DAT + configure uniforms)',
        'parameters': [
            {'name': 'path', 'type': 'string', 'required': True, 'description': 'GLSL TOP operator path'},
            {'name': 'glsl', 'type': 'string', 'required': True, 'description': 'GLSL pixel shader code'},
            {'name': 'uniforms', 'type': 'array', 'required': False, 'description': 'Uniform definitions [{name, type, default, expression}]'},
        ],
    },
]


def handle_tools_list(body):
    """Return schemas for all registered tools, enabling AI agent discovery."""
    return _success(f'{len(TOOL_SCHEMAS)} tools available', {'tools': TOOL_SCHEMAS})
