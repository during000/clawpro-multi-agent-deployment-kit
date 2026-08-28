#!/usr/bin/env python3
"""
API Coverage Analysis — compare integration test requests against OpenAPI spec.

Usage:
    python3 scripts/api_coverage.py --spec docs/openapi.json --data coverage-output/coverage-data.json

Outputs:
    - Route coverage: which operations were called
    - Parameter coverage: which params were passed
    - Status code distribution: boundary test completeness
    - Uncovered list: what's missing
"""

import argparse
import json
import os
import re
import sys
from collections import defaultdict


# ─────────────────────────────────────────────────────────────────────────────
# Path matching (OpenAPI path templates → regex)
# ─────────────────────────────────────────────────────────────────────────────

def path_to_regex(openapi_path):
    """Convert /captcha/{id} to regex pattern /captcha/[^/]+"""
    pattern = re.sub(r"\{[^}]+\}", r"[^/]+", openapi_path)
    return re.compile(f"^{pattern}$")


def build_route_index(spec):
    """Build a lookup: (method, compiled_regex) → (path_template, operation)"""
    index = []
    for path_template, methods in spec["paths"].items():
        regex = path_to_regex(path_template)
        for method, op in methods.items():
            index.append((method.upper(), regex, path_template, op))
    return index


def match_route(route_index, method, path):
    """Match a request to an OpenAPI operation. Returns (path_template, operation) or None."""
    for (op_method, regex, path_template, op) in route_index:
        if op_method == method and regex.match(path):
            return path_template, op
    return None, None


# ─────────────────────────────────────────────────────────────────────────────
# Extract expected params from operation
# ─────────────────────────────────────────────────────────────────────────────

def get_expected_params(op):
    """Extract all expected parameter names from an operation."""
    params = set()
    # Query/path parameters
    for p in op.get("parameters", []):
        params.add(p["name"])
    # Request body properties
    rb = op.get("requestBody", {}).get("content", {})
    for content_type, content in rb.items():
        schema = content.get("schema", {})
        for prop_name in schema.get("properties", {}).keys():
            params.add(prop_name)
    return params


def get_required_params(op):
    """Extract required parameter names."""
    required = set()
    for p in op.get("parameters", []):
        if p.get("required"):
            required.add(p["name"])
    rb = op.get("requestBody", {}).get("content", {})
    for content_type, content in rb.items():
        schema = content.get("schema", {})
        for name in schema.get("required", []):
            required.add(name)
    return required


# ─────────────────────────────────────────────────────────────────────────────
# Main analysis
# ─────────────────────────────────────────────────────────────────────────────

def analyze(spec, coverage_data):
    """Run coverage analysis and return structured results."""
    route_index = build_route_index(spec)

    # Prepare expected data from spec
    all_operations = {}  # key: "METHOD /path" → operation
    for path_template, methods in spec["paths"].items():
        for method, op in methods.items():
            key = f"{method.upper()} {path_template}"
            all_operations[key] = op

    # Track actual coverage
    route_hits = defaultdict(int)  # "METHOD /path" → call count
    param_hits = defaultdict(set)  # "METHOD /path" → set of param names seen
    status_hits = defaultdict(set)  # "METHOD /path" → set of status codes

    unmatched = []  # requests that didn't match any spec route

    for frame in coverage_data:
        method = frame["method"]
        path = frame["path"]
        status_code = frame["status_code"]
        query_keys = frame.get("query_keys", [])
        body_keys = frame.get("body_keys", [])

        path_template, op = match_route(route_index, method, path)
        if path_template is None:
            unmatched.append(f"{method} {path}")
            continue

        key = f"{method} {path_template}"
        route_hits[key] += 1
        status_hits[key].add(status_code)
        for k in query_keys:
            param_hits[key].add(k)
        for k in body_keys:
            param_hits[key].add(k)

    return {
        "all_operations": all_operations,
        "route_hits": route_hits,
        "param_hits": param_hits,
        "status_hits": status_hits,
        "unmatched": unmatched,
    }


# ─────────────────────────────────────────────────────────────────────────────
# Incremental analysis (new APIs in this PR vs base branch)
# ─────────────────────────────────────────────────────────────────────────────

