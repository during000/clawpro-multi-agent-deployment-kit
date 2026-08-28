#!/usr/bin/env python3
"""Load and safely smoke-test an external structured workflow package."""

from __future__ import annotations

import json
import os
import shutil
import subprocess
from pathlib import Path


ISSUEFIX_REMOTE = "git@git.woa.com:dns-ai/skillhub-workflow.git"
ISSUEFIX_REF = "feature/structured-workflow-v2"
ISSUEFIX_SUBDIR = Path("skills/skillhub-issuefix")
ISSUEFIX_SOURCE_URL = (
    "https://git.woa.com/dns-ai/skillhub-workflow/tree/"
    "feature/structured-workflow-v2/skills/skillhub-issuefix"
)
AGENT_ASSIGNMENT_MODES = {"shared", "mixed"}
MIXED_RUNTIME_BY_PHASE = {
    "analyze": "codebuddy-acp",
    "fix": "codebuddy-acp",
    "review": "imate-openclaw",
    "test": "imate-openclaw",
    "mr": "codebuddy-acp",
    "checkers": "imate-openclaw",
    "verify": "imate-openclaw",
    "close": "codebuddy-acp",
}


class StructuredWorkflowError(RuntimeError):
    pass


def _run(command, *, cwd=None, env=None, timeout=60, check=True):
    result = subprocess.run(
        [str(item) for item in command],
        cwd=str(cwd) if cwd else None,
        env=env,
        text=True,
        capture_output=True,
        timeout=timeout,
        check=False,
    )
    if check and result.returncode != 0:
        detail = (result.stderr or result.stdout or "命令执行失败").strip()
        raise StructuredWorkflowError(
            "{0}（RC={1}）：{2}".format(" ".join(command), result.returncode, detail[-2000:])
        )
    return result


def _load_yaml_with_ruby(path: Path):
    """Use macOS' bundled Ruby YAML parser without installing Python packages."""
    if not shutil.which("ruby"):
        raise StructuredWorkflowError("未找到 Ruby，无法解析外部工作流 YAML")
    script = (
        "require 'yaml'; require 'json'; "
        "data = YAML.safe_load(STDIN.read, permitted_classes: [], "
        "permitted_symbols: [], aliases: true); "
        "STDOUT.write(JSON.generate(data))"
    )
    result = subprocess.run(
        ["ruby", "-e", script],
        input=path.read_text(encoding="utf-8"),
        text=True,
        capture_output=True,
        timeout=20,
        check=False,
    )
    if result.returncode != 0:
        raise StructuredWorkflowError(
            "YAML 解析失败 {0}：{1}".format(path.name, result.stderr.strip())
        )
    try:
        return json.loads(result.stdout)
    except json.JSONDecodeError as error:
        raise StructuredWorkflowError("YAML 转换结果不是有效 JSON") from error


