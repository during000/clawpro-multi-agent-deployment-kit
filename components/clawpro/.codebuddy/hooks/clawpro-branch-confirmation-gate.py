#!/usr/bin/env python3
"""Block ClawPro mutations until the user confirms the routed feature branch."""

from __future__ import annotations

import json
import hashlib
import os
import re
import subprocess
import sys
import tempfile
import time
from pathlib import Path
from typing import Any


REPO_ROOT = Path(__file__).resolve().parents[2]
CONFIRMATIONS = {
    "确认",
    "继续",
    "可以",
    "同意",
    "确认继续",
    "确认，可以继续",
    "确认继续开发",
}
MARKER = "【分支匹配待确认】"
BRANCH_PATTERN = re.compile(r"^分支：(?P<branch>feature/[^\s]+)$")
CACHE_DIR = Path(tempfile.gettempdir()) / "clawpro-codebuddy-hook-cache"
CACHE_TTL_SECONDS = 300


def emit_decision(decision: str, reason: str) -> None:
    print(
        json.dumps(
            {
                "continue": decision != "deny",
                **({"stopReason": reason} if decision == "deny" else {}),
                "hookSpecificOutput": {
                    "hookEventName": "PreToolUse",
                    "permissionDecision": decision,
                    "permissionDecisionReason": reason,
                }
            },
            ensure_ascii=False,
        )
    )


def content_text(value: Any) -> str:
    if isinstance(value, str):
        return value
    if isinstance(value, list):
        return "\n".join(content_text(item) for item in value)
    if isinstance(value, dict):
        parts: list[str] = []
        for key in ("text", "content", "input_text", "output_text"):
            if key in value:
                parts.append(content_text(value[key]))
        return "\n".join(part for part in parts if part)
    return ""


def read_messages(transcript_path: str) -> list[dict[str, str]]:
    messages: list[dict[str, str]] = []
    try:
        with Path(transcript_path).open(encoding="utf-8") as transcript:
            for line in transcript:
                try:
                    item = json.loads(line)
                except json.JSONDecodeError:
                    continue
                if item.get("type") != "message" or item.get("role") not in {"user", "assistant"}:
                    continue
                messages.append(
                    {
                        "role": item["role"],
                        "text": content_text(item.get("content", "")).strip(),
                    }
                )
    except OSError:
        return []
    return messages


def route_card_branch(text: str) -> str | None:
    lines = [line.strip() for line in text.strip().splitlines() if line.strip()]
    for index in range(len(lines) - 2):
        if lines[index] != MARKER or lines[index + 2] != "请确认是否在该分支继续。":
            continue
        match = BRANCH_PATTERN.fullmatch(lines[index + 1])
        if match:
            return match.group("branch")
    return None


def confirmed_route(messages: list[dict[str, str]]) -> str | None:
    """Return the latest explicitly approved route for this conversation.

    Approval remains valid for later prompts in the same conversation. A newer
    route card supersedes it and must be explicitly confirmed before writes can
    continue. Repository and actual-branch checks below keep the approval scoped.
    """
    latest_card_index: int | None = None
    latest_branch: str | None = None
    for index, message in enumerate(messages):
        if message["role"] != "assistant":
            continue
        branch = route_card_branch(message["text"])
        if branch:
            latest_card_index = index
            latest_branch = branch
    if latest_card_index is None or latest_branch is None:
        return None
    next_user = next(
        (
            message["text"].strip("。！! ")
            for message in messages[latest_card_index + 1 :]
            if message["role"] == "user"
        ),
        None,
    )
    return latest_branch if next_user in CONFIRMATIONS else None


def target_path(payload: dict[str, Any]) -> Path | None:
    tool_input = payload.get("tool_input") or {}
    for key in ("file_path", "path", "notebook_path"):
        value = tool_input.get(key)
        if isinstance(value, str) and value:
            path = Path(value).expanduser()
            if not path.is_absolute():
                path = Path(payload.get("cwd") or ".") / path
            return path.resolve()
    return None


def repo_branch(path: Path) -> tuple[Path, str] | None:
    probe = path if path.is_dir() else path.parent
    result = subprocess.run(
        ["git", "-C", str(probe), "rev-parse", "--show-toplevel"],
        capture_output=True,
        text=True,
        check=False,
    )
    if result.returncode != 0:
        return None
    repo = Path(result.stdout.strip()).resolve()
    branch_result = subprocess.run(
        ["git", "-C", str(repo), "branch", "--show-current"],
        capture_output=True,
        text=True,
        check=False,
    )
    if branch_result.returncode != 0:
        return None
    return repo, branch_result.stdout.strip()