def compute_incremental(spec, base_spec, results):
    """Find new endpoints and params added in this PR (not in base_spec)."""
    route_hits = results["route_hits"]
    param_hits = results["param_hits"]

    # Find new operations (in spec but not in base)
    new_operations = []  # list of "METHOD /path"
    for path, methods in spec["paths"].items():
        for method, op in methods.items():
            key = f"{method.upper()} {path}"
            # Check if this operation existed in base
            base_methods = base_spec.get("paths", {}).get(path, {})
            if method not in base_methods:
                new_operations.append(key)

    # Find new params (operation exists in base but has new params)
    new_params = {}  # "METHOD /path" → set of new param names
    for path, methods in spec["paths"].items():
        for method, op in methods.items():
            key = f"{method.upper()} {path}"
            base_methods = base_spec.get("paths", {}).get(path, {})
            if method not in base_methods:
                continue  # entirely new operation, already counted above
            base_op = base_methods[method]
            current_params = get_expected_params(op)
            base_params = get_expected_params(base_op)
            added = current_params - base_params
            if added:
                new_params[key] = added

    # Coverage status for new operations
    new_ops_covered = [op for op in new_operations if op in route_hits]
    new_ops_uncovered = [op for op in new_operations if op not in route_hits]

    # Coverage status for new params
    new_params_covered = 0
    new_params_total = 0
    new_params_detail = []  # list of (key, param_name, covered_bool)
    for key, params in new_params.items():
        actual = param_hits.get(key, set())
        for p in params:
            new_params_total += 1
            covered = p in actual
            if covered:
                new_params_covered += 1
            new_params_detail.append((key, p, covered))

    return {
        "new_operations": new_operations,
        "new_ops_covered": new_ops_covered,
        "new_ops_uncovered": new_ops_uncovered,
        "new_params": new_params,
        "new_params_covered": new_params_covered,
        "new_params_total": new_params_total,
        "new_params_detail": new_params_detail,
    }


# ─────────────────────────────────────────────────────────────────────────────
# Report generation
# ─────────────────────────────────────────────────────────────────────────────

def print_report(results):
    """Print coverage report to stdout."""
    all_ops = results["all_operations"]
    route_hits = results["route_hits"]
    param_hits = results["param_hits"]
    status_hits = results["status_hits"]
    unmatched = results["unmatched"]

    total_ops = len(all_ops)
    covered_ops = len(route_hits)
    uncovered_ops = total_ops - covered_ops

    # Param stats
    total_params = 0
    covered_params = 0
    for key, op in all_ops.items():
        expected = get_expected_params(op)
        total_params += len(expected)
        actual = param_hits.get(key, set())
        covered_params += len(expected & actual)

    print("=" * 70)
    print("  API Coverage Report")
    print("=" * 70)
    print()

    # ── Route Coverage ──
    pct = (covered_ops / total_ops * 100) if total_ops else 0
    print(f"  Route Coverage: {covered_ops}/{total_ops} ({pct:.1f}%)")
    print()

    # ── Parameter Coverage ──
    pct_p = (covered_params / total_params * 100) if total_params else 0
    print(f"  Parameter Coverage: {covered_params}/{total_params} ({pct_p:.1f}%)")
    print()

    # ── Status Code Coverage (boundary tests) ──
    boundary_complete = 0
    boundary_total = 0
    for key, op in all_ops.items():
        if key not in route_hits:
            continue
        # Check if endpoint has auth requirement
        security = op.get("security", [])
        needs_auth = any(s for s in security if s)  # non-empty security
        if needs_auth:
            boundary_total += 1
            statuses = status_hits.get(key, set())
            # Consider boundary tested if we see both success + an error code
            if statuses & {401, 403, 400, 404, 409}:
                boundary_complete += 1

    if boundary_total:
        pct_b = (boundary_complete / boundary_total * 100)
        print(f"  Boundary Test Coverage: {boundary_complete}/{boundary_total} ({pct_b:.1f}%)")
        print(f"    (Endpoints that returned at least one 4xx error code)")
    print()

    # ── Top covered routes ──
    print("-" * 70)
    print("  Top Covered Routes (by call count)")
    print("-" * 70)
    sorted_hits = sorted(route_hits.items(), key=lambda x: -x[1])
    for key, count in sorted_hits[:15]:
        statuses = sorted(status_hits.get(key, set()))
        status_str = ",".join(str(s) for s in statuses)
        print(f"    {count:4d} calls  {key:<45s} [{status_str}]")
    if len(sorted_hits) > 15:
        print(f"    ... and {len(sorted_hits) - 15} more")
    print()

    # ── Uncovered routes ──
    print("-" * 70)
    print(f"  Uncovered Routes ({uncovered_ops})")
    print("-" * 70)
    uncovered_list = sorted(k for k in all_ops if k not in route_hits)
    for key in uncovered_list:
        print(f"    {key}")
    print()

    # ── Parameter detail (per covered route) ──
    print("-" * 70)
    print("  Parameter Coverage Detail (covered routes with missing params)")
    print("-" * 70)
    missing_params_routes = []
    for key in sorted(route_hits.keys()):
        op = all_ops[key]
        expected = get_expected_params(op)
        if not expected:
            continue
        actual = param_hits.get(key, set())
        missing = expected - actual
        if missing:
            missing_params_routes.append((key, expected, actual, missing))

    for key, expected, actual, missing in missing_params_routes:
        covered_p = len(expected) - len(missing)
        print(f"  {key}  ({covered_p}/{len(expected)} params)")
        for p in sorted(missing):
            print(f"    ✗ {p}")
    if not missing_params_routes:
        print("    All covered routes have full parameter coverage!")
    print()

    # ── Unmatched requests ──
    if unmatched:
        unique_unmatched = sorted(set(unmatched))
        print("-" * 70)
        print(f"  Unmatched Requests ({len(unique_unmatched)} unique, not in spec)")
        print("-" * 70)
        for req in unique_unmatched[:20]:
            print(f"    {req}")
        if len(unique_unmatched) > 20:
            print(f"    ... and {len(unique_unmatched) - 20} more")
        print()

    print("=" * 70)