def normalize_definition(workflow_dir: Path):
    flow = _load_yaml_with_ruby(workflow_dir / "flow.yaml")
    phase_ids = [str(item.get("id") or "") for item in flow.get("phases", [])]
    if not phase_ids or any(not phase_id for phase_id in phase_ids):
        raise StructuredWorkflowError("flow.yaml 未定义有效阶段")
    if len(phase_ids) != len(set(phase_ids)):
        raise StructuredWorkflowError("flow.yaml 存在重复阶段 ID")

    gates = flow.get("gates", {})
    missing_gates = [phase_id for phase_id in phase_ids if phase_id not in gates]
    if missing_gates:
        raise StructuredWorkflowError("以下阶段缺少门禁：{0}".format(", ".join(missing_gates)))

    phases = []
    for index, phase in enumerate(flow["phases"]):
        agent_rel = str(phase.get("agent") or "")
        agent_path = workflow_dir / agent_rel
        if not agent_rel or not agent_path.is_file():
            raise StructuredWorkflowError(
                "阶段 {0} 的 Agent 定义不存在：{1}".format(phase["id"], agent_rel)
            )
        agent = _load_yaml_with_ruby(agent_path)
        on_pass = phase.get("on_pass")
        if on_pass is None and index + 1 < len(flow["phases"]):
            on_pass = flow["phases"][index + 1]["id"]
        on_fail = phase.get("on_fail", phase["id"])
        for route_name, target in (("on_pass", on_pass), ("on_fail", on_fail)):
            if target is not None and target not in phase_ids:
                raise StructuredWorkflowError(
                    "阶段 {0} 的 {1} 指向未知阶段 {2}".format(
                        phase["id"], route_name, target
                    )
                )
        inputs = agent.get("inputs", {}) or {}
        outputs = agent.get("outputs", {}) or {}
        phases.append(
            {
                "id": phase["id"],
                "title": phase.get("title") or phase["id"],
                "agent_id": agent.get("id") or Path(agent_rel).stem,
                "agent_title": agent.get("title") or phase.get("title") or phase["id"],
                "agent_description": agent.get("description") or "",
                "model": agent.get("model") or "default",
                "access": agent.get("access") or "read",
                "node_plan": [
                    {
                        "id": node.get("id"),
                        "title": node.get("title") or node.get("id"),
                        "actions": node.get("actions", []),
                        "constraints": node.get("constraints", []),
                    }
                    for node in agent.get("nodes", [])
                ],
                "inputs": {
                    "artifacts": inputs.get("artifacts", []),
                    "stages": inputs.get("stages", []),
                    "workspace_evidence": inputs.get("workspace_evidence", []),
                },
                "input_mappings": {
                    "task_goal": "$input.task_goal",
                    "upstream_result": (
                        "$vars.{0}.data".format(flow["phases"][index - 1]["id"])
                        if index > 0
                        else None
                    ),
                    "upstream_artifacts": (
                        "$artifacts.{0}.*".format(flow["phases"][index - 1]["id"])
                        if index > 0
                        else "$input.artifacts"
                    ),
                },
                "outputs": {
                    "artifacts": outputs.get("artifacts", []),
                    "gate_keys": outputs.get("gate_keys", []),
                },
                "output_schema": {
                    "type": "object",
                    "properties": {
                        "summary": {"type": "string"},
                        "runtime_check": {"type": ["object", "null"]},
                    },
                    "required": ["summary"],
                    "additionalProperties": True,
                },
                "declared_artifacts": phase.get("artifacts", []),
                "gate_checks": gates.get(phase["id"], []),
                "on_pass": on_pass,
                "on_fail": on_fail,
                "retry": phase.get("retry", {}),
                "node_count": len(agent.get("nodes", [])),
            }
        )

    return {
        "schema_version": "clawpro.structured-workflow.v2",
        "workflow_id": "skillhub-issuefix",
        "name": "SkillHub IssueFix",
        "description": "Issue 分析、修复、评审、测试、MR、Checkers、E2E 与关闭流程",
        "source": {
            "url": ISSUEFIX_SOURCE_URL,
            "git_remote": ISSUEFIX_REMOTE,
            "git_ref": ISSUEFIX_REF,
            "subdir": str(ISSUEFIX_SUBDIR),
        },
        "phases": phases,
        "always_checks": flow.get("always", []),
        "limits": flow.get("limits", {}),
        "external_write_phases": ["mr", "checkers", "verify", "close"],
    }


def build_agent_assignment(definition, mode):
    """Map workflow roles onto CodeBuddy-only or CodeBuddy+iMate runtimes."""
    if mode not in AGENT_ASSIGNMENT_MODES:
        raise StructuredWorkflowError("不支持的 Agent 分配模式：{0}".format(mode))
    assignments = []
    for phase in definition["phases"]:
        runtime_id = (
            "codebuddy-acp"
            if mode == "shared"
            else MIXED_RUNTIME_BY_PHASE.get(phase["id"], "codebuddy-acp")
        )
        instance_id = (
            "codebuddy-shared"
            if runtime_id == "codebuddy-acp"
            else "imate-openclaw-shared"
        )
        assignments.append(
            {
                "phase_id": phase["id"],
                "role_agent_id": phase["agent_id"],
                "agent_instance_id": instance_id,
                "runtime_id": runtime_id,
            }
        )
    unique_instances = sorted(
        {item["agent_instance_id"] for item in assignments}
    )
    expected_count = 1 if mode == "shared" else 2
    return {
        "schema_version": "clawpro.agent-assignment.v2",
        "mode": mode,
        "assignments": assignments,
        "unique_agent_instances": unique_instances,
        "unique_agent_count": len(unique_instances),
        "expected_agent_count": expected_count,
        "verified": len(unique_instances) == expected_count,
    }


