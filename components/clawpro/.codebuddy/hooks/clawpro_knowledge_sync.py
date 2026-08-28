#!/usr/bin/env python3
"""Sync the Gongfeng ClawPro knowledge branch before every user prompt.

The hook fetches ``harness-knowledge-store`` without switching the current
worktree, exports only the enterprise knowledge/spec directories to a
user-local cache, and injects the cache path into the current task.

Personal/session memory is intentionally excluded even if it appears on the
source branch.
"""

from __future__ import annotations

import argparse
import fcntl
import hashlib
import io
import json
import os
import re
import shutil
import subprocess
import sys
import tarfile
import tempfile
from datetime import datetime, timezone
from pathlib import Path, PurePosixPath
from typing import Any


SOURCE_BRANCH = "harness-knowledge-store"
SYNC_REF = "refs/clawpro-harness/harness-knowledge-store"
STORE_ROOT = "harness-knowledge-store"
ALLOWED_PATHS = (
    f"{STORE_ROOT}/README.md",
    f"{STORE_ROOT}/clawpro",
    f"{STORE_ROOT}/specs",
)
BLOCKED_NAMES = {"MEMORY.md", "USER.md", "IDENTITY.md", "memory"}
RECALL_EXCLUDED_PARTS = {"_inbox", ".internal", "_meta"}
TEXT_SUFFIXES = {".md", ".txt", ".json", ".yaml", ".yml"}
MAX_DOCUMENT_BYTES = 1024 * 1024
KNOWLEDGE_ID_RE = re.compile(r"^- 知识 ID：`([^`]+)`\s*$", re.MULTILINE)
KNOWLEDGE_STATUS_RE = re.compile(r"^- 状态：`([^`]+)`\s*$", re.MULTILINE)
SUPERSEDES_RE = re.compile(r"^- 替代知识：(.+?)\s*$", re.MULTILINE)
GENERIC_QUERY_TERMS = {
    "agent",
    "这里",
    "什么",
    "现在",
    "一下",
    "一个",
    "这个",
    "那个",
    "哪些",
    "可以",
    "是否",
    "如何",
    "怎么",
    "我们",
    "你们",
    "他们",
    "知道",
    "知识",
    "读取",
    "召回",
}
DOMAIN_KEYWORDS = (
    "userpromptsubmit",
    "harness-knowledge-store",
    "openclaw",
    "clawpro",
    "teamai",
    "oneid",
    "token",
    "agent",
    "hook",
    "本地 agent",
    "云端 agent",
    "企业知识库",
    "企业规范库",
    "企业插件",
    "企业技能",
    "用户权限",
    "用户管理",
    "项目协作",
    "工作流",
    "多租户",
    "智能体",
    "知识库",
    "工具库",
    "插件",
    "技能",
    "模型",
    "通道",
    "租户",
    "实例",
    "镜像",
    "权限",
    "计费",
    "收费",
    "配额",
    "审计",
    "安全",
    "网络",
    "记忆",
    "安装",
    "触发",
    "接管",
    "企微",
    "飞书",
    "mcp",
    "vpc",
    "cvm",
)
DOMAIN_SIGNALS = {
    keyword
    for keyword in DOMAIN_KEYWORDS
    if keyword
    not in {
        "agent",
        "安装",
        "权限",
        "安全",
        "网络",
    }
}
QUESTION_FILLER_RE = re.compile(
    r"帮我|请问|查询|看下|看一下|告诉我|为什么|怎么|如何|是否|可以|"
    r"支持|哪些|什么|知道|读取|召回|给我|一下"
)
DEFAULT_CACHE_ROOT = Path.home() / ".clawpro-harness" / "gongfeng-knowledge-cache"
PROJECT_ROOT = Path(
    os.environ.get("CODEBUDDY_PROJECT_DIR", Path(__file__).resolve().parents[2])
).expanduser().resolve()
ORIGIN_PATTERN = re.compile(
    r"^(?:(?:ssh|https?)://(?:git@)?|git@)git\.woa\.com[:/]"
    r"cvm-openclaw/openclaw-enterprise(?:\.git)?/?$",
    re.IGNORECASE,
)


