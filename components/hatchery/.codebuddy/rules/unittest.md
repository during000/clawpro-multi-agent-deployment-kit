---
type: always
---
## hatchery 单元测试规范

### 测试流程（先描述后实现）

1. **Plan 阶段**：用自然语言描述测试用例（场景、输入、预期输出）
2. **Implement 阶段**：根据自然语言描述编写测试代码
3. **UT 阶段**：运行测试、记录覆盖率

### 自然语言用例格式

在 `02-plan.md` 的测试用例章节中，按如下格式描述：

```
### 测试用例

| # | 场景 | 输入 | 预期输出 | 级别 |
|---|------|------|---------|------|
| 1 | 正常创建用户 | name="test", password="123456" | 返回 200, 用户已创建 | P0 |
| 2 | 用户名重复 | name="admin"（已存在） | 返回 409, "用户已存在" | P0 |
| 3 | 密码为空 | name="test", password="" | 返回 400, "密码不能为空" | P1 |
```

- P0：核心路径，必须覆盖
- P1：边界条件，应当覆盖
- P2：异常路径，建议覆盖

### 测试文件组织

```
model/
  user.go
  user_test.go          # model 层单元测试
controller/
  auth.go
  auth_test.go          # controller 层单元测试
main_test.go            # 端到端 handler 测试
test/
  integration/          # 集成测试（需要启动服务）
```

### 运行命令

```bash
# 全量测试（带竞态检测）
go test ./... -v -race

# 单个包测试
go test ./model/ -v -race

# 带覆盖率
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# 运行特定测试
go test ./controller/ -run TestHandleLogin -v
```

### 编写规范

详见 `docs/testing.md`（DB 初始化、异步任务处理、句柄释放），以下为强制要求：

- 【必须】测试文件名 `*_test.go`，与被测文件同目录
- 【必须】测试函数名 `Test<FunctionName>_<Scenario>`，如 `TestCreateUser_DuplicateName`
- 【推荐】使用 table-driven tests 组织多场景
- 【推荐】使用 `t.Run()` 做子测试
- 【必须】测试 DB 通过 `model.UseDBForTest(db)` 注入，用 `t.Cleanup()` 自动归还
- 【必须】不操作 `hatchery.db`，使用 `:memory:` 或 `test.db`
- 【必须】不依赖测试执行顺序
- 【必须】涉及 `distlock` 的场景使用 `model.UseDBForTestWithDriver(db, "sqlite")`
- 【必须】异步任务测试要么 mock 异步发起函数，要么用 WaitGroup 等待完成

### Table-Driven Test 示例

```go
func TestParseToken(t *testing.T) {
    tests := []struct {
        name    string
        token   string
        wantErr bool
    }{
        {"valid token", "abc123", false},
        {"empty token", "", true},
        {"expired token", "expired_xxx", true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := parseToken(tt.token)
            if (err != nil) != tt.wantErr {
                t.Errorf("parseToken() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### 覆盖率要求

| 维度 | 要求 |
|------|------|
| 新增代码覆盖率 | >= 80%（新增/修改的函数） |
| P0 用例通过率 | 100% |
| 整体覆盖率 | 不强制阈值，但不应因新代码下降 |

- 【必须】新增的 handler / model 方法必须有测试覆盖核心路径
- 【推荐】复杂业务逻辑（条件分支 >= 3）覆盖率建议 >= 90%
- 【必须】增量覆盖率通过 CI 脚本校验：`BASE_BRANCH=origin/master bash .ci/ci-check-coverage.sh`（默认增量阈值 60%）

### 集成测试

集成测试通过 K8s 部署真实 hatchery 实例，Python 脚本端到端验证 API。

```bash
# 运行全量
make test IMAGE=your-registry/hatchery:dev TEST_ARGS="--ak AKIDxxx --sk xxx"

# 只跑某个模块
make test IMAGE=your-registry/hatchery:dev TEST_ARGS="--run admin_user"

# 生成 API 覆盖率报告（含增量）
make test IMAGE=your-registry/hatchery:dev BASE_BRANCH=origin/master TEST_ARGS="--report-dir ./test-report"
```

### API 增量覆盖率要求

- 【必须】新增 API 接口必须有集成测试用例覆盖（出现在 `route_hits` 中）
- 【必须】新增请求参数必须在集成测试中被传入覆盖（出现在 `param_hits` 中）
- 通过增量覆盖率报告验证，报告中 `new_ops_uncovered` 和未覆盖的 `new_params` 应为空

**获取增量覆盖率报告**：

```bash
# 1. 生成 OpenAPI spec（当前分支 + 基线）
make openapi BASE_BRANCH=origin/master

# 2. 运行集成测试
make test IMAGE=<镜像> BASE_BRANCH=origin/master TEST_ARGS="--report-dir ./test-report"

# 3. 查看报告
# test-report/coverage.html 中「Incremental Coverage」章节列出：
#   - 新增接口是否被调用
#   - 新增参数是否被传入
```

**工作原理**：`test/api_coverage.py --base-spec docs/openapi_base.json` 对比当前 spec 与基线 spec，找出新增的 operation 和 param，再与实际测试帧数据交叉比对。

详见 `test/README.md`。
