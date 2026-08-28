#!/usr/bin/env python3
"""
Convert docs/API.md to OpenAPI 3.0.3 YAML spec.

Usage:
    python3 scripts/api_md_to_openapi.py
    # Output: docs/openapi.yaml

The script parses the structured markdown in API.md, extracting:
- Endpoint definitions (method + path)
- Parameter tables (name, type, required, description)
- Content-Type declarations
- Permission levels
- Response examples
"""

import json
import os
import re
import sys
from collections import OrderedDict


# ─────────────────────────────────────────────────────────────────────────────
# Output as JSON (valid OpenAPI format, avoids YAML escaping issues)
# ─────────────────────────────────────────────────────────────────────────────

def spec_dump(spec):
    """Serialize spec as JSON (OpenAPI supports both JSON and YAML)."""
    return json.dumps(spec, indent=2, ensure_ascii=False) + "\n"


# ─────────────────────────────────────────────────────────────────────────────
# Type mapping
# ─────────────────────────────────────────────────────────────────────────────

def map_type(type_str):
    """Map API.md type annotation to OpenAPI schema."""
    t = type_str.strip().lower()
    if t in ("string", "str"):
        return {"type": "string"}
    if t in ("int", "uint", "integer", "number", "int64", "uint64"):
        return {"type": "integer"}
    if t in ("float", "float64", "double"):
        return {"type": "number"}
    if t in ("bool", "boolean"):
        return {"type": "boolean"}
    if t in ("string[]", "[]string", "array<string>"):
        return {"type": "array", "items": {"type": "string"}}
    if t in ("int[]", "uint[]", "[]int", "[]uint", "array<int>", "array<uint>"):
        return {"type": "array", "items": {"type": "integer"}}
    if t in ("object", "json", "map", "dict"):
        return {"type": "object"}
    if "[]" in t or "array" in t:
        return {"type": "array", "items": {"type": "string"}}
    # Default
    return {"type": "string"}


# ─────────────────────────────────────────────────────────────────────────────
# Markdown Parser
# ─────────────────────────────────────────────────────────────────────────────

# Match: ### `GET /path` or ### `POST /path` — description
ENDPOINT_RE = re.compile(
    r"^###\s+`(GET|POST|PUT|DELETE|PATCH|ANY|GET/POST|GET/PUT)\s+(/[^`]*)`"
)

# Match parameter table row: | name | type | required | description |
PARAM_ROW_RE = re.compile(
    r"^\|\s*(\w[\w\.\[\]]*)\s*\|\s*([^|]+?)\s*\|\s*([^|]+?)\s*\|\s*(.+?)\s*\|$"
)

# Match parameter table header: | 参数 | 类型 | 必填 | 说明 |
PARAM_HEADER_RE = re.compile(
    r"^\|\s*参数\s*\|\s*类型\s*\|\s*(必填|必须)\s*\|\s*说明\s*\|"
)

# Match Content-Type declaration
CONTENT_TYPE_RE = re.compile(
    r"\*\*Content-Type[：:]\*\*\s*`([^`]+)`"
)

# Match permission declaration
PERMISSION_RE = re.compile(
    r"\*\*权限[：:]\*\*\s*(.+)"
)


