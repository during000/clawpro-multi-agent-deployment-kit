"""
OpenClaw Gateway 集成测试工具模块

提供 Gateway WebSocket 客户端和验证函数，用于集成测试中的模型/通道可用性校验。

导出:
    ChannelConfig              - 通道测试用户 ID 配置
    default_channel_config     - 默认通道配置单例
    OpenClawGateway            - WebSocket 客户端（gateway_url 和 token 必传，从实例获取）
    connect_from_inst          - 从 InstanceContext 创建并连接 Gateway 客户端
    verify_model_config_via_inst - 通过 InstanceContext 验证模型配置
    verify_channel_delivery      - 验证通道投递
    verify_model_available       - 通过 InstanceContext 确认模型就绪
    verify_channel_configured    - 通过 InstanceContext 确认通道已配置

使用方式:
    from helpers import verify_model_available, verify_channel_configured
    verify_model_available(inst)
    verify_channel_configured(inst, "feishu")
"""

import json
import time
import threading
import uuid

import websocket

from helpers import config


# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
#  配置
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

PROTOCOL_VERSION = 4
MIN_PROTOCOL_VERSION = 3
REQUEST_TIMEOUT = 60
CHAT_TIMEOUT = 60


class ChannelConfig:
    """通道测试用户 ID 配置。优先级: 构造函数参数 > config.py 配置值（环境变量，惰性加载）"""

    def __init__(self, feishu_open_id=None, wecom_user_id=None,
                 qqbot_c2c_open_id=None, dingtalk_user_id=None):
        self._feishu_open_id = feishu_open_id
        self._wecom_user_id = wecom_user_id
        self._qqbot_c2c_open_id = qqbot_c2c_open_id
        self._dingtalk_user_id = dingtalk_user_id

    @property
    def feishu_open_id(self):
        return self._feishu_open_id or config.FEISHU_OPEN_ID

    @property
    def wecom_user_id(self):
        return self._wecom_user_id or config.WECOM_USER_ID

    @property
    def qqbot_c2c_open_id(self):
        return self._qqbot_c2c_open_id or config.QQBOT_C2C_OPEN_ID

    @property
    def dingtalk_user_id(self):
        return self._dingtalk_user_id or config.DINGTALK_USER_ID

    def __repr__(self):
        return (f"ChannelConfig(feishu={self.feishu_open_id}, wecom={self.wecom_user_id}, "
                f"qqbot={self.qqbot_c2c_open_id}, dingtalk={self.dingtalk_user_id})")


# 默认通道配置（各属性惰性读取环境变量）
default_channel_config = ChannelConfig()


# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
#  数据结构
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

class ChannelStatus:
    """Channel 状态信息"""

    def __init__(self, channel_id, data, accounts=None):
        self.id = channel_id
        self.configured = data.get("configured", False)
        self.running = data.get("running", False)
        self.last_start_at = data.get("lastStartAt")
        self.last_stop_at = data.get("lastStopAt")
        self.last_error = data.get("lastError")
        self.accounts = accounts or []

    @property
    def available(self):
        return self.configured and self.running

    def __repr__(self):
        status = "✓ 可用" if self.available else "✗ 不可用"
        return f"Channel({self.id}: {status}, configured={self.configured}, running={self.running})"


class SessionInfo:
    """Session 信息"""

    def __init__(self, data):
        self.key = data.get("key", "")
        self.status = data.get("status", "")
        self.title = data.get("title", "")
        self.origin = data.get("origin", {})

    @property
    def provider(self):
        return self.origin.get("provider", "")

    @property
    def surface(self):
        return self.origin.get("surface", "")

    @property
    def chat_type(self):
        return self.origin.get("chatType", "")

    @property
    def channel(self):
        parts = self.key.split(":")
        if len(parts) >= 3:
            ch = parts[2]
            if ch not in ("main", "dashboard", "cron"):
                return ch
        return self.provider or ""

    def __repr__(self):
        return f"Session(key={self.key}, channel={self.channel}, status={self.status})"


class CronJobInfo:
    """Cron 定时任务信息"""

    def __init__(self, data):
        self.id = data.get("id", "")
        self.name = data.get("name", "")
        self.enabled = data.get("enabled", False)
        self.created_at_ms = data.get("createdAtMs", 0)
        self.schedule = data.get("schedule", {})
        self.session_target = data.get("sessionTarget", "")
        self.wake_mode = data.get("wakeMode", "")
        self.payload = data.get("payload", {})
        self.delivery = data.get("delivery", {})
        self.state = data.get("state", {})
        self._raw = data

    @property
    def schedule_expr(self):
        return self.schedule.get("expr", "")

    @property
    def delivery_channel(self):
        return self.delivery.get("channel", "")

    @property
    def delivery_to(self):
        return self.delivery.get("to", "")

    @property
    def delivery_mode(self):
        return self.delivery.get("mode", "")

    @property
    def last_run_status(self):
        return self.state.get("lastRunStatus", "")

    @property
    def last_delivery_status(self):
        return self.state.get("lastDeliveryStatus", "")

    @property
    def last_delivered(self):
        return self.state.get("lastDelivered", False)

    @property
    def next_run_at_ms(self):
        return self.state.get("nextRunAtMs", 0)

    def __repr__(self):
        status = "✓ 启用" if self.enabled else "✗ 禁用"
        delivery_info = f"{self.delivery_channel}→{self.delivery_to}" if self.delivery_to else self.delivery_channel or "无"
        return (f"CronJob({self.id[:8]}.. {status}, name={self.name!r}, "
                f"schedule={self.schedule_expr!r}, delivery={delivery_info})")


