# 集成测试框架

本目录包含 hatchery 的端到端集成测试。测试通过 K8s 部署真实的 hatchery 实例，然后运行 Python 测试脚本对 API 进行验证。

## 前置准备

在运行集成测试之前，需要准备以下内容：

### 1. 腾讯云 AK/SK

测试过程中 hatchery 需要调用腾讯云 API（创建 CVM、管理安全组、操作 VPC 等），因此需要一组有效的3205597606账号的凭证：
- **SecretId**（AK）和 **SecretKey**（SK）
- 运行时通过 `--ak` / `--sk` 参数传入

必须要用3205597606账号的原因是ClawPro的多项功能需要白名单的方式才能调用云API,其他账号没有添加白名单。

### 2. Kubeconfig

测试编排器需要连接 K8s 集群来部署测试实例：
- 默认读取 `~/.kube/config` 或 `$KUBECONFIG` 环境变量指向的文件


可在控制台 TKE（腾讯云容器服务），下载集群的 kubeconfig 文件: https://console.cloud.tencent.com/tke2/cluster/sub/list/basic/info/apiserver?rid=1&clusterId=cls-m29jw7bu

必须使用k8s集群部署的原因是CVM 实例需要回调 hatchery, 所以需要一个可以访问的公网ip，这是llm代理可以工作的前提。

### 3. 容器镜像仓库权限

测试需要将 hatchery 镜像推送到镜像仓库，因此需要本地登陆镜像仓库：
可以参考 https://console.cloud.tencent.com/tcr/repository/tcr/tcr-4531z75r/cvm-openclaw/hatchery/1/tagList `仓库信息` tab页面操作

### 4.（可选）模型和通道凭证

部分测试用例需要真实的 AI 模型和通道凭证。如果不提供，相关测试会失败：

- **AI 模型**：`MODEL_ID`、`MODEL_API_KEY`、`MODEL_URL`、`MODEL_TYPE`
- **飞书**：`FEISHU_APP_ID`、`FEISHU_APP_SECRET`
- **企微**：`WECOM_BOT_ID`、`WECOM_SECRET`
- **钉钉**：`DDINGTALK_CLIENT_ID`、`DDINGTALK_CLIENT_SECRET`
- **QQ 机器人**：`QQBOT_APP_ID`、`QQBOT_APP_SECRET`

这些变量可通过环境变量注入:
```
export MODEL_ID="test/abc"
```

## 依赖

### 本机环境

| 依赖 | 版本要求 | 说明 |
|------|----------|------|
| Go | >= 1.22 | 编译测试编排器 |
| Python 3 | >= 3.8 | 运行测试脚本 |
| docker | - | 用于构建并推送测试镜像 |
| kubectl | - | 需要能连接目标集群（用于诊断，非必须） |

### Python 依赖

测试脚本依赖以下 Python 包：

| 包 | 用途 |
|----|------|
| `requests` | HTTP 客户端，所有 API 调用 |
| `websocket-client` | WebSocket 连接测试（仅 gateway 相关用例） |

安装：

```bash
pip3 install requests websocket-client
```

## 使用方式

### 构建并推送镜像

测试需要先配置并登陆好镜像仓库。在运行测试前，先构建并推送：

```bash
# 构建镜像
docker build -t your-registry/hatchery:dev .

# 推送到仓库（确保已经docker login完成）
docker push your-registry/hatchery:dev
```

### 快速开始

```bash
# 从项目根目录运行
make test IMAGE=your-registry/hatchery:dev
```

### 完整参数

```bash
make test IMAGE=your-registry/hatchery:tag TEST_ARGS="<额外参数>"
```

| Makefile 变量 | 说明 |
|---------------|------|
| `IMAGE` | **(必填)** 要部署的 hatchery 容器镜像 |
| `TEST_ARGS` | 传递给测试编排器的额外命令行参数 |
| `BASE_BRANCH` | （可选）基线分支，用于增量覆盖率对比，如 `origin/master` |

