import importlib.util
import json
import os
import pathlib
import tempfile
import unittest


MODULE_PATH = pathlib.Path(__file__).resolve().parents[1] / "td" / "heartbeat.py"


def load_heartbeat_module():
    spec = importlib.util.spec_from_file_location("heartbeat_under_test", MODULE_PATH)
    module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(module)
    return module


class FakeHandlerModule:
    CONNECTOR_NAME = "TDCliServer"
    CONNECTOR_VERSION = "0.1.0"
    PROTOCOL_VERSION = 1
    CONNECTOR_INSTALL_MODE = "tox"


class FakeHandlerDAT:
    module = FakeHandlerModule()


class FakePortParam:
    val = 9500

    def __int__(self):
        return self.val


class FakeWebserverDAT:
    par = type("ParCollection", (), {"port": FakePortParam()})()


class HeartbeatTests(unittest.TestCase):
    def setUp(self):
        self.module = load_heartbeat_module()
        self.temp_home = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp_home.cleanup)
        self.original_home = os.environ.get("HOME")
        os.environ["HOME"] = self.temp_home.name
        self.addCleanup(self._restore_home)
        self.module.project = type("Project", (), {"name": "TestProject", "folder": "/shows/test.toe", "realTime": False})()
        self.module.app = type("App", (), {"version": "2023.1", "build": "12340"})()
        self.module.absTime = type("AbsTime", (), {"stepSeconds": 0.5})()
        self.module.debug = lambda *_args, **_kwargs: None
        self.module.op = lambda path=None: {
            "/": None,
            "handler": FakeHandlerDAT(),
            "webserver1": FakeWebserverDAT(),
        }.get(path)

    def _restore_home(self):
        if self.original_home is None:
            os.environ.pop("HOME", None)
        else:
            os.environ["HOME"] = self.original_home

    def test_write_heartbeat_includes_connector_metadata(self):
        self.module._write_heartbeat()

        instances_dir = pathlib.Path(self.temp_home.name) / ".td-cli" / "instances"
        heartbeat_files = list(instances_dir.glob("*.json"))
        self.assertEqual(len(heartbeat_files), 1)

        payload = json.loads(heartbeat_files[0].read_text())
        self.assertEqual(payload["connectorName"], "TDCliServer")
        self.assertEqual(payload["connectorVersion"], "0.1.0")
        self.assertEqual(payload["protocolVersion"], 1)
        self.assertEqual(payload["connectorInstallMode"], "tox")
