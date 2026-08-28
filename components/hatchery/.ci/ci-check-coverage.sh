#!/usr/bin/env bash
# CI 脚本：检查 PR 的单元测试覆盖率（全量 + 增量）
# 用法：BASE_BRANCH=master bash .ci/ci-check-coverage.sh
# 需要：go, git
#
# 环境变量：
#   BASE_BRANCH            基线分支，默认 master
#   TOTAL_THRESHOLD        全量覆盖率阈值（%），默认 0（不检查）
#   INCREMENTAL_THRESHOLD  增量覆盖率阈值（%），默认 60
#   EXCLUDE_PATTERNS       排除的文件 glob 模式（逗号分隔），默认空

set -euo pipefail

BASE_BRANCH="${BASE_BRANCH:-master}"
TOTAL_THRESHOLD="${TOTAL_THRESHOLD:-0}"
INCREMENTAL_THRESHOLD="${INCREMENTAL_THRESHOLD:-60}"
EXCLUDE_PATTERNS="${EXCLUDE_PATTERNS:-}"
COVERAGE_FILE="coverage.out"
REPORT_DIR="coverage-report"
REPORT_FILE="index.html"

# ---------------------------------------------------------------------------
# 辅助函数：将行号列表压缩为范围表示（如 45-48, 62, 123-125）
# ---------------------------------------------------------------------------
format_line_ranges() {
    local lines=("$@")
    # 排序
    IFS=$'\n' sorted=($(sort -n <<< "${lines[*]}")); unset IFS

    local result=""
    local range_start=${sorted[0]}
    local range_end=${sorted[0]}

    for ((i = 1; i < ${#sorted[@]}; i++)); do
        if [ "${sorted[$i]}" -eq $((range_end + 1)) ]; then
            range_end=${sorted[$i]}
        else
            if [ "$range_start" -eq "$range_end" ]; then
                result="${result:+${result}, }${range_start}"
            else
                result="${result:+${result}, }${range_start}-${range_end}"
            fi
            range_start=${sorted[$i]}
            range_end=${sorted[$i]}
        fi
    done
    # 最后一段
    if [ "$range_start" -eq "$range_end" ]; then
        result="${result:+${result}, }${range_start}"
    else
        result="${result:+${result}, }${range_start}-${range_end}"
    fi
    echo "$result"
}

# ---------------------------------------------------------------------------
# 0. 前置检查
# ---------------------------------------------------------------------------
if ! command -v go &>/dev/null; then
    echo "ERROR: go is required but not found" >&2
    exit 1
fi

echo "==> Fetching base branch ${BASE_BRANCH}..."
git fetch origin "$BASE_BRANCH" --quiet 2>/dev/null || true
BASE_REF="origin/${BASE_BRANCH}"

# ---------------------------------------------------------------------------
# 1. 获取 PR 变更的 Go 源文件
# ---------------------------------------------------------------------------
echo "==> Detecting changed Go files against ${BASE_REF}..."

ALL_CHANGED_GO=$(git diff --name-only --diff-filter=ACMR "$BASE_REF"...HEAD -- '*.go' \
    | grep -v '^vendor/' \
    | grep -v '^test/' \
    || true)

# 无任何 Go 文件变更（源码+测试）→ 跳过
if [ -z "$ALL_CHANGED_GO" ]; then
    echo "==> No Go files changed in this PR. PASS"
    exit 0
fi

CHANGED_GO_FILES=$(echo "$ALL_CHANGED_GO" | grep -v '_test\.go$' || true)

# 应用排除模式
if [ -n "$EXCLUDE_PATTERNS" ] && [ -n "$CHANGED_GO_FILES" ]; then
    IFS=',' read -ra PATTERNS <<< "$EXCLUDE_PATTERNS"
    for pat in "${PATTERNS[@]}"; do
        pat=$(echo "$pat" | xargs)  # trim whitespace
        CHANGED_GO_FILES=$(echo "$CHANGED_GO_FILES" | grep -v "$pat" || true)
    done
fi

if [ -z "$CHANGED_GO_FILES" ]; then
    echo "==> No Go source files changed in this PR. Skipping incremental coverage."
else
    echo "  Changed files:"
    echo "$CHANGED_GO_FILES" | sed 's/^/    /'
fi

# ---------------------------------------------------------------------------
# 2. 获取变更文件所属的 Go 包
# ---------------------------------------------------------------------------
CHANGED_PACKAGES=""
if [ -n "$CHANGED_GO_FILES" ]; then
    while IFS= read -r f; do
        dir=$(dirname "$f")
        if [ "$dir" = "." ]; then
            pkg="."
        else
            pkg="./${dir}"
        fi
        if ! echo "$CHANGED_PACKAGES" | grep -qx "$pkg" 2>/dev/null; then
            CHANGED_PACKAGES="${CHANGED_PACKAGES:+${CHANGED_PACKAGES}
}${pkg}"
        fi
    done <<< "$CHANGED_GO_FILES"

    echo "  Related packages:"
    echo "$CHANGED_PACKAGES" | sed 's/^/    /'
fi

# ---------------------------------------------------------------------------
# 3. 运行测试 & 生成覆盖率数据
# ---------------------------------------------------------------------------
echo ""
echo "==> Running tests with coverage (full project)..."

MODULE_NAME=$(head -1 go.mod | awk '{print $2}')

# 全量运行所有包的测试，统计全项目覆盖率
# -coverpkg=./... 确保无测试文件的包也计入覆盖率分母
go test ./... \
    -coverprofile="$COVERAGE_FILE" \
    -covermode=atomic \
    -coverpkg=./... \
    -count=1 \
    -timeout=600s

echo "  Coverage profile generated: $COVERAGE_FILE"

# ---------------------------------------------------------------------------
# 4. 计算全量覆盖率（全项目）
# ---------------------------------------------------------------------------
echo ""
echo "==> Calculating coverage..."

TOTAL_COVERAGE=$(go tool cover -func="$COVERAGE_FILE" | tail -1 | awk '{print $NF}' | tr -d '%')
echo "  Total coverage: ${TOTAL_COVERAGE}%"

# ---------------------------------------------------------------------------
# 5. 计算增量覆盖率（仅 PR 变更行）
#    使用 awk 一次性完成，避免 shell 嵌套循环启动大量子进程
# ---------------------------------------------------------------------------

COVERED=0
NOT_COVERED=0
TOTAL_INCREMENTAL=0
UNCOVERED_DETAILS=""
INCREMENTAL_COVERAGE="N/A"

if [ -n "$CHANGED_GO_FILES" ]; then

# 5a. 从 git diff 提取每个文件的新增行号 → 临时文件（格式: file:line）
ADDED_LINES_FILE=$(mktemp)
while IFS= read -r file; do
    [ -z "$file" ] && continue
    git diff --unified=0 "$BASE_REF"...HEAD -- "$file" | \
        awk -v f="$file" '/^@@/ {
            for (i = 1; i <= NF; i++) {
                if (substr($i, 1, 1) == "+") {
                    range = substr($i, 2)
                    split(range, parts, ",")
                    start = parts[1] + 0
                    count = (parts[2] != "") ? parts[2] + 0 : 1
                    for (j = start; j < start + count; j++)
                        print f ":" j
                    break
                }
            }
        }' >> "$ADDED_LINES_FILE"
done <<< "$CHANGED_GO_FILES"

# 5b. 用 awk 一次性交叉比对 coverage.out 与变更行
# 注意：-coverpkg 会导致多个测试包对同一文件生成多条 coverage 记录，
# 同一区间可能在某包中 count=0、在另一包中 count>0。
# 因此读取阶段按 path+区间去重，取 max(count)。
INCREMENTAL_RESULT=$(awk -v module="$MODULE_NAME" '
BEGIN { covered = 0; not_covered = 0 }

# 第一个文件: coverage.out
# 按 path + startLine.startCol + endLine.endCol 去重，取最大 hit count
FILENAME == ARGV[1] {
    if (/^mode:/) next
    # module/file.go:startLine.startCol,endLine.endCol numStmt count
    idx = index($0, ":")
    path = substr($0, 1, idx - 1)
    rest = substr($0, idx + 1)
    split(rest, tokens, " ")
    loc = tokens[1]; cnt = tokens[3] + 0
    split(loc, lr, ",")
    split(lr[1], sl, "."); split(lr[2], el, ".")
    s = sl[1] + 0; e = el[1] + 0
    sc = sl[2] + 0; ec = el[2] + 0
    key = path SUBSEP s SUBSEP sc SUBSEP e SUBSEP ec
    if (!(key in dedup) || cnt > dedup[key]) {
        if (!(key in dedup)) {
            n = cov_n[path] + 0
            cov_s[path, n] = s
            cov_e[path, n] = e
            cov_sc[path, n] = sc
            cov_ec[path, n] = ec
            cov_idx[key] = n
            cov_n[path] = n + 1
        }
        dedup[key] = cnt
        cov_h[path, cov_idx[key]] = cnt
    }
    # 记录包级别有 coverage 数据
    pkg = path; sub("/[^/]+$", "", pkg)
    pkg_has_cov[pkg] = 1
    next
}

# 第二个文件: 变更行列表
FILENAME == ARGV[2] {
    idx = index($0, ":")
    file = substr($0, 1, idx - 1)
    line = substr($0, idx + 1) + 0
    cp = module "/" file
    nc = cov_n[cp] + 0
    if (nc == 0) {
        # 文件不在 coverage profile 中，检查同包是否有 coverage 数据
        pkg = cp; sub("/[^/]+$", "", pkg)
        if (pkg in pkg_has_cov) {
            # 包有测试但该文件无可执行语句（纯 const/type/var 声明），跳过
            next
        }
        # 包确实无测试，无法精确判断可执行行，走启发式过滤
        no_cov[file] = no_cov[file] (no_cov[file] ? " " : "") line
        next
    }
    matched = 0; best_cnt = 0
    for (i = 0; i < nc; i++) {
        if (line >= cov_s[cp, i] && line <= cov_e[cp, i]) {
            matched = 1
            if (cov_h[cp, i] > best_cnt) { best_cnt = cov_h[cp, i] }
        }
    }
    if (matched) {
        if (best_cnt > 0) covered++
        else { not_covered++; uncov[file] = uncov[file] (uncov[file] ? " " : "") line }
        next
    }
    # 不在任何 coverage 区间 → 不可执行行，不计入
}

END {
    print covered, not_covered
    for (f in uncov) print "UNCOV:" f ":" uncov[f]
    for (f in no_cov) print "NOCOV:" f ":" no_cov[f]
}
' "$COVERAGE_FILE" "$ADDED_LINES_FILE")

rm -f "$ADDED_LINES_FILE"

# 5c. 解析 awk 输出
COVERED=$(echo "$INCREMENTAL_RESULT" | head -1 | awk '{print $1}')
NOT_COVERED=$(echo "$INCREMENTAL_RESULT" | head -1 | awk '{print $2}')
UNCOVERED_DETAILS=""
NOCOV_FILES=""

while IFS= read -r uline; do
    case "$uline" in
        UNCOV:*)
            entry="${uline#UNCOV:}"
            file="${entry%%:*}"
            lines_str="${entry#*:}"
            read -ra nums <<< "$lines_str"
            ranges=$(format_line_ranges "${nums[@]}")
            UNCOVERED_DETAILS="${UNCOVERED_DETAILS}  ${file}: ${ranges}
"
            ;;
        NOCOV:*)
            entry="${uline#NOCOV:}"
            file="${entry%%:*}"
            lines_str="${entry#*:}"
            # 包没有测试 → 无 coverage profile。
            # 回查源文件，只过滤空行和 // 注释行，其余全算未覆盖。
            exec_lines=()
            read -ra all_nums <<< "$lines_str"
            for ln in "${all_nums[@]}"; do
                line_content=$(awk -v n="$ln" 'NR==n{print;exit}' "$file" 2>/dev/null || true)
                trimmed=$(echo "$line_content" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
                [ -z "$trimmed" ] && continue                           # 空行
                [[ "$trimmed" == //* ]] && continue                     # // 注释
                exec_lines+=("$ln")
            done
            if [ ${#exec_lines[@]} -gt 0 ]; then
                NOT_COVERED=$((NOT_COVERED + ${#exec_lines[@]}))
                ranges=$(format_line_ranges "${exec_lines[@]}")
                UNCOVERED_DETAILS="${UNCOVERED_DETAILS}  ${file}: ${ranges}
"
                NOCOV_FILES="${NOCOV_FILES}  ${file} (no test in package, ${#exec_lines[@]} executable lines uncovered)
"
            fi
            ;;
    esac
done <<< "$INCREMENTAL_RESULT"

if [ -n "$NOCOV_FILES" ]; then
    echo ""
    echo "  WARNING: The following changed files have no test coverage data"
    echo "  (package has no tests). Executable lines are counted as uncovered:"
    echo "$NOCOV_FILES"
fi

# 计算增量覆盖率
TOTAL_INCREMENTAL=$((COVERED + NOT_COVERED))
if [ "$TOTAL_INCREMENTAL" -gt 0 ]; then
    INCREMENTAL_COVERAGE=$(awk "BEGIN {printf \"%.1f\", ${COVERED} / ${TOTAL_INCREMENTAL} * 100}")
fi

fi  # end of [ -n "$CHANGED_GO_FILES" ]

# ---------------------------------------------------------------------------
# 6. 输出报告（终端 + HTML）
# ---------------------------------------------------------------------------
echo ""
echo "============================================================"
echo "  PR Coverage Report"
echo "============================================================"
echo ""

FAILED=false

# --- 全量覆盖率判断 ---
TOTAL_STATUS="disabled"
if [ "$TOTAL_THRESHOLD" -gt 0 ]; then
    TOTAL_PASS=$(awk "BEGIN {print ($TOTAL_COVERAGE >= $TOTAL_THRESHOLD) ? 1 : 0}")
    if [ "$TOTAL_PASS" -eq 1 ]; then
        echo "  Total coverage:        ${TOTAL_COVERAGE}%  (threshold: ${TOTAL_THRESHOLD}%)  PASS"
        TOTAL_STATUS="pass"
    else
        echo "  Total coverage:        ${TOTAL_COVERAGE}%  (threshold: ${TOTAL_THRESHOLD}%)  FAIL"
        TOTAL_STATUS="fail"
        FAILED=true
    fi
else
    echo "  Total coverage:        ${TOTAL_COVERAGE}%  (threshold: disabled)"
fi

# --- 增量覆盖率判断 ---
INC_STATUS="disabled"
if [ "$INCREMENTAL_COVERAGE" = "N/A" ]; then
    echo "  Incremental coverage:  N/A (no executable lines changed)"
    INC_STATUS="na"
elif [ "$INCREMENTAL_THRESHOLD" -gt 0 ]; then
    INC_PASS=$(awk "BEGIN {print ($INCREMENTAL_COVERAGE >= $INCREMENTAL_THRESHOLD) ? 1 : 0}")
    if [ "$INC_PASS" -eq 1 ]; then
        echo "  Incremental coverage:  ${INCREMENTAL_COVERAGE}%  (${COVERED}/${TOTAL_INCREMENTAL} lines, threshold: ${INCREMENTAL_THRESHOLD}%)  PASS"
        INC_STATUS="pass"
    else
        echo "  Incremental coverage:  ${INCREMENTAL_COVERAGE}%  (${COVERED}/${TOTAL_INCREMENTAL} lines, threshold: ${INCREMENTAL_THRESHOLD}%)  FAIL"
        INC_STATUS="fail"
        FAILED=true
    fi
else
    echo "  Incremental coverage:  ${INCREMENTAL_COVERAGE}%  (${COVERED}/${TOTAL_INCREMENTAL} lines, threshold: disabled)"
fi

echo ""

# --- 未覆盖行详情（终端） ---
if [ -n "$UNCOVERED_DETAILS" ]; then
    echo "------------------------------------------------------------"
    echo "  Uncovered changed lines:"
    echo "$UNCOVERED_DETAILS"
    echo "------------------------------------------------------------"
    echo ""
fi

# ---------------------------------------------------------------------------
# 7. 生成 go tool cover 原生 HTML（仅变更文件的覆盖率着色）
# ---------------------------------------------------------------------------
GO_COVER_HTML=""
if [ -f "$COVERAGE_FILE" ] && [ -n "$CHANGED_GO_FILES" ]; then
    # 从 coverage.out 中只保留变更文件的条目
    FILTERED_COVERAGE="coverage_filtered.out"
    head -1 "$COVERAGE_FILE" > "$FILTERED_COVERAGE"  # 保留 mode: 行
    while IFS= read -r file; do
        [ -z "$file" ] && continue
        cover_path="${MODULE_NAME}/${file}"
        grep "^${cover_path}:" "$COVERAGE_FILE" >> "$FILTERED_COVERAGE" 2>/dev/null || true
    done <<< "$CHANGED_GO_FILES"

    if [ "$(wc -l < "$FILTERED_COVERAGE")" -gt 1 ]; then
        go tool cover -html="$FILTERED_COVERAGE" -o coverage_detail.html 2>/dev/null || true
        if [ -f coverage_detail.html ]; then
            GO_COVER_HTML="coverage_detail.html"
        fi
    fi
    rm -f "$FILTERED_COVERAGE"
fi

# ---------------------------------------------------------------------------
# 8. 生成 HTML 报告
# ---------------------------------------------------------------------------
mkdir -p "$REPORT_DIR"

# 总体状态
if [ "$FAILED" = true ]; then
    OVERALL="FAILED"
    OVERALL_COLOR="#e03131"
    OVERALL_BG="#fff5f5"
else
    OVERALL="PASSED"
    OVERALL_COLOR="#2b8a3e"
    OVERALL_BG="#ebfbee"
fi

# 构建覆盖率环形图 SVG（百分比 → 弧度）
make_ring_svg() {
    local pct="$1" color="$2" label="$3"
    local r=40 cx=50 cy=50 sw=8
    local circumference=$(awk "BEGIN {printf \"%.2f\", 2 * 3.14159265 * $r}")
    local dash=$(awk "BEGIN {printf \"%.2f\", $circumference * $pct / 100}")
    local gap=$(awk "BEGIN {printf \"%.2f\", $circumference - $dash}")
    cat <<SVGEOF
<svg width="120" height="140" viewBox="0 0 100 120">
  <circle cx="$cx" cy="$cy" r="$r" fill="none" stroke="#e9ecef" stroke-width="$sw"/>
  <circle cx="$cx" cy="$cy" r="$r" fill="none" stroke="$color" stroke-width="$sw"
          stroke-dasharray="$dash $gap" stroke-linecap="round"
          transform="rotate(-90 $cx $cy)"/>
  <text x="$cx" y="$((cy+2))" text-anchor="middle" font-size="14" font-weight="700" fill="#212529">${pct}%</text>
  <text x="$cx" y="108" text-anchor="middle" font-size="9" fill="#868e96">$label</text>
</svg>
SVGEOF
}

# 覆盖率颜色
pct_color() {
    local pct="$1"
    local v=$(awk "BEGIN {print int($pct)}")
    if [ "$v" -ge 80 ]; then echo "#2b8a3e"
    elif [ "$v" -ge 60 ]; then echo "#e8590c"
    else echo "#e03131"
    fi
}

TOTAL_COV_COLOR=$(pct_color "$TOTAL_COVERAGE")
if [ "$INCREMENTAL_COVERAGE" = "N/A" ]; then
    INC_COV_COLOR="#868e96"
    INC_COV_DISPLAY="0"
else
    INC_COV_COLOR=$(pct_color "$INCREMENTAL_COVERAGE")
    INC_COV_DISPLAY="$INCREMENTAL_COVERAGE"
fi

PKG_RING=$(make_ring_svg "$TOTAL_COVERAGE" "$TOTAL_COV_COLOR" "Total")
INC_RING=$(make_ring_svg "$INC_COV_DISPLAY" "$INC_COV_COLOR" "Incremental")

# 构建每个变更文件的覆盖率表格行
# 使用 awk 对 coverage.out 做去重（按 path+startLine+endLine 取 max count）并按语句数加权
FILE_ROWS=""
if [ -n "$CHANGED_GO_FILES" ]; then
while IFS= read -r file; do
    [ -z "$file" ] && continue
    cover_path="${MODULE_NAME}/${file}"

    # 去重后按语句数加权计算：covered_stmts / total_stmts
    read -r file_covered file_total <<< "$(grep "^${cover_path}:" "$COVERAGE_FILE" 2>/dev/null | awk '
    {
        split($0, a, ":")
        rest = a[length(a)]
        split(rest, tokens, " ")
        loc = tokens[1]; stmts = tokens[2] + 0; cnt = tokens[3] + 0
        split(loc, lr, ",")
        split(lr[1], sl, "."); split(lr[2], el, ".")
        s = sl[1] + 0; e = el[1] + 0
        key = s SUBSEP e
        if (!(key in dedup) || cnt > dedup[key]) {
            dedup[key] = cnt
            stmt[key] = stmts
        }
    }
    END {
        total = 0; covered = 0
        for (k in dedup) {
            total += stmt[k]
            if (dedup[k] > 0) covered += stmt[k]
        }
        print covered, total
    }
    ')"
    file_covered=${file_covered:-0}
    file_total=${file_total:-0}

    if [ "$file_total" -gt 0 ]; then
        file_pct=$(awk "BEGIN {printf \"%.1f\", $file_covered / $file_total * 100}")
    else
        file_pct="0.0"
    fi
    file_color=$(pct_color "$file_pct")

    # 该文件未覆盖的变更行
    uncov_lines=""
    if echo "$UNCOVERED_DETAILS" | grep -q "^  ${file}:" 2>/dev/null; then
        uncov_lines=$(echo "$UNCOVERED_DETAILS" | grep "^  ${file}:" | sed "s|^  ${file}: ||")
    fi

    FILE_ROWS="${FILE_ROWS}
    <tr>
      <td style=\"padding:8px 12px;font-family:monospace;font-size:13px\">${file}</td>
      <td style=\"padding:8px 12px;text-align:center\">
        <span style=\"color:${file_color};font-weight:600\">${file_pct}%</span>
        <span style=\"color:#adb5bd;font-size:12px\">(${file_covered}/${file_total} stmts)</span>
      </td>
      <td style=\"padding:8px 12px;font-family:monospace;font-size:12px;color:#868e96\">${uncov_lines:-<span style='color:#2b8a3e'>-</span>}</td>
    </tr>"
done <<< "$CHANGED_GO_FILES"
fi  # end of [ -n "$CHANGED_GO_FILES" ] for file rows

# 详细覆盖率页面链接（如果有）
DETAIL_LINK=""
if [ -n "$GO_COVER_HTML" ]; then
    cp "$GO_COVER_HTML" "${REPORT_DIR}/detail.html"
    DETAIL_LINK="<a href=\"detail.html\" style=\"color:#1971c2;text-decoration:none;font-size:13px\">查看源码级覆盖率着色 &rarr;</a>"
fi

# 生成最终 HTML
cat > "${REPORT_DIR}/${REPORT_FILE}" <<HTMLEOF
<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>PR Coverage Report</title>
<style>
  * { margin:0; padding:0; box-sizing:border-box; }
  body { font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,"Helvetica Neue",Arial,sans-serif; background:#f8f9fa; color:#212529; padding:24px; }
  .container { max-width:900px; margin:0 auto; }
  .header { background:${OVERALL_BG}; border:1px solid ${OVERALL_COLOR}33; border-radius:12px; padding:20px 28px; margin-bottom:24px; display:flex; align-items:center; justify-content:space-between; }
  .header h1 { font-size:20px; color:#212529; }
  .badge { display:inline-block; padding:4px 14px; border-radius:20px; font-size:13px; font-weight:600; color:#fff; background:${OVERALL_COLOR}; }
  .cards { display:flex; gap:20px; margin-bottom:24px; }
  .card { flex:1; background:#fff; border:1px solid #dee2e6; border-radius:12px; padding:20px; text-align:center; }
  .card h3 { font-size:13px; color:#868e96; margin-bottom:4px; text-transform:uppercase; letter-spacing:0.5px; }
  .card .threshold { font-size:12px; color:#adb5bd; margin-top:4px; }
  .section { background:#fff; border:1px solid #dee2e6; border-radius:12px; padding:20px 24px; margin-bottom:20px; }
  .section h2 { font-size:16px; margin-bottom:14px; color:#343a40; }
  table { width:100%; border-collapse:collapse; }
  thead th { padding:8px 12px; text-align:left; font-size:12px; color:#868e96; text-transform:uppercase; letter-spacing:0.5px; border-bottom:2px solid #e9ecef; }
  tbody tr { border-bottom:1px solid #f1f3f5; }
  tbody tr:last-child { border-bottom:none; }
  .footer { text-align:center; color:#adb5bd; font-size:12px; margin-top:24px; }
</style>
</head>
<body>
<div class="container">

  <!-- 标题 -->
  <div class="header">
    <div>
      <h1>PR Coverage Report</h1>
      <span style="font-size:13px;color:#868e96">Base: ${BASE_BRANCH} &nbsp;|&nbsp; $(date '+%Y-%m-%d %H:%M:%S')</span>
    </div>
    <span class="badge">${OVERALL}</span>
  </div>

  <!-- 覆盖率概览卡片 -->
  <div class="cards">
    <div class="card">
      <h3>Total Coverage</h3>
      ${PKG_RING}
      <div class="threshold">Threshold: $([ "$TOTAL_THRESHOLD" -gt 0 ] && echo "${TOTAL_THRESHOLD}%" || echo "disabled")</div>
    </div>
    <div class="card">
      <h3>Incremental Coverage</h3>
      ${INC_RING}
      <div class="threshold">$([ "$INCREMENTAL_COVERAGE" = "N/A" ] && echo "No executable lines changed" || echo "${COVERED}/${TOTAL_INCREMENTAL} lines | Threshold: $([ "$INCREMENTAL_THRESHOLD" -gt 0 ] && echo "${INCREMENTAL_THRESHOLD}%" || echo "disabled")")</div>
    </div>
  </div>

  <!-- 变更文件明细 -->
  <div class="section">
    <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:14px">
      <h2 style="margin-bottom:0">Changed Files</h2>
      ${DETAIL_LINK}
    </div>
    <table>
      <thead>
        <tr>
          <th>File</th>
          <th style="text-align:center">Coverage</th>
          <th>Uncovered Lines</th>
        </tr>
      </thead>
      <tbody>
        ${FILE_ROWS}
      </tbody>
    </table>
  </div>

  <div class="footer">Generated by ci-check-coverage.sh</div>
</div>
</body>
</html>
HTMLEOF

echo "  HTML report generated: ${REPORT_DIR}/${REPORT_FILE}"

# ---------------------------------------------------------------------------
# 9. 判断结果
# ---------------------------------------------------------------------------
rm -f "$COVERAGE_FILE" "$GO_COVER_HTML"

if [ "$FAILED" = true ]; then
    echo "==> Coverage check FAILED"
    exit 1
else
    echo "==> Coverage check PASSED"
    exit 0
fi
