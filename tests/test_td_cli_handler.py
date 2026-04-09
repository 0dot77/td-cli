import importlib.util
import json
import os
import pathlib
import tempfile
import time
import unittest


MODULE_PATH = pathlib.Path(__file__).resolve().parents[1] / "td" / "td_cli_handler.py"


def load_handler_module():
    spec = importlib.util.spec_from_file_location("td_cli_handler_under_test", MODULE_PATH)
    module = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(module)
    return module


class FakeConnection:
    def __init__(self, owner):
        self.owner = owner


class FakeConnector:
    def __init__(self, owner):
        self.owner = owner
        self.connections = []

    def connect(self, other):
        self.connections.append(FakeConnection(other.owner))


class FakePage:
    def __init__(self, name):
        self.name = name


class FakeParam:
    def __init__(self, name, value, default=None, mode="CONSTANT", expr=""):
        self.name = name
        self.label = name
        self._val = value
        self.default = value if default is None else default
        self.mode = mode
        self.expr = expr
        self.page = FakePage("Test")

    @property
    def val(self):
        return self._val

    @val.setter
    def val(self, value):
        self._val = value


class FakeParCollection:
    def __init__(self, params):
        for param in params:
            setattr(self, param.name, param)


class FakeDAT:
    family = "DAT"
    isTable = False

    def __init__(self, path="/project1/text1", text="original"):
        self.path = path
        self.name = path.rsplit("/", 1)[-1]
        self.text = text
        self.numRows = 1
        self.numCols = 1

    def clear(self):
        self.text = ""

    def appendRow(self, row):
        self.text = "\n".join(filter(None, [self.text, "\t".join(str(cell) for cell in row)]))


class FakeOp:
    def __init__(self, path, name, op_type="nullTOP", family="TOP", params=None, is_comp=False):
        self.path = path
        self.name = name
        self.type = op_type
        self.family = family
        self.nodeX = 0
        self.nodeY = 0
        self.comment = ""
        self.isCOMP = is_comp
        self.outputConnectors = [FakeConnector(self)]
        self.inputConnectors = [FakeConnector(self)]
        self._params = params or []
        self.par = FakeParCollection(self._params)
        self.children = []
        self._parent = None

    def pars(self):
        return self._params

    def findChildren(self, depth=1, family=None):
        if depth <= 0:
            return []

        result = []
        for child in self.children:
            if family is None or child.family == family:
                result.append(child)
            if depth > 1:
                result.extend(child.findChildren(depth=depth - 1, family=family))
        return result

    def create(self, op_type, name):
        child_path = f"{self.path.rstrip('/')}/{name}" if self.path != "/" else f"/{name}"
        child = FakeOp(child_path, name, op_type=op_type, params=self._created_params(name), is_comp=op_type.endswith("COMP"))
        child._parent = self
        self.children.append(child)
        return child

    def _created_params(self, name):
        return []

    def parent(self):
        return self._parent

    def destroy(self):
        if self._parent is not None:
            self._parent.children = [child for child in self._parent.children if child is not self]


class FakeImportParent(FakeOp):
    def __init__(self, path, name):
        super().__init__(path, name, op_type="containerCOMP", family="COMP", is_comp=True)

    def _created_params(self, name):
        return [
            FakeParam("speed", 0, default=0),
            FakeParam("enabled", False, default=False),
            FakeParam("size", (0, 0), default=(0, 0)),
            FakeParam("script", "", default="", expr=""),
        ]