class CronRunInfo:
    """Cron 运行记录"""

    def __init__(self, data):
        self.run_id = data.get("runId", "")
        self.status = data.get("status", "")
        self.delivered = data.get("delivered", False)
        self.delivery_status = data.get("deliveryStatus", "")
        self.summary = data.get("summary", "")
        self.error = data.get("error")
        self.started_at_ms = data.get("startedAtMs", 0)
        self.finished_at_ms = data.get("finishedAtMs", 0)
        self._raw = data

    @property
    def duration_ms(self):
        if self.finished_at_ms and self.started_at_ms:
            return self.finished_at_ms - self.started_at_ms
        return 0

    def __repr__(self):
        delivered = "✓ 已投递" if self.delivered else "✗ 未投递"
        return f"CronRun(status={self.status}, {delivered}, summary={self.summary[:60]!r})"


class ChatResult:
    """聊天结果"""

    def __init__(self):
        self.success = False
        self.assistant_text = ""
        self.events = []
        self.elapsed = 0.0
        self.error = None

    @property
    def has_reply(self):
        return bool(self.assistant_text)

    def __repr__(self):
        if self.success:
            preview = self.assistant_text[:80] + ("..." if len(self.assistant_text) > 80 else "")
            return f"ChatResult(ok, {self.elapsed:.1f}s, reply={preview!r})"
        return f"ChatResult(failed: {self.error})"


# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
#  Gateway WebSocket 客户端
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