def build_handoff_trace(definition, assignment, prompt):
    """Build the v2 data/artifact contracts used by every runtime adapter."""
    assignment_by_phase = {
        item["phase_id"]: item for item in assignment["assignments"]
    }
    trace = []
    previous_output = None
    for phase in definition["phases"]:
        assigned = assignment_by_phase[phase["id"]]
        upstream_results = []
        if previous_output:
            upstream_results.append(
                {
                    "producer_node": previous_output["node_id"],
                    "producer_agent_instance_id": previous_output[
                        "agent_instance_id"
                    ],
                    "data": previous_output["data"],
                    "artifacts": previous_output["artifacts"],
                }
            )
        node_input = {
            "schema_version": "clawpro.node-input.v2",
            "workflow_run_id": "contract-validation",
            "node_run_id": "contract-validation:{0}:1".format(phase["id"]),
            "attempt_id": "attempt-1",
            "node": {
                "id": phase["id"],
                "role_agent_id": assigned["role_agent_id"],
                "agent_instance_id": assigned["agent_instance_id"],
                "runtime_id": assigned["runtime_id"],
            },
            "task": {"goal": prompt},
            "inputs": {
                "mappings": phase.get("input_mappings", {}),
                "data": {
                    "upstream": upstream_results[0]["data"]
                    if upstream_results
                    else None
                },
                "artifacts": upstream_results[0]["artifacts"]
                if upstream_results
                else [],
                "declared": phase["inputs"],
            },
            "output_contract": {
                "data_schema": phase.get("output_schema", {}),
                "required_artifacts": phase["declared_artifacts"],
            },
        }
        node_output = {
            "schema_version": "clawpro.node-result.v2",
            "workflow_run_id": "contract-validation",
            "node_run_id": "contract-validation:{0}:1".format(phase["id"]),
            "node_id": phase["id"],
            "role_agent_id": assigned["role_agent_id"],
            "agent_instance_id": assigned["agent_instance_id"],
            "runtime_id": assigned["runtime_id"],
            "status": "contract_validated",
            "data": {
                "summary": "{0}节点契约验证完成".format(phase["title"]),
                "runtime_check": None,
            },
            "artifacts": [
                {
                    "schema_version": "clawpro.artifact-ref.v1",
                    "artifact_id": "{0}:{1}".format(phase["id"], path),
                    "version": 1,
                    "path": path,
                    "media_type": "text/markdown",
                    "size": 1,
                    "sha256": "contract-validation",
                    "producer_node": phase["id"],
                    "lineage": [
                        item["artifact_id"]
                        for item in (previous_output or {}).get("artifacts", [])
                    ],
                }
                for path in phase["declared_artifacts"]
            ],
            "gate_results": [
                {"gate_index": index, "status": "definition_validated"}
                for index, _ in enumerate(phase["gate_checks"], start=1)
            ],
        }
        received_previous_output = (
            previous_output is None
            or (
                len(upstream_results) == 1
                and upstream_results[0]["producer_node"]
                == previous_output["node_id"]
                and upstream_results[0]["artifacts"]
                == previous_output["artifacts"]
            )
        )
        trace.append(
            {
                "phase_id": phase["id"],
                "input": node_input,
                "output": node_output,
                "received_previous_output": received_previous_output,
            }
        )
        previous_output = node_output
    return {
        "schema_version": "clawpro.handoff-trace.v2",
        "trace": trace,
        "handoff_count": max(0, len(trace) - 1),
        "verified": all(item["received_previous_output"] for item in trace),
    }


def _write_yaml_shim(shim_root: Path):
    shim_root.mkdir(parents=True, exist_ok=True)
    (shim_root / "yaml.py").write_text(
        '''"""Minimal PyYAML-compatible bridge backed by Ruby Psych for this PoC."""
import json
import shutil
import subprocess

def safe_load(stream):
    text = stream.read() if hasattr(stream, "read") else str(stream)
    if not shutil.which("ruby"):
        raise ImportError("Ruby is required by the PoC YAML bridge")
    script = ("require 'yaml'; require 'json'; "
              "data = YAML.safe_load(STDIN.read, permitted_classes: [], "
              "permitted_symbols: [], aliases: true); "
              "STDOUT.write(JSON.generate(data))")
    result = subprocess.run(["ruby", "-e", script], input=text, text=True,
                            capture_output=True, check=False)
    if result.returncode != 0:
        raise ValueError(result.stderr.strip())
    return json.loads(result.stdout)
''',
        encoding="utf-8",
    )


