"""
SSE (Server-Sent Events) 解析 + 飞书二维码 payload 验证辅助函数

包含:
- parse_sse: 解析 SSE 流
- assert_qrcode_payload: 断言 qrcode payload 格式正确
- verify_feishu_qrcode_payload: 完整验证飞书 auto-channel 二维码流程
"""

import json
from typing import Dict, Iterable, Iterator, Optional

import helpers


def parse_sse(lines: Iterable[str]) -> Iterator[Dict[str, str]]:
    """解析 SSE 文本流，逐条 yield {event, data}"""
    event: Optional[str] = None
    data_lines = []

    for raw_line in lines:
        line = raw_line.rstrip("\n")
        if line.endswith("\r"):
            line = line[:-1]

        if line == "":
            if event or data_lines:
                yield {
                    "event": event or "message",
                    "data": "\n".join(data_lines),
                }
            event = None
            data_lines = []
            continue

        if line.startswith(":"):
            continue
        if line.startswith("event:"):
            event = line[len("event:"):].strip()
        elif line.startswith("data:"):
            data_lines.append(line[len("data:"):].strip())


def assert_qrcode_payload(payload: dict) -> None:
    """断言 qrcode SSE payload 已归一化为 action=show_qrcode, mode=url, content=裸URL"""
    assert payload.get("action") == "show_qrcode", f"unexpected action: {payload}"
    assert payload.get("mode") == "url", f"expected mode=url, got {payload.get('mode')}: {payload}"

    content = payload.get("content")
    assert isinstance(content, str) and content, f"content should be a non-empty string: {payload}"
    assert content.startswith(("https://", "http://")), f"content should be a bare URL: {content!r}"
    assert "verification_uri" not in content, f"content should be unwrapped URL, got raw JSON: {content!r}"

    print(f"    qrcode payload 归一化正确 ✓ mode=url content={content}")


def verify_feishu_qrcode_payload(user_token: str, instance_db_id: int) -> None:
    """完整验证飞书 auto-channel 二维码 SSE 流程"""
    print(">>> 步骤 1：调用 auto-channel（飞书）...")
    resp = helpers.user_auto_channel(user_token, instance_db_id, "feishu", timeout=120)
    try:
        assert resp.status_code == 200, f"auto-channel 应返回 200，实际: {resp.status_code} {resp.text}"
        content_type = resp.headers.get("Content-Type", "")
        assert "text/event-stream" in content_type, (
            f"飞书 auto-channel 应返回 SSE 流，实际: {content_type}"
        )
        print("    SSE 流建立 ✓")

        for msg in parse_sse(resp.iter_lines(decode_unicode=True)):
            event = msg["event"]
            data = msg["data"]
            if not data:
                continue

            if event == "qrcode":
                assert_qrcode_payload(json.loads(data))
                return

            if event == "fail":
                raise AssertionError(f"SSE fail event: {data}")

            if event == "finish":
                raise AssertionError(f"finish arrived before qrcode: {data}")

        raise AssertionError("未收到 qrcode SSE 事件")
    finally:
        resp.close()