class OpenClawGateway:
    """OpenClaw Gateway WebSocket 客户端"""

    def __init__(self, gateway_url, token):
        self.gateway_url = gateway_url
        self.token = token
        self.ws_url = self.gateway_url.replace("http://", "ws://").replace("https://", "wss://")
        self.ws = None
        self.connected = False
        self._pending = {}
        self._lock = threading.Lock()
        self._recv_thread = None
        self._hello_payload = None
        self._session_events = {}
        self._session_done = {}
        self._global_events = []
        # cron 执行完成等待器: {job_id: (Event, result_dict)}
        self._cron_waiters = {}

    # ──── 连接管理 ────

    def connect(self, timeout=15, verbose=True):
        """建立 WebSocket 连接并完成握手"""
        ws_url_with_token = f"{self.ws_url}?token={self.token}"
        if verbose:
            print(f"[连接] 正在连接 {self.ws_url} ...")

        self.ws = websocket.create_connection(ws_url_with_token, timeout=timeout)

        # 接收 challenge
        challenge = json.loads(self.ws.recv())
        if challenge.get("event") != "connect.challenge":
            raise RuntimeError(f"期望 connect.challenge，收到: {challenge.get('event')}")

        # 发送 connect
        req_id = str(uuid.uuid4())
        self.ws.send(json.dumps({
            "type": "req",
            "id": req_id,
            "method": "connect",
            "params": {
                "minProtocol": MIN_PROTOCOL_VERSION,
                "maxProtocol": PROTOCOL_VERSION,
                "client": {"id": "openclaw-control-ui", "version": "1.0.0", "platform": "linux", "mode": "ui"},
                "role": "operator",
                "scopes": ["operator.read", "operator.write", "operator.admin"],
                "caps": [],
                "commands": [],
                "permissions": {},
                "auth": {"token": self.token},
                "locale": "zh-CN",
                "userAgent": "openclaw-integration-test/1.0.0",
            },
        }))

        # 接收 hello
        hello = json.loads(self.ws.recv())
        if not hello.get("ok"):
            raise RuntimeError(f"连接失败: {json.dumps(hello.get('error', {}), ensure_ascii=False)}")

        self._hello_payload = hello.get("payload", {})
        self.connected = True

        if verbose:
            server = self._hello_payload.get("server", {})
            features = self._hello_payload.get("features", {})
            print(f"[连接] 成功！版本={server.get('version', 'N/A')} "
                  f"methods={len(features.get('methods', []))} "
                  f"events={len(features.get('events', []))}")

        # 启动后台接收线程
        self._recv_thread = threading.Thread(target=self._recv_loop, daemon=True)
        self._recv_thread.start()

        # 订阅全局事件
        self._fire("sessions.subscribe", {})

        return self._hello_payload

    def close(self):
        """关闭连接"""
        self.connected = False
        if self.ws:
            try:
                self.ws.close()
            except Exception:
                pass

    def __enter__(self):
        self.connect()
        return self

    def __exit__(self, *args):
        self.close()

    # ──── 底层收发 ────

    def _recv_loop(self):
        while self.connected:
            try:
                raw = self.ws.recv()
                if not raw:
                    continue
                self._dispatch(json.loads(raw))
            except websocket.WebSocketConnectionClosedException:
                self.connected = False
                break
            except Exception:
                if not self.connected:
                    break

    def _dispatch(self, msg):
        msg_type = msg.get("type")

        if msg_type == "res":
            msg_id = msg.get("id")
            with self._lock:
                if msg_id in self._pending:
                    self._pending[msg_id]["result"] = msg
                    self._pending[msg_id]["event"].set()

        elif msg_type == "event":
            event_name = msg.get("event", "")
            payload = msg.get("payload", {})

            if event_name == "session.message":
                session_key = payload.get("sessionKey", "")
                with self._lock:
                    if session_key in self._session_events:
                        self._session_events[session_key].append(msg)

            elif event_name == "sessions.changed":
                phase = payload.get("phase", "")
                session_key = payload.get("key", "")
                if phase in ("done", "end"):
                    with self._lock:
                        if session_key in self._session_done:
                            self._session_done[session_key].set()
                        for evt in self._session_done.values():
                            if not evt.is_set():
                                evt.set()

            elif event_name == "cron":
                action = payload.get("action", "")
                if action == "finished":
                    # 通知所有 cron waiter（单 gateway 实例通常同一时刻只有一个 cron 在跑）
                    with self._lock:
                        for _job_id, (evt, result_holder) in self._cron_waiters.items():
                            result_holder["payload"] = payload
                            evt.set()

            self._global_events.append(msg)

    def request(self, method, params=None, timeout=REQUEST_TIMEOUT):
        """发送 RPC 请求并同步等待响应"""
        req_id = str(uuid.uuid4())
        wait_event = threading.Event()

        with self._lock:
            self._pending[req_id] = {"event": wait_event, "result": None}

        self.ws.send(json.dumps({
            "type": "req", "id": req_id,
            "method": method, "params": params or {},
        }))

        if not wait_event.wait(timeout):
            with self._lock:
                self._pending.pop(req_id, None)
            raise TimeoutError(f"请求超时 ({timeout}s): {method}")

        with self._lock:
            result = self._pending.pop(req_id)["result"]
        return result

    def _fire(self, method, params):
        """发送请求不等待响应（fire-and-forget）"""
        self.ws.send(json.dumps({
            "type": "req", "id": str(uuid.uuid4()),
            "method": method, "params": params or {},
        }))

    # ──── Channel 状态 ────

    def get_channels_status(self):
        """获取所有 channel 的状态。返回: dict[str, ChannelStatus]"""
        resp = self.request("channels.status")
        if not resp.get("ok"):
            raise RuntimeError(f"channels.status 失败: {resp.get('error')}")

        payload = resp.get("payload", {})
        channels_data = payload.get("channels", {})
        accounts_data = payload.get("channelAccounts", {})

        return {
            ch_id: ChannelStatus(ch_id, ch_info, accounts_data.get(ch_id, []))
            for ch_id, ch_info in channels_data.items()
        }

    def is_channel_available(self, channel_id):
        """检查指定 channel 是否可用。返回: (bool, ChannelStatus|None)"""
        channels = self.get_channels_status()
        ch = channels.get(channel_id)
        return (ch.available, ch) if ch else (False, None)

    # ──── Session 管理 ────

    def get_sessions(self):
        """获取所有 session 列表。返回: list[SessionInfo]"""
        resp = self.request("sessions.list")
        if not resp.get("ok"):
            raise RuntimeError(f"sessions.list 失败: {resp.get('error')}")
        return [SessionInfo(s) for s in resp.get("payload", {}).get("sessions", [])]

    def get_sessions_by_channel(self, channel_id):
        """获取指定 channel 的所有 session"""
        return [s for s in self.get_sessions() if s.channel == channel_id]

    @staticmethod
    def get_main_session():
        """获取 main 会话的 key"""
        return "agent:main:main"

    # ──── 发消息 & 等待回复 ────

    def send_message(self, session_key, message, timeout=CHAT_TIMEOUT):
        """向指定 session 发送消息并等待 AI 回复。返回: ChatResult"""
        result = ChatResult()
        start = time.time()

        with self._lock:
            self._session_events[session_key] = []
            self._session_done[session_key] = threading.Event()

        self._fire("sessions.messages.subscribe", {"key": session_key})
        time.sleep(0.1)

        resp = self.request("sessions.send", {
            "key": session_key,
            "message": message,
            "idempotencyKey": str(uuid.uuid4()),
        }, timeout=30)

        if not resp.get("ok"):
            result.error = resp.get("error", {})
            result.elapsed = time.time() - start
            return result

        done_event = self._session_done[session_key]
        done_event.wait(timeout=timeout)
        time.sleep(0.5)

        assistant_texts = []
        with self._lock:
            events = self._session_events.pop(session_key, [])
            self._session_done.pop(session_key, None)

        for evt in events:
            msg = evt.get("payload", {}).get("message", {})
            if msg.get("role") == "assistant":
                content = msg.get("content", "")
                if isinstance(content, list):
                    assistant_texts.extend(
                        part["text"] for part in content
                        if isinstance(part, dict) and part.get("text")
                    )
                elif isinstance(content, str):
                    assistant_texts.append(content)

        result.success = True
        result.assistant_text = "".join(assistant_texts)
        result.events = events
        result.elapsed = time.time() - start
        return result

    def send_message_to_main(self, message, timeout=CHAT_TIMEOUT):
        """向 main 会话发送消息（快捷方法）"""
        return self.send_message(self.get_main_session(), message, timeout)

    # ──── 模型查询 ────

    def get_models(self):
        """获取可用模型列表"""
        resp = self.request("models.list")
        if not resp.get("ok"):
            raise RuntimeError(f"models.list 失败: {resp.get('error')}")
        return resp.get("payload", {}).get("models", [])

    def get_default_model_config(self):
        """
        获取实例的默认模型配置。

        返回: {"primary": "provider/model-id" | None, "fallbacks": [...]}
        """
        resp = self.request("config.get", {})
        if resp.get("ok"):
            model_config = (
                resp.get("payload", {})
                .get("config", {})
                .get("agents", {})
                .get("defaults", {})
                .get("model", {})
            )
            if model_config.get("primary"):
                return {
                    "primary": model_config["primary"],
                    "fallbacks": model_config.get("fallbacks", []),
                }
        return {"primary": None, "fallbacks": []}

    def get_hello_payload(self):
        """获取连接时收到的 hello payload"""
        return self._hello_payload or {}

    # ──── Cron 定时任务 ────

    def cron_list(self):
        """获取所有 cron 定时任务。返回: list[CronJobInfo]"""
        resp = self.request("cron.list")
        if not resp.get("ok"):
            raise RuntimeError(f"cron.list 失败: {resp.get('error')}")
        return [CronJobInfo(j) for j in resp.get("payload", {}).get("jobs", [])]

    def cron_add(self, name, schedule_expr, message, delivery=None,
                 enabled=True, session_target="isolated", wake_mode="now", tz="Asia/Shanghai"):
        """创建 cron 定时任务。返回: CronJobInfo"""
        params = {
            "name": name,
            "enabled": enabled,
            "schedule": {"kind": "cron", "expr": schedule_expr, "tz": tz},
            "sessionTarget": session_target,
            "wakeMode": wake_mode,
            "payload": {"kind": "agentTurn", "message": message},
        }
        if delivery:
            params["delivery"] = delivery

        resp = self.request("cron.add", params)
        if not resp.get("ok"):
            raise RuntimeError(f"cron.add 失败: {resp.get('error')}")
        return CronJobInfo(resp.get("payload", {}))

    def cron_update(self, job_id, patch):
        """更新 cron 定时任务。返回: CronJobInfo"""
        resp = self.request("cron.update", {"jobId": job_id, "patch": patch})
        if not resp.get("ok"):
            raise RuntimeError(f"cron.update 失败: {resp.get('error')}")
        return CronJobInfo(resp.get("payload", {}))

    def cron_remove(self, job_id):
        """删除 cron 定时任务"""
        resp = self.request("cron.remove", {"id": job_id})
        if not resp.get("ok"):
            raise RuntimeError(f"cron.remove 失败: {resp.get('error')}")
        return True

    def cron_run(self, job_id, wait=True, timeout=45):
        """手动触发 cron 任务执行。返回执行结果 dict"""
        start = time.time()

        # 注册 cron 完成等待器（必须在发请求之前注册，避免竞态）
        cron_event = threading.Event()
        cron_result_holder = {"payload": None}
        if wait:
            with self._lock:
                self._cron_waiters[job_id] = (cron_event, cron_result_holder)

        resp = self.request("cron.run", {"id": job_id})
        if not resp.get("ok"):
            with self._lock:
                self._cron_waiters.pop(job_id, None)
            raise RuntimeError(f"cron.run 失败: {resp.get('error')}")

        run_id = resp.get("payload", {}).get("runId", "")
        result = {
            "run_id": run_id, "status": "enqueued",
            "delivered": False, "delivery_status": "",
            "summary": "", "error": None, "elapsed": 0.0,
        }

        if not wait:
            result["elapsed"] = time.time() - start
            return result

        try:
            # 等待 _dispatch 路由过来的 cron finished 事件
            got_event = cron_event.wait(timeout=timeout)
            if got_event and cron_result_holder["payload"]:
                payload = cron_result_holder["payload"]
                result["status"] = payload.get("status", "unknown")
                result["delivered"] = payload.get("delivered", False)
                result["delivery_status"] = payload.get("deliveryStatus", "")
                result["summary"] = payload.get("summary", "")
                error = payload.get("error")
                result["error"] = error if error != "none" else None
            elif not got_event:
                result["error"] = f"等待 cron 完成超时 ({timeout}s)"
        finally:
            with self._lock:
                self._cron_waiters.pop(job_id, None)

        result["elapsed"] = time.time() - start
        return result

    def cron_runs(self, job_id, limit=10):
        """获取 cron 任务运行历史。返回: list[CronRunInfo]"""
        resp = self.request("cron.runs", {"jobId": job_id, "limit": limit})
        if not resp.get("ok"):
            raise RuntimeError(f"cron.runs 失败: {resp.get('error')}")
        return [CronRunInfo(r) for r in resp.get("payload", {}).get("runs", [])]


# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
#  验证函数
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

# 通道元数据
CHANNEL_META = {
    "feishu":              {"display": "飞书",      "to_fn": lambda cfg: f"user:{cfg.feishu_open_id}"},
    "wecom":              {"display": "企微",      "to_fn": lambda cfg: cfg.wecom_user_id},
    "qqbot":              {"display": "QQ 机器人", "to_fn": lambda cfg: f"qqbot:c2c:{cfg.qqbot_c2c_open_id}"},
    "ddingtalk":          {"display": "钉钉",      "to_fn": lambda cfg: cfg.dingtalk_user_id},
    "dingtalk-connector": {"display": "钉钉",      "to_fn": lambda cfg: cfg.dingtalk_user_id},
}

# Hatchery API 通道名 → Gateway 通道名映射
# Gateway 插件注册的名字可能与 Hatchery API 不一致
CHANNEL_NAME_GATEWAY_MAP = {
    "ddingtalk": "dingtalk-connector",
}


def parse_model_ref(model_ref):
    """解析 "provider/model-id"。返回: (provider, model_id)"""
    if not model_ref:
        return None, None
    if "/" in model_ref:
        provider, model_id = model_ref.split("/", 1)
        return provider, model_id
    return None, model_ref


def connect_from_inst(inst, timeout=15):
    """从 InstanceContext 创建并连接 Gateway 客户端"""
    assert inst.gateway_url, "InstanceContext 缺少 gateway_url"
    assert inst.gateway_token, "InstanceContext 缺少 gateway_token"
    base_url = inst.gateway_url.split("?")[0] if "?" in inst.gateway_url else inst.gateway_url
    gw = OpenClawGateway(gateway_url=base_url, token=inst.gateway_token)
    gw.connect(timeout=timeout, verbose=False)
    return gw