def parse_api_md(filepath):
    """Parse API.md and return list of endpoint definitions."""
    with open(filepath, "r", encoding="utf-8") as f:
        lines = f.readlines()

    endpoints = []
    current = None
    in_param_table = False
    section_lines = []

    for i, line in enumerate(lines):
        # Check for new endpoint heading
        m = ENDPOINT_RE.match(line)
        if m:
            # Save previous endpoint
            if current:
                _finalize_endpoint(current, section_lines)
                endpoints.append(current)

            methods_str = m.group(1)
            path = m.group(2).strip()
            # Handle multi-method like GET/POST
            methods = methods_str.split("/")

            current = {
                "methods": methods,
                "path": path,
                "summary": "",
                "description": "",
                "parameters": [],
                "content_type": None,
                "permission": "",
                "responses": {},
            }
            section_lines = []
            in_param_table = False
            continue

        if current is not None:
            section_lines.append(line)

            # Check for Content-Type
            ct_m = CONTENT_TYPE_RE.search(line)
            if ct_m and current["content_type"] is None:
                current["content_type"] = ct_m.group(1).strip()

            # Check for permission
            perm_m = PERMISSION_RE.search(line)
            if perm_m:
                current["permission"] = perm_m.group(1).strip()

            # Detect start of parameter table
            stripped = line.strip()
            if PARAM_HEADER_RE.match(stripped):
                in_param_table = True
                continue

            # Parse parameter rows
            if in_param_table:
                # Skip separator line
                if stripped.startswith("|---"):
                    continue
                pm = PARAM_ROW_RE.match(stripped)
                if pm:
                    param_name = pm.group(1).strip()
                    param_type = pm.group(2).strip()
                    param_required = pm.group(3).strip()
                    param_desc = pm.group(4).strip()
                    current["parameters"].append({
                        "name": param_name,
                        "type": param_type,
                        "required": param_required,
                        "description": param_desc,
                    })
                else:
                    # End of table
                    in_param_table = False

    # Don't forget the last endpoint
    if current:
        _finalize_endpoint(current, section_lines)
        endpoints.append(current)

    return endpoints


def _finalize_endpoint(endpoint, section_lines):
    """Extract summary from section content."""
    # First non-empty line after heading is the summary
    for line in section_lines:
        stripped = line.strip()
        if stripped and not stripped.startswith("-") and not stripped.startswith("|") and not stripped.startswith(">") and not stripped.startswith("```") and not stripped.startswith("*"):
            endpoint["summary"] = stripped[:120]
            break

    # Note: per-param query vs body split is handled in _build_operation
    # by checking each param's description for "Query" (line 314).
    # Do NOT set a global _param_location_hint here — it would override
    # the x-www-form-urlencoded per-param split logic and send ALL params
    # to query if any single param description mentions "Query 参数".


# ─────────────────────────────────────────────────────────────────────────────
# OpenAPI Generation
# ─────────────────────────────────────────────────────────────────────────────

def build_openapi(endpoints):
    """Build OpenAPI 3.0.3 spec dict from parsed endpoints."""
    spec = OrderedDict()
    spec["openapi"] = "3.0.3"
    spec["info"] = OrderedDict([
        ("title", "Hatchery API"),
        ("version", "1.0.0"),
        ("description", "Hatchery API - auto-generated from docs/API.md"),
    ])
    spec["servers"] = [{"url": "http://localhost:8080", "description": "Local development"}]

    spec["tags"] = [
        {"name": "auth", "description": "Authentication"},
        {"name": "instance", "description": "Instance management (openclaw)"},
        {"name": "admin", "description": "Admin panel"},
        {"name": "quota", "description": "User quota"},
        {"name": "llm", "description": "LLM Proxy"},
        {"name": "other", "description": "Other"},
    ]

    paths = OrderedDict()

    for ep in endpoints:
        path = ep["path"]
        # Normalize path params: {id}.png → {id}
        openapi_path = re.sub(r"\{([^}]+)\}\.\w+", r"{\1}", path)

        if openapi_path not in paths:
            paths[openapi_path] = OrderedDict()

        for method in ep["methods"]:
            method_lower = method.lower()
            if method_lower == "any":
                # Register both GET and POST for ANY
                for m in ["get", "post"]:
                    paths[openapi_path][m] = _build_operation(ep, m)
            else:
                paths[openapi_path][method_lower] = _build_operation(ep, method_lower)

    spec["paths"] = paths

    spec["components"] = OrderedDict([
        ("securitySchemes", OrderedDict([
            ("BearerAuth", OrderedDict([
                ("type", "http"),
                ("scheme", "bearer"),
                ("description", "API Token (hk-xxx) or Admin Token"),
            ])),
            ("CookieAuth", OrderedDict([
                ("type", "apiKey"),
                ("in", "cookie"),
                ("name", "hatchery-session"),
                ("description", "Session cookie from POST /login"),
            ])),
        ])),
    ])

    return spec


