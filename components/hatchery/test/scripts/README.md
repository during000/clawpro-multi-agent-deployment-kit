# 集成测试脚本

## 快速开始

```bash
# 1. 设置必需环境变量
export API=http://localhost:8080    # hatchery 服务地址
export ADMIN_TOKEN=<管理员Token>     # 种子管理员 Token

# 2. 可选环境变量
export TOKEN=<普通用户Token>         # 用于权限测试
export IDENTIFIER=ci-$(date +%s)    # 测试资源标识（便于清理）

# 3. 运行单个测试
cd test/scripts
python3 admin_user/test_admin_user_lifecycle.py

# 4. 运行一组测试
python3 admin_sg/test_admin_sg_create.py
python3 admin_sg/test_admin_sg_list.py

# 5. 运行全部测试
make integration-test  # 从项目根目录
```

## 环境变量

| 变量 | 必填 | 说明 |
|------|------|------|
| `API` | ✅ | hatchery 服务地址（如 `http://localhost:8080`） |
| `ADMIN_TOKEN` | ✅ | 种子管理员 Token |
| `TOKEN` | ❌ | 普通用户 Token（用于权限测试中的非管理员场景） |
| `IDENTIFIER` | ❌ | 测试资源命名标识（便于按前缀清理残留资源） |
| `MODEL_ID` | ❌ | 模型测试需要（如 `gpt-4o`） |
| `MODEL_API_KEY` | ❌ | 模型测试需要 |
| `MODEL_URL` | ❌ | 模型测试需要（如 `https://api.openai.com`） |
| `BACKUP_SG_ID` | ❌ | 安全组绑定测试需要（一个已存在的 SG ID） |

### 显示控制

| 变量 | 说明 |
|------|------|
| `QUIET=1` | 静默模式：不打印每次 HTTP 调用的帧输出，仅打印用例汇总 |
| `TRACE=1` | 调试模式：失败时打印完整 traceback + 逐帧明细 |
| `SHOW_TOKEN=1` | cURL 中显示真实 Token（默认脱敏为 `***`） |
| `RESP_MAX=0` | 响应正文不截断（默认 800 字符） |
| `NO_COLOR=1` | 禁用 ANSI 颜色输出 |

## 目录结构

```
test/scripts/
├── helpers/             # 测试辅助模块（不直接运行）
│   ├── __init__.py      # 统一导出
│   ├── api.py           # 统一 re-export 门面
│   ├── client.py        # HTTP 客户端 + 帧记录引擎
│   ├── config.py        # 环境变量配置
│   ├── testing.py       # 断言 / 运行器 / 帧摘要
│   ├── model.py         # 模型管理 helper
│   ├── instance.py      # 实例管理 helper
│   ├── channel.py       # 渠道管理 helper
│   ├── skill.py         # 技能管理 helper
│   ├── user_mgmt.py     # 用户管理 helper
│   ├── user_groups.py   # 用户组 helper
│   ├── local_agent.py   # 本地 agent (source=local) helper：reporter 冒充 + setup_local_instance
│   └── openclaw_gateway.py  # Gateway 连接 helper
├── admin_sg/            # 安全组管理测试
├── admin_user/          # 用户管理测试
├── admin_user_groups/   # 用户组测试
├── admin_usage/         # 用量统计测试
├── base/                # 基础环境测试
├── channel/             # 渠道配置测试
├── model/               # 模型配置测试
├── quota/               # 配额测试
├── skill/               # 技能测试
├── local_agent/         # 本地 agent (source=local) e2e：lifecycle / reporter / skill flow / pending / cvm-reject / availability
└── README.md            # 本文件（含两种风格的完整模板）
```

## 架构说明

### 帧记录引擎

所有 HTTP 请求都经过**帧记录引擎**（`client.py` 中的 `ApiClient._execute`），自动输出：

```
  ── [#001] POST /admin/create  → OK  200 [expect=200]  (45ms)
     cURL: curl -X POST 'http://localhost:8080/admin/create' -H 'Authorization: Bearer ***' ...
     Body (form):
           username=test-user
           password=***
     Resp:
           {"id": 42, "username": "test-user", ...}
```

特性：
- 每个请求自动编号（`#001`, `#002`, ...）
- 自动打印可复制的 cURL 命令（Token 脱敏）
- 显示请求体（JSON/form）和响应体（自动截断）
- `expect` 断言：支持 `None`（不断言）/ `int` / `tuple`（多选）

### 测试运行器帧摘要

`run_tests()` 在每个测试用例后自动输出帧摘要：