def _with_gateway_retry(inst, fn, *, timeout=60, poll_interval=10, verbose=True, label="操作"):
    """
    通用 Gateway 重试工具：重试连接 Gateway 并执行 fn(gw)。

    - fn(gw) 应返回结果值。
    - 若 fn 内抛出 AssertionError 则立即外抛（属于逻辑断言失败，不应重试）。
    - 其他异常（连接失败、超时等）会重试直到 timeout。
    - 返回 fn(gw) 的返回值。
    - 超时抛 AssertionError 之前，会调用 /openclaw/check-gateway-access 打印 hatchery 侧
      对 Gateway 的探测结果（accessible/message），便于区分"重启慢"与"Gateway 真挂了"。

    参数:
        inst:           InstanceContext，需包含 gateway_url / gateway_token
        fn:             接受一个 gw 参数的可调用对象
        timeout:        总超时时间（秒）
        poll_interval:  重试间隔（秒）
        verbose:        是否打印重试日志
        label:          日志中的操作名称
    """
    deadline = time.time() + timeout
    last_error = None
    attempt = 0

    while time.time() < deadline:
        attempt += 1
        try:
            gw = connect_from_inst(inst, timeout=min(15, max(5, deadline - time.time())))
            try:
                return fn(gw)
            finally:
                gw.close()
        except AssertionError:
            raise
        except Exception as e:
            last_error = e
            remaining = deadline - time.time()
            if remaining <= 0:
                break
            if verbose:
                print(f"    Gateway 连接失败（第 {attempt} 次），{poll_interval}s 后重试: {e}")
            time.sleep(min(poll_interval, remaining))

    # 超时前做一次 hatchery 侧诊断：用 check-gateway-access 看 Gateway 端口/服务是否真就绪
    diag = ""
    try:
        from helpers.instance import check_gateway_access  # 局部导入避免循环依赖
        if getattr(inst, "user_token", None) and getattr(inst, "db_id", None):
            data = check_gateway_access(inst.user_token, inst.db_id)
            if data is None:
                diag = "check-gateway-access 返回非 200"
            else:
                diag = (
                    f"accessible={data.get('accessible')}, "
                    f"port={data.get('port')}, "
                    f"message={data.get('message')!r}"
                )
    except Exception as diag_err:
        diag = f"诊断调用失败: {diag_err}"

    if verbose and diag:
        print(f"    [诊断] hatchery 侧 Gateway 探测: {diag}")

    raise AssertionError(
        f"{label} 超时 ({timeout}s)，最后错误: {last_error}"
        + (f" | 诊断: {diag}" if diag else "")
    )


