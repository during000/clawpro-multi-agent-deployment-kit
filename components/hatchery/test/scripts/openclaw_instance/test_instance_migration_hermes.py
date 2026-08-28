#!/usr/bin/env python3
"""
集成测试：实例管理 - Hermes 数据迁移导出契约

覆盖接口：
    POST /openclaw/migration/export  Hermes 实例生成专属导出脚本
    GET  /openclaw/migration/status  未执行脚本时文件尚未就绪
    POST /openclaw/migration/import  文件未上传时拒绝导入

本用例创建真实 Hermes 实例并调用已部署 Hatchery API。默认只验证安全契约；设置
HERMES_MIGRATION_E2E=1 后，会通过 TAT 执行返回脚本并完成一次同实例 export/import
恢复闭环。导出脚本包含临时 SMH 凭证，请勿把完整响应写入测试日志。
"""

import base64
import json
import os
import re
import shlex
import sys
import time

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers import setup_admin, setup_user
from helpers.api import ApiClient, health_check, run_tests, user_client
from helpers.hermes import HERMES_AGENT_TYPE, setup_hermes_instance
from helpers.instance import delete_instance, list_instances


SCENARIO = f"hermes-migration-{int(time.time())}"
HERMES_INSTANCE = None
HERMES_USER = None
HERMES_CLIENT = None
MIGRATION_ID = None
MIGRATION_FILE_KEY = None
MIGRATION_SCRIPT = None
BOOTSTRAP_ADMIN_TOKEN = os.environ.get("BOOTSTRAP_ADMIN_TOKEN", "").strip()

EXPECTED_EXCLUDED_PATHS = {
    "hermes-agent",
    "audio_cache",
    "image_cache",
    "cache",
    "logs",
    "sandboxes",
    "gateway.pid",
    "config.yaml.lock",
    "cron/.tick.lock",
}


def _require_smh_ready():
    """SMH 未启用或配置不完整时整组跳过，避免创建无意义的 CVM。"""
    assert BOOTSTRAP_ADMIN_TOKEN, (
        "BOOTSTRAP_ADMIN_TOKEN 未设置，无法读取非 OpenAPI 的 /admin/smh/config"
    )
    bootstrap_admin = ApiClient(BOOTSTRAP_ADMIN_TOKEN, timeout=30)
    smh_config = bootstrap_admin.get("/admin/smh/config")
    if smh_config.get("smh_enabled") and smh_config.get("is_configured"):
        return

    print("=" * 60)
    print("Hermes 迁移集成测试 SKIPPED：SMH 未启用或配置不完整")
    print("=" * 60)
    sys.exit(0)


def _migration_client():
    """关闭帧输出，避免导出响应中的临时 SMH 凭证进入 CI 日志。"""
    assert HERMES_USER is not None
    return ApiClient(
        HERMES_USER.token,
        openapi=True,
        timeout=120,
        trace=False,
    )


def _extract_excluded_paths(script):
    match = re.search(r"^EXCLUDED_PATHS_B64='([^']+)'", script, re.MULTILINE)
    assert match is not None, "Hermes 导出脚本缺少 EXCLUDED_PATHS_B64"
    try:
        decoded = base64.b64decode(match.group(1), validate=True)
        paths = json.loads(decoded)
    except (ValueError, json.JSONDecodeError) as exc:
        raise AssertionError("EXCLUDED_PATHS_B64 不是合法的 base64 JSON") from exc
    assert isinstance(paths, list), "EXCLUDED_PATHS_B64 解码结果应为列表"
    return set(paths)


def test_01_instance_is_hermes():
    """真实实例记录的 agent_type 为 hermes"""
    assert HERMES_INSTANCE is not None
    data = HERMES_CLIENT.get(
        "/openclaw/list",
        params={"id": HERMES_INSTANCE.db_id, "page_size": 1},
    )
    instances = data.get("instances") or []
    assert len(instances) == 1, (
        f"未找到 Hermes 实例 db_id={HERMES_INSTANCE.db_id}"
    )
    agent_type = instances[0].get("agent_type") or instances[0].get("AgentType")
    assert agent_type == HERMES_AGENT_TYPE, (
        f"agent_type={agent_type!r}, want {HERMES_AGENT_TYPE!r}"
    )
    print(f"    Hermes 实例确认 ✓ db_id={HERMES_INSTANCE.db_id}")