def is_repo_path(path: Path) -> bool:
    try:
        path.resolve().relative_to(REPO_ROOT)
        return True
    except ValueError:
        return False


def is_safe_routing_bash(command: str) -> bool:
    normalized = " ".join(command.strip().split())
    safe_patterns = (
        r"^python3 \.codebuddy/scripts/sync-clawpro-branches\.py(?: --origin origin)?$",
        r"^git (status|branch|remote|rev-parse|diff|log|show|ls-files|ls-remote|check-ignore)( |$)",
        r"^git fetch( |$)",
        r"^git switch (feature/[^ ;&|]+|--track origin/feature/[^ ;&|]+)$",
        r"^(pwd|ls|find|rg|head|tail|wc)( |$)",
        r"^sed -n (.+)$",
    )
    return any(re.match(pattern, normalized) for pattern in safe_patterns)


def requires_fresh_remote_check(tool_name: str, tool_input: dict[str, Any]) -> bool:
    """Critical Git operations always bypass the short-lived remote cache."""
    if "Bash" not in tool_name and "Shell" not in tool_name:
        return False
    command = str(tool_input.get("command") or tool_input.get("cmd") or "")
    return bool(
        re.search(
            r"\bgit(?:\s+-C\s+(?:\"[^\"]+\"|'[^']+'|\S+))?\s+"
            r"(?:commit|push|rebase|merge|cherry-pick|reset|clean|tag)\b",
            command,
        )
    )


def remote_cache_path(
    transcript_path: str,
    repo: Path,
    branch: str,
    tracking_head: str,
) -> Path:
    identity = "\0".join((transcript_path, str(repo), branch, tracking_head))
    digest = hashlib.sha256(identity.encode("utf-8")).hexdigest()
    return CACHE_DIR / f"{digest}.ok"


def remote_cache_is_fresh(cache_path: Path, now: float | None = None) -> bool:
    try:
        checked_at = cache_path.stat().st_mtime
    except OSError:
        return False
    current = time.time() if now is None else now
    return current - checked_at <= CACHE_TTL_SECONDS


def refresh_remote_cache(cache_path: Path) -> None:
    try:
        CACHE_DIR.mkdir(mode=0o700, parents=True, exist_ok=True)
        cache_path.write_text("verified\n", encoding="utf-8")
        cache_path.chmod(0o600)
    except OSError:
        # Cache failure must not weaken validation; it only makes later calls slower.
        return


def self_test() -> int:
    valid = [
        {
            "role": "assistant",
            "text": (
                "【分支匹配待确认】\n"
                "分支：feature/admin-standards-library\n"
                "请确认是否在该分支继续。"
            ),
        },
        {"role": "user", "text": "确认"},
    ]
    invalid = [
        {"role": "assistant", "text": "我准备开始修改。"},
        {"role": "user", "text": "确认"},
    ]
    verbose = [
        {
            "role": "assistant",
            "text": (
                "已定位页面。\n"
                "【分支匹配待确认】\n"
                "分支：feature/admin-standards-library\n"
                "请确认是否在该分支继续。"
            ),
        },
        {"role": "user", "text": "确认"},
    ]
    continued = valid + [
        {"role": "assistant", "text": "已完成第一处调整。"},
        {"role": "user", "text": "再把同一个页面的提示文案改短一点"},
    ]
    reroute_pending = continued + [
        {
            "role": "assistant",
            "text": (
                "【分支匹配待确认】\n"
                "分支：feature/admin-tool-plugin\n"
                "请确认是否在该分支继续。"
            ),
        },
        {"role": "user", "text": "先等等"},
    ]
    reroute_confirmed = reroute_pending[:-1] + [{"role": "user", "text": "确认"}]
    cache_probe = CACHE_DIR / "self-test.ok"
    refresh_remote_cache(cache_probe)
    cache_passed = remote_cache_is_fresh(cache_probe)
    try:
        cache_probe.unlink()
    except OSError:
        pass
    passed = (
        confirmed_route(valid) == "feature/admin-standards-library"
        and confirmed_route(invalid) is None
        and confirmed_route(verbose) == "feature/admin-standards-library"
        and confirmed_route(continued) == "feature/admin-standards-library"
        and confirmed_route(reroute_pending) is None
        and confirmed_route(reroute_confirmed) == "feature/admin-tool-plugin"
        and is_safe_routing_bash("git fetch origin")
        and is_safe_routing_bash("git ls-remote --heads origin")
        and is_safe_routing_bash("python3 .codebuddy/scripts/sync-clawpro-branches.py --origin origin")
        and not is_safe_routing_bash("pnpm build")
        and requires_fresh_remote_check("Bash", {"command": "git commit -m test"})
        and requires_fresh_remote_check("Shell", {"cmd": "pnpm check && git push origin feature/x"})
        and not requires_fresh_remote_check("Bash", {"command": "pnpm build"})
        and cache_passed
    )
    print("PASS" if passed else "FAIL")
    return 0 if passed else 1


