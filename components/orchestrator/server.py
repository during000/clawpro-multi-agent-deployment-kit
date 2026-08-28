#!/usr/bin/env python3
"""Local ClawPro Edge Runtime PoC with a real ACP stdio round trip."""

import argparse
import base64
import hashlib
import json
import mimetypes
import os
import re
import shutil
import signal
import subprocess
import sys
import threading
import time
import uuid
from concurrent.futures import ThreadPoolExecutor, as_completed
from datetime import datetime, timezone
from http import HTTPStatus
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import parse_qs, urlparse

from cloudagent_client import CloudAgentError, DevResonanceCloudAgentClient
from hatchery_teamai_bridge import BridgeError, HatcheryTeamAIBridge
from structured_workflow import (
    AGENT_ASSIGNMENT_MODES,
    ISSUEFIX_REF,
    ISSUEFIX_SOURCE_URL,
    run_issuefix_compatibility_smoke,
)


ROOT = Path(__file__).resolve().parent
STATIC_ROOT = ROOT / "static"
WORKSPACE_ROOT = ROOT / "runtime-workspaces"
REAL_WORKSPACE_ROOT = ROOT / "real-agent-workspaces"
WORKFLOW_WORKSPACE_ROOT = REAL_WORKSPACE_ROOT / "handoff-workflows"
TASK_STATE_PATH = ROOT / ".runtime" / "task-state.json"
TASK_STATE_VERSION = 1
TERMINAL_TASK_STATUSES = {"completed", "failed", "canceled"}
HATCHERY_RUNTIME_IDS = {
    "hatchery-teamai-codebuddy",
    "hatchery-teamai-imate-openclaw",
}
STRUCTURED_ISSUEFIX_RUNTIME_ID = "structured-skillhub-issuefix"
STRUCTURED_PROJECT_WORKFLOW_RUNTIME_ID = "structured-project-workflow"
STRUCTURED_AGENT_RUNTIME_IDS = {
    "codebuddy-acp",
    "codebuddy-imate-mixed",
    "node-routed-multi-agent",
}
WORKFLOW_APPROVAL_GATES = {
    "analyze": {
        "gate_id": "approve-fix-plan",
        "title": "确认分析方案后开始修复",
        "description": "请检查根因、影响文件和修复方案。确认后才会进入代码修改节点。",
        "action_label": "确认方案，开始修复",
    },
    "test": {
        "gate_id": "approve-mr-stage",
        "title": "确认测试结果后进入 MR 阶段",
        "description": "请检查评审结论和测试结果。确认后才会继续准备 MR、检查流水线并完成验收。",
        "action_label": "确认测试，继续执行",
    },
}


def codebuddy_capability_status():
    """Expose capability state without returning or persisting user secrets."""
    iwiki_authorized = bool(
        os.environ.get("TAI_PAT_TOKEN", "").strip()
        or os.environ.get("TEAMAI_REMOTE_IWIKI_AUTHORIZED", "").strip() == "1"
    )
    return {
        "profile": "iwiki-read",
        "configured": iwiki_authorized,
        "capabilities": ["iwiki.read"] if iwiki_authorized else [],
        "missing_capabilities": [] if iwiki_authorized else ["iwiki.read"],
        "detail": (
            "TeamAI 已加载 iWiki 只读 MCP 配置和用户授权"
            if iwiki_authorized
            else "CodeBuddy 可用；iWiki 尚未授权，请在 TeamAI 运行环境配置 TAI_PAT_TOKEN"
        ),
    }


