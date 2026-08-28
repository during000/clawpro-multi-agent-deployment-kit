"""DevResonance CloudAgent HTTP executor used by the ClawPro workflow PoC.

The browser only submits a stable CloudAgent id.  Runtime/session resolution and
service credentials stay on the server.  Two deployment shapes are supported:

1. A DevResonance gateway that resolves ``agent_id`` to a live sandbox session.
2. An explicit per-agent direct-prompt route supplied through server env vars.

No token or sandbox JWT is ever returned to the frontend or persisted in task
state.
"""

from __future__ import annotations

import json
import os
import time
import urllib.error
import urllib.parse
import urllib.request
from dataclasses import dataclass


class CloudAgentError(RuntimeError):
    """Raised when a CloudAgent cannot be resolved or invoked."""


CLOUDAGENT_CATALOG = (
    {
        "id": "calendar-scan-agent",
        "name": "日程扫描",
        "owner": "bingbingxu",
        "capabilities": ["calendar.read", "opportunity.read"],
    },
    {
        "id": "pre-visit-reminder-agent",
        "name": "拜访前候选采集",
        "owner": "bingbingxu",
        "capabilities": ["calendar.read", "opportunity.read", "visit.read"],
    },
    {
        "id": "message-assemble-agent",
        "name": "消息组装",
        "owner": "bingbingxu",
        "capabilities": ["message.draft"],
    },
    {
        "id": "message-notify-agent",
        "name": "消息通知",
        "owner": "bingbingxu",
        "capabilities": ["message.send"],
    },
)

AGENTOS_POC_AGENT = {
    "id": "clawpro-poc-agent",
    "name": "ClawPro 云端通用 Agent",
    "owner": "clawpro",
    "capabilities": ["text.generate", "artifact.create"],
}


@dataclass(frozen=True)
class CloudAgentRoute:
    url: str
    session_id: str = ""
    cancel_url: str = ""


