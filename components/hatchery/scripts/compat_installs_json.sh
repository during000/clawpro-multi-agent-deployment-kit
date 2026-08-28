#!/bin/bash
# =============================================================================
# OpenClaw 4.x → 5.x 兼容脚本：~/.openclaw/plugins/installs.json 路径修复
# -----------------------------------------------------------------------------
# 背景：
#   一键升级（4.x → 5.x）流程中，~/.openclaw/plugins/installs.json 通过 SMH
#   备份恢复机制覆盖回新镜像。但其中各插件条目的 installPath 仍为 4.x 时代的
#   旧路径（~/.openclaw/extensions/<plugin>），而 5.x 已将插件迁移到
#   ~/.openclaw/npm/node_modules/<scope>/<plugin>，且 postinstall 阶段清理了
#   旧 extensions/ 目录下的 node_modules，导致插件加载报错
#   "Cannot find module 'xxx'"。
#
# 本脚本作用：
#   1. 读取 ~/.openclaw/plugins/installs.json
#   2. 对白名单中声明的插件，若 installPath 仍指向旧 extensions/ 路径，则把
#      该插件对应的 JSON 配置块「整块替换」为符合 5.x schema 的新条目，字段含：
#        - source           固定 "npm"
#        - spec             "<npm_pkg>@<version>"
#        - installPath      ~/.openclaw/npm/node_modules/<npm_pkg>
#        - version          以 npm 包目录下 package.json 的 version 为准
#        - resolvedName     <npm_pkg>
#        - resolvedVersion  同 version
#        - resolvedSpec     同 spec
#      同时清理 4.x 残留字段（integrity / shasum / resolvedAt / installedAt 等）。
#   3. 原子写回，保留备份 installs.json.bak.<timestamp>
#
# 设计原则：
#   - 幂等：已经是新路径的条目自动跳过（noop）
#   - 安全：写入前后做 JSON 合法性校验（jq -e .），失败回滚
#   - 容错：单个插件处理失败不影响其他插件，最终汇总日志退出码
#   - 隔离：不修改未在白名单中的插件条目，最大程度避免误伤
#
# 输出（前端展示用）：每行一条 [compat-installs] 前缀的日志
# =============================================================================

set -u  # 不开 -e：单插件失败不终止全局

export PATH="$HOME/.npm-global/bin:$PATH"
export XDG_RUNTIME_DIR=/run/user/$(id -u)

# ========== 日志系统初始化 ==========
# 使用用户家目录，避免非 root 运行时无法写入 /var/log
# 同时输出到终端（供 RunScript 抓取并回传）和落盘日志文件（便于事后排查）
LOG_DIR="${HOME}/.openclaw/logs"
if [ ! -d "$LOG_DIR" ]; then
    mkdir -p "$LOG_DIR"
fi
SCRIPT_NAME="compat_installs_json"
LOG_FILE="$LOG_DIR/${SCRIPT_NAME}.log"
exec > >(tee -a "$LOG_FILE") 2>&1
echo ""
echo "========== 日志开始: $(date '+%Y-%m-%d %H:%M:%S') =========="

LOG_PREFIX="[compat-installs]"
TS="$(date +%s)"

INSTALLS_JSON="${HOME}/.openclaw/plugins/installs.json"
NPM_ROOT="${HOME}/.openclaw/npm/node_modules"
# openclaw >= 2026.5.28 起，部分插件（如 memory-tencentdb、openclaw-lark）安装位置
# 从 npm/node_modules/<scope>/<pkg> 迁移到 npm/projects/<scope-pkg-version-hash>，
# 这里作为回退探测路径，命中后直接使用真实子目录作为 installPath。
PROJECTS_ROOT="${HOME}/.openclaw/npm/projects"
EXT_ROOT="${HOME}/.openclaw/extensions"

echo "${LOG_PREFIX} 开始执行 installs.json 4.x → 5.x 路径兼容修复"
echo "${LOG_PREFIX} HOME=${HOME}"
echo "${LOG_PREFIX} installs.json=${INSTALLS_JSON}"
echo "${LOG_PREFIX} npm_root=${NPM_ROOT}"
echo "${LOG_PREFIX} projects_root=${PROJECTS_ROOT}"
echo "${LOG_PREFIX} log_file=${LOG_FILE}"

# ---------- 前置依赖检查 ----------
if ! command -v jq >/dev/null 2>&1; then
    echo "${LOG_PREFIX} ERROR: 缺少 jq 命令，无法处理 JSON，已跳过"
    exit 0  # 升级流程容错：兼容脚本失败不应阻断升级整体成功
fi

if [ ! -f "${INSTALLS_JSON}" ]; then
    echo "${LOG_PREFIX} installs.json 不存在，无需处理"
    exit 0
fi

if ! jq -e . "${INSTALLS_JSON}" >/dev/null 2>&1; then
    echo "${LOG_PREFIX} ERROR: installs.json 不是合法 JSON，已跳过（避免误伤）"
    exit 0
fi