def test_02_export_builds_hermes_script():
    """导出接口为 Hermes 生成目录、兼容开关和安全校验逻辑"""
    global MIGRATION_ID, MIGRATION_FILE_KEY, MIGRATION_SCRIPT

    response = _migration_client().post(
        "/openclaw/migration/export",
        data={"id": HERMES_INSTANCE.db_id},
        expect=None,
        raw=True,
    )
    assert response.status_code == 200, (
        f"Hermes migration export status={response.status_code}, "
        f"body={response.text[:300]}"
    )

    body = response.json()
    MIGRATION_ID = body.get("migration_id")
    MIGRATION_FILE_KEY = body.get("file_key")
    MIGRATION_SCRIPT = body.get("script") or ""
    assert MIGRATION_ID, "导出响应缺少 migration_id"
    assert MIGRATION_FILE_KEY, "导出响应缺少 file_key"
    assert MIGRATION_SCRIPT, "导出响应缺少 script"

    expected_file_key = (
        f"migrations/{HERMES_INSTANCE.instance_id}/agent-export.tgz"
    )
    assert MIGRATION_FILE_KEY == expected_file_key, (
        f"file_key={MIGRATION_FILE_KEY!r}, want {expected_file_key!r}"
    )

    required_markers = {
        'AGENT_DIR="$HOME/.hermes"': "Hermes 数据目录",
        "ALLOW_AGENT_ROOT_CHANGE_WARNING='1'": "Hermes 专属兼容开关",
        "build_included_business_manifest": "业务树指纹校验",
        "is_only_agent_root_changed_warning": "根目录告警精确分类",
        'tar tf "$ARCHIVE_PATH"': "归档完整性校验",
    }
    for marker, label in required_markers.items():
        assert marker in MIGRATION_SCRIPT, f"Hermes 导出脚本缺少{label}"
    assert "ALLOW_AGENT_ROOT_CHANGE_WARNING='0'" not in MIGRATION_SCRIPT, (
        "Hermes 导出脚本错误关闭了专属兼容开关"
    )

    excluded_paths = _extract_excluded_paths(MIGRATION_SCRIPT)
    assert excluded_paths == EXPECTED_EXCLUDED_PATHS, (
        f"Hermes 排除路径不匹配: got={sorted(excluded_paths)}, "
        f"want={sorted(EXPECTED_EXCLUDED_PATHS)}"
    )
    print(
        f"    Hermes 导出脚本契约通过 ✓ migration_id={MIGRATION_ID}, "
        f"excluded_paths={len(excluded_paths)}"
    )


def test_03_status_reports_file_not_ready():
    """未执行导出脚本时迁移记录存在，但 SMH 文件尚未就绪"""
    assert MIGRATION_ID is not None
    response = HERMES_CLIENT.get(
        "/openclaw/migration/status",
        params={"id": HERMES_INSTANCE.db_id},
        expect=None,
        raw=True,
        timeout=60,
    )
    assert response.status_code == 200, (
        f"migration status={response.status_code}, body={response.text[:300]}"
    )
    body = response.json()
    assert body.get("has_migration") is True, f"迁移记录缺失: {body}"
    assert body.get("migration_id") == MIGRATION_ID, (
        f"migration_id={body.get('migration_id')}, want {MIGRATION_ID}"
    )
    assert body.get("file_key") == MIGRATION_FILE_KEY
    assert body.get("file_ready") is False, f"文件不应已就绪: {body}"
    assert body.get("can_import") is False, f"此时不应允许导入: {body}"
    print("    未上传状态契约通过 ✓")


def test_04_import_rejects_missing_upload():
    """Hermes 导出文件未上传时 import 必须拒绝"""
    response = HERMES_CLIENT.post(
        "/openclaw/migration/import",
        data={"id": HERMES_INSTANCE.db_id},
        expect=None,
        raw=True,
        timeout=60,
    )
    assert response.status_code == 400, (
        f"未上传迁移包时 import status={response.status_code}, "
        f"body={response.text[:300]}"
    )
    error = (response.json() or {}).get("error", "")
    assert "上传" in error or "文件" in error, (
        f"import 错误信息未说明迁移文件未就绪: {error!r}"
    )
    print("    未上传禁止导入 ✓")