def _build_operation(ep, method):
    """Build a single operation object."""
    op = OrderedDict()

    # Tag
    path = ep["path"]
    if path.startswith("/admin/"):
        op["tags"] = ["admin"]
    elif path.startswith("/openclaw/"):
        op["tags"] = ["instance"]
    elif path.startswith("/quota/"):
        op["tags"] = ["quota"]
    elif path.startswith("/v1/"):
        op["tags"] = ["llm"]
    elif path.startswith("/login") or path.startswith("/logout") or path.startswith("/auth/") or path.startswith("/api-token"):
        op["tags"] = ["auth"]
    else:
        op["tags"] = ["other"]

    # Summary
    if ep["summary"]:
        op["summary"] = ep["summary"]

    # OperationId
    op["operationId"] = _make_operation_id(method, path)

    # Security
    perm = ep["permission"]
    if "公开" in perm:
        op["security"] = [{}]  # No auth required
    elif "管理员" in perm:
        op["security"] = [{"BearerAuth": []}]
    else:
        op["security"] = [{"BearerAuth": []}, {"CookieAuth": []}]

    # Parameters and RequestBody
    params = ep["parameters"]
    content_type = ep["content_type"]
    param_hint = ep.get("_param_location_hint", "")

    if not params:
        # No params defined
        pass
    elif method == "get":
        # GET → all params are query
        op["parameters"] = _build_query_params(params)
    elif content_type and "application/json" in content_type:
        # JSON body
        op["requestBody"] = _build_json_body(params)
    elif content_type and "multipart/form-data" in content_type:
        # Multipart
        op["requestBody"] = _build_multipart_body(params)
    elif param_hint == "query":
        # Explicit query hint
        op["parameters"] = _build_query_params(params)
    elif content_type and "x-www-form-urlencoded" in content_type:
        # Form body — but some params might be query (e.g. ?id=)
        query_params = [p for p in params if "Query" in p.get("description", "") or "query" in p.get("description", "").lower()]
        body_params = [p for p in params if p not in query_params]
        if query_params:
            op["parameters"] = _build_query_params(query_params)
        if body_params:
            op["requestBody"] = _build_form_body(body_params)
        elif not query_params:
            op["requestBody"] = _build_form_body(params)
    else:
        # Default: POST without explicit content-type → check if params look like query
        # Heuristic: if params have id/instance_id and it's a simple action, use query
        if all(p["name"] in ("id", "instance_id") for p in params):
            op["parameters"] = _build_query_params(params)
        elif method == "post":
            op["requestBody"] = _build_form_body(params)
        else:
            op["parameters"] = _build_query_params(params)

    # Responses
    op["responses"] = _build_responses(ep)

    return op


def _build_query_params(params):
    """Build OpenAPI query parameters list."""
    result = []
    for p in params:
        param = OrderedDict()
        param["name"] = p["name"]
        param["in"] = "query"
        param["required"] = p["required"] == "是"
        param["description"] = p["description"]
        param["schema"] = map_type(p["type"])
        result.append(param)
    return result


def _build_json_body(params):
    """Build OpenAPI requestBody with JSON schema."""
    properties = OrderedDict()
    required = []
    for p in params:
        properties[p["name"]] = map_type(p["type"])
        properties[p["name"]]["description"] = p["description"]
        if p["required"] == "是":
            required.append(p["name"])

    schema = OrderedDict([("type", "object"), ("properties", properties)])
    if required:
        schema["required"] = required

    return OrderedDict([
        ("required", bool(required)),
        ("content", OrderedDict([
            ("application/json", OrderedDict([
                ("schema", schema),
            ])),
        ])),
    ])


def _build_form_body(params):
    """Build OpenAPI requestBody with form-urlencoded schema."""
    properties = OrderedDict()
    required = []
    for p in params:
        properties[p["name"]] = map_type(p["type"])
        properties[p["name"]]["description"] = p["description"]
        if p["required"] == "是":
            required.append(p["name"])

    schema = OrderedDict([("type", "object"), ("properties", properties)])
    if required:
        schema["required"] = required

    return OrderedDict([
        ("required", bool(required)),
        ("content", OrderedDict([
            ("application/x-www-form-urlencoded", OrderedDict([
                ("schema", schema),
            ])),
        ])),
    ])


