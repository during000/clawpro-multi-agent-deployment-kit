# SkillHub 安全扫描接入说明

> 面向产品经理 / 前端同学，帮助理解两家安全扫描的送审内容、审查结果字段含义，以及前端展示建议。
>
> 最后更新：2026-03-30

---

## 一、总体说明

SkillHub 目前接入了**两家安全扫描服务**，对用户发布的 Skill 包（ZIP 文件）进行安全检测。两家扫描**并行运行、独立判定**，最终聚合结果决定 Skill 是否自动通过审核。

| 扫描服务 | 供应商标识（provider） | 说明 |
|----------|----------------------|------|
| **科恩 XTI** | `keen-xti` | 腾讯科恩实验室的安全扫描平台，支持静态分析 + 动态分析，可识别木马、病毒、恶意代码等 |
| **三部 AI 扫描** | `tencent-sanbu` | 腾讯三部 AI 安全扫描平台，基于 AI 引擎进行规则匹配和风险评估 |

### 审核流程简述

```
用户发布 Skill
    ↓
同时创建两条扫描记录（keen-xti + tencent-sanbu）
    ↓
两个 Worker 分别提交 ZIP 包到各自平台
    ↓
轮询查询扫描结果
    ↓
两家都通过 → 安全扫描聚合状态 = passed → 触发自动审批
任一家不通过 → 安全扫描聚合状态 = failed → 不自动审批
```

---

## 二、送审内容

### 2.1 送审的是什么？

两家安全扫描送审的都是**用户发布 Skill 时上传的完整 ZIP 包**，即 `skills/{slug}/{version}.zip`，存储在 COS 上。

ZIP 包内通常包含：
- `SKILL.md`（Skill 描述文件）
- 各种脚本文件（`.js`、`.py`、`.sh` 等）
- 配置文件（`.json`、`.yaml` 等）
- 其他资源文件

### 2.2 两家送审的区别

| 维度 | 科恩 XTI | 三部 AI |
|------|----------|---------|
| **送审文件** | 原始 ZIP 包 | 确定性重新打包后的 ZIP 包（统一时间戳 + 文件排序） |
| **文件标识** | MD5 哈希（32 位） | SHA256 哈希（格式 `sha256:<64位hex>`） |
| **分析方式** | 静态分析 + 动态分析（沙箱运行） | AI 引擎规则匹配 |
| **扫描耗时** | 通常 1-10 分钟 | 通常 1-5 分钟 |

---

## 三、审查结果详解

### 3.1 API 返回的安全扫描数据结构

前端通过 **Dashboard 版本列表接口** 可以获取每个版本的安全扫描结果。每个版本包含以下字段：

```json
{
  "versionId": 100,
  "version": "1.0.0",
  "reviewStatus": "approved",
  "securityScanStatus": "passed",
  "contentAuditStatus": "passed",
  "securityScans": [
    {
      "provider": "keen-xti",
      "status": "passed",
      "attempts": 1,
      "taskId": "abc123def456",
      "fileHash": "d41d8cd98f00b204e9800998ecf8427e",
      "result": "white",
      "threatLevel": 0,
      "resultDetail": { ... },
      "lastError": null,
      "submittedAt": 1773849000000,
      "scannedAt": 1773849300000
    },
    {
      "provider": "tencent-sanbu",
      "status": "passed",
      "attempts": 1,
      "taskId": null,
      "fileHash": "sha256:e3b0c44298fc1c149afbf4c8996fb924...",
      "result": "benign",
      "threatLevel": null,
      "resultDetail": { ... },
      "lastError": null,
      "submittedAt": 1773849000000,
      "scannedAt": 1773849090000
    }
  ]
}
```

### 3.2 版本级聚合字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `securityScanStatus` | string \| null | **安全扫描聚合状态**，综合所有 provider 的结果。可能的值见下表 |
| `contentAuditStatus` | string \| null | 内容合规审核聚合状态（非安全扫描，此处不展开） |
| `reviewStatus` | string \| null | 最终审核状态。安全扫描 + 内容合规全部通过后自动变为 `approved` |

**`securityScanStatus` 聚合规则：**