# ─────────────────────────────────────────────────────────────────────────────
# HTML Report
# ─────────────────────────────────────────────────────────────────────────────

def generate_html_report(results, output_path):
    """Generate a styled HTML coverage report."""
    import html as html_mod
    from datetime import datetime

    all_ops = results["all_operations"]
    route_hits = results["route_hits"]
    param_hits = results["param_hits"]
    status_hits = results["status_hits"]
    unmatched = results["unmatched"]

    total_ops = len(all_ops)
    covered_ops = len(route_hits)
    uncovered_ops = total_ops - covered_ops
    route_pct = (covered_ops / total_ops * 100) if total_ops else 0

    total_params = 0
    covered_params = 0
    for key, op in all_ops.items():
        expected = get_expected_params(op)
        total_params += len(expected)
        actual = param_hits.get(key, set())
        covered_params += len(expected & actual)
    param_pct = (covered_params / total_params * 100) if total_params else 0

    total_requests = sum(route_hits.values())

    # Build covered routes table rows
    sorted_hits = sorted(route_hits.items(), key=lambda x: -x[1])
    covered_rows = ""
    for key, count in sorted_hits:
        op = all_ops[key]
        expected = get_expected_params(op)
        actual = param_hits.get(key, set())
        param_covered = len(expected & actual)
        param_total = len(expected)
        statuses = sorted(status_hits.get(key, set()))
        status_badges = ""
        for s in statuses:
            color = "#2b8a3e" if s < 400 else "#e8590c" if s < 500 else "#e03131"
            status_badges += f'<span style="display:inline-block;padding:1px 6px;border-radius:3px;font-size:11px;background:{color}22;color:{color};margin-right:3px">{s}</span>'
        param_bar = ""
        if param_total > 0:
            pct = param_covered / param_total * 100
            bar_color = "#2b8a3e" if pct == 100 else "#e8590c" if pct >= 50 else "#e03131"
            param_bar = f'<div style="display:flex;align-items:center;gap:6px"><div style="flex:1;height:6px;background:#e9ecef;border-radius:3px;overflow:hidden"><div style="width:{pct:.0f}%;height:100%;background:{bar_color}"></div></div><span style="font-size:11px;color:#868e96;white-space:nowrap">{param_covered}/{param_total}</span></div>'
        else:
            param_bar = '<span style="font-size:11px;color:#adb5bd">—</span>'

        method, path = key.split(" ", 1)
        method_color = {"GET": "#1971c2", "POST": "#2b8a3e", "PUT": "#e8590c", "DELETE": "#e03131", "PATCH": "#9c36b5"}.get(method, "#495057")
        covered_rows += f'''<tr>
            <td style="padding:6px 10px"><span style="display:inline-block;width:50px;font-size:11px;font-weight:600;color:{method_color}">{html_mod.escape(method)}</span><code style="font-size:12px">{html_mod.escape(path)}</code></td>
            <td style="padding:6px 10px;text-align:center;font-family:monospace;font-size:12px">{count}</td>
            <td style="padding:6px 10px;min-width:120px">{param_bar}</td>
            <td style="padding:6px 10px">{status_badges}</td>
        </tr>'''

    # Build uncovered routes table rows
    uncovered_list = sorted(k for k in all_ops if k not in route_hits)
    uncovered_rows = ""
    for key in uncovered_list:
        method, path = key.split(" ", 1)
        method_color = {"GET": "#1971c2", "POST": "#2b8a3e", "PUT": "#e8590c", "DELETE": "#e03131", "PATCH": "#9c36b5"}.get(method, "#495057")
        op = all_ops[key]
        param_count = len(get_expected_params(op))
        uncovered_rows += f'''<tr>
            <td style="padding:5px 10px"><span style="display:inline-block;width:50px;font-size:11px;font-weight:600;color:{method_color}">{html_mod.escape(method)}</span><code style="font-size:12px">{html_mod.escape(path)}</code></td>
            <td style="padding:5px 10px;text-align:center;font-size:12px;color:#868e96">{param_count} params</td>
        </tr>'''

    # Build missing params section
    missing_params_html = ""
    for key in sorted(route_hits.keys()):
        op = all_ops[key]
        expected = get_expected_params(op)
        if not expected:
            continue
        actual = param_hits.get(key, set())
        missing = expected - actual
        if not missing:
            continue
        covered_p = len(expected) - len(missing)
        method, path = key.split(" ", 1)
        params_list = "".join(f'<span style="display:inline-block;padding:2px 8px;margin:2px;border-radius:3px;font-size:11px;background:#fff5f5;color:#e03131;border:1px solid #e0313133">{html_mod.escape(p)}</span>' for p in sorted(missing))
        missing_params_html += f'''<div style="margin-bottom:12px;padding:10px;background:#fff;border:1px solid #e9ecef;border-radius:6px">
            <div style="font-size:12px;font-weight:600;margin-bottom:4px"><code>{html_mod.escape(key)}</code> <span style="color:#868e96;font-weight:400">({covered_p}/{len(expected)} covered)</span></div>
            <div>Missing: {params_list}</div>
        </div>'''

    if not missing_params_html:
        missing_params_html = '<p style="color:#2b8a3e;font-size:13px">All covered routes have full parameter coverage!</p>'

    # Route coverage color
    route_color = "#2b8a3e" if route_pct >= 80 else "#e8590c" if route_pct >= 50 else "#e03131"
    param_color = "#2b8a3e" if param_pct >= 80 else "#e8590c" if param_pct >= 50 else "#e03131"

    report_html = f'''<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>API Coverage Report</title>
<style>
  * {{ margin:0; padding:0; box-sizing:border-box; }}
  body {{ font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,"Helvetica Neue",Arial,sans-serif; background:#f8f9fa; color:#212529; padding:24px; }}
  .container {{ max-width:1100px; margin:0 auto; }}
  .header {{ background:#fff; border:1px solid #dee2e6; border-radius:12px; padding:20px 28px; margin-bottom:24px; }}
  .header h1 {{ font-size:20px; color:#212529; margin-bottom:4px; }}
  .header .meta {{ font-size:12px; color:#868e96; }}
  .cards {{ display:flex; gap:16px; margin-bottom:24px; flex-wrap:wrap; }}
  .card {{ flex:1; min-width:140px; background:#fff; border:1px solid #dee2e6; border-radius:12px; padding:16px; text-align:center; }}
  .card h3 {{ font-size:11px; color:#868e96; margin-bottom:6px; text-transform:uppercase; letter-spacing:0.5px; }}
  .card .value {{ font-size:26px; font-weight:700; }}
  .card .sub {{ font-size:11px; color:#adb5bd; margin-top:2px; }}
  .section {{ background:#fff; border:1px solid #dee2e6; border-radius:12px; padding:18px 22px; margin-bottom:20px; }}
  .section h2 {{ font-size:15px; margin-bottom:12px; color:#343a40; }}
  table {{ width:100%; border-collapse:collapse; }}
  thead th {{ padding:6px 10px; text-align:left; font-size:11px; color:#868e96; text-transform:uppercase; letter-spacing:0.4px; border-bottom:2px solid #e9ecef; }}
  tbody tr {{ border-bottom:1px solid #f1f3f5; }}
  tbody tr:last-child {{ border-bottom:none; }}
  code {{ background:#f1f3f5; padding:1px 5px; border-radius:3px; font-size:12px; }}
  .progress-ring {{ display:inline-block; position:relative; width:80px; height:80px; }}
  .footer {{ text-align:center; color:#adb5bd; font-size:11px; margin-top:24px; }}
</style>
</head>
<body>
<div class="container">
  <div class="header">
    <h1>API Coverage Report</h1>
    <div class="meta">Generated: {datetime.now().strftime("%Y-%m-%d %H:%M:%S")} &nbsp;|&nbsp; {total_requests} total requests from integration tests</div>
  </div>

  <div class="cards">
    <div class="card">
      <h3>Route Coverage</h3>
      <div class="value" style="color:{route_color}">{route_pct:.1f}%</div>
      <div class="sub">{covered_ops} / {total_ops} operations</div>
    </div>
    <div class="card">
      <h3>Parameter Coverage</h3>
      <div class="value" style="color:{param_color}">{param_pct:.1f}%</div>
      <div class="sub">{covered_params} / {total_params} params</div>
    </div>
    <div class="card">
      <h3>Covered Routes</h3>
      <div class="value" style="color:#1971c2">{covered_ops}</div>
      <div class="sub">operations tested</div>
    </div>
    <div class="card">
      <h3>Uncovered</h3>
      <div class="value" style="color:#e03131">{uncovered_ops}</div>
      <div class="sub">operations not tested</div>
    </div>
    <div class="card">
      <h3>Total Requests</h3>
      <div class="value" style="color:#495057">{total_requests}</div>
      <div class="sub">HTTP calls</div>
    </div>
  </div>

  <div class="section">
    <h2>Covered Routes ({covered_ops})</h2>
    <table>
      <thead><tr><th>Endpoint</th><th style="text-align:center">Calls</th><th>Params</th><th>Status Codes</th></tr></thead>
      <tbody>{covered_rows}</tbody>
    </table>
  </div>

  <div class="section">
    <h2>Missing Parameters</h2>
    {missing_params_html}
  </div>

  <div class="section">
    <h2>Uncovered Routes ({uncovered_ops})</h2>
    <table>
      <thead><tr><th>Endpoint</th><th style="text-align:center">Expected Params</th></tr></thead>
      <tbody>{uncovered_rows}</tbody>
    </table>
  </div>

  {{incremental_section}}

  <div class="footer">Generated by hatchery api_coverage.py</div>
</div>
</body>
</html>'''

    # Build incremental section if available
    incremental = results.get("incremental")
    if incremental and (incremental["new_operations"] or incremental["new_params"]):
        new_ops = incremental["new_operations"]
        new_ops_covered = incremental["new_ops_covered"]
        new_ops_uncovered = incremental["new_ops_uncovered"]
        new_params_detail = incremental["new_params_detail"]
        new_params_covered = incremental["new_params_covered"]
        new_params_total = incremental["new_params_total"]

        inc_route_pct = (len(new_ops_covered) / len(new_ops) * 100) if new_ops else 0
        inc_param_pct = (new_params_covered / new_params_total * 100) if new_params_total else 0
        inc_route_color = "#2b8a3e" if inc_route_pct == 100 else "#e8590c" if inc_route_pct >= 50 else "#e03131"
        inc_param_color = "#2b8a3e" if inc_param_pct == 100 else "#e8590c" if inc_param_pct >= 50 else "#e03131"

        # New operations table
        new_ops_rows = ""
        for key in sorted(new_ops):
            covered = key in route_hits
            icon = "✓" if covered else "✗"
            color = "#2b8a3e" if covered else "#e03131"
            method, path = key.split(" ", 1)
            method_color = {"GET": "#1971c2", "POST": "#2b8a3e", "PUT": "#e8590c", "DELETE": "#e03131"}.get(method, "#495057")
            new_ops_rows += f'''<tr>
                <td style="padding:5px 10px"><span style="display:inline-block;width:50px;font-size:11px;font-weight:600;color:{method_color}">{method}</span><code style="font-size:12px">{path}</code></td>
                <td style="padding:5px 10px;text-align:center;color:{color};font-weight:600">{icon}</td>
            </tr>'''

        # New params table
        new_params_rows = ""
        for key, param, covered in sorted(new_params_detail):
            icon = "✓" if covered else "✗"
            color = "#2b8a3e" if covered else "#e03131"
            new_params_rows += f'''<tr>
                <td style="padding:4px 10px;font-size:12px"><code>{key}</code></td>
                <td style="padding:4px 10px;font-size:12px"><code>{param}</code></td>
                <td style="padding:4px 10px;text-align:center;color:{color};font-weight:600">{icon}</td>
            </tr>'''

        incremental_section = f'''
  <div class="section" style="border-color:#1971c233;background:#f0f7ff">
    <h2 style="color:#1971c2">🆕 Incremental Coverage (new in this PR)</h2>
    <div style="display:flex;gap:20px;margin-bottom:16px">
      <div style="padding:10px 16px;background:#fff;border-radius:8px;border:1px solid #dee2e6;text-align:center">
        <div style="font-size:11px;color:#868e96;text-transform:uppercase">New Routes</div>
        <div style="font-size:22px;font-weight:700;color:{inc_route_color}">{len(new_ops_covered)}/{len(new_ops)}</div>
        <div style="font-size:11px;color:#adb5bd">covered</div>
      </div>
      <div style="padding:10px 16px;background:#fff;border-radius:8px;border:1px solid #dee2e6;text-align:center">
        <div style="font-size:11px;color:#868e96;text-transform:uppercase">New Params</div>
        <div style="font-size:22px;font-weight:700;color:{inc_param_color}">{new_params_covered}/{new_params_total}</div>
        <div style="font-size:11px;color:#adb5bd">covered</div>
      </div>
    </div>
    {"<h3 style='font-size:13px;margin-bottom:8px'>New Endpoints</h3><table><thead><tr><th>Endpoint</th><th style=text-align:center>Covered</th></tr></thead><tbody>" + new_ops_rows + "</tbody></table>" if new_ops_rows else ""}
    {"<h3 style='font-size:13px;margin:12px 0 8px'>New Parameters</h3><table><thead><tr><th>Endpoint</th><th>Parameter</th><th style=text-align:center>Covered</th></tr></thead><tbody>" + new_params_rows + "</tbody></table>" if new_params_rows else ""}
  </div>'''
    else:
        incremental_section = ""

    report_html = report_html.replace("{incremental_section}", incremental_section)

    os.makedirs(os.path.dirname(output_path) if os.path.dirname(output_path) else ".", exist_ok=True)
    with open(output_path, "w", encoding="utf-8") as f:
        f.write(report_html)