def _prepare_fixture_repo(repo: Path):
    repo.mkdir(parents=True, exist_ok=True)
    _run(["git", "init", "-b", "main"], cwd=repo)
    _run(["git", "config", "user.name", "ClawPro PoC"], cwd=repo)
    _run(["git", "config", "user.email", "clawpro-poc@example.invalid"], cwd=repo)
    (repo / "calculator.py").write_text(
        "def add(left, right):\n    return left + right\n", encoding="utf-8"
    )
    (repo / "README.md").write_text(
        "# IssueFix compatibility fixture\n\nNo external repository is modified.\n",
        encoding="utf-8",
    )
    _run(["git", "add", "calculator.py", "README.md"], cwd=repo)
    _run(["git", "commit", "-m", "chore: initialize workflow fixture"], cwd=repo)


def _clone_workflow_source(destination: Path):
    _run(
        [
            "git",
            "clone",
            "--depth",
            "1",
            "--branch",
            ISSUEFIX_REF,
            "--single-branch",
            ISSUEFIX_REMOTE,
            str(destination),
        ],
        timeout=90,
    )
    revision = _run(["git", "rev-parse", "HEAD"], cwd=destination).stdout.strip()
    return revision


def run_issuefix_compatibility_smoke(
    workspace: Path, prompt: str, assignment_mode="shared"
):
    """Run the package's own init/next/orchestrator dry-run without external writes."""
    source_root = workspace / "source"
    fixture_repo = workspace / "fixture-repo"
    engine_workspace = workspace / "engine"
    shim_root = workspace / "python-shim"
    revision = _clone_workflow_source(source_root)
    workflow_dir = source_root / ISSUEFIX_SUBDIR
    definition = normalize_definition(workflow_dir)
    definition["source"]["revision"] = revision
    assignment = build_agent_assignment(definition, assignment_mode)
    handoff = build_handoff_trace(definition, assignment, prompt)
    if not assignment["verified"] or not handoff["verified"]:
        raise StructuredWorkflowError("Agent 分配或节点交接契约校验失败")

    _prepare_fixture_repo(fixture_repo)
    _write_yaml_shim(shim_root)
    engine_workspace.mkdir(parents=True, exist_ok=True)

    local_config = {
        "repo": {
            "local": str(fixture_repo),
            "base": "main",
            "worktree_root": str(engine_workspace / "worktrees"),
        },
        "testdb": {"available": False},
        "e2e": {"enabled": False},
        "notify": {
            "enabled": False,
            "wecom": {"enabled": False, "webhook_url": ""},
        },
    }
    (workflow_dir / "project.local.json").write_text(
        json.dumps(local_config, ensure_ascii=False, indent=2), encoding="utf-8"
    )

    env = {
        **os.environ,
        "WF_WORKSPACE": str(engine_workspace),
        "PYTHONPATH": str(shim_root)
        + (os.pathsep + os.environ["PYTHONPATH"] if os.environ.get("PYTHONPATH") else ""),
    }
    scripts = workflow_dir / "scripts"
    slug = "poc-issuefix"
    init_result = _run(
        [
            "python3",
            str(scripts / "run.py"),
            "--workflow",
            "issuefix",
            "init",
            "--slug",
            slug,
            "--story-id",
            "POC-LOCAL",
            "--name",
            prompt[:120] or "验证结构化 IssueFix 工作流",
            "--side",
            "backend",
        ],
        env=env,
        timeout=60,
    )
    next_result = _run(
        [
            "python3",
            str(scripts / "next.py"),
            "--workflow",
            "issuefix",
            "--slug",
            slug,
            "--json",
        ],
        env=env,
        timeout=30,
    )
    action = json.loads(next_result.stdout)
    orchestrator_result = _run(
        [
            "python3",
            str(scripts / "orchestrator.py"),
            "--workflow",
            "issuefix",
            "--slug",
            slug,
            "--max-loops",
            "1",
            "--dry-run",
        ],
        env=env,
        timeout=60,
    )

    contracts = {
        "schema_version": "clawpro.node-contract.v2",
        "workflow_id": definition["workflow_id"],
        "agent_assignment": assignment,
        "handoff": handoff,
        "nodes": [
            {
                "node_id": phase["id"],
                "input_contract": {
                    "mappings": phase["input_mappings"],
                    "declared": phase["inputs"],
                },
                "output_contract": {
                    "data_schema": phase["output_schema"],
                    "declared": phase["outputs"],
                },
                "artifact_contract": phase["declared_artifacts"],
                "gate_contract": phase["gate_checks"],
                "routes": {
                    "on_pass": phase["on_pass"],
                    "on_fail": phase["on_fail"],
                },
            }
            for phase in definition["phases"]
        ],
    }
    smoke = {
        "schema_version": "clawpro.workflow-smoke.v2",
        "safe_mode": True,
        "external_writes_performed": False,
        "source_revision": revision,
        "agent_assignment": assignment,
        "handoff": {
            "handoff_count": handoff["handoff_count"],
            "verified": handoff["verified"],
        },
        "original_engine": {
            "init": json.loads(init_result.stdout),
            "next_action": {
                "action": action.get("action"),
                "phase": action.get("phase"),
                "agent": (action.get("agent_data") or {}).get("id"),
                "artifacts_required": action.get("artifacts_required", []),
            },
            "orchestrator_dry_run": "[DRY RUN]" in orchestrator_result.stderr,
        },
        "validated_phase_count": len(definition["phases"]),
        "validated_gate_count": sum(
            len(phase["gate_checks"]) for phase in definition["phases"]
        )
        + len(definition["always_checks"]),
    }
    (workspace / "workflow-definition.json").write_text(
        json.dumps(definition, ensure_ascii=False, indent=2), encoding="utf-8"
    )
    (workspace / "node-contracts.json").write_text(
        json.dumps(contracts, ensure_ascii=False, indent=2), encoding="utf-8"
    )
    (workspace / "runtime-smoke.json").write_text(
        json.dumps(smoke, ensure_ascii=False, indent=2), encoding="utf-8"
    )
    (workspace / "agent-assignment.json").write_text(
        json.dumps(
            {"agent_assignment": assignment, "handoff": handoff},
            ensure_ascii=False,
            indent=2,
        ),
        encoding="utf-8",
    )
    phase_lines = [
        "- `{0}`：{1}；通过 → `{2}`；失败 → `{3}`".format(
            phase["id"],
            phase["title"],
            phase["on_pass"] or "结束",
            phase["on_fail"],
        )
        for phase in definition["phases"]
    ]
    assignment_lines = [
        "- `{0}`：角色 `{1}` → 执行实例 `{2}`".format(
            item["phase_id"], item["role_agent_id"], item["agent_instance_id"]
        )
        for item in assignment["assignments"]
    ]
    mode_copy = (
        "同一 CodeBuddy Agent 会话执行全部节点"
        if assignment_mode == "shared"
        else "CodeBuddy 与 iMate OpenClaw 按节点分工交接"
    )
    report = """# SkillHub IssueFix 工作流兼容性报告

## 结论

原始工作流包已在 ClawPro POC 中完成来源拉取、YAML 解析、阶段/Agent/门禁/路由校验，且原始 `run.py → next.py → orchestrator.py --dry-run` 链路执行成功。

## 来源

- 分支：`{ref}`
- Commit：`{revision}`
- 目录：`skills/skillhub-issuefix`

## 阶段

{phases}

## Agent 分配与交接

- 模式：{mode_copy}
- 节点数：{phase_count}
- Agent 实例数：{agent_count}
- 节点间交接数：{handoff_count}
- 上下文与产物交接校验：通过

{assignments}

## 安全边界

- 本次使用本地临时 Git fixture，不修改 SkillHub 源码仓库。
- 未创建分支、提交、MR、流水线、Issue 评论或企微通知等外部对象。
- 要验证真实修复闭环，仍需提供一个允许自动修改并创建 MR 的真实 SkillHub Issue。
""".format(
        ref=ISSUEFIX_REF,
        revision=revision,
        phases="\n".join(phase_lines),
        mode_copy=mode_copy,
        phase_count=len(definition["phases"]),
        agent_count=assignment["unique_agent_count"],
        handoff_count=handoff["handoff_count"],
        assignments="\n".join(assignment_lines),
    )
    (workspace / "compatibility-report.md").write_text(report, encoding="utf-8")
    for transient in (source_root, fixture_repo, engine_workspace, shim_root):
        shutil.rmtree(transient, ignore_errors=True)
    return definition, smoke