| 聚合状态 | 含义 | 触发条件 |
|----------|------|----------|
| `pending` | 等待扫描 | 刚创建，尚未提交 |
| `scanning` | 扫描中 | 至少一家正在扫描 |
| `passed` | ✅ 全部通过 | **所有 provider 都通过** |
| `failed` | ❌ 扫描不通过 | **任一 provider 不通过** |
| `error` | ⚠️ 扫描异常 | 任一 provider 出错（会自动重试） |

### 3.3 单个 Provider 扫描结果字段

`securityScans` 数组中每个元素代表一个扫描供应商的结果：

| 字段 | 类型 | 说明 |
|------|------|------|
| `provider` | string | 扫描供应商标识：`keen-xti` 或 `tencent-sanbu` |
| `status` | string | 该 provider 的扫描状态（见下方状态表） |
| `attempts` | number | 已提交扫描次数（出错重试时会递增） |
| `taskId` | string \| null | 供应商返回的任务 ID（科恩有，三部无） |
| `fileHash` | string \| null | 扫描样本的文件哈希。科恩为 MD5，三部为 `sha256:xxx` 格式 |
| `result` | string \| null | **扫描结论**（核心字段，详见 3.4） |
| `threatLevel` | number \| null | **威胁等级**（核心字段，详见 3.5） |
| `resultDetail` | object \| null | **完整扫描结果详情**（核心字段，详见 3.6 和 3.7） |
| `lastError` | string \| null | 最近一次扫描错误信息（仅 error 状态时有值） |
| `submittedAt` | number \| null | 提交扫描时间（毫秒时间戳） |
| `scannedAt` | number \| null | 扫描完成时间（毫秒时间戳） |

**`status` 状态说明：**

| 状态 | 含义 | 是否终态 |
|------|------|----------|
| `pending` | 等待提交 | 否 |
| `submitted` | 已提交到扫描平台 | 否 |
| `scanning` | 扫描平台正在分析中 | 否 |
| `passed` | ✅ 扫描通过，无安全风险 | ✅ 是 |
| `failed` | ❌ 扫描不通过，存在安全风险 | ✅ 是 |
| `error` | ⚠️ 扫描出错（网络异常等），系统会自动重试 | 否 |

### 3.4 `result` 字段 — 扫描结论

两家扫描平台返回的结论值不同：

#### 科恩 XTI 的 `result` 值

| result 值 | 含义 | 对应 status |
|-----------|------|-------------|
| `white` | 🟢 安全，无威胁 | `passed` |
| `suspicious` | 🟡 可疑，存在潜在风险 | `failed` |
| `black` | 🔴 恶意，确认存在安全威胁 | `failed` |

#### 三部 AI 的 `result` 值

| result 值 | 含义 | 对应 status |
|-----------|------|-------------|
| `benign` | 🟢 安全，无风险 | `passed` |
| `suspicious` | 🟡 可疑，存在潜在风险 | `passed` |
| `malicious` | 🔴 恶意，确认存在安全威胁 | `failed` |

### 3.5 `threatLevel` 字段 — 威胁等级

#### 科恩 XTI 的 `threatLevel`

科恩返回数值型威胁等级（0-4）：

| threatLevel | 含义 | 说明 |
|-------------|------|------|
| `0` | 🟢 无威胁 | 文件安全 |
| `1` | 🟢 低风险 | 存在轻微可疑行为，但不构成威胁 |
| `2` | 🟡 中风险 | 存在可疑行为，建议关注 |
| `3` | 🟠 高风险 | 存在较明确的恶意行为 |
| `4` | 🔴 严重威胁 | 确认恶意，如木马、病毒等 |

#### 三部 AI 的 `threatLevel`

三部 AI 目前**不返回数值型威胁等级**（`threatLevel` 为 `null`），风险程度通过 `result`（`risk_level`）字段体现。

### 3.6 `resultDetail` 字段 — 科恩 XTI 详情

科恩的 `resultDetail` 存储的是 `AnalysisSummary` 对象，包含丰富的分析信息：