def main() -> int:
    if len(sys.argv) > 1 and sys.argv[1] == "--self-test":
        return self_test()
    try:
        payload = json.load(sys.stdin)
    except (json.JSONDecodeError, OSError):
        emit_decision("deny", "ClawPro 门禁无法读取工具调用信息，已阻止写操作。")
        return 0

    cwd = Path(payload.get("cwd") or ".").resolve()
    destination = target_path(payload)
    if not is_repo_path(destination or cwd):
        return 0

    tool_name = str(payload.get("tool_name") or "")
    tool_input = payload.get("tool_input") or {}
    if "Bash" in tool_name or "Shell" in tool_name:
        command = str(tool_input.get("command") or tool_input.get("cmd") or "")
        if is_safe_routing_bash(command):
            return 0

    messages = read_messages(str(payload.get("transcript_path") or ""))
    selected_branch = confirmed_route(messages)
    if not selected_branch:
        emit_decision(
            "deny",
            "当前对话尚无有效的 ClawPro 分支确认，或已有新的待确认分支。请只输出目标 feature 分支的【分支匹配待确认】卡片，并等待用户明确确认。",
        )
        return 0

    probe = destination or cwd
    branch_info = repo_branch(probe)
    if not branch_info:
        emit_decision(
            "deny",
            "无法确认本次写操作所属的 Git 仓库。请将工具工作目录设置为已确认的目标仓库后重试。",
        )
        return 0
    repo, actual_branch = branch_info
    if actual_branch != selected_branch:
        emit_decision(
            "deny",
            f"分支不一致：用户确认的是 {selected_branch}，但工具目标位于 {repo} 的 {actual_branch}。禁止修改。",
        )
        return 0
    remote_ref_result = subprocess.run(
        ["git", "-C", str(repo), "rev-parse", f"refs/remotes/origin/{selected_branch}"],
        capture_output=True,
        text=True,
        check=False,
    )
    tracking_head = remote_ref_result.stdout.strip().lower()
    if remote_ref_result.returncode != 0 or not tracking_head:
        emit_decision(
            "deny",
            "缺少目标分支的本地远端引用。请执行 git fetch origin --prune 后重试。",
        )
        return 0
    cache_path = remote_cache_path(
        str(payload.get("transcript_path") or ""),
        repo,
        selected_branch,
        tracking_head,
    )
    if (
        not requires_fresh_remote_check(tool_name, tool_input)
        and remote_cache_is_fresh(cache_path)
    ):
        return 0
    remote_env = dict(os.environ)
    remote_env["GIT_TERMINAL_PROMPT"] = "0"
    try:
        live_result = subprocess.run(
            [
                "git",
                "-C",
                str(repo),
                "ls-remote",
                "--heads",
                "origin",
                f"refs/heads/{selected_branch}",
            ],
            capture_output=True,
            text=True,
            check=False,
            timeout=6,
            env=remote_env,
        )
    except subprocess.TimeoutExpired:
        emit_decision("deny", "工蜂实时分支校验超时，已阻止写操作；请检查网络后重试。")
        return 0
    live_fields = live_result.stdout.strip().split()
    live_head = live_fields[0].lower() if live_result.returncode == 0 and live_fields else ""
    if not live_head:
        emit_decision("deny", "工蜂未返回已确认的远端分支，已阻止写操作。请重新执行分支路由。")
        return 0
    if tracking_head != live_head:
        emit_decision(
            "deny",
            "目标分支在工蜂已更新。请执行 git fetch origin --prune，同步后在同一分支重试；无需再次展示 SHA 或时间。",
        )
        return 0
    refresh_remote_cache(cache_path)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