# ─────────────────────────────────────────────────────────────────────────────
# CLI
# ─────────────────────────────────────────────────────────────────────────────

def main():
    parser = argparse.ArgumentParser(description="API Coverage Analysis")
    parser.add_argument("--spec", required=True, help="Path to OpenAPI spec (docs/openapi.json)")
    parser.add_argument("--base-spec", default="", help="Path to base branch OpenAPI spec for incremental analysis")
    parser.add_argument("--data", required=True, help="Path to coverage-data.json (merged from test scripts)")
    parser.add_argument("--json", action="store_true", help="Output as JSON instead of text")
    parser.add_argument("--html", default="", help="Output HTML report to this file path")
    args = parser.parse_args()

    with open(args.spec, "r", encoding="utf-8") as f:
        spec = json.load(f)

    with open(args.data, "r", encoding="utf-8") as f:
        coverage_data = json.load(f)

    base_spec = None
    if args.base_spec:
        try:
            with open(args.base_spec, "r", encoding="utf-8") as f:
                base_spec = json.load(f)
            print(f"Loaded base spec: {len(base_spec['paths'])} paths", file=sys.stderr)
        except Exception as e:
            print(f"Warning: failed to load base spec: {e}", file=sys.stderr)

    print(f"Loaded spec: {len(spec['paths'])} paths", file=sys.stderr)
    print(f"Loaded coverage data: {len(coverage_data)} requests", file=sys.stderr)

    results = analyze(spec, coverage_data)

    # Compute incremental (new endpoints/params not in base)
    if base_spec:
        results["incremental"] = compute_incremental(spec, base_spec, results)

    if args.html:
        generate_html_report(results, args.html)
        print(f"HTML report: {args.html}", file=sys.stderr)
    elif args.json:
        # JSON output for programmatic use
        output = {
            "total_operations": len(results["all_operations"]),
            "covered_operations": len(results["route_hits"]),
            "route_hits": {k: v for k, v in results["route_hits"].items()},
            "param_hits": {k: sorted(v) for k, v in results["param_hits"].items()},
            "status_hits": {k: sorted(v) for k, v in results["status_hits"].items()},
            "unmatched_count": len(set(results["unmatched"])),
        }
        print(json.dumps(output, indent=2, ensure_ascii=False))
    else:
        print_report(results)


if __name__ == "__main__":
    main()