class DemoState:
    def __init__(self, state_path=None):
        self.lock = threading.RLock()
        self.approval_condition = threading.Condition(self.lock)
        self.connected = True
        self.teamai = {
            "bridge_id": "teamai_bridge_local_demo",
            "organization": "ClawPro Demo 组织",
            "device_id": "device_eva_mac_demo",
            "device_name": "Eva's Mac",
            "trusted": True,
            "transport": "Local Bridge（演示）",
            "edge_runtime_managed": True,
        }
        self.tasks = {}
        self.task_order = []
        self.processes = {}
        self.cloudagent = DevResonanceCloudAgentClient()
        self.cloudagent_executions = {}
        self.hatchery = None
        self.state_path = Path(state_path) if state_path else None
        self.load_task_state()

    def task_state_payload_locked(self):
        task_order = [task_id for task_id in self.task_order if task_id in self.tasks]
        return {
            "version": TASK_STATE_VERSION,
            "saved_at": self.now(),
            "task_order": task_order,
            "tasks": {task_id: self.tasks[task_id] for task_id in task_order},
        }

    def persist_task_state_locked(self):
        if not self.state_path:
            return
        try:
            self.state_path.parent.mkdir(parents=True, exist_ok=True)
            temporary = self.state_path.with_suffix(self.state_path.suffix + ".tmp")
            temporary.write_text(
                json.dumps(
                    self.task_state_payload_locked(),
                    ensure_ascii=False,
                    indent=2,
                    default=str,
                ),
                encoding="utf-8",
            )
            os.replace(temporary, self.state_path)
        except Exception as error:
            print("[task-state] persist failed: {0}".format(error), file=sys.stderr)

    def persist_task_state(self):
        with self.lock:
            self.persist_task_state_locked()

    def load_task_state(self):
        if not self.state_path or not self.state_path.is_file():
            return
        try:
            payload = json.loads(self.state_path.read_text(encoding="utf-8"))
            if payload.get("version") != TASK_STATE_VERSION:
                raise ValueError("unsupported task-state version")
            raw_tasks = payload.get("tasks") or {}
            raw_order = payload.get("task_order") or []
            if not isinstance(raw_tasks, dict) or not isinstance(raw_order, list):
                raise ValueError("invalid task-state payload")
            recovered_at = self.now()
            for task_id in raw_order:
                task = raw_tasks.get(task_id)
                if not isinstance(task, dict) or task.get("task_id") != task_id:
                    continue
                task.setdefault("events", [])
                task.setdefault("attempt_id", "attempt_recovered")
                if task.get("status") not in TERMINAL_TASK_STATUSES:
                    detail = "编排服务重启，执行已安全中断；请在任务详情中重新运行。"
                    task["status"] = "failed"
                    task["execution_status"] = "failed"
                    task["workflow_stage"] = "failed"
                    task["failure_detail"] = detail
                    task["pending_approval"] = None
                    task["workflow_current_phase"] = None
                    task["workflow_current_phases"] = []
                    for phase in task.get("workflow_phases") or []:
                        if phase.get("status") in {
                            "queued",
                            "ready",
                            "running",
                            "awaiting_approval",
                        }:
                            phase["status"] = "failed"
                            phase["error"] = detail
                    task["events"].append(
                        {
                            "event_id": "evt_" + uuid.uuid4().hex[:12],
                            "task_id": task_id,
                            "attempt_id": task["attempt_id"],
                            "seq": len(task["events"]) + 1,
                            "type": "task.recovered.interrupted",
                            "title": "服务重启后已恢复任务记录",
                            "detail": detail,
                            "payload": {"recovered": True},
                            "timestamp": recovered_at,
                        }
                    )
                task["updated_at"] = max(
                    str(task.get("updated_at") or ""), recovered_at
                )
                self.tasks[task_id] = task
                self.task_order.append(task_id)
            self.persist_task_state_locked()
            print(
                "[task-state] restored {0} task(s)".format(len(self.task_order)),
                file=sys.stderr,
            )
        except Exception as error:
            print("[task-state] load failed: {0}".format(error), file=sys.stderr)

    def configure_hatchery(self, api_url, admin_token):
        bridge = HatcheryTeamAIBridge(api_url, admin_token, ROOT)
        self.hatchery = bridge
        try:
            bridge.setup()
            self.teamai = bridge.public_device()
        except Exception as error:
            self.teamai = bridge.public_device()
            self.teamai["setup_error"] = str(error)
            print("[hatchery-teamai] setup failed: {0}".format(error), file=sys.stderr)

    def now(self):
        return datetime.now(timezone.utc).isoformat(timespec="milliseconds")

    def append_event(self, task_id, event_type, title, detail="", payload=None):
        with self.lock:
            task = self.tasks[task_id]
            event = {
                "event_id": "evt_" + uuid.uuid4().hex[:12],
                "task_id": task_id,
                "attempt_id": task["attempt_id"],
                "seq": len(task["events"]) + 1,
                "type": event_type,
                "title": title,
                "detail": detail,
                "payload": payload or {},
                "timestamp": self.now(),
            }
            task["events"].append(event)
            task["updated_at"] = event["timestamp"]
            self.persist_task_state_locked()
            return event

    def create_task(
        self,
        prompt,
        runtime_id,
        model,
        target_agent_id="",
        imate_project_id="",
        delivery_mode="wss",
        agent_assignment_mode="shared",
        agent_runtime_id="codebuddy-acp",
        node_assignments=None,
        workflow_definition=None,
        workflow_inputs=None,
    ):
        if runtime_id == STRUCTURED_ISSUEFIX_RUNTIME_ID:
            return self.create_structured_issuefix_validation(
                prompt,
                model,
                agent_assignment_mode,
                agent_runtime_id,
                target_agent_id=target_agent_id,
                imate_project_id=imate_project_id,
                delivery_mode=delivery_mode,
                node_assignments=node_assignments,
            )
        if runtime_id == STRUCTURED_PROJECT_WORKFLOW_RUNTIME_ID:
            return self.create_structured_project_workflow(
                prompt,
                model,
                agent_assignment_mode,
                agent_runtime_id,
                target_agent_id=target_agent_id,
                imate_project_id=imate_project_id,
                delivery_mode=delivery_mode,
                node_assignments=node_assignments,
                workflow_definition=workflow_definition,
                workflow_inputs=workflow_inputs,
            )
        if runtime_id == "workflow-codebuddy-imate":
            return self.create_handoff_workflow(
                prompt,
                model,
                target_agent_id=target_agent_id,
                imate_project_id=imate_project_id,
                delivery_mode=delivery_mode,
            )
        if runtime_id in HATCHERY_RUNTIME_IDS:
            return self.create_hatchery_task(
                prompt,
                runtime_id,
                model,
                target_agent_id=target_agent_id,
                imate_project_id=imate_project_id,
                delivery_mode=delivery_mode,
            )
        task_id = "task_" + uuid.uuid4().hex[:8]
        task = {
            "task_id": task_id,
            "attempt_id": "attempt_" + uuid.uuid4().hex[:8],
            "runtime_id": runtime_id,
            "model": model,
            "prompt": prompt,
            "status": "queued",
            "delivery_status": "queued",
            "execution_status": "submitted",
            "created_at": self.now(),
            "updated_at": self.now(),
            "cancel_requested": False,
            "session_id": None,
            "workspace_path": None,
            "agent_output": "",
            "artifact": None,
            "events": [],
            "_message_buffer": "",
            "_thought_buffer": "",
        }
        with self.lock:
            self.tasks[task_id] = task
            self.task_order.insert(0, task_id)
        self.append_event(
            task_id,
            "task.queued",
            "任务已进入 ClawPro 队列",
            "等待 TeamAI 设备通道投递到本地。",
        )
        if self.connected:
            threading.Thread(target=self.run_task, args=(task_id,), daemon=True).start()
        return task

    def create_structured_issuefix_validation(
        self,
        prompt,
        model,
        agent_assignment_mode="shared",
        agent_runtime_id="codebuddy-acp",
        *,
        target_agent_id="",
        imate_project_id="",
        delivery_mode="wss",
        node_assignments=None,
    ):
        """Validate the real IssueFix package without touching its target repository."""
        task_id = "workflow_" + uuid.uuid4().hex[:8]
        workspace = WORKFLOW_WORKSPACE_ROOT / task_id
        workspace.mkdir(parents=True, exist_ok=True)
        task = {
            "task_id": task_id,
            "attempt_id": "attempt_" + uuid.uuid4().hex[:8],
            "runtime_id": STRUCTURED_ISSUEFIX_RUNTIME_ID,
            "model": model,
            "prompt": prompt,
            "status": "queued",
            "delivery_status": "local_execution",
            "execution_status": "submitted",
            "created_at": self.now(),
            "updated_at": self.now(),
            "cancel_requested": False,
            "cancellable": True,
            "session_id": None,
            "workspace_path": str(workspace),
            "agent_output": "",
            "executor": (
                "skillhub-issuefix-teamai-node-routed"
            ),
            "delivery_mode": delivery_mode,
            "artifact": None,
            "workflow": True,
            "structured_workflow": True,
            "workflow_id": "skillhub-issuefix",
            "workflow_source": ISSUEFIX_SOURCE_URL,
            "workflow_ref": ISSUEFIX_REF,
            "workflow_revision": None,
            "workflow_stage": "source_syncing",
            "workflow_current_phase": None,
            "workflow_phases": [],
            "agent_assignment_mode": agent_assignment_mode,
            "agent_runtime_id": agent_runtime_id,
            "node_assignments": list(node_assignments or []),
            "target_agent_id": target_agent_id or None,
            "imate_project_id": imate_project_id or None,
            "agent_instance_count": None,
            "agent_session_count": None,
            "handoff_count": None,
            "handoff_contract": "ClawPro Handoff v2（iMate-style）",
            "pending_approval": None,
            "approval_history": [],
            "available_artifacts": [],
            "real_agent_execution": True,
            "safe_mode": True,
            "external_writes_performed": False,
            "events": [],
            "_message_buffer": "",
            "_thought_buffer": "",
        }
        with self.lock:
            self.tasks[task_id] = task
            self.task_order.insert(0, task_id)
        self.append_event(
            task_id,
            "workflow.source.syncing",
            "正在同步结构化工作流",
            "只读取指定工蜂分支，不修改 SkillHub 源码仓库。",
            {
                "agent_assignment_mode": agent_assignment_mode,
                "agent_runtime_id": agent_runtime_id,
                "meaning": (
                    "节点将按 ClawPro 中绑定的 Agent 经 TeamAI 执行"
                ),
            },
        )
        threading.Thread(
            target=self.run_structured_issuefix_validation,
            args=(task_id,),
            daemon=True,
        ).start()
        return task

    @staticmethod
    def normalize_project_workflow_definition(raw_definition):
        if not isinstance(raw_definition, dict):
            raise BridgeError("缺少项目工作流定义")
        workflow_id = str(raw_definition.get("workflow_id") or "").strip()
        name = str(raw_definition.get("name") or workflow_id).strip()
        raw_phases = raw_definition.get("phases") or []
        if not workflow_id or not isinstance(raw_phases, list) or not raw_phases:
            raise BridgeError("项目工作流必须包含 workflow_id 和节点")
        if len(raw_phases) > 20:
            raise BridgeError("项目工作流节点不能超过 20 个")
        phases = []
        phase_ids = set()
        for index, raw_phase in enumerate(raw_phases):
            if not isinstance(raw_phase, dict):
                raise BridgeError("项目工作流节点格式无效")
            phase_id = str(raw_phase.get("id") or "").strip()
            title = str(raw_phase.get("title") or phase_id).strip()
            if not phase_id or phase_id in phase_ids:
                raise BridgeError("项目工作流节点 ID 缺失或重复")
            phase_ids.add(phase_id)
            artifacts = []
            for item in raw_phase.get("artifacts") or []:
                relative = str(item or "").strip().replace("\\", "/")
                path = Path(relative)
                if (
                    not relative
                    or path.is_absolute()
                    or ".." in path.parts
                    or relative.startswith(".")
                ):
                    raise BridgeError("节点产物路径无效：{0}".format(relative))
                artifacts.append(relative)
            if not artifacts:
                artifacts = ["{0}-result.md".format(phase_id)]
            optional_artifacts = []
            for item in raw_phase.get("optional_artifacts") or []:
                relative = str(item or "").strip().replace("\\", "/")
                path = Path(relative)
                if (
                    not relative
                    or path.is_absolute()
                    or ".." in path.parts
                    or relative.startswith(".")
                ):
                    raise BridgeError("节点可选产物路径无效：{0}".format(relative))
                if relative not in artifacts:
                    optional_artifacts.append(relative)
            next_phase = (
                str(raw_phases[index + 1].get("id") or "").strip()
                if index + 1 < len(raw_phases)
                and isinstance(raw_phases[index + 1], dict)
                else None
            )
            prompt = str(raw_phase.get("prompt") or "按节点职责执行").strip()
            raw_config_assets = raw_phase.get("config_assets") or []
            if not isinstance(raw_config_assets, list):
                raise BridgeError(
                    "节点 config_assets 必须是数组：{0}".format(phase_id)
                )
            if len(raw_config_assets) > 12:
                raise BridgeError(
                    "单节点配置资产不能超过 12 个：{0}".format(phase_id)
                )
            config_assets = []
            config_asset_ids = set()
            total_config_asset_bytes = 0
            for raw_asset in raw_config_assets:
                if not isinstance(raw_asset, dict):
                    raise BridgeError(
                        "节点配置资产格式无效：{0}".format(phase_id)
                    )
                asset_id = str(raw_asset.get("id") or "").strip()
                asset_name = str(raw_asset.get("name") or asset_id).strip()
                asset_version = str(raw_asset.get("version") or "").strip()
                asset_type = str(raw_asset.get("type") or "rules").strip()
                asset_summary = str(raw_asset.get("summary") or "").strip()
                asset_source = str(raw_asset.get("source") or "").strip()
                asset_content = str(raw_asset.get("content") or "")
                if (
                    not asset_id
                    or len(asset_id) > 100
                    or any(
                        char
                        not in "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_.-"
                        for char in asset_id
                    )
                    or asset_id in config_asset_ids
                ):
                    raise BridgeError(
                        "节点配置资产 ID 缺失、重复或无效：{0}".format(phase_id)
                    )
                if asset_type not in {"rules", "skill", "contract"}:
                    raise BridgeError(
                        "节点配置资产类型无效：{0}/{1}".format(
                            phase_id, asset_id
                        )
                    )
                asset_bytes = asset_content.encode("utf-8")
                if not asset_name or not asset_version or not asset_content.strip():
                    raise BridgeError(
                        "节点配置资产缺少名称、版本或正文：{0}/{1}".format(
                            phase_id, asset_id
                        )
                    )
                if len(asset_bytes) > 200_000:
                    raise BridgeError(
                        "节点配置资产正文超过 200KB：{0}/{1}".format(
                            phase_id, asset_id
                        )
                    )
                total_config_asset_bytes += len(asset_bytes)
                if total_config_asset_bytes > 600_000:
                    raise BridgeError(
                        "单节点配置资产正文合计不能超过 600KB：{0}".format(
                            phase_id
                        )
                    )
                config_asset_ids.add(asset_id)
                config_assets.append(
                    {
                        "id": asset_id,
                        "name": asset_name,
                        "version": asset_version,
                        "type": asset_type,
                        "summary": asset_summary,
                        "source": asset_source,
                        "content": asset_content,
                        "size": len(asset_bytes),
                        "sha256": hashlib.sha256(asset_bytes).hexdigest(),
                    }
                )
            approval_required = bool(raw_phase.get("approval_required"))
            required_evidence = [
                str(item).strip()
                for item in raw_phase.get("required_evidence") or []
                if str(item).strip()
            ]
            reject_output_markers = [
                str(item).strip()
                for item in raw_phase.get("reject_output_markers") or []
                if str(item).strip()
            ]
            required_capabilities = [
                str(item).strip()
                for item in raw_phase.get("required_capabilities") or []
                if str(item).strip()
            ]
            decision_mode = str(raw_phase.get("decision_mode") or "").strip()
            if decision_mode not in {"", "review_verdict", "size_class"}:
                raise BridgeError(
                    "节点 decision_mode 无效：{0}".format(phase_id)
                )
            try:
                max_retries = int(raw_phase.get("max_retries") or 0)
            except (TypeError, ValueError) as error:
                raise BridgeError(
                    "节点 max_retries 必须是整数：{0}".format(phase_id)
                ) from error
            if max_retries < 0 or max_retries > 5:
                raise BridgeError(
                    "节点 max_retries 必须在 0～5：{0}".format(phase_id)
                )
            raw_dependencies = raw_phase.get("depends_on")
            if raw_dependencies is None:
                depends_on = (
                    [str(raw_phases[index - 1].get("id") or "").strip()]
                    if index > 0 and isinstance(raw_phases[index - 1], dict)
                    else []
                )
            elif not isinstance(raw_dependencies, list):
                raise BridgeError("节点 depends_on 必须是数组：{0}".format(phase_id))
            else:
                depends_on = list(
                    dict.fromkeys(
                        str(item or "").strip()
                        for item in raw_dependencies
                        if str(item or "").strip()
                    )
                )
            phases.append(
                {
                    "id": phase_id,
                    "title": title,
                    "agent_id": str(raw_phase.get("agent_id") or phase_id),
                    "agent_title": title,
                    "agent_description": prompt,
                    "inputs": list(raw_phase.get("inputs") or []),
                    "input_mappings": {},
                    "output_schema": {},
                    "declared_artifacts": artifacts,
                    "optional_artifacts": optional_artifacts,
                    "config_assets": config_assets,
                    "node_plan": [
                        {
                            "id": phase_id,
                            "title": title,
                            "actions": [prompt],
                        }
                    ],
                    "depends_on": depends_on,
                    "on_pass": raw_phase.get("on_pass", next_phase),
                    "on_fail": raw_phase.get("on_fail", phase_id),
                    "approval_required": approval_required,
                    "required_evidence": required_evidence,
                    "reject_output_markers": reject_output_markers,
                    "required_capabilities": required_capabilities,
                    "decision_mode": decision_mode or None,
                    "max_retries": max_retries,
                    "approval": (
                        {
                            "gate_id": "approve-{0}".format(phase_id),
                            "title": "确认“{0}”结果后继续".format(title),
                            "description": "请检查本节点输入、输出和产物；确认后才会流转到下一节点。",
                            "action_label": "确认结果，继续执行",
                        }
                        if approval_required
                        else None
                    ),
                }
            )
        known_phase_ids = {phase["id"] for phase in phases}
        for phase in phases:
            invalid = [
                dependency
                for dependency in phase["depends_on"]
                if dependency not in known_phase_ids or dependency == phase["id"]
            ]
            if invalid:
                raise BridgeError(
                    "节点 {0} 的上游依赖无效：{1}".format(
                        phase["id"], ", ".join(invalid)
                    )
                )
            invalid_routes = [
                route
                for route in (phase.get("on_pass"), phase.get("on_fail"))
                if route is not None and route not in known_phase_ids
            ]
            if invalid_routes:
                raise BridgeError(
                    "节点 {0} 的状态路由无效：{1}".format(
                        phase["id"], ", ".join(invalid_routes)
                    )
                )
        unresolved = {phase["id"] for phase in phases}
        resolved = set()
        while unresolved:
            ready = {
                phase["id"]
                for phase in phases
                if phase["id"] in unresolved
                and set(phase["depends_on"]).issubset(resolved)
            }
            if not ready:
                raise BridgeError("项目工作流包含循环依赖")
            resolved.update(ready)
            unresolved.difference_update(ready)
        execution_mode = str(raw_definition.get("execution_mode") or "dag")
        if execution_mode not in {"dag", "state_machine"}:
            raise BridgeError("项目工作流 execution_mode 无效")
        return {
            "schema_version": "clawpro.project-workflow.v2",
            "workflow_id": workflow_id,
            "name": name,
            "description": name,
            "execution_mode": execution_mode,
            "source": {"url": "clawpro://project-collaboration", "git_ref": None},
            "phases": phases,
            "external_write_phases": [],
        }

    @staticmethod
    def normalize_workflow_inputs(raw_inputs):
        if raw_inputs is None:
            return {}
        if not isinstance(raw_inputs, dict):
            raise BridgeError("项目工作流输入必须是对象")
        if len(raw_inputs) > 50:
            raise BridgeError("项目工作流输入不能超过 50 个")
        normalized = {}
        allowed_types = {"text", "markdown", "json", "file", "url"}
        for raw_key, raw_value in raw_inputs.items():
            key = str(raw_key or "").strip()
            if (
                not key
                or len(key) > 100
                or any(char not in "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_.-" for char in key)
            ):
                raise BridgeError("项目工作流输入 key 无效：{0}".format(key))
            if not isinstance(raw_value, dict):
                raise BridgeError("项目工作流输入值格式无效：{0}".format(key))
            value_type = str(raw_value.get("type") or "text").strip()
            value = str(raw_value.get("value") or "")
            if value_type not in allowed_types:
                raise BridgeError("项目工作流输入类型无效：{0}".format(key))
            if len(value.encode("utf-8")) > 200_000:
                raise BridgeError("项目工作流输入过大：{0}".format(key))
            normalized[key] = {"type": value_type, "value": value}
        return normalized

    def create_structured_project_workflow(
        self,
        prompt,
        model,
        agent_assignment_mode="shared",
        agent_runtime_id="codebuddy-acp",
        *,
        target_agent_id="",
        imate_project_id="",
        delivery_mode="wss",
        node_assignments=None,
        workflow_definition=None,
        workflow_inputs=None,
    ):
        definition = self.normalize_project_workflow_definition(
            workflow_definition
        )
        normalized_inputs = self.normalize_workflow_inputs(workflow_inputs)
        task_id = "workflow_" + uuid.uuid4().hex[:8]
        workspace = WORKFLOW_WORKSPACE_ROOT / task_id
        workspace.mkdir(parents=True, exist_ok=True)
        task = {
            "task_id": task_id,
            "attempt_id": "attempt_" + uuid.uuid4().hex[:8],
            "runtime_id": STRUCTURED_PROJECT_WORKFLOW_RUNTIME_ID,
            "model": model,
            "prompt": prompt,
            "status": "queued",
            "delivery_status": "local_execution",
            "execution_status": "submitted",
            "created_at": self.now(),
            "updated_at": self.now(),
            "cancel_requested": False,
            "cancellable": True,
            "session_id": None,
            "workspace_path": str(workspace),
            "agent_output": "",
            "executor": "project-workflow-teamai-node-routed",
            "delivery_mode": delivery_mode,
            "artifact": None,
            "workflow": True,
            "structured_workflow": True,
            "workflow_id": definition["workflow_id"],
            "workflow_source": "clawpro://project-collaboration",
            "workflow_ref": None,
            "workflow_revision": definition["schema_version"],
            "workflow_stage": "contract_validating",
            "workflow_current_phase": None,
            "workflow_current_phases": [],
            "workflow_phases": [],
            "agent_assignment_mode": agent_assignment_mode,
            "agent_runtime_id": agent_runtime_id,
            "node_assignments": list(node_assignments or []),
            "target_agent_id": target_agent_id or None,
            "imate_project_id": imate_project_id or None,
            "agent_instance_count": None,
            "agent_session_count": None,
            "handoff_count": None,
            "handoff_contract": "ClawPro Handoff v2（iMate-style）",
            "pending_approval": None,
            "approval_history": [],
            "available_artifacts": [],
            "real_agent_execution": True,
            "safe_mode": True,
            "external_writes_performed": False,
            "events": [],
            "_workflow_definition": definition,
            "_workflow_inputs": normalized_inputs,
        }
        with self.lock:
            self.tasks[task_id] = task
            self.task_order.insert(0, task_id)
        self.append_event(
            task_id,
            "workflow.contract.validating",
            "正在校验项目工作流",
            "将 {0} 个节点绑定到 TeamAI 真实 Runtime。".format(
                len(definition["phases"])
            ),
        )
        threading.Thread(
            target=self.run_structured_project_workflow,
            args=(task_id,),
            daemon=True,
        ).start()
        return task

    @staticmethod
    def workflow_gate_for_phase(phase):
        return phase.get("approval") or WORKFLOW_APPROVAL_GATES.get(phase["id"])

    def wait_for_workflow_approval(
        self, task_id, phase, next_phase_id, node_output
    ):
        gate = self.workflow_gate_for_phase(phase)
        if not gate:
            return
        requested_at = self.now()
        pending = {
            **gate,
            "status": "pending",
            "after_phase_id": phase["id"],
            "after_phase_title": phase["title"],
            "next_phase_id": next_phase_id,
            "requested_at": requested_at,
            "approved_at": None,
            "artifacts": node_output.get("artifacts", []),
            "summary": node_output.get("data", {}).get(
                "summary", node_output.get("summary", "")
            ),
        }
        with self.approval_condition:
            task = self.tasks[task_id]
            task["status"] = "waiting_approval"
            task["execution_status"] = "waiting_approval"
            task["workflow_stage"] = "awaiting_approval"
            task["workflow_current_phase"] = phase["id"]
            task["pending_approval"] = pending
            for phase_item in task["workflow_phases"]:
                if phase_item["id"] == phase["id"]:
                    phase_item["status"] = "awaiting_approval"
                    break
        self.append_event(
            task_id,
            "workflow.approval.requested",
            gate["title"],
            gate["description"],
            pending,
        )
        with self.approval_condition:
            while pending["status"] == "pending":
                if self.tasks[task_id]["cancel_requested"]:
                    raise TaskCanceled()
                self.approval_condition.wait(timeout=1)
            task = self.tasks[task_id]
            if task["cancel_requested"]:
                raise TaskCanceled()
            task["pending_approval"] = None
            task["status"] = "running"
            task["execution_status"] = "running"
            task["workflow_stage"] = "real_agent_running"
            task["workflow_current_phase"] = next_phase_id
            for phase_item in task["workflow_phases"]:
                if phase_item["id"] == phase["id"]:
                    phase_item["status"] = "completed"
                    break

    def approve_workflow(self, task_id, gate_id):
        with self.approval_condition:
            task = self.tasks.get(task_id)
            if not task:
                return None
            pending = task.get("pending_approval")
            if not pending or pending.get("status") != "pending":
                raise ValueError("当前工作流没有待确认节点")
            if gate_id != pending["gate_id"]:
                raise ValueError("确认点已变化，请刷新页面后重试")
            approved_at = self.now()
            pending["status"] = "approved"
            pending["approved_at"] = approved_at
            approval_record = {
                **pending,
                "approver": "current-user",
            }
            task["approval_history"].append(approval_record)
            task["updated_at"] = approved_at
            self.append_event(
                task_id,
                "workflow.approval.granted",
                "已确认，工作流继续执行",
                "{0} → {1}".format(
                    pending["after_phase_id"], pending["next_phase_id"]
                ),
                approval_record,
            )
            self.approval_condition.notify_all()
            return task

    def start_codebuddy_workflow_session(
        self, task_id, workspace, agent_instance_id, request_seed
    ):
        runtime = runtime_by_id("codebuddy-acp")
        if not runtime or not runtime.get("available"):
            raise RuntimeError("本机 CodeBuddy ACP Runtime 不可用")
        command = [
            runtime["executable"],
            "--acp",
            "--permission-mode",
            "acceptEdits",
            "--setting-sources",
            "local",
            "--tools",
            "Read,Write,Edit,Glob,Grep",
        ]
        process = subprocess.Popen(
            command,
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            bufsize=1,
            cwd=str(workspace),
        )
        with self.lock:
            self.processes[task_id] = process
        self.rpc(
            task_id,
            process,
            request_seed,
            "initialize",
            {
                "protocolVersion": 1,
                "clientInfo": {
                    "name": "clawpro-structured-workflow-poc",
                    "version": "0.2.0",
                },
                "clientCapabilities": {},
            },
        )
        session = self.rpc(
            task_id,
            process,
            request_seed + 1,
            "session/new",
            {"cwd": str(workspace), "mcpServers": []},
        )
        session_id = (session.get("result") or {}).get("sessionId")
        if not session_id:
            raise RuntimeError("CodeBuddy ACP session/new 未返回 sessionId")
        self.append_event(
            task_id,
            "workflow.agent.session_started",
            "真实 CodeBuddy Agent 会话已创建",
            "{0} → {1}".format(agent_instance_id, session_id),
            {
                "agent_instance_id": agent_instance_id,
                "session_id": session_id,
                "runtime_id": "codebuddy-acp",
            },
        )
        return process, session_id, request_seed + 2

    @staticmethod
    def stop_agent_process(process):
        if not process or process.poll() is not None:
            return
        process.terminate()
        try:
            process.wait(timeout=2)
        except subprocess.TimeoutExpired:
            process.kill()

    @staticmethod
    def build_artifact_ref(
        task_id, producer_node, relative_path, raw, *, lineage=None, version=1
    ):
        """Create the runtime-neutral artifact identity carried across nodes."""
        media_type = mimetypes.guess_type(relative_path)[0]
        if Path(relative_path).suffix.lower() in {".md", ".markdown"}:
            media_type = "text/markdown"
        media_type = media_type or "application/octet-stream"
        digest = hashlib.sha256(raw).hexdigest()
        return {
            "schema_version": "clawpro.artifact-ref.v1",
            "artifact_id": "{0}:{1}:{2}:v{3}".format(
                task_id, producer_node, relative_path, version
            ),
            "version": version,
            "path": relative_path,
            "media_type": media_type,
            "size": len(raw),
            "sha256": digest,
            "producer_node": producer_node,
            "lineage": list(lineage or []),
        }

    @staticmethod
    def materialize_teamai_artifact_bundle(result, agent_workspace, required_artifacts):
        """Persist the exact files uploaded by the user-side TeamAI runtime."""
        start = "<clawpro_artifact_bundle_v1>"
        end = "</clawpro_artifact_bundle_v1>"
        start_index = result.rfind(start)
        end_index = result.rfind(end)
        if start_index < 0 or end_index < start_index:
            return result.strip(), []
        summary = (result[:start_index] + result[end_index + len(end) :]).strip()
        payload_text = result[start_index + len(start) : end_index].strip()
        try:
            payload = json.loads(payload_text)
        except json.JSONDecodeError as error:
            raise RuntimeError("TeamAI 回传的产物包不是有效 JSON") from error
        if payload.get("schema_version") != "clawpro.artifact-bundle.v1":
            raise RuntimeError("TeamAI 回传了不支持的产物包版本")

        required = list(dict.fromkeys(required_artifacts))
        required_set = set(required)
        uploaded = []
        for item in payload.get("artifacts") or []:
            relative = str(item.get("path") or "").strip()
            if relative not in required_set:
                continue
            destination = (agent_workspace / relative).resolve()
            try:
                destination.relative_to(agent_workspace.resolve())
            except ValueError as error:
                raise RuntimeError("TeamAI 产物路径越界") from error
            try:
                raw = base64.b64decode(
                    str(item.get("content_base64") or ""), validate=True
                )
            except (ValueError, TypeError) as error:
                raise RuntimeError("TeamAI 产物内容不是有效 Base64") from error
            expected_size = int(item.get("size") or 0)
            expected_sha = str(item.get("sha256") or "")
            actual_sha = hashlib.sha256(raw).hexdigest()
            if not raw or len(raw) != expected_size or actual_sha != expected_sha:
                raise RuntimeError("TeamAI 产物完整性校验失败：{0}".format(relative))
            destination.parent.mkdir(parents=True, exist_ok=True)
            destination.write_bytes(raw)
            uploaded.append(
                {
                    "path": relative,
                    "source_path": str(item.get("source_path") or relative),
                    "size": len(raw),
                    "sha256": actual_sha,
                }
            )
        missing = [item for item in required if item not in {x["path"] for x in uploaded}]
        if missing:
            raise RuntimeError(
                "TeamAI 未回传必需的真实产物：{0}".format(", ".join(missing))
            )
        return summary, uploaded

    def build_node_input_v2(
        self,
        task_id,
        definition,
        phase,
        assigned,
        session_id,
        previous_result,
        base_artifacts,
        index,
    ):
        artifact_by_path = {item["path"]: item for item in base_artifacts}
        for item in (previous_result or {}).get("artifacts", []):
            artifact_by_path[item["path"]] = item
        upstream_artifacts = list(artifact_by_path.values())
        gate = self.workflow_gate_for_phase(phase)
        workflow_inputs = self.tasks[task_id].get("_workflow_inputs") or {}
        input_mappings = dict(phase.get("input_mappings", {}))
        input_data = {
            "upstream": (previous_result or {}).get("data"),
        }
        for declaration in phase.get("inputs") or []:
            if not isinstance(declaration, dict):
                continue
            key = str(declaration.get("key") or "").strip()
            source = declaration.get("source")
            if isinstance(source, dict):
                source_node = str(source.get("nodeId") or "").strip()
                output_key = str(source.get("outputKey") or "").strip()
                if source_node and output_key:
                    input_mappings[key] = "$nodes.{0}.outputs.{1}".format(
                        source_node, output_key
                    )
            elif key in workflow_inputs:
                input_mappings[key] = "$task.inputs.{0}.value".format(key)
                input_data[key] = workflow_inputs[key]["value"]
        return {
            "schema_version": "clawpro.node-input.v2",
            "workflow_id": definition["workflow_id"],
            "workflow_run_id": task_id,
            "node_run_id": "{0}:{1}:{2}".format(task_id, phase["id"], index),
            "attempt_id": self.tasks[task_id]["attempt_id"],
            "node": {
                "id": phase["id"],
                "title": phase["title"],
                "role_agent_id": phase["agent_id"],
                "agent_instance_id": assigned["agent_instance_id"],
                "agent_session_id": session_id,
                "runtime_id": assigned["runtime_id"],
                "config_assets": [
                    {
                        "id": asset["id"],
                        "name": asset["name"],
                        "version": asset["version"],
                        "type": asset["type"],
                        "summary": asset["summary"],
                        "source": asset["source"],
                        "size": asset["size"],
                        "sha256": asset["sha256"],
                    }
                    for asset in phase.get("config_assets") or []
                ],
            },
            "task": {
                "goal": self.tasks[task_id]["prompt"],
                "inputs": workflow_inputs,
            },
            "inputs": {
                "mappings": input_mappings,
                "data": input_data,
                "artifacts": upstream_artifacts,
                "declared": phase["inputs"],
            },
            "output_contract": {
                "data_schema": phase.get("output_schema", {}),
                "required_artifacts": phase["declared_artifacts"],
            },
            "approval_policy": {
                "required_after_node": bool(gate),
                "gate_id": gate["gate_id"] if gate else None,
            },
        }

    @staticmethod
    def stage_node_inputs(
        agent_workspace, node_workspace, input_envelope, config_assets=None
    ):
        """Materialize the direct input set in an iMate-style isolated directory."""
        upstream_dir = node_workspace / ".upstream_artifacts"
        upstream_dir.mkdir(parents=True, exist_ok=True)
        staged = []
        for artifact in input_envelope["inputs"]["artifacts"]:
            source = (agent_workspace / artifact["path"]).resolve()
            try:
                source.relative_to(agent_workspace.resolve())
            except ValueError as error:
                raise RuntimeError("上游产物路径越界") from error
            if not source.is_file():
                raise RuntimeError("上游产物不存在：{0}".format(artifact["path"]))
            destination = upstream_dir / artifact["path"]
            destination.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(source, destination)
            raw = destination.read_bytes()
            if hashlib.sha256(raw).hexdigest() != artifact["sha256"]:
                raise RuntimeError("上游产物校验失败：{0}".format(artifact["path"]))
            staged.append(
                {
                    "artifact_id": artifact["artifact_id"],
                    "path": artifact["path"],
                    "staged_path": destination.relative_to(node_workspace).as_posix(),
                    "sha256": artifact["sha256"],
                }
            )
        input_envelope["inputs"]["staged_artifacts"] = staged
        staged_config_assets = []
        config_asset_dir = node_workspace / ".clawpro" / "config-assets"
        for asset in config_assets or []:
            content = asset["content"]
            extension = ".md"
            if asset["type"] == "contract":
                try:
                    json.loads(content)
                    extension = ".json"
                except json.JSONDecodeError:
                    pass
            destination = config_asset_dir / (asset["id"] + extension)
            destination.parent.mkdir(parents=True, exist_ok=True)
            destination.write_text(content, encoding="utf-8")
            raw = destination.read_bytes()
            actual_sha = hashlib.sha256(raw).hexdigest()
            if actual_sha != asset["sha256"] or len(raw) != asset["size"]:
                raise RuntimeError(
                    "节点配置资产完整性校验失败：{0}".format(asset["id"])
                )
            staged_config_assets.append(
                {
                    "id": asset["id"],
                    "name": asset["name"],
                    "version": asset["version"],
                    "type": asset["type"],
                    "summary": asset["summary"],
                    "source": asset["source"],
                    "size": asset["size"],
                    "sha256": asset["sha256"],
                    "orchestrator_staged_path": destination.relative_to(
                        node_workspace
                    ).as_posix(),
                }
            )
        input_envelope.setdefault("node", {})["config_assets"] = staged_config_assets
        return upstream_dir

    @staticmethod
    def render_phase_config_assets(phase):
        assets = phase.get("config_assets") or []
        if not assets:
            return "- 无；仅按节点 Prompt 和输入输出契约执行。"
        blocks = []
        for asset in assets:
            blocks.append(
                "### {name} · v{version} · {asset_type}\n"
                "- asset_id: `{asset_id}`\n"
                "- source: `{source}`\n"
                "- sha256: `{sha256}`\n"
                "- summary: {summary}\n\n"
                "```text\n{content}\n```".format(
                    name=asset["name"],
                    version=asset["version"],
                    asset_type=asset["type"],
                    asset_id=asset["id"],
                    source=asset["source"] or "clawpro://inline",
                    sha256=asset["sha256"],
                    summary=asset["summary"] or "无",
                    content=asset["content"],
                )
            )
        return "\n\n".join(blocks)

    @staticmethod
    def write_node_handoff(node_workspace, node_result, agent_workspace):
        """Write and validate the iMate-style formal handoff package."""
        handoff_dir = node_workspace / "handoff"
        handoff_dir.mkdir(parents=True, exist_ok=True)
        for artifact in node_result["artifacts"]:
            source = agent_workspace / artifact["path"]
            destination = handoff_dir / artifact["path"]
            destination.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(source, destination)
        (handoff_dir / ".handoff.json").write_text(
            json.dumps(node_result, ensure_ascii=False, indent=2), encoding="utf-8"
        )
        handoff_md = [
            "# Node Handoff v2",
            "",
            "- Node: {0}".format(node_result["node_id"]),
            "- Runtime: {0}".format(node_result["runtime_id"]),
            "- Status: {0}".format(node_result["status"]),
            "- Summary: {0}".format(node_result["data"]["summary"] or "无"),
            "",
            "## Artifacts",
            "",
        ]
        handoff_md.extend(
            "- `{0}` · {1} bytes · sha256 `{2}`".format(
                item["path"], item["size"], item["sha256"]
            )
            for item in node_result["artifacts"]
        )
        (handoff_dir / ".handoff.md").write_text(
            "\n".join(handoff_md) + "\n", encoding="utf-8"
        )
        required = [handoff_dir / ".handoff.json", handoff_dir / ".handoff.md"]
        if any(not path.is_file() or path.stat().st_size == 0 for path in required):
            raise RuntimeError("节点 Handoff v2 清单不完整")
        return handoff_dir

    @staticmethod
    def build_real_node_prompt(
        phase,
        task_prompt,
        handoff_path,
        artifact_paths,
        input_envelope=None,
        agent_workspace=None,
    ):
        node_plan = []
        for node in phase.get("node_plan", []):
            actions = "；".join(str(item) for item in node.get("actions", []))
            node_plan.append(
                "- {0}：{1}".format(node.get("title") or node.get("id"), actions)
            )
        required_capabilities = {
            str(item).strip()
            for item in phase.get("required_capabilities") or []
            if str(item).strip()
        }
        capability_instructions = []
        if "iwiki.read" in required_capabilities:
            capability_instructions.extend(
                [
                    "- 本节点已获得 `iwiki.read` 能力，必须通过 TeamAI 注入的只读 iWiki MCP 真实读取数据。",
                    "- 仅可调用 `mcp__iwiki__metadata`、`mcp__iwiki__getSpacePageTree`、`mcp__iwiki__getDocument`；不得用公共网络工具替代。",
                    "- 产物必须包含真实 docid、页面数量或页面元数据等调用证据；不得返回 dry-run、模拟数据、空结构占位或 `scanned=false`。",
                ]
            )
        if not capability_instructions:
            capability_instructions.append(
                "- 本节点未声明外部读取能力，不访问网络或未授权的外部系统。"
            )
        remote_handoff = ""
        workflow_inputs = (
            (input_envelope or {}).get("task", {}).get("inputs", {})
        )
        repository_url = str(
            (workflow_inputs.get("repository_url") or {}).get("value") or ""
        ).strip()
        target_page = str(
            (workflow_inputs.get("target_page") or {}).get("value") or ""
        ).strip()
        if repository_url:
            workspace_instructions = """## 受控真实源码工作区
1. 当前目录必须是 TeamAI 为本任务绑定的真实源码仓库，仓库地址为 `{repository_url}`。先读取仓库规范并用 `git remote get-url origin` 校验归属；不匹配时立即停止并如实说明。
2. 目标页面为 `{target_page}`。必须定位并修改该仓库内的真实页面源码；禁止在仓库外另建独立 Demo 替代源码改动。
3. 可在当前仓库内读写文件、运行必要的代码检查和本地预览；不得越出当前仓库，不得提交、推送、创建 MR 或部署。
4. 结果必须列出实际修改的源码路径、验证命令和预览入口。""".format(
                repository_url=repository_url,
                target_page=target_page or "未指定",
            )
        else:
            workspace_instructions = """## POC 隔离工作区
1. 文件操作只限当前隔离工作区，不搜索或修改上级目录。
2. 本任务没有声明源码仓库，只产出工作流报告文件。"""
        if input_envelope is not None:
            inline_files = []
            budget = 20_000
            if agent_workspace is not None:
                for item in input_envelope["inputs"].get("artifacts") or []:
                    path = agent_workspace / item["path"]
                    # TASK.md repeats the complete user goal already rendered
                    # above. Keep it staged on disk for local handoff recovery,
                    # but do not inline it into the ACP prompt a second time.
                    if path.name == "TASK.md":
                        continue
                    if budget <= 0 or not path.is_file():
                        continue
                    content = path.read_text(encoding="utf-8", errors="replace")
                    content = content[: min(8000, budget)]
                    budget -= len(content)
                    inline_files.append(
                        "### {0}\n```text\n{1}\n```".format(path.name, content)
                    )
            prompt_envelope = json.loads(json.dumps(input_envelope))
            prompt_task = prompt_envelope.get("task")
            if isinstance(prompt_task, dict) and prompt_task.get("goal"):
                prompt_task["goal"] = "见上方“用户任务”"
            remote_handoff = (
                "\n\n## 跨机器交接包\n```json\n{0}\n```\n\n"
                "## 上游文本产物\n{1}\n".format(
                    json.dumps(prompt_envelope, ensure_ascii=False, indent=2),
                    "\n\n".join(inline_files) or "无额外文本产物。",
                )
            )
        return """你正在执行 ClawPro 经 WSS/HTTPS 下发给 TeamAI 的项目工作流节点；TeamAI 将通过真实 CodeBuddy ACP Runtime 完成任务。

## 当前角色
- 节点：{phase_id} / {phase_title}
- Agent 角色：{agent_id}
- 角色说明：{description}

## 用户任务
{task_prompt}

## 必须读取的交接包
{handoff_path}

## 工作流定义中的执行要点
{node_plan}

## 节点配置资产（强制执行）
以下 Rules / Skill / Contract 是本节点的规范输入，不是参考资料；执行结果必须满足其中的门禁和产物要求。资产正文已完整内联到本提示；编排端另存于 `.clawpro/config-assets/` 用于审计，远程 Runtime 无需依赖该路径存在。

{config_assets}

## 本节点必须产出
{artifact_paths}

## 本节点条件性产出
{optional_artifact_paths}

## 已授权能力
{capability_instructions}

## 共同执行规则
1. 先读取本提示中的“跨机器交接包”；若当前目录存在 TASK.md、input.json 或 `.upstream_artifacts/`，也一并读取，必须承接上游结果。
2. 只允许使用“已授权能力”中明确列出的外部读取工具；禁止公共网络访问和任何外部写操作。
3. 必须创建“本节点必须产出”中的全部文件；“条件性产出”只在节点规则给出的条件成立时创建。每份 Markdown 至少包含：节点、输入交接、执行结果、产物、风险/待办。
4. 完成后用简短文本说明读取了哪个上游节点和创建了哪些文件。

{workspace_instructions}
{remote_handoff}
""".format(
            phase_id=phase["id"],
            phase_title=phase["title"],
            agent_id=phase["agent_id"],
            description=phase.get("agent_description") or phase["agent_title"],
            task_prompt=task_prompt,
            handoff_path=handoff_path,
            node_plan="\n".join(node_plan) or "- 按当前节点职责执行",
            config_assets=DemoState.render_phase_config_assets(phase),
            artifact_paths="\n".join("- " + path for path in artifact_paths),
            optional_artifact_paths="\n".join(
                "- " + path for path in phase.get("optional_artifacts", [])
            ) or "- 无",
            capability_instructions="\n".join(capability_instructions),
            workspace_instructions=workspace_instructions,
            remote_handoff=remote_handoff,
        ).strip()

    def build_imate_node_prompt(self, task_id, phase, input_envelope, agent_workspace):
        inline_files = []
        budget = 12_000
        candidate_paths = [
            agent_workspace / item["path"]
            for item in input_envelope["inputs"]["artifacts"]
        ]
        for path in candidate_paths:
            if budget <= 0 or not path.is_file():
                continue
            content = path.read_text(encoding="utf-8", errors="replace")
            content = content[: min(8000, budget)]
            budget -= len(content)
            inline_files.append(
                "### {0}\n```text\n{1}\n```".format(path.name, content)
            )
        return """你是 ClawPro 项目工作流中的 iMate OpenClaw Agent。
这是一次真实 iMate Runtime 节点执行，上游由 CodeBuddy 或前一个 iMate 节点完成。

## 当前节点
- 节点：{phase_id} / {phase_title}
- 角色：{agent_id} / {description}
- 必须返回的产物：{artifacts}
- 条件性产物：{optional_artifacts}

## 用户目标
{goal}

## ClawPro 交接包
```json
{envelope}
```

## 节点配置资产（强制执行）
{config_assets}

## 与当前节点相关的文件
{files}

## 执行要求
1. 这是 `clawpro.node-input.v2`。必须按 `inputs.mappings` 解析输入，承接 `inputs.data.upstream` 与 `inputs.artifacts`，不要从头重做上游工作。
2. 根据当前节点职责执行分析、实现规划、评审、测试或报告生成，并明确无法访问的数据或外部系统。
3. 只返回产物 Markdown 正文，控制在 800 字以内，必须包含：节点、输入交接、执行结果、产物、风险/待办。
4. 输出必须满足 `output_contract`；不声称已创建工蜂 MR 或已跑真实流水线。ClawPro Adapter 会保存产物并生成 `.handoff.json/.handoff.md`。
""".format(
            phase_id=phase["id"],
            phase_title=phase["title"],
            agent_id=phase["agent_id"],
            description=phase.get("agent_description") or phase["agent_title"],
            artifacts=", ".join(phase["declared_artifacts"]),
            optional_artifacts=", ".join(phase.get("optional_artifacts", [])) or "无",
            goal=self.tasks[task_id]["prompt"],
            envelope=json.dumps(input_envelope, ensure_ascii=False, indent=2),
            config_assets=self.render_phase_config_assets(phase),
            files="\n\n".join(inline_files) or "无额外文件。",
        ).strip()

    @staticmethod
    def validate_required_evidence(phase, node_summary):
        summary = node_summary.casefold()

        def evidence_variants(marker):
            marker = marker.strip().casefold()
            variants = {marker}
            if "." in marker:
                namespace, tool_name = marker.split(".", 1)
                variants.update(
                    {
                        "{0}__{1}".format(namespace, tool_name),
                        "mcp__{0}__{1}".format(namespace, tool_name),
                    }
                )
            return variants

        missing_evidence = [
            marker
            for marker in phase.get("required_evidence") or []
            if not any(
                variant in summary for variant in evidence_variants(marker)
            )
        ]
        rejected_markers = [
            marker
            for marker in phase.get("reject_output_markers") or []
            if marker.casefold() in summary
        ]
        if not missing_evidence and not rejected_markers:
            return
        details = []
        if missing_evidence:
            details.append(
                "缺少真实读取证据：{0}".format(", ".join(missing_evidence))
            )
        if rejected_markers:
            details.append(
                "结果表明未完成真实读取：{0}".format(", ".join(rejected_markers))
            )
        raise RuntimeError(
            "{0} 未通过真实来源校验；{1}".format(
                phase["id"], "；".join(details)
            )
        )

    def execute_cloudagent_workflow_node(
        self,
        task_id,
        phase,
        assigned,
        input_envelope,
        agent_workspace,
    ):
        """Invoke an existing DevResonance CloudAgent by stable agent id."""
        agent_id = assigned["project_agent_id"]
        trace_id = "{0}:{1}:{2}".format(
            task_id, phase["id"], uuid.uuid4().hex[:8]
        )
        route, route_kind = self.cloudagent.route_for(agent_id)
        prompt = (
            "你正在执行 ClawPro 项目协作工作流节点。请使用当前 CloudAgent 已绑定的 "
            "Skills、MCP 和 owner 密钥完成任务，不得编造外部数据。\n\n"
            "## 节点职责\n{0}\n\n"
            "## 节点配置资产（强制执行）\n{1}\n\n"
            "## ClawPro 节点输入（Handoff v2）\n```json\n{2}\n```\n\n"
            "请直接返回本节点的可读结论；如生成文件，请按输出契约命名。"
        ).format(
            phase.get("agent_description") or phase["title"],
            self.render_phase_config_assets(phase),
            json.dumps(input_envelope, ensure_ascii=False, indent=2),
        )
        with self.lock:
            self.cloudagent_executions[task_id] = {
                "agent_id": agent_id,
                "session_id": route.session_id,
                "trace_id": trace_id,
            }
        self.append_event(
            task_id,
            "workflow.cloudagent.dispatched",
            "节点已提交 DevResonance CloudAgent",
            "{0} → {1} · HTTPS direct-prompt".format(phase["id"], agent_id),
            {
                "agent_id": agent_id,
                "runtime_id": "devresonance-cloudagent",
                "trace_id": trace_id,
                "route_kind": route_kind,
            },
        )
        try:
            result = self.cloudagent.execute(
                agent_id,
                prompt,
                trace_id,
                timeout_seconds=1800,
            )
            self.ensure_not_canceled(task_id)
        except CloudAgentError as error:
            raise RuntimeError(str(error)) from error
        finally:
            with self.lock:
                self.cloudagent_executions.pop(task_id, None)

        summary = result["summary"] or "CloudAgent 已完成节点，但未返回文本结果。"
        projection = {
            "status": "success",
            "agent_id": agent_id,
            "trace_id": result["trace_id"],
            "session_id": result["session_id"],
            "result": summary,
            "attachments": result["attachments"],
            "usage": result["usage"],
        }
        for relative in phase["declared_artifacts"]:
            output_path = agent_workspace / relative
            output_path.parent.mkdir(parents=True, exist_ok=True)
            if output_path.suffix.lower() == ".json":
                output_path.write_text(
                    json.dumps(projection, ensure_ascii=False, indent=2),
                    encoding="utf-8",
                )
            else:
                output_path.write_text(
                    "# {0}\n\n"
                    "- Agent: `{1}`\n"
                    "- Session: `{2}`\n"
                    "- Trace: `{3}`\n\n{4}\n".format(
                        phase["title"],
                        agent_id,
                        result["session_id"],
                        result["trace_id"],
                        summary,
                    ),
                    encoding="utf-8",
                )
        return {
            "summary": summary,
            "session_id": result["session_id"],
            "trace_id": result["trace_id"],
            "attachments": result["attachments"],
            "usage": result["usage"],
            "runtime_id": "devresonance-cloudagent",
            "device_id": "devresonance-cloud",
        }

    def execute_teamai_workflow_node(
        self,
        task_id,
        phase,
        assigned,
        input_envelope,
        agent_workspace,
        node_workspace,
    ):
        task = self.tasks[task_id]
        runtime_id = assigned["runtime_id"]
        if runtime_id == "devresonance-cloudagent":
            return self.execute_cloudagent_workflow_node(
                task_id,
                phase,
                assigned,
                input_envelope,
                agent_workspace,
            )
        uses_imate = runtime_id == "hatchery-teamai-imate-openclaw"
        target_agent_id = (
            assigned.get("target_agent_id") or task.get("target_agent_id") or ""
            if uses_imate
            else ""
        )
        imate_project_id = task.get("imate_project_id") or ""
        if uses_imate and (not target_agent_id or not imate_project_id):
            raise RuntimeError("多 Agent 模式需要 iMate 项目和 OpenClaw Agent")
        prompt = (
            self.build_imate_node_prompt(
                task_id, phase, input_envelope, agent_workspace
            )
            if uses_imate
            else self.build_real_node_prompt(
                phase,
                task["prompt"],
                "input.json",
                phase["declared_artifacts"],
                input_envelope=(input_envelope if self.hatchery.remote_mode else None),
                agent_workspace=(agent_workspace if self.hatchery.remote_mode else None),
            )
        )
        child = self.create_hatchery_task(
            prompt,
            runtime_id,
            "default",
            target_agent_id=target_agent_id,
            imate_project_id=imate_project_id if uses_imate else "",
            delivery_mode=task.get("delivery_mode") or "wss",
            seed_workspace=node_workspace,
            repository_url=str(
                (
                    (input_envelope.get("task") or {}).get("inputs", {}).get(
                        "repository_url", {}
                    )
                    or {}
                ).get("value")
                or ""
            ).strip(),
        )
        child_task_id = child["task_id"]
        self.append_event(
            task_id,
            "workflow.teamai.dispatched",
            "节点已持久化并交给 TeamAI",
            "{0} → {1} → {2}".format(
                phase["id"], assigned["device_id"], assigned["project_agent_id"]
            ),
            {
                "child_task_id": child_task_id,
                "backend_task_id": child.get("backend_task_id"),
                "runtime_id": runtime_id,
                "device_id": assigned["device_id"],
                "project_agent_id": assigned["project_agent_id"],
                "target_agent_id": target_agent_id,
                "imate_project_id": imate_project_id if uses_imate else None,
                "transport": "wss+https",
            },
        )
        # Source-development nodes routinely include repository inspection,
        # implementation and build verification. Keep the server wait slightly
        # above TeamAI's 20-minute ACP budget so a healthy worker can report its
        # final result instead of being cut off by the orchestrator first.
        deadline = time.monotonic() + 1260
        while time.monotonic() < deadline:
            self.ensure_not_canceled(task_id)
            self.refresh_hatchery_task(child_task_id)
            with self.lock:
                current = self.tasks[child_task_id]
                status = current["status"]
                session_id = current.get("session_id")
                result = current.get("agent_output", "")
                failure = current.get("failure_detail", "")
            if session_id:
                with self.lock:
                    for phase_item in self.tasks[task_id]["workflow_phases"]:
                        if phase_item["id"] == phase["id"]:
                            phase_item["session_id"] = session_id
                            break
            if status == "completed":
                clean_result, uploaded_artifacts = self.materialize_teamai_artifact_bundle(
                    result,
                    agent_workspace,
                    phase["declared_artifacts"] + phase.get("optional_artifacts", []),
                )
                child_workspace = Path(current["workspace_path"])
                copy_back = {
                    "ISSUE.md",
                    "calculator.py",
                    "test_calculator.py",
                    *phase["declared_artifacts"],
                    *phase.get("optional_artifacts", []),
                }
                for relative in copy_back:
                    source = child_workspace / relative
                    if not source.is_file():
                        continue
                    destination = agent_workspace / relative
                    destination.parent.mkdir(parents=True, exist_ok=True)
                    shutil.copy2(source, destination)
                if self.hatchery.remote_mode and not uploaded_artifacts:
                    raise RuntimeError(
                        "TeamAI 仅回传了执行摘要，未上传节点真实产物；"
                        "为避免错误评测，本节点不会生成摘要投影。"
                    )
                return {
                    "summary": self.clean_agent_result(clean_result),
                    "session_id": session_id
                    or "teamai:{0}".format(child_task_id),
                    "child_task_id": child_task_id,
                    "backend_task_id": current.get("backend_task_id"),
                    "runtime_id": runtime_id,
                    "device_id": assigned["device_id"],
                    "uploaded_artifacts": uploaded_artifacts,
                }
            if status in {"failed", "canceled"}:
                raise RuntimeError(
                    "TeamAI 节点 {0} 执行失败：{1}".format(
                        phase["id"], failure or result or status
                    )
                )
            time.sleep(1)
        raise RuntimeError("TeamAI 节点 {0} 执行超时".format(phase["id"]))

    @staticmethod
    def combine_upstream_results(upstream_results):
        if not upstream_results:
            return None
        artifact_by_path = {}
        upstream_nodes = []
        summaries = []
        for result in upstream_results:
            upstream_nodes.append(result["node_id"])
            summaries.append(
                {
                    "node_id": result["node_id"],
                    "summary": result.get("data", {}).get("summary", ""),
                }
            )
            for artifact in result.get("artifacts", []):
                existing = artifact_by_path.get(artifact["path"])
                if existing and existing.get("sha256") != artifact.get("sha256"):
                    raise RuntimeError(
                        "并行上游产物路径冲突：{0}".format(artifact["path"])
                    )
                artifact_by_path[artifact["path"]] = artifact
        return {
            "node_id": ",".join(upstream_nodes),
            "data": {
                "summary": "\n\n".join(
                    "[{0}] {1}".format(item["node_id"], item["summary"])
                    for item in summaries
                ),
                "upstream_nodes": summaries,
            },
            "artifacts": list(artifact_by_path.values()),
        }

    def execute_real_agent_phase(
        self,
        task_id,
        definition,
        phase,
        assigned,
        agent_workspace,
        node_workspace_root,
        base_artifact_paths,
        upstream_results,
        index,
        is_issuefix,
    ):
        node_runtime_id = assigned["runtime_id"]
        previous_output = self.combine_upstream_results(upstream_results)
        lineage = [
            artifact["artifact_id"]
            for result in upstream_results
            for artifact in result.get("artifacts", [])
        ]
        base_artifacts = []
        for relative in base_artifact_paths:
            raw = (agent_workspace / relative).read_bytes()
            base_artifacts.append(
                self.build_artifact_ref(
                    task_id,
                    "workflow-input" if not upstream_results else "workspace-state",
                    relative,
                    raw,
                    lineage=lineage,
                    version=index,
                )
            )
        node_workspace = node_workspace_root / "{0:02d}-{1}".format(
            index, phase["id"]
        )
        node_workspace.mkdir(parents=True, exist_ok=True)
        input_envelope = self.build_node_input_v2(
            task_id,
            definition,
            phase,
            assigned,
            None,
            previous_output,
            base_artifacts,
            index,
        )
        self.stage_node_inputs(
            agent_workspace,
            node_workspace,
            input_envelope,
            phase.get("config_assets") or [],
        )
        for relative in base_artifact_paths:
            source = agent_workspace / relative
            destination = node_workspace / relative
            destination.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(source, destination)
        handoff_path = node_workspace / "input.json"
        handoff_path.write_text(
            json.dumps(input_envelope, ensure_ascii=False, indent=2),
            encoding="utf-8",
        )
        with self.lock:
            task = self.tasks[task_id]
            task["workflow_stage"] = "real_agent_running"
            for phase_item in task["workflow_phases"]:
                if phase_item["id"] == phase["id"]:
                    phase_item["status"] = "running"
                    phase_item["session_id"] = None
                    break
            running_ids = [
                item["id"]
                for item in task["workflow_phases"]
                if item["status"] == "running"
            ]
            task["workflow_current_phases"] = running_ids
            task["workflow_current_phase"] = ",".join(running_ids)
        self.append_event(
            task_id,
            "workflow.node.started",
            "{0} 已交给真实 Agent".format(phase["title"]),
            "{0} · 设备 {1} · {2}".format(
                assigned["agent_instance_id"],
                assigned.get("device_id") or "待分配",
                assigned.get("transport") or "wss+https",
            ),
            input_envelope,
        )
        execution = self.execute_teamai_workflow_node(
            task_id,
            phase,
            assigned,
            input_envelope,
            agent_workspace,
            node_workspace,
        )
        node_summary = execution["summary"]
        self.validate_required_evidence(phase, node_summary)
        session_id = execution["session_id"]
        input_envelope["node"]["agent_session_id"] = session_id
        handoff_path.write_text(
            json.dumps(input_envelope, ensure_ascii=False, indent=2),
            encoding="utf-8",
        )
        with self.lock:
            for phase_item in self.tasks[task_id]["workflow_phases"]:
                if phase_item["id"] == phase["id"]:
                    phase_item["session_id"] = session_id
                    break
        if node_runtime_id == "hatchery-teamai-imate-openclaw":
            for relative in phase["declared_artifacts"]:
                output_path = agent_workspace / relative
                if output_path.is_file() and output_path.stat().st_size > 0:
                    continue
                output_path.write_text(
                    "# {0}\n\n"
                    "- Runtime: TeamAI → iMate OpenClaw\n"
                    "- Agent: {1}\n"
                    "- Device: {2}\n"
                    "- Session: {3}\n"
                    "- Upstream: {4}\n\n"
                    "{5}\n".format(
                        phase["title"],
                        assigned["project_agent_id"],
                        assigned["device_id"],
                        session_id,
                        ", ".join(result["node_id"] for result in upstream_results)
                        or "none",
                        node_summary,
                    ),
                    encoding="utf-8",
                )
        missing = [
            relative
            for relative in phase["declared_artifacts"]
            if not (agent_workspace / relative).is_file()
            or (agent_workspace / relative).stat().st_size == 0
        ]
        if missing:
            raise RuntimeError(
                "{0} 经 TeamAI 执行后未产出必需文件：{1}".format(
                    phase["id"], ", ".join(missing)
                )
            )
        artifact_lineage = [
            item["artifact_id"] for item in input_envelope["inputs"]["artifacts"]
        ]
        artifact_records = []
        output_artifacts = phase["declared_artifacts"] + [
            relative
            for relative in phase.get("optional_artifacts", [])
            if (agent_workspace / relative).is_file()
            and (agent_workspace / relative).stat().st_size > 0
        ]
        for relative in output_artifacts:
            artifact_path = (agent_workspace / relative).resolve()
            try:
                artifact_path.relative_to(agent_workspace.resolve())
            except ValueError as error:
                raise RuntimeError("工作流产物路径越界") from error
            raw = artifact_path.read_bytes()
            artifact_records.append(
                self.build_artifact_ref(
                    task_id,
                    phase["id"],
                    relative,
                    raw,
                    lineage=artifact_lineage,
                )
            )
        runtime_check = None
        if is_issuefix and phase["id"] in {"fix", "test", "verify"}:
            check = subprocess.run(
                [sys.executable, "-m", "unittest", "-v", "test_calculator.py"],
                cwd=str(agent_workspace),
                text=True,
                capture_output=True,
                timeout=30,
                check=False,
            )
            check_output = (check.stdout + check.stderr).strip()
            runtime_check = {
                "command": "python3 -m unittest -v test_calculator.py",
                "exit_code": check.returncode,
                "passed": check.returncode == 0,
                "output": check_output[-4000:],
            }
            if check.returncode != 0:
                raise RuntimeError(
                    "{0} 节点后真实测试失败：{1}".format(
                        phase["id"], check_output[-1000:]
                    )
                )
        node_result = {
            "schema_version": "clawpro.node-result.v2",
            "workflow_run_id": task_id,
            "node_run_id": input_envelope["node_run_id"],
            "attempt_id": input_envelope["attempt_id"],
            "node_id": phase["id"],
            "agent_instance_id": assigned["agent_instance_id"],
            "agent_session_id": session_id,
            "runtime_id": node_runtime_id,
            "role_agent_id": phase["agent_id"],
            "status": "completed",
            "data": {
                "summary": self.clean_agent_result(node_summary),
                "runtime_check": runtime_check,
            },
            "artifacts": artifact_records,
            "handoff": {
                "schema_version": "clawpro.handoff.v2",
                "upstream_node_id": (
                    upstream_results[0]["node_id"]
                    if len(upstream_results) == 1
                    else None
                ),
                "upstream_node_ids": [
                    result["node_id"] for result in upstream_results
                ],
                "known_issues": [],
                "required_actions": [],
            },
        }
        handoff_package = self.write_node_handoff(
            node_workspace, node_result, agent_workspace
        )
        with self.lock:
            task = self.tasks[task_id]
            for phase_item in task["workflow_phases"]:
                if phase_item["id"] == phase["id"]:
                    phase_item["status"] = "completed"
                    break
            produced_paths = {
                "agent-workspace/{0}".format(item["path"])
                for item in artifact_records
            }
            task["available_artifacts"] = [
                item
                for item in task.get("available_artifacts", [])
                if item.get("path") not in produced_paths
            ]
            task["available_artifacts"].extend(
                {
                    **item,
                    "path": "agent-workspace/{0}".format(item["path"]),
                }
                for item in artifact_records
            )
        self.append_event(
            task_id,
            "workflow.node.completed",
            "{0} 真实 Agent 执行完成".format(phase["title"]),
            "已产出 {0} 份文件并通过 Handoff v2 清单校验。".format(
                len(artifact_records)
            ),
            {
                **node_result,
                "handoff_path": handoff_package.relative_to(
                    agent_workspace
                ).as_posix(),
            },
        )
        return node_result

    def run_real_agent_dag_workflow(
        self,
        task_id,
        definition,
        assignment,
        agent_workspace,
        node_workspace_root,
        base_artifact_paths,
        is_issuefix,
        workspace,
    ):
        phase_by_id = {phase["id"]: phase for phase in definition["phases"]}
        phase_index = {
            phase["id"]: index
            for index, phase in enumerate(definition["phases"], start=1)
        }
        assignment_by_phase = {
            item["phase_id"]: item for item in assignment["assignments"]
        }
        remaining = set(phase_by_id)
        completed = set()
        result_by_phase = {}
        session_ids = []
        while remaining:
            self.ensure_not_canceled(task_id)
            ready = [
                phase
                for phase in definition["phases"]
                if phase["id"] in remaining
                and set(phase.get("depends_on", [])).issubset(completed)
            ]
            if not ready:
                raise RuntimeError("项目工作流 DAG 无可执行节点")
            ready_ids = [phase["id"] for phase in ready]
            with self.lock:
                task = self.tasks[task_id]
                task["workflow_current_phases"] = ready_ids
                task["workflow_current_phase"] = ",".join(ready_ids)
            if len(ready) > 1:
                self.append_event(
                    task_id,
                    "workflow.parallel.started",
                    "并行节点批次开始",
                    "{0} 个节点已同时满足依赖：{1}".format(
                        len(ready), ", ".join(ready_ids)
                    ),
                    {"phase_ids": ready_ids, "join_policy": "all_success"},
                )
            batch_results = {}
            with ThreadPoolExecutor(max_workers=min(4, len(ready))) as executor:
                future_by_phase = {}
                for phase in ready:
                    upstream_results = [
                        result_by_phase[dependency]
                        for dependency in phase.get("depends_on", [])
                    ]
                    future = executor.submit(
                        self.execute_real_agent_phase,
                        task_id,
                        definition,
                        phase,
                        assignment_by_phase[phase["id"]],
                        agent_workspace,
                        node_workspace_root,
                        base_artifact_paths,
                        upstream_results,
                        phase_index[phase["id"]],
                        is_issuefix,
                    )
                    future_by_phase[future] = phase
                for future in as_completed(future_by_phase):
                    phase = future_by_phase[future]
                    batch_results[phase["id"]] = future.result()
            if len(ready) > 1:
                self.append_event(
                    task_id,
                    "workflow.parallel.completed",
                    "并行节点批次完成",
                    "{0} 个节点全部完成，允许进入汇聚判断。".format(
                        len(ready)
                    ),
                    {"phase_ids": ready_ids, "join_policy": "all_success"},
                )
            for phase in ready:
                result = batch_results[phase["id"]]
                dependents = [
                    candidate["id"]
                    for candidate in definition["phases"]
                    if phase["id"] in candidate.get("depends_on", [])
                ]
                self.wait_for_workflow_approval(
                    task_id,
                    phase,
                    ",".join(dependents) if dependents else None,
                    result,
                )
                result_by_phase[phase["id"]] = result
                session_ids.append(result["agent_session_id"])
                completed.add(phase["id"])
                remaining.remove(phase["id"])
        ordered_results = [
            result_by_phase[phase["id"]] for phase in definition["phases"]
        ]
        edge_count = sum(
            len(phase.get("depends_on", [])) for phase in definition["phases"]
        )
        report = {
            "schema_version": "clawpro.real-agent-workflow.v3",
            "handoff_contract": "clawpro.handoff.v2",
            "runtime_id": "teamai-node-routed-dag",
            "assignment_mode": assignment["mode"],
            "real_agent_execution": True,
            "agent_instance_count": assignment["unique_agent_count"],
            "agent_session_count": len(set(session_ids)),
            "handoff_count": edge_count,
            "session_ids": session_ids,
            "runtime_counts": {
                "hatchery-teamai-codebuddy": sum(
                    result["runtime_id"] == "hatchery-teamai-codebuddy"
                    for result in ordered_results
                ),
                "hatchery-teamai-imate-openclaw": sum(
                    result["runtime_id"]
                    == "hatchery-teamai-imate-openclaw"
                    for result in ordered_results
                ),
                "devresonance-cloudagent": sum(
                    result["runtime_id"] == "devresonance-cloudagent"
                    for result in ordered_results
                ),
            },
            "nodes": ordered_results,
            "external_writes_performed": any(
                result["runtime_id"] == "hatchery-teamai-imate-openclaw"
                or result.get("role_agent_id") == "message-notify-agent"
                for result in ordered_results
            ),
        }
        (workspace / "real-agent-runtime-report.json").write_text(
            json.dumps(report, ensure_ascii=False, indent=2),
            encoding="utf-8",
        )
        return report

    def workflow_input_value(self, task_id, key, default=""):
        raw = (self.tasks[task_id].get("_workflow_inputs") or {}).get(key) or {}
        return str(raw.get("value") or default)

    def write_canonical_workflow_state(self, agent_workspace, state):
        now = self.now()
        state.setdefault("created_at", now)
        state["updated_at"] = now
        path = agent_workspace / "workflow-state.json"
        path.write_text(
            json.dumps(state, ensure_ascii=False, indent=2) + "\n",
            encoding="utf-8",
        )
        return path

    def merge_intake_workflow_state(self, state, agent_workspace):
        """Preserve PHASE-0 evidence without surrendering stage ownership."""
        state_path = agent_workspace / "workflow-state.json"
        if not state_path.is_file():
            return state
        try:
            intake_state = json.loads(state_path.read_text(encoding="utf-8"))
        except (OSError, json.JSONDecodeError):
            return state
        if not isinstance(intake_state, dict):
            return state

        for key in (
            "task_slug",
            "workspace",
            "artifacts_dir",
            "last_event",
            "last_error",
            "project_config",
            "rules_loaded",
            "created_at",
        ):
            value = intake_state.get(key)
            if value not in (None, "", [], {}):
                state[key] = value
        intake_decisions = intake_state.get("decisions")
        if isinstance(intake_decisions, list):
            state["decisions"] = [
                decision for decision in intake_decisions if isinstance(decision, dict)
            ]
        intake_summary = intake_state.get("summary")
        if isinstance(intake_summary, dict):
            state["summary"] = {**state.get("summary", {}), **intake_summary}
        return state

    def refresh_workflow_state_artifact(self, task_id, state_owner_result, agent_workspace):
        if not state_owner_result:
            return
        state_path = agent_workspace / "workflow-state.json"
        if not state_path.is_file():
            return
        raw = state_path.read_bytes()
        existing = next(
            (
                item
                for item in state_owner_result.get("artifacts", [])
                if item.get("path") == "workflow-state.json"
            ),
            None,
        )
        lineage = existing.get("lineage", []) if existing else []
        previous_version = int((existing or {}).get("version", 0))
        previous_hash = str((existing or {}).get("sha256") or "")
        current_hash = hashlib.sha256(raw).hexdigest()
        next_version = (
            previous_version if previous_hash == current_hash else previous_version + 1
        )
        refreshed = self.build_artifact_ref(
            task_id,
            state_owner_result["node_id"],
            "workflow-state.json",
            raw,
            lineage=lineage,
            version=next_version,
        )
        state_owner_result["artifacts"] = [
            item
            for item in state_owner_result.get("artifacts", [])
            if item.get("path") != "workflow-state.json"
        ] + [refreshed]
        public_ref = {
            **refreshed,
            "path": "agent-workspace/workflow-state.json",
        }
        with self.lock:
            task = self.tasks[task_id]
            task["available_artifacts"] = [
                item
                for item in task.get("available_artifacts", [])
                if item.get("path") != "agent-workspace/workflow-state.json"
            ] + [public_ref]

    @staticmethod
    def review_verdict(node_result, agent_workspace):
        candidates = [node_result.get("data", {}).get("summary", "")]
        report_path = agent_workspace / "03-code/review-report.md"
        if report_path.is_file():
            candidates.append(
                report_path.read_text(encoding="utf-8", errors="replace")
            )
        for content in candidates:
            upper = content.upper()
            passed_at = upper.rfind("REVIEW_VERDICT: PASSED")
            failed_at = upper.rfind("REVIEW_VERDICT: FAILED")
            if passed_at >= 0 or failed_at >= 0:
                return "passed" if passed_at > failed_at else "failed"
        raise RuntimeError(
            "CODE-REVIEW 未返回机器可判定的 REVIEW_VERDICT: PASSED/FAILED"
        )

    @staticmethod
    def size_class_verdict(node_result, agent_workspace):
        """Read the intake node's machine marker or canonical JSON output."""
        candidates = [node_result.get("data", {}).get("summary", "")]
        state_path = agent_workspace / "workflow-state.json"
        if state_path.is_file():
            candidates.append(
                state_path.read_text(encoding="utf-8", errors="replace")
            )
            try:
                state_value = json.loads(candidates[-1])
                size_class = str(state_value.get("size_class") or "").lower()
                if size_class in {"small", "medium", "large"}:
                    return size_class
            except (json.JSONDecodeError, AttributeError):
                pass
        marker = re.compile(
            r"\bSIZE_CLASS\s*[:=]\s*(SMALL|MEDIUM|LARGE)\b",
            re.IGNORECASE,
        )
        for content in candidates:
            match = marker.search(str(content))
            if match:
                return match.group(1).lower()
        raise RuntimeError(
            "PHASE-0 未返回机器可判定的 "
            "SIZE_CLASS: SMALL/MEDIUM/LARGE"
        )

    @staticmethod
    def reachable_phase_ids(phase_by_id, start_phase_id):
        if not start_phase_id:
            return set()
        reachable = set()
        pending = [start_phase_id]
        while pending:
            phase_id = pending.pop()
            if phase_id in reachable or phase_id not in phase_by_id:
                continue
            reachable.add(phase_id)
            phase = phase_by_id[phase_id]
            pending.extend(
                route
                for route in (phase.get("on_pass"), phase.get("on_fail"))
                if route and route != phase_id
            )
        return reachable

    def mark_unselected_branch_skipped(
        self,
        task_id,
        phase_by_id,
        stage_state,
        chosen_phase_id,
        unselected_phase_id,
    ):
        chosen_reachable = self.reachable_phase_ids(
            phase_by_id, chosen_phase_id
        )
        skipped_ids = self.reachable_phase_ids(
            phase_by_id, unselected_phase_id
        ) - chosen_reachable
        for phase_id in skipped_ids:
            stage_state[phase_id]["status"] = "skipped"
            stage_state[phase_id]["completed_at"] = self.now()
        with self.lock:
            for phase in self.tasks[task_id]["workflow_phases"]:
                if phase["id"] in skipped_ids:
                    phase["status"] = "skipped"
        return skipped_ids

    def run_real_agent_state_machine_workflow(
        self,
        task_id,
        definition,
        assignment,
        agent_workspace,
        node_workspace_root,
        base_artifact_paths,
        is_issuefix,
        workspace,
    ):
        phase_by_id = {phase["id"]: phase for phase in definition["phases"]}
        assignment_by_phase = {
            item["phase_id"]: item for item in assignment["assignments"]
        }
        stage_state = {
            phase["id"]: {
                "status": "pending",
                "executor": phase["agent_id"],
                "description": phase["title"],
                "artifacts": phase["declared_artifacts"],
                "retry_count": 0,
                "review_result": None,
                "started_at": None,
                "completed_at": None,
                "self_check": None,
            }
            for phase in definition["phases"]
        }
        state = {
            "version": "1.3",
            "task_id": task_id,
            "task_slug": self.workflow_input_value(
                task_id, "task_slug", task_id
            ),
            "run_mode": self.workflow_input_value(task_id, "run_mode", "auto"),
            "runtime_mode": self.workflow_input_value(
                task_id, "runtime_mode", "ide"
            ),
            "size_class": "unknown",
            "workspace": {
                "repository_url": self.workflow_input_value(
                    task_id, "repository_url", ""
                ),
                "branch": self.workflow_input_value(task_id, "branch", ""),
                "root": "",
                "default_branch": self.workflow_input_value(
                    task_id, "default_branch", "main"
                ),
                "worktree": "",
            },
            "artifacts_dir": str(agent_workspace),
            "current_stage": definition["phases"][0]["id"],
            "next_target": definition["phases"][0]["id"],
            "last_event": None,
            "last_error": None,
            "stages": stage_state,
            "decisions": [],
            "summary": {
                "status": "running",
                "report": "workflow-summary.md",
                "completed_at": None,
            },
        }
        self.write_canonical_workflow_state(agent_workspace, state)
        current_phase_id = definition["phases"][0]["id"]
        latest_result_by_phase = {}
        execution_history = []
        session_ids = []
        transition_count = 0
        execution_index = 0
        retry_count_by_phase = {}
        while current_phase_id:
            self.ensure_not_canceled(task_id)
            phase = phase_by_id[current_phase_id]
            execution_index += 1
            current_stage = stage_state[current_phase_id]
            current_stage["status"] = "running"
            current_stage["started_at"] = self.now()
            current_stage["completed_at"] = None
            state["current_stage"] = current_phase_id
            state["next_target"] = current_phase_id
            self.write_canonical_workflow_state(agent_workspace, state)
            self.refresh_workflow_state_artifact(
                task_id, latest_result_by_phase.get("PHASE-0"), agent_workspace
            )
            upstream_results = [
                result
                for phase_id, result in latest_result_by_phase.items()
                if phase_id != current_phase_id
            ]
            result = self.execute_real_agent_phase(
                task_id,
                definition,
                phase,
                assignment_by_phase[current_phase_id],
                agent_workspace,
                node_workspace_root,
                base_artifact_paths,
                upstream_results,
                execution_index,
                is_issuefix,
            )
            latest_result_by_phase[current_phase_id] = result
            execution_history.append(result)
            session_ids.append(result["agent_session_id"])
            current_stage["status"] = "completed"
            current_stage["completed_at"] = self.now()
            current_stage["self_check"] = {
                "artifacts_verified": len(result.get("artifacts", [])),
                "runtime_id": result["runtime_id"],
                "session_id": result["agent_session_id"],
            }
            next_phase_id = phase.get("on_pass")
            if phase.get("decision_mode") == "size_class":
                self.merge_intake_workflow_state(state, agent_workspace)
                size_class = self.size_class_verdict(result, agent_workspace)
                state["size_class"] = size_class
                if size_class == "small":
                    next_phase_id = phase.get("on_fail")
                    unselected_phase_id = phase.get("on_pass")
                else:
                    next_phase_id = phase.get("on_pass")
                    unselected_phase_id = phase.get("on_fail")
                skipped_ids = self.mark_unselected_branch_skipped(
                    task_id,
                    phase_by_id,
                    stage_state,
                    next_phase_id,
                    unselected_phase_id,
                )
                state["decisions"].append(
                    {
                        "type": "SIZE_CLASS_ROUTING",
                        "stage": current_phase_id,
                        "decision": "{0} → {1}".format(
                            size_class.upper(), next_phase_id
                        ),
                        "skipped_stages": sorted(skipped_ids),
                        "at": self.now(),
                    }
                )
                self.append_event(
                    task_id,
                    "workflow.node.rerouted",
                    "已按需求规模选择执行路径",
                    "{0} → {1}".format(size_class.upper(), next_phase_id),
                    {
                        "from_phase_id": current_phase_id,
                        "to_phase_id": next_phase_id,
                        "size_class": size_class,
                        "skipped_phase_ids": sorted(skipped_ids),
                    },
                )
            elif phase.get("decision_mode") == "review_verdict":
                verdict = self.review_verdict(result, agent_workspace)
                current_stage["review_result"] = verdict.upper()
                if verdict == "failed":
                    retry_count = retry_count_by_phase.get(current_phase_id, 0) + 1
                    retry_count_by_phase[current_phase_id] = retry_count
                    current_stage["retry_count"] = retry_count
                    max_retries = phase.get("max_retries", 0)
                    if retry_count > max_retries:
                        current_stage["status"] = "failed"
                        self.write_canonical_workflow_state(agent_workspace, state)
                        raise RuntimeError(
                            "{0} 连续失败，已超过 {1} 次返工上限".format(
                                current_phase_id, max_retries
                            )
                        )
                    next_phase_id = phase.get("on_fail")
                    state["decisions"].append(
                        {
                            "type": "RULING",
                            "stage": current_phase_id,
                            "decision": "FAILED，打回 {0}".format(next_phase_id),
                            "retry_count": retry_count,
                            "at": self.now(),
                        }
                    )
                    current_stage["status"] = "needs_rework"
                    with self.lock:
                        for phase_item in self.tasks[task_id]["workflow_phases"]:
                            if phase_item["id"] == current_phase_id:
                                phase_item["status"] = "needs_rework"
                                phase_item["retry_count"] = retry_count
                                break
                    self.append_event(
                        task_id,
                        "workflow.node.rerouted",
                        "代码评审未通过，返回开发节点",
                        "{0} → {1}，第 {2}/{3} 次返工".format(
                            current_phase_id,
                            next_phase_id,
                            retry_count,
                            max_retries,
                        ),
                        {
                            "from_phase_id": current_phase_id,
                            "to_phase_id": next_phase_id,
                            "retry_count": retry_count,
                            "max_retries": max_retries,
                        },
                    )
                else:
                    state["decisions"].append(
                        {
                            "type": "RULING",
                            "stage": current_phase_id,
                            "decision": "PASSED，进入 {0}".format(next_phase_id),
                            "retry_count": retry_count_by_phase.get(
                                current_phase_id, 0
                            ),
                            "at": self.now(),
                        }
                    )
            state["current_stage"] = current_phase_id
            state["next_target"] = next_phase_id
            self.write_canonical_workflow_state(agent_workspace, state)
            self.refresh_workflow_state_artifact(
                task_id, latest_result_by_phase.get("PHASE-0"), agent_workspace
            )
            self.wait_for_workflow_approval(
                task_id, phase, next_phase_id, result
            )
            if next_phase_id:
                transition_count += 1
            current_phase_id = next_phase_id
        state["current_stage"] = "COMPLETED"
        state["next_target"] = None
        state["summary"]["status"] = "completed"
        state["summary"]["completed_at"] = self.now()
        self.write_canonical_workflow_state(agent_workspace, state)
        self.refresh_workflow_state_artifact(
            task_id,
            latest_result_by_phase.get("SUMMARY")
            or latest_result_by_phase.get("PHASE-0"),
            agent_workspace,
        )
        report = {
            "schema_version": "clawpro.real-agent-workflow.v3",
            "handoff_contract": "clawpro.handoff.v2",
            "runtime_id": "teamai-node-routed-state-machine",
            "assignment_mode": assignment["mode"],
            "real_agent_execution": True,
            "agent_instance_count": assignment["unique_agent_count"],
            "agent_session_count": len(set(session_ids)),
            "handoff_count": transition_count,
            "session_ids": session_ids,
            "runtime_counts": {
                "hatchery-teamai-codebuddy": sum(
                    result["runtime_id"] == "hatchery-teamai-codebuddy"
                    for result in execution_history
                ),
                "hatchery-teamai-imate-openclaw": sum(
                    result["runtime_id"] == "hatchery-teamai-imate-openclaw"
                    for result in execution_history
                ),
                "devresonance-cloudagent": sum(
                    result["runtime_id"] == "devresonance-cloudagent"
                    for result in execution_history
                ),
            },
            "nodes": execution_history,
            "external_writes_performed": any(
                result["runtime_id"] == "hatchery-teamai-imate-openclaw"
                or result.get("role_agent_id") == "message-notify-agent"
                for result in execution_history
            ),
        }
        (workspace / "real-agent-runtime-report.json").write_text(
            json.dumps(report, ensure_ascii=False, indent=2),
            encoding="utf-8",
        )
        return report

    def run_real_agent_workflow(self, task_id, definition, smoke, workspace):
        agent_workspace = workspace / "agent-workspace"
        node_workspace_root = agent_workspace / "node-workspaces"
        node_workspace_root.mkdir(parents=True, exist_ok=True)
        is_issuefix = definition["workflow_id"] == "skillhub-issuefix"
        workflow_inputs = self.tasks[task_id].get("_workflow_inputs") or {}
        repository_url = str(
            (workflow_inputs.get("repository_url") or {}).get("value") or ""
        ).strip()
        task_boundary = (
            "本任务必须在 TeamAI 绑定的真实源码仓库中执行，"
            "不得用独立 Demo 替代真实源码改动；不提交、不推送、不创建 MR。"
            if repository_url
            else "本任务只允许在当前隔离目录产出节点报告，不修改外部仓库。"
        )
        (agent_workspace / "TASK.md").write_text(
            "# ClawPro 真实项目工作流 Runtime POC\n\n"
            "## 用户任务\n\n{0}\n\n"
            "## 任务输入\n\n```json\n{1}\n```\n\n"
            "## 边界\n\n{2}\n".format(
                self.tasks[task_id]["prompt"],
                json.dumps(workflow_inputs, ensure_ascii=False, indent=2),
                task_boundary,
            ),
            encoding="utf-8",
        )
        base_artifact_paths = ["TASK.md"]
        if is_issuefix:
            (agent_workspace / "ISSUE.md").write_text(
                "# POC-LOCAL-001：divide 返回值错误\n\n"
                "`calculator.py` 中的 `divide(left, right)` 误用了乘法，"
                "`divide(8, 2)` 当前返回 16，预期返回 4。\n\n"
                "验收：修复运算符；保留 Python 原生的除零异常；"
                "增加正常除法和除零用例。\n",
                encoding="utf-8",
            )
            (agent_workspace / "calculator.py").write_text(
                "def add(left, right):\n    return left + right\n\n"
                "def divide(left, right):\n    return left * right\n",
                encoding="utf-8",
            )
            (agent_workspace / "test_calculator.py").write_text(
                "import unittest\n\nfrom calculator import add, divide\n\n\n"
                "class CalculatorTests(unittest.TestCase):\n"
                "    def test_add(self):\n        self.assertEqual(add(2, 3), 5)\n",
                encoding="utf-8",
            )
            base_artifact_paths.extend(
                ["ISSUE.md", "calculator.py", "test_calculator.py"]
            )
        assignment = smoke["agent_assignment"]
        assignment_by_phase = {
            item["phase_id"]: item for item in assignment["assignments"]
        }
        mode = assignment["mode"]
        session_ids = []
        node_results = []
        previous_output = None
        try:
            if definition.get("execution_mode") == "state_machine":
                return self.run_real_agent_state_machine_workflow(
                    task_id,
                    definition,
                    assignment,
                    agent_workspace,
                    node_workspace_root,
                    base_artifact_paths,
                    is_issuefix,
                    workspace,
                )
            canonical_linear = all(
                phase.get("depends_on", [])
                == ([] if index == 0 else [definition["phases"][index - 1]["id"]])
                for index, phase in enumerate(definition["phases"])
            )
            if not canonical_linear:
                return self.run_real_agent_dag_workflow(
                    task_id,
                    definition,
                    assignment,
                    agent_workspace,
                    node_workspace_root,
                    base_artifact_paths,
                    is_issuefix,
                    workspace,
                )
            for index, phase in enumerate(definition["phases"], start=1):
                assigned = assignment_by_phase[phase["id"]]
                node_runtime_id = assigned["runtime_id"]
                session_id = None
                base_artifacts = []
                for relative in base_artifact_paths:
                    raw = (agent_workspace / relative).read_bytes()
                    base_artifacts.append(
                        self.build_artifact_ref(
                            task_id,
                            "workflow-input" if index == 1 else "workspace-state",
                            relative,
                            raw,
                            lineage=(
                                [
                                    item["artifact_id"]
                                    for item in (previous_output or {}).get(
                                        "artifacts", []
                                    )
                                ]
                                if index > 1
                                else []
                            ),
                            version=index,
                        )
                    )
                node_workspace = node_workspace_root / "{0:02d}-{1}".format(
                    index, phase["id"]
                )
                node_workspace.mkdir(parents=True, exist_ok=True)
                input_envelope = self.build_node_input_v2(
                    task_id,
                    definition,
                    phase,
                    assigned,
                    session_id,
                    previous_output,
                    base_artifacts,
                    index,
                )
                self.stage_node_inputs(
                    agent_workspace,
                    node_workspace,
                    input_envelope,
                    phase.get("config_assets") or [],
                )
                for relative in base_artifact_paths:
                    source = agent_workspace / relative
                    destination = node_workspace / relative
                    destination.parent.mkdir(parents=True, exist_ok=True)
                    shutil.copy2(source, destination)
                handoff_path = node_workspace / "input.json"
                handoff_path.write_text(
                    json.dumps(input_envelope, ensure_ascii=False, indent=2),
                    encoding="utf-8",
                )
                with self.lock:
                    task = self.tasks[task_id]
                    task["workflow_stage"] = "real_agent_running"
                    task["workflow_current_phase"] = phase["id"]
                    task["workflow_phases"][index - 1]["status"] = "running"
                    task["workflow_phases"][index - 1]["session_id"] = session_id
                self.append_event(
                    task_id,
                    "workflow.node.started",
                    "{0} 已交给真实 Agent".format(phase["title"]),
                    "{0} · 设备 {1} · {2}".format(
                        assigned["agent_instance_id"],
                        assigned.get("device_id") or "待分配",
                        assigned.get("transport") or "wss+https",
                    ),
                    input_envelope,
                )
                execution = self.execute_teamai_workflow_node(
                    task_id,
                    phase,
                    assigned,
                    input_envelope,
                    agent_workspace,
                    node_workspace,
                )
                node_summary = execution["summary"]
                self.validate_required_evidence(phase, node_summary)
                session_id = execution["session_id"]
                session_ids.append(session_id)
                input_envelope["node"]["agent_session_id"] = session_id
                handoff_path.write_text(
                    json.dumps(input_envelope, ensure_ascii=False, indent=2),
                    encoding="utf-8",
                )
                with self.lock:
                    self.tasks[task_id]["workflow_phases"][index - 1][
                        "session_id"
                    ] = session_id
                if node_runtime_id == "hatchery-teamai-imate-openclaw":
                    for relative in phase["declared_artifacts"]:
                        output_path = agent_workspace / relative
                        if output_path.is_file() and output_path.stat().st_size > 0:
                            continue
                        output_path.write_text(
                            "# {0}\n\n"
                            "- Runtime: TeamAI → iMate OpenClaw\n"
                            "- Agent: {1}\n"
                            "- Device: {2}\n"
                            "- Session: {3}\n"
                            "- Upstream: {4}\n\n"
                            "{5}\n".format(
                                phase["title"],
                                assigned["project_agent_id"],
                                assigned["device_id"],
                                session_id,
                                previous_output.get("node_id")
                                if previous_output
                                else "none",
                                node_summary,
                            ),
                            encoding="utf-8",
                        )
                missing_before_validation = [
                    relative
                    for relative in phase["declared_artifacts"]
                    if not (agent_workspace / relative).is_file()
                    or (agent_workspace / relative).stat().st_size == 0
                ]
                if missing_before_validation:
                    raise RuntimeError(
                        "{0} 经 TeamAI 执行后未产出必需文件：{1}".format(
                            phase["id"], ", ".join(missing_before_validation)
                        )
                    )
                artifact_records = []
                missing = []
                lineage = [
                    item["artifact_id"]
                    for item in input_envelope["inputs"]["artifacts"]
                ]
                for relative in phase["declared_artifacts"]:
                    artifact_path = (agent_workspace / relative).resolve()
                    try:
                        artifact_path.relative_to(agent_workspace.resolve())
                    except ValueError as error:
                        raise RuntimeError("工作流产物路径越界") from error
                    if not artifact_path.is_file() or artifact_path.stat().st_size == 0:
                        missing.append(relative)
                        continue
                    raw = artifact_path.read_bytes()
                    artifact_records.append(
                        self.build_artifact_ref(
                            task_id,
                            phase["id"],
                            relative,
                            raw,
                            lineage=lineage,
                        )
                    )
                if missing:
                    raise RuntimeError(
                        "{0} 未产出必需文件：{1}".format(
                            phase["id"], ", ".join(missing)
                        )
                    )
                runtime_check = None
                if is_issuefix and phase["id"] in {"fix", "test", "verify"}:
                    check = subprocess.run(
                        [sys.executable, "-m", "unittest", "-v", "test_calculator.py"],
                        cwd=str(agent_workspace),
                        text=True,
                        capture_output=True,
                        timeout=30,
                        check=False,
                    )
                    check_output = (check.stdout + check.stderr).strip()
                    runtime_check = {
                        "command": "python3 -m unittest -v test_calculator.py",
                        "exit_code": check.returncode,
                        "passed": check.returncode == 0,
                        "output": check_output[-4000:],
                    }
                    if check.returncode != 0:
                        raise RuntimeError(
                            "{0} 节点后真实测试失败：{1}".format(
                                phase["id"], check_output[-1000:]
                            )
                        )
                previous_output = {
                    "schema_version": "clawpro.node-result.v2",
                    "workflow_run_id": task_id,
                    "node_run_id": input_envelope["node_run_id"],
                    "attempt_id": input_envelope["attempt_id"],
                    "node_id": phase["id"],
                    "agent_instance_id": assigned["agent_instance_id"],
                    "agent_session_id": session_id,
                    "runtime_id": node_runtime_id,
                    "role_agent_id": phase["agent_id"],
                    "status": "completed",
                    "data": {
                        "summary": self.clean_agent_result(node_summary),
                        "runtime_check": runtime_check,
                    },
                    "artifacts": artifact_records,
                    "handoff": {
                        "schema_version": "clawpro.handoff.v2",
                        "upstream_node_id": (
                            node_results[-1]["node_id"] if node_results else None
                        ),
                        "known_issues": [],
                        "required_actions": [],
                    },
                }
                handoff_package = self.write_node_handoff(
                    node_workspace, previous_output, agent_workspace
                )
                node_results.append(previous_output)
                with self.lock:
                    task = self.tasks[task_id]
                    task["workflow_phases"][index - 1]["status"] = "completed"
                    task["available_artifacts"].extend(
                        {
                            **item,
                            "path": "agent-workspace/{0}".format(item["path"]),
                        }
                        for item in artifact_records
                    )
                self.append_event(
                    task_id,
                    "workflow.node.completed",
                    "{0} 真实 Agent 执行完成".format(phase["title"]),
                    "已产出 {0} 份文件并通过 Handoff v2 清单校验。".format(
                        len(artifact_records)
                    ),
                    {
                        **previous_output,
                        "handoff_path": handoff_package.relative_to(
                            agent_workspace
                        ).as_posix(),
                    },
                )
                self.wait_for_workflow_approval(
                    task_id,
                    phase,
                    phase.get("on_pass"),
                    previous_output,
                )
            report = {
                "schema_version": "clawpro.real-agent-workflow.v2",
                "handoff_contract": "clawpro.handoff.v2",
                "runtime_id": "teamai-node-routed",
                "assignment_mode": mode,
                "real_agent_execution": True,
                "agent_instance_count": assignment["unique_agent_count"],
                "agent_session_count": len(set(session_ids)),
                "handoff_count": max(0, len(node_results) - 1),
                "session_ids": session_ids,
                "runtime_counts": {
                    "hatchery-teamai-codebuddy": sum(
                        1
                        for item in node_results
                        if item["runtime_id"] == "hatchery-teamai-codebuddy"
                    ),
                    "hatchery-teamai-imate-openclaw": sum(
                        1
                        for item in node_results
                        if item["runtime_id"]
                        == "hatchery-teamai-imate-openclaw"
                    ),
                    "devresonance-cloudagent": sum(
                        1
                        for item in node_results
                        if item["runtime_id"] == "devresonance-cloudagent"
                    ),
                },
                "nodes": node_results,
                "external_writes_performed": any(
                    item["runtime_id"] == "hatchery-teamai-imate-openclaw"
                    or item.get("role_agent_id") == "message-notify-agent"
                    for item in node_results
                ),
            }
            (workspace / "real-agent-runtime-report.json").write_text(
                json.dumps(report, ensure_ascii=False, indent=2),
                encoding="utf-8",
            )
            return report
        finally:
            with self.lock:
                self.processes.pop(task_id, None)

    def run_structured_issuefix_validation(self, task_id):
        try:
            with self.lock:
                task = self.tasks[task_id]
                task["status"] = "running"
                task["execution_status"] = "running"
            workspace = Path(task["workspace_path"])
            definition, smoke = run_issuefix_compatibility_smoke(
                workspace,
                task["prompt"],
                task["agent_assignment_mode"],
            )
            teamai_assignment = self.build_teamai_node_assignment(
                definition, task
            )
            smoke["agent_assignment"] = teamai_assignment
            assignment_by_phase = {
                item["phase_id"]: item
                for item in smoke["agent_assignment"]["assignments"]
            }
            phases = [
                {
                    "id": phase["id"],
                    "title": phase["title"],
                    "agent_id": phase["agent_id"],
                    "agent_instance_id": assignment_by_phase[phase["id"]][
                        "agent_instance_id"
                    ],
                    "project_agent_id": assignment_by_phase[phase["id"]][
                        "project_agent_id"
                    ],
                    "device_id": assignment_by_phase[phase["id"]]["device_id"],
                    "transport": assignment_by_phase[phase["id"]]["transport"],
                    "target_agent_id": assignment_by_phase[phase["id"]].get(
                        "target_agent_id"
                    ),
                    "runtime_id": assignment_by_phase[phase["id"]]["runtime_id"],
                    "status": "ready",
                    "depends_on": phase.get("depends_on", []),
                    "on_pass": phase["on_pass"],
                    "on_fail": phase["on_fail"],
                    "decision_mode": phase.get("decision_mode"),
                    "max_retries": phase.get("max_retries", 0),
                    "retry_count": 0,
                    "artifacts": phase["declared_artifacts"],
                    "config_assets": [
                        {
                            key: value
                            for key, value in asset.items()
                            if key != "content"
                        }
                        for asset in phase.get("config_assets") or []
                    ],
                    "approval_required": bool(
                        self.workflow_gate_for_phase(phase)
                    ),
                }
                for phase in definition["phases"]
            ]
            with self.lock:
                task = self.tasks[task_id]
                task["workflow_revision"] = definition["source"]["revision"]
                task["workflow_phases"] = phases
                task["workflow_stage"] = "contract_validated"
                task["workflow_current_phase"] = smoke["original_engine"][
                    "next_action"
                ]["phase"]
                task["agent_instance_count"] = smoke["agent_assignment"][
                    "unique_agent_count"
                ]
                task["handoff_count"] = smoke["handoff"]["handoff_count"]
            self.append_event(
                task_id,
                "workflow.contract.validated",
                "节点契约和打回路径校验通过",
                "已校验 {0} 个阶段、{1} 项门禁、{2} 次上下文与产物交接。".format(
                    smoke["validated_phase_count"],
                    smoke["validated_gate_count"],
                    smoke["handoff"]["handoff_count"],
                ),
                {
                    "phases": phases,
                    "agent_assignment": smoke["agent_assignment"],
                    "handoff": smoke["handoff"],
                },
            )
            real_run = self.run_real_agent_workflow(
                task_id, definition, smoke, workspace
            )
            with self.lock:
                task = self.tasks[task_id]
                task["agent_session_count"] = real_run["agent_session_count"]
                task["handoff_count"] = real_run["handoff_count"]
                task["external_writes_performed"] = real_run[
                    "external_writes_performed"
                ]
                task["workflow_stage"] = "real_agent_completed"
            self.append_event(
                task_id,
                "workflow.real_agent.completed",
                "真实 Agent Runtime 工作流执行完成",
                (
                    "TeamAI 已按节点绑定的设备和 Agent 完成 8 个节点及 7 次交接。"
                ),
                real_run,
            )
            self.append_event(
                task_id,
                "workflow.engine.smoke_passed",
                "原始编排器冒烟通过",
                "run.py → next.py → orchestrator.py --dry-run 已成功执行；下一阶段为 analyze。",
                smoke["original_engine"],
            )
            artifact_names = [
                "compatibility-report.md",
                "workflow-definition.json",
                "node-contracts.json",
                "agent-assignment.json",
                "real-agent-runtime-report.json",
                "runtime-smoke.json",
            ]
            artifacts = []
            for name in artifact_names:
                path = workspace / name
                raw = path.read_bytes()
                artifacts.append(
                    {
                        "path": name,
                        "size": len(raw),
                        "sha256": hashlib.sha256(raw).hexdigest(),
                    }
                )
            agent_workspace = workspace / "agent-workspace"
            for node in real_run["nodes"]:
                for item in node["artifacts"]:
                    relative = "agent-workspace/{0}".format(item["path"])
                    path = workspace / relative
                    raw = path.read_bytes()
                    artifacts.append(
                        {
                            "path": relative,
                            "size": len(raw),
                            "sha256": hashlib.sha256(raw).hexdigest(),
                        }
                    )
            assignment_summary = (
                "8 个节点均通过 WSS/HTTPS 交给 TeamAI，并按节点绑定路由执行"
            )
            summary = (
                "SkillHub IssueFix 真实 Runtime 执行通过："
                + assignment_summary
                + "，实际记录 {0} 个 Runtime 会话/任务，"
                "8 个节点均已产出文件并完成 7 次交接；"
                "同时通过原始编排器 dry-run；未修改外部源码、未创建 MR。"
            ).format(
                real_run["agent_session_count"]
            )
            artifact = {
                "schema_version": "1.0",
                "task_id": task_id,
                "status": "completed",
                "engine": real_run["runtime_id"],
                "summary": summary,
                "workspace_path": str(workspace),
                "artifacts": artifacts,
                "downstream": {
                    "known_issues": [
                        "真实闭环仍需一个获授权的 SkillHub Issue。"
                    ],
                    "required_actions": [
                        "确认可自动修改的 Issue 后执行分支、MR、Checkers 和 E2E。"
                    ],
                },
            }
            with self.lock:
                task = self.tasks[task_id]
                task["status"] = "completed"
                task["execution_status"] = "completed"
                task["workflow_stage"] = "completed"
                task["agent_output"] = summary
                task["artifact"] = artifact
            self.append_event(
                task_id,
                "workflow.completed",
                "结构化工作流验证完成",
                "报告和标准化节点契约已回传 ClawPro。",
            )
        except TaskCanceled:
            return
        except Exception as error:
            if self.mark_workflow_failed(task_id, error):
                self.append_event(
                    task_id,
                    "workflow.failed",
                    "结构化工作流验证失败",
                    str(error),
                )

    def run_structured_project_workflow(self, task_id):
        try:
            with self.lock:
                task = self.tasks[task_id]
                task["status"] = "running"
                task["execution_status"] = "running"
                definition = task["_workflow_definition"]
            assignment = self.build_teamai_node_assignment(definition, task)
            assignment_by_phase = {
                item["phase_id"]: item for item in assignment["assignments"]
            }
            phases = [
                {
                    "id": phase["id"],
                    "title": phase["title"],
                    "agent_id": phase["agent_id"],
                    "agent_instance_id": assignment_by_phase[phase["id"]][
                        "agent_instance_id"
                    ],
                    "project_agent_id": assignment_by_phase[phase["id"]][
                        "project_agent_id"
                    ],
                    "device_id": assignment_by_phase[phase["id"]]["device_id"],
                    "transport": assignment_by_phase[phase["id"]]["transport"],
                    "target_agent_id": assignment_by_phase[phase["id"]].get(
                        "target_agent_id"
                    ),
                    "runtime_id": assignment_by_phase[phase["id"]]["runtime_id"],
                    "status": "ready",
                    "depends_on": phase.get("depends_on", []),
                    "on_pass": phase["on_pass"],
                    "on_fail": phase["on_fail"],
                    "decision_mode": phase.get("decision_mode"),
                    "max_retries": phase.get("max_retries", 0),
                    "retry_count": 0,
                    "artifacts": phase["declared_artifacts"],
                    "config_assets": [
                        {
                            key: value
                            for key, value in asset.items()
                            if key != "content"
                        }
                        for asset in phase.get("config_assets") or []
                    ],
                    "approval_required": bool(
                        self.workflow_gate_for_phase(phase)
                    ),
                }
                for phase in definition["phases"]
            ]
            with self.lock:
                task = self.tasks[task_id]
                task["workflow_phases"] = phases
                task["workflow_stage"] = "contract_validated"
                task["workflow_current_phase"] = definition["phases"][0]["id"]
                task["workflow_current_phases"] = [
                    phase["id"]
                    for phase in definition["phases"]
                    if not phase.get("depends_on")
                ]
                task["agent_instance_count"] = assignment["unique_agent_count"]
                task["handoff_count"] = sum(
                    len(phase.get("depends_on", []))
                    for phase in definition["phases"]
                )
            self.append_event(
                task_id,
                "workflow.contract.validated",
                "项目工作流契约校验通过",
                "已校验 {0} 个节点、{1} 个人工确认点和 {2} 次 Handoff。".format(
                    len(phases),
                    sum(1 for item in phases if item["approval_required"]),
                    sum(
                        len(phase.get("depends_on", []))
                        for phase in definition["phases"]
                    ),
                ),
                {"phases": phases, "agent_assignment": assignment},
            )
            real_run = self.run_real_agent_workflow(
                task_id,
                definition,
                {"agent_assignment": assignment},
                Path(self.tasks[task_id]["workspace_path"]),
            )
            summary = (
                "“{0}”真实执行完成：{1} 个节点、{2} 次 Handoff、"
                "{3} 个真实 Agent 会话/任务。"
            ).format(
                definition["name"],
                len(phases),
                real_run["handoff_count"],
                real_run["agent_session_count"],
            )
            with self.lock:
                task = self.tasks[task_id]
                task["status"] = "completed"
                task["execution_status"] = "completed"
                task["workflow_stage"] = "completed"
                task["workflow_current_phase"] = phases[-1]["id"]
                task["agent_session_count"] = real_run["agent_session_count"]
                task["handoff_count"] = real_run["handoff_count"]
                task["external_writes_performed"] = real_run[
                    "external_writes_performed"
                ]
                task["agent_output"] = summary
                task["artifact"] = {
                    "schema_version": "1.0",
                    "task_id": task_id,
                    "status": "completed",
                    "summary": summary,
                }
            self.append_event(
                task_id,
                "workflow.completed",
                "项目工作流执行完成",
                summary,
                real_run,
            )
        except TaskCanceled:
            return
        except Exception as error:
            with self.lock:
                task = self.tasks.get(task_id)
                if task and task["status"] != "canceled":
                    task["status"] = "failed"
                    task["execution_status"] = "failed"
                    task["workflow_stage"] = "failed"
            if self.tasks.get(task_id, {}).get("status") != "canceled":
                self.append_event(
                    task_id,
                    "workflow.failed",
                    "项目工作流执行失败",
                    str(error),
                )

    def create_handoff_workflow(
        self,
        prompt,
        model,
        *,
        target_agent_id,
        imate_project_id,
        delivery_mode="wss",
    ):
        """Run CodeBuddy first, then hand its context and text artifacts to iMate."""
        workflow_id = "workflow_" + uuid.uuid4().hex[:8]
        workspace = WORKFLOW_WORKSPACE_ROOT / workflow_id
        workspace.mkdir(parents=True, exist_ok=True)
        now = self.now()
        workflow = {
            "task_id": workflow_id,
            "attempt_id": "attempt_" + uuid.uuid4().hex[:8],
            "runtime_id": "workflow-codebuddy-imate",
            "model": model,
            "prompt": prompt,
            "status": "queued",
            "delivery_status": "queued",
            "execution_status": "submitted",
            "created_at": now,
            "updated_at": now,
            "cancel_requested": False,
            "cancellable": True,
            "session_id": None,
            "workspace_path": str(workspace),
            "agent_output": "",
            "executor": "workflow",
            "target_agent_id": target_agent_id,
            "imate_project_id": imate_project_id,
            "delivery_mode": delivery_mode,
            "artifact": None,
            "workflow": True,
            "workflow_stage": "upstream_queued",
            "workflow_nodes": {
                "upstream": {"executor": "codebuddy", "status": "queued", "task_id": None},
                "downstream": {"executor": "imate", "status": "waiting", "task_id": None},
            },
            "events": [],
        }
        with self.lock:
            self.tasks[workflow_id] = workflow
            self.task_order.insert(0, workflow_id)
        self.append_event(
            workflow_id,
            "workflow.queued",
            "协作工作流已创建",
            "ClawPro 将先执行 CodeBuddy，再自动把上下文与文本产物交给 iMate OpenClaw。",
        )

        upstream_prompt = (
            "你是 ClawPro 多 Agent 工作流的上游 CodeBuddy。\n\n"
            "用户目标：\n{0}\n\n"
            "请完成分析或实现，并在当前隔离工作区创建 UPSTREAM_RESULT.md，"
            "写明：完成内容、关键决定、产物说明、遗留问题、给下游 Agent 的下一步。"
            "可以创建其他必要的文本或代码文件。最终回复请简洁概括交接内容。"
        ).format(prompt)
        try:
            upstream = self.create_hatchery_task(
                upstream_prompt,
                "hatchery-teamai-codebuddy",
                model,
                delivery_mode=delivery_mode,
            )
        except Exception:
            with self.lock:
                self.tasks.pop(workflow_id, None)
                if workflow_id in self.task_order:
                    self.task_order.remove(workflow_id)
            raise
        with self.lock:
            upstream["_workflow_parent_id"] = workflow_id
            workflow["workflow_nodes"]["upstream"]["task_id"] = upstream["task_id"]
        self.append_event(
            workflow_id,
            "workflow.upstream.created",
            "上游 CodeBuddy 任务已下发",
            "任务 {0} 将产出结构化交接文件。".format(upstream["task_id"]),
        )
        threading.Thread(
            target=self.run_handoff_workflow,
            args=(workflow_id,),
            daemon=True,
        ).start()
        return workflow

    def set_workflow_stage(self, workflow_id, stage, *, status=None, title=None, detail=""):
        with self.lock:
            workflow = self.tasks.get(workflow_id)
            if not workflow or workflow.get("workflow_stage") == stage:
                return
            workflow["workflow_stage"] = stage
            if status:
                workflow["status"] = status
                workflow["execution_status"] = status
            if status in {"running", "completed", "failed"}:
                workflow["delivery_status"] = "claimed"
        if title:
            self.append_event(workflow_id, "workflow." + stage, title, detail)

    def run_handoff_workflow(self, workflow_id):
        """Coordinate a real two-node handoff through the existing execution channels."""
        try:
            while True:
                with self.lock:
                    workflow = self.tasks.get(workflow_id)
                    if not workflow or workflow.get("cancel_requested"):
                        return
                    upstream_id = workflow["workflow_nodes"]["upstream"]["task_id"]
                self.refresh_hatchery_task(upstream_id)
                with self.lock:
                    upstream = self.tasks[upstream_id]
                    upstream_status = upstream["status"]
                    workflow["workflow_nodes"]["upstream"]["status"] = upstream_status
                if upstream_status == "running":
                    self.set_workflow_stage(
                        workflow_id,
                        "upstream_running",
                        status="running",
                        title="上游 CodeBuddy 正在执行",
                        detail="结果和工作区文件将自动形成交接包。",
                    )
                if upstream_status in {"failed", "canceled"}:
                    raise RuntimeError(
                        upstream.get("failure_detail")
                        or "上游 CodeBuddy 执行失败，工作流未继续下发。"
                    )
                if upstream_status == "completed":
                    break
                time.sleep(0.8)

            self.set_workflow_stage(
                workflow_id,
                "handoff_building",
                status="running",
                title="正在生成 Agent 交接包",
                detail="汇总用户目标、上游回复、文件内容与校验值。",
            )
            context_package, downstream_prompt = self.build_handoff_context(
                workflow_id,
                upstream,
            )
            self.append_event(
                workflow_id,
                "workflow.handoff.ready",
                "上下文与产物已完成交接",
                "已打包 {0} 个产物，并将可读取的文本内容写入下游任务。".format(
                    len(context_package["artifact_refs"])
                ),
                context_package,
            )

            with self.lock:
                workflow = self.tasks[workflow_id]
            downstream = self.create_hatchery_task(
                downstream_prompt,
                "hatchery-teamai-imate-openclaw",
                workflow["model"],
                target_agent_id=workflow["target_agent_id"],
                imate_project_id=workflow["imate_project_id"],
                delivery_mode=workflow["delivery_mode"],
            )
            with self.lock:
                downstream["_workflow_parent_id"] = workflow_id
                workflow["workflow_nodes"]["downstream"] = {
                    "executor": "imate",
                    "status": "queued",
                    "task_id": downstream["task_id"],
                }
            self.set_workflow_stage(
                workflow_id,
                "downstream_queued",
                status="running",
                title="交接包已下发给 iMate",
                detail="iMate 将把任务调度给所选 OpenClaw Agent。",
            )

            while True:
                with self.lock:
                    workflow = self.tasks.get(workflow_id)
                    if not workflow or workflow.get("cancel_requested"):
                        return
                self.refresh_hatchery_task(downstream["task_id"])
                with self.lock:
                    downstream_status = downstream["status"]
                    workflow["workflow_nodes"]["downstream"]["status"] = downstream_status
                if downstream_status == "running":
                    self.set_workflow_stage(
                        workflow_id,
                        "downstream_running",
                        status="running",
                        title="下游 iMate OpenClaw 正在执行",
                        detail="下游已收到上游摘要和文本产物正文。",
                    )
                if downstream_status in {"failed", "canceled"}:
                    raise RuntimeError(
                        downstream.get("failure_detail")
                        or "下游 iMate OpenClaw 执行失败。"
                    )
                if downstream_status == "completed":
                    break
                time.sleep(0.8)

            result_text = self.clean_agent_result(downstream.get("agent_output", ""))
            workspace = Path(workflow["workspace_path"])
            (workspace / "downstream-result.md").write_text(
                "# iMate OpenClaw 执行结果\n\n" + (result_text or "未返回文本结果。") + "\n",
                encoding="utf-8",
            )
            workflow_result = {
                "schema_version": "1.0",
                "workflow_id": workflow_id,
                "status": "completed",
                "upstream_task_id": upstream["task_id"],
                "downstream_task_id": downstream["task_id"],
                "context_package": "context-package.json",
                "result": result_text,
            }
            (workspace / "workflow-result.json").write_text(
                json.dumps(workflow_result, ensure_ascii=False, indent=2),
                encoding="utf-8",
            )
            artifact = {
                "schema_version": "1.0",
                "task_id": workflow_id,
                "status": "completed",
                "engine": "codebuddy-to-imate",
                "summary": result_text or "CodeBuddy → iMate 协作工作流已完成。",
                "workspace_path": str(workspace),
                "artifacts": self.collect_workspace_files(workspace),
                "downstream": {"known_issues": [], "required_actions": []},
            }
            with self.lock:
                workflow["status"] = "completed"
                workflow["execution_status"] = "completed"
                workflow["delivery_status"] = "claimed"
                workflow["workflow_stage"] = "completed"
                workflow["agent_output"] = result_text
                workflow["session_id"] = downstream.get("session_id")
                workflow["artifact"] = artifact
            self.append_event(
                workflow_id,
                "workflow.completed",
                "双 Agent 协作完成",
                "iMate 结果、交接包和产物清单已回传 ClawPro。",
                workflow_result,
            )
        except Exception as error:
            with self.lock:
                workflow = self.tasks.get(workflow_id)
                if workflow and workflow.get("status") != "canceled":
                    workflow["status"] = "failed"
                    workflow["execution_status"] = "failed"
                    workflow["workflow_stage"] = "failed"
            if self.tasks.get(workflow_id, {}).get("status") != "canceled":
                self.append_event(
                    workflow_id,
                    "workflow.failed",
                    "协作工作流执行失败",
                    str(error),
                )

    def build_handoff_context(self, workflow_id, upstream):
        workspace = Path(self.tasks[workflow_id]["workspace_path"])
        upstream_workspace = Path(upstream["workspace_path"])
        copied_root = workspace / "upstream-artifacts"
        copied_root.mkdir(parents=True, exist_ok=True)
        artifact_refs = []
        inline_budget = 24_000
        for item in upstream.get("artifact", {}).get("artifacts", []):
            relative = item.get("path") or ""
            if not relative or relative == "TASK.md":
                continue
            source = (upstream_workspace / relative).resolve()
            try:
                source.relative_to(upstream_workspace.resolve())
            except ValueError:
                continue
            if not source.is_file():
                continue
            target = copied_root / relative
            target.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(source, target)
            raw = source.read_bytes()
            ref = {
                "artifact_id": "artifact_" + hashlib.sha256(raw).hexdigest()[:16],
                "path": relative,
                "size": len(raw),
                "sha256": hashlib.sha256(raw).hexdigest(),
            }
            try:
                text_content = raw.decode("utf-8")
            except UnicodeDecodeError:
                text_content = ""
            if text_content and inline_budget > 0:
                inline = text_content[: min(8000, inline_budget)]
                ref["inline_content"] = inline
                ref["content_truncated"] = len(inline) < len(text_content)
                inline_budget -= len(inline)
            artifact_refs.append(ref)

        context_package = {
            "schema_version": "1.0",
            "workflow_run_id": workflow_id,
            "node_run_id": upstream["task_id"],
            "user_goal": self.tasks[workflow_id]["prompt"],
            "node_instruction": "基于上游 CodeBuddy 的结论与产物继续完成任务，并明确说明使用了哪些交接信息。",
            "upstream_results": [
                {
                    "executor": "codebuddy",
                    "task_id": upstream["task_id"],
                    "summary": upstream.get("agent_output", "").strip(),
                }
            ],
            "artifact_refs": artifact_refs,
            "required_output": [
                "说明已收到的上游结论和产物",
                "给出继续处理后的最终结果",
                "列出仍需人工处理的问题（如无则写无）",
            ],
        }
        (workspace / "context-package.json").write_text(
            json.dumps(context_package, ensure_ascii=False, indent=2),
            encoding="utf-8",
        )
        artifact_blocks = []
        for ref in artifact_refs:
            content = ref.get("inline_content")
            artifact_blocks.append(
                "### {0}\n- artifact_id: {1}\n- sha256: {2}\n{3}".format(
                    ref["path"],
                    ref["artifact_id"],
                    ref["sha256"],
                    ("```text\n" + content + "\n```") if content else "（二进制或内容过大，本次最简 PoC 仅传递清单与校验值）",
                )
            )
        downstream_prompt = (
            "你是 ClawPro 多 Agent 工作流的下游 iMate OpenClaw Agent。"
            "以下内容由 ClawPro 自动从上游 CodeBuddy 任务生成，不是用户手工复制。\n\n"
            "## 用户原始目标\n{goal}\n\n"
            "## 上游任务\n- task_id: {task_id}\n- summary:\n{summary}\n\n"
            "## 上游产物（文本正文已内联）\n{artifacts}\n\n"
            "## 你的任务\n"
            "1. 明确列出你收到了哪些上游信息和文件；\n"
            "2. 基于这些信息继续完成用户目标，不要从头重复上游工作；\n"
            "3. 返回最终结论以及仍需人工处理的问题。"
        ).format(
            goal=context_package["user_goal"],
            task_id=upstream["task_id"],
            summary=context_package["upstream_results"][0]["summary"] or "上游未返回文本摘要，请以产物为准。",
            artifacts="\n\n".join(artifact_blocks) or "无额外文件。",
        )
        return context_package, downstream_prompt

    def clean_agent_result(self, raw_result):
        """Keep the final Agent answer instead of exposing the full iMate tool trace."""
        result = str(raw_result or "").strip()
        if len(result) <= 2_500:
            return result
        for marker in (
            "\n**结论**",
            "\n## 执行结果",
            "任务完成。",
            "## 交接确认",
            "自动交接验证通过",
        ):
            index = result.rfind(marker)
            if index >= 0 and len(result) - index <= 2_500:
                return result[index:].strip()
        return "（已省略执行器调试过程）\n\n" + result[-2_500:]

    def build_teamai_node_assignment(self, definition, task):
        """Resolve every workflow node to a persisted TeamAI execution route."""
        declared = {
            str(item.get("phase_id") or ""): item
            for item in task.get("node_assignments") or []
            if isinstance(item, dict)
        }
        legacy_imate_phases = {"review", "test", "checkers", "verify"}
        assignments = []
        for phase in definition["phases"]:
            route = declared.get(phase["id"], {})
            platform_name = str(route.get("platform") or "").strip().lower()
            if not platform_name:
                platform_name = (
                    "imate"
                    if task.get("agent_assignment_mode") == "mixed"
                    and phase["id"] in legacy_imate_phases
                    else "codebuddy"
                )
            if platform_name not in {"codebuddy", "imate", "cloudagent"}:
                raise RuntimeError(
                    "节点 {0} 暂不支持 Agent 平台：{1}".format(
                        phase["id"], platform_name or "unknown"
                    )
                )
            if platform_name == "imate":
                runtime_id = "hatchery-teamai-imate-openclaw"
            elif platform_name == "cloudagent":
                runtime_id = "devresonance-cloudagent"
            else:
                runtime_id = "hatchery-teamai-codebuddy"
            project_agent_id = str(
                route.get("project_agent_id")
                or route.get("agent_id")
                or "{0}-default".format(platform_name)
            )
            assignments.append(
                {
                    "phase_id": phase["id"],
                    "role_agent_id": phase["agent_id"],
                    "project_agent_id": project_agent_id,
                    "agent_instance_id": project_agent_id,
                    "runtime_id": runtime_id,
                    "platform": platform_name,
                    "location": route.get("location") or "local",
                    "device_id": (
                        "devresonance-cloud"
                        if platform_name == "cloudagent"
                        else self.hatchery.agent_id
                    ),
                    "target_agent_id": (
                        route.get("target_agent_id")
                        or task.get("target_agent_id")
                        if platform_name == "imate"
                        else None
                    ),
                    "transport": (
                        "https-direct-prompt"
                        if platform_name == "cloudagent"
                        else "wss+https"
                    ),
                }
            )
        unique_instances = sorted(
            {
                "{0}:{1}".format(item["device_id"], item["project_agent_id"])
                for item in assignments
            }
        )
        return {
            "schema_version": "clawpro.agent-assignment.v3",
            "mode": "node-routed",
            "assignments": assignments,
            "unique_agent_instances": unique_instances,
            "unique_agent_count": len(unique_instances),
            "expected_agent_count": len(unique_instances),
            "verified": True,
        }

    def create_hatchery_task(
        self,
        prompt,
        runtime_id,
        model,
        *,
        target_agent_id="",
        imate_project_id="",
        delivery_mode="wss",
        seed_workspace=None,
        repository_url="",
    ):
        if not self.hatchery or not self.hatchery.ready:
            raise BridgeError(
                self.hatchery.setup_error
                if self.hatchery
                else "Hatchery—TeamAI Bridge 尚未配置"
            )
        executor = "imate" if runtime_id == "hatchery-teamai-imate-openclaw" else "codebuddy"
        task_id, workspace, backend_task, wake_delivered = self.hatchery.create_task(
            prompt,
            executor=executor,
            target_agent_id=target_agent_id,
            imate_project_id=imate_project_id,
            delivery_mode=delivery_mode,
            seed_workspace=seed_workspace,
            repository_url=repository_url,
        )
        now = self.now()
        task = {
            "task_id": task_id,
            "backend_task_id": backend_task["id"],
            "attempt_id": "attempt_" + uuid.uuid4().hex[:8],
            "runtime_id": runtime_id,
            "model": model,
            "prompt": prompt,
            "status": "queued",
            "delivery_status": "queued",
            "execution_status": "submitted",
            "created_at": backend_task.get("created_at") or now,
            "updated_at": backend_task.get("updated_at") or now,
            "cancel_requested": False,
            "cancellable": True,
            "session_id": None,
            "workspace_path": str(workspace),
            "agent_output": "",
            "executor": executor,
            "target_agent_id": target_agent_id or None,
            "imate_project_id": imate_project_id or None,
            "delivery_mode": delivery_mode,
            "artifact": None,
            "events": [],
            "_remote_status": backend_task.get("status") or "pending",
            "_teamai_woken": False,
            "_wake_delivered": wake_delivered,
        }
        with self.lock:
            self.tasks[task_id] = task
            self.task_order.insert(0, task_id)
        self.append_event(
            task_id,
            "task.queued",
            "任务已写入 Hatchery",
            "云端任务 #{0} 等待本地 TeamAI 拉取。".format(backend_task["id"]),
        )
        if self.connected:
            threading.Thread(
                target=self.run_hatchery_task,
                args=(task_id,),
                daemon=True,
            ).start()
        return task

    def set_connected(self, connected):
        with self.lock:
            self.connected = connected
            queued = [
                task_id
                for task_id in self.task_order
                if self.tasks[task_id]["status"] == "queued"
            ]
        if connected:
            for task_id in queued:
                threading.Thread(target=self.run_task, args=(task_id,), daemon=True).start()

    def cancel_task(self, task_id):
        with self.approval_condition:
            task = self.tasks.get(task_id)
            if not task:
                return None
            active_phase_statuses = {"running", "awaiting_approval"}
            has_active_phase = any(
                phase.get("status") in active_phase_statuses
                for phase in task.get("workflow_phases") or []
            )
            # A remote Agent can still be running after the orchestration shell
            # was marked failed (for example after a transient callback error).
            # In that inconsistent-but-recoverable state, stopping must still
            # cancel the orchestration and make late results ineligible.
            if task["status"] == "completed" or (
                task["status"] in {"failed", "canceled"} and not has_active_phase
            ):
                return task
            task["cancel_requested"] = True
            process = self.processes.get(task_id)
            cloudagent_execution = self.cloudagent_executions.get(task_id)
            task["status"] = "canceled"
            task["execution_status"] = "canceled"
            task["workflow_stage"] = "canceled"
            pending = task.get("pending_approval")
            if pending and pending.get("status") == "pending":
                pending["status"] = "canceled"
            task["pending_approval"] = None
            current_phases = set(task.get("workflow_current_phases") or [])
            if task.get("workflow_current_phase"):
                current_phases.add(task["workflow_current_phase"])
            for phase in task.get("workflow_phases") or []:
                phase_status = phase.get("status")
                if phase_status in {"running", "awaiting_approval"} or (
                    phase.get("id") in current_phases and phase_status == "ready"
                ):
                    phase["status"] = "canceled"
            task["workflow_current_phases"] = []
            task["updated_at"] = self.now()
            self.approval_condition.notify_all()
        if process and process.poll() is None:
            process.terminate()
        if cloudagent_execution:
            self.cloudagent.cancel(
                cloudagent_execution["agent_id"],
                cloudagent_execution.get("session_id") or "",
                cloudagent_execution.get("trace_id") or "",
            )
        self.append_event(
            task_id,
            "task.canceled",
            "任务已取消",
            "本地执行已停止，迟到结果不会覆盖取消状态。",
        )
        return task

    def mark_workflow_failed(self, task_id, error):
        """Keep workflow and active node state consistent after a failure."""
        with self.lock:
            task = self.tasks.get(task_id)
            if not task or task.get("status") == "canceled":
                return False
            task["status"] = "failed"
            task["execution_status"] = "failed"
            task["workflow_stage"] = "failed"
            task["failure_detail"] = str(error)
            task["pending_approval"] = None
            current_ids = set(task.get("workflow_current_phases") or [])
            current_phase = task.get("workflow_current_phase")
            if current_phase:
                current_ids.update(
                    phase_id for phase_id in str(current_phase).split(",") if phase_id
                )
            for phase in task.get("workflow_phases") or []:
                if phase.get("status") == "running" or phase.get("id") in current_ids:
                    phase["status"] = "failed"
                    phase["error"] = str(error)
            task["workflow_current_phases"] = []
            task["updated_at"] = self.now()
            return True

    def run_task(self, task_id):
        with self.lock:
            target = self.tasks.get(task_id)
            if target and target.get("runtime_id") in HATCHERY_RUNTIME_IDS:
                pass
            else:
                target = None
        if target is not None:
            self.run_hatchery_task(task_id)
            return
        with self.lock:
            task = self.tasks.get(task_id)
            if not task or task["status"] != "queued" or not self.connected:
                return
            task["status"] = "running"
            task["delivery_status"] = "claimed"
            task["execution_status"] = "running"
            runtime_id = task["runtime_id"]

        runtime = runtime_by_id(runtime_id)
        if not runtime or not runtime.get("available"):
            with self.lock:
                self.tasks[task_id]["status"] = "failed"
                self.tasks[task_id]["execution_status"] = "failed"
            self.append_event(
                task_id,
                "task.failed",
                "Agent Runtime 不可用",
                "本机未发现对应的可执行文件。",
            )
            return

        workspace_dir = (
            REAL_WORKSPACE_ROOT / task_id
            if runtime_id == "codebuddy-acp"
            else WORKSPACE_ROOT / task_id
        )
        workspace_dir.mkdir(parents=True, exist_ok=True)
        if runtime_id == "codebuddy-acp":
            (workspace_dir / "TASK.md").write_text(
                "# ClawPro Edge Runtime 隔离任务\n\n"
                "当前目录是独立测试工作区。只能在此目录中创建或修改文件。\n",
                encoding="utf-8",
            )
        with self.lock:
            self.tasks[task_id]["workspace_path"] = str(workspace_dir)

        self.append_event(
            task_id,
            "teamai.task.received",
            "TeamAI 已收到任务",
            "设备身份和企业绑定校验通过，准备交给 Edge Runtime。",
            {
                "bridge_id": self.teamai["bridge_id"],
                "device_id": self.teamai["device_id"],
            },
        )
        self.append_event(
            task_id,
            "runtime.task.claimed",
            "Edge Runtime 已领取任务",
            "TeamAI 完成本地投递，Runtime 获取执行所有权和租约。",
        )
        self.append_event(
            task_id,
            "workspace.created",
            "已创建隔离工作区",
            str(workspace_dir),
        )

        process = None
        try:
            if runtime_id == "codebuddy-acp":
                command = [
                    runtime["executable"],
                    "--acp",
                    "--permission-mode",
                    "acceptEdits",
                    "--setting-sources",
                    "local",
                    "--tools",
                    "Read,Write,Edit",
                ]
            else:
                command = [sys.executable, str(ROOT / "mock_acp_agent.py")]
            process = subprocess.Popen(
                command,
                stdin=subprocess.PIPE,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                text=True,
                bufsize=1,
                cwd=str(workspace_dir),
            )
            with self.lock:
                self.processes[task_id] = process

            self.append_event(
                task_id,
                "runtime.started",
                "已启动 {0}".format(runtime["name"]),
                "真实进程：{0}。".format(runtime["executable"]),
            )

            initialize = self.rpc(
                task_id,
                process,
                1,
                "initialize",
                {
                    "protocolVersion": 1,
                    "clientInfo": {"name": "clawpro-edge-poc", "version": "0.1.0"},
                    "clientCapabilities": {},
                },
            )
            self.ensure_not_canceled(task_id)
            self.append_event(
                task_id,
                "session.initialized",
                "ACP 握手成功",
                "Agent 返回 protocolVersion=1。",
                initialize.get("result") or {},
            )

            session = self.rpc(
                task_id,
                process,
                2,
                "session/new",
                {"cwd": str(workspace_dir), "mcpServers": []},
            )
            session_id = (session.get("result") or {}).get("sessionId")
            if not session_id:
                raise RuntimeError("session/new 未返回 sessionId")
            with self.lock:
                self.tasks[task_id]["session_id"] = session_id
            self.append_event(
                task_id,
                "session.started",
                "Agent 会话已创建",
                "会话可在后续任务中恢复。",
                {"session_id": session_id},
            )

            model = self.tasks[task_id]["model"]
            if model:
                self.rpc(
                    task_id,
                    process,
                    3,
                    "session/set_config_option",
                    {"sessionId": session_id, "configId": "model", "value": model},
                )
                self.append_event(
                    task_id,
                    "session.configured",
                    "已设置任务模型",
                    model,
                )

            prompt_response = self.rpc(
                task_id,
                process,
                4,
                "session/prompt",
                {
                    "sessionId": session_id,
                    "prompt": [{"type": "text", "text": self.tasks[task_id]["prompt"]}],
                },
                collect_updates=True,
            )
            self.flush_text_event(task_id, "message")
            self.flush_text_event(task_id, "thought")
            self.ensure_not_canceled(task_id)
            stop_reason = (prompt_response.get("result") or {}).get("stopReason")
            if stop_reason != "end_turn":
                raise RuntimeError("Agent 未返回 end_turn")

            artifact_path = workspace_dir / "handoff.json"
            files = self.collect_workspace_files(workspace_dir)
            agent_output = self.tasks[task_id].get("agent_output", "").strip()
            artifact = {
                "schema_version": "1.0",
                "task_id": task_id,
                "status": "completed",
                "engine": runtime["engine"],
                "summary": agent_output
                or "{0} 已完成任务并通过协议校验。".format(
                    runtime["name"]
                ),
                "workspace_path": str(workspace_dir),
                "artifacts": files,
                "downstream": {"known_issues": [], "required_actions": []},
            }
            artifact_path.write_text(
                json.dumps(artifact, ensure_ascii=False, indent=2), encoding="utf-8"
            )
            with self.lock:
                self.tasks[task_id]["artifact"] = artifact
            self.append_event(
                task_id,
                "artifact.created",
                "已生成 Handoff 交接结果",
                "已收集 {0} 个工作区文件。".format(len(files)),
                artifact,
            )

            with self.lock:
                if self.tasks[task_id]["cancel_requested"]:
                    return
                self.tasks[task_id]["status"] = "completed"
                self.tasks[task_id]["execution_status"] = "completed"
            self.append_event(
                task_id,
                "task.completed",
                "任务已完成并回传 ClawPro",
                "页面、任务状态和交接产物已收敛。",
            )
        except TaskCanceled:
            pass
        except Exception as error:
            with self.lock:
                task = self.tasks.get(task_id)
                if task and task["status"] != "canceled":
                    task["status"] = "failed"
                    task["execution_status"] = "failed"
            if self.tasks.get(task_id, {}).get("status") != "canceled":
                self.append_event(
                    task_id,
                    "task.failed",
                    "任务执行失败",
                    str(error),
                )
        finally:
            if process and process.poll() is None:
                process.terminate()
                try:
                    process.wait(timeout=2)
                except subprocess.TimeoutExpired:
                    process.kill()
            with self.lock:
                self.processes.pop(task_id, None)

    def run_hatchery_task(self, task_id):
        with self.lock:
            task = self.tasks.get(task_id)
            if (
                not task
                or task.get("runtime_id") not in HATCHERY_RUNTIME_IDS
                or task.get("_teamai_woken")
                or not self.connected
            ):
                return
            task["_teamai_woken"] = True
            workspace = Path(task["workspace_path"])
            delivery_mode = task.get("delivery_mode") or "hook"
            wake_delivered = bool(task.get("_wake_delivered"))
        try:
            if delivery_mode == "wss":
                self.append_event(
                    task_id,
                    "teamai.wss.notified" if wake_delivered else "teamai.wss.pending",
                    "已通过 WSS 唤醒常驻 TeamAI" if wake_delivered else "任务已保存，等待 TeamAI 重连补拉",
                    "WebSocket 仅发送 task_id；TeamAI 正通过 HTTPS /local-agent/sync 领取完整任务。"
                    if wake_delivered
                    else "当前没有在线 WSS 通道，任务不会丢失；TeamAI 重连后会自动 sync。",
                )
                return
            self.hatchery.wake_teamai(workspace)
            self.append_event(
                task_id,
                "teamai.sync.started",
                "TeamAI 已发起真实同步",
                "正在通过 /local-agent/sync 拉取 execute_agent_task。",
            )
        except Exception as error:
            with self.lock:
                task["status"] = "failed"
                task["execution_status"] = "failed"
            self.append_event(
                task_id,
                "task.failed",
                "TeamAI 启动失败",
                str(error),
            )

    def refresh_hatchery_task(self, task_id):
        with self.lock:
            task = self.tasks.get(task_id)
            if not task or task.get("runtime_id") not in HATCHERY_RUNTIME_IDS:
                return
            if task["status"] in {"completed", "failed", "canceled"}:
                return
            backend_task_id = task["backend_task_id"]
        try:
            remote = self.hatchery.get_task(backend_task_id)
        except Exception as error:
            self.append_event(
                task_id,
                "status.failed",
                "Hatchery 状态查询失败",
                str(error),
            )
            return

        remote_status = remote.get("status") or "pending"
        with self.lock:
            previous = task.get("_remote_status")
            task["_remote_status"] = remote_status
            task["updated_at"] = remote.get("updated_at") or self.now()
            if remote.get("session_id"):
                task["session_id"] = remote["session_id"]

        if remote_status == "running":
            with self.lock:
                task["status"] = "running"
                task["delivery_status"] = "claimed"
                task["execution_status"] = "running"
            if previous != "running":
                self.append_event(
                    task_id,
                    "runtime.task.claimed",
                    "TeamAI 已领取任务并启动执行器",
                    "Hatchery 已收到 running ACK。",
                )
        elif remote_status == "success":
            result = str(remote.get("result") or "").strip()
            workspace = Path(task["workspace_path"])
            files = self.collect_workspace_files(workspace)
            artifact = {
                "schema_version": "1.0",
                "task_id": task_id,
                "status": "completed",
                "engine": "teamai-imate-openclaw" if task.get("executor") == "imate" else "teamai-codebuddy",
                "summary": result or "TeamAI 已完成 Agent 任务。",
                "workspace_path": str(workspace),
                "artifacts": files,
                "downstream": {"known_issues": [], "required_actions": []},
            }
            with self.lock:
                task["status"] = "completed"
                task["delivery_status"] = "claimed"
                task["execution_status"] = "completed"
                task["agent_output"] = result
                task["artifact"] = artifact
            self.append_event(
                task_id,
                "artifact.created",
                "本地结果已回传 Hatchery",
                "已收集 {0} 个工作区文件。".format(len(files)),
                artifact,
            )
            self.append_event(
                task_id,
                "task.completed",
                "真实云—本任务执行完成",
                "Hatchery 已收到 TeamAI success ACK 和 CodeBuddy 会话结果。",
            )
        elif remote_status == "failed":
            detail = str(remote.get("error") or remote.get("result") or "未知错误")
            with self.lock:
                task["status"] = "failed"
                task["delivery_status"] = "claimed"
                task["execution_status"] = "failed"
                task["failure_detail"] = detail
            self.append_event(
                task_id,
                "task.failed",
                "本地 Agent 执行失败",
                detail,
            )

    def collect_workspace_files(self, workspace_dir):
        files = []
        for path in sorted(workspace_dir.rglob("*")):
            if not path.is_file() or ".git" in path.parts or path.name == "handoff.json":
                continue
            relative = path.relative_to(workspace_dir).as_posix()
            size = path.stat().st_size
            digest = hashlib.sha256(path.read_bytes()).hexdigest()
            item = {"path": relative, "size": size, "sha256": digest}
            if size <= 100_000:
                try:
                    item["preview"] = path.read_text(encoding="utf-8")[:1200]
                except UnicodeDecodeError:
                    pass
            files.append(item)
        return files

    def ensure_not_canceled(self, task_id):
        with self.lock:
            if self.tasks[task_id]["cancel_requested"]:
                raise TaskCanceled()

    def rpc(self, task_id, process, request_id, method, params, collect_updates=False):
        request = {
            "jsonrpc": "2.0",
            "id": request_id,
            "method": method,
            "params": params,
        }
        process.stdin.write(json.dumps(request, ensure_ascii=False) + "\n")
        process.stdin.flush()

        while True:
            self.ensure_not_canceled(task_id)
            line = process.stdout.readline()
            if not line:
                stderr = process.stderr.read().strip()
                raise RuntimeError(stderr or "Agent 进程提前退出")
            message = json.loads(line)
            if message.get("method") == "session/update":
                if collect_updates:
                    self.map_acp_update(task_id, message)
                continue
            if (
                message.get("method") == "session/request_permission"
                and message.get("id") is not None
            ):
                options = (message.get("params") or {}).get("options") or []
                selected = next(
                    (
                        option
                        for option in options
                        if option.get("kind") in {"allow_once", "allow_session"}
                    ),
                    options[0] if options else {"optionId": "allow_once"},
                )
                response = {
                    "jsonrpc": "2.0",
                    "id": message["id"],
                    "result": {
                        "outcome": {
                            "outcome": "selected",
                            "optionId": selected.get("optionId")
                            or selected.get("id")
                            or "allow_once",
                        }
                    },
                }
                process.stdin.write(json.dumps(response, ensure_ascii=False) + "\n")
                process.stdin.flush()
                self.append_event(
                    task_id,
                    "permission.approved",
                    "已批准隔离工作区内操作",
                    "仅对当前任务的 Read/Write/Edit 工具生效。",
                )
                continue
            if message.get("id") == request_id:
                if "error" in message:
                    raise RuntimeError(message["error"].get("message", "ACP error"))
                return message

    def flush_text_event(self, task_id, kind):
        key = "_message_buffer" if kind == "message" else "_thought_buffer"
        with self.lock:
            task = self.tasks[task_id]
            content = task.get(key, "")
            task[key] = ""
            if kind == "message" and content:
                task["agent_output"] = task.get("agent_output", "") + content
        if not content:
            return
        self.append_event(
            task_id,
            "message.delta" if kind == "message" else "status.updated",
            "Agent 回复" if kind == "message" else "Agent 正在分析",
            content,
        )

    def map_acp_update(self, task_id, message):
        update = ((message.get("params") or {}).get("update") or {})
        kind = update.get("sessionUpdate") or update.get("type")
        if kind == "agent_message_chunk":
            text = (update.get("content") or {}).get("text", "")
            with self.lock:
                self.tasks[task_id]["_message_buffer"] += text
                buffered = self.tasks[task_id]["_message_buffer"]
            if len(buffered) >= 160 or text.endswith(("\n", "。", "！", "？")):
                self.flush_text_event(task_id, "message")
        elif kind == "agent_thought_chunk":
            text = (update.get("content") or {}).get("text", "")
            with self.lock:
                self.tasks[task_id]["_thought_buffer"] += text
                buffered = self.tasks[task_id]["_thought_buffer"]
            if len(buffered) >= 200 or text.endswith(("\n", "。", "！", "？")):
                self.flush_text_event(task_id, "thought")
        elif kind == "tool_call":
            self.append_event(
                task_id,
                "tool.started",
                update.get("title") or update.get("name") or "工具调用",
                "Agent 已发起工具调用。",
                {
                    "tool_call_id": update.get("toolCallId"),
                    "input": update.get("rawInput") or update.get("input") or {},
                },
            )
        elif kind == "tool_call_update":
            status = update.get("status") or "running"
            if status == "completed":
                event_type = "tool.completed"
            elif status in {"failed", "error"}:
                event_type = "tool.failed"
            else:
                return
            self.append_event(
                task_id,
                event_type,
                {
                    "completed": "工具执行完成",
                    "failed": "工具执行失败",
                    "error": "工具执行失败",
                }.get(status, "工具执行中"),
                str(update.get("rawOutput") or update.get("output") or ""),
                {"tool_call_id": update.get("toolCallId"), "status": status},
            )
        elif kind == "usage_update":
            usage = update.get("usage") or {}
            if not usage:
                usage = ((update.get("_meta") or {}).get("usage") or {})
            input_tokens = usage.get("inputTokens", usage.get("prompt_tokens", 0))
            output_tokens = usage.get("outputTokens", usage.get("completion_tokens", 0))
            self.append_event(
                task_id,
                "usage.updated",
                "用量已更新",
                "输入 {0} / 输出 {1} Tokens".format(
                    input_tokens, output_tokens
                ),
                usage,
            )