def _response_data(body):
    data = body.get("data") or body.get("Data") or {}
    return data.get("Response") or data.get("response") or {}


def _run_tat_command(content, *, timeout=900):
    """在测试 Hermes 实例上执行命令并等待终态，全程禁止输出命令正文。"""
    client = _migration_client()
    response = client.post(
        "/openclaw/lightclaw/run-command",
        params={"id": HERMES_INSTANCE.db_id},
        json={
            "Content": base64.b64encode(content.encode()).decode(),
            "InstanceIds": [HERMES_INSTANCE.instance_id],
            "CommandType": "SHELL",
        },
        expect=None,
        raw=True,
        timeout=60,
    )
    assert response.status_code == 200, (
        f"TAT run-command HTTP status={response.status_code}"
    )
    body = response.json()
    assert body.get("code") == 0, f"TAT run-command code={body.get('code')!r}"
    invocation_id = _response_data(body).get("InvocationId")
    assert invocation_id, "TAT run-command 响应缺少 InvocationId"

    deadline = time.time() + timeout
    task_id = None
    task_status = ""
    while time.time() < deadline:
        response = client.post(
            "/openclaw/lightclaw/describe-invocations",
            params={"id": HERMES_INSTANCE.db_id},
            json={"InvocationIds": [invocation_id]},
            expect=None,
            raw=True,
            timeout=60,
        )
        assert response.status_code == 200, (
            f"TAT describe-invocations HTTP status={response.status_code}"
        )
        body = response.json()
        assert body.get("code") == 0, (
            f"TAT describe-invocations code={body.get('code')!r}"
        )
        invocations = _response_data(body).get("InvocationSet") or []
        if invocations:
            tasks = invocations[0].get("InvocationTaskBasicInfoSet") or []
            if tasks:
                task_id = tasks[0].get("InvocationTaskId")
                task_status = tasks[0].get("TaskStatus") or ""
                if task_status in {
                    "SUCCESS",
                    "FAILED",
                    "TIMEOUT",
                    "DELIVER_FAILED",
                    "START_FAILED",
                }:
                    break
        time.sleep(5)
    else:
        raise AssertionError(f"TAT 命令在 {timeout}s 内未完成")

    assert task_id, "TAT invocation 缺少 InvocationTaskId"
    response = client.post(
        "/openclaw/lightclaw/describe-invocation-tasks",
        params={"id": HERMES_INSTANCE.db_id},
        json={"InvocationTaskIds": [task_id], "HideOutput": False},
        expect=None,
        raw=True,
        timeout=60,
    )
    assert response.status_code == 200, (
        f"TAT describe-invocation-tasks HTTP status={response.status_code}"
    )
    body = response.json()
    assert body.get("code") == 0, (
        f"TAT describe-invocation-tasks code={body.get('code')!r}"
    )
    tasks = _response_data(body).get("InvocationTaskSet") or []
    assert tasks, "TAT describe-invocation-tasks 未返回任务"
    task = tasks[0]
    output_b64 = task.get("Output") or ""
    try:
        output = base64.b64decode(output_b64).decode(errors="replace")
    except ValueError as exc:
        raise AssertionError("TAT 输出不是合法 base64") from exc
    exit_code = task.get("ExitCode")
    assert task_status == "SUCCESS" and exit_code in (0, "0", None), (
        f"TAT 命令失败: status={task_status}, exit_code={exit_code}, "
        f"output_tail={output[-1000:]!r}"
    )
    return output


def _wait_migration_file_ready(timeout=300):
    deadline = time.time() + timeout
    while time.time() < deadline:
        response = _migration_client().get(
            "/openclaw/migration/status",
            params={"id": HERMES_INSTANCE.db_id},
            expect=None,
            raw=True,
            timeout=60,
        )
        assert response.status_code == 200, (
            f"migration status HTTP status={response.status_code}"
        )
        body = response.json()
        if body.get("file_ready") and body.get("can_import"):
            return body
        time.sleep(5)
    raise AssertionError(f"迁移文件在 {timeout}s 内未就绪")


