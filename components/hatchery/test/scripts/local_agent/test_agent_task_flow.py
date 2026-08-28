#!/usr/bin/env python3
"""ClawPro → TeamAI/Edge Runtime → 本地 Agent 任务闭环。"""

import os
import sys
import time

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers import (  # noqa: E402
    admin_client,
    check_env,
    enable_local_agent_feature,
    reporter_ack,
    reporter_sync,
    setup_admin,
    setup_local_instance,
    setup_user,
    user_client,
)


def main():
    check_env()
    admin = setup_admin("agent-task")
    enable_local_agent_feature(admin.token)
    user = setup_user(admin.token, "agent-task")
    local_agent = setup_local_instance(user.token, "agent-task", agent_type="codebuddy")

    project_name = f"agent-task-{int(time.time())}"
    created = admin_client(admin.token).post(
        "/admin/projects/create",
        json={"name": project_name},
        timeout=30,
    )
    project_id = created["project"]["id"]
    admin_client(admin.token).post(
        "/admin/projects/members/add",
        json={"id": project_id, "user_ids": [user.user_id]},
        timeout=30,
    )

    workspace_path = f"/tmp/{project_name}"
    user_client(user.token).post(
        "/local-agent/report",
        json={
            "local_agent_id": local_agent.agent_id,
            "agent_type": local_agent.agent_type,
            "agent_version": local_agent.agent_version,
            "host_name": local_agent.host_name,
            "os": local_agent.os,
            "workspaces": [{
                "path": workspace_path,
                "name": project_name,
                "ide_type": "codebuddy",
                "project_id": project_id,
                "skills": [],
                "rules": [],
            }],
        },
        timeout=30,
    )

    created_task = user_client(user.token).post(
        "/agent-tasks/create",
        json={
            "instance_id": local_agent.db_id,
            "project_id": project_id,
            "workspace_path": workspace_path,
            "prompt": "创建 hello.txt，内容为 hello from ClawPro",
        },
        timeout=30,
    )["task"]
    assert created_task["status"] == "pending", created_task

    sync_data = reporter_sync(user.token, local_agent)
    command = next(
        item for item in sync_data["cmds"]
        if item["id"] == created_task["id"] and item["type"] == "execute_agent_task"
    )
    assert command["workspace_path"] == workspace_path, command
    assert command["project_id"] == project_id, command

    reporter_ack(
        user.token,
        created_task["id"],
        "running",
        ack_type="execute_agent_task",
        result="开始执行\n",
        session_id="integration-session",
    )
    reporter_ack(
        user.token,
        created_task["id"],
        "success",
        ack_type="execute_agent_task",
        result="开始执行\n执行完成",
    )

    tasks = user_client(user.token).get(
        "/agent-tasks",
        params={"id": created_task["id"]},
    )["tasks"]
    assert len(tasks) == 1, tasks
    assert tasks[0]["status"] == "success", tasks[0]
    assert tasks[0]["result"] == "开始执行\n执行完成", tasks[0]
    assert tasks[0]["session_id"] == "integration-session", tasks[0]
    print("ClawPro 本地 Agent 任务闭环通过 ✓")


if __name__ == "__main__":
    main()