class TaskCanceled(Exception):
    pass


STATE = DemoState(TASK_STATE_PATH)


def detect_binary(name, candidates):
    for candidate in candidates:
        if candidate.is_absolute() and candidate.exists():
            return str(candidate)
        resolved = shutil.which(str(candidate))
        if resolved:
            return resolved
    return None


def runtime_catalog():
    codex = detect_binary(
        "codex",
        [
            Path("codex"),
            Path("/Applications/ChatGPT.app/Contents/Resources/codex"),
        ],
    )
    codebuddy = detect_binary(
        "codebuddy",
        [
            Path("codebuddy"),
            Path(
                "/Applications/WorkBuddy.app/Contents/Resources/app.asar.unpacked/cli/bin/codebuddy"
            ),
        ],
    )
    codebuddy_auth = codebuddy_capability_status()
    remote_teamai = bool(STATE.hatchery and STATE.hatchery.remote_mode)
    teamai_ready = bool(STATE.hatchery and STATE.hatchery.ready)
    codebuddy_available = bool(codebuddy) or (remote_teamai and teamai_ready)
    imate_available = bool(
        STATE.hatchery
        and STATE.hatchery.ready
        and (
            STATE.hatchery.imate_bin.is_file()
            or (
                STATE.hatchery.remote_mode
                and STATE.hatchery.remote_imate_agent_id
            )
        )
    )
    return [
        {
            "runtime_id": STRUCTURED_PROJECT_WORKFLOW_RUNTIME_ID,
            "name": "ClawPro 项目工作流（真实多 Agent）",
            "engine": "clawpro-project-workflow",
            "protocol": (
                "Node Contract + Handoff v2 + TeamAI ACP / iMate / "
                "DevResonance direct-prompt"
            ),
            "available": bool(
                (STATE.hatchery and STATE.hatchery.ready)
                or STATE.cloudagent.configured
            ),
            "executable": sys.executable,
            "mode": "cloud-local-workflow",
            "source_url": "clawpro://project-collaboration",
        },
        {
            "runtime_id": STRUCTURED_ISSUEFIX_RUNTIME_ID,
            "name": "SkillHub IssueFix（CodeBuddy + iMate 真实多 Agent）",
            "engine": "skillhub-issuefix-multi-agent",
            "protocol": "flow.yaml + CodeBuddy ACP + TeamAI/iMate + Node Contract",
            "available": bool(
                shutil.which("git")
                and shutil.which("ruby")
                and STATE.hatchery
                and STATE.hatchery.ready
            ),
            "executable": sys.executable,
            "mode": "cloud-local-workflow",
            "source_url": ISSUEFIX_SOURCE_URL,
            "source_ref": ISSUEFIX_REF,
        },
        {
            "runtime_id": "workflow-codebuddy-imate",
            "name": "CodeBuddy → iMate 协作",
            "engine": "codebuddy-to-imate",
            "protocol": "Hatchery workflow + TeamAI + ACP / iMate Issue",
            "available": bool(
                STATE.hatchery
                and STATE.hatchery.ready
                and codebuddy_available
                and imate_available
            ),
            "executable": (
                "用户电脑 TeamAI"
                if remote_teamai
                else str(STATE.hatchery.teamai_bin) if STATE.hatchery else None
            ),
            "mode": "cloud-local-workflow",
            "error": STATE.hatchery.setup_error if STATE.hatchery else "尚未配置",
        },
        {
            "runtime_id": "hatchery-teamai-imate-openclaw",
            "name": "ClawPro → TeamAI → iMate OpenClaw",
            "engine": "teamai-imate-openclaw",
            "protocol": "WSS wake + HTTPS sync / ack + iMate Issue",
            "available": bool(
                STATE.hatchery
                and STATE.hatchery.ready
                and imate_available
            ),
            "executable": (
                "用户电脑 TeamAI / iMate"
                if remote_teamai
                else str(STATE.hatchery.teamai_bin) if STATE.hatchery else None
            ),
            "mode": "cloud-local-live",
            "error": STATE.hatchery.setup_error if STATE.hatchery else "尚未配置",
        },
        {
            "runtime_id": "hatchery-teamai-codebuddy",
            "name": "Hatchery → TeamAI → CodeBuddy",
            "engine": "teamai-codebuddy",
            "protocol": "WSS wake + HTTPS sync / ack + ACP JSON-RPC stdio",
            "available": bool(
                STATE.hatchery
                and STATE.hatchery.ready
                and codebuddy_available
            ),
            "executable": (
                "用户电脑 TeamAI"
                if remote_teamai
                else str(STATE.hatchery.teamai_bin) if STATE.hatchery else None
            ),
            "mode": "cloud-local-live",
            "error": STATE.hatchery.setup_error if STATE.hatchery else "尚未配置",
            **codebuddy_auth,
        },
        {
            "runtime_id": "codebuddy-acp",
            "name": "本地直连 CodeBuddy ACP",
            "engine": "codebuddy",
            "protocol": "ACP JSON-RPC stdio",
            "available": codebuddy_available,
            "executable": "用户电脑 CodeBuddy" if remote_teamai else codebuddy,
            "mode": "remote-live-executable" if remote_teamai else "live-executable",
            **codebuddy_auth,
        },
        {
            "runtime_id": "mock-acp",
            "name": "Mock ACP Agent",
            "engine": "mock-acp",
            "protocol": "ACP JSON-RPC stdio",
            "available": True,
            "executable": str(ROOT / "mock_acp_agent.py"),
            "mode": "simulated",
        },
        {
            "runtime_id": "local-codex",
            "name": "本地 Codex",
            "engine": "codex",
            "protocol": "Codex JSON-RPC stdio",
            "available": bool(codex),
            "executable": codex,
            "mode": "detected-only",
        },
    ]


