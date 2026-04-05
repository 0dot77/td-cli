"""
td-cli Web Server DAT Callbacks
Place this code in the Web Server DAT's callbacks parameter.
The Web Server DAT should be named 'webserver1' inside TDCliServer COMP.
"""

import json
from typing import Dict, Any


def onHTTPRequest(dat: 'webserverDAT', request: Dict[str, Any],
                  response: Dict[str, Any]) -> Dict[str, Any]:
    uri = request['uri']
    method = request['method']

    # CORS headers for local development
    response['Access-Control-Allow-Origin'] = '*'
    response['Access-Control-Allow-Methods'] = 'GET, POST, OPTIONS'
    response['Access-Control-Allow-Headers'] = 'Content-Type'
    response['content-type'] = 'application/json'

    if method == 'OPTIONS':
        response['statusCode'] = 204
        response['statusReason'] = 'No Content'
        response['data'] = ''
        return response

    try:
        if method == 'GET' and uri == '/health':
            result = {
                'success': True,
                'message': 'td-cli server running',
                'data': {
                    'version': '0.1.0',
                    'project': project.name,
                    'tdVersion': app.version,
                    'tdBuild': app.build,
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
