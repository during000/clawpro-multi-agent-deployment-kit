import json
import os
import threading
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from unittest.mock import patch

from cloudagent_client import CloudAgentError, DevResonanceCloudAgentClient


class _GatewayHandler(BaseHTTPRequestHandler):
    request_payload = None
    response_payload = None
    request_path = None

    def log_message(self, _format, *_args):
        return

    def _write_json(self, payload, status=200):
        body = json.dumps(payload).encode("utf-8")
        self.send_response(status)
        self.send_header("content-type", "application/json")
        self.send_header("content-length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def _agentos_runtime(self):
        return {
            "id": "agentos-runtime-test",
            "runtimeName": "clawpro-poc-agent",
            "status": "RUNNING",
            "links": {
                "acpLink": {
                    "url": "http://127.0.0.1:{0}/acp".format(
                        self.server.server_port
                    ),
                    "token": "short-lived-test-token",
                }
            },
            "sessions": [],
        }

    def do_GET(self):
        if self.path.startswith("/runtimes?"):
            self._write_json(
                {"code": 0, "msg": "ok", "data": {"items": []}}
            )
            return
        if self.path == "/runtimes/agentos-runtime-test":
            self._write_json(
                {"code": 0, "msg": "ok", "data": self._agentos_runtime()}
            )
            return
        self._write_json({"code": 1, "msg": "not found"}, 404)

    def do_POST(self):
        length = int(self.headers.get("content-length") or 0)
        self.__class__.request_payload = json.loads(self.rfile.read(length))
        self.__class__.request_path = self.path
        if self.path == "/runtimes":
            self._write_json(
                {"code": 0, "msg": "ok", "data": self._agentos_runtime()},
                201,
            )
            return
        if self.path == "/api/session/direct-prompt":
            self._write_json(
                {
                    "code": 0,
                    "msg": "ok",
                    "data": {
                        "status": "success",
                        "result": "ClawPro 已成功调用 CloudAgent。",
                        "traceId": "agentos-trace-test",
                        "attachments": [],
                        "usage": {"inputTokens": 8, "outputTokens": 4},
                    },
                }
            )
            return
        payload = self.__class__.response_payload or {
                "code": 0,
                "msg": "ok",
                "data": {
                    "status": "success",
                    "result": "检查结果44813，已正确关联商机。",
                    "sessionId": "agent-session-test",
                    "traceId": "trace-test",
                    "attachments": [],
                    "usage": {"inputTokens": 10, "outputTokens": 5},
                },
            }
        body = json.dumps(payload).encode("utf-8")
        self.send_response(200)
        self.send_header("content-type", "application/json")
        self.send_header("content-length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)


class CloudAgentClientTests(unittest.TestCase):
    def setUp(self):
        _GatewayHandler.response_payload = None
        _GatewayHandler.request_path = None
        self.server = ThreadingHTTPServer(("127.0.0.1", 0), _GatewayHandler)
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)
        self.thread.start()

    def tearDown(self):
        self.server.shutdown()
        self.server.server_close()
        self.thread.join(timeout=2)

    def test_gateway_resolves_agent_and_returns_direct_prompt_result(self):
        base_url = "http://127.0.0.1:{0}".format(self.server.server_port)
        with patch.dict(
            os.environ,
            {
                "CLOUDAGENT_GATEWAY_URL": base_url,
                "CLOUDAGENT_API_KEY": "server-only-test-key",
                "AGENTOS_API_KEY": "",
            },
            clear=False,
        ):
            client = DevResonanceCloudAgentClient()
            result = client.execute(
                "calendar-scan-agent", "生成 44813 号日程的会前简报", "trace-test", 5
            )

        self.assertEqual(result["session_id"], "agent-session-test")
        self.assertIn("44813", result["summary"])
        self.assertEqual(
            _GatewayHandler.request_payload["agent_id"], "calendar-scan-agent"
        )
        self.assertTrue(_GatewayHandler.request_payload["sync"])

    def test_catalog_is_offline_without_server_side_route(self):
        with patch.dict(
            os.environ,
            {
                "CLOUDAGENT_GATEWAY_URL": "",
                "CLOUDAGENT_DIRECT_PROMPT_ROUTES_JSON": "",
                "AGENTOS_API_KEY": "",
            },
            clear=False,
        ):
            client = DevResonanceCloudAgentClient()

        self.assertFalse(client.configured)
        self.assertTrue(all(not item["available"] for item in client.public_agents()))

    def test_direct_prompt_code_error_is_not_treated_as_success(self):
        _GatewayHandler.response_payload = {
            "code": 1,
            "msg": "agent is not shared",
        }
        base_url = "http://127.0.0.1:{0}".format(self.server.server_port)
        with patch.dict(
            os.environ,
            {"CLOUDAGENT_GATEWAY_URL": base_url, "AGENTOS_API_KEY": ""},
            clear=False,
        ):
            client = DevResonanceCloudAgentClient()
            with self.assertRaisesRegex(CloudAgentError, "agent is not shared"):
                client.execute("calendar-scan-agent", "probe", "trace-error", 5)

    def test_agentos_creates_runtime_and_executes_direct_prompt(self):
        base_url = "http://127.0.0.1:{0}".format(self.server.server_port)
        with patch.dict(
            os.environ,
            {
                "CLOUDAGENT_GATEWAY_URL": "",
                "CLOUDAGENT_DIRECT_PROMPT_ROUTES_JSON": "",
                "AGENTOS_BASE_URL": base_url,
                "AGENTOS_API_KEY": "ck_test-agentos-key",
                "AGENTOS_RUNTIME_ID": "",
                "AGENTOS_SOURCE_APP": "clawpro",
                "CODEBUDDY_API_KEY": "ck_test-codebuddy-key",
            },
            clear=False,
        ):
            client = DevResonanceCloudAgentClient()
            result = client.execute(
                "clawpro-poc-agent",
                "只回复固定回执",
                "agentos-trace-test",
                5,
            )

        self.assertEqual(result["session_id"], "agentos-runtime-test")
        self.assertEqual(result["summary"], "ClawPro 已成功调用 CloudAgent。")
        self.assertEqual(_GatewayHandler.request_path, "/api/session/direct-prompt")
        self.assertTrue(
            next(
                item
                for item in client.public_agents()
                if item["id"] == "clawpro-poc-agent"
            )["available"]
        )


if __name__ == "__main__":
    unittest.main()
