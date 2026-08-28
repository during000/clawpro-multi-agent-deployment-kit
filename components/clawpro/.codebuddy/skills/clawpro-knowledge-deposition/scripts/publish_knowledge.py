#!/usr/bin/env python3
"""Publish one structured ClawPro knowledge entry to Gongfeng.

V2 stores time and domain in metadata, and routes verified files by topic:

- verified -> ``clawpro/topics/<topic>/<slug>--<short-id>.md``

The publisher works in a temporary Git repository, so the caller's feature
branch, index, FETCH_HEAD, and worktree remain untouched.  Content that does not
meet the verified-knowledge gate must be skipped before this script is called.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import shutil
import subprocess
import tempfile
from datetime import date, datetime, timezone
from pathlib import Path
from typing import Any


TARGET_BRANCH = "harness-knowledge-store"
STORE_ROOT = Path("harness-knowledge-store/clawpro")
VERIFIED_ROOT = STORE_ROOT / "topics"
MAX_RETRIES = 3
VALID_TYPES = {
    "fact",
    "decision",
    "constraint",
    "runbook",
    "root-cause",
    "implementation",
}
VALID_CONFIDENCE = {"medium", "high"}
ORIGIN_PATTERN = re.compile(
    r"^(?:(?:ssh|https?)://(?:git@)?|git@)git\.woa\.com[:/]"
    r"cvm-openclaw/openclaw-enterprise(?:\.git)?/?$",
    re.IGNORECASE,
)
CONTROL_RE = re.compile(r"[\x00-\x08\x0b\x0c\x0e-\x1f\x7f]")
TAG_RE = re.compile(r"^[A-Za-z0-9._+\-\u4e00-\u9fff]{1,32}$")
SLUG_RE = re.compile(r"^[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$")
KNOWLEDGE_REF_RE = re.compile(r"^[A-Za-z0-9._-]{4,64}$")
KNOWLEDGE_ID_RE = re.compile(r"^- 知识 ID：`([^`]+)`\s*$", re.MULTILINE)
SECRET_PATTERNS = (
    re.compile(r"-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----", re.IGNORECASE),
    re.compile(r"\bBearer\s+[A-Za-z0-9._~+/=-]{12,}", re.IGNORECASE),
    re.compile(r"\bAKID[A-Za-z0-9]{12,}\b"),
    re.compile(r"\bhk-[A-Za-z0-9]{16,}\b", re.IGNORECASE),
    re.compile(
        r"(?i)\b(?:api[_-]?key|access[_-]?token|refresh[_-]?token|"
        r"password|passwd|secret|authorization)\b\s*[:=]\s*"
        r"[\"']?[^\s\"']{8,}"
    ),
)


class PublishError(RuntimeError):
    """A safe, actionable publishing failure."""


def emit(payload: dict[str, Any]) -> None:
    print(json.dumps(payload, ensure_ascii=False))


def run(
    args: list[str],
    *,
    cwd: Path | None = None,
    timeout: int = 60,
    check: bool = True,
) -> subprocess.CompletedProcess[str]:
    try:
        completed = subprocess.run(
            args,
            cwd=cwd,
            text=True,
            capture_output=True,
            timeout=timeout,
            check=False,
        )
    except (OSError, subprocess.TimeoutExpired) as exc:
        raise PublishError(f"{args[0]} command failed: {exc}") from exc
    if check and completed.returncode != 0:
        detail = (completed.stderr or completed.stdout).strip()
        raise PublishError(detail[-800:] or f"{args[0]} exited {completed.returncode}")
    return completed


def git(repo: Path, args: list[str], *, timeout: int = 60, check: bool = True) -> str:
    return run(
        ["git", "-C", str(repo), *args],
        timeout=timeout,
        check=check,
    ).stdout.strip()


def project_root() -> Path:
    configured = os.environ.get("CODEBUDDY_PROJECT_DIR")
    candidate = (
        Path(configured).expanduser()
        if configured
        else Path(__file__).resolve().parents[4]
    )
    return candidate.resolve()


def validate_project(repo: Path) -> str:
    root = git(repo, ["rev-parse", "--show-toplevel"])
    if Path(root).resolve() != repo:
        raise PublishError("CODEBUDDY_PROJECT_DIR is not the repository root")
    origin = git(repo, ["remote", "get-url", "origin"])
    normalized = origin.strip().replace("\\", "/")
    if not ORIGIN_PATTERN.search(normalized):
        raise PublishError("origin is not cvm-openclaw/openclaw-enterprise")
    return origin


def clean_text(name: str, value: str, limit: int) -> str:
    text = value.strip()
    if not text:
        raise PublishError(f"{name} cannot be empty")
    if len(text) > limit:
        raise PublishError(f"{name} exceeds {limit} characters")
    if CONTROL_RE.search(text):
        raise PublishError(f"{name} contains control characters")
    return text


def clean_optional_list(
    name: str,
    values: list[str],
    *,
    max_items: int,
    item_limit: int,
) -> list[str]:
    cleaned: list[str] = []
    seen: set[str] = set()
    for raw in values:
        value = clean_text(name, raw, item_limit)
        if "\n" in value or "`" in value:
            raise PublishError(f"{name} values must be single-line plain text")
        normalized = value.lower()
        if normalized in seen:
            continue
        seen.add(normalized)
        cleaned.append(value)
    if len(cleaned) > max_items:
        raise PublishError(f"{name} accepts at most {max_items} values")
    return cleaned


def validate_no_secrets(values: list[str]) -> None:
    combined = "\n".join(values)
    for pattern in SECRET_PATTERNS:
        if pattern.search(combined):
            raise PublishError("knowledge entry contains a credential-like secret")


def validate_slug(name: str, value: str) -> str:
    slug = value.strip().lower()
    if not SLUG_RE.fullmatch(slug):
        raise PublishError(
            f"{name} must be a 3-50 character lowercase kebab-case slug"
        )
    return slug


def validate_choice(name: str, value: str, choices: set[str]) -> str:
    normalized = value.strip().lower()
    if normalized not in choices:
        raise PublishError(f"{name} must be one of: {', '.join(sorted(choices))}")
    return normalized


def validate_date(value: str) -> str:
    try:
        return date.fromisoformat(value).isoformat()
    except ValueError as exc:
        raise PublishError("valid-from must use YYYY-MM-DD") from exc


def normalize_for_hash(value: str) -> str:
    return re.sub(r"\s+", " ", value).strip().lower()


def content_digest(
    *,
    knowledge_type: str,
    domain: str,
    topic: str,
    title: str,
    summary: str,
    scope: str,
    evidence: str,
    boundary: str,
) -> str:
    canonical = "\n".join(
        normalize_for_hash(value)
        for value in (
            knowledge_type,
            domain,
            topic,
            title,
            summary,
            scope,
            evidence,
            boundary,
        )
    )
    return hashlib.sha256(canonical.encode("utf-8")).hexdigest()


def sanitize_tags(raw_tags: list[str]) -> list[str]:
    tags: list[str] = []
    seen: set[str] = set()
    for raw in raw_tags:
        tag = raw.strip()
        lowered = tag.lower()
        if not tag or lowered in seen:
            continue
        if not TAG_RE.fullmatch(tag):
            raise PublishError(f"invalid tag: {tag}")
        seen.add(lowered)
        tags.append(tag)
    if not 1 <= len(tags) <= 6:
        raise PublishError("provide between 1 and 6 tags")
    return tags


def markdown_value(value: str) -> str:
    return value.replace("\r\n", "\n").replace("\r", "\n")


def markdown_list(values: list[str]) -> str:
    return "、".join(f"`{value}`" for value in values) if values else "无"


def entry_path(
    *,
    topic: str,
    slug: str,
    knowledge_id: str,
) -> Path:
    return VERIFIED_ROOT / topic / f"{slug}--{knowledge_id[:8]}.md"


def render_entry(
    *,
    knowledge_id: str,
    title: str,
    summary: str,
    scope: str,
    evidence: str,
    boundary: str,
    knowledge_type: str,
    domain: str,
    topic: str,
    confidence: str,
    tags: list[str],
    aliases: list[str],
    entities: list[str],
    source_refs: list[str],
    supersedes: list[str],
    session_id: str,
    source_commit: str,
    valid_from: str,
    created_at: datetime,
) -> str:
    session_hash = hashlib.sha256(session_id.encode("utf-8")).hexdigest()[:16]
    return (
        f"# {markdown_value(title)}\n\n"
        f"- 知识 ID：`{knowledge_id}`\n"
        "- 状态：`verified`\n"
        f"- 类型：`{knowledge_type}`\n"
        f"- 领域：`{domain}`\n"
        f"- 主题：`{topic}`\n"
        f"- 置信度：`{confidence}`\n"
        f"- 创建时间：`{created_at.isoformat()}`\n"
        f"- 更新时间：`{created_at.isoformat()}`\n"
        f"- 生效时间：`{valid_from}`\n"
        f"- 来源提交：`{source_commit}`\n"
        f"- 会话标识：`sha256:{session_hash}`\n"
        f"- 标签：{markdown_list(tags)}\n"
        f"- 别名：{markdown_list(aliases)}\n"
        f"- 实体：{markdown_list(entities)}\n"
        f"- 来源引用：{markdown_list(source_refs)}\n"
        f"- 替代知识：{markdown_list(supersedes)}\n\n"
        "## 结论\n\n"
        f"{markdown_value(summary)}\n\n"
        "## 适用范围\n\n"
        f"{markdown_value(scope)}\n\n"
        "## 核验依据\n\n"
        f"{markdown_value(evidence)}\n\n"
        "## 边界与待确认项\n\n"
        f"{markdown_value(boundary)}\n"
    )


def identity(repo: Path) -> tuple[str, str]:
    name = git(repo, ["config", "--get", "user.name"], check=False).strip()
    email = git(repo, ["config", "--get", "user.email"], check=False).strip()
    return (
        name or "ClawPro Knowledge Bot",
        email or "clawpro-knowledge-bot@tencent.com",
    )


def is_non_fast_forward(detail: str) -> bool:
    lowered = detail.lower()
    return (
        "non-fast-forward" in lowered
        or "fetch first" in lowered
        or "[rejected]" in lowered
    )


def find_entry_by_id(repo: Path, knowledge_id: str) -> Path | None:
    root = repo / STORE_ROOT
    if not root.is_dir():
        return None
    for path in sorted(root.rglob("*.md")):
        try:
            content = path.read_text(encoding="utf-8", errors="replace")
        except OSError:
            continue
        match = KNOWLEDGE_ID_RE.search(content)
        if match and match.group(1) == knowledge_id:
            return path
    return None


def publish_once(
    *,
    origin: str,
    relative_path: Path,
    content: str,
    title: str,
    knowledge_id: str,
    author_name: str,
    author_email: str,
) -> tuple[str, str]:
    temporary = Path(tempfile.mkdtemp(prefix="clawpro-knowledge-publish-"))
    try:
        git(temporary, ["init", "--quiet"])
        git(temporary, ["remote", "add", "origin", origin])
        git(
            temporary,
            [
                "fetch",
                "--quiet",
                "--no-tags",
                "--depth=1",
                "origin",
                f"refs/heads/{TARGET_BRANCH}",
            ],
            timeout=90,
        )
        git(temporary, ["checkout", "--quiet", "--detach", "FETCH_HEAD"])

        target = temporary / relative_path
        existing = find_entry_by_id(temporary, knowledge_id)
        if existing:
            return "duplicate", git(temporary, ["rev-parse", "HEAD"])

        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(content, encoding="utf-8")
        git(temporary, ["add", "--", relative_path.as_posix()])
        git(
            temporary,
            [
                "-c",
                f"user.name={author_name}",
                "-c",
                f"user.email={author_email}",
                "commit",
                "--quiet",
                "-m",
                f"knowledge: deposit {title[:50]}",
            ],
        )
        commit = git(temporary, ["rev-parse", "HEAD"])
        pushed = run(
            [
                "git",
                "-C",
                str(temporary),
                "push",
                "--porcelain",
                "origin",
                f"HEAD:refs/heads/{TARGET_BRANCH}",
            ],
            timeout=90,
            check=False,
        )
        if pushed.returncode != 0:
            detail = (pushed.stderr or pushed.stdout).strip()
            if is_non_fast_forward(detail):
                raise PublishError(f"retryable non-fast-forward: {detail[-500:]}")
            raise PublishError(detail[-800:] or "git push failed")
        return "published", commit
    finally:
        shutil.rmtree(temporary, ignore_errors=True)


def publish_with_retry(
    *,
    origin: str,
    relative_path: Path,
    content: str,
    title: str,
    knowledge_id: str,
    author_name: str,
    author_email: str,
) -> tuple[str, str, int]:
    last_error = ""
    for attempt in range(1, MAX_RETRIES + 1):
        try:
            result, commit = publish_once(
                origin=origin,
                relative_path=relative_path,
                content=content,
                title=title,
                knowledge_id=knowledge_id,
                author_name=author_name,
                author_email=author_email,
            )
            return result, commit, attempt
        except PublishError as exc:
            last_error = str(exc)
            if "retryable non-fast-forward" not in last_error:
                raise
    raise PublishError(
        f"knowledge branch changed concurrently after {MAX_RETRIES} attempts: "
        f"{last_error[-300:]}"
    )


def self_test() -> int:
    assert ORIGIN_PATTERN.search(
        "git@git.woa.com:cvm-openclaw/openclaw-enterprise.git"
    )
    assert not ORIGIN_PATTERN.search("git@git.woa.com:other/repository.git")
    assert validate_slug("domain", "local-agent") == "local-agent"
    digest = content_digest(
        knowledge_type="implementation",
        domain="local-agent",
        topic="knowledge-hook",
        title="Stop Hook 知识沉淀",
        summary="每轮结束触发一次知识判定。",
        scope="ClawPro CodeBuddy Hook",
        evidence=".codebuddy/settings.json",
        boundary="无",
    )
    assert len(digest) == 64
    assert sanitize_tags(["Hook", "知识库", "Hook"]) == ["Hook", "知识库"]
    assert entry_path(
        topic="knowledge-hook",
        slug="verified-only",
        knowledge_id=digest[:16],
    ) == VERIFIED_ROOT / "knowledge-hook" / f"verified-only--{digest[:8]}.md"
    try:
        synthetic_secret = "API_" + "TOKEN=" + "hk-" + ("1" * 30)
        validate_no_secrets([synthetic_secret])
    except PublishError:
        pass
    else:
        raise AssertionError("credential-like value was not rejected")

    created_at = datetime(2026, 1, 1, tzinfo=timezone.utc)
    verified_content = render_entry(
        knowledge_id=digest[:16],
        title="Stop Hook 知识沉淀",
        summary="每轮结束触发一次知识判定。",
        scope="ClawPro CodeBuddy Hook",
        evidence=".codebuddy/settings.json",
        boundary="无",
        knowledge_type="implementation",
        domain="local-agent",
        topic="knowledge-hook",
        confidence="high",
        tags=["Hook", "知识库"],
        aliases=["知识沉淀 Hook"],
        entities=["TeamAI"],
        source_refs=["code:.codebuddy/settings.json"],
        supersedes=[],
        session_id="session-1",
        source_commit="a" * 40,
        valid_from="2026-01-01",
        created_at=created_at,
    )
    assert "session-1" not in verified_content

    with tempfile.TemporaryDirectory(prefix="clawpro-knowledge-self-test-") as root:
        test_root = Path(root)
        remote = test_root / "remote.git"
        seed = test_root / "seed"
        run(["git", "init", "--quiet", "--bare", str(remote)])
        run(["git", "init", "--quiet", str(seed)])
        git(seed, ["config", "user.name", "Self Test"])
        git(seed, ["config", "user.email", "self-test@example.com"])
        readme = seed / "harness-knowledge-store" / "README.md"
        readme.parent.mkdir(parents=True)
        readme.write_text("# Knowledge store\n", encoding="utf-8")
        git(seed, ["add", "."])
        git(seed, ["commit", "--quiet", "-m", "seed knowledge branch"])
        git(seed, ["branch", "-M", TARGET_BRANCH])
        git(seed, ["remote", "add", "origin", str(remote)])
        git(seed, ["push", "--quiet", "-u", "origin", TARGET_BRANCH])

        verified_path = entry_path(
            topic="knowledge-hook",
            slug="verified-only",
            knowledge_id=digest[:16],
        )
        result, _ = publish_once(
            origin=str(remote),
            relative_path=verified_path,
            content=verified_content,
            title="Stop Hook 知识沉淀",
            knowledge_id=digest[:16],
            author_name="Self Test",
            author_email="self-test@example.com",
        )
        assert result == "published"
        duplicate, _ = publish_once(
            origin=str(remote),
            relative_path=verified_path,
            content=verified_content,
            title="Stop Hook 知识沉淀",
            knowledge_id=digest[:16],
            author_name="Self Test",
            author_email="self-test@example.com",
        )
        assert duplicate == "duplicate"

        second_digest = content_digest(
            knowledge_type="implementation",
            domain="knowledge-governance",
            topic="knowledge-recall",
            title="用户提问前召回 Top-K 企业知识",
            summary="UserPromptSubmit 同步知识分支并召回最相关的正式知识。",
            scope="ClawPro CodeBuddy 知识召回 Hook",
            evidence=".codebuddy/hooks/clawpro_knowledge_sync.py self-test",
            boundary="同步失败时降级放行。",
        )
        second_content = render_entry(
            knowledge_id=second_digest[:16],
            title="用户提问前召回 Top-K 企业知识",
            summary="UserPromptSubmit 同步知识分支并召回最相关的正式知识。",
            scope="ClawPro CodeBuddy 知识召回 Hook",
            evidence=".codebuddy/hooks/clawpro_knowledge_sync.py self-test",
            boundary="同步失败时降级放行。",
            knowledge_type="implementation",
            domain="knowledge-governance",
            topic="knowledge-recall",
            confidence="high",
            tags=["Hook", "知识召回"],
            aliases=["知识同步 Hook"],
            entities=["ClawPro", "UserPromptSubmit"],
            source_refs=["code:.codebuddy/hooks/clawpro_knowledge_sync.py"],
            supersedes=[],
            session_id="session-2",
            source_commit="b" * 40,
            valid_from="2026-01-01",
            created_at=created_at,
        )
        second_path = entry_path(
            topic="knowledge-recall",
            slug="top-k-before-prompt",
            knowledge_id=second_digest[:16],
        )
        second_result, _ = publish_once(
            origin=str(remote),
            relative_path=second_path,
            content=second_content,
            title="用户提问前召回 Top-K 企业知识",
            knowledge_id=second_digest[:16],
            author_name="Self Test",
            author_email="self-test@example.com",
        )
        assert second_result == "published"

        inspection = test_root / "inspection"
        run(
            [
                "git",
                "clone",
                "--quiet",
                "--branch",
                TARGET_BRANCH,
                str(remote),
                str(inspection),
            ]
        )
        mocked_entries = sorted((inspection / VERIFIED_ROOT).rglob("*.md"))
        assert len(mocked_entries) == 2
        assert inspection / verified_path in mocked_entries
        assert inspection / second_path in mocked_entries
    print("publish_knowledge self-test passed (2 mocked deposits)")
    return 0


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--title")
    parser.add_argument("--summary")
    parser.add_argument("--scope")
    parser.add_argument("--evidence")
    parser.add_argument("--boundary")
    parser.add_argument("--knowledge-type")
    parser.add_argument("--domain")
    parser.add_argument("--topic")
    parser.add_argument("--slug")
    parser.add_argument("--confidence", default="medium")
    parser.add_argument("--valid-from")
    parser.add_argument("--tag", action="append", default=[])
    parser.add_argument("--alias", action="append", default=[])
    parser.add_argument("--entity", action="append", default=[])
    parser.add_argument("--source-ref", action="append", default=[])
    parser.add_argument("--supersedes", action="append", default=[])
    parser.add_argument("--session-id")
    parser.add_argument("--dry-run", action="store_true")
    parser.add_argument("--self-test", action="store_true")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    if args.self_test:
        return self_test()

    try:
        title = clean_text("title", str(args.title or ""), 120)
        if "\n" in title:
            raise PublishError("title must be a single line")
        summary = clean_text("summary", str(args.summary or ""), 4000)
        scope = clean_text("scope", str(args.scope or ""), 1600)
        evidence = clean_text("evidence", str(args.evidence or ""), 3000)
        boundary = clean_text("boundary", str(args.boundary or ""), 2000)
        session_id = clean_text("session-id", str(args.session_id or ""), 256)
        knowledge_type = validate_choice(
            "knowledge-type",
            str(args.knowledge_type or ""),
            VALID_TYPES,
        )
        domain = validate_slug("domain", str(args.domain or ""))
        topic = validate_slug("topic", str(args.topic or ""))
        slug = validate_slug("slug", str(args.slug or args.topic or ""))
        confidence = validate_choice(
            "confidence",
            str(args.confidence),
            VALID_CONFIDENCE,
        )
        valid_from = validate_date(str(args.valid_from or date.today().isoformat()))
        tags = sanitize_tags(list(args.tag))
        aliases = clean_optional_list(
            "alias", list(args.alias), max_items=8, item_limit=80
        )
        entities = clean_optional_list(
            "entity", list(args.entity), max_items=12, item_limit=80
        )
        source_refs = clean_optional_list(
            "source-ref", list(args.source_ref), max_items=12, item_limit=240
        )
        if not source_refs:
            raise PublishError("provide at least one source-ref")
        supersedes = clean_optional_list(
            "supersedes", list(args.supersedes), max_items=8, item_limit=64
        )
        if any(not KNOWLEDGE_REF_RE.fullmatch(value) for value in supersedes):
            raise PublishError("supersedes values must be knowledge IDs")
        validate_no_secrets(
            [
                title,
                summary,
                scope,
                evidence,
                boundary,
                *tags,
                *aliases,
                *entities,
                *source_refs,
                *supersedes,
            ]
        )

        repo = project_root()
        origin = validate_project(repo)
        source_commit = git(repo, ["rev-parse", "HEAD"])
        digest = content_digest(
            knowledge_type=knowledge_type,
            domain=domain,
            topic=topic,
            title=title,
            summary=summary,
            scope=scope,
            evidence=evidence,
            boundary=boundary,
        )
        knowledge_id = digest[:16]
        now = datetime.now(timezone.utc)
        relative_path = entry_path(
            topic=topic,
            slug=slug,
            knowledge_id=knowledge_id,
        )
        content = render_entry(
            knowledge_id=knowledge_id,
            title=title,
            summary=summary,
            scope=scope,
            evidence=evidence,
            boundary=boundary,
            knowledge_type=knowledge_type,
            domain=domain,
            topic=topic,
            confidence=confidence,
            tags=tags,
            aliases=aliases,
            entities=entities,
            source_refs=source_refs,
            supersedes=supersedes,
            session_id=session_id,
            source_commit=source_commit,
            valid_from=valid_from,
            created_at=now,
        )

        if args.dry_run:
            emit(
                {
                    "status": "dry_run",
                    "knowledge_status": "verified",
                    "branch": TARGET_BRANCH,
                    "path": relative_path.as_posix(),
                    "knowledge_id": knowledge_id,
                }
            )
            return 0

        author_name, author_email = identity(repo)
        result, commit, attempts = publish_with_retry(
            origin=origin,
            relative_path=relative_path,
            content=content,
            title=title,
            knowledge_id=knowledge_id,
            author_name=author_name,
            author_email=author_email,
        )
        emit(
            {
                "status": result,
                "knowledge_status": "verified",
                "branch": TARGET_BRANCH,
                "path": relative_path.as_posix(),
                "knowledge_id": knowledge_id,
                "commit": commit,
                "attempts": attempts,
            }
        )
        return 0
    except Exception as exc:
        message = str(exc).replace("\n", " ")[:800]
        emit({"status": "error", "error": message})
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
