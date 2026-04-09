import importlib.util
import os
import pathlib
import unittest
from unittest import mock


MODULE_PATH = pathlib.Path(__file__).resolve().parents[1] / "td" / "webserver_callbacks.py"


def load_callbacks_module():
    spec = importlib.util.spec_from_file_location("webserver_callbacks_under_test", MODULE_PATH)
    module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(module)
    return module


class FakeHandlerModule:
    def __init__(self):
        self.calls = []

    def handle_request(self, uri, body):
        self.calls.append((uri, body))
        return {"success": True, "message": "ok", "data": {"uri": uri}}


class FakeHandlerDAT:
    def __init__(self, module):
        self.module = module


class WebserverCallbackTests(unittest.TestCase):
    def setUp(self):
        self.module = load_callbacks_module()
        self.handler_module = FakeHandlerModule()
        self.module.op = lambda path: FakeHandlerDAT(self.handler_module) if path == "handler" else None

    def test_rejects_missing_token_when_configured(self):
        request = {"uri": "/tools/list", "method": "POST", "data": "{}"}
        response = {}

        with mock.patch.dict(os.environ, {"TD_CLI_TOKEN": "secret"}):
            result = self.module.onHTTPRequest(None, request, response)

        self.assertEqual(result["statusCode"], 401)
        self.assertEqual(self.handler_module.calls, [])

    def test_accepts_matching_token_header(self):
        request = {
            "uri": "/tools/list",
            "method": "POST",
            "data": "{}",
            "headers": {"X-TD-CLI-Token": "secret"},
        }
        response = {}

        with mock.patch.dict(os.environ, {"TD_CLI_TOKEN": "secret"}):
            result = self.module.onHTTPRequest(None, request, response)

        self.assertEqual(result["statusCode"], 200)
        self.assertEqual(self.handler_module.calls, [("/tools/list", {})])

    def test_accepts_bearer_token_header(self):
        request = {
            "uri": "/tools/list",
            "method": "POST",
            "data": "{}",
            "headers": {"Authorization": "Bearer secret"},
        }
        response = {}

        with mock.patch.dict(os.environ, {"TD_CLI_TOKEN": "secret"}):
            result = self.module.onHTTPRequest(None, request, response)

        self.assertEqual(result["statusCode"], 200)
        self.assertEqual(self.handler_module.calls, [("/tools/list", {})])