class SyncError(RuntimeError):
    """A safe, user-actionable knowledge sync failure."""


def utc_now() -> str:
    return datetime.now(timezone.utc).isoformat()


def emit(payload: dict[str, object]) -> None:
    print(json.dumps(payload, ensure_ascii=False))


def allow_silently() -> None:
    emit({"continue": True, "suppressOutput": True})


def additional_context(message: str) -> None:
    emit(
        {
            "continue": True,
            "hookSpecificOutput": {
                "hookEventName": "UserPromptSubmit",
                "additionalContext": message,
            },
        }
    )


def run_git(repo: Path, args: list[str], timeout: int = 40) -> str:
    try:
        completed = subprocess.run(
            ["git", "-C", str(repo), *args],
            text=True,
            capture_output=True,
            timeout=timeout,
            check=False,
        )
    except (OSError, subprocess.TimeoutExpired) as exc:
        raise SyncError(f"git {' '.join(args[:2])} failed: {exc}") from exc
    if completed.returncode != 0:
        detail = (completed.stderr or completed.stdout).strip()
        raise SyncError(detail[-600:] or f"git {' '.join(args[:2])} exited {completed.returncode}")
    return completed.stdout.strip()


def is_expected_origin(url: str) -> bool:
    normalized = url.strip().replace("\\", "/")
    return bool(ORIGIN_PATTERN.search(normalized))


def validate_project(repo: Path) -> None:
    root = run_git(repo, ["rev-parse", "--show-toplevel"])
    if Path(root).resolve() != repo:
        raise SyncError(f"CODEBUDDY_PROJECT_DIR is not the repository root: {repo}")
    origin = run_git(repo, ["remote", "get-url", "origin"])
    if not is_expected_origin(origin):
        raise SyncError("origin is not cvm-openclaw/openclaw-enterprise; knowledge sync skipped")


def safe_archive_name(name: str) -> PurePosixPath:
    path = PurePosixPath(name)
    if path.is_absolute() or ".." in path.parts:
        raise SyncError(f"unsafe archive path: {name}")
    if name != STORE_ROOT and not any(
        name == allowed or name.startswith(f"{allowed}/") for allowed in ALLOWED_PATHS
    ):
        raise SyncError(f"unexpected archive path: {name}")
    if any(part in BLOCKED_NAMES for part in path.parts):
        raise SyncError(f"blocked personal-memory path: {name}")
    return path


def extract_archive(data: bytes, destination: Path) -> None:
    try:
        archive = tarfile.open(fileobj=io.BytesIO(data), mode="r:")
    except tarfile.TarError as exc:
        raise SyncError(f"invalid git archive: {exc}") from exc

    with archive:
        for member in archive.getmembers():
            relative = safe_archive_name(member.name)
            target = destination.joinpath(*relative.parts)
            if member.isdir():
                target.mkdir(parents=True, exist_ok=True)
                continue
            if not member.isfile():
                raise SyncError(f"unsupported archive entry: {member.name}")
            source = archive.extractfile(member)
            if source is None:
                raise SyncError(f"cannot read archive entry: {member.name}")
            target.parent.mkdir(parents=True, exist_ok=True)
            with source, target.open("wb") as output:
                shutil.copyfileobj(source, output)