```json
{
  "threat_level": 4,
  "threat_type": ["trojan"],
  "tags": [
    {
      "tag": "data_exfiltration",
      "desc": "Attempts to exfiltrate sensitive data"
    }
  ],
  "confidence": 90,
  "description": "Trojan detected in skill package",
  "evidence": "Suspicious network calls found in main.js",
  "virus_name": ["Trojan.Generic"],
  "malware_type": "trojan",
  "subfiles": [
    {
      "description": "Contains obfuscated malicious code",
      "verdict": "malicious",
      "virus_name": ["Trojan.JS.Agent"],
      "md5": "f1e2d3c4b5a6f1e2d3c4b5a6f1e2d3c4",
      "threat_level": 4,
      "evidence": "Obfuscated eval() calls",
      "file_path": "src/main.js",
      "malware_type": "trojan"
    }
  ]
}
```

**各字段详细说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `threat_level` | number | 威胁等级（0-4），同外层 `threatLevel` |
| `threat_type` | string[] | 威胁类型标签数组，如 `["trojan"]`、`["adware"]`、`["ransomware"]` 等 |
| `tags` | array | 行为标签数组，每个标签包含 `tag`（标签名）和 `desc`（描述） |
| `confidence` | number | 置信度（0-100），表示分析结果的可信程度 |
| `description` | string | 分析结论的文字描述，如 "No threat detected" 或 "Trojan detected in skill package" |
| `evidence` | string | 证据说明，描述发现威胁的具体依据 |
| `virus_name` | string[] | 病毒/恶意软件名称列表，如 `["Trojan.Generic"]` |
| `malware_type` | string | 恶意软件类型，如 `trojan`（木马）、`adware`（广告软件）、`ransomware`（勒索软件）等 |
| `subfiles` | array | **子文件分析结果**（重要），列出 ZIP 包内每个被检测到问题的文件 |

**`subfiles` 子文件分析字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `file_path` | string | 问题文件在 ZIP 包内的路径，如 `src/main.js` |
| `verdict` | string | 该文件的判定结论：`malicious`（恶意）/ `suspicious`（可疑） |
| `description` | string | 问题描述 |
| `evidence` | string | 具体证据 |
| `virus_name` | string[] | 病毒名称 |
| `md5` | string | 该文件的 MD5 |
| `threat_level` | number | 该文件的威胁等级（0-4） |
| `malware_type` | string | 恶意软件类型 |

**通过时的 `resultDetail` 示例：**

```json
{
  "threat_level": 0,
  "threat_type": [],
  "tags": [],
  "confidence": 95,
  "description": "No threat detected",
  "evidence": "",
  "virus_name": [],
  "malware_type": "",
  "subfiles": []
}
```

### 3.7 `resultDetail` 字段 — 三部 AI 详情

三部 AI 的 `resultDetail` 存储的是完整的 `QueryData` 对象：

```json
{
  "content_hash": "sha256:a1b2c3d4...",
  "risk_level": "malicious",
  "engine_version": 2,
  "scan_items": [
    {
      "scan_type": "static_analysis",
      "rule_list": [
        {
          "rule_id": "RULE_001",
          "description": "Detected obfuscated code execution"
        },
        {
          "rule_id": "RULE_002",
          "description": "Suspicious network access pattern"
        }
      ]
    }
  ],
  "scanned_at": "2026-03-29T10:10:00Z"
}
```

**各字段详细说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `content_hash` | string | 文件的 SHA256 哈希标识 |
| `risk_level` | string | 风险等级：`benign`（安全）/ `suspicious`（可疑）/ `malicious`（恶意） |
| `engine_version` | number | 扫描引擎版本号 |
| `scan_items` | array | **扫描结果项数组**（核心），列出所有命中的扫描规则 |
| `scanned_at` | string | 扫描完成时间（ISO 8601 格式） |

**`scan_items` 扫描结果项字段：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `scan_type` | string | 扫描类型，如 `static_analysis`（静态分析） |
| `rule_list` | array | 命中的规则列表 |
| `rule_list[].rule_id` | string | 规则 ID |
| `rule_list[].description` | string | 规则描述，说明检测到的具体问题 |

**通过时的 `resultDetail` 示例：**

```json
{
  "content_hash": "sha256:e3b0c44298fc1c149afbf4c8996fb924...",
  "risk_level": "benign",
  "engine_version": 2,
  "scan_items": [],
  "scanned_at": "2026-03-29T10:05:00Z"
}
```

