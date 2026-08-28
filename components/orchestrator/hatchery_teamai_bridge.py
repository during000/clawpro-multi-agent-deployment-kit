#!/usr/bin/env python3
"""Bridge from the ClawPro PoC UI to Hatchery and a local or remote TeamAI."""

from __future__ import annotations

import json
import os
import platform
import shutil
import subprocess
import threading
import time
import uuid
from pathlib import Path
from urllib.error import HTTPError, URLError
from urllib.parse import urlencode
from urllib.request import Request, urlopen


class BridgeError(RuntimeError):
    pass


class HatcheryTeamAIBridge:
    """Create real Hatchery tasks and wake the local TeamAI consumer."""

    def __init__(self, api_url: str, admin_token: str, root: Path):
        self.api_url = api_url.rstrip("/")
        self.admin_token = admin_token
        self.root = root
        self.workspace_root = root / "real-agent-workspaces" / "hatchery-teamai"
        self.teamai_home = root / ".teamai-live"
        self.remote_mode = os.environ.get("TEAMAI_REMOTE_MODE", "").strip() == "1"
        self.remote_workspace = os.environ.get("TEAMAI_REMOTE_WORKSPACE", "").strip()
        self.remote_source_workspace_root = os.environ.get(
            "TEAMAI_REMOTE_SOURCE_WORKSPACE_ROOT",
            str(Path(self.remote_workspace).parent) if self.remote_workspace else "",
        ).strip()
        self.public_endpoint = os.environ.get("TEAMAI_PUBLIC_ENDPOINT", "").strip().rstrip("/")
        self.remote_device_name = os.environ.get(
            "TEAMAI_REMOTE_DEVICE_NAME", "用户电脑 TeamAI"
        ).strip()
        self.remote_imate_agent_id = os.environ.get(
            "TEAMAI_REMOTE_IMATE_AGENT_ID", ""
        ).strip()
        self.remote_imate_agent_name = os.environ.get(
            "TEAMAI_REMOTE_IMATE_AGENT_NAME", "iMate OpenClaw"
        ).strip()
        self.bootstrap_path = root / ".runtime" / "remote-teamai-bootstrap.json"
        self.teamai_bin = Path(
            os.environ.get("TEAMAI_BIN", str(Path.home() / ".teamai/bin/teamai"))
        )
        self.imate_bin = Path(
            os.environ.get("TEAMAI_IMATE_PATH", str(Path.home() / ".local/bin/imate"))
        )
        self.lock = threading.RLock()
        self.ready = False
        self.setup_error = ""
        self.user_id = 0
        self.user_token = ""
        self.project_id = 0
        self.project_name = "ClawPro 本地 Agent 联调"
        self.agent_id = uuid.uuid4().hex[:16]
        self.agent_version = "0.19.0-local-live"
        self.host_name = platform.node() or "local-mac"
        self.instance_id = ""
        self.instance_db_id = 0
        self.workspace_bindings: dict[str, dict] = {}
        self.listener_process: subprocess.Popen | None = None
        self.listener_log_handle = None
        self.listener_started_at = ""

    def request(self, method: str, path: str, *, token: str = "", body=None, form=None):
        data = None
        headers = {"Accept": "application/json"}
        if token:
            headers["Authorization"] = f"Bearer {token}"
        if body is not None:
            data = json.dumps(body, ensure_ascii=False).encode("utf-8")
            headers["Content-Type"] = "application/json"
        elif form is not None:
            data = urlencode(form, doseq=True).encode("utf-8")
            headers["Content-Type"] = "application/x-www-form-urlencoded"
        request = Request(self.api_url + path, data=data, headers=headers, method=method)
        try:
            with urlopen(request, timeout=30) as response:
                raw = response.read()
                return json.loads(raw.decode("utf-8")) if raw else {}
        except HTTPError as error:
            detail = error.read().decode("utf-8", errors="replace")
            try:
                detail = json.loads(detail).get("error", detail)
            except json.JSONDecodeError:
                pass
            raise BridgeError(f"Hatchery {method} {path} 失败：{detail}") from error
        except URLError as error:
            raise BridgeError(f"无法连接 Hatchery：{self.api_url}（{error.reason}）") from error

    def setup(self):
        try:
            health = self.request("GET", "/health")
            if health.get("status") != "ok":
                raise BridgeError(f"Hatchery 健康检查失败：{health}")
            if not self.remote_mode and not self.teamai_bin.is_file():
                raise BridgeError(f"未找到 TeamAI：{self.teamai_bin}")
            if self.remote_mode:
                if not self.remote_workspace or not Path(self.remote_workspace).is_absolute():
                    raise BridgeError("远程 TeamAI 模式需要绝对路径 TEAMAI_REMOTE_WORKSPACE")
                if not self.public_endpoint.startswith(("http://", "https://")):
                    raise BridgeError("远程 TeamAI 模式需要 TEAMAI_PUBLIC_ENDPOINT")

            config = self.request("GET", "/admin/config", token=self.admin_token)
            site_config = config.get("config", config)
            if site_config.get("local_agent_enabled") not in (True, "true", 1):
                self.request(
                    "POST",
                    "/admin/config",
                    token=self.admin_token,
                    form={"local_agent_enabled": "true"},
                )

            if self.remote_mode and self.bootstrap_path.is_file():
                self.load_remote_bootstrap()
            else:
                self.provision_test_identity()

            if self.remote_mode:
                self.write_remote_bootstrap()
                try:
                    self.resolve_instance()
                except BridgeError:
                    # The user-side TeamAI reports the device after receiving
                    # this bootstrap. Task creation retries resolution.
                    self.instance_id = ""
                    self.instance_db_id = 0
                self.ready = True
                self.setup_error = ""
                return

            bootstrap = self.workspace_root / "bootstrap"
            bootstrap.mkdir(parents=True, exist_ok=True)
            self.report_workspace(bootstrap)
            self.resolve_instance()
            self.write_teamai_config()
            self.start_resident_listener(bootstrap)
            self.ready = True
            self.setup_error = ""
        except Exception as error:
            self.ready = False
            self.setup_error = str(error)
            raise

    def provision_test_identity(self):
        suffix = f"{int(time.time())}-{uuid.uuid4().hex[:5]}"
        username = f"inttest-user-agent-task-ui-{suffix}"
        user = self.request(
            "POST",
            "/admin/create",
            token=self.admin_token,
            form={
                "username": username,
                "password": f"Ui{uuid.uuid4().hex[:12]}!8",
                "role": "user",
                "instance_quota": "10",
                "token_quota_day": "-1",
            },
        )
        self.user_id = int(user["id"])
        token_info = self.request(
            "GET",
            f"/admin/user-token?id={self.user_id}",
            token=self.admin_token,
        )
        self.user_token = token_info["token"]

        project = self.request(
            "POST",
            "/admin/projects/create",
            token=self.admin_token,
            body={"name": f"{self.project_name}-{suffix}"},
        )["project"]
        self.project_id = int(project["id"])
        self.project_name = project["name"]
        self.request(
            "POST",
            "/admin/projects/members/add",
            token=self.admin_token,
            body={"id": self.project_id, "user_ids": [self.user_id]},
        )

    def load_remote_bootstrap(self):
        data = json.loads(self.bootstrap_path.read_text(encoding="utf-8"))
        self.user_id = int(data["user_id"])
        self.user_token = str(data["token"])
        self.project_id = int(data["project_id"])
        self.project_name = str(data["project_name"])
        self.agent_id = str(data["agent_id"])

    def remote_bootstrap_data(self):
        return {
            "endpoint": self.public_endpoint,
            "token": self.user_token,
            "user_id": self.user_id,
            "project_id": self.project_id,
            "project_name": self.project_name,
            "agent_id": self.agent_id,
            "workspace": self.remote_workspace,
            "routes": {
                "projects": "/projects/mine",
                "report": "/local-agent/report",
                "sync": "/local-agent/sync",
                "ack": "/local-agent/commands/ack",
                "wakeTicket": "/local-agent/wake-ticket",
                "wake": "/local-agent/wake",
                "getConfig": "/local-agent/get-config",
            },
        }

    def write_remote_bootstrap(self):
        self.bootstrap_path.parent.mkdir(parents=True, exist_ok=True)
        self.bootstrap_path.write_text(
            json.dumps(self.remote_bootstrap_data(), ensure_ascii=False, indent=2),
            encoding="utf-8",
        )
        self.bootstrap_path.chmod(0o600)

    def report_workspace(self, workspace: Path):
        workspace.mkdir(parents=True, exist_ok=True)
        self.request(
            "POST",
            "/local-agent/report",
            token=self.user_token,
            body={
                "local_agent_id": self.agent_id,
                "agent_type": "codebuddy",
                "agent_version": self.agent_version,
                "host_name": self.host_name,
                "os": f"{platform.system().lower()}/{platform.machine()}",
                "workspaces": [
                    {
                        "path": str(workspace),
                        "name": self.project_name,
                        "ide_type": "codebuddy",
                        "project_id": self.project_id,
                        "skills": [],
                        "rules": [],
                    }
                ],
            },
        )
        self.workspace_bindings[str(workspace)] = {
            "projectId": self.project_id,
            "projectName": self.project_name,
            "boundAt": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
            "ideType": "codebuddy",
        }

    def resolve_instance(self):
        data = self.request(
            "GET",
            "/openclaw/list?page=1&page_size=100",
            token=self.user_token,
        )
        expected = f"local-codebuddy-{self.agent_id[-6:]}"
        for instance in data.get("instances", []):
            instance_id = instance.get("instance_id") or instance.get("InstanceId")
            if instance_id == expected:
                self.instance_id = instance_id
                self.instance_db_id = int(instance.get("id") or instance.get("ID"))
                return
        raise BridgeError(f"Hatchery 未返回本地 CodeBuddy 实例：{expected}")

    def write_teamai_config(self):
        self.teamai_home.mkdir(parents=True, exist_ok=True)
        config_path = self.teamai_home / "config.json"
        config_path.write_text(
            json.dumps(
                {
                    "endpoint": self.api_url,
                    "token": self.user_token,
                    "createdAt": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
                    "routes": {
                        "projects": "/projects/mine",
                        "report": "/local-agent/report",
                        "sync": "/local-agent/sync",
                        "ack": "/local-agent/commands/ack",
                        "wakeTicket": "/local-agent/wake-ticket",
                        "wake": "/local-agent/wake",
                        "getConfig": "/local-agent/get-config",
                    },
                    "workspaceBindings": self.workspace_bindings,
                },
                ensure_ascii=False,
                indent=2,
            ),
            encoding="utf-8",
        )
        config_path.chmod(0o600)

    def list_imate_openclaw_agents(self):
        if self.remote_mode:
            if not self.remote_imate_agent_id:
                return []
            return [
                {
                    "id": self.remote_imate_agent_id,
                    "name": self.remote_imate_agent_name,
                    "status": "enabled",
                    "online_status": "online",
                }
            ]
        if not self.imate_bin.is_file():
            raise BridgeError(f"未找到 iMate CLI：{self.imate_bin}")
        result = subprocess.run(
            [str(self.imate_bin), "agent", "list", "--output", "json"],
            text=True,
            capture_output=True,
            timeout=20,
            check=False,
        )
        if result.returncode != 0:
            raise BridgeError((result.stderr or result.stdout or "iMate Agent 列表读取失败").strip())
        try:
            agents = json.loads(result.stdout)
        except json.JSONDecodeError as error:
            raise BridgeError("iMate Agent 列表不是有效 JSON") from error
        return [
            {
                "id": str(agent.get("id") or ""),
                "name": str(agent.get("name") or "未命名 OpenClaw Agent"),
                "status": str(agent.get("status") or "unknown"),
                "online_status": str(agent.get("online_status") or agent.get("display_state") or "unknown"),
            }
            for agent in agents
            if agent.get("provider") == "openclaw"
            and agent.get("kind") == "imate"
            and agent.get("enabled") is not False
            and (agent.get("online_status") == "online" or agent.get("display_state") == "online")
            and agent.get("id")
        ]

    def create_task(
        self,
        prompt: str,
        *,
        executor: str = "codebuddy",
        target_agent_id: str = "",
        imate_project_id: str = "",
        delivery_mode: str = "wss",
        seed_workspace: Path | None = None,
        repository_url: str = "",
    ):
        if not self.ready:
            raise BridgeError(self.setup_error or "Hatchery—TeamAI Bridge 尚未就绪")
        with self.lock:
            local_id = "cloud_" + uuid.uuid4().hex[:10]
            if self.remote_mode:
                self.resolve_instance()
                workspace = self.remote_workspace_for_repository(repository_url)
            else:
                workspace = self.workspace_root / local_id
                workspace.mkdir(parents=True, exist_ok=True)
                if seed_workspace:
                    source_root = seed_workspace.resolve()
                    for source in source_root.rglob("*"):
                        if not source.is_file() or ".git" in source.parts:
                            continue
                        relative = source.relative_to(source_root)
                        destination = workspace / relative
                        destination.parent.mkdir(parents=True, exist_ok=True)
                        shutil.copy2(source, destination)
                task_file = workspace / "TASK.md"
                if not task_file.exists():
                    task_file.write_text(
                        "# ClawPro 云—本 Agent 任务\n\n"
                        "该目录由 Hatchery 绑定到本地 CodeBuddy，只允许在此目录内操作。\n",
                        encoding="utf-8",
                    )
                self.report_workspace(workspace)
                self.write_teamai_config()
            response = self.request(
                "POST",
                "/agent-tasks/create",
                token=self.user_token,
                body={
                    "instance_id": self.instance_db_id,
                    "project_id": self.project_id,
                    "workspace_path": str(workspace),
                    "prompt": prompt,
                    "executor": executor,
                    "target_agent_id": target_agent_id,
                    "imate_project_id": imate_project_id,
                    "delivery_mode": delivery_mode,
                },
            )
            task = response["task"]
        return local_id, workspace, task, bool(response.get("wake_delivered"))

    def remote_workspace_for_repository(self, repository_url: str) -> Path:
        """Resolve a declared repository to a deterministic user-side workspace.

        The server cannot inspect the user's filesystem. TeamAI is still the
        trust boundary: it accepts the task only when this exact path exists and
        is bound to the same ClawPro project.
        """
        repository_url = str(repository_url or "").strip().rstrip("/")
        if not repository_url:
            return Path(self.remote_workspace)
        repository_name = repository_url.rsplit("/", 1)[-1]
        if ":" in repository_name:
            repository_name = repository_name.rsplit(":", 1)[-1]
        if repository_name.endswith(".git"):
            repository_name = repository_name[:-4]
        allowed = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789._-"
        if (
            not repository_name
            or repository_name in {".", ".."}
            or any(character not in allowed for character in repository_name)
        ):
            raise BridgeError("无法从仓库地址确定受控工作区")
        if not self.remote_source_workspace_root:
            raise BridgeError("未配置用户侧源码工作区根目录")
        return Path(self.remote_source_workspace_root) / repository_name

    def listener_env(self):
        return {
            **os.environ,
            "TEAMAI_LOCAL_AGENT_HOME": str(self.teamai_home),
            "TEAMAI_LOCAL_AGENT_ID": self.agent_id,
            "TEAMAI_BIND_PROMPT_ENABLED": "0",
            "TEAMAI_AGENT_TASK_TIMEOUT_MS": "1200000",
            "TEAMAI_IMATE_PATH": str(self.imate_bin),
        }

    def start_resident_listener(self, workspace: Path):
        if self.remote_mode:
            return
        if self.listener_process and self.listener_process.poll() is None:
            return
        log_path = self.teamai_home / "resident-listener.log"
        self.listener_log_handle = log_path.open("a", encoding="utf-8")
        self.listener_process = subprocess.Popen(
            [
                str(self.teamai_bin),
                "agent-task-listen",
                "--tool",
                "codebuddy",
                "--cwd",
                str(workspace),
            ],
            env=self.listener_env(),
            stdout=self.listener_log_handle,
            stderr=subprocess.STDOUT,
            text=True,
        )
        time.sleep(0.35)
        if self.listener_process.poll() is not None:
            detail = log_path.read_text(encoding="utf-8", errors="replace")[-1600:]
            raise BridgeError(detail.strip() or "TeamAI 常驻监听器启动失败")
        self.listener_started_at = time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())

    def stop_resident_listener(self):
        if self.remote_mode:
            return
        process = self.listener_process
        if process and process.poll() is None:
            process.terminate()
            try:
                process.wait(timeout=3)
            except subprocess.TimeoutExpired:
                process.kill()
        self.listener_process = None
        if self.listener_log_handle:
            self.listener_log_handle.close()
            self.listener_log_handle = None

    def wake_teamai(self, workspace: Path):
        if self.remote_mode:
            # Hatchery already notified the user-side resident listener via WSS.
            return
        env = self.listener_env()
        hook_input = json.dumps(
            {
                "cwd": str(workspace),
                "hook_event_name": "UserPromptSubmit",
                "session_id": "clawpro-live-ui",
                "prompt": "poll ClawPro tasks",
            },
            ensure_ascii=False,
        )
        result = subprocess.run(
            [
                str(self.teamai_bin),
                "hook-dispatch",
                "prompt-submit",
                "--tool",
                "codebuddy",
                "--stdin",
            ],
            input=hook_input,
            text=True,
            capture_output=True,
            timeout=15,
            env=env,
            check=False,
        )
        if result.returncode != 0:
            detail = (result.stderr or result.stdout or "TeamAI 启动失败")[-1200:]
            raise BridgeError(detail.strip())

    def get_task(self, backend_task_id: int):
        data = self.request(
            "GET",
            f"/agent-tasks?id={backend_task_id}",
            token=self.user_token,
        )
        tasks = data.get("tasks", [])
        if not tasks:
            raise BridgeError(f"Hatchery 任务不存在：{backend_task_id}")
        return tasks[0]

    def public_device(self):
        if self.remote_mode:
            try:
                self.resolve_instance()
            except BridgeError:
                pass
            listener_online = self.instance_db_id > 0
        else:
            listener_online = bool(
                self.listener_process and self.listener_process.poll() is None
            )
        return {
            "bridge_id": (
                "teamai-remote-agent-live"
                if self.remote_mode
                else "teamai-local-agent-live"
            ),
            "organization": self.project_name,
            "device_id": self.agent_id,
            "device_name": self.remote_device_name if self.remote_mode else self.host_name,
            "trusted": self.ready,
            "transport": "WSS 唤醒 + HTTPS sync / ack（真实联调）",
            "edge_runtime_managed": self.ready,
            "instance_id": self.instance_id,
            "setup_error": self.setup_error,
            "resident_listener": {
                "online": listener_online,
                "pid": (
                    None
                    if self.remote_mode
                    else self.listener_process.pid if listener_online else None
                ),
                "started_at": self.listener_started_at or None,
            },
        }
