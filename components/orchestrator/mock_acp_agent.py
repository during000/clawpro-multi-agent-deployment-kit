#!/usr/bin/env python3
"""A tiny ACP stdio Agent used by the local Edge Runtime demo."""

import json
import sys
import time
import uuid


def send(payload):
    sys.stdout.write(json.dumps(payload, ensure_ascii=False) + "\n")
    sys.stdout.flush()


def update(session_id, payload):
    send(
        {
            "jsonrpc": "2.0",
            "method": "session/update",
            "params": {"sessionId": session_id, "update": payload},
        }
    )


def main():
    session_id = "ses_" + uuid.uuid4().hex[:10]
    selected_model = "mock-fast"

    for raw_line in sys.stdin:
        try:
            request = json.loads(raw_line)
        except json.JSONDecodeError:
            continue

        method = request.get("method")
        request_id = request.get("id")
        params = request.get("params") or {}

        if method == "initialize":
            send(
                {
                    "jsonrpc": "2.0",
                    "id": request_id,
                    "result": {
                        "protocolVersion": 1,
                        "agentCapabilities": {
                            "loadSession": True,
                            "mcpCapabilities": {"http": True, "sse": True},
                        },
                        "agentInfo": {
                            "name": "clawpro-mock-acp",
                            "version": "0.1.0",
                        },
                    },
                }
            )
        elif method == "session/new":
            send(
                {
                    "jsonrpc": "2.0",
                    "id": request_id,
                    "result": {
                        "sessionId": session_id,
                        "configOptions": [
                            {
                                "id": "model",
                                "name": "Model",
                                "category": "model",
                                "type": "select",
                                "currentValue": selected_model,
                                "options": [
                                    {"value": "mock-fast", "name": "Mock Fast"},
                                    {"value": "mock-deep", "name": "Mock Deep"},
                                ],
                            }
                        ],
                    },
                }
            )
        elif method == "session/resume":
            session_id = params.get("sessionId") or session_id
            send(
                {
                    "jsonrpc": "2.0",
                    "id": request_id,
                    "result": {"sessionId": session_id},
                }
            )
        elif method == "session/set_config_option":
            selected_model = params.get("value") or selected_model
            send(
                {
                    "jsonrpc": "2.0",
                    "id": request_id,
                    "result": {
                        "configOptions": [
                            {
                                "id": "model",
                                "name": "Model",
                                "category": "model",
                                "type": "select",
                                "currentValue": selected_model,
                                "options": [
                                    {"value": "mock-fast", "name": "Mock Fast"},
                                    {"value": "mock-deep", "name": "Mock Deep"},
                                ],
                            }
                        ]
                    },
                }
            )
        elif method == "session/prompt":
            prompt_parts = params.get("prompt") or []
            prompt = " ".join(
                part.get("text", "")
                for part in prompt_parts
                if part.get("type") == "text"
            ).strip()

            update(
                session_id,
                {
                    "sessionUpdate": "agent_message_chunk",
                    "content": {
                        "type": "text",
                        "text": "已收到任务，正在检查项目上下文。",
                    },
                },
            )
            time.sleep(0.45)
            update(
                session_id,
                {
                    "sessionUpdate": "tool_call",
                    "toolCallId": "tool_workspace_scan",
                    "title": "Workspace Scan",
                    "rawInput": {
                        "command": "scan_project_context",
                        "prompt": prompt,
                    },
                },
            )
            time.sleep(0.65)
            update(
                session_id,
                {
                    "sessionUpdate": "tool_call_update",
                    "toolCallId": "tool_workspace_scan",
                    "status": "completed",
                    "rawOutput": "项目上下文检查完成；未发现阻塞项。",
                },
            )
            time.sleep(0.45)
            update(
                session_id,
                {
                    "sessionUpdate": "agent_message_chunk",
                    "content": {
                        "type": "text",
                        "text": "任务已完成，并生成可供下游节点使用的交接结果。",
                    },
                },
            )
            update(
                session_id,
                {
                    "sessionUpdate": "usage_update",
                    "usage": {"inputTokens": 86, "outputTokens": 42},
                },
            )
            send(
                {
                    "jsonrpc": "2.0",
                    "id": request_id,
                    "result": {"stopReason": "end_turn"},
                }
            )
        elif method == "session/cancel":
            continue
        elif request_id is not None:
            send(
                {
                    "jsonrpc": "2.0",
                    "id": request_id,
                    "error": {"code": -32601, "message": "method not found"},
                }
            )


if __name__ == "__main__":
    main()