def verify_model_config(gw, expected_primary=None, expected_fallbacks=None, verbose=True,
                        model_name_map=None):
    """
    验证实例的默认模型配置（primary + fallback）。

    参数:
        gw:                 已连接的 OpenClawGateway 实例
        expected_primary:   期望的 primary 引用，None 时只验证存在
        expected_fallbacks: 期望的 fallback 引用列表，None 时不验证
        verbose:            是否打印详细信息
        model_name_map:     可选，{model_ref: model_name} 映射，用于日志中显示可读名称

    返回: dict {success, primary, fallbacks, primary_provider, primary_model_id, error}
    """
    result = {
        "success": False, "primary": None, "fallbacks": [],
        "primary_provider": None, "primary_model_id": None, "error": None,
    }

    def _display(ref):
        """返回带 model_name 的可读显示"""
        if model_name_map and ref in model_name_map:
            return f"{ref} [{model_name_map[ref]}]"
        return ref

    try:
        model_config = gw.get_default_model_config()
        primary_ref = model_config["primary"]
        fallback_refs = model_config["fallbacks"]

        result["primary"] = primary_ref
        result["fallbacks"] = fallback_refs

        if verbose:
            print(f"  Primary:   {_display(primary_ref) if primary_ref else '(未配置)'}")
            for i, fb in enumerate(fallback_refs, 1):
                print(f"  Fallback {i}: {_display(fb)}")
            if not fallback_refs:
                print(f"  Fallback:  (无)")

        if not primary_ref:
            result["error"] = "未找到 agents.defaults.model.primary 配置"
            return result

        provider, model_id = parse_model_ref(primary_ref)
        result["primary_provider"] = provider
        result["primary_model_id"] = model_id

        if expected_primary is not None and primary_ref != expected_primary:
            result["error"] = f"primary 不匹配: 期望 {expected_primary!r}, 实际 {primary_ref!r}"
            return result

        if expected_fallbacks is not None and sorted(fallback_refs) != sorted(expected_fallbacks):
            result["error"] = f"fallback 不匹配: 期望 {expected_fallbacks!r}, 实际 {fallback_refs!r}"
            return result

        result["success"] = True

    except Exception as e:
        result["error"] = str(e)

    return result


def verify_model_config_via_inst(inst, expected_primary=None, expected_fallbacks=None,
                                  expected_fallback_count=None, timeout=60, poll_interval=10,
                                  verbose=True, model_name_map=None):
    """
    通过 InstanceContext 验证模型配置（verify_model_config 的便捷封装）。
    内置重试机制：Gateway 可能暂时 502。

    额外支持 expected_fallback_count 校验 fallback 数量。
    失败时抛出 AssertionError。
    """

    def _check(gw):
        result = verify_model_config(gw, expected_primary=expected_primary,
                                     expected_fallbacks=expected_fallbacks, verbose=verbose,
                                     model_name_map=model_name_map)

        assert result["success"], f"模型配置验证失败: {result['error']}"

        if expected_fallback_count is not None:
            # Gateway 可能将 primary 也放入 fallbacks 列表中作为兜底，
            # 计算"额外的 fallback"数量时需要排除与 primary 相同的条目。
            primary_ref = result.get("primary")
            all_fallbacks = result.get("fallbacks", [])
            extra_fallbacks = [fb for fb in all_fallbacks if fb != primary_ref]
            actual_count = len(extra_fallbacks)
            assert actual_count == expected_fallback_count, (
                f"fallback 数量不匹配: 期望 {expected_fallback_count}, 实际 {actual_count}, "
                f"fallbacks={all_fallbacks}, primary={primary_ref}"
            )
            if verbose:
                print(f"    fallback 数量校验通过 ✓ (extra_count={actual_count})")

        return result

    return _with_gateway_retry(
        inst, _check,
        timeout=timeout, poll_interval=poll_interval, verbose=verbose,
        label="verify_model_config_via_inst",
    )


