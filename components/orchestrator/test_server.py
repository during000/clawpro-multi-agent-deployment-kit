import json
import base64
import hashlib
import tempfile
import threading
import time
import unittest
from types import SimpleNamespace
from unittest.mock import patch
from pathlib import Path

from server import DemoState, TaskCanceled, codebuddy_capability_status
from hatchery_teamai_bridge import BridgeError, HatcheryTeamAIBridge
from structured_workflow import (
    build_agent_assignment,
    build_handoff_trace,
    normalize_definition,
)


class TaskPersistenceTests(unittest.TestCase):
    @staticmethod
    def task(task_id, status):
        now = "2026-08-25T00:00:00.000+00:00"
        return {
            "task_id": task_id,
            "attempt_id": "attempt_test",
            "runtime_id": "structured-project-workflow",
            "status": status,
            "execution_status": status,
            "workflow_stage": status,
            "workflow_current_phase": "analyze" if status == "running" else None,
            "workflow_current_phases": ["analyze"] if status == "running" else [],
            "workflow_phases": [
                {
                    "id": "analyze",
                    "title": "分析",
                    "status": status,
                    "artifacts": [],
                }
            ],
            "pending_approval": None,
            "created_at": now,
            "updated_at": now,
            "events": [],
        }

    def test_completed_task_survives_state_reload(self):
        with tempfile.TemporaryDirectory() as temp_dir:
            state_path = Path(temp_dir) / "task-state.json"
            state = DemoState(state_path)
            task = self.task("workflow_completed", "completed")
            state.tasks[task["task_id"]] = task
            state.task_order.append(task["task_id"])
            state.append_event(task["task_id"], "workflow.completed", "完成")

            restored = DemoState(state_path)

            self.assertIn(task["task_id"], restored.tasks)
            self.assertEqual(restored.tasks[task["task_id"]]["status"], "completed")
            self.assertEqual(
                restored.tasks[task["task_id"]]["events"][0]["type"],
                "workflow.completed",
            )

    def test_running_task_is_restored_as_interrupted_failure(self):
        with tempfile.TemporaryDirectory() as temp_dir:
            state_path = Path(temp_dir) / "task-state.json"
            state = DemoState(state_path)
            task = self.task("workflow_running", "running")
            state.tasks[task["task_id"]] = task
            state.task_order.append(task["task_id"])
            state.append_event(task["task_id"], "workflow.node.started", "开始")

            restored = DemoState(state_path)
            recovered = restored.tasks[task["task_id"]]

            self.assertEqual(recovered["status"], "failed")
            self.assertEqual(recovered["workflow_stage"], "failed")
            self.assertIsNone(recovered["workflow_current_phase"])
            self.assertEqual(recovered["workflow_phases"][0]["status"], "failed")
            self.assertEqual(
                recovered["events"][-1]["type"],
                "task.recovered.interrupted",
            )


class HandoffContextTests(unittest.TestCase):
    def test_keeps_only_final_agent_answer_from_long_trace(self):
        state = DemoState()
        trace = "tool output\n" * 1000 + "任务完成。\nHANDOFF-CONTEXT-OK"

        result = state.clean_agent_result(trace)

        self.assertEqual(result, "任务完成。\nHANDOFF-CONTEXT-OK")

    def test_builds_context_and_copies_text_artifact(self):
        state = DemoState()
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            upstream_workspace = root / "upstream"
            workflow_workspace = root / "workflow"
            upstream_workspace.mkdir()
            workflow_workspace.mkdir()
            (upstream_workspace / "TASK.md").write_text("internal", encoding="utf-8")
            (upstream_workspace / "UPSTREAM_RESULT.md").write_text(
                "上游结论：采用方案 A。", encoding="utf-8"
            )

            workflow_id = "workflow_test"
            state.tasks[workflow_id] = {
                "prompt": "完成跨 Agent 方案",
                "workspace_path": str(workflow_workspace),
            }
            upstream = {
                "task_id": "task_upstream",
                "workspace_path": str(upstream_workspace),
                "agent_output": "已完成上游分析。",
                "artifact": {
                    "artifacts": state.collect_workspace_files(upstream_workspace)
                },
            }

            package, downstream_prompt = state.build_handoff_context(
                workflow_id, upstream
            )

            self.assertEqual(package["workflow_run_id"], workflow_id)
            self.assertEqual(len(package["artifact_refs"]), 1)
            self.assertEqual(
                package["artifact_refs"][0]["path"], "UPSTREAM_RESULT.md"
            )
            self.assertIn("上游结论：采用方案 A。", downstream_prompt)
            self.assertIn("已完成上游分析。", downstream_prompt)
            self.assertTrue(
                (workflow_workspace / "upstream-artifacts" / "UPSTREAM_RESULT.md").is_file()
            )
            self.assertTrue((workflow_workspace / "context-package.json").is_file())

    def test_materializes_full_teamai_artifact_bundle_with_integrity_metadata(self):
        state = DemoState()
        raw = ("# 完整需求分析\n\n" + "验收边界与测试结论。\n" * 120).encode("utf-8")
        relative = "01-requirement/requirement-report.md"
        bundle = {
            "schema_version": "clawpro.artifact-bundle.v1",
            "required_artifacts": [relative],
            "artifacts": [
                {
                    "path": relative,
                    "source_path": "server/artifacts/demo/" + relative,
                    "size": len(raw),
                    "sha256": hashlib.sha256(raw).hexdigest(),
                    "content_base64": base64.b64encode(raw).decode("ascii"),
                }
            ],
        }
        result = (
            "节点执行完成。\n\n<clawpro_artifact_bundle_v1>\n"
            + json.dumps(bundle)
            + "\n</clawpro_artifact_bundle_v1>"
        )

        with tempfile.TemporaryDirectory() as temp_dir:
            workspace = Path(temp_dir)
            summary, uploaded = state.materialize_teamai_artifact_bundle(
                result, workspace, [relative]
            )

            self.assertEqual(summary, "节点执行完成。")
            self.assertEqual((workspace / relative).read_bytes(), raw)
            self.assertEqual(uploaded[0]["size"], len(raw))
            self.assertEqual(uploaded[0]["sha256"], hashlib.sha256(raw).hexdigest())

    def test_rejects_summary_only_remote_result_for_required_artifacts(self):
        state = DemoState()
        with tempfile.TemporaryDirectory() as temp_dir:
            summary, uploaded = state.materialize_teamai_artifact_bundle(
                "只返回了摘要", Path(temp_dir), ["report.md"]
            )
        self.assertEqual(summary, "只返回了摘要")
        self.assertEqual(uploaded, [])