```
>>> 创建用户
  ── [#001] POST /admin/create → OK 200 ...
  ── [#002] GET /admin/users → OK 200 ...
    ┄┄┄ 帧摘要: 2 请求, 2✓, 总耗时 89ms ┄┄┄
    ✓ PASS
```

全局结果区包含 HTTP 请求统计：

```
============================================================
结果 - 用户管理全生命周期
------------------------------------------------------------
  HTTP 请求: 28 总计, 28 成功, 0 失败, 总耗时 1234ms
------------------------------------------------------------
  全部通过 ✓ (13 tests)
============================================================
```

### 预置客户端

| 客户端 | 说明 | 用途 |
|--------|------|------|
| `seed` | 种子管理员（`ADMIN_TOKEN`） | 所有管理员 API 测试 |
| `anon` | 无认证（空 Token） | 测试 401/403 |
| `bad_token` | 错误 Token | 测试 401/403 |

### OOP 调用风格

```python
from helpers.api import seed, anon, bad_token

# GET — 默认 expect=200，返回 JSON dict
data = seed.get("/admin/users", params={"page": 1})
users = data.get("users")

# POST JSON
seed.post("/admin/update-user", json={"email": "new@x.com"}, params={"id": 1})

# POST form
seed.post("/admin/create", data={"username": "test", "password": "123"})

# PUT / DELETE / PATCH
seed.put("/admin/models/update", params={"id": 1}, json={"name": "new"})
seed.delete("/admin/models/delete", params={"id": 1})

# 获取 raw Response（用于手动断言）
resp = seed.post("/admin/xxx", json=body, expect=None, raw=True)
assert resp.status_code == 400

# 权限测试
anon.post("/admin/xxx", json={}, expect=(401, 403))
bad_token.get("/admin/xxx", expect=(401, 403))
```

---

## 编写新测试

本框架支持两种测试风格，根据场景复杂度选择：

| 风格 | 适用场景 | 运行方式 | 示例 |
|------|----------|----------|------|
| **简单场景** | 单一接口的多种输入/边界测试 | `run_tests()` 自动发现并运行 `test_*` 函数 | `admin_sg/test_admin_sg_create.py` |
| **复杂场景** | 多资源联动、有前置依赖（用户/实例/模型/通道） | `main()` 中手动编排步骤 | `channel/test_qq_channel.py` |

---

### 风格一：简单场景（run_tests 自动运行）

适用于：对单个或少量接口进行正常/异常/边界测试，无需复杂前置依赖。

**特点**：
- 测试函数以 `test_` 开头，按数字排序自动执行
- `run_tests()` 自动发现、运行、统计帧摘要
- 清理逻辑在 `finally` 中确保执行

**模板**：

```python
#!/usr/bin/env python3
"""
<接口标题> 集成测试

覆盖接口：
    POST /admin/xxx/create   正常 / 参数校验 / 权限
    GET  /admin/xxx          列表查询
"""
import os
import sys
import time

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))
from helpers.api import (
    seed, anon, bad_token, ApiClient,
    health_check, run_tests, auth_test_suite,
    assert_fields, assert_status,
    cleanup_users_by_prefix,
    IDENTIFIER,
)

PREFIX = f"it-xxx-{int(time.time())}"


# ─── 工具函数 ───

def do_create(body: dict, headers=None):
    if headers:
        tmp = ApiClient("", timeout=30)
        return tmp.post("/admin/xxx/create", json=body,
                        expect=None, raw=True, extra_headers=headers)
    return seed.post("/admin/xxx/create", json=body,
                     expect=None, raw=True)


# ─── 测试用例（按编号顺序执行） ───

def test_01_auth():
    """认证三件套"""
    auth_test_suite(
        lambda headers: do_create({"name": "auth-test"}, headers=headers),
        label="xxx_create",
    )

def test_02_create_ok():
    """正常创建 → 200"""
    resp = do_create({"name": f"{PREFIX}-item"})
    assert resp.status_code == 200
    data = resp.json()
    assert "id" in data
    print(f"    OK (id={data['id']})")

def test_03_create_validation():
    """参数校验：name 为空 → 400"""
    resp = do_create({"name": ""})
    assert resp.status_code == 400

def test_04_list():
    """查询列表"""
    data = seed.get("/admin/xxx")
    assert isinstance(data.get("items"), list)
    print(f"    count={len(data['items'])}")


# ─── 清理 ───

def cleanup():
    try:
        cleanup_users_by_prefix(PREFIX)
    except Exception as e:
        print(f"[cleanup] {e}")


# ─── 入口 ───

def main():
    health_check()
    try:
        run_tests(globals(), title="<接口标题>", ordered=True, abort_on_fail=True)
    finally:
        cleanup()

if __name__ == "__main__":
    main()
```

