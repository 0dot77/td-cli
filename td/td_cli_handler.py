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
]


def handle_tools_list(body):
    """Return schemas for all registered tools, enabling AI agent discovery."""
    return _success(f'{len(TOOL_SCHEMAS)} tools available', {'tools': TOOL_SCHEMAS})