def verify_channel_delivery(gw, channel, config=None,
                            message="这是一条集成测试消息，请简短回复确认收到。",
                            timeout=60, channel_ready_timeout=90, verbose=False):
    """
    验证指定通道的完整投递流程:
    channel 状态检查(含等待running) → 创建定时任务 → 立刻触发 → 验证结果 → 清理

    参数:
        channel_ready_timeout: 等待通道变为 running=True 的超时时间（秒），默认 90s。
            钉钉等通道需要与第三方服务器建立长连接，configured=True 后仍需时间才能 running=True。

    返回: dict {success, channel, channel_available, job_created, run_result, delivered, error}
    """
    meta = CHANNEL_META.get(channel)
    if not meta:
        return {"success": False, "channel": channel, "error": f"不支持的通道: {channel}",
                "channel_available": False, "job_created": False, "run_result": None, "delivered": False}

    # 将 Hatchery API 通道名映射为 Gateway 侧通道名
    gw_channel = CHANNEL_NAME_GATEWAY_MAP.get(channel, channel)

    cfg = config or default_channel_config
    display = meta["display"]
    created_job_id = None

    result = {
        "success": False, "channel": channel, "channel_available": False,
        "job_created": False, "run_result": None, "delivered": False, "error": None,
    }

    try:
        # 1. channel 状态 — 带重试等待 running=True
        if verbose:
            print(f"  [{display}] 检查 channel 状态...")

        poll_interval = 10
        deadline = time.time() + channel_ready_timeout
        available = False
        ch_status = None
        poll_attempt = 0

        while time.time() < deadline:
            poll_attempt += 1
            available, ch_status = gw.is_channel_available(gw_channel)
            if available:
                break
            # 钉钉通道特殊处理：通道级别 running 永远为 False，
            # configured=True 即视为可用
            if gw_channel == "dingtalk-connector" and ch_status and ch_status.configured:
                available = True
                if verbose:
                    print(f"  [{display}] 钉钉通道 configured=True，视为可用")
                break
            # 如果 configured=True 但 running=False，说明通道正在启动中，继续等待
            if ch_status and ch_status.configured and not ch_status.running:
                remaining = deadline - time.time()
                if remaining <= 0:
                    break
                if verbose:
                    print(f"  [{display}] configured=True 但 running=False，"
                          f"等待通道就绪（第 {poll_attempt} 次，{int(remaining)}s 剩余）...")
                time.sleep(min(poll_interval, remaining))
                continue
            # 其他情况（通道不存在或未配置）直接退出
            break

        result["channel_available"] = available

        if not available:
            result["error"] = (
                f"{display} channel 不可用 "
                f"(configured={getattr(ch_status, 'configured', False)}, "
                f"running={getattr(ch_status, 'running', False)})"
            )
            return result

        # 2. 创建定时任务
        delivery_to = meta["to_fn"](cfg)
        job = gw.cron_add(
            name=f"inttest-{channel}-{int(time.time())}",
            schedule_expr="0 10 * * *",
            message=message,
            delivery={"mode": "announce", "channel": gw_channel, "to": delivery_to},
            enabled=True,
        )
        created_job_id = job.id
        result["job_created"] = True

        if verbose:
            print(f"  [{display}] 创建成功 (id={job.id[:12]}..., to={delivery_to})")

        # 3. 立刻触发（含重试：gateway 重启后 cron job 可能被中断）
        max_run_attempts = 3
        run_result = None
        triggered = False

        for run_attempt in range(1, max_run_attempts + 1):
            run_result = gw.cron_run(job.id, wait=True, timeout=timeout)
            result["run_result"] = run_result
            result["delivered"] = run_result["delivered"]

            # 4. 验证结果
            if not run_result["run_id"]:
                result["error"] = "未获取到 run_id"
                return result

            triggered = run_result["status"] in ("ok", "delivered", "enqueued") or run_result["delivered"]

            if triggered:
                break

            # 如果首次判定未成功，二次查询 cron_runs 历史确认（消息可能实际已投递但状态上报延迟）
            if created_job_id:
                if verbose:
                    print(f"  [{display}] 第 {run_attempt} 次触发未投递 (status={run_result['status']}), 等待后二次确认...")
                time.sleep(10)
                try:
                    runs = gw.cron_runs(created_job_id, limit=5)
                    for run in runs:
                        if getattr(run, "delivered", False) or getattr(run, "status", "") in ("ok", "delivered"):
                            triggered = True
                            result["delivered"] = True
                            if verbose:
                                print(f"  [{display}] 二次确认投递成功 ✓")
                            break
                except Exception:
                    pass

            if triggered:
                break

            # 未成功且还有重试机会：可能是 gateway restart 导致中断，等待后重试
            if run_attempt < max_run_attempts:
                if verbose:
                    print(f"  [{display}] 第 {run_attempt} 次触发失败，等待 5s 后重试...")
                time.sleep(5)

        if triggered:
            result["success"] = True
        else:
            result["error"] = f"触发异常: status={run_result['status']}, delivered={run_result['delivered']}"

        return result

    except Exception as e:
        result["error"] = str(e)
        return result

    finally:
        if created_job_id:
            try:
                gw.cron_remove(created_job_id)
            except Exception:
                pass


# ──── 便捷单通道函数 ────

def verify_feishu_delivery(gw, **kwargs):
    """验证飞书通道投递"""
    return verify_channel_delivery(gw, channel="feishu", **kwargs)


def verify_wecom_delivery(gw, **kwargs):
    """验证企微通道投递"""
    return verify_channel_delivery(gw, channel="wecom", **kwargs)


def verify_qqbot_delivery(gw, **kwargs):
    """验证 QQ 机器人通道投递"""
    return verify_channel_delivery(gw, channel="qqbot", **kwargs)


def verify_dingtalk_delivery(gw, **kwargs):
    """验证钉钉通道投递"""
    return verify_channel_delivery(gw, channel="ddingtalk", **kwargs)


# ──── 通过 InstanceContext 的便捷验证 ────

def verify_model_available(inst, timeout=180, poll_interval=5, verbose=True):
    """
    通过 Gateway WS 验证模型已就绪（primary 非空）。
    内置重试机制：添加模型后 agent 可能在重启，Gateway 会返回 502。

    set_model.sh 写完 openclaw.json 后会 systemctl --user 重启 openclaw-gateway，
    重启窗口期内 5800 端口对外不可达，反向代理回 502。默认 180s/5s 给重启留足空间。

    失败时抛出 AssertionError。
    """

    def _check(gw):
        model_config = gw.get_default_model_config()
        primary = model_config.get("primary")

        if verbose:
            print(f"    模型配置: primary={primary}, fallbacks={model_config.get('fallbacks', [])}")

        assert primary, f"Gateway 连接成功但 primary 模型未配置: {model_config}"

        if verbose:
            print(f"    模型可用性验证通过 (via Gateway WS)")
        return True

    return _with_gateway_retry(
        inst, _check,
        timeout=timeout, poll_interval=poll_interval, verbose=verbose,
        label="verify_model_available",
    )