class StructuredWorkflowTests(unittest.TestCase):
    def test_intake_state_merge_preserves_agent_evidence_and_canonical_stages(self):
        state = DemoState()
        canonical_stages = {"PHASE-0": {"status": "completed"}}
        canonical = {
            "task_slug": "workflow_fallback",
            "workspace": {"repository_url": "", "root": ""},
            "stages": canonical_stages,
            "decisions": [],
            "summary": {"status": "running", "report": "workflow-summary.md"},
        }
        with tempfile.TemporaryDirectory() as temp_dir:
            agent_workspace = Path(temp_dir)
            (agent_workspace / "workflow-state.json").write_text(
                json.dumps(
                    {
                        "task_slug": "contest_school_registry_20260827_2109",
                        "workspace": {
                            "repository_url": "https://git.example.com/demo.git",
                            "branch": "feature/demo",
                            "root": "/workspace/demo",
                        },
                        "artifacts_dir": "/workspace/artifacts/demo",
                        "stages": {"PHASE-0": {"status": "agent-owned"}},
                        "decisions": [
                            {
                                "type": "SIZE_CLASS_EVIDENCE",
                                "metrics": {
                                    "estimated_files": 9,
                                    "module_count": 3,
                                    "risk_level": "high",
                                },
                            }
                        ],
                    },
                    ensure_ascii=False,
                ),
                encoding="utf-8",
            )

            merged = state.merge_intake_workflow_state(canonical, agent_workspace)

        self.assertEqual(merged["task_slug"], "contest_school_registry_20260827_2109")
        self.assertEqual(merged["workspace"]["branch"], "feature/demo")
        self.assertEqual(merged["artifacts_dir"], "/workspace/artifacts/demo")
        self.assertEqual(merged["stages"], canonical_stages)
        self.assertEqual(
            merged["decisions"][0]["metrics"]["estimated_files"], 9
        )

    def test_workflow_state_artifact_versions_change_only_with_content(self):
        state = DemoState()
        task_id = "workflow_state_versions"
        state.tasks[task_id] = {"available_artifacts": []}
        owner = {"node_id": "PHASE-0", "artifacts": []}
        with tempfile.TemporaryDirectory() as temp_dir:
            agent_workspace = Path(temp_dir)
            state_path = agent_workspace / "workflow-state.json"
            state_path.write_text('{"status":"running"}\n', encoding="utf-8")
            state.refresh_workflow_state_artifact(task_id, owner, agent_workspace)
            first = owner["artifacts"][0]

            state.refresh_workflow_state_artifact(task_id, owner, agent_workspace)
            unchanged = owner["artifacts"][0]

            state_path.write_text('{"status":"completed"}\n', encoding="utf-8")
            state.refresh_workflow_state_artifact(task_id, owner, agent_workspace)
            changed = owner["artifacts"][0]

        self.assertEqual(first["version"], 1)
        self.assertEqual(unchanged["version"], 1)
        self.assertEqual(changed["version"], 2)
        self.assertNotEqual(first["sha256"], changed["sha256"])

    def test_project_workflow_keeps_optional_artifacts_out_of_required_gate(self):
        state = DemoState()
        definition = state.normalize_project_workflow_definition(
            {
                "workflow_id": "optional-artifacts",
                "phases": [
                    {
                        "id": "develop",
                        "artifacts": ["03-code/change-report.md"],
                        "optional_artifacts": ["03-code/api-docs.md"],
                    }
                ],
            }
        )

        phase = definition["phases"][0]
        self.assertEqual(
            phase["declared_artifacts"], ["03-code/change-report.md"]
        )
        self.assertEqual(phase["optional_artifacts"], ["03-code/api-docs.md"])

    @staticmethod
    def assignment_definition():
        return {
            "phases": [
                {
                    "id": phase_id,
                    "title": title,
                    "agent_id": role,
                    "inputs": {"artifacts": [], "stages": []},
                    "declared_artifacts": [artifact],
                    "gate_checks": [{"kind": "file_exists", "path": artifact}],
                }
                for phase_id, title, role, artifact in (
                    ("analyze", "分析", "analyst", "analysis.md"),
                    ("fix", "修复", "fixer", "fix.md"),
                    ("review", "评审", "reviewer", "review.md"),
                )
            ]
        }

    def test_shared_agent_mode_preserves_roles_and_reuses_one_instance(self):
        definition = self.assignment_definition()

        assignment = build_agent_assignment(definition, "shared")
        handoff = build_handoff_trace(definition, assignment, "修复 Issue")

        self.assertTrue(assignment["verified"])
        self.assertEqual(assignment["unique_agent_count"], 1)
        self.assertEqual(
            {item["role_agent_id"] for item in assignment["assignments"]},
            {"analyst", "fixer", "reviewer"},
        )
        self.assertEqual(handoff["handoff_count"], 2)
        self.assertTrue(handoff["verified"])

    def test_mixed_agent_mode_routes_between_codebuddy_and_imate(self):
        definition = self.assignment_definition()

        assignment = build_agent_assignment(definition, "mixed")
        handoff = build_handoff_trace(definition, assignment, "修复 Issue")

        self.assertTrue(assignment["verified"])
        self.assertEqual(assignment["unique_agent_count"], 2)
        self.assertEqual(
            [item["runtime_id"] for item in assignment["assignments"]],
            ["codebuddy-acp", "codebuddy-acp", "imate-openclaw"],
        )
        self.assertEqual(handoff["handoff_count"], len(definition["phases"]) - 1)
        self.assertTrue(handoff["verified"])
        self.assertEqual(
            handoff["trace"][1]["input"]["inputs"]["artifacts"][0]["producer_node"],
            "analyze",
        )
        self.assertEqual(
            handoff["trace"][1]["input"]["schema_version"],
            "clawpro.node-input.v2",
        )
        self.assertEqual(
            handoff["trace"][0]["output"]["schema_version"],
            "clawpro.node-result.v2",
        )

    def test_node_routes_bind_teamai_device_and_selected_project_agent(self):
        state = DemoState()
        state.hatchery = type("Bridge", (), {"agent_id": "device-teamai-01"})()
        definition = self.assignment_definition()
        task = {
            "agent_assignment_mode": "mixed",
            "target_agent_id": "imate-openclaw-01",
            "node_assignments": [
                {
                    "phase_id": "analyze",
                    "project_agent_id": "project-codebuddy-carol",
                    "platform": "codebuddy",
                    "location": "local",
                },
                {
                    "phase_id": "fix",
                    "project_agent_id": "project-codebuddy-carol",
                    "platform": "codebuddy",
                    "location": "local",
                },
                {
                    "phase_id": "review",
                    "project_agent_id": "project-imate-openclaw",
                    "platform": "imate",
                    "location": "cloud",
                    "target_agent_id": "imate-openclaw-node-01",
                },
            ],
        }

        assignment = state.build_teamai_node_assignment(definition, task)

        self.assertEqual(assignment["schema_version"], "clawpro.agent-assignment.v3")
        self.assertEqual(assignment["mode"], "node-routed")
        self.assertTrue(assignment["verified"])
        self.assertEqual(
            [item["runtime_id"] for item in assignment["assignments"]],
            [
                "hatchery-teamai-codebuddy",
                "hatchery-teamai-codebuddy",
                "hatchery-teamai-imate-openclaw",
            ],
        )
        self.assertEqual(
            {item["device_id"] for item in assignment["assignments"]},
            {"device-teamai-01"},
        )
        self.assertEqual(
            assignment["assignments"][2]["target_agent_id"],
            "imate-openclaw-node-01",
        )

    def test_cloudagent_nodes_route_directly_without_local_teamai(self):
        state = DemoState()
        definition = {
            "phases": [
                {
                    "id": "calendar-match",
                    "agent_id": "calendar-scan-agent",
                }
            ]
        }
        task = {
            "agent_assignment_mode": "shared",
            "node_assignments": [
                {
                    "phase_id": "calendar-match",
                    "project_agent_id": "calendar-scan-agent",
                    "platform": "cloudagent",
                    "location": "cloud",
                }
            ],
        }

        assignment = state.build_teamai_node_assignment(definition, task)
        route = assignment["assignments"][0]

        self.assertEqual(route["runtime_id"], "devresonance-cloudagent")
        self.assertEqual(route["device_id"], "devresonance-cloud")
        self.assertEqual(route["transport"], "https-direct-prompt")

    def test_normalizes_phase_contracts_and_routes(self):
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            (root / "agents").mkdir()
            (root / "flow.yaml").write_text(
                """version: 2
phases:
  - id: analyze
    title: 分析
    agent: agents/analyst.yaml
    artifacts: [plan.md]
    on_pass: fix
    on_fail: analyze
  - id: fix
    title: 修复
    agent: agents/fixer.yaml
    artifacts: [report.md]
    on_fail: fix
gates:
  analyze:
    - {kind: file_min_lines, path: plan.md, min_lines: 1}
  fix:
    - {kind: file_min_lines, path: report.md, min_lines: 1}
always: []
limits: {max_retry_per_phase: 2}
""",
                encoding="utf-8",
            )
            (root / "agents" / "analyst.yaml").write_text(
                """id: analyst
title: 分析者
inputs: {artifacts: [], stages: []}
outputs:
  artifacts: [{path: plan.md, min_lines: 1}]
  gate_keys: []
nodes: [{id: recon, type: recon}]
""",
                encoding="utf-8",
            )
            (root / "agents" / "fixer.yaml").write_text(
                """id: fixer
title: 修复者
inputs: {artifacts: [plan.md], stages: [analyze]}
outputs:
  artifacts: [{path: report.md, min_lines: 1}]
  gate_keys: [gates.build]
nodes: [{id: fix, type: execute}]
""",
                encoding="utf-8",
            )

            definition = normalize_definition(root)

            self.assertEqual(definition["schema_version"], "clawpro.structured-workflow.v2")
            self.assertEqual([item["id"] for item in definition["phases"]], ["analyze", "fix"])
            self.assertEqual(definition["phases"][0]["on_pass"], "fix")
            self.assertIsNone(definition["phases"][1]["on_pass"])
            self.assertEqual(definition["phases"][1]["inputs"]["artifacts"], ["plan.md"])
            self.assertEqual(
                definition["phases"][1]["input_mappings"]["upstream_result"],
                "$vars.analyze.data",
            )

    def test_v2_handoff_materializes_and_validates_artifacts(self):
        state = DemoState()
        with tempfile.TemporaryDirectory() as temp_dir:
            agent_workspace = Path(temp_dir) / "agent-workspace"
            node_workspace = agent_workspace / "node-workspaces" / "01-analyze"
            agent_workspace.mkdir(parents=True)
            source = agent_workspace / "ISSUE.md"
            source.write_text("issue input", encoding="utf-8")
            raw = source.read_bytes()
            source_ref = state.build_artifact_ref(
                "workflow_test", "workflow-input", "ISSUE.md", raw
            )
            envelope = {"inputs": {"artifacts": [source_ref]}}

            state.stage_node_inputs(agent_workspace, node_workspace, envelope)
            self.assertTrue(
                (node_workspace / ".upstream_artifacts" / "ISSUE.md").is_file()
            )
            self.assertEqual(
                envelope["inputs"]["staged_artifacts"][0]["sha256"],
                source_ref["sha256"],
            )

            output = agent_workspace / "analysis.md"
            output.write_text("analysis result", encoding="utf-8")
            output_ref = state.build_artifact_ref(
                "workflow_test",
                "analyze",
                "analysis.md",
                output.read_bytes(),
                lineage=[source_ref["artifact_id"]],
            )
            result = {
                "schema_version": "clawpro.node-result.v2",
                "node_id": "analyze",
                "runtime_id": "codebuddy-acp",
                "status": "completed",
                "data": {"summary": "done", "runtime_check": None},
                "artifacts": [output_ref],
            }
            handoff = state.write_node_handoff(
                node_workspace, result, agent_workspace
            )

            self.assertTrue((handoff / ".handoff.json").is_file())
            self.assertTrue((handoff / ".handoff.md").is_file())
            self.assertTrue((handoff / "analysis.md").is_file())
            self.assertEqual(output_ref["lineage"], [source_ref["artifact_id"]])

    def test_normalizes_project_workflow_and_dynamic_approval(self):
        state = DemoState()
        definition = state.normalize_project_workflow_definition(
            {
                "workflow_id": "knowledge-inspection",
                "name": "知识库巡检",
                "phases": [
                    {
                        "id": "scan",
                        "title": "全量读取",
                        "agent_id": "reader",
                        "prompt": "递归读取知识库",
                        "artifacts": ["pages.json"],
                        "required_evidence": ["iwiki.metadata", "4025707654"],
                        "reject_output_markers": ["模拟"],
                        "required_capabilities": ["iwiki.read"],
                    },
                    {
                        "id": "report",
                        "title": "报告生成",
                        "agent_id": "auditor",
                        "prompt": "生成审计报告",
                        "artifacts": ["report.md"],
                        "approval_required": True,
                    },
                ],
            }
        )

        self.assertEqual(definition["workflow_id"], "knowledge-inspection")
        self.assertEqual(definition["phases"][0]["on_pass"], "report")
        self.assertEqual(
            definition["phases"][0]["required_evidence"],
            ["iwiki.metadata", "4025707654"],
        )
        self.assertEqual(
            definition["phases"][0]["required_capabilities"],
            ["iwiki.read"],
        )
        self.assertEqual(
            definition["phases"][1]["declared_artifacts"], ["report.md"]
        )
        self.assertEqual(
            state.workflow_gate_for_phase(definition["phases"][1])["gate_id"],
            "approve-report",
        )

    def test_project_workflow_normalizes_and_stages_node_config_assets(self):
        state = DemoState()
        definition = state.normalize_project_workflow_definition(
            {
                "workflow_id": "asset-backed-workflow",
                "phases": [
                    {
                        "id": "PHASE-0",
                        "title": "需求分流",
                        "artifacts": ["workflow-state.json"],
                        "config_assets": [
                            {
                                "id": "intake-router",
                                "name": "需求接入与分流",
                                "version": "1.4",
                                "type": "rules",
                                "summary": "按文件数、模块数和风险分级",
                                "source": "dns-ai/multi-agents-devflow@fixed",
                                "content": "同名 slug 且涉及数据迁移时必须判定为 LARGE。",
                            },
                            {
                                "id": "node-handoff-contract",
                                "name": "节点交接契约",
                                "version": "1.4",
                                "type": "contract",
                                "summary": "约束节点输入输出",
                                "source": "dns-ai/multi-agents-devflow@fixed",
                                "content": '{"schema_version":"clawpro.node-input.v2"}',
                            },
                        ],
                    }
                ],
            }
        )
        phase = definition["phases"][0]

        self.assertEqual(len(phase["config_assets"]), 2)
        self.assertEqual(
            phase["config_assets"][0]["sha256"],
            hashlib.sha256(
                phase["config_assets"][0]["content"].encode("utf-8")
            ).hexdigest(),
        )

        envelope = {"node": {}, "inputs": {"artifacts": []}}
        with tempfile.TemporaryDirectory() as temp_dir:
            root = Path(temp_dir)
            state.stage_node_inputs(
                root,
                root / "node",
                envelope,
                phase["config_assets"],
            )
            rules_path = root / "node/.clawpro/config-assets/intake-router.md"
            contract_path = (
                root / "node/.clawpro/config-assets/node-handoff-contract.json"
            )

            self.assertTrue(rules_path.is_file())
            self.assertTrue(contract_path.is_file())
            self.assertIn("必须判定为 LARGE", rules_path.read_text("utf-8"))
            self.assertEqual(
                envelope["node"]["config_assets"][0][
                    "orchestrator_staged_path"
                ],
                ".clawpro/config-assets/intake-router.md",
            )

    def test_node_config_assets_are_injected_into_codebuddy_and_imate_prompts(self):
        state = DemoState()
        asset_content = "关键未决问题必须为 0，否则节点必须阻断。"
        asset_bytes = asset_content.encode("utf-8")
        phase = {
            "id": "TASK-01",
            "title": "需求分析",
            "agent_id": "requirement-analysis",
            "agent_title": "需求分析",
            "agent_description": "分析真实源码与需求",
            "node_plan": [],
            "required_capabilities": [],
            "declared_artifacts": ["01-requirement/requirement-report.md"],
            "optional_artifacts": [],
            "config_assets": [
                {
                    "id": "requirement-analysis",
                    "name": "需求分析与澄清",
                    "version": "1.4",
                    "type": "rules",
                    "summary": "未决问题门禁",
                    "source": "dns-ai/multi-agents-devflow@fixed",
                    "content": asset_content,
                    "size": len(asset_bytes),
                    "sha256": hashlib.sha256(asset_bytes).hexdigest(),
                }
            ],
        }
        envelope = {
            "schema_version": "clawpro.node-input.v2",
            "task": {"goal": "分析校园赛事需求", "inputs": {}},
            "inputs": {"artifacts": []},
        }

        codebuddy_prompt = state.build_real_node_prompt(
            phase,
            "分析校园赛事需求",
            "input.json",
            phase["declared_artifacts"],
            input_envelope=envelope,
        )
        state.tasks["workflow_assets"] = {"prompt": "分析校园赛事需求"}
        imate_prompt = state.build_imate_node_prompt(
            "workflow_assets", phase, envelope, Path(".")
        )

        for prompt in (codebuddy_prompt, imate_prompt):
            self.assertIn("节点配置资产（强制执行）", prompt)
            self.assertIn("需求分析与澄清", prompt)
            self.assertIn(asset_content, prompt)
            self.assertIn(phase["config_assets"][0]["sha256"], prompt)

    def test_node_config_assets_are_injected_into_cloudagent_prompt(self):
        state = DemoState()
        content = "必须执行 Bad Cases 前置扫描。"
        raw = content.encode("utf-8")
        phase = {
            "id": "TASK-01",
            "title": "需求分析",
            "agent_description": "分析校园赛事需求",
            "declared_artifacts": ["requirement-report.md"],
            "config_assets": [
                {
                    "id": "bad-cases",
                    "name": "Bad Cases 扫描规则",
                    "version": "1.4",
                    "type": "rules",
                    "summary": "扫描已知错误模式",
                    "source": "dns-ai/multi-agents-devflow@fixed",
                    "content": content,
                    "size": len(raw),
                    "sha256": hashlib.sha256(raw).hexdigest(),
                }
            ],
        }
        captured = {}
        state.cloudagent.route_for = lambda _agent_id: (
            SimpleNamespace(session_id="session-cloudagent"),
            "agent",
        )

        def fake_execute(agent_id, prompt, trace_id, timeout_seconds):
            captured["prompt"] = prompt
            return {
                "summary": "已完成真实分析",
                "trace_id": trace_id,
                "session_id": "session-cloudagent",
                "attachments": [],
                "usage": {},
            }

        state.cloudagent.execute = fake_execute
        task_id = "workflow_cloud_assets"
        state.tasks[task_id] = {
            "attempt_id": "attempt_cloud_assets",
            "events": [],
            "cancel_requested": False,
        }
        with tempfile.TemporaryDirectory() as temp_dir:
            result = state.execute_cloudagent_workflow_node(
                task_id,
                phase,
                {"project_agent_id": "cloudagent:test"},
                {"schema_version": "clawpro.node-input.v2"},
                Path(temp_dir),
            )

        self.assertEqual(result["session_id"], "session-cloudagent")
        self.assertIn("节点配置资产（强制执行）", captured["prompt"])
        self.assertIn("Bad Cases 扫描规则", captured["prompt"])
        self.assertIn(content, captured["prompt"])

    def test_project_workflow_rejects_empty_config_asset_content(self):
        state = DemoState()
        with self.assertRaisesRegex(BridgeError, "缺少名称、版本或正文"):
            state.normalize_project_workflow_definition(
                {
                    "workflow_id": "invalid-assets",
                    "phases": [
                        {
                            "id": "analyze",
                            "config_assets": [
                                {
                                    "id": "rules",
                                    "name": "规则",
                                    "version": "1",
                                    "type": "rules",
                                    "content": "",
                                }
                            ],
                        }
                    ],
                }
            )

    def test_state_machine_routes_failed_review_back_and_persists_state(self):
        state = DemoState()
        definition = state.normalize_project_workflow_definition(
            {
                "workflow_id": "review-loop",
                "name": "评审返工闭环",
                "execution_mode": "state_machine",
                "phases": [
                    {
                        "id": "PHASE-0",
                        "title": "初始化",
                        "artifacts": ["workflow-state.json"],
                        "on_pass": "TASK-03",
                    },
                    {
                        "id": "TASK-03",
                        "title": "开发",
                        "artifacts": ["03-code/change-report.md"],
                        "on_pass": "CODE-REVIEW",
                    },
                    {
                        "id": "CODE-REVIEW",
                        "title": "评审",
                        "artifacts": ["03-code/review-report.md"],
                        "on_pass": "SUMMARY",
                        "on_fail": "TASK-03",
                        "decision_mode": "review_verdict",
                        "max_retries": 2,
                    },
                    {
                        "id": "SUMMARY",
                        "title": "汇总",
                        "artifacts": ["workflow-summary.md"],
                        "on_pass": None,
                    },
                ],
            }
        )
        task_id = "workflow_review_loop"
        state.tasks[task_id] = {
            "attempt_id": "attempt_review_loop",
            "prompt": "验证评审返工",
            "cancel_requested": False,
            "workflow_phases": [
                {"id": phase["id"], "status": "ready", "retry_count": 0}
                for phase in definition["phases"]
            ],
            "workflow_current_phase": None,
            "workflow_current_phases": [],
            "workflow_stage": "contract_validated",
            "available_artifacts": [],
            "events": [],
            "_workflow_inputs": state.normalize_workflow_inputs(
                {
                    "task_slug": {"type": "text", "value": "review-loop"},
                    "run_mode": {"type": "text", "value": "auto"},
                    "runtime_mode": {"type": "text", "value": "ide"},
                }
            ),
        }
        assignment = {
            "mode": "node-routed",
            "unique_agent_count": 2,
            "assignments": [
                {
                    "phase_id": phase["id"],
                    "runtime_id": (
                        "hatchery-teamai-imate-openclaw"
                        if phase["id"] == "CODE-REVIEW"
                        else "hatchery-teamai-codebuddy"
                    ),
                    "agent_instance_id": "agent-" + phase["id"],
                    "project_agent_id": "project-" + phase["id"],
                    "device_id": "device-1",
                    "transport": "wss+https",
                }
                for phase in definition["phases"]
            ],
        }
        review_attempt = {"count": 0}

        def fake_execute(
            task_id_arg,
            definition_arg,
            phase,
            assigned,
            agent_workspace,
            node_workspace_root,
            base_artifact_paths,
            upstream_results,
            index,
            is_issuefix,
        ):
            del definition_arg, node_workspace_root, base_artifact_paths, is_issuefix
            for relative in phase["declared_artifacts"]:
                path = agent_workspace / relative
                path.parent.mkdir(parents=True, exist_ok=True)
                if relative == "workflow-state.json" and path.exists():
                    continue
                path.write_text("{0} output".format(phase["id"]), encoding="utf-8")
            summary = "done"
            if phase["id"] == "CODE-REVIEW":
                review_attempt["count"] += 1
                summary = (
                    "REVIEW_VERDICT: FAILED"
                    if review_attempt["count"] == 1
                    else "REVIEW_VERDICT: PASSED"
                )
                (agent_workspace / "03-code/review-report.md").write_text(
                    summary, encoding="utf-8"
                )
            artifacts = []
            for relative in phase["declared_artifacts"]:
                raw = (agent_workspace / relative).read_bytes()
                artifacts.append(
                    state.build_artifact_ref(
                        task_id_arg, phase["id"], relative, raw
                    )
                )
            return {
                "node_id": phase["id"],
                "agent_session_id": "session-{0}-{1}".format(phase["id"], index),
                "runtime_id": assigned["runtime_id"],
                "data": {"summary": summary},
                "artifacts": artifacts,
            }

        with tempfile.TemporaryDirectory() as temp_dir:
            workspace = Path(temp_dir)
            agent_workspace = workspace / "agent-workspace"
            node_workspace_root = agent_workspace / "node-workspaces"
            node_workspace_root.mkdir(parents=True)
            (agent_workspace / "TASK.md").write_text("task", encoding="utf-8")
            with patch.object(
                state, "execute_real_agent_phase", side_effect=fake_execute
            ):
                report = state.run_real_agent_state_machine_workflow(
                    task_id,
                    definition,
                    assignment,
                    agent_workspace,
                    node_workspace_root,
                    ["TASK.md"],
                    False,
                    workspace,
                )

            self.assertEqual(
                [node["node_id"] for node in report["nodes"]],
                [
                    "PHASE-0",
                    "TASK-03",
                    "CODE-REVIEW",
                    "TASK-03",
                    "CODE-REVIEW",
                    "SUMMARY",
                ],
            )
            saved_state = json.loads(
                (agent_workspace / "workflow-state.json").read_text(encoding="utf-8")
            )
            self.assertEqual(saved_state["version"], "1.3")
            self.assertEqual(saved_state["current_stage"], "COMPLETED")
            self.assertEqual(saved_state["stages"]["CODE-REVIEW"]["retry_count"], 1)
            self.assertEqual(saved_state["stages"]["CODE-REVIEW"]["review_result"], "PASSED")
            self.assertEqual(len(saved_state["decisions"]), 2)

    def test_state_machine_routes_small_and_medium_to_different_branches(self):
        for size_class, expected_nodes, skipped_node in (
            ("SMALL", ["PHASE-0", "SOLO", "SUMMARY"], "TASK-01"),
            ("MEDIUM", ["PHASE-0", "TASK-01", "SUMMARY"], "SOLO"),
        ):
            with self.subTest(size_class=size_class):
                state = DemoState()
                definition = state.normalize_project_workflow_definition(
                    {
                        "workflow_id": "size-routing",
                        "name": "需求规模分流",
                        "execution_mode": "state_machine",
                        "phases": [
                            {
                                "id": "PHASE-0",
                                "title": "初始化与分级",
                                "artifacts": ["workflow-state.json"],
                                "on_pass": "TASK-01",
                                "on_fail": "SOLO",
                                "decision_mode": "size_class",
                            },
                            {
                                "id": "SOLO",
                                "title": "小需求独立开发",
                                "depends_on": ["PHASE-0"],
                                "artifacts": ["01-solo/solo-report.md"],
                                "on_pass": "SUMMARY",
                            },
                            {
                                "id": "TASK-01",
                                "title": "完整需求分析",
                                "depends_on": ["PHASE-0"],
                                "artifacts": ["01-requirement/report.md"],
                                "on_pass": "SUMMARY",
                            },
                            {
                                "id": "SUMMARY",
                                "title": "汇总",
                                "depends_on": ["SOLO", "TASK-01"],
                                "artifacts": ["workflow-summary.md"],
                                "on_pass": None,
                            },
                        ],
                    }
                )
                task_id = "workflow_size_{0}".format(size_class.lower())
                state.tasks[task_id] = {
                    "attempt_id": "attempt_" + task_id,
                    "prompt": "验证需求分流",
                    "cancel_requested": False,
                    "workflow_phases": [
                        {"id": phase["id"], "status": "ready", "retry_count": 0}
                        for phase in definition["phases"]
                    ],
                    "workflow_current_phase": None,
                    "workflow_current_phases": [],
                    "workflow_stage": "contract_validated",
                    "available_artifacts": [],
                    "events": [],
                    "_workflow_inputs": {},
                }
                assignment = {
                    "mode": "node-routed",
                    "unique_agent_count": 1,
                    "assignments": [
                        {
                            "phase_id": phase["id"],
                            "runtime_id": "hatchery-teamai-codebuddy",
                            "agent_instance_id": "agent-1",
                            "project_agent_id": "project-agent-1",
                            "device_id": "device-1",
                            "transport": "wss+https",
                        }
                        for phase in definition["phases"]
                    ],
                }

                def fake_execute(
                    task_id_arg,
                    definition_arg,
                    phase,
                    assigned,
                    agent_workspace,
                    node_workspace_root,
                    base_artifact_paths,
                    upstream_results,
                    index,
                    is_issuefix,
                ):
                    del (
                        definition_arg,
                        node_workspace_root,
                        base_artifact_paths,
                        upstream_results,
                        is_issuefix,
                    )
                    for relative in phase["declared_artifacts"]:
                        path = agent_workspace / relative
                        path.parent.mkdir(parents=True, exist_ok=True)
                        if relative == "workflow-state.json" and path.exists():
                            continue
                        path.write_text(phase["id"] + " output", encoding="utf-8")
                    summary = (
                        "SIZE_CLASS: " + size_class
                        if phase["id"] == "PHASE-0"
                        else "done"
                    )
                    artifacts = [
                        state.build_artifact_ref(
                            task_id_arg,
                            phase["id"],
                            relative,
                            (agent_workspace / relative).read_bytes(),
                        )
                        for relative in phase["declared_artifacts"]
                    ]
                    return {
                        "node_id": phase["id"],
                        "agent_session_id": "session-{0}-{1}".format(
                            phase["id"], index
                        ),
                        "runtime_id": assigned["runtime_id"],
                        "data": {"summary": summary},
                        "artifacts": artifacts,
                    }

                with tempfile.TemporaryDirectory() as temp_dir:
                    workspace = Path(temp_dir)
                    agent_workspace = workspace / "agent-workspace"
                    node_workspace_root = agent_workspace / "node-workspaces"
                    node_workspace_root.mkdir(parents=True)
                    (agent_workspace / "TASK.md").write_text(
                        "task", encoding="utf-8"
                    )
                    with patch.object(
                        state,
                        "execute_real_agent_phase",
                        side_effect=fake_execute,
                    ):
                        report = state.run_real_agent_state_machine_workflow(
                            task_id,
                            definition,
                            assignment,
                            agent_workspace,
                            node_workspace_root,
                            ["TASK.md"],
                            False,
                            workspace,
                        )

                    self.assertEqual(
                        [node["node_id"] for node in report["nodes"]],
                        expected_nodes,
                    )
                    saved_state = json.loads(
                        (agent_workspace / "workflow-state.json").read_text(
                            encoding="utf-8"
                        )
                    )
                    self.assertEqual(saved_state["size_class"], size_class.lower())
                    self.assertEqual(
                        saved_state["stages"][skipped_node]["status"], "skipped"
                    )
                    self.assertEqual(
                        next(
                            phase["status"]
                            for phase in state.tasks[task_id]["workflow_phases"]
                            if phase["id"] == skipped_node
                        ),
                        "skipped",
                    )

    def test_first_node_handoff_contains_concrete_task_input_values(self):
        state = DemoState()
        task_id = "workflow_inputs"
        state.tasks[task_id] = {
            "attempt_id": "attempt_inputs",
            "prompt": "修改项目协作页",
            "_workflow_inputs": state.normalize_workflow_inputs(
                {
                    "requirement": {"type": "markdown", "value": "展开节点详情"},
                    "repository_url": {
                        "type": "url",
                        "value": "https://git.woa.com/cvm-openclaw/openclaw-enterprise",
                    },
                    "target_page": {
                        "type": "text",
                        "value": "/project-collaboration",
                    },
                }
            ),
        }
        definition = state.normalize_project_workflow_definition(
            {
                "workflow_id": "frontend-demo",
                "phases": [
                    {
                        "id": "develop",
                        "inputs": [
                            {"key": "requirement", "type": "markdown"},
                            {"key": "repository_url", "type": "url"},
                            {"key": "target_page", "type": "text"},
                        ],
                        "artifacts": ["frontend-demo.md"],
                    }
                ],
            }
        )
        envelope = state.build_node_input_v2(
            task_id,
            definition,
            definition["phases"][0],
            {
                "agent_instance_id": "codebuddy-local",
                "runtime_id": "hatchery-teamai-codebuddy",
            },
            None,
            None,
            [],
            1,
        )

        self.assertEqual(
            envelope["inputs"]["data"]["repository_url"],
            "https://git.woa.com/cvm-openclaw/openclaw-enterprise",
        )
        self.assertEqual(
            envelope["inputs"]["data"]["target_page"],
            "/project-collaboration",
        )
        self.assertEqual(
            envelope["inputs"]["mappings"]["repository_url"],
            "$task.inputs.repository_url.value",
        )
        prompt = state.build_real_node_prompt(
            definition["phases"][0],
            state.tasks[task_id]["prompt"],
            "input.json",
            definition["phases"][0]["declared_artifacts"],
            input_envelope=envelope,
        )
        self.assertIn("受控真实源码工作区", prompt)
        self.assertIn("git remote get-url origin", prompt)
        self.assertIn("禁止在仓库外另建独立 Demo", prompt)

    def test_normalizes_parallel_dag_and_rejects_cycles(self):
        state = DemoState()
        definition = state.normalize_project_workflow_definition(
            {
                "workflow_id": "image-release",
                "phases": [
                    {"id": "build", "artifacts": ["build.json"], "depends_on": []},
                    {"id": "qa-a", "artifacts": ["qa-a.md"], "depends_on": ["build"]},
                    {"id": "qa-b", "artifacts": ["qa-b.md"], "depends_on": ["build"]},
                    {
                        "id": "release",
                        "artifacts": ["release.md"],
                        "depends_on": ["qa-a", "qa-b"],
                    },
                ],
            }
        )

        self.assertEqual(definition["schema_version"], "clawpro.project-workflow.v2")
        self.assertEqual(definition["phases"][1]["depends_on"], ["build"])
        self.assertEqual(definition["phases"][3]["depends_on"], ["qa-a", "qa-b"])

        with self.assertRaisesRegex(BridgeError, "循环依赖"):
            state.normalize_project_workflow_definition(
                {
                    "workflow_id": "cycle",
                    "phases": [
                        {"id": "a", "depends_on": ["b"]},
                        {"id": "b", "depends_on": ["a"]},
                    ],
                }
            )

    def test_dag_scheduler_runs_ready_nodes_concurrently_and_joins_all(self):
        state = DemoState()
        phases = [
            {"id": "build", "depends_on": [], "approval": None},
            {"id": "qa-a", "depends_on": ["build"], "approval": None},
            {"id": "qa-b", "depends_on": ["build"], "approval": None},
            {"id": "release", "depends_on": ["qa-a", "qa-b"], "approval": None},
        ]
        definition = {"phases": phases}
        assignments = [
            {
                "phase_id": phase["id"],
                "runtime_id": "hatchery-teamai-codebuddy",
            }
            for phase in phases
        ]
        assignment = {
            "mode": "mixed",
            "assignments": assignments,
            "unique_agent_count": 2,
        }
        state.tasks["workflow_dag"] = {
            "id": "workflow_dag",
            "attempt_id": "attempt_dag",
            "status": "running",
            "execution_status": "running",
            "workflow_stage": "real_agent_running",
            "cancel_requested": False,
            "events": [],
            "approval_history": [],
            "pending_approval": None,
            "workflow_phases": [
                {"id": phase["id"], "status": "ready"} for phase in phases
            ],
            "available_artifacts": [],
            "workflow_current_phase": None,
            "workflow_current_phases": [],
        }
        qa_barrier = threading.Barrier(2)
        started = {}
        received_upstream = {}
        lock = threading.Lock()

        def fake_execute(
            task_id,
            workflow_definition,
            phase,
            assigned,
            agent_workspace,
            node_workspace_root,
            base_artifact_paths,
            upstream_results,
            index,
            is_issuefix,
        ):
            with lock:
                started[phase["id"]] = time.monotonic()
                received_upstream[phase["id"]] = [
                    item["node_id"] for item in upstream_results
                ]
            if phase["id"].startswith("qa-"):
                qa_barrier.wait(timeout=1)
                time.sleep(0.02)
            return {
                "node_id": phase["id"],
                "agent_session_id": "session-" + phase["id"],
                "runtime_id": assigned["runtime_id"],
                "data": {"summary": phase["id"] + " done"},
                "artifacts": [],
            }

        with tempfile.TemporaryDirectory() as temp_dir, patch.object(
            state, "execute_real_agent_phase", side_effect=fake_execute
        ):
            root = Path(temp_dir)
            report = state.run_real_agent_dag_workflow(
                "workflow_dag",
                definition,
                assignment,
                root,
                root / "nodes",
                [],
                False,
                root,
            )

        self.assertLess(abs(started["qa-a"] - started["qa-b"]), 0.2)
        self.assertEqual(set(received_upstream["release"]), {"qa-a", "qa-b"})
        self.assertEqual(report["handoff_count"], 4)
        self.assertEqual(report["agent_session_count"], 4)

    def test_required_evidence_rejects_simulated_or_unverified_output(self):
        phase = {
            "id": "scan",
            "required_evidence": ["iwiki.metadata", "4025707654"],
            "reject_output_markers": ["模拟"],
        }

        DemoState.validate_required_evidence(
            phase,
            "已调用 iwiki.metadata，根 docid 4025707654，返回真实目录。",
        )
        with self.assertRaisesRegex(RuntimeError, "缺少真实读取证据"):
            DemoState.validate_required_evidence(phase, "已读取知识库。")
        with self.assertRaisesRegex(RuntimeError, "未完成真实读取"):
            DemoState.validate_required_evidence(
                phase,
                "iwiki.metadata 读取 4025707654，以下为模拟结果。",
            )

    def test_required_evidence_accepts_actual_mcp_tool_names(self):
        phase = {
            "id": "kb-scan",
            "required_evidence": [
                "iwiki.metadata",
                "iwiki.getSpacePageTree",
                "4025707654",
            ],
            "reject_output_markers": ["模拟"],
        }

        DemoState.validate_required_evidence(
            phase,
            (
                "已调用 mcp__iwiki__metadata 读取根 docid 4025707654，"
                "并调用 mcp__iwiki__getSpacePageTree 递归读取目录。"
            ),
        )

    def test_codebuddy_capability_depends_on_teamai_user_authorization(self):
        with patch.dict("os.environ", {}, clear=True):
            status = codebuddy_capability_status()
            self.assertTrue(status["configured"] is False)
            self.assertEqual(status["capabilities"], [])
            self.assertEqual(status["missing_capabilities"], ["iwiki.read"])

        with patch.dict("os.environ", {"TAI_PAT_TOKEN": "secret"}, clear=True):
            status = codebuddy_capability_status()
            self.assertTrue(status["configured"])
            self.assertEqual(status["capabilities"], ["iwiki.read"])
            self.assertNotIn("secret", str(status))

    def test_real_node_prompt_allows_only_authorized_iwiki_mcp(self):
        prompt = DemoState.build_real_node_prompt(
            {
                "id": "kb-scan",
                "title": "全量读取与链接图",
                "agent_id": "reader",
                "agent_title": "知识库读取",
                "agent_description": "递归读取 iWiki",
                "node_plan": [],
                "required_capabilities": ["iwiki.read"],
            },
            "巡检知识库",
            "input.json",
            ["pages.json", "link_graph.json"],
        )

        self.assertIn("mcp__iwiki__getSpacePageTree", prompt)
        self.assertIn("必须通过 TeamAI 注入的只读 iWiki MCP 真实读取数据", prompt)
        self.assertIn("禁止公共网络访问和任何外部写操作", prompt)
        self.assertNotIn("不访问网络，不提交 Git", prompt)
        self.assertNotIn("仅限 Read/Write/Edit/Glob/Grep", prompt)

    def test_real_node_prompt_keeps_external_access_closed_without_capability(self):
        prompt = DemoState.build_real_node_prompt(
            {
                "id": "report",
                "title": "报告生成",
                "agent_id": "writer",
                "agent_title": "报告生成",
                "agent_description": "汇总本地交接产物",
                "node_plan": [],
                "required_capabilities": [],
            },
            "生成报告",
            "input.json",
            ["report.md"],
        )

        self.assertIn("未声明外部读取能力", prompt)
        self.assertNotIn("mcp__iwiki__getDocument", prompt)

    def test_real_node_prompt_inlines_handoff_for_remote_teamai(self):
        with tempfile.TemporaryDirectory() as temp_dir:
            workspace = Path(temp_dir)
            (workspace / "upstream.md").write_text("上游真实结论", encoding="utf-8")
            prompt = DemoState.build_real_node_prompt(
                {
                    "id": "report",
                    "title": "报告生成",
                    "agent_id": "writer",
                    "agent_title": "报告生成",
                    "agent_description": "承接上游结论",
                    "node_plan": [],
                    "required_capabilities": [],
                },
                "生成报告",
                "input.json",
                ["report.md"],
                input_envelope={
                    "schema_version": "clawpro.node-input.v2",
                    "inputs": {"artifacts": [{"path": "upstream.md"}]},
                },
                agent_workspace=workspace,
            )

        self.assertIn("跨机器交接包", prompt)
        self.assertIn("clawpro.node-input.v2", prompt)
        self.assertIn("上游真实结论", prompt)

    def test_real_node_prompt_deduplicates_task_goal_and_task_markdown(self):
        with tempfile.TemporaryDirectory() as temp_dir:
            workspace = Path(temp_dir)
            (workspace / "TASK.md").write_text("重复的完整任务正文", encoding="utf-8")
            prompt = DemoState.build_real_node_prompt(
                {
                    "id": "develop",
                    "title": "前端开发",
                    "agent_id": "developer",
                    "agent_title": "前端开发",
                    "agent_description": "修改真实源码",
                    "node_plan": [],
                    "required_capabilities": [],
                },
                "唯一用户任务",
                "input.json",
                ["result.md"],
                input_envelope={
                    "schema_version": "clawpro.node-input.v2",
                    "task": {"goal": "唯一用户任务", "inputs": {}},
                    "inputs": {"artifacts": [{"path": "TASK.md"}]},
                },
                agent_workspace=workspace,
            )

        self.assertEqual(prompt.count("唯一用户任务"), 1)
        self.assertNotIn("重复的完整任务正文", prompt)
        self.assertIn("见上方“用户任务”", prompt)

    def test_remote_bridge_bootstrap_keeps_teamai_on_user_device(self):
        with tempfile.TemporaryDirectory() as temp_dir, patch.dict(
            "os.environ",
            {
                "TEAMAI_REMOTE_MODE": "1",
                "TEAMAI_REMOTE_WORKSPACE": "/srv/clawpro/remote-workspace",
                "TEAMAI_PUBLIC_ENDPOINT": "https://clawpro.example.test",
            },
            clear=True,
        ):
            bridge = HatcheryTeamAIBridge(
                "http://127.0.0.1:8091", "admin", Path(temp_dir)
            )
            bridge.user_id = 1
            bridge.user_token = "user-token"
            bridge.project_id = 2
            bridge.project_name = "部署验证"

            bridge.write_remote_bootstrap()
            data = json.loads(bridge.bootstrap_path.read_text(encoding="utf-8"))

            self.assertTrue(bridge.remote_mode)
            self.assertEqual(data["endpoint"], "https://clawpro.example.test")
            self.assertEqual(data["workspace"], "/srv/clawpro/remote-workspace")
            self.assertEqual(data["token"], "user-token")
            self.assertIsNone(bridge.start_resident_listener(Path(temp_dir)))


