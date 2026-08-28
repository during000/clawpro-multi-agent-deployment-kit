#!/usr/bin/env python3
"""Live Hatchery -> TeamAI -> CodeBuddy ACP end-to-end verification."""

from __future__ import annotations

import json
import os
import pathlib
import shutil
import subprocess
import sys
import tempfile
import time

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers import (  # noqa: E402
    admin_client,
    check_env,
    enable_local_agent_feature,
    setup_admin,
    setup_local_instance,
    setup_user,
    user_client,
)
from helpers import config as test_config  # noqa: E402


def main() -> None:
    check_env()
    admin = setup_admin("agent-task-teamai-live")
    enable_local_agent_feature(admin.token)
    user = setup_user(admin.token, "agent-task-teamai-live")
    local_agent = setup_local_instance(
        user.token,
        "agent-task-teamai-live",
        agent_type="codebuddy",
    )

    project_name = f"agent-task-teamai-live-{int(time.time())}"
    project = admin_client(admin.token).post(
        "/admin/projects/create",
        json={"name": project_name},
        timeout=30,
    )["project"]
    admin_client(admin.token).post(
        "/admin/projects/members/add",
        json={"id": project["id"], "user_ids": [user.user_id]},
        timeout=30,
    )

    temp_root = pathlib.Path(tempfile.mkdtemp(prefix="clawpro-teamai-live-"))
    workspace = temp_root / "workspace"
    local_agent_home = temp_root / "local-agent"
    workspace.mkdir()
    local_agent_home.mkdir()
    expected = "Hatchery -> TeamAI -> CodeBuddy ACP works"
    proof_path = workspace / "hatchery-teamai-proof.txt"

    try:
        user_client(user.token).post(
            "/local-agent/report",
            json={
                "local_agent_id": local_agent.agent_id,
                "agent_type": "codebuddy",
                "agent_version": local_agent.agent_version,
                "host_name": local_agent.host_name,
                "os": local_agent.os,
                "workspaces": [{
                    "path": str(workspace),
                    "name": project_name,
                    "ide_type": "codebuddy",
                    "project_id": project["id"],
                    "skills": [],
                    "rules": [],
                }],
            },
            timeout=30,
        )

        task = user_client(user.token).post(
            "/agent-tasks/create",
            json={
                "instance_id": local_agent.db_id,
                "project_id": project["id"],
                "workspace_path": str(workspace),
                "prompt": (
                    "Create a file named hatchery-teamai-proof.txt in the current "
                    f"workspace. Its complete contents must be exactly: {expected}"
                ),
            },
            timeout=30,
        )["task"]

        config_path = local_agent_home / "config.json"
        config_path.write_text(json.dumps({
            "endpoint": test_config.BASE_URL,
            "token": user.token,
            "createdAt": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
            "routes": {
                "projects": "/projects/mine",
                "report": "/local-agent/report",
                "sync": "/local-agent/sync",
                "ack": "/local-agent/commands/ack",
                "getConfig": "/local-agent/get-config",
            },
            "workspaceBindings": {
                str(workspace): {
                    "projectId": project["id"],
                    "projectName": project_name,
                    "boundAt": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
                    "ideType": "codebuddy",
                },
            },
        }, ensure_ascii=False, indent=2), encoding="utf-8")
        config_path.chmod(0o600)

        teamai_bin = os.environ.get(
            "TEAMAI_BIN",
            os.path.expanduser("~/.teamai/bin/teamai"),
        )
        env = {
            **os.environ,
            "TEAMAI_LOCAL_AGENT_HOME": str(local_agent_home),
            "TEAMAI_LOCAL_AGENT_ID": local_agent.agent_id,
            "TEAMAI_BIND_PROMPT_ENABLED": "0",
            "TEAMAI_AGENT_TASK_TIMEOUT_MS": "180000",
        }
        hook_input = json.dumps({
            "cwd": str(workspace),
            "hook_event_name": "UserPromptSubmit",
            "session_id": "hatchery-teamai-live",
            "prompt": "poll ClawPro tasks",
        })
        result = subprocess.run(
            [teamai_bin, "hook-dispatch", "prompt-submit", "--tool", "codebuddy", "--stdin"],
            input=hook_input,
            text=True,
            capture_output=True,
            timeout=15,
            env=env,
            check=False,
        )
        if result.returncode != 0:
            raise RuntimeError(
                f"TeamAI hook failed ({result.returncode}): "
                f"{(result.stderr or result.stdout)[-1000:]}"
            )

        deadline = time.time() + 180
        latest = None
        while time.time() < deadline:
            tasks = user_client(user.token).get(
                "/agent-tasks",
                params={"id": task["id"]},
            )["tasks"]
            latest = tasks[0] if tasks else None
            if latest and latest["status"] in {"success", "failed"}:
                break
            time.sleep(1)

        if not latest or latest["status"] != "success":
            raise RuntimeError(f"Agent task did not succeed: {latest}")
        proof = proof_path.read_text(encoding="utf-8").strip()
        if proof != expected:
            raise RuntimeError(f"Unexpected proof contents: {proof}")
        if not latest.get("session_id") or not latest.get("result", "").strip():
            raise RuntimeError("Task result is missing session_id or Agent output")

        print(json.dumps({
            "ok": True,
            "task_id": task["id"],
            "status": latest["status"],
            "session_id": latest["session_id"],
            "proof": proof,
        }, ensure_ascii=False, indent=2))
    finally:
        shutil.rmtree(temp_root, ignore_errors=True)


if __name__ == "__main__":
    main()
