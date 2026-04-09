import importlib.util
import pathlib
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


class FakeDAT:
    family = "DAT"

    def __init__(self, text="original"):
        self.text = text


class FakeOp:
    def __init__(self, path, name):
        self.path = path
        self.name = name
        self.outputConnectors = [FakeConnector(self)]
        self.inputConnectors = [FakeConnector(self)]


class TDCliHandlerTests(unittest.TestCase):
    def setUp(self):
        self.module = load_handler_module()

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


if __name__ == "__main__":
    unittest.main()
