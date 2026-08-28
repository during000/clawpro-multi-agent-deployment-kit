#!/usr/bin/env bash
# CI 脚本：验证 基线 init.sql + 新增迁移 SQL = 当前 init.sql
# 用法：BASE_BRANCH=main bash scripts/ci-check-schema.sh
# 需要：docker, git

set -euo pipefail

BASE_BRANCH="${BASE_BRANCH:-master}"
CONTAINER_NAME="hatchery-schema-check-${RANDOM}"
MYSQL_ROOT_PASS="ciRootPass${RANDOM}"
DB_MIGRATED="db_migrated"
DB_FRESH="db_fresh"

cleanup() {
    echo "==> Cleaning up..."
    docker rm -f "$CONTAINER_NAME" 2>/dev/null || true
}
trap cleanup EXIT

# 封装 mysql/mysqldump 命令，仅屏蔽密码警告，保留其他错误输出
run_mysql() {
    docker exec -i "$CONTAINER_NAME" mysql -uroot -p"$MYSQL_ROOT_PASS" \
        --default-character-set=utf8mb4 "$@" 2> >(grep -v '\[Warning\].*Using a password' >&2)
}

run_mysqldump() {
    docker exec "$CONTAINER_NAME" mysqldump -uroot -p"$MYSQL_ROOT_PASS" \
        --no-data --skip-comments --skip-add-drop-table --skip-set-charset --compact \
        "$@" 2> >(grep -v '\[Warning\].*Using a password' >&2)
}

# ---------------------------------------------------------------------------
# 0. 检查前置条件
# ---------------------------------------------------------------------------
if ! command -v docker &>/dev/null; then
    echo "ERROR: docker is required but not found" >&2
    exit 1
fi

# ---------------------------------------------------------------------------
# 1. 判断是否有 schema 变更
# ---------------------------------------------------------------------------
echo "==> Checking for schema changes against ${BASE_BRANCH}..."

# 确保有 base 分支的引用（CI 环境中可能需要 fetch）
git fetch origin "$BASE_BRANCH" --quiet 2>/dev/null || true
BASE_REF="origin/${BASE_BRANCH}"

INIT_CHANGED=false
if ! git diff --quiet "$BASE_REF"...HEAD -- sql/init.sql 2>/dev/null; then
    INIT_CHANGED=true
fi

MIGRATION_FILES=$(git diff --name-only --diff-filter=A "$BASE_REF"...HEAD -- 'sql/*.sql' | grep -v 'sql/init.sql' | sort || true)
DELETED_MIGRATION_FILES=$(git diff --name-only --diff-filter=D "$BASE_REF"...HEAD -- 'sql/*.sql' | grep -v 'sql/init.sql' | sort || true)

if [ "$INIT_CHANGED" = false ] && [ -z "$MIGRATION_FILES" ] && [ -z "$DELETED_MIGRATION_FILES" ]; then
    echo "==> No schema changes detected. PASS"
    exit 0
fi

if [ "$INIT_CHANGED" = true ] && [ -z "$MIGRATION_FILES" ] && [ -z "$DELETED_MIGRATION_FILES" ]; then
    echo "ERROR: sql/init.sql changed but no new migration file added." >&2
    echo "Every schema change needs both init.sql update AND a migration SQL file." >&2
    exit 1
fi

if [ "$INIT_CHANGED" = true ] && [ -z "$MIGRATION_FILES" ] && [ -n "$DELETED_MIGRATION_FILES" ]; then
    echo "  Revert detected: init.sql changed and migration file(s) deleted:"
    echo "$DELETED_MIGRATION_FILES" | sed 's/^/    /'
    echo "==> Schema revert. PASS"
    exit 0
fi

if [ "$INIT_CHANGED" = false ] && [ -n "$MIGRATION_FILES" ]; then
    echo "ERROR: New migration file(s) added but sql/init.sql not updated:" >&2
    echo "$MIGRATION_FILES" >&2
    exit 1
fi

if [ "$INIT_CHANGED" = false ] && [ -n "$DELETED_MIGRATION_FILES" ]; then
    echo "ERROR: Migration file(s) deleted but sql/init.sql not updated:" >&2
    echo "$DELETED_MIGRATION_FILES" >&2
    exit 1
fi

echo "  init.sql changed: $INIT_CHANGED"
echo "  New migration files:"
echo "$MIGRATION_FILES" | sed 's/^/    /'

# ---------------------------------------------------------------------------
# 1.5 如果目标分支是 Release/YYYY_MM_DD 格式，要求新增迁移文件以 MMDD- 开头
# ---------------------------------------------------------------------------
if [[ "$BASE_BRANCH" =~ ^Release/[0-9]{4}_([0-9]{2})_([0-9]{2})$ ]] && [ -n "$MIGRATION_FILES" ]; then
    REQUIRED_PREFIX="${BASH_REMATCH[1]}${BASH_REMATCH[2]}-"
    echo "==> Release branch detected, requiring migration files to start with '${REQUIRED_PREFIX}'..."
    BAD_FILES=""
    for f in $MIGRATION_FILES; do
        fname=$(basename "$f")
        if [[ ! "$fname" =~ ^${REQUIRED_PREFIX} ]]; then
            BAD_FILES="${BAD_FILES}    ${f} (expected prefix: ${REQUIRED_PREFIX})\n"
        fi
    done
    if [ -n "$BAD_FILES" ]; then
        echo "ERROR: Migration file(s) do not match Release branch date prefix:" >&2
        echo -e "$BAD_FILES" >&2
        exit 1
    fi
