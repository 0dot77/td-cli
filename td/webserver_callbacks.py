"""
td-cli Web Server DAT Callbacks
Place this code in the Web Server DAT's callbacks parameter.
The Web Server DAT should be named 'webserver1' inside TDCliServer COMP.
"""

import json
import hmac
import os
import threading
from typing import Dict, Any

# Serialize all command requests to prevent race conditions
# when multiple CLI agents hit the same TD instance concurrently.
# RLock allows reentrant acquisition on the same thread to avoid deadlocks
# if TD's Web Server DAT reenters during a long-running handler.
_request_lock = threading.RLock()


def _get_required_token() -> str:
    return os.environ.get('TD_CLI_TOKEN', '').strip()


def _get_connector_metadata() -> Dict[str, Any]:
    handler = op('handler')
    module = getattr(handler, 'module', None) if handler else None
    return {
        'connectorName': getattr(module, 'CONNECTOR_NAME', 'TDCliServer'),
        'connectorVersion': getattr(module, 'CONNECTOR_VERSION', '0.1.0'),
        'protocolVersion': getattr(module, 'PROTOCOL_VERSION', 1),
        'connectorInstallMode': getattr(module, 'CONNECTOR_INSTALL_MODE', 'tox'),
    }


def _normalize_headers(headers: Any) -> Dict[str, str]:
    if not isinstance(headers, dict):
        return {}
    return {str(k).lower(): str(v) for k, v in headers.items()}


def _extract_request_token(request: Dict[str, Any]) -> str:
    headers = _normalize_headers(request.get('headers', {}))
    token = headers.get('x-td-cli-token', '').strip()
    if token:
        return token

    auth = headers.get('authorization', '').strip()
    if auth.lower().startswith('bearer '):
        return auth[7:].strip()

    return ''


def _write_unauthorized(response: Dict[str, Any]) -> Dict[str, Any]:
    response['statusCode'] = 401
    response['statusReason'] = 'Unauthorized'
    response['data'] = json.dumps({
        'success': False,
        'message': 'Unauthorized',
        'data': None,
    })
    return response


def onHTTPRequest(dat: 'webserverDAT', request: Dict[str, Any],
                  response: Dict[str, Any]) -> Dict[str, Any]:
    uri = request['uri']
    method = request['method']

    # CORS headers — restrict to localhost origins only
    origin = request.get('headers', {}).get('Origin', '') or request.get('headers', {}).get('origin', '')
    allowed_origins = ('http://localhost', 'http://127.0.0.1', 'https://localhost', 'https://127.0.0.1')
    if any(origin == o or origin.startswith(o + ':') for o in allowed_origins):
        response['Access-Control-Allow-Origin'] = origin
    else:
        response['Access-Control-Allow-Origin'] = 'http://127.0.0.1'
    response['Access-Control-Allow-Methods'] = 'GET, POST, OPTIONS'
    response['Access-Control-Allow-Headers'] = 'Content-Type, X-TD-CLI-Token, Authorization'
    response['content-type'] = 'application/json'

    if method == 'OPTIONS':
        response['statusCode'] = 204
        response['statusReason'] = 'No Content'
        response['data'] = ''
        return response

    required_token = _get_required_token()
    if required_token:
        provided_token = _extract_request_token(request)
        if not provided_token or not hmac.compare_digest(provided_token, required_token):
            return _write_unauthorized(response)

    with _request_lock:
        try:
            if method == 'GET' and uri == '/health':
                metadata = _get_connector_metadata()
                result = {
                    'success': True,
                    'message': 'td-cli server running',
                    'data': {
                        'version': '0.1.0',
                        'project': project.name,
                        'tdVersion': app.version,
                        'tdBuild': app.build,
                        'connectorName': metadata['connectorName'],
                        'connectorVersion': metadata['connectorVersion'],
                        'protocolVersion': metadata['protocolVersion'],
                        'connectorInstallMode': metadata['connectorInstallMode'],
                    }
                }
                response['statusCode'] = 200
                response['statusReason'] = 'OK'
                response['data'] = json.dumps(result)

            elif method == 'POST':
                body = json.loads(request['data']) if request.get('data') else {}
                handler = op('handler')
                result = handler.module.handle_request(uri, body)
                response['statusCode'] = 200
                response['statusReason'] = 'OK'
                response['data'] = json.dumps(result)

            else:
                response['statusCode'] = 404
                response['statusReason'] = 'Not Found'
                response['data'] = json.dumps({
                    'success': False,
                    'message': f'Unknown endpoint: {method} {uri}',
                    'data': None
                })

        except Exception as e:
            import traceback
            response['statusCode'] = 500
            response['statusReason'] = 'Internal Server Error'
            response['data'] = json.dumps({
                'success': False,
                'message': str(e),
                'data': {'traceback': traceback.format_exc()}
            })

    return response


def onWebSocketOpen(dat: 'webserverDAT', client: str, uri: str):
    return


def onWebSocketClose(dat: 'webserverDAT', client: str):
    return


def onWebSocketReceiveText(dat: 'webserverDAT', client: str, data: str):
    dat.webSocketSendText(client, data)
    return


def onWebSocketReceiveBinary(dat: 'webserverDAT', client: str, data: bytes):
    dat.webSocketSendBinary(client, data)
    return


def onWebSocketReceivePing(dat: 'webserverDAT', client: str, data: bytes):
    dat.webSocketSendPong(client, data=data)
    return


def onWebSocketReceivePong(dat: 'webserverDAT', client: str, data: bytes):
    return


def onServerStart(dat: 'webserverDAT'):
    port = dat.par.port.val
    debug(f'td-cli server started on port {port}')


def onServerStop(dat: 'webserverDAT'):
    debug('td-cli server stopped')