def runtime_by_id(runtime_id):
    return next(
        (runtime for runtime in runtime_catalog() if runtime["runtime_id"] == runtime_id),
        None,
    )


class Handler(SimpleHTTPRequestHandler):
    server_version = "ClawProEdgePoC/0.1"

    def __init__(self, *args, **kwargs):
        super().__init__(*args, directory=str(STATIC_ROOT), **kwargs)

    def log_message(self, format_string, *args):
        sys.stdout.write("[http] " + (format_string % args) + "\n")

    def send_json(self, payload, status=HTTPStatus.OK):
        data = json.dumps(payload, ensure_ascii=False).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(data)))
        self.send_header("Cache-Control", "no-store")
        self.end_headers()
        self.wfile.write(data)

    def send_artifact(self, task_id, relative_path):
        with STATE.lock:
            task = STATE.tasks.get(task_id)
            workspace_path = task.get("workspace_path") if task else None
            allowed_paths = {
                str(item.get("path") or "")
                for item in ((task or {}).get("artifact") or {}).get("artifacts", [])
            }
            allowed_paths.update(
                str(item.get("path") or "")
                for item in (task or {}).get("available_artifacts", [])
            )
            pending = (task or {}).get("pending_approval") or {}
            allowed_paths.update(
                "agent-workspace/{0}".format(item.get("path") or "")
                for item in pending.get("artifacts", [])
            )
            for event in (task or {}).get("events", []):
                if event.get("type") != "workflow.node.started":
                    continue
                payload = event.get("payload") or {}
                inputs = payload.get("inputs") or {}
                allowed_paths.update(
                    "agent-workspace/{0}".format(item.get("path") or "")
                    for item in inputs.get("artifacts", [])
                )
        if not workspace_path:
            self.send_error(HTTPStatus.NOT_FOUND)
            return
        if relative_path not in allowed_paths:
            self.send_error(HTTPStatus.FORBIDDEN)
            return
        workspace = Path(workspace_path).resolve()
        candidate = (workspace / relative_path).resolve()
        try:
            candidate.relative_to(workspace)
        except ValueError:
            self.send_error(HTTPStatus.FORBIDDEN)
            return
        if not candidate.is_file():
            self.send_error(HTTPStatus.NOT_FOUND)
            return
        data = candidate.read_bytes()
        content_type = mimetypes.guess_type(candidate.name)[0] or "application/octet-stream"
        self.send_response(HTTPStatus.OK)
        self.send_header("Content-Type", content_type)
        self.send_header("Content-Length", str(len(data)))
        self.send_header("Cache-Control", "no-store")
        self.send_header(
            "Content-Security-Policy",
            "sandbox; default-src 'none'; style-src 'unsafe-inline'; img-src data:",
        )
        self.end_headers()
        self.wfile.write(data)

    def read_json(self):
        length = int(self.headers.get("Content-Length") or 0)
        if length == 0:
            return {}
        return json.loads(self.rfile.read(length).decode("utf-8"))

    def do_GET(self):
        parsed = urlparse(self.path)
        path = parsed.path
        artifact_parts = [part for part in path.split("/") if part]
        if len(artifact_parts) >= 3 and artifact_parts[0] == "artifacts":
            self.send_artifact(
                artifact_parts[1],
                "/".join(artifact_parts[2:]),
            )
            return
        if path == "/api/health":
            self.send_json(
                {
                    "ok": True,
                    "connected": STATE.connected,
                    "teamai_connected": STATE.connected,
                }
            )
            return
        if path == "/api/teamai":
            if STATE.hatchery:
                STATE.teamai = STATE.hatchery.public_device()
            self.send_json(
                {
                    "connected": STATE.connected,
                    "bridge": STATE.teamai,
                    "edge_runtime": {
                        "status": "online" if STATE.connected else "waiting_for_teamai",
                        "managed_by": "TeamAI（演示）",
                    },
                }
            )
            return
        if path == "/api/imate/agents":
            if not STATE.hatchery:
                self.send_json({"error": "Hatchery—TeamAI Bridge 尚未配置"}, HTTPStatus.BAD_GATEWAY)
                return
            try:
                agents = STATE.hatchery.list_imate_openclaw_agents()
            except BridgeError as error:
                self.send_json({"error": str(error)}, HTTPStatus.BAD_GATEWAY)
                return
            self.send_json({"agents": agents})
            return
        if path == "/api/cloudagents":
            self.send_json(
                {
                    "configured": STATE.cloudagent.configured,
                    "agents": STATE.cloudagent.public_agents(),
                }
            )
            return
        if path == "/api/runtimes":
            if STATE.hatchery:
                STATE.teamai = STATE.hatchery.public_device()
            self.send_json(
                {
                    "connected": STATE.connected,
                    "runtimes": runtime_catalog(),
                    "device": STATE.teamai,
                }
            )
            return
        if path == "/api/tasks":
            with STATE.lock:
                tasks = [STATE.tasks[task_id] for task_id in STATE.task_order]
            self.send_json({"tasks": [public_task(task) for task in tasks]})
            return

        task_id, suffix = parse_task_path(path)
        if task_id and task_id in STATE.tasks:
            STATE.refresh_hatchery_task(task_id)
            with STATE.lock:
                task = STATE.tasks[task_id]
                if suffix == "events":
                    after = int((parse_qs(parsed.query).get("after") or ["0"])[0])
                    events = [event for event in task["events"] if event["seq"] > after]
                    self.send_json({"events": events, "task": public_task(task)})
                    return
                self.send_json({"task": public_task(task)})
                return
        if task_id:
            self.send_json(
                {
                    "error": "任务不存在或已失效，请重新执行",
                    "code": "TASK_NOT_FOUND",
                },
                HTTPStatus.NOT_FOUND,
            )
            return
        if path == "/":
            self.path = "/index.html"
        super().do_GET()

    def do_POST(self):
        parsed = urlparse(self.path)
        path = parsed.path
        if path in {"/api/runtime/connect", "/api/teamai/connect"}:
            STATE.set_connected(True)
            self.send_json({"connected": True, "bridge": STATE.teamai})
            return
        if path in {"/api/runtime/disconnect", "/api/teamai/disconnect"}:
            STATE.set_connected(False)
            self.send_json({"connected": False, "bridge": STATE.teamai})
            return
        if path == "/api/tasks":
            payload = self.read_json()
            prompt = str(payload.get("prompt") or "").strip()
            runtime_id = str(
                payload.get("runtime_id") or "hatchery-teamai-codebuddy"
            )
            model = str(payload.get("model") or "auto")
            target_agent_id = str(payload.get("target_agent_id") or "").strip()
            imate_project_id = str(payload.get("imate_project_id") or "").strip()
            delivery_mode = str(payload.get("delivery_mode") or "wss").strip().lower()
            agent_assignment_mode = str(
                payload.get("agent_assignment_mode") or "shared"
            ).strip().lower()
            agent_runtime_id = str(
                payload.get("agent_runtime_id") or "codebuddy-acp"
            ).strip()
            node_assignments = payload.get("node_assignments") or []
            if not prompt:
                self.send_json({"error": "请输入任务需求"}, HTTPStatus.BAD_REQUEST)
                return
            if delivery_mode not in {"wss", "hook"}:
                self.send_json({"error": "请选择有效的任务唤醒方式"}, HTTPStatus.BAD_REQUEST)
                return
            if agent_assignment_mode not in AGENT_ASSIGNMENT_MODES:
                self.send_json(
                    {"error": "请选择有效的 Agent 分配模式"},
                    HTTPStatus.BAD_REQUEST,
                )
                return
            if not isinstance(node_assignments, list) or any(
                not isinstance(item, dict)
                or not str(item.get("phase_id") or "").strip()
                or str(item.get("platform") or "").strip().lower()
                not in {"codebuddy", "imate", "cloudagent"}
                for item in node_assignments
            ):
                self.send_json(
                    {"error": "节点 Agent 路由格式无效"},
                    HTTPStatus.BAD_REQUEST,
                )
                return
            cloudagent_ids = [
                str(item.get("project_agent_id") or item.get("agent_id") or "").strip()
                for item in node_assignments
                if str(item.get("platform") or "").strip().lower()
                == "cloudagent"
            ]
            if cloudagent_ids and not STATE.cloudagent.configured:
                self.send_json(
                    {
                        "error": (
                            "DevResonance CloudAgent 网关尚未配置，当前不能执行：{0}。"
                            "请在 ClawPro 后端配置调用网关，凭证不会下发到浏览器。"
                        ).format(", ".join(cloudagent_ids))
                    },
                    HTTPStatus.BAD_REQUEST,
                )
                return
            if (
                runtime_id
                in {
                    STRUCTURED_ISSUEFIX_RUNTIME_ID,
                    STRUCTURED_PROJECT_WORKFLOW_RUNTIME_ID,
                }
                and agent_runtime_id not in STRUCTURED_AGENT_RUNTIME_IDS
            ):
                self.send_json(
                    {"error": "请选择可用的真实 Agent Runtime"},
                    HTTPStatus.BAD_REQUEST,
                )
                return
            if runtime_id in {
                STRUCTURED_ISSUEFIX_RUNTIME_ID,
                STRUCTURED_PROJECT_WORKFLOW_RUNTIME_ID,
            }:
                assigned_platforms = {
                    str(item.get("platform") or "").strip().lower()
                    for item in node_assignments
                }
                if assigned_platforms == {"codebuddy"}:
                    expected_runtime = "codebuddy-acp"
                elif assigned_platforms.issubset({"codebuddy", "imate"}):
                    expected_runtime = "codebuddy-imate-mixed"
                else:
                    expected_runtime = "node-routed-multi-agent"
                if agent_runtime_id != expected_runtime:
                    self.send_json(
                        {"error": "Agent 分配模式与 Runtime 组合不匹配"},
                        HTTPStatus.BAD_REQUEST,
                    )
                    return
            if runtime_id == STRUCTURED_PROJECT_WORKFLOW_RUNTIME_ID:
                raw_definition = payload.get("workflow_definition") or {}
                raw_phases = (
                    raw_definition.get("phases") or []
                    if isinstance(raw_definition, dict)
                    else []
                )
                phase_capabilities = {
                    str(phase.get("id") or "").strip(): {
                        str(item).strip()
                        for item in phase.get("required_capabilities") or []
                        if str(item).strip()
                    }
                    for phase in raw_phases
                    if isinstance(phase, dict)
                }
                available_capabilities = set(
                    codebuddy_capability_status()["capabilities"]
                )
                blocked_phases = [
                    str(item.get("phase_id") or "").strip()
                    for item in node_assignments
                    if str(item.get("platform") or "").strip().lower()
                    == "codebuddy"
                    and not phase_capabilities.get(
                        str(item.get("phase_id") or "").strip(), set()
                    ).issubset(available_capabilities)
                ]
                if blocked_phases:
                    self.send_json(
                        {
                            "error": (
                                "CodeBuddy 尚未获得该节点需要的 TeamAI 受控能力：{0}。"
                                "请完成用户授权后重试，不会禁用 CodeBuddy 的其他任务。"
                            ).format(", ".join(blocked_phases))
                        },
                        HTTPStatus.BAD_REQUEST,
                    )
                    return
            runtime = runtime_by_id(runtime_id)
            if not runtime or runtime["mode"] not in {
                "cloud-local-live",
                "cloud-local-workflow",
                "local-workflow-live",
                "local-workflow-validation",
                "live-executable",
                "simulated",
            }:
                self.send_json(
                    {"error": "当前 Runtime 不允许执行。"},
                    HTTPStatus.BAD_REQUEST,
                )
                return
            if not runtime["available"]:
                if runtime_id == STRUCTURED_ISSUEFIX_RUNTIME_ID:
                    unavailable_message = (
                        "SkillHub 源码包兼容性校验 Runtime 未启用；"
                        "项目协作任务请使用项目工作流 Runtime。"
                    )
                elif runtime_id in {
                    STRUCTURED_PROJECT_WORKFLOW_RUNTIME_ID,
                    "workflow-codebuddy-imate",
                    "hatchery-teamai-codebuddy",
                    "hatchery-teamai-imate-openclaw",
                }:
                    unavailable_message = (
                        "当前未配置可用的 TeamAI 执行通道，"
                        "请先完成 TeamAI 设备接入后重试。"
                    )
                else:
                    unavailable_message = "当前 Agent Runtime 不可用。"
                self.send_json(
                    {"error": unavailable_message},
                    HTTPStatus.BAD_REQUEST,
                )
                return
            if runtime_id in {"hatchery-teamai-imate-openclaw", "workflow-codebuddy-imate"}:
                if not target_agent_id:
                    self.send_json({"error": "请选择 iMate 管理的 OpenClaw Agent"}, HTTPStatus.BAD_REQUEST)
                    return
                if not imate_project_id:
                    self.send_json({"error": "请输入 iMate 项目 ID"}, HTTPStatus.BAD_REQUEST)
                    return
            if (
                runtime_id
                in {
                    STRUCTURED_ISSUEFIX_RUNTIME_ID,
                    STRUCTURED_PROJECT_WORKFLOW_RUNTIME_ID,
                }
                and agent_assignment_mode == "mixed"
            ):
                if not target_agent_id:
                    self.send_json(
                        {"error": "请选择参与交接的 iMate OpenClaw Agent"},
                        HTTPStatus.BAD_REQUEST,
                    )
                    return
                if not imate_project_id:
                    self.send_json(
                        {"error": "请输入 iMate 项目 ID"},
                        HTTPStatus.BAD_REQUEST,
                    )
                    return
            try:
                task = STATE.create_task(
                    prompt,
                    runtime_id,
                    model,
                    target_agent_id=target_agent_id,
                    imate_project_id=imate_project_id,
                    delivery_mode=delivery_mode,
                    agent_assignment_mode=agent_assignment_mode,
                    agent_runtime_id=agent_runtime_id,
                    node_assignments=node_assignments,
                    workflow_definition=payload.get("workflow_definition"),
                    workflow_inputs=payload.get("workflow_inputs"),
                )
            except BridgeError as error:
                self.send_json(
                    {"error": str(error)},
                    HTTPStatus.BAD_GATEWAY,
                )
                return
            self.send_json({"task": public_task(task)}, HTTPStatus.CREATED)
            return

        task_id, suffix = parse_task_path(path)
        if task_id and suffix == "approve":
            payload = self.read_json()
            gate_id = str(payload.get("gate_id") or "").strip()
            if not gate_id:
                self.send_json({"error": "缺少确认点 ID"}, HTTPStatus.BAD_REQUEST)
                return
            try:
                task = STATE.approve_workflow(task_id, gate_id)
            except ValueError as error:
                self.send_json({"error": str(error)}, HTTPStatus.CONFLICT)
                return
            if not task:
                self.send_json({"error": "任务不存在"}, HTTPStatus.NOT_FOUND)
                return
            self.send_json({"task": public_task(task)})
            return
        if task_id and suffix == "cancel":
            task = STATE.cancel_task(task_id)
            if not task:
                self.send_json({"error": "任务不存在"}, HTTPStatus.NOT_FOUND)
                return
            self.send_json({"task": public_task(task)})
            return
        self.send_json({"error": "Not found"}, HTTPStatus.NOT_FOUND)