fi

# ---------------------------------------------------------------------------
# 2. 启动 MySQL 8.0 容器
# ---------------------------------------------------------------------------
echo "==> Starting MySQL 8.0 container ($CONTAINER_NAME)..."
docker run -d \
    --name "$CONTAINER_NAME" \
    -e MYSQL_ROOT_PASSWORD="$MYSQL_ROOT_PASS" \
    mysql:8.0 \
    --default-authentication-plugin=mysql_native_password \
    --character-set-server=utf8mb4 \
    --collation-server=utf8mb4_unicode_ci \
    >/dev/null

echo "==> Waiting for MySQL to be ready..."
for i in $(seq 1 90); do
    if docker exec "$CONTAINER_NAME" mysql -uroot -p"$MYSQL_ROOT_PASS" -e "SELECT 1" &>/dev/null; then
        echo "    MySQL ready after ${i}s"
        break
    fi
    if [ "$i" -eq 90 ]; then
        echo "ERROR: MySQL did not become ready in 90s" >&2
        exit 1
    fi
    sleep 1
done

# ---------------------------------------------------------------------------
# 3. 创建两个数据库
# ---------------------------------------------------------------------------
echo "==> Creating databases..."
run_mysql -e "CREATE DATABASE $DB_MIGRATED CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
run_mysql -e "CREATE DATABASE $DB_FRESH CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

# ---------------------------------------------------------------------------
# 4. 导入基线 init.sql 到 db_migrated
# ---------------------------------------------------------------------------
echo "==> Importing baseline init.sql into $DB_MIGRATED..."
git show "$BASE_REF":sql/init.sql | run_mysql "$DB_MIGRATED"

# ---------------------------------------------------------------------------
# 5. 按顺序执行新增迁移文件到 db_migrated
# ---------------------------------------------------------------------------
echo "==> Applying migration files to $DB_MIGRATED..."
for f in $MIGRATION_FILES; do
    echo "    Applying: $f"
    git show "HEAD:$f" | run_mysql "$DB_MIGRATED"
done

# ---------------------------------------------------------------------------
# 6. 导入当前 init.sql 到 db_fresh
# ---------------------------------------------------------------------------
echo "==> Importing current init.sql into $DB_FRESH..."
git show HEAD:sql/init.sql | run_mysql "$DB_FRESH"

# ---------------------------------------------------------------------------
# 7. 导出 schema 并规范化
# ---------------------------------------------------------------------------
echo "==> Dumping and comparing schemas..."

normalize_schema() {
    sed -E \
        -e 's/ AUTO_INCREMENT=[0-9]+//' \
        -e 's/ CHARACTER SET [a-zA-Z0-9_]+//' \
        -e '/^\/\*!/d' \
        -e '/^--/d' \
        -e '/^$/d' \
    | awk '
    # Per-table normalization: keep CREATE TABLE structure intact,
    # sort column/index lines within each table (after stripping
    # trailing commas) so that definition order differences do not
    # produce false diffs.
    /^CREATE TABLE/ {
        in_table = 1
        header = $0
        # extract table name for annotation
        tbl = header
        sub(/^CREATE TABLE /, "", tbl)
        sub(/ \(.*/, "", tbl)
        delete lines
        n = 0
        next
    }
    in_table && /^\)/ {
        # end of CREATE TABLE — emit header, sorted body, closing line with table name
        in_table = 0
        print header
        # sort collected lines
        for (i = 0; i < n; i++)
            for (j = i + 1; j < n; j++)
                if (lines[i] > lines[j]) {
                    tmp = lines[i]; lines[i] = lines[j]; lines[j] = tmp
                }
        for (i = 0; i < n; i++)
            print lines[i]
        print $0 " -- " tbl
        next
    }
    in_table {
        # strip trailing comma so sort is not affected by position
        line = $0
        sub(/,$/, "", line)
        lines[n++] = line
        next
    }
    { print }
    '
}

SCHEMA_MIGRATED=$(run_mysqldump "$DB_MIGRATED" | normalize_schema)
SCHEMA_FRESH=$(run_mysqldump "$DB_FRESH" | normalize_schema)

# ---------------------------------------------------------------------------
# 8. 对比结果
# ---------------------------------------------------------------------------
DIFF_RESULT=$(diff -u <(echo "$SCHEMA_MIGRATED") <(echo "$SCHEMA_FRESH") || true)

if [ -z "$DIFF_RESULT" ]; then
    echo ""
    echo "==> Schema check PASSED!"
    echo "    baseline init.sql + migration files = current init.sql"
    exit 0
else
    echo "" >&2
    echo "==> Schema check FAILED!" >&2
    echo "    baseline init.sql + migration files != current init.sql" >&2
    echo "" >&2
    echo "Diff (- = migrated, + = fresh init.sql):" >&2
    echo "$DIFF_RESULT" >&2
    exit 1
fi
