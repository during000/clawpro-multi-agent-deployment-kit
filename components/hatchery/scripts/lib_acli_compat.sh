#!/bin/bash
#
# lib_acli_compat.sh
#
# 作用：为 hermes 脚本提供 acli 兼容层。考虑到存量 hermes 实例尚未预装 acli，
# 本库提供"探测 → 自动安装 → 失败回退"的统一入口。各业务脚本通过 `# %INCLUDE%`
# 引入本库后，可调用 ensure_acli 决定走 acli 路径还是原有 harness/skillhub
# 兜底路径。
#
# 对外导出函数：
#   ensure_acli          → stdout: "acli" / "fallback"
#       探测 acli，若不存在则尝试静默安装；安装成功返回 "acli"，失败返回 "fallback"。
#       业务脚本在开头调用一次，后续复用变量。
#
#   has_acli                          → 退出码 0/1
#       仅探测当前 acli 是否可用，不触发安装。适用于轻量场景。
#
#   acli_install_url                  → 输出 install.sh 的 URL（便于调试日志）
#
# 安装来源：
#   curl -fsSL https://acli-1325194254.cos.accelerate.myqcloud.com/acli/install.sh \
#       | sudo env ACLI_PRODUCT=clawpro bash
#
# 约束：
#   - 本库不能 `set -euo pipefail`，避免污染调用方
#   - 所有诊断日志写到 stderr，stdout 仅返回标识 "acli" / "fallback"
#   - 安装失败时不抛错，由调用方决定是否进入 fallback

# shellcheck shell=bash

_ACLI_INSTALL_URL="https://acli-1325194254.cos.accelerate.myqcloud.com/acli/install.sh"
_ACLI_INSTALL_PRODUCT="clawpro"

acli_install_url() {
    printf '%s' "$_ACLI_INSTALL_URL"
}

# has_acli — 仅探测，不安装
has_acli() {
    command -v acli >/dev/null 2>&1
}

# _acli_compat_log — 内部日志，写到 stderr
_acli_compat_log() {
    printf '[lib_acli_compat] %s\n' "$*" >&2
}

# _acli_ensure_product — 确保 identity.yaml 中有 product: clawpro
# 存量实例安装 acli 时可能没传 ACLI_PRODUCT，导致 identity.yaml 缺少 product 字段
_acli_ensure_product() {
    local identity_file="/etc/acli/identity.yaml"
    [ -f "$identity_file" ] || return 0
    if grep -q "^product:" "$identity_file" 2>/dev/null; then
        return 0
    fi
    # 补写 product: clawpro
    _acli_compat_log "identity.yaml 缺少 product 字段，补写 product: ${_ACLI_INSTALL_PRODUCT}"
    if [ "$(id -u 2>/dev/null || echo 1)" -eq 0 ]; then
        echo "product: ${_ACLI_INSTALL_PRODUCT}" >> "$identity_file"
    elif command -v sudo >/dev/null 2>&1; then
        echo "product: ${_ACLI_INSTALL_PRODUCT}" | sudo -n tee -a "$identity_file" >/dev/null 2>&1 || \
            _acli_compat_log "无权限写入 $identity_file"
    else
        _acli_compat_log "非 root 且无 sudo，无法写入 $identity_file"
    fi
}

# _acli_compat_install — 实际触发 acli 安装
# 返回 0 表示安装后 acli 命令可用；返回非 0 表示失败。
_acli_compat_install() {
    if ! command -v curl >/dev/null 2>&1; then
        _acli_compat_log "curl 不可用，跳过 acli 安装"
        return 1
    fi

    # sudo 探测：当前用户不是 root，且没有可用 sudo，无法装到 /usr/local/bin
    local sudo_cmd=""
    if [ "$(id -u 2>/dev/null || echo 1)" -eq 0 ]; then
        sudo_cmd=""
    elif command -v sudo >/dev/null 2>&1; then
        sudo_cmd="sudo"
    else
        _acli_compat_log "非 root 用户且 sudo 不可用，跳过 acli 安装"
        return 1
    fi

    _acli_compat_log "尝试安装 acli: ${_ACLI_INSTALL_URL}"

    # 使用 timeout 限制安装总时长（90s），避免阻塞业务脚本
    local install_cmd
    if [ -n "$sudo_cmd" ]; then
        install_cmd="curl -fsSL --connect-timeout 10 --max-time 60 '${_ACLI_INSTALL_URL}' | ${sudo_cmd} -n env ACLI_PRODUCT=${_ACLI_INSTALL_PRODUCT} bash"
    else
        install_cmd="curl -fsSL --connect-timeout 10 --max-time 60 '${_ACLI_INSTALL_URL}' | env ACLI_PRODUCT=${_ACLI_INSTALL_PRODUCT} bash"
    fi

    if command -v timeout >/dev/null 2>&1; then
        bash -c "timeout --kill-after=10 90 bash -c \"$install_cmd\"" >&2 2>&1 || {
            _acli_compat_log "acli 安装失败或超时（90s）"
            return 1
        }
    else
        bash -c "$install_cmd" >&2 2>&1 || {
            _acli_compat_log "acli 安装失败"
            return 1
        }
    fi

    hash -r 2>/dev/null || true

    if command -v acli >/dev/null 2>&1; then
        _acli_compat_log "acli 安装成功: $(command -v acli)"
        return 0
    fi

    _acli_compat_log "acli 安装脚本执行完成，但 acli 命令仍不可用"
    return 1
}

# ensure_acli — 主入口
#
# 行为流程：
#   1) acli 已存在 → stdout 输出 "acli"
#   2) 否则触发安装，安装成功 → "acli"；失败 → "fallback"
#
# 用法（脚本开头调用一次，后续复用变量）：
#   _acli_mode="$(ensure_acli 2>>"$LOG_FILE")"
#   if [ "$_acli_mode" = "acli" ]; then
#       acli xxx ...
#   else
#       # 原有 harness / curl / skillhub 兜底逻辑
#   fi
ensure_acli() {
    if has_acli; then
        _acli_ensure_product
        timeout 30 acli self-update >/dev/null 2>&1 || _acli_compat_log "self-update 失败或超时，继续使用当前版本"
        hash -r 2>/dev/null || true
        printf 'acli\n'
        return 0
    fi

    if _acli_compat_install; then
        printf 'acli\n'
    else
        printf 'fallback\n'
    fi
    return 0
}

# ensure_acli_light — 轻量版：仅探测+安装，跳过 self-update 和 identity 补写
# 适用场景：高频轮询（如就绪探测每 30s 一次），self-update 可能卡满 TAT 超时窗口
ensure_acli_light() {
    if has_acli; then
        printf 'acli\n'
        return 0
    fi
    if _acli_compat_install; then
        printf 'acli\n'
    else
        printf 'fallback\n'
    fi
    return 0
}