def _build_multipart_body(params):
    """Build OpenAPI requestBody with multipart/form-data schema."""
    properties = OrderedDict()
    required = []
    for p in params:
        if p["type"].lower() in ("file", "binary"):
            properties[p["name"]] = OrderedDict([
                ("type", "string"),
                ("format", "binary"),
                ("description", p["description"]),
            ])
        else:
            properties[p["name"]] = map_type(p["type"])
            properties[p["name"]]["description"] = p["description"]
        if p["required"] == "是":
            required.append(p["name"])

    schema = OrderedDict([("type", "object"), ("properties", properties)])
    if required:
        schema["required"] = required

    return OrderedDict([
        ("required", bool(required)),
        ("content", OrderedDict([
            ("multipart/form-data", OrderedDict([
                ("schema", schema),
            ])),
        ])),
    ])


def _build_responses(ep):
    """Build basic responses based on permission and content."""
    responses = OrderedDict()
    responses["200"] = OrderedDict([
        ("description", "Success"),
        ("content", OrderedDict([
            ("application/json", OrderedDict([
                ("schema", OrderedDict([("type", "object")])),
            ])),
        ])),
    ])

    perm = ep["permission"]
    if "登录" in perm or "管理员" in perm:
        responses["401"] = OrderedDict([("description", "Unauthorized")])
    if "管理员" in perm:
        responses["403"] = OrderedDict([("description", "Forbidden - admin required")])

    return responses


def _make_operation_id(method, path):
    """Generate operationId from method + path."""
    # /openclaw/list → openclaw_list
    # /admin/config/security-group → admin_config_security_group
    parts = path.strip("/").replace("-", "_").replace("{", "").replace("}", "").split("/")
    name = "_".join(parts) if parts and parts[0] else "root"
    return f"{method}_{name}"


# ─────────────────────────────────────────────────────────────────────────────
# Main
# ─────────────────────────────────────────────────────────────────────────────

def main():
    import argparse as _argparse
    parser = _argparse.ArgumentParser(description="Convert API.md to OpenAPI 3.0 JSON")
    parser.add_argument("--input", default="", help="Input API.md path (default: docs/API.md)")
    parser.add_argument("--output", default="", help="Output JSON path (default: docs/openapi.json)")
    args = parser.parse_args()

    script_dir = os.path.dirname(os.path.abspath(__file__))
    project_root = os.path.dirname(script_dir)
    api_md_path = args.input or os.path.join(project_root, "docs", "API.md")
    output_path = args.output or os.path.join(project_root, "docs", "openapi.json")

    if not os.path.exists(api_md_path):
        print(f"Error: {api_md_path} not found", file=sys.stderr)
        sys.exit(1)

    print(f"Parsing {api_md_path} ...")
    endpoints = parse_api_md(api_md_path)
    print(f"  Found {len(endpoints)} endpoint definitions")

    # Count unique paths
    unique_paths = set()
    for ep in endpoints:
        unique_paths.add(ep["path"])
    print(f"  Unique paths: {len(unique_paths)}")

    # Count endpoints with parameters
    with_params = sum(1 for ep in endpoints if ep["parameters"])
    print(f"  Endpoints with parameters: {with_params}")

    print(f"\nGenerating OpenAPI spec ...")
    spec = build_openapi(endpoints)

    # Count operations
    op_count = 0
    for path_item in spec["paths"].values():
        op_count += len(path_item)
    print(f"  Total operations: {op_count}")

    print(f"\nWriting {output_path} ...")
    content = spec_dump(spec)
    with open(output_path, "w", encoding="utf-8") as f:
        f.write(content)

    file_size = os.path.getsize(output_path)
    print(f"  Done! ({file_size} bytes)")
    print(f"\nValidation:")
    print(f"  Paths: {len(spec['paths'])}")
    print(f"  Operations: {op_count}")

    # Print sample
    print(f"\nSample paths (first 10):")
    for i, (path, methods) in enumerate(spec["paths"].items()):
        if i >= 10:
            break
        method_list = ", ".join(methods.keys())
        print(f"  {method_list.upper():12s} {path}")


if __name__ == "__main__":
    main()