> 通过时 `scan_items` 为空数组，表示没有命中任何风险规则。

---

## 四、两家扫描结果对比总结

| 维度 | 科恩 XTI（`keen-xti`） | 三部 AI（`tencent-sanbu`） |
|------|------------------------|---------------------------|
| **通过结论** | `result = "white"` | `result = "benign"` |
| **可疑结论** | `result = "suspicious"` | `result = "suspicious"` |
| **恶意结论** | `result = "black"` | `result = "malicious"` |
| **威胁等级** | `threatLevel`：0-4 数值 | `threatLevel`：null（不返回数值） |
| **详情核心** | `resultDetail.subfiles`：逐文件分析，精确到 ZIP 包内哪个文件有问题 | `resultDetail.scan_items`：按扫描类型列出命中的规则 |
| **详情特色** | 有 `virus_name`（病毒名）、`malware_type`（恶意软件类型）、`confidence`（置信度）、`evidence`（证据） | 有 `rule_id`（规则 ID）、`description`（规则描述）、`engine_version`（引擎版本） |
| **通过时详情** | `subfiles` 为空数组，`description` 为 "No threat detected" | `scan_items` 为空数组 |
| **不通过时详情** | 列出具体哪些文件有问题、什么类型的威胁、具体证据 | 列出命中了哪些安全规则 |

---

## 五、前端展示建议

### 5.1 版本列表页 — 扫描状态展示

建议在版本列表中展示**聚合状态**（`securityScanStatus`），用简洁的标签/图标表示：

| 聚合状态 | 建议展示 | 颜色 |
|----------|----------|------|
| `pending` | 🕐 等待扫描 | 灰色 |
| `scanning` | 🔄 扫描中 | 蓝色 |
| `passed` | ✅ 安全 | 绿色 |
| `failed` | ❌ 存在风险 | 红色 |
| `error` | ⚠️ 扫描异常 | 橙色 |

### 5.2 版本详情页 — 扫描结果展示

点击版本进入详情后，可以展示两家扫描的具体结果：

#### 方案 A：简洁模式（推荐）

展示两张卡片，每张卡片代表一家扫描服务：

```
┌─────────────────────────────────────┐
│ 🔬 科恩 XTI 安全扫描                │
│                                      │
│ 状态：✅ 通过                        │
│ 结论：安全（white）                  │
│ 威胁等级：0 级（无威胁）             │
│ 置信度：95%                          │
│ 说明：No threat detected             │
│ 扫描时间：2026-03-29 18:05:00       │
└─────────────────────────────────────┘

┌─────────────────────────────────────┐
│ 🤖 三部 AI 安全扫描                 │
│                                      │
│ 状态：✅ 通过                        │
│ 结论：安全（benign）                 │
│ 命中规则：无                         │
│ 引擎版本：v2                         │
│ 扫描时间：2026-03-29 18:05:00       │
└─────────────────────────────────────┘
```

#### 方案 B：不通过时展示详情

当扫描不通过时，展示具体的风险信息：

**科恩 XTI 不通过示例：**

```
┌─────────────────────────────────────────────────┐
│ 🔬 科恩 XTI 安全扫描                            │
│                                                  │
│ 状态：❌ 不通过                                  │
│ 结论：恶意（black）                              │
│ 威胁等级：4 级（严重威胁）                       │
│ 威胁类型：trojan（木马）                         │
│ 病毒名称：Trojan.Generic                        │
│ 说明：Trojan detected in skill package           │
│                                                  │
│ 📁 问题文件：                                    │
│ ┌───────────────────────────────────────────┐    │
│ │ src/main.js                               │    │
│ │ 判定：malicious | 威胁等级：4             │    │
│ │ 病毒：Trojan.JS.Agent                     │    │
│ │ 证据：Obfuscated eval() calls             │    │
│ └───────────────────────────────────────────┘    │
└─────────────────────────────────────────────────┘
```

**三部 AI 不通过示例：**