# ---------- 备份 ----------
BACKUP_FILE="${INSTALLS_JSON}.bak.${TS}"
if cp -p "${INSTALLS_JSON}" "${BACKUP_FILE}"; then
    echo "${LOG_PREFIX} 已备份原文件到 ${BACKUP_FILE}"
else
    echo "${LOG_PREFIX} ERROR: 备份失败，已跳过（避免无备份直接修改）"
    exit 0
fi

# =============================================================================
# === WHITELIST BEGIN =========================================================
# 白名单格式：每行一条，"<plugin_id> <npm_package_name> [projects_glob]"
#   - plugin_id        ：installs.json 中的 key（即 4.x 旧条目的 id）
#   - npm_package_name ：5.x 在 ~/.openclaw/npm/node_modules/ 下的实际目录名
#                        （含 @scope，例如 "@tencentdb-agent-memory/memory-tencentdb"，
#                          或不带 scope 的 plain 包名，例如 "lightclawbot"）
#   - projects_glob    ：可选第三列。openclaw >= 2026.5.28 起，包安装位置变为
#                        ~/.openclaw/npm/projects/<glob>（带版本/哈希后缀）。
#                        当 npm/node_modules/<npm_pkg> 不存在时，回退到该 glob
#                        匹配 ~/.openclaw/npm/projects/ 下的真实目录。
#
# 注意：
#   1. 仅对白名单中显式声明的插件做重写，未列出的插件保持原样
#   2. 若 node_modules 与 projects 路径都找不到包，跳过该插件（不破坏旧条目）
#   3. 后续新增兼容项请在此处追加，无需修改下方处理逻辑
# -----------------------------------------------------------------------------
WHITELIST=(
    "memory-tencentdb @tencentdb-agent-memory/memory-tencentdb tencentdb-agent-memory-memory-tencentdb-*"
)
# === WHITELIST END ===========================================================
# =============================================================================