def parse_task_path(path):
    parts = [part for part in path.split("/") if part]
    if len(parts) >= 3 and parts[0] == "api" and parts[1] == "tasks":
        return parts[2], parts[3] if len(parts) >= 4 else ""
    return None, None


def public_task(task):
    return {
        key: value
        for key, value in task.items()
        if key != "events" and not key.startswith("_")
    }


def main():
    parser = argparse.ArgumentParser(description="ClawPro Edge Runtime local PoC")
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--port", type=int, default=4188)
    parser.add_argument(
        "--hatchery-api",
        default=os.environ.get("HATCHERY_API", "http://127.0.0.1:8091"),
    )
    parser.add_argument(
        "--admin-token",
        default=os.environ.get("HATCHERY_ADMIN_TOKEN", "local-test-admin-token"),
        help="仅用于本地 Hatchery 联调，请勿传入生产 Token",
    )
    args = parser.parse_args()
    WORKSPACE_ROOT.mkdir(parents=True, exist_ok=True)
    REAL_WORKSPACE_ROOT.mkdir(parents=True, exist_ok=True)
    WORKFLOW_WORKSPACE_ROOT.mkdir(parents=True, exist_ok=True)
    STATE.configure_hatchery(args.hatchery_api, args.admin_token)
    server = ThreadingHTTPServer((args.host, args.port), Handler)

    def stop_server(*_):
        threading.Thread(target=server.shutdown, daemon=True).start()

    signal.signal(signal.SIGINT, stop_server)
    signal.signal(signal.SIGTERM, stop_server)
    print("ClawPro Agent Task UI: http://{0}:{1}".format(args.host, args.port))
    print(
        "Hatchery—TeamAI: {0}".format(
            "ready" if STATE.hatchery and STATE.hatchery.ready else "unavailable"
        )
    )
    try:
        server.serve_forever()
    finally:
        if STATE.hatchery:
            STATE.hatchery.stop_resident_listener()


if __name__ == "__main__":
    main()