class WorkflowApprovalTests(unittest.TestCase):
    def test_waits_for_matching_manual_approval_before_next_phase(self):
        state = DemoState()
        task_id = "workflow_approval"
        state.tasks[task_id] = {
            "task_id": task_id,
            "attempt_id": "attempt_approval",
            "status": "running",
            "execution_status": "running",
            "workflow_stage": "real_agent_running",
            "workflow_current_phase": "analyze",
            "workflow_phases": [
                {"id": "analyze", "status": "completed"},
                {"id": "fix", "status": "ready"},
            ],
            "pending_approval": None,
            "approval_history": [],
            "cancel_requested": False,
            "events": [],
            "updated_at": state.now(),
        }
        phase = {"id": "analyze", "title": "分析", "on_pass": "fix"}
        output = {
            "summary": "根因和修复方案已确认。",
            "artifacts": [{"path": "fix-plan.md", "size": 32, "sha256": "abc"}],
        }
        failures = []

        def wait_for_approval():
            try:
                state.wait_for_workflow_approval(task_id, phase, "fix", output)
            except Exception as error:  # pragma: no cover - surfaced by assertion
                failures.append(error)

        worker = threading.Thread(target=wait_for_approval)
        worker.start()
        deadline = time.monotonic() + 2
        while time.monotonic() < deadline:
            if state.tasks[task_id]["status"] == "waiting_approval":
                break
            time.sleep(0.01)

        task = state.tasks[task_id]
        self.assertEqual(task["workflow_stage"], "awaiting_approval")
        self.assertEqual(task["workflow_phases"][0]["status"], "awaiting_approval")
        gate_id = task["pending_approval"]["gate_id"]
        state.approve_workflow(task_id, gate_id)
        worker.join(timeout=2)

        self.assertFalse(worker.is_alive())
        self.assertEqual(failures, [])
        self.assertEqual(task["status"], "running")
        self.assertEqual(task["workflow_current_phase"], "fix")
        self.assertIsNone(task["pending_approval"])
        self.assertEqual(task["workflow_phases"][0]["status"], "completed")
        self.assertEqual(len(task["approval_history"]), 1)
        self.assertEqual(task["approval_history"][0]["gate_id"], "approve-fix-plan")

    def test_rejects_stale_or_wrong_gate(self):
        state = DemoState()
        task_id = "workflow_wrong_gate"
        state.tasks[task_id] = {
            "task_id": task_id,
            "attempt_id": "attempt_wrong_gate",
            "status": "waiting_approval",
            "execution_status": "waiting_approval",
            "workflow_stage": "awaiting_approval",
            "workflow_current_phase": "test",
            "workflow_phases": [],
            "pending_approval": {
                "gate_id": "approve-mr-stage",
                "status": "pending",
            },
            "approval_history": [],
            "cancel_requested": False,
            "events": [],
            "updated_at": state.now(),
        }

        with self.assertRaisesRegex(ValueError, "确认点已变化"):
            state.approve_workflow(task_id, "approve-fix-plan")

    def test_cancel_stops_waiting_workflow_without_starting_next_phase(self):
        state = DemoState()
        task_id = "workflow_cancel"
        state.tasks[task_id] = {
            "task_id": task_id,
            "attempt_id": "attempt_cancel",
            "status": "running",
            "execution_status": "running",
            "workflow_stage": "real_agent_running",
            "workflow_current_phase": "analyze",
            "workflow_phases": [
                {"id": "analyze", "status": "completed"},
                {"id": "fix", "status": "ready"},
            ],
            "pending_approval": None,
            "approval_history": [],
            "cancel_requested": False,
            "events": [],
            "updated_at": state.now(),
        }
        phase = {"id": "analyze", "title": "分析", "on_pass": "fix"}
        output = {
            "summary": "等待用户确认。",
            "artifacts": [{"path": "fix-plan.md", "size": 32, "sha256": "abc"}],
        }
        failures = []

        def wait_for_approval():
            try:
                state.wait_for_workflow_approval(task_id, phase, "fix", output)
            except Exception as error:  # pragma: no cover - asserted below
                failures.append(error)

        worker = threading.Thread(target=wait_for_approval)
        worker.start()
        deadline = time.monotonic() + 2
        while time.monotonic() < deadline:
            if state.tasks[task_id]["status"] == "waiting_approval":
                break
            time.sleep(0.01)

        state.cancel_task(task_id)
        worker.join(timeout=2)

        task = state.tasks[task_id]
        self.assertFalse(worker.is_alive())
        self.assertEqual(len(failures), 1)
        self.assertIsInstance(failures[0], TaskCanceled)
        self.assertEqual(task["status"], "canceled")
        self.assertEqual(task["workflow_stage"], "canceled")
        self.assertIsNone(task["pending_approval"])
        self.assertEqual(task["workflow_phases"][0]["status"], "canceled")
        self.assertEqual(task["workflow_phases"][1]["status"], "ready")

    def test_cancel_stops_running_phase_when_parent_was_marked_failed(self):
        state = DemoState()
        task_id = "workflow_failed_with_live_child"
        state.tasks[task_id] = {
            "task_id": task_id,
            "attempt_id": "attempt_failed_live_child",
            "status": "failed",
            "execution_status": "failed",
            "workflow_stage": "failed",
            "workflow_current_phase": "review",
            # Reproduces a persisted stale snapshot: the singular current phase
            # is newer than the parallel-phase list.
            "workflow_current_phases": ["analyze"],
            "workflow_phases": [
                {"id": "analyze", "status": "completed"},
                {"id": "review", "status": "running"},
                {"id": "test", "status": "ready"},
            ],
            "pending_approval": None,
            "approval_history": [],
            "cancel_requested": False,
            "events": [],
            "updated_at": state.now(),
        }

        state.cancel_task(task_id)

        task = state.tasks[task_id]
        self.assertTrue(task["cancel_requested"])
        self.assertEqual(task["status"], "canceled")
        self.assertEqual(task["execution_status"], "canceled")
        self.assertEqual(task["workflow_stage"], "canceled")
        self.assertEqual(task["workflow_phases"][1]["status"], "canceled")
        self.assertEqual(task["workflow_phases"][2]["status"], "ready")


class WorkflowFailureStateTests(unittest.TestCase):
    def test_marks_running_nodes_failed_and_clears_active_phase(self):
        state = DemoState()
        task_id = "workflow_failure"
        state.tasks[task_id] = {
            "task_id": task_id,
            "status": "running",
            "execution_status": "running",
            "workflow_stage": "real_agent_running",
            "workflow_current_phase": "verify",
            "workflow_current_phases": ["verify"],
            "workflow_phases": [
                {"id": "analyze", "status": "completed"},
                {"id": "verify", "status": "running"},
            ],
            "pending_approval": None,
            "events": [],
            "updated_at": state.now(),
        }

        changed = state.mark_workflow_failed(task_id, "duplicate task lock")

        task = state.tasks[task_id]
        self.assertTrue(changed)
        self.assertEqual(task["status"], "failed")
        self.assertEqual(task["workflow_stage"], "failed")
        self.assertEqual(task["workflow_current_phases"], [])
        self.assertEqual(task["workflow_phases"][0]["status"], "completed")
        self.assertEqual(task["workflow_phases"][1]["status"], "failed")
        self.assertEqual(task["workflow_phases"][1]["error"], "duplicate task lock")


if __name__ == "__main__":
    unittest.main()