if [ ${#WHITELIST[@]} -eq 0 ]; then
    echo "${LOG_PREFIX} 白名单为空，无需处理（noop）"
    exit 0
fi

# ---------- 处理函数 ----------
# 参数：$1 = plugin_id, $2 = npm_package_name, $3 = projects_glob (可选)
# 返回：0 已处理 / 1 已是新路径(noop) / 2 跳过(npm 包不存在) / 3 错误
process_plugin() {
    local plugin_id="$1"
    local npm_pkg="$2"
    local projects_glob="${3:-}"

    # 1) 双路径探测：优先老路径 npm/node_modules/<npm_pkg>（< 5.28），
    #    找不到时回退到新路径 npm/projects/<projects_glob>（>= 5.28）。
    #    采用磁盘自驱动策略，不依赖 openclaw 版本判断，幂等性更强。
    local npm_pkg_dir=""
    if [ -d "${NPM_ROOT}/${npm_pkg}" ]; then
        npm_pkg_dir="${NPM_ROOT}/${npm_pkg}"
    elif [ -n "${projects_glob}" ]; then
        # glob 匹配 projects/ 下带版本/哈希后缀的目录；存在多个时取首个命中
        # shellcheck disable=SC2086
        for d in ${PROJECTS_ROOT}/${projects_glob}; do
            if [ -d "$d" ]; then
                npm_pkg_dir="$d"
                break
            fi
        done
    fi

    if [ -z "${npm_pkg_dir}" ]; then
        echo "${LOG_PREFIX} [${plugin_id}] SKIP: npm 包目录不存在（已尝试 ${NPM_ROOT}/${npm_pkg}${projects_glob:+ 与 ${PROJECTS_ROOT}/${projects_glob}}）"
        return 2
    fi
    local pkg_json="${npm_pkg_dir}/package.json"
    if [ ! -f "${pkg_json}" ]; then
        echo "${LOG_PREFIX} [${plugin_id}] SKIP: package.json 不存在 ${pkg_json}"
        return 2
    fi

    # 2) 读取 installs.json 中现有条目（5.x 把所有插件条目放在 .installRecords 下）
    local current_path
    current_path="$(jq -r --arg id "${plugin_id}" '.installRecords[$id].installPath // empty' "${INSTALLS_JSON}")"
    if [ -z "${current_path}" ]; then
        echo "${LOG_PREFIX} [${plugin_id}] SKIP: installs.json 中无此插件条目"
        return 2
    fi

    # 3) 幂等：已经是 npm 路径则跳过（含 node_modules 与 projects 两种 5.x 安装位置）
    case "${current_path}" in
        */npm/node_modules/*|*/npm/projects/*)
            echo "${LOG_PREFIX} [${plugin_id}] NOOP: 已指向 npm 路径 ${current_path}"
            return 1
            ;;
    esac

    # 4) 从 npm 目录下的 package.json 读取真实的 name / version
    #    一切以 package.json 为准（避免白名单手写笔误）：
    #      - version：覆盖 installs.json 中遗留的 4.x 旧版本号（如 0.3.3 → 0.3.4）
    #      - name   ：用于 resolvedName / spec / resolvedSpec
    local pkg_name pkg_version
    pkg_name="$(jq -r '.name    // ""' "${pkg_json}")"
    pkg_version="$(jq -r '.version // ""' "${pkg_json}")"

    if [ -z "${pkg_version}" ]; then
        echo "${LOG_PREFIX} [${plugin_id}] ERROR: package.json 缺少 version 字段"
        return 3
    fi
    if [ -z "${pkg_name}" ]; then
        echo "${LOG_PREFIX} [${plugin_id}] ERROR: package.json 缺少 name 字段"
        return 3
    fi
    # 校验白名单声明的包名与 package.json 中实际的 name 一致（防止白名单写错）
    if [ "${pkg_name}" != "${npm_pkg}" ]; then
        echo "${LOG_PREFIX} [${plugin_id}] WARN: 白名单声明 ${npm_pkg}，但 package.json name 为 ${pkg_name}，以 package.json 为准"
    fi

    # 5) 构造完整的 5.x 标准条目（整块替换，不保留 4.x 残留字段）
    #    schema 与 installs.json 中其他原生 5.x 条目（如 dingtalk-connector / feishu）保持一致：
    #      source / spec / installPath / version / resolvedName / resolvedVersion / resolvedSpec
    local new_spec="${pkg_name}@${pkg_version}"
    local new_block
    new_block="$(jq -n \
        --arg source          "npm" \
        --arg spec            "${new_spec}" \
        --arg installPath     "${npm_pkg_dir}" \
        --arg version         "${pkg_version}" \
        --arg resolvedName    "${pkg_name}" \
        --arg resolvedVersion "${pkg_version}" \
        --arg resolvedSpec    "${new_spec}" \
        '{
            source:          $source,
            spec:            $spec,
            installPath:     $installPath,
            version:         $version,
            resolvedName:    $resolvedName,
            resolvedVersion: $resolvedVersion,
            resolvedSpec:    $resolvedSpec
        }')"

    # 6) 原子整块替换：直接覆盖该 plugin_id 对应的整块配置，
    #    清理 4.x 残留字段（integrity / shasum / resolvedAt / installedAt 等），
    #    保证与原生 5.x 条目 schema 一致
    local tmp_file="${INSTALLS_JSON}.tmp.${TS}.$$"
    if jq --arg id "${plugin_id}" --argjson newblock "${new_block}" \
            '.installRecords[$id] = $newblock' \
            "${INSTALLS_JSON}" > "${tmp_file}"; then

        # 校验生成的 JSON 合法
        if jq -e . "${tmp_file}" >/dev/null 2>&1; then
            mv "${tmp_file}" "${INSTALLS_JSON}"
            echo "${LOG_PREFIX} [${plugin_id}] OK: ${current_path} → ${npm_pkg_dir} (v${pkg_version})"
            return 0
        else
            rm -f "${tmp_file}"
            echo "${LOG_PREFIX} [${plugin_id}] ERROR: 生成的 JSON 不合法，已回滚"
            return 3
        fi
    else
        rm -f "${tmp_file}"
        echo "${LOG_PREFIX} [${plugin_id}] ERROR: jq 处理失败"
        return 3
    fi
}

# ---------- 主循环 ----------
total=0
ok_count=0
noop_count=0
skip_count=0
err_count=0

for entry in "${WHITELIST[@]}"; do
    # 跳过空行/注释
    [ -z "${entry}" ] && continue
    case "${entry}" in \#*) continue ;; esac

    plugin_id="$(echo "${entry}"     | awk '{print $1}')"
    npm_pkg="$(echo "${entry}"       | awk '{print $2}')"
    projects_glob="$(echo "${entry}" | awk '{print $3}')"

    if [ -z "${plugin_id}" ] || [ -z "${npm_pkg}" ]; then
        echo "${LOG_PREFIX} WARN: 白名单格式错误，跳过：${entry}"
        continue
    fi

    total=$((total + 1))
    process_plugin "${plugin_id}" "${npm_pkg}" "${projects_glob}"
    case $? in
        0) ok_count=$((ok_count + 1)) ;;
        1) noop_count=$((noop_count + 1)) ;;
        2) skip_count=$((skip_count + 1)) ;;
        3) err_count=$((err_count + 1)) ;;
    esac
done

# ---------- 汇总 ----------
echo "${LOG_PREFIX} 处理完成: total=${total} ok=${ok_count} noop=${noop_count} skip=${skip_count} err=${err_count}"

if [ "${ok_count}" -eq 0 ] && [ "${err_count}" -eq 0 ]; then
    # 全部 noop/skip，回滚多余备份
    rm -f "${BACKUP_FILE}"
    echo "${LOG_PREFIX} 无实际修改，已删除多余备份 ${BACKUP_FILE}"
fi

echo "========== 日志结束: $(date '+%Y-%m-%d %H:%M:%S') =========="
echo ""

# 兼容脚本永远以 0 退出，避免阻断升级主流程；问题通过日志体现
exit 0