---

### 风格二：复杂场景（main 手动编排）

适用于：需要多步前置依赖（创建用户 → 创建实例 → 绑定模型 → 配置通道）的端到端测试。

**特点**：
- `main()` 中手动编排测试步骤，按业务流程顺序执行
- 使用 `helpers` 中的 `setup_*` / `teardown_*` 函数创建依赖资源
- `try/finally` 确保资源清理
- 步骤间可传递上下文（用户 token、实例 ID 等）

**模板**：

```python
#!/usr/bin/env python3
"""
TC-X.X <场景标题>

使用方式：
  export API=http://localhost:8080
  export ADMIN_TOKEN=xxx
  export MODEL_ID=gpt-4o  MODEL_API_KEY=xxx  MODEL_URL=https://api.openai.com
  python3 test_xxx.py
"""
import os
import sys
import time
import traceback

sys.path.insert(0, os.path.join(os.path.dirname(__file__), ".."))

from helpers import config
import helpers
from helpers import (
    check_env, require_model_config,
    setup_admin, setup_user, setup_instance,
    setup_model, teardown_model,
    retry_on_gateway_restart,
)


def main():
    # ── 环境检查 ──
    check_env()
    require_model_config()
    print()

    # ── 创建依赖资源 ──
    admin = setup_admin("test-prefix")
    user = None
    inst = None
    model_ctx = None

    try:
        helpers.ensure_gateway_ui_enabled(admin.token)

        # Step 1: 创建模型
        model_ctx = setup_model(
            admin.token,
            model_id=config.MODEL_ID,
            model_name=f"IntTest ({config.MODEL_ID})",
            api_key=config.MODEL_API_KEY,
            url=config.MODEL_URL,
        )

        # Step 2: 创建用户 + 实例
        user = setup_user(admin.token, "test-prefix")
        inst = setup_instance(user.token, "test-prefix")

        # Step 3: 绑定模型到实例
        print(">>> 步骤 1：绑定模型 ...")
        resp = retry_on_gateway_restart(
            lambda: helpers.user_add_model(user.token, inst.db_id, model_ctx.db_id)
        )
        assert resp.status_code == 200, f"绑定失败: {resp.status_code} {resp.text}"
        print("    绑定成功 ✓")
        time.sleep(3)

        # Step 4: 核心业务逻辑测试
        print(">>> 步骤 2：执行核心操作 ...")
        # ... 实际测试逻辑 ...
        print("    操作成功 ✓")

        # Step 5: 验证结果
        print(">>> 步骤 3：验证结果 ...")
        # ... 断言验证 ...
        print("    验证通过 ✓")

        # Step 6: 清理/删除操作
        print(">>> 步骤 4：清理配置 ...")
        # ... 删除测试资源 ...
        print("    清理完成 ✓")

        print()
        print("TC-X.X 测试通过 ✅")

    except Exception as e:
        print(f"\nTC-X.X 测试失败 ❌: {e}")
        traceback.print_exc()
        sys.exit(1)

    finally:
        # 确保模型等全局资源被清理
        if model_ctx:
            teardown_model(admin.token, model_ctx)


if __name__ == "__main__":
    main()
```

---

### 如何选择风格

```
需要创建 用户/实例/模型 等前置依赖？
  ├─ 是 → 复杂场景（风格二）
  └─ 否 → 简单场景（风格一）
```

**简单场景**的判断标准：
- 只需要 `seed`（管理员客户端）就能调通全部接口
- 不需要创建临时用户、实例、模型等资源
- 每个 `test_*` 函数相对独立（或只有简单的顺序依赖）

**复杂场景**的判断标准：
- 测试需要先创建用户 → 创建实例 → 绑定模型等多步前置
- 步骤间共享状态（user token、instance ID）
- 端到端验证（如通道配置 + WebSocket 验证 + 消息推送）

---

### 最佳实践

- **命名规范**：测试函数以 `test_` 开头，按数字排序（`test_01_xxx`, `test_02_xxx`）
- **有序执行**：`run_tests(globals(), ordered=True, abort_on_fail=True)` 确保依赖顺序
- **认证测试**：使用 `auth_test_suite` 统一测试无认证/错误 token/非管理员场景
- **资源清理**：清理函数放在 `try/finally` 中确保执行
- **资源命名**：用 `PREFIX` 或 `IDENTIFIER` 命名测试资源，便于按前缀清理
- **make_api_fn**：对于简单的 API 调用，用 `make_api_fn` 工厂减少样板代码
- **环境隔离**：复杂场景用 `setup_admin("prefix")` 创建独立管理员，避免污染全局状态
