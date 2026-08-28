#!/usr/bin/env python3
"""Join ClawPro ownership metadata with the live Gongfeng feature branch list."""

from __future__ import annotations

import argparse
import csv
import io
import os
import subprocess
import sys
import tempfile
from pathlib import Path


OWNERSHIP_RELATIVE = Path(".codebuddy/docs/clawpro-branch-ownership.tsv")
LIVE_MAP_NAME = "clawpro-branch-map.live.tsv"
OWNERSHIP_FIELDS = ("branch", "module", "page_or_tab", "team")
LIVE_FIELDS = (
    "branch",
    "module",
    "page_or_tab",
    "team",
    "registration_status",
    "remote_status",
    "remote_head",
)


def git(repo: Path, *args: str, timeout: int = 12) -> str:
    result = subprocess.run(
        ["git", "-C", str(repo), *args],
        capture_output=True,
        text=True,
        check=False,
        timeout=timeout,
        env={**os.environ, "GIT_TERMINAL_PROMPT": "0"},
    )
    if result.returncode != 0:
        detail = result.stderr.strip() or result.stdout.strip() or "unknown git error"
        raise RuntimeError(f"git {' '.join(args)} failed: {detail}")
    return result.stdout


def repo_root() -> Path:
    result = subprocess.run(
        ["git", "rev-parse", "--show-toplevel"],
        capture_output=True,
        text=True,
        check=False,
    )
    if result.returncode != 0 or not result.stdout.strip():
        raise RuntimeError("current directory is not inside a Git repository")
    return Path(result.stdout.strip()).resolve()


def live_map_path(repo: Path) -> Path:
    value = git(repo, "rev-parse", "--git-path", LIVE_MAP_NAME).strip()
    path = Path(value)
    return path.resolve() if path.is_absolute() else (repo / path).resolve()


def read_ownership(path: Path) -> list[dict[str, str]]:
    with path.open(encoding="utf-8", newline="") as handle:
        reader = csv.DictReader(handle, delimiter="\t")
        if tuple(reader.fieldnames or ()) != OWNERSHIP_FIELDS:
            raise RuntimeError(
                f"unexpected ownership TSV header: {reader.fieldnames}; "
                f"expected {list(OWNERSHIP_FIELDS)}"
            )
        rows = [{field: (row.get(field) or "").strip() for field in OWNERSHIP_FIELDS} for row in reader]

    seen: set[str] = set()
    for row in rows:
        branch = row["branch"]
        if not branch.startswith("feature/"):
            raise RuntimeError(f"ownership branch must start with feature/: {branch}")
        if branch in seen:
            raise RuntimeError(f"duplicate ownership branch: {branch}")
        seen.add(branch)
    return rows


def read_remote_feature_heads(repo: Path, origin: str) -> dict[str, str]:
    output = git(repo, "ls-remote", "--heads", origin)
    heads: dict[str, str] = {}
    prefix = "refs/heads/"
    for line in output.splitlines():
        fields = line.split()
        if len(fields) != 2 or not fields[1].startswith(prefix):
            continue
        branch = fields[1][len(prefix) :]
        if branch.startswith("feature/"):
            heads[branch] = fields[0].lower()
    return heads


def build_live_rows(
    ownership: list[dict[str, str]],
    remote_heads: dict[str, str],
) -> list[dict[str, str]]:
    rows: list[dict[str, str]] = []
    registered = {row["branch"] for row in ownership}

    for row in ownership:
        branch = row["branch"]
        head = remote_heads.get(branch, "")
        rows.append(
            {
                **row,
                "registration_status": "registered",
                "remote_status": "active" if head else "missing",
                "remote_head": head,
            }
        )

    for branch in sorted(set(remote_heads) - registered):
        rows.append(
            {
                "branch": branch,
                "module": "未登记",
                "page_or_tab": "待确认",
                "team": "待确认",
                "registration_status": "unregistered",
                "remote_status": "active",
                "remote_head": remote_heads[branch],
            }
        )
    return rows


def render_tsv(rows: list[dict[str, str]]) -> str:
    output = io.StringIO(newline="")
    writer = csv.DictWriter(output, fieldnames=LIVE_FIELDS, delimiter="\t", lineterminator="\n")
    writer.writeheader()
    writer.writerows(rows)
    return output.getvalue()


def write_if_changed(path: Path, content: str) -> bool:
    try:
        if path.read_text(encoding="utf-8") == content:
            return False
    except OSError:
        pass

    path.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.NamedTemporaryFile(
        mode="w",
        encoding="utf-8",
        dir=path.parent,
        prefix=f".{path.name}.",
        delete=False,
    ) as handle:
        handle.write(content)
        temporary = Path(handle.name)
    temporary.chmod(0o600)
    os.replace(temporary, path)
    return True


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--origin", default="origin", help="Git remote name (default: origin)")
    parser.add_argument("--check", action="store_true", help="report drift without writing the live TSV")
    parser.add_argument("--stdout", action="store_true", help="print the generated TSV instead of its summary")
    parser.add_argument("--self-test", action="store_true", help="run an offline transformation test")
    return parser.parse_args()


def self_test() -> int:
    ownership = [
        {"branch": "feature/known", "module": "管控端", "page_or_tab": "已知页面", "team": "计算"},
        {"branch": "feature/removed", "module": "用户端", "page_or_tab": "旧页面", "team": "计算"},
    ]
    rows = build_live_rows(
        ownership,
        {
            "feature/known": "a" * 40,
            "feature/new": "b" * 40,
        },
    )
    by_branch = {row["branch"]: row for row in rows}
    passed = (
        by_branch["feature/known"]["remote_status"] == "active"
        and by_branch["feature/removed"]["remote_status"] == "missing"
        and by_branch["feature/new"]["registration_status"] == "unregistered"
        and by_branch["feature/new"]["remote_head"] == "b" * 40
        and render_tsv(rows).splitlines()[0] == "\t".join(LIVE_FIELDS)
    )
    print("PASS" if passed else "FAIL")
    return 0 if passed else 1


def main() -> int:
    args = parse_args()
    if args.self_test:
        return self_test()
    try:
        repo = repo_root()
        ownership = read_ownership(repo / OWNERSHIP_RELATIVE)
        remote_heads = read_remote_feature_heads(repo, args.origin)
        rows = build_live_rows(ownership, remote_heads)
        content = render_tsv(rows)
        target = live_map_path(repo)

        current = ""
        try:
            current = target.read_text(encoding="utf-8")
        except OSError:
            pass
        changed = current != content

        if args.stdout:
            sys.stdout.write(content)
        else:
            if not args.check:
                changed = write_if_changed(target, content)
            active = sum(row["registration_status"] == "registered" and row["remote_status"] == "active" for row in rows)
            missing = sum(row["remote_status"] == "missing" for row in rows)
            unregistered = sum(row["registration_status"] == "unregistered" for row in rows)
            print(f"LIVE_TSV={target}")
            print(
                f"registered_active={active} registered_missing={missing} "
                f"remote_unregistered={unregistered} changed={'yes' if changed else 'no'}"
            )
        return 1 if args.check and changed else 0
    except (OSError, RuntimeError, subprocess.TimeoutExpired) as error:
        print(f"branch sync failed: {error}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
