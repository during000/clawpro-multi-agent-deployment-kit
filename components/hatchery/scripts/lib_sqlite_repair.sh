#!/bin/bash
#
# lib_sqlite_repair.sh
#
# 无损优先的 SQLite 修复公共库，供 backup_pre_reinstall.sh / openclaw_recovery.sh 复用，
# 避免两处各写一份修复逻辑导致维护漂移。
#
# 依赖：sqlite3 命令。
# 约束：本库只“尝试无损修复”，绝不删除数据库文件——
#       “删库重建空库”这种会丢数据的兜底策略由调用方（recovery）自行决定，backup 严禁删库。
# 输出：人类可读进度一律走 stderr，避免污染调用方 stdout 的机器契约
#       （如 backup 的 BACKUP_DIR_PATH: / ARCHIVE_SIZE:）。

# sqlite_integrity_ok <db>
#   integrity_check 通过返回 0，否则非 0。
sqlite_integrity_ok() {
    local db="$1"
    [ -f "$db" ] || return 1
    [ "$(sqlite3 "$db" 'PRAGMA integrity_check;' 2>/dev/null | head -1)" = "ok" ]
}

# sqlite_lossless_repair <db>
#   无损优先修复单个库。修复成功时就地替换主库并清理 -wal/-shm。
#   返回码：
#     0  库最终 integrity ok（本来就好 / WAL 收敛 / .recover / .dump 成功）
#     1  无损与部分恢复均失败（真不可恢复）——主库保持原样不动（不删除），交调用方决策
#     2  无法判定（sqlite3 缺失）——调用方应按“跳过、由下游兜底”处理，不可当作不可恢复
sqlite_lossless_repair() {
    local db="$1"
    if [ ! -f "$db" ]; then
        echo "  [repair] 库文件不存在，视为无需修复: $db" >&2
        return 0
    fi
    if ! command -v sqlite3 >/dev/null 2>&1; then
        echo "  [repair] sqlite3 缺失，无法修复: $db" >&2
        return 2
    fi

    # (1) WAL 收敛后复检——不少 malformed 仅因 -wal 与主库不一致，checkpoint 即可救活（数据零丢失）
    sqlite3 "$db" "PRAGMA wal_checkpoint(TRUNCATE);" >/dev/null 2>&1 || true
    if sqlite_integrity_ok "$db"; then
        echo "  [repair] OK integrity ok（WAL 收敛后无需重建）: $db" >&2
        return 0
    fi

    echo "  [repair] integrity 异常，尝试无损重建: $db" >&2
    local tmp="${db}.repair.$$"

    # (2) .recover 无损重建——尽最大努力保留全部可读数据
    rm -f "$tmp" 2>/dev/null || true
    if sqlite3 "$db" ".recover" 2>/dev/null | sqlite3 "$tmp" 2>/dev/null \
       && sqlite_integrity_ok "$tmp"; then
        rm -f "${db}-wal" "${db}-shm" 2>/dev/null || true
        mv -f "$tmp" "$db"
        echo "  [repair] OK .recover 无损重建成功: $db" >&2
        return 0
    fi

    # (3) .dump 部分恢复——抢救损坏点之前的数据
    rm -f "$tmp" 2>/dev/null || true
    if sqlite3 "$db" ".dump" 2>/dev/null | sqlite3 "$tmp" 2>/dev/null \
       && sqlite_integrity_ok "$tmp"; then
        rm -f "${db}-wal" "${db}-shm" 2>/dev/null || true
        mv -f "$tmp" "$db"
        echo "  [repair] OK .dump 部分恢复成功（可能丢失损坏点之后的数据）: $db" >&2
        return 0
    fi

    rm -f "$tmp" 2>/dev/null || true
    echo "  [repair] FAIL 无损/部分恢复均失败（真不可恢复）: $db" >&2
    return 1
}