```
┌─────────────────────────────────────────────────┐
│ 🤖 三部 AI 安全扫描                             │
│                                                  │
│ 状态：❌ 不通过                                  │
│ 结论：恶意（malicious）                          │
│                                                  │
│ 📋 命中规则：                                    │
│ ┌───────────────────────────────────────────┐    │
│ │ RULE_001: Detected obfuscated code        │    │
│ │           execution                       │    │
│ │ RULE_002: Suspicious network access       │    │
│ │           pattern                         │    │
│ └───────────────────────────────────────────┘    │
└─────────────────────────────────────────────────┘
```

### 5.3 展示字段优先级建议

根据用户关注度，建议按以下优先级展示字段：

**必须展示（用户最关心）：**

| 优先级 | 字段 | 说明 |
|--------|------|------|
| ⭐⭐⭐ | `status` | 扫描状态（通过/不通过/扫描中） |
| ⭐⭐⭐ | `result` | 扫描结论（建议翻译为中文展示） |
| ⭐⭐⭐ | 科恩 `resultDetail.description` | 结论描述 |
| ⭐⭐⭐ | 三部 `resultDetail.scan_items` | 命中的规则列表 |

**建议展示（帮助用户理解）：**

| 优先级 | 字段 | 说明 |
|--------|------|------|
| ⭐⭐ | 科恩 `resultDetail.threat_level` | 威胁等级 |
| ⭐⭐ | 科恩 `resultDetail.threat_type` | 威胁类型 |
| ⭐⭐ | 科恩 `resultDetail.subfiles` | 问题文件列表（不通过时） |
| ⭐⭐ | 科恩 `resultDetail.virus_name` | 病毒名称（不通过时） |
| ⭐⭐ | `scannedAt` | 扫描完成时间 |

**可选展示（技术细节）：**

| 优先级 | 字段 | 说明 |
|--------|------|------|
| ⭐ | 科恩 `resultDetail.confidence` | 置信度 |
| ⭐ | 科恩 `resultDetail.evidence` | 证据 |
| ⭐ | 科恩 `resultDetail.malware_type` | 恶意软件类型 |
| ⭐ | 三部 `resultDetail.engine_version` | 引擎版本 |
| ⭐ | `fileHash` | 文件哈希 |
| ⭐ | `attempts` | 扫描尝试次数 |

### 5.4 `result` 字段中文翻译建议

| provider | 原始值 | 建议中文展示 |
|----------|--------|-------------|
| 科恩 | `white` | 安全 |
| 科恩 | `suspicious` | 可疑 |
| 科恩 | `black` | 恶意 |
| 三部 | `benign` | 安全 |
| 三部 | `suspicious` | 可疑 |
| 三部 | `malicious` | 恶意 |

### 5.5 `threatLevel` 中文翻译建议（科恩）

| threatLevel | 建议中文展示 | 颜色 |
|-------------|-------------|------|
| 0 | 无威胁 | 绿色 |
| 1 | 低风险 | 绿色 |
| 2 | 中风险 | 黄色 |
| 3 | 高风险 | 橙色 |
| 4 | 严重威胁 | 红色 |

---

## 六、常见问题

### Q1：两家扫描都必须通过才能上架吗？

**是的。** 两家扫描的结果会聚合，只有当 `keen-xti` 和 `tencent-sanbu` 都为 `passed` 时，`securityScanStatus` 才会变为 `passed`，进而触发自动审批。

### Q2：扫描不通过后用户怎么办？

目前用户需要修改代码后重新发布一个新版本。新版本会重新触发两家扫描。

### Q3：扫描出错（error）会怎样？

系统会自动重试。`error` 状态不是终态，Worker 会在下一轮轮询时重新提交扫描。

### Q4：扫描通常需要多久？

- 科恩 XTI：通常 1-10 分钟（包含静态分析和可能的动态分析）
- 三部 AI：通常 1-5 分钟

### Q5：`resultDetail` 在通过和不通过时都有值吗？

**是的。** 无论扫描通过还是不通过，`resultDetail` 都会有值。通过时内容较简单（如 `subfiles` 或 `scan_items` 为空数组），不通过时会包含详细的风险信息。

### Q6：两家扫描的 `provider` 名称是固定的吗？

是的。目前固定为 `keen-xti` 和 `tencent-sanbu`，前端可以据此区分并展示不同的 UI。