### 编排器命令行参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--image` | manifest 中的镜像 | 覆盖容器镜像地址 |
| `--ak` | 无 | 腾讯云 SecretId，直接注入容器环境变量 |
| `--sk` | 无 | 腾讯云 SecretKey，直接注入容器环境变量 |
| `--kubeconfig` | `~/.kube/config` | kubeconfig 文件路径 |
| `--namespace` | `clawpro-test` | K8s namespace |
| `--manifest` | 自动搜索 `test/hatchery-statefulset.yaml` | K8s 部署清单路径 |
| `--timeout` | `1h` | 全局超时时间 |
| `--concurrency` | `3` | 最大并发测试脚本数 |
| `--script-timeout` | `15m` | 单个脚本超时时间 |
| `--report-dir` | 不生成 | HTML 报告输出目录（同时启用 API 覆盖率收集） |
| `--no-cleanup` | `false` | 测试完成后不清理 K8s 资源（调试用） |
| `--run` | 无（运行全部） | 过滤测试脚本：目录名、文件名或 glob 模式 |
| `--cls-topic-id` | 无（不上报） | 腾讯云 CLS 日志主题 ID，开启后测试结果将上报到此主题（缺失则不上报） |
| `--cls-region` | `ap-guangzhou` | 腾讯云 CLS 所在地域 |
| `--cls-ak` | 无 | 上报 CLS 专用的腾讯云 SecretId（与应用自身的 `--ak`/`--sk` 解耦） |
| `--cls-sk` | 无 | 上报 CLS 专用的腾讯云 SecretKey（与应用自身的 `--ak`/`--sk` 解耦） |

### 常见用法示例

```bash
# 基本运行
make test IMAGE=my-registry/hatchery:dev

# 直接传入腾讯云凭证（无需集群中预置 Secret）
make test IMAGE=my-registry/hatchery:dev TEST_ARGS="--ak AKIDxxxxx --sk xxxxx"

# 只跑某个目录下的测试
make test IMAGE=my-registry/hatchery:dev TEST_ARGS="--run admin_user"

# 只跑某个文件
make test IMAGE=my-registry/hatchery:dev TEST_ARGS="--run quota/test_quota_logs.py"

# 用 glob 模式匹配文件名
make test IMAGE=my-registry/hatchery:dev TEST_ARGS="--run 'test_admin_sg_*'"

# 生成 HTML 报告（含覆盖率）
make test IMAGE=my-registry/hatchery:dev TEST_ARGS="--report-dir ./test-report"

# 生成 HTML 报告 + 增量覆盖率（对比 master）
make test IMAGE=my-registry/hatchery:dev BASE_BRANCH=origin/master TEST_ARGS="--report-dir ./test-report"
```

### CLS 测试结果上报（可选）

测试结果（每条用例 + 流水线级汇总）可以上报到腾讯云 CLS，便于在 CLS 控制台做统一检索与看板。

该能力为 best-effort：仅当 `--cls-topic-id`、`--cls-ak`、`--cls-sk` 三个参数**全部**提供时才启用，
任一缺失则静默跳过（不影响测试主流程）。`--cls-region` 缺省为 `ap-guangzhou`。

上报数据包含两类日志：

- **用例级**（无 `stage` 字段）：`run_id`、`script`、`status`(pass/fail)、`duration_ms`、`error`、`output`
- **流水线级**（`stage=pipeline`）：`run_id`、`status`(success/fail)、`total`、`passed`、`failed`、`pass_rate`、`duration_ms`

CLS 上报采用独立凭证（`--cls-ak`/`--cls-sk`），与应用自身部署使用的 `--ak`/`--sk` 完全解耦，互不干扰。

启用示例：

```bash
make test IMAGE=my-registry/hatchery:dev \
  TEST_ARGS="--report-dir ./test-report \
             --cls-topic-id your-topic-id \
             --cls-region ap-guangzhou \
             --cls-ak <CLS_SECRET_ID> \
             --cls-sk <CLS_SECRET_KEY>"
```

## 执行流程

测试编排器按以下顺序执行：

```
1. 生成 OpenAPI spec（make openapi → docs/openapi.json）
2. 连接 K8s 集群
3. 加载并修改 manifest（随机化 label、注入 admin-token 和 identifier）
4. 部署 Service + StatefulSet 到 K8s
5. 等待 Pod 就绪
6. 等待 LoadBalancer 分配外部 IP
7. 注入 domain 参数并重启 Pod
8. 健康检查通过
9. 运行 init.py（配置 CVM 模板、安全组、VPC、创建测试用户）
10. 发现并并发执行 test/scripts/ 下的 test_*.py 脚本
11. 检查 Pod 是否发生异常重启
12. 收集 API 覆盖率数据（合并各脚本的帧记录）
13. 生成覆盖率 HTML 报告（coverage.html）
14. 运行 cleanup.py（销毁云资源）
15. 删除 K8s 资源（Service + StatefulSet）
16. 输出测试结果 / 生成报告（index.html）
```