def write_json_atomic(path: Path, payload: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_name(f".{path.name}.{os.getpid()}.tmp")
    temporary.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    os.replace(temporary, path)


def update_current_link(cache_root: Path, knowledge_root: Path) -> Path:
    current = cache_root / "current"
    temporary = cache_root / f".current.{os.getpid()}"
    temporary.unlink(missing_ok=True)
    temporary.symlink_to(os.path.relpath(knowledge_root, cache_root), target_is_directory=True)
    os.replace(temporary, current)
    return current


def export_snapshot(repo: Path, cache_root: Path, commit: str) -> Path:
    snapshots = cache_root / "snapshots"
    snapshots.mkdir(parents=True, exist_ok=True)
    snapshot = snapshots / commit
    knowledge_root = snapshot / STORE_ROOT

    if not knowledge_root.is_dir():
        archive = subprocess.run(
            ["git", "-C", str(repo), "archive", "--format=tar", commit, *ALLOWED_PATHS],
            capture_output=True,
            timeout=10,
            check=False,
        )
        if archive.returncode != 0:
            detail = archive.stderr.decode("utf-8", errors="replace").strip()
            raise SyncError(detail[-600:] or "git archive failed")

        temporary = Path(tempfile.mkdtemp(prefix=".snapshot-", dir=snapshots))
        try:
            extract_archive(archive.stdout, temporary)
            exported_root = temporary / STORE_ROOT
            if not (exported_root / "clawpro").is_dir():
                raise SyncError("archive does not contain ClawPro knowledge")
            write_json_atomic(
                exported_root / "_sync.json",
                {
                    "schema_version": "clawpro-gongfeng-knowledge-cache/v1",
                    "source_branch": SOURCE_BRANCH,
                    "source_commit": commit,
                    "synced_at": utc_now(),
                    "included_paths": list(ALLOWED_PATHS),
                    "excluded_personal_memory": sorted(BLOCKED_NAMES),
                },
            )
            os.replace(temporary, snapshot)
        except Exception:
            shutil.rmtree(temporary, ignore_errors=True)
            raise

    current = update_current_link(cache_root, knowledge_root)
    write_json_atomic(
        cache_root / "current.json",
        {
            "source_branch": SOURCE_BRANCH,
            "source_commit": commit,
            "knowledge_root": str(current),
            "synced_at": utc_now(),
        },
    )
    return current


def sync_knowledge(repo: Path, cache_root: Path) -> tuple[str, Path]:
    validate_project(repo)
    cache_root.mkdir(parents=True, exist_ok=True)
    lock_path = cache_root / "sync.lock"
    with lock_path.open("a+", encoding="utf-8") as lock:
        fcntl.flock(lock.fileno(), fcntl.LOCK_EX)
        run_git(
            repo,
            [
                "fetch",
                "--no-write-fetch-head",
                "--no-tags",
                "origin",
                f"+refs/heads/{SOURCE_BRANCH}:{SYNC_REF}",
            ],
            timeout=30,
        )
        commit = run_git(repo, ["rev-parse", "--verify", f"{SYNC_REF}^{{commit}}"])
        current = export_snapshot(repo, cache_root, commit)
        return commit, current


def prompt_text(payload: dict[str, Any]) -> str:
    for key in ("prompt", "user_prompt", "message", "input"):
        value = payload.get(key)
        if isinstance(value, str) and value.strip():
            return value.strip()
    return ""


def query_terms(prompt: str) -> list[str]:
    terms: list[str] = []
    seen: set[str] = set()

    def add(term: str) -> None:
        normalized = term.strip().lower()
        if len(normalized) < 2 or normalized in GENERIC_QUERY_TERMS or normalized in seen:
            return
        seen.add(normalized)
        terms.append(normalized)

    for token in re.findall(r"[A-Za-z][A-Za-z0-9._+-]{1,}", prompt):
        add(token)
    lowered_prompt = prompt.lower()
    for keyword in sorted(DOMAIN_KEYWORDS, key=len, reverse=True):
        if keyword in lowered_prompt:
            add(keyword)
    for segment in re.findall(r"[\u4e00-\u9fff]+", prompt):
        for chunk in QUESTION_FILLER_RE.split(segment):
            cleaned = chunk.strip("的了吧吗呢啊呀后前")
            if 2 <= len(cleaned) <= 12:
                add(cleaned)
    return terms[:32]


def is_domain_prompt(prompt: str) -> bool:
    lowered = prompt.lower()
    return any(signal in lowered for signal in DOMAIN_SIGNALS)


def markdown_metadata(content: str) -> tuple[str | None, str, set[str]]:
    knowledge_id_match = KNOWLEDGE_ID_RE.search(content)
    status_match = KNOWLEDGE_STATUS_RE.search(content)
    supersedes_match = SUPERSEDES_RE.search(content)
    supersedes = (
        set(re.findall(r"`([^`]+)`", supersedes_match.group(1)))
        if supersedes_match
        else set()
    )
    return (
        knowledge_id_match.group(1) if knowledge_id_match else None,
        status_match.group(1).lower() if status_match else "verified",
        supersedes,
    )


def knowledge_documents(knowledge_root: Path) -> list[tuple[str, str, str]]:
    candidates: list[tuple[str, str, str, str | None, str, set[str]]] = []
    for subdir in ("clawpro", "specs"):
        root = knowledge_root / subdir
        if not root.is_dir():
            continue
        for path in sorted(root.rglob("*")):
            if (
                not path.is_file()
                or path.is_symlink()
                or path.suffix.lower() not in TEXT_SUFFIXES
                or path.stat().st_size > MAX_DOCUMENT_BYTES
                or any(part in BLOCKED_NAMES for part in path.parts)
                or any(part in RECALL_EXCLUDED_PARTS for part in path.parts)
            ):
                continue
            relative = path.relative_to(knowledge_root).as_posix()
            content = path.read_text(encoding="utf-8", errors="replace")
            headings = " ".join(
                re.findall(r"^#{1,4}\s+(.+?)\s*$", content, flags=re.MULTILINE)[:12]
            )
            knowledge_id, status, supersedes = markdown_metadata(content)
            candidates.append(
                (relative, headings, content, knowledge_id, status, supersedes)
            )

    superseded_ids = {
        knowledge_id
        for _, _, _, _, status, supersedes in candidates
        if status == "verified"
        for knowledge_id in supersedes
    }
    return [
        (relative, headings, content)
        for relative, headings, content, knowledge_id, status, _ in candidates
        if status == "verified"
        and (knowledge_id is None or knowledge_id not in superseded_ids)
    ]


def recall_knowledge(
    knowledge_root: Path,
    prompt: str,
    top_k: int,
) -> tuple[list[str], list[dict[str, Any]]]:
    terms = query_terms(prompt)
    if not terms:
        return [], []

    documents = knowledge_documents(knowledge_root)
    document_frequency = {
        term: sum(
            1
            for relative, headings, content in documents
            if term in f"{relative}\n{headings}\n{content}".lower()
        )
        for term in terms
    }
    ranked: list[dict[str, Any]] = []
    document_count = max(1, len(documents))

    for relative, headings, content in documents:
        path_text = relative.lower()
        heading_text = headings.lower()
        body_text = content.lower()
        score = 0.0
        path_hits: list[str] = []
        heading_hits: list[str] = []
        content_hits: list[str] = []

        for term in terms:
            frequency = document_frequency.get(term, document_count)
            rarity = 1.0 + (document_count - frequency) / document_count
            if term in path_text:
                score += 18.0 * rarity
                path_hits.append(term)
            if term in heading_text:
                score += 9.0 * rarity
                heading_hits.append(term)
            occurrences = body_text.count(term)
            if occurrences:
                score += min(occurrences, 4) * 2.0 * rarity
                content_hits.append(term)

        if score <= 0:
            continue
        reasons: list[str] = []
        if path_hits:
            reasons.append(f"路径命中：{', '.join(path_hits[:5])}")
        if heading_hits:
            reasons.append(f"标题命中：{', '.join(heading_hits[:5])}")
        if content_hits:
            reasons.append(f"正文命中：{', '.join(content_hits[:5])}")
        ranked.append(
            {
                "path": relative,
                "absolute_path": str(knowledge_root / relative),
                "score": round(score, 2),
                "reason": "；".join(reasons),
                "matched_terms": sorted(set(path_hits + heading_hits + content_hits)),
            }
        )

    ranked.sort(key=lambda item: (-float(item["score"]), str(item["path"])))
    if not ranked:
        return terms, []
    if not is_domain_prompt(prompt) and float(ranked[0]["score"]) < 45:
        return terms, []
    return terms, ranked[:top_k]


def configured_top_k() -> int:
    raw = os.environ.get("CLAWPRO_KNOWLEDGE_TOP_K", "3")
    try:
        value = int(raw)
    except ValueError:
        value = 3
    return min(10, max(1, value))


def retrieval_audit(
    cache_root: Path,
    hook_input: dict[str, Any],
    prompt: str,
    commit: str,
    recalled: list[dict[str, Any]],
) -> Path:
    log_path = cache_root / "retrieval-log.jsonl"
    session = str(hook_input.get("session_id") or "")
    record = {
        "ts": utc_now(),
        "source_branch": SOURCE_BRANCH,
        "source_commit": commit,
        "session_sha256": hashlib.sha256(session.encode("utf-8")).hexdigest() if session else None,
        "query_sha256": hashlib.sha256(prompt.encode("utf-8")).hexdigest(),
        "matched_terms": sorted(
            {
                term
                for item in recalled
                for term in item.get("matched_terms", [])
            }
        ),
        "recalled": [
            {
                "path": item["path"],
                "score": item["score"],
                "reason": item["reason"],
            }
            for item in recalled
        ],
    }
    lock_path = cache_root / "retrieval-log.lock"
    with lock_path.open("a+", encoding="utf-8") as lock:
        fcntl.flock(lock.fileno(), fcntl.LOCK_EX)
        with log_path.open("a", encoding="utf-8") as output:
            output.write(json.dumps(record, ensure_ascii=False) + "\n")
    return log_path


def recall_context(
    knowledge_root: Path,
    commit: str,
    terms: list[str],
    recalled: list[dict[str, Any]],
    audit_log: Path,
) -> str:
    base = (
        "ClawPro 工蜂知识库已在本轮消息前同步。"
        f"来源分支：{SOURCE_BRANCH}；提交：{commit}；知识根目录：{knowledge_root}。"
        "仅同步 clawpro/ 与 specs/，已排除 MEMORY.md、memory/、USER.md、IDENTITY.md。"
        "正式召回只使用 verified 知识，已排除 _inbox、candidate 及被新版本替代的旧知识。"
    )
    if not recalled:
        return (
            f"{base}本轮没有高置信度知识命中，不强制读取文件；"
            "如果任务涉及 ClawPro/OpenClaw，请从知识根目录继续检索并明确引用来源。"
            f"召回审计：{audit_log}。"
        )

    lines = []
    for index, item in enumerate(recalled, start=1):
        lines.append(
            f"{index}. {item['absolute_path']}｜score={item['score']}｜{item['reason']}"
        )
    return (
        f"{base}本轮召回关键词：{', '.join(terms[:12])}。"
        "回答前优先逐一读取以下 Top-K 文件，并仅使用与问题相关的内容：\n"
        + "\n".join(lines)
        + f"\n召回审计：{audit_log}。"
    )


def load_hook_input() -> dict[str, Any]:
    try:
        payload = json.load(sys.stdin)
    except Exception:
        return {}
    return payload if isinstance(payload, dict) else {}


def self_test() -> int:
    valid_origins = (
        "git@git.woa.com:cvm-openclaw/openclaw-enterprise.git",
        "ssh://git@git.woa.com/cvm-openclaw/openclaw-enterprise.git",
        "https://git.woa.com/cvm-openclaw/openclaw-enterprise",
    )
    assert all(is_expected_origin(origin) for origin in valid_origins)
    assert not is_expected_origin("git@git.woa.com:other/openclaw-enterprise.git")
    assert safe_archive_name(f"{STORE_ROOT}/clawpro/product/overview.md")
    assert safe_archive_name(f"{STORE_ROOT}/specs/doc-writing.md")
    for blocked in (
        f"{STORE_ROOT}/cloud-shrimp-memory/MEMORY.md",
        f"{STORE_ROOT}/clawpro/memory/note.md",
        "../MEMORY.md",
    ):
        try:
            safe_archive_name(blocked)
        except SyncError:
            continue
        raise AssertionError(f"blocked path was accepted: {blocked}")

    with tempfile.TemporaryDirectory(prefix="clawpro-recall-self-test-") as temporary:
        root = Path(temporary)
        plugins = root / "clawpro" / "modules" / "M5-plugins.md"
        users = root / "clawpro" / "modules" / "M1-auth-and-users.md"
        inbox = root / "clawpro" / "_inbox" / "hook" / "candidate.md"
        old = root / "clawpro" / "topics" / "hook" / "old.md"
        replacement = root / "clawpro" / "topics" / "hook" / "replacement.md"
        plugins.parent.mkdir(parents=True)
        plugins.write_text("# 插件管理\n企业插件、Hook 与 npm 安装配置。\n", encoding="utf-8")
        users.write_text("# 用户认证\nOneID 用户与权限管理。\n", encoding="utf-8")
        inbox.parent.mkdir(parents=True)
        inbox.write_text(
            "# 候选 Hook\n\n- 知识 ID：`candidate-hook`\n- 状态：`candidate`\n\n"
            "## 结论\n\n候选知识不参与召回。\n",
            encoding="utf-8",
        )
        old.parent.mkdir(parents=True)
        old.write_text(
            "# 旧 Hook 结论\n\n- 知识 ID：`old-hook`\n- 状态：`verified`\n\n"
            "## 结论\n\n旧版 Hook 结论。\n",
            encoding="utf-8",
        )
        replacement.write_text(
            "# 新 Hook 结论\n\n- 知识 ID：`new-hook`\n- 状态：`verified`\n"
            "- 替代知识：`old-hook`\n\n## 结论\n\n新版 Hook 结论。\n",
            encoding="utf-8",
        )
        _, plugin_results = recall_knowledge(root, "插件 Hook 如何配置", 3)
        _, user_results = recall_knowledge(root, "OneID 用户认证", 3)
        _, hook_results = recall_knowledge(root, "新版 Hook 结论", 5)
        _, unrelated_results = recall_knowledge(root, "帮我查询明天北京天气", 3)
        assert any(
            item["path"].endswith("M5-plugins.md") for item in plugin_results
        )
        assert any(
            item["path"].endswith("M1-auth-and-users.md") for item in user_results
        )
        assert any(item["path"].endswith("replacement.md") for item in hook_results)
        assert not any("_inbox" in item["path"] for item in hook_results)
        assert not any(item["path"].endswith("old.md") for item in hook_results)
        assert not unrelated_results
    print("clawpro_knowledge_sync self-test passed")
    return 0


def main() -> int:
    parser = argparse.ArgumentParser(add_help=False)
    parser.add_argument("--self-test", action="store_true")
    args, _ = parser.parse_known_args()
    if args.self_test:
        return self_test()

    hook_input = load_hook_input()
    event = hook_input.get("hook_event_name")
    if event and event != "UserPromptSubmit":
        allow_silently()
        return 0

    configured_cache = os.environ.get("CLAWPRO_KNOWLEDGE_CACHE_ROOT")
    cache_root = (
        Path(configured_cache).expanduser().resolve()
        if configured_cache
        else DEFAULT_CACHE_ROOT
    )

    try:
        commit, knowledge_root = sync_knowledge(PROJECT_ROOT, cache_root)
        prompt = prompt_text(hook_input)
        terms, recalled = recall_knowledge(knowledge_root, prompt, configured_top_k())
        audit_log = retrieval_audit(
            cache_root,
            hook_input,
            prompt,
            commit,
            recalled,
        )
        additional_context(recall_context(knowledge_root, commit, terms, recalled, audit_log))
    except Exception as exc:
        reason = str(exc).replace("\n", " ")[:800]
        additional_context(
            "ClawPro 工蜂知识库本轮同步失败，已降级继续执行，不阻塞用户消息。"
            "不要声称已经读取最新工蜂知识；依赖最新事实时请标记待确认。"
            f"原因：{reason}"
        )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