class TDCliHandlerTests(unittest.TestCase):
    def setUp(self):
        self.module = load_handler_module()
        self.temp_home = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp_home.cleanup)
        self.original_home = os.environ.get("HOME")
        os.environ["HOME"] = self.temp_home.name
        self.addCleanup(self._restore_home)
        self.module.project = type("Project", (), {"name": "TestProject", "folder": "/shows/test.toe"})()
        self.module.absTime = type("AbsTime", (), {"seconds": 12.5})()
        self.module.app = type("App", (), {"version": "2023.1", "build": "12340"})()

    def _restore_home(self):
        if self.original_home is None:
            os.environ.pop("HOME", None)
        else:
            os.environ["HOME"] = self.original_home

    def test_handle_exec_return_value(self):
        result = self.module.handle_exec({"code": "return 42"})

        self.assertTrue(result["success"])
        self.assertEqual(result["data"]["result"], "42")

    def test_handle_exec_stdout(self):
        result = self.module.handle_exec({"code": "print('hello')"})

        self.assertTrue(result["success"])
        self.assertIsNone(result["data"]["result"])
        self.assertEqual(result["data"]["stdout"], "hello\n")

    def test_handle_connect_rejects_invalid_source_index(self):
        src = FakeOp("/src", "src")
        dst = FakeOp("/dst", "dst")
        ops = {src.path: src, dst.path: dst}
        self.module.op = ops.get

        result = self.module.handle_connect(
            {"src": src.path, "dst": dst.path, "srcIndex": 2, "dstIndex": 0}
        )

        self.assertFalse(result["success"])
        self.assertIn("Source connector index out of range", result["message"])

    def test_handle_connect_rejects_invalid_destination_index(self):
        src = FakeOp("/src", "src")
        dst = FakeOp("/dst", "dst")
        ops = {src.path: src, dst.path: dst}
        self.module.op = ops.get

        result = self.module.handle_connect(
            {"src": src.path, "dst": dst.path, "srcIndex": 0, "dstIndex": -1}
        )

        self.assertFalse(result["success"])
        self.assertIn("Destination connector index out of range", result["message"])

    def test_handle_dat_write_accepts_empty_string(self):
        dat = FakeDAT(text="before")
        self.module.op = {"/project1/text1": dat}.get

        result = self.module.handle_dat_write({"path": "/project1/text1", "content": ""})

        self.assertTrue(result["success"])
        self.assertEqual(dat.text, "")
        self.assertIn("backupId", result["data"])
        backup_payload = self._read_backup_payload(result["data"]["backupPath"])
        self.assertEqual(backup_payload["before"]["content"], "before")

    def test_network_export_preserves_parameter_types(self):
        root = FakeOp("/", "root", op_type="containerCOMP", family="COMP", is_comp=True)
        child = FakeOp(
            "/typed",
            "typed",
            params=[
                FakeParam("speed", 3, default=0),
                FakeParam("gain", 0.5, default=0.0),
                FakeParam("enabled", True, default=False),
                FakeParam("label", "hello", default=""),
                FakeParam("size", (1, 2, 3), default=(0, 0, 0)),
                FakeParam("script", "", default="", expr="me.time.seconds"),
            ],
        )
        root.children.append(child)

        self.module.op = {"/": root}.get
        result = self.module.handle_network_export({"path": "/", "depth": 1})

        self.assertTrue(result["success"])
        snapshot = result["data"]
        self.assertEqual(snapshot["version"], 2)
        node = snapshot["nodes"][0]
        self.assertEqual(node["parameters"]["speed"]["value"], 3)
        self.assertEqual(node["parameters"]["speed"]["valueType"], "int")
        self.assertEqual(node["parameters"]["gain"]["valueType"], "float")
        self.assertEqual(node["parameters"]["enabled"]["valueType"], "bool")
        self.assertEqual(node["parameters"]["label"]["valueType"], "str")
        self.assertEqual(node["parameters"]["size"]["value"], [1, 2, 3])
        self.assertEqual(node["parameters"]["size"]["valueType"], "tuple")
        self.assertEqual(node["parameters"]["script"]["expression"], "me.time.seconds")

    def test_network_import_restores_typed_values_and_reports_missing_params(self):
        parent = FakeImportParent("/target", "target")
        self.module.op = {"/target": parent}.get

        snapshot = {
            "version": 2,
            "rootPath": "/project1",
            "nodes": [
                {
                    "path": "/project1/typed",
                    "name": "typed",
                    "type": "nullTOP",
                    "family": "TOP",
                    "nodeX": 10,
                    "nodeY": 20,
                    "comment": "copied",
                    "inputs": [],
                    "parameters": {
                        "speed": {"value": 3, "valueType": "int"},
                        "enabled": {"value": True, "valueType": "bool"},
                        "size": {"value": [1, 2], "valueType": "tuple"},
                        "script": {"value": "", "valueType": "str", "expression": "me.time.seconds * 2"},
                        "missing": {"value": "x", "valueType": "str"},
                    },
                }
            ],
        }

        result = self.module.handle_network_import({"snapshot": snapshot, "targetPath": "/target"})

        self.assertTrue(result["success"])
        self.assertEqual(result["data"]["warningCount"], 1)
        self.assertEqual(len(result["data"]["parameterFailures"]), 1)
        self.assertEqual(result["data"]["parameterFailures"][0]["parameter"], "missing")
        self.assertIn("backupId", result["data"])
        backup_payload = self._read_backup_payload(result["data"]["backupPath"])
        self.assertEqual(backup_payload["targetPath"], "/target")

        created = parent.children[0]
        self.assertEqual(created.nodeX, 10)
        self.assertEqual(created.nodeY, 20)
        self.assertEqual(created.comment, "copied")
        self.assertEqual(created.par.speed.val, 3)
        self.assertIs(created.par.enabled.val, True)
        self.assertEqual(created.par.size.val, (1, 2))
        self.assertEqual(created.par.script.expr, "me.time.seconds * 2")

    def test_handle_ops_delete_creates_backup_snapshot(self):
        parent = FakeOp("/project1", "project1", op_type="containerCOMP", family="COMP", is_comp=True)
        child = FakeOp("/project1/noise1", "noise1")
        child._parent = parent
        parent.children.append(child)
        self.module.op = {parent.path: parent, child.path: child}.get

        result = self.module.handle_ops_delete({"path": child.path})

        self.assertTrue(result["success"])
        self.assertEqual(parent.children, [])
        backup_payload = self._read_backup_payload(result["data"]["backupPath"])
        self.assertEqual(backup_payload["targetPath"], child.path)
        self.assertEqual(backup_payload["snapshot"]["rootPath"], child.path)

    def test_backup_list_returns_newest_first(self):
        dat = FakeDAT(text="before")
        self.module.op = {"/project1/text1": dat}.get

        first = self.module.handle_dat_write({"path": "/project1/text1", "content": "one"})
        time.sleep(0.001)
        second = self.module.handle_dat_write({"path": "/project1/text1", "content": "two"})
        result = self.module.handle_backup_list({"limit": 10})

        self.assertTrue(result["success"])
        backups = result["data"]["backups"]
        self.assertGreaterEqual(len(backups), 2)
        self.assertEqual(backups[0]["id"], second["data"]["backupId"])
        self.assertEqual(backups[1]["id"], first["data"]["backupId"])

    def test_backup_restore_recovers_previous_dat_content(self):
        dat = FakeDAT(text="before")
        self.module.op = {"/project1/text1": dat}.get

        write_result = self.module.handle_dat_write({"path": "/project1/text1", "content": "after"})
        self.assertEqual(dat.text, "after")

        restore_result = self.module.handle_backup_restore({"id": write_result["data"]["backupId"]})

        self.assertTrue(restore_result["success"])
        self.assertEqual(dat.text, "before")
        self.assertEqual(restore_result["data"]["restoredKind"], "dat-write")

    def test_backup_restore_recovers_deleted_operator(self):
        parent = FakeOp("/project1", "project1", op_type="containerCOMP", family="COMP", is_comp=True)
        child = FakeOp("/project1/noise1", "noise1")
        child._parent = parent
        parent.children.append(child)
        self.module.op = {parent.path: parent, child.path: child}.get

        delete_result = self.module.handle_ops_delete({"path": child.path})
        restore_result = self.module.handle_backup_restore({"id": delete_result["data"]["backupId"]})

        self.assertTrue(restore_result["success"])
        self.assertEqual(len(parent.children), 1)
        self.assertEqual(parent.children[0].name, "noise1")

    def test_handle_request_writes_audit_log_with_request_and_backup_ids(self):
        dat = FakeDAT(text="before")
        self.module.op = {"/project1/text1": dat}.get

        result = self.module.handle_request("/dat/write", {"path": "/project1/text1", "content": "after"})

        self.assertTrue(result["success"])
        self.assertIn("requestId", result["data"])
        events = self._read_events()
        self.assertEqual(len(events), 1)
        event = events[0]
        self.assertEqual(event["route"], "/dat/write")
        self.assertEqual(event["targetPath"], "/project1/text1")
        self.assertEqual(event["requestId"], result["data"]["requestId"])
        self.assertEqual(event["backupId"], result["data"]["backupId"])

    def test_logs_list_and_tail_ordering(self):
        dat = FakeDAT(text="before")
        self.module.op = {"/project1/text1": dat}.get

        first = self.module.handle_request("/dat/write", {"path": "/project1/text1", "content": "one"})
        time.sleep(0.001)
        second = self.module.handle_request("/dat/write", {"path": "/project1/text1", "content": "two"})

        listed = self.module.handle_logs_list({"limit": 10})
        tailed = self.module.handle_logs_tail({"limit": 10})

        self.assertTrue(listed["success"])
        self.assertTrue(tailed["success"])
        self.assertEqual(listed["data"]["events"][0]["requestId"], second["data"]["requestId"])
        self.assertEqual(listed["data"]["events"][1]["requestId"], first["data"]["requestId"])
        self.assertEqual(tailed["data"]["events"][0]["requestId"], first["data"]["requestId"])
        self.assertEqual(tailed["data"]["events"][1]["requestId"], second["data"]["requestId"])

    def test_handle_request_routes_logs_endpoints(self):
        dat = FakeDAT(text="before")
        self.module.op = {"/project1/text1": dat}.get
        self.module.handle_request("/dat/write", {"path": "/project1/text1", "content": "one"})

        listed = self.module.handle_request("/logs/list", {"limit": 5})
        tailed = self.module.handle_request("/logs/tail", {"limit": 5})

        self.assertTrue(listed["success"])
        self.assertTrue(tailed["success"])
        self.assertEqual(listed["data"]["events"][0]["route"], "/dat/write")
        self.assertEqual(tailed["data"]["events"][0]["route"], "/dat/write")

    def _read_backup_payload(self, backup_path):
        with open(backup_path, "r", encoding="utf-8") as handle:
            backup_record = json.load(handle)
        return backup_record["payload"]

    def _read_events(self):
        with open(self.module._events_log_path(), "r", encoding="utf-8") as handle:
            return [json.loads(line) for line in handle if line.strip()]


if __name__ == "__main__":
    unittest.main()