每次测试运行会生成随机后缀（如 `a3f2b1c0`），确保：
- K8s 资源名称唯一（支持多人/多次并发测试）
- 数据通过 `--identifier` 实现租户隔离

## 环境变量（测试脚本）

以下环境变量由编排器自动注入，无需手动设置：

| 变量 | 说明 |
|------|------|
| `API` | hatchery API 地址（如 `http://1.2.3.4/`） |
| `TOKEN` | 普通测试用户的 API Token |
| `ADMIN_TOKEN` | 管理员 API Token |
| `IDENTIFIER` | 本次测试的唯一标识符 |

以下变量通过环境变量注入（特定测试需要）：

| 变量 | 需要的测试 | 说明 |
|------|------------|------|
| `MODEL_ID` | model/、channel/、skill/、base/ | AI 模型 ID |
| `MODEL_API_KEY` | 同上 | 模型 API Key |
| `MODEL_URL` | 同上 | 模型 API 地址 |
| `MODEL_TYPE` | 同上 | 模型类型（`openai-completions` / `anthropic-messages`） |
| `FEISHU_APP_ID` / `FEISHU_APP_SECRET` | channel/ | 飞书应用凭证 |
| `WECOM_BOT_ID` / `WECOM_SECRET` | channel/ | 企微机器人凭证 |
| `QQBOT_APP_ID` / `QQBOT_APP_SECRET` | channel/ | QQ 机器人凭证 |
| `DDINGTALK_CLIENT_ID` / `DDINGTALK_CLIENT_SECRET` | channel/ | 钉钉凭证 |

## 编写测试脚本

参见 [test/scripts/README.md](scripts/README.md)，包含详细的模板和最佳实践。

关键约定：
- 文件名以 `test_` 开头、`.py` 结尾
- 放在 `test/scripts/` 的子目录中
- 通过 `sys.path.insert` 引入 `helpers` 模块
- 使用环境变量 `API`、`TOKEN`、`ADMIN_TOKEN` 连接服务

## API 覆盖率

当指定 `--report-dir` 时，测试框架自动收集 API 覆盖率数据并生成 HTML 报告。

### 工作原理

1. **数据收集**：每个 Python 测试脚本的 HTTP 调用（通过 `helpers/client.py` 的 `ApiClient`）会自动记录帧数据（method、path、query params、body params、status code）
2. **数据合并**：测试结束后编排器合并所有脚本的帧数据为 `coverage-data.json`
3. **报告生成**：调用 `test/api_coverage.py` 将帧数据与 OpenAPI spec（`docs/openapi.json`）比对，生成 `coverage.html`

### 输出文件

| 文件 | 说明 |
|------|------|
| `<report-dir>/index.html` | 测试结果报告（顶部有覆盖率链接） |
| `<report-dir>/coverage.html` | API 覆盖率报告 |
| `<report-dir>/coverage-data.json` | 覆盖率原始数据 |

### 覆盖率报告内容

- **路由覆盖率**：OpenAPI spec 中定义的接口被调用了多少
- **参数覆盖率**：每个接口文档定义的参数被传入了多少
- **状态码覆盖**：每个接口返回了哪些 HTTP 状态码（判断是否做了边界测试）
- **未覆盖清单**：从未被调用的接口列表
- **缺失参数**：已覆盖接口中未被测试传入的参数
- **增量覆盖率**（需 `BASE_BRANCH` 或 CI 环境）：本次 PR 新增的接口/参数是否被集成测试覆盖

### 工具脚本

| 脚本 | 用途 |
|------|------|
| `test/api_md_to_openapi.py` | 从 `docs/API.md` 生成 `docs/openapi.json`（OpenAPI 3.0 spec） |
| `test/api_coverage.py` | 覆盖率分析，支持 `--html`/`--json`/文本输出 |

手动使用：

```bash
# 生成 OpenAPI spec
make openapi

# 生成 OpenAPI spec + base spec（增量对比）
make openapi BASE_BRANCH=origin/master

# 手动分析覆盖率数据
python3 test/api_coverage.py --spec docs/openapi.json --data test-report/coverage-data.json

# 生成 HTML 报告（含增量）
python3 test/api_coverage.py --spec docs/openapi.json --base-spec docs/openapi_base.json --data test-report/coverage-data.json --html coverage.html
```