def _wait_migration_done(timeout=900):
    deadline = time.time() + timeout
    last_status = ""
    while time.time() < deadline:
        response = _migration_client().get(
            "/openclaw/migration/progress",
            params={"id": HERMES_INSTANCE.db_id},
            expect=None,
            raw=True,
            timeout=60,
        )
        assert response.status_code == 200, (
            f"migration progress HTTP status={response.status_code}"
        )
        body = response.json()
        last_status = body.get("status") or ""
        if last_status == "done":
            return body
        if last_status == "failed":
            raise AssertionError(
                "Hermes migration import 失败，请结合服务端迁移日志定位"
            )
        time.sleep(5)
    raise AssertionError(
        f"Hermes migration import 在 {timeout}s 内未完成，最后状态={last_status!r}"
    )


def test_05_real_export_import_round_trip():
    """可选真实 E2E：写标记 → SMH 导出 → 删除 → import 恢复"""
    if os.environ.get("HERMES_MIGRATION_E2E") != "1":
        print("    SKIP (设置 HERMES_MIGRATION_E2E=1 后执行真实迁移闭环)")
        return

    assert MIGRATION_SCRIPT, "缺少前序 export 返回脚本"
    marker = f"hermes-migration-it-{int(time.time())}"
    marker_path = "$HOME/.hermes/.codex-migration-it-marker"
    create_marker = (
        f"mkdir -p \"$HOME/.hermes\" && "
        f"printf %s {shlex.quote(marker)} > \"{marker_path}\""
    )
    _run_tat_command(create_marker, timeout=120)

    export_output = _run_tat_command(MIGRATION_SCRIPT, timeout=900)
    assert "导出成功" in export_output, "真实导出未输出成功标记"
    ready = _wait_migration_file_ready()
    assert ready.get("migration_id") == MIGRATION_ID

    _run_tat_command(f"rm -f \"{marker_path}\"", timeout=120)
    response = _migration_client().post(
        "/openclaw/migration/import",
        data={"id": HERMES_INSTANCE.db_id},
        expect=None,
        raw=True,
        timeout=60,
    )
    assert response.status_code == 200, (
        f"Hermes migration import status={response.status_code}, "
        f"body={response.text[:300]}"
    )
    _wait_migration_done()

    restored_marker = _run_tat_command(
        f"cat \"{marker_path}\"",
        timeout=120,
    ).strip()
    assert restored_marker == marker, (
        f"迁移标记未正确恢复: got={restored_marker!r}"
    )
    print("    Hermes 真实导出/import 闭环通过 ✓")


def _cleanup_hermes_instances():
    if HERMES_USER is None:
        return
    try:
        instances = list_instances(HERMES_USER.token).get("instances") or []
    except Exception as exc:
        print(f"    [cleanup] 查询 Hermes 实例失败（交由全局 cleanup）: {exc}")
        return

    for instance in instances:
        db_id = instance.get("id") or instance.get("ID")
        if not db_id:
            continue
        try:
            result = delete_instance(HERMES_USER.token, db_id)
            print(f"    [cleanup] 删除实例 db_id={db_id}: ok={result.get('ok')}")
        except Exception as exc:
            print(
                f"    [cleanup] 删除实例 db_id={db_id} 失败"
                f"（交由全局 cleanup）: {exc}"
            )


def main():
    global HERMES_INSTANCE, HERMES_USER, HERMES_CLIENT

    health_check()
    _require_smh_ready()
    admin = setup_admin(SCENARIO)
    HERMES_USER = setup_user(admin.token, SCENARIO)

    try:
        HERMES_INSTANCE = setup_hermes_instance(
            HERMES_USER.token,
            "migration",
        )
        HERMES_CLIENT = user_client(HERMES_USER.token)
        print()
        run_tests(
            globals(),
            title="Hermes 数据迁移导出集成测试",
            ordered=True,
            abort_on_fail=True,
        )
    finally:
        _cleanup_hermes_instances()


if __name__ == "__main__":
    main()