class DevResonanceCloudAgentClient:
    """Small stdlib-only client for DevResonance direct-prompt compatible APIs."""

    def __init__(self):
        self.gateway_url = os.environ.get("CLOUDAGENT_GATEWAY_URL", "").strip().rstrip("/")
        self.gateway_path = os.environ.get(
            "CLOUDAGENT_GATEWAY_PATH",
            "/api/cloudagents/{agent_id}/direct-prompt",
        ).strip()
        self.api_key = os.environ.get("CLOUDAGENT_API_KEY", "").strip()
        self.api_key_header = os.environ.get(
            "CLOUDAGENT_API_KEY_HEADER", "x-api-key"
        ).strip()
        self.routes = self._load_routes(
            os.environ.get("CLOUDAGENT_DIRECT_PROMPT_ROUTES_JSON", "")
        )
        self.agentos_base_url = os.environ.get(
            "AGENTOS_BASE_URL", "https://www.codebuddy.cn/v2/agentos"
        ).strip().rstrip("/")
        self.agentos_api_key = os.environ.get("AGENTOS_API_KEY", "").strip()
        self.agentos_source_app = os.environ.get(
            "AGENTOS_SOURCE_APP", "clawpro"
        ).strip()
        self.agentos_runtime_id = os.environ.get(
            "AGENTOS_RUNTIME_ID", ""
        ).strip()
        self.agentos_runtime_name = os.environ.get(
            "AGENTOS_RUNTIME_NAME", "clawpro-poc-agent"
        ).strip()
        self.agentos_system_prompt = os.environ.get(
            "AGENTOS_SYSTEM_PROMPT",
            (
                "You are a concise ClawPro cloud agent. Follow the task, use only "
                "available tools and data, never fabricate external facts, and return "
                "a clear result."
            ),
        ).strip()

    @staticmethod
    def _load_routes(raw):
        if not str(raw or "").strip():
            return {}
        try:
            payload = json.loads(raw)
        except json.JSONDecodeError as error:
            raise CloudAgentError(
                "CLOUDAGENT_DIRECT_PROMPT_ROUTES_JSON 不是合法 JSON"
            ) from error
        if not isinstance(payload, dict):
            raise CloudAgentError(
                "CLOUDAGENT_DIRECT_PROMPT_ROUTES_JSON 必须是对象"
            )
        routes = {}
        for agent_id, raw_route in payload.items():
            if isinstance(raw_route, str):
                raw_route = {"url": raw_route}
            if not isinstance(raw_route, dict):
                continue
            url = str(raw_route.get("url") or "").strip()
            if not url:
                continue
            routes[str(agent_id)] = CloudAgentRoute(
                url=url,
                session_id=str(raw_route.get("session_id") or "").strip(),
                cancel_url=str(raw_route.get("cancel_url") or "").strip(),
            )
        return routes

    @property
    def configured(self):
        return bool(self.gateway_url or self.routes or self.agentos_api_key)

    @property
    def agentos_configured(self):
        return bool(self.agentos_api_key and self.agentos_base_url)

    def route_for(self, agent_id):
        if agent_id == AGENTOS_POC_AGENT["id"] and self.agentos_configured:
            runtime_id = self.agentos_runtime_id or "auto"
            return (
                CloudAgentRoute(
                    url="agentos://runtime/{0}".format(runtime_id),
                    session_id=runtime_id if runtime_id != "auto" else "",
                ),
                "agentos-direct-prompt",
            )
        explicit = self.routes.get(agent_id)
        if explicit:
            return explicit, "direct-prompt"
        if self.gateway_url:
            encoded = urllib.parse.quote(agent_id, safe="")
            path = self.gateway_path.format(agent_id=encoded)
            if not path.startswith("/"):
                path = "/" + path
            return CloudAgentRoute(url=self.gateway_url + path), "gateway"
        raise CloudAgentError(
            "DevResonance CloudAgent 网关未配置；请在 ClawPro 后端设置 "
            "CLOUDAGENT_GATEWAY_URL，或注入每个 Agent 的 direct-prompt 路由"
        )

    def public_agents(self):
        result = []
        for item in CLOUDAGENT_CATALOG:
            explicit = self.routes.get(item["id"])
            available = bool(explicit or self.gateway_url)
            result.append(
                {
                    **item,
                    "platform": "cloudagent",
                    "location": "cloud",
                    "status": "online" if available else "offline",
                    "available": available,
                    "runtime_id": "devresonance-cloudagent",
                    "detail": (
                        "DevResonance CloudAgent · HTTPS direct-prompt"
                        if available
                        else "等待 ClawPro 后端配置 DevResonance 调用网关"
                    ),
                }
            )
        result.insert(
            0,
            {
                **AGENTOS_POC_AGENT,
                "platform": "cloudagent",
                "location": "cloud",
                "status": "online" if self.agentos_configured else "offline",
                "available": self.agentos_configured,
                "runtime_id": "devresonance-cloudagent",
                "detail": (
                    "AgentOS Runtime · HTTPS direct-prompt · ClawPro 服务端受控凭证"
                    if self.agentos_configured
                    else "等待 ClawPro 后端配置 AgentOS API Key"
                ),
            },
        )
        return result

    def _headers(self):
        headers = {"Content-Type": "application/json"}
        if self.api_key:
            headers[self.api_key_header] = self.api_key
        return headers

    @staticmethod
    def _decode_response(response):
        raw = response.read()
        if not raw:
            return {}
        try:
            return json.loads(raw.decode("utf-8"))
        except (UnicodeDecodeError, json.JSONDecodeError) as error:
            raise CloudAgentError("CloudAgent 返回了无法解析的响应") from error

    def _agentos_headers(self, *, bearer_token=""):
        headers = {"Content-Type": "application/json"}
        if bearer_token:
            headers["Authorization"] = "Bearer " + bearer_token
        else:
            headers["x-api-key"] = self.agentos_api_key
            headers["X-Source-App"] = self.agentos_source_app
        return headers

    def _agentos_request(self, path, *, method="GET", payload=None, timeout=60):
        request = urllib.request.Request(
            self.agentos_base_url + path,
            data=(
                json.dumps(payload, ensure_ascii=False).encode("utf-8")
                if payload is not None
                else None
            ),
            headers=self._agentos_headers(),
            method=method,
        )
        try:
            with urllib.request.urlopen(request, timeout=timeout) as response:
                body = self._decode_response(response)
        except urllib.error.HTTPError as error:
            detail = error.read().decode("utf-8", errors="replace")[-1000:]
            raise CloudAgentError(
                "AgentOS 调用失败（HTTP {0}）：{1}".format(error.code, detail)
            ) from error
        except urllib.error.URLError as error:
            raise CloudAgentError(
                "无法连接 AgentOS：{0}".format(error.reason)
            ) from error
        if body.get("code") not in (None, 0, "0"):
            raise CloudAgentError(
                "AgentOS 调用失败（code={0}）：{1}".format(
                    body.get("code"), body.get("msg") or "服务端拒绝请求"
                )
            )
        return body.get("data") or {}

    def _find_agentos_runtime(self):
        query = urllib.parse.urlencode(
            {"metadata.source": "clawpro-poc", "pageSize": 100}
        )
        data = self._agentos_request("/runtimes?" + query)
        for item in data.get("items") or []:
            if str(item.get("runtimeName") or "") == self.agentos_runtime_name:
                return item
        return None

    def _create_agentos_runtime(self):
        codebuddy_key = os.environ.get("CODEBUDDY_API_KEY", "").strip()
        if not codebuddy_key:
            codebuddy_key = self.agentos_api_key
        payload = {
            "runtimeName": self.agentos_runtime_name,
            "visibility": "PRIVATE",
            "agentManifest": {
                "id": AGENTOS_POC_AGENT["id"],
                "name": "ClawPro POC Agent",
                "manifestVersion": "1.0",
                "system_prompt": self.agentos_system_prompt,
                "secrets": [
                    {"key": "CODEBUDDY_API_KEY", "value": codebuddy_key}
                ],
            },
            "metadata": {
                "source": "clawpro-poc",
                "purpose": "agent-call-demo",
            },
        }
        return self._agentos_request(
            "/runtimes", method="POST", payload=payload, timeout=120
        )

    def _get_agentos_runtime(self, runtime_id):
        return self._agentos_request(
            "/runtimes/{0}".format(urllib.parse.quote(str(runtime_id), safe=""))
        )

    def _ensure_agentos_runtime(self, timeout_seconds):
        runtime = None
        if self.agentos_runtime_id:
            runtime = self._get_agentos_runtime(self.agentos_runtime_id)
        else:
            runtime = self._find_agentos_runtime()
            if not runtime:
                runtime = self._create_agentos_runtime()
            self.agentos_runtime_id = str(runtime.get("id") or "")
        if not self.agentos_runtime_id:
            raise CloudAgentError("AgentOS 未返回 Runtime ID")

        deadline = time.monotonic() + min(max(timeout_seconds, 10), 180)
        while str(runtime.get("status") or "").upper() != "RUNNING":
            status = str(runtime.get("status") or "").upper()
            if status in {"FAILED", "STOPPED", "DELETED"}:
                raise CloudAgentError(
                    "AgentOS Runtime 当前不可用：{0}".format(status)
                )
            if time.monotonic() >= deadline:
                raise CloudAgentError("等待 AgentOS Runtime 就绪超时")
            time.sleep(1)
            runtime = self._get_agentos_runtime(self.agentos_runtime_id)
        return self._get_agentos_runtime(self.agentos_runtime_id)

    def _execute_agentos(self, prompt, trace_id, timeout_seconds):
        runtime = self._ensure_agentos_runtime(timeout_seconds)
        acp_link = ((runtime.get("links") or {}).get("acpLink") or {})
        acp_url = str(acp_link.get("url") or "").strip()
        bearer_token = str(acp_link.get("token") or "").strip()
        sessions = runtime.get("sessions") or []
        # AgentOS creates an initial Session whose id equals the Runtime id. Some
        # Runtime detail responses omit the sessions collection after startup,
        # while direct-prompt still accepts that stable initial Session id.
        session_id = str(
            (sessions[0] if sessions else {}).get("sessionId")
            or runtime.get("id")
            or self.agentos_runtime_id
            or ""
        )
        if not acp_url or not bearer_token or not session_id:
            raise CloudAgentError("AgentOS Runtime 缺少 direct-prompt 调用信息")

        parsed = urllib.parse.urlsplit(acp_url)
        direct_prompt_url = urllib.parse.urlunsplit(
            (parsed.scheme, parsed.netloc, "/api/session/direct-prompt", "", "")
        )
        payload = {
            "sessionId": session_id,
            "message": [{"type": "text", "text": prompt}],
            "traceId": trace_id,
            "uploadArtifacts": "tool",
            "sync": True,
            "maxDurationMs": min(
                max(int(timeout_seconds * 1000), 1), 1_800_000
            ),
            "attachmentTtlSec": 3600,
        }
        request = urllib.request.Request(
            direct_prompt_url,
            data=json.dumps(payload, ensure_ascii=False).encode("utf-8"),
            headers=self._agentos_headers(bearer_token=bearer_token),
            method="POST",
        )
        try:
            with urllib.request.urlopen(
                request, timeout=timeout_seconds + 30
            ) as response:
                body = self._decode_response(response)
        except urllib.error.HTTPError as error:
            detail = error.read().decode("utf-8", errors="replace")[-1000:]
            raise CloudAgentError(
                "AgentOS direct-prompt 调用失败（HTTP {0}）：{1}".format(
                    error.code, detail
                )
            ) from error
        except urllib.error.URLError as error:
            raise CloudAgentError(
                "无法连接 AgentOS direct-prompt：{0}".format(error.reason)
            ) from error

        data = body.get("data") if isinstance(body, dict) else None
        if not isinstance(data, dict):
            data = {}
        if body.get("code") not in (None, 0, "0"):
            raise CloudAgentError(
                "AgentOS direct-prompt 调用失败（code={0}）：{1}".format(
                    body.get("code"), body.get("msg") or "服务端拒绝请求"
                )
            )
        status = str(data.get("status") or "").strip().lower()
        error_message = str(data.get("error") or "").strip()
        if status not in {"success", "completed", "ok"} or error_message:
            raise CloudAgentError(
                "AgentOS direct-prompt 执行失败：{0}".format(
                    error_message or status or "unknown"
                )
            )
        return {
            "summary": str(data.get("result") or "").strip(),
            "session_id": session_id,
            "trace_id": str(data.get("traceId") or trace_id),
            "attachments": data.get("attachments") or [],
            "usage": data.get("usage"),
            "route_kind": "agentos-direct-prompt",
        }

    def execute(self, agent_id, prompt, trace_id, timeout_seconds=1800):
        if agent_id == AGENTOS_POC_AGENT["id"] and self.agentos_configured:
            return self._execute_agentos(prompt, trace_id, timeout_seconds)
        route, route_kind = self.route_for(agent_id)
        payload = {
            "agentId": agent_id,
            "agent_id": agent_id,
            "message": [{"type": "text", "text": prompt}],
            "prompt": prompt,
            "traceId": trace_id,
            "trace_id": trace_id,
            "uploadArtifacts": "tool",
            "sync": True,
            "maxDurationMs": min(max(int(timeout_seconds * 1000), 1), 1_800_000),
            "attachmentTtlSec": 3600,
        }
        if route.session_id:
            payload["sessionId"] = route.session_id
        request = urllib.request.Request(
            route.url,
            data=json.dumps(payload, ensure_ascii=False).encode("utf-8"),
            headers=self._headers(),
            method="POST",
        )
        try:
            with urllib.request.urlopen(request, timeout=timeout_seconds + 30) as response:
                body = self._decode_response(response)
        except urllib.error.HTTPError as error:
            detail = error.read().decode("utf-8", errors="replace")[-1000:]
            raise CloudAgentError(
                "CloudAgent {0} 调用失败（HTTP {1}）：{2}".format(
                    agent_id, error.code, detail
                )
            ) from error
        except urllib.error.URLError as error:
            raise CloudAgentError(
                "无法连接 CloudAgent {0}：{1}".format(agent_id, error.reason)
            ) from error

        data = body.get("data") if isinstance(body, dict) else None
        if not isinstance(data, dict):
            data = body if isinstance(body, dict) else {}
        response_code = body.get("code") if isinstance(body, dict) else None
        response_message = str(
            body.get("msg") if isinstance(body, dict) else ""
        ).strip()
        if response_code not in (None, 0, "0"):
            raise CloudAgentError(
                "CloudAgent {0} 执行失败（code={1}）：{2}".format(
                    agent_id,
                    response_code,
                    response_message or "服务端拒绝请求",
                )
            )
        status = str(data.get("status") or body.get("status") or "success")
        error_message = str(data.get("error") or body.get("error") or "").strip()
        if status not in {"success", "completed", "ok"} or error_message:
            raise CloudAgentError(
                "CloudAgent {0} 执行失败：{1}".format(
                    agent_id, error_message or status
                )
            )
        result = str(
            data.get("result")
            or data.get("output")
            or body.get("result")
            or body.get("output")
            or ""
        ).strip()
        return {
            "summary": result,
            "session_id": str(
                data.get("sessionId")
                or body.get("sessionId")
                or route.session_id
                or "cloudagent:{0}:{1}".format(agent_id, trace_id)
            ),
            "trace_id": str(data.get("traceId") or body.get("traceId") or trace_id),
            "attachments": data.get("attachments") or body.get("attachments") or [],
            "usage": data.get("usage") or body.get("usage"),
            "route_kind": route_kind,
        }

    def cancel(self, agent_id, session_id, trace_id):
        if agent_id == AGENTOS_POC_AGENT["id"] and self.agentos_configured:
            return False
        route, _ = self.route_for(agent_id)
        cancel_url = route.cancel_url
        if not cancel_url:
            return False
        request = urllib.request.Request(
            cancel_url,
            data=json.dumps(
                {
                    "agentId": agent_id,
                    "sessionId": session_id,
                    "traceId": trace_id,
                }
            ).encode("utf-8"),
            headers=self._headers(),
            method="POST",
        )
        try:
            with urllib.request.urlopen(request, timeout=15):
                return True
        except (urllib.error.HTTPError, urllib.error.URLError):
            return False