def verify_channel_configured(inst, channel_name, timeout=60, poll_interval=10, verbose=True):
    """
    通过 Gateway WS 验证通道配置已生效（configured=true）。
    内置重试机制：Gateway 可能暂时 502。

    返回: ChannelStatus
    失败时抛出 AssertionError。
    """
    # 将 Hatchery API 通道名映射为 Gateway 侧通道名
    gw_channel_name = CHANNEL_NAME_GATEWAY_MAP.get(channel_name, channel_name)

    deadline = time.time() + timeout
    last_error = None
    attempt = 0
    diagnosed = False  # 只做一次深度诊断，避免刷屏

    while time.time() < deadline:
        attempt += 1
        try:
            gw = connect_from_inst(inst, timeout=min(15, max(5, deadline - time.time())))
            try:
                channels = gw.get_channels_status()
                ch = channels.get(gw_channel_name)

                if ch is None:
                    # 通道尚未出现在 Gateway 中，可能配置还在同步，继续重试
                    remaining = deadline - time.time()
                    if remaining <= 0:
                        gw_hint = f" (gateway_name={gw_channel_name})" if gw_channel_name != channel_name else ""
                        raise AssertionError(
                            f"通道 {channel_name}{gw_hint} 未在 Gateway channels.status 中找到, "
                            f"可用通道: {list(channels.keys())}"
                        )
                    if verbose:
                        gw_hint = f" (gateway={gw_channel_name})" if gw_channel_name != channel_name else ""
                        print(f"    通道 {channel_name}{gw_hint} 尚未出现（第 {attempt} 次），"
                              f"可用通道: {list(channels.keys())}，{poll_interval}s 后重试")

                    # 通道列表为空时做深度诊断（仅一次）
                    if not channels and not diagnosed and verbose:
                        diagnosed = True
                        _diagnose_empty_channels(gw, channel_name)

                    time.sleep(min(poll_interval, remaining))
                    continue

                if not ch.configured:
                    # 通道已出现但尚未配置完成，继续重试
                    remaining = deadline - time.time()
                    if remaining <= 0:
                        raise AssertionError(f"通道 {channel_name} 未配置 (configured=false)")
                    if verbose:
                        print(f"    通道 {channel_name}: configured=false（第 {attempt} 次），"
                              f"{poll_interval}s 后重试")
                    time.sleep(min(poll_interval, remaining))
                    continue

                if verbose:
                    print(f"    通道 {channel_name}: configured={ch.configured}, running={ch.running}")
                    if ch.last_error:
                        print(f"    ⚠ lastError={ch.last_error}")
                    print(f"    通道 {channel_name} 配置已确认 ✓ (via Gateway WS)")

                return ch
            finally:
                gw.close()
        except AssertionError:
            raise
        except Exception as e:
            last_error = e
            remaining = deadline - time.time()
            if remaining <= 0:
                break
            if verbose:
                print(f"    Gateway 连接失败（第 {attempt} 次），{poll_interval}s 后重试: {e}")
            time.sleep(min(poll_interval, remaining))

    # 超时前做最终诊断
    if verbose:
        try:
            gw = connect_from_inst(inst, timeout=10)
            try:
                print(f"    ┌─── 最终诊断（超时 {timeout}s）───")
                _diagnose_empty_channels(gw, channel_name)
            finally:
                gw.close()
        except Exception as e:
            print(f"    最终诊断连接失败: {e}")

    raise AssertionError(f"verify_channel_configured 超时 ({timeout}s)，最后错误: {last_error}")


def _diagnose_empty_channels(gw, channel_name):
    """
    当 channels.status 返回空字典时，进行深度诊断：
    - 打印 channels.status 原始响应
    - 打印 config.get 完整配置（含通道详细配置，可定位字段缺失问题）
    """
    print("    ┌─── 诊断：channels.status 返回空 ───")

    # 1. channels.status 原始响应（不经过解析）
    try:
        raw_resp = gw.request("channels.status", {}, timeout=10)
        print(f"    │ channels.status 原始响应: {json.dumps(raw_resp, ensure_ascii=False)}")
    except Exception as e:
        print(f"    │ channels.status 请求异常: {e}")

    # 2. 完整实例配置（核心诊断信息）
    try:
        resp = gw.request("config.get", {}, timeout=10)
        if resp.get("ok"):
            cfg = resp.get("payload", {}).get("config", {})
            # 输出完整配置，便于排查通道字段是否缺失
            print(f"    │ config.get 完整配置:")
            cfg_str = json.dumps(cfg, ensure_ascii=False, indent=2)
            for line in cfg_str.split("\n"):
                print(f"    │   {line}")
        else:
            print(f"    │ config.get 失败: {resp.get('error')}")
    except Exception as e:
        print(f"    │ config.get 异常: {e}")

    print("    └───────────────────────────────────")
