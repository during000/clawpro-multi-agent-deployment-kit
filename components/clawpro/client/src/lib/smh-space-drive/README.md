# SMH Space Drive UI Kit 接入文档

## 简介

`smh-space-drive` 是一个基于 React 的云盘文件管理 UI Kit，提供完整的文件浏览、上传、下载、删除、重命名、移动等功能。接入方只需提供必要的配置参数和一个 `accessToken` 获取函数，即可快速嵌入云盘能力。

---

## 安装

```bash
npm install smh-space-drive
```

### 宿主项目 peerDependencies 要求

| 依赖 | 版本要求 |
|------|---------|
| `react` | `^18.0.0` |
| `react-dom` | `^18.0.0` |

> 宿主项目需自行安装上述 peerDependencies。UI Kit 内部不依赖任何第三方 UI 组件库，所有 UI 均为原生实现。

---

## SDK 依赖说明

`smh-space-drive` UI Kit 的完整运行依赖腾讯云 SMH 官方提供的两个 SDK，分别用于前端和后端：

### smh-js-sdk（前端 SDK）

| 项目 | 说明 |
|------|------|
| 使用方 | `smh-space-drive`（UI Kit 内部已集成） |
| 版本要求 | `^1.0.5` |
| 安装 | UI Kit 内部已包含，**接入方无需单独安装** |
| 用途 | 前端直连 SMH API，完成文件浏览、上传、下载、删除等操作 |

**核心能力：**

- **目录操作** — `client.directory.listDirectoryByPage()` / `createDirectory()` / `deleteDirectory()`：文件列表分页查询、创建/删除目录
- **文件操作** — `client.file.deleteFile()` / `infoFile()`：文件删除、获取文件详情
- **文件上传** — `client.createUploadTask()`：支持分片上传、断点续传、秒传检测，提供进度回调和状态变更通知
- **使用量统计** — `client.usage.getUsage()`：获取当前空间的配额和已用容量

**UI Kit 内部使用示例（接入方无需关心）：**

```javascript
import { SMHClient, TaskStatus } from 'smh-js-sdk'

const client = new SMHClient({
  basePath: 'https://api.tencentsmh.cn',
  libraryId: 'smhxxx-xxxxx',
  spaceId: 'spaceyyy',
  accessToken: 'your_access_token',
})

// 获取文件列表
const res = await client.directory.listDirectoryByPage({
  filePath: '',
  byPage: 1,
  page: 1,
  pageSize: 100,
})

// 上传文件（支持分片、断点续传）
const uploader = client.createUploadTask({
  filePath: 'docs/hello.txt',
  file: fileObject,
  conflictResolutionStrategy: 'overwrite',
  onStateChange: (checkpoint, state, error) => {
    if (state === TaskStatus.SUCCESS) console.log('上传成功')
  },
  onProgress: (info) => console.log(`进度: ${Math.floor(info.progress * 100)}%`),
})
uploader.start()
```

### smh-node-sdk（后端 SDK）

| 项目 | 说明 |
|------|------|
| 使用方 | 接入方的后端服务 |
| 版本要求 | `^1.0.0` |
| 安装 | `npm install smh-node-sdk` |
| 用途 | 后端调用 SMH 管理 API，签发 accessToken、管理空间和配额 |

**核心能力：**

- **Token 签发** — `smh.token.createToken()`：签发 `admin` / `space_admin` 级别的 accessToken（需要 `library_secret`，仅后端持有）
- **空间管理** — `smh.space.listSpace()` / `createSpace()` / `deleteSpace()`：租户空间的增删查
- **配额管理** — `smh.quota.getQuota()` / `createQuota()` / `updateQuota()`：存储配额的查询与设置
- **使用量统计** — `smh.usage.getUsage()`：获取空间容量使用情况

**后端使用示例：**

```javascript
const { SMHClient } = require('smh-node-sdk')

const smh = new SMHClient({ basePath: 'https://api.tencentsmh.cn' })
smh.setDefaultLibraryId('smhxxx-xxxxx')

// 签发 space_admin 级别的 token（供前端 UI Kit 使用）
const tokenData = await smh.token.createToken({
  libraryId: 'smhxxx-xxxxx',
  librarySecret: 'your_secret',  // ⚠️ 仅后端持有
  spaceId: 'spaceyyy',
  grant: 'space_admin',
  period: 7200,  // 2 小时
})

// 创建租户空间
const space = await smh.space.createSpace({
  libraryId: 'smhxxx-xxxxx',
  accessToken: adminToken,
  userId: 'user123',
  createSpaceRequest: {
    allowPhoto: true,
    isPublicRead: true,
    allowVideo: true,
  },
})

// 查询空间使用量
const usage = await smh.usage.getUsage({
  libraryId: 'smhxxx-xxxxx',
  spaceIds: 'spaceyyy',
  accessToken: adminToken,
  userId: 'user123',
})
```

### 两个 SDK 的分工关系

```
┌──────────────────────────────────────────────────────────┐
│                      浏览器 (前端)                        │
│                                                          │
│  smh-space-drive 内部使用 smh-js-sdk 直连 SMH API        │
│  ├── 文件浏览、上传、下载、删除、重命名、移动              │
│  └── 前端持有 accessToken（由后端签发，不含 secret）       │
│                                                          │
└──────────────────────────┬───────────────────────────────┘
                           │ 调用 getAccessToken()
                           │ 获取 accessToken
                           ▼
┌──────────────────────────────────────────────────────────┐
│                    Node.js 服务端 (后端)                   │
│                                                          │
│  接入方后端使用 smh-node-sdk 调用 SMH 管理 API            │
│  ├── 签发 accessToken（需要 library_secret，仅后端持有）   │
│  ├── 空间管理（创建/删除租户空间）                         │
│  ├── 配额管理（查询/修改存储上限）                         │
│  └── 使用量统计                                           │
│                                                          │
└──────────────────────────────────────────────────────────┘
```

> ⚠️ **安全原则**：`library_secret` 仅存在于服务端，前端通过后端签发的 `accessToken` 访问 SMH API，永远不会接触到密钥。UI Kit 内部的 `smh-js-sdk` 只需要 `accessToken` 即可工作。

---

## 快速开始

```jsx
import { SpaceDrive } from 'client/src/lib/smh-space-drive/smh-space-drive'

function App() {
  // accessToken 提供函数（详见下方说明）
  const getAccessToken = async () => {
    const res = await fetch('/api/your-backend/get-smh-token', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ spaceId: 'your-space-id' }),
    })
    const data = await res.json()
    return {
      accessToken: data.accessToken,
      expiresAt: data.expiresAt, // 毫秒级时间戳
    }
  }

  return (
    <SpaceDrive
      basePath="https://api.tencentsmh.cn"
      libraryId="smhxxx-xxxxx"
      spaceId="space-xxxxx"
      getAccessToken={getAccessToken}
    />
  )
}
```

---

## Props 参数说明

### `<SpaceDrive />` 组件

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `basePath` | `string` | ✅ | - | SMH API 基础路径，如 `https://api.tencentsmh.cn` |
| `libraryId` | `string` | ✅ | - | SMH 媒体库 ID，由腾讯云 SMH 控制台创建后获取 |
| `spaceId` | `string` | ✅ | - | SMH 空间 ID，标识一个独立的存储空间 |
| `getAccessToken` | `Function` | ✅ | - | accessToken 提供函数，详见下方 |
| `showUserCard` | `boolean` | ❌ | `true` | 是否在侧边栏显示用户信息卡片 |

---

## 核心概念

### 参数获取方式

#### 1. `basePath` — SMH API 基础路径

SMH 服务的 API 域名地址，由腾讯云 SMH 服务提供。格式如：

```
https://api.tencentsmh.cn
```

> 该值通常由后端配置提供，不同环境（测试/生产）可能不同。

#### 2. `libraryId` — 媒体库 ID

在 [腾讯云 SMH 控制台](https://console.cloud.tencent.com/smh) 创建媒体库后获得的唯一标识。格式如：

```
smhxxx-xxxxx
```

> 一个 `libraryId` 下可以包含多个 `spaceId`（空间）。

#### 3. `spaceId` — 空间 ID

每个用户/租户对应一个独立的存储空间，由后端通过 SMH API 创建。格式如：

```
spaceyyyyyy
```

> 不同用户应使用不同的 `spaceId`，以实现数据隔离。

#### 4. `getAccessToken` — Token 提供函数（重点）

这是接入方**必须实现**的核心函数。UI Kit 内部会在以下时机调用此函数：

- **初始化时**：组件挂载后首次获取 token
- **token 即将过期时**：UI Kit 内部每 30 秒检查一次，当 token 距过期不足 5 分钟时自动调用续期
- **请求发现 token 失效时**：API 请求返回 token 无效错误时自动重试

**函数签名：**

```typescript
type GetAccessToken = () => Promise<{
  accessToken: string  // SMH 访问令牌
  expiresAt: number    // 过期时间，毫秒级时间戳（如 Date.now() + 2 * 60 * 60 * 1000）
}>
```

**实现示例：**

```jsx
const getAccessToken = async () => {
  const res = await fetch('/api/your-backend/generate-space-token', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ spaceId: 'your-space-id' }),
  })

  if (!res.ok) {
    throw new Error('获取访问令牌失败')
  }

  const data = await res.json()
  return {
    accessToken: data.accessToken,  // SMH accessToken
    expiresAt: data.expiresAt,      // 毫秒级过期时间戳
  }
}
```

> ⚠️ **重要**：`getAccessToken` 函数的引用应保持稳定（使用 `useCallback` 或定义在组件外部），避免因引用变化导致组件重复初始化。

---

## 后端需要提供的接口

接入方的后端需要实现一个 **生成 SMH Space Token** 的接口，供前端 `getAccessToken` 函数调用。

### 接口职责

1. 验证当前用户的身份和权限
2. 调用 SMH API 生成指定 `spaceId` 的 `accessToken`
3. 返回 `accessToken` 和过期时间

### SMH 生成 Token 的 API

```
GET {basePath}/api/v1/token?library_id={libraryId}&library_secret={librarySecret}&Grant=space_admin&Period=7200&spaceId={spaceId}
```

| 参数 | 说明 |
|------|------|
| `library_id` | 媒体库 ID |
| `library_secret` | 媒体库密钥（**仅后端持有，切勿暴露给前端**） |
| `Grant` | 授权级别，推荐 `space_admin` |
| `Period` | token 有效期（秒），推荐 `7200`（2 小时） |
| `spaceId` | 目标空间 ID |

### 后端接口返回格式（建议）

```json
{
  "accessToken": "xxx-access-token-xxx",
  "expiresAt": 1711468800000,
  "libraryId": "smhxxx-xxxxx",
  "spaceId": "spaceyyyyyy",
  "basePath": "https://api.tencentsmh.cn"
}
```

> 其中 `expiresAt` 为**毫秒级时间戳**，计算方式：`Date.now() + Period * 1000`

### Node.js 后端示例

```javascript
app.post('/api/generate-space-token', async (req, res) => {
  const { spaceId } = req.body

  // 1. 验证用户权限（根据业务逻辑）
  // ...

  // 2. 调用 SMH API 获取 token
  const params = new URLSearchParams({
    library_id: process.env.SMH_LIBRARY_ID,
    library_secret: process.env.SMH_LIBRARY_SECRET,
    Grant: 'space_admin',
    Period: '7200', // 2 小时
    spaceId,
  })

  const response = await fetch(`${process.env.SMH_BASE_PATH}/api/v1/token?${params}`)
  const tokenData = await response.json()

  // 3. 计算过期时间并返回
  const expiresAt = Date.now() + 2 * 60 * 60 * 1000

  res.json({
    accessToken: tokenData.accessToken,
    expiresAt,
    libraryId: process.env.SMH_LIBRARY_ID,
    spaceId,
    basePath: process.env.SMH_BASE_PATH,
  })
})
```

---

## 完整接入示例

以下是一个在管理后台中嵌入云盘的完整示例：

```jsx
import { useState, useCallback, useRef } from 'react'
import { Drawer, Button } from 'tdesign-react' // 示例使用 TDesign，宿主项目可使用任意 UI 库
import { SpaceDrive, clearConfig as clearDriveConfig } from 'smh-space-drive'
import 'smh-space-drive/dist/style.css'

export default function MyPage() {
  const [drawerVisible, setDrawerVisible] = useState(false)
  const [driveConfig, setDriveConfig] = useState(null)
  const spaceIdRef = useRef('')
  const cachedTokenRef = useRef(null)

  // accessToken 提供函数（引用稳定）
  const getAccessToken = useCallback(async () => {
    // 如果有缓存的 token（首次进入时预获取），直接返回
    if (cachedTokenRef.current) {
      const cached = cachedTokenRef.current
      cachedTokenRef.current = null
      return cached
    }

    const res = await fetch('/api/generate-space-token', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ spaceId: spaceIdRef.current }),
    })
    if (!res.ok) throw new Error('获取访问令牌失败')
    const data = await res.json()
    return {
      accessToken: data.accessToken,
      expiresAt: data.expiresAt,
    }
  }, [])

  // 打开云盘
  const handleOpenDrive = async (spaceId) => {
    spaceIdRef.current = spaceId

    // 预获取 token 和配置信息
    const res = await fetch('/api/generate-space-token', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      credentials: 'include',
      body: JSON.stringify({ spaceId }),
    })
    const data = await res.json()

    // 缓存首次 token，避免 UI Kit 初始化时重复请求
    cachedTokenRef.current = {
      accessToken: data.accessToken,
      expiresAt: data.expiresAt,
    }

    setDriveConfig({
      basePath: data.basePath,
      libraryId: data.libraryId,
      spaceId,
      getAccessToken,
    })
    setDrawerVisible(true)
  }

  // 关闭云盘
  const handleCloseDrive = () => {
    setDrawerVisible(false)
    setTimeout(() => {
      setDriveConfig(null)
      cachedTokenRef.current = null
      clearDriveConfig() // 清理 UI Kit 内部状态
    }, 300)
  }

  return (
    <div>
      <Button onClick={() => handleOpenDrive('space-xxxxx')}>
        打开云盘
      </Button>

      <Drawer
        visible={drawerVisible}
        onClose={handleCloseDrive}
        size="85%"
        destroyOnClose
      >
        {driveConfig && (
          <SpaceDrive
            basePath={driveConfig.basePath}
            libraryId={driveConfig.libraryId}
            spaceId={driveConfig.spaceId}
            getAccessToken={driveConfig.getAccessToken}
            showUserCard={false}
          />
        )}
      </Drawer>
    </div>
  )
}
```

---

## 导出的工具函数（高级用法）

除了 `<SpaceDrive />` 组件，UI Kit 还导出了以下工具函数，供高级场景使用：

**主入口（`smh-space-drive`）— 组件 & Token 管理：**

```javascript
import {
  SpaceDrive,          // 核心组件
  FilePage,            // 底层文件管理组件
  setSmhConfig,        // 手动设置 SMH 配置
  getAccessToken,      // 获取当前 accessToken
  getLibraryId,        // 获取当前 libraryId
  getSpaceId,          // 获取当前 spaceId
  getBasePath,         // 获取当前 basePath
  getTokenExpireInfo,  // 获取 token 过期时间信息
  isTokenExpiringSoon, // 检查 token 是否即将过期
  isTokenExpired,      // 检查 token 是否已过期
  ensureValidToken,    // 确保 token 有效（自动续期）
  initToken,           // 初始化 token
  clearConfig,         // 清除所有配置和 token
} from 'smh-space-drive'
```

**服务层入口（`smh-space-drive/services/smh`）— 文件操作 API：**

```javascript
import {
  getFileList,               // 获取文件列表（分页）
  uploadFile,                // 上传文件（分片/断点续传/秒传）
  delFile,                   // 删除文件
  delDirectory,              // 删除目录
  createDirectory,           // 创建文件夹
  moveFile,                  // 移动文件
  moveDirectory,             // 移动目录
  renameFile,                // 重命名文件
  renameDirectory,           // 重命名目录
  getFileInfo,               // 获取文件详情
  getPreview,                // 获取预览地址或内容
  getDocPreviewUrl,          // 获取文档在线预览 URL
  downloadFile,              // 下载文件（触发浏览器下载）
  getFilePreviewUrlOrContent,// 自动判断类型获取预览
  getSpaceUsage,             // 获取空间配额和使用量
  resetClient,               // 重置 SDK Client 实例
} from 'smh-space-drive/services/smh'
```

---

## 仅使用服务层（自定义 UI）

如果你希望**完全自定义 UI 界面**，只使用 `smh-space-drive` 提供的服务层能力（Token 管理 + 文件操作 API），可以跳过 `<SpaceDrive />` 组件，直接调用底层服务函数。

> 服务层不依赖任何 UI 组件（无 React、无 TDesign 依赖），可以在任意前端框架中使用。

### 接入步骤

#### 1. 初始化配置 & Token

在应用启动时，调用 `setSmhConfig` 设置 SMH 连接参数，然后调用 `initToken` 完成首次 Token 获取：

```javascript
import {
  setSmhConfig,
  initToken,
  clearConfig,
} from 'smh-space-drive'

// 1. 设置配置
setSmhConfig({
  basePath: 'https://api.tencentsmh.cn',
  libraryId: 'smhxxx-xxxxx',
  spaceId: 'spaceyyyyyy',
  getAccessToken: async () => {
    const res = await fetch('/api/your-backend/generate-space-token', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ spaceId: 'spaceyyyyyy' }),
    })
    const data = await res.json()
    return { accessToken: data.accessToken, expiresAt: data.expiresAt }
  },
  // 可选：自定义错误提示（不传则静默，仅 throw Error）
  onError: ({ message }) => {
    // 替换为你自己的 UI 提示方式，如 antd message、Element Plus ElMessage 等
    console.error('[SMH Error]', message)
  },
})

// 2. 初始化 Token（必须在调用任何文件操作 API 之前完成）
await initToken()

// 3. 应用卸载时清理
// clearConfig()
```

#### 2. 调用文件操作 API

初始化完成后，直接 `import` 服务层的文件操作函数即可使用。**所有 API 内部会自动管理 Token 续期**，调用方无需关心 Token 生命周期。

```javascript
import {
  getFileList,
  uploadFile,
  delFile,
  delDirectory,
  createDirectory,
  moveFile,
  moveDirectory,
  renameFile,
  renameDirectory,
  getPreview,
  getDocPreviewUrl,
  downloadFile,
  getFileInfo,
  getFilePreviewUrlOrContent,
  getSpaceUsage,
  resetClient,
} from 'smh-space-drive/services/smh'
```

> 注意：文件操作 API 从 `smh-space-drive/services/smh` 路径导入，而非主入口。

### 服务层 API 一览

#### 配置与 Token 管理（`smh-space-drive`）

| 函数 | 说明 | 调用时机 |
|------|------|----------|
| `setSmhConfig(config)` | 设置 SMH 连接配置 | **应用启动时**，必须最先调用 |
| `initToken()` | 初始化 Token（首次获取） | **应用启动时**，`setSmhConfig` 之后调用 |
| `ensureValidToken()` | 确保 Token 有效，自动续期 | 一般无需手动调用，文件操作 API 内部已自动调用 |
| `isTokenExpiringSoon()` | 检查 Token 是否即将过期（5 分钟内） | 需要自行实现定时续期逻辑时使用 |
| `isTokenExpired()` | 检查 Token 是否已过期 | 需要判断 Token 状态时使用 |
| `getAccessToken()` | 获取当前 accessToken 字符串 | 需要手动拼接请求时使用 |
| `getTokenExpireInfo()` | 获取 Token 过期时间信息 | 需要展示 Token 状态时使用 |
| `getLibraryId()` | 获取当前 libraryId | 需要手动拼接请求时使用 |
| `getSpaceId()` | 获取当前 spaceId | 需要手动拼接请求时使用 |
| `getBasePath()` | 获取当前 basePath | 需要手动拼接请求时使用 |
| `clearConfig()` | 清除所有配置和 Token | **应用卸载 / 切换用户时**调用 |

#### 文件操作 API（`smh-space-drive/services/smh`）

| 函数 | 说明 | 典型场景 |
|------|------|----------|
| `getFileList(dirPath, { page, pageSize })` | 获取目录下的文件列表（分页） | 文件列表页、目录浏览 |
| `uploadFile(file, filePath, callbacks)` | 上传文件（支持分片、断点续传、秒传） | 文件上传功能 |
| `delFile(filePath)` | 删除单个文件 | 文件删除操作 |
| `delDirectory(dirPath)` | 删除目录 | 目录删除操作 |
| `createDirectory(dirPath)` | 创建文件夹 | 新建文件夹功能 |
| `moveFile(fromPath, toPath)` | 移动文件到目标路径 | 文件移动 / 整理 |
| `moveDirectory(fromPath, toPath)` | 移动目录到目标路径 | 目录移动 / 整理 |
| `renameFile(oldPath, newPath)` | 重命名文件 | 文件重命名 |
| `renameDirectory(oldPath, newPath)` | 重命名目录 | 目录重命名 |
| `getFileInfo(filePath)` | 获取文件详情信息 | 文件详情页、属性面板 |
| `getPreview(filePath, isDoc?)` | 获取文件预览地址或内容 | 图片/视频预览、文档内容读取 |
| `getDocPreviewUrl(filePath)` | 获取文档在线预览 URL | iframe 嵌入预览 Office 文档 |
| `downloadFile(filePath, fileName?)` | 下载文件（触发浏览器原生下载） | 文件下载功能 |
| `getFilePreviewUrlOrContent(file)` | 根据文件类型自动获取预览链接或内容 | 通用预览（自动判断图片/视频/文档） |
| `getSpaceUsage()` | 获取空间配额和已用容量 | 存储用量展示、容量告警 |
| `resetClient()` | 重置内部 SDK Client 实例 | 切换空间 / 配置变更后调用 |

### 完整示例：Vue 3 中使用服务层

以下示例展示在 Vue 3 项目中，仅使用服务层构建自定义文件管理界面：

```vue
<template>
  <div class="file-manager">
    <div class="toolbar">
      <button @click="handleCreateFolder">新建文件夹</button>
      <input type="file" @change="handleUpload" />
      <span>已用: {{ formatSize(usage.used) }} / {{ formatSize(usage.total) }}</span>
    </div>

    <ul class="file-list">
      <li v-for="file in files" :key="file.name" @click="handleOpen(file)">
        <span>{{ file.type === 'dir' ? '📁' : '📄' }} {{ file.name }}</span>
        <button @click.stop="handleDelete(file)">删除</button>
        <button @click.stop="handleDownload(file)" v-if="file.type !== 'dir'">下载</button>
      </li>
    </ul>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import {
  setSmhConfig, initToken, clearConfig,
  isTokenExpiringSoon, ensureValidToken,
} from 'smh-space-drive'
import {
  getFileList, uploadFile, delFile, delDirectory,
  createDirectory, downloadFile, getSpaceUsage,
} from 'smh-space-drive/services/smh'

const files = ref([])
const currentPath = ref('')
const usage = ref({ used: 0, total: 0 })

// 初始化
onMounted(async () => {
  setSmhConfig({
    basePath: 'https://api.tencentsmh.cn',
    libraryId: 'smhxxx-xxxxx',
    spaceId: 'spaceyyyyyy',
    getAccessToken: async () => {
      const res = await fetch('/api/generate-space-token', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ spaceId: 'spaceyyyyyy' }),
      })
      const data = await res.json()
      return { accessToken: data.accessToken, expiresAt: data.expiresAt }
    },
    onError: ({ message }) => ElMessage.error(message), // Element Plus 提示
  })

  await initToken()
  await loadFiles()

  const usageData = await getSpaceUsage()
  if (usageData) usage.value = usageData
})

// Token 定时续期
const timer = setInterval(() => {
  if (isTokenExpiringSoon()) ensureValidToken().catch(() => {})
}, 30 * 1000)

onUnmounted(() => {
  clearInterval(timer)
  clearConfig()
})

// 加载文件列表
async function loadFiles() {
  const data = await getFileList(currentPath.value, { page: 1, pageSize: 100 })
  files.value = data.contents || []
}

// 新建文件夹
async function handleCreateFolder() {
  const name = prompt('请输入文件夹名称')
  if (!name) return
  const path = currentPath.value ? `${currentPath.value}/${name}` : name
  await createDirectory(path)
  await loadFiles()
}

// 上传文件
async function handleUpload(e) {
  const file = e.target.files[0]
  if (!file) return
  const filePath = currentPath.value ? `${currentPath.value}/${file.name}` : file.name
  await uploadFile(file, filePath, {
    onProgressCallback: (percent) => console.log(`上传进度: ${percent}%`),
    onSuccessCallback: () => loadFiles(),
    onErrorCallback: () => console.error('上传失败'),
  })
}

// 删除
async function handleDelete(file) {
  if (file.type === 'dir') {
    await delDirectory(currentPath.value ? `${currentPath.value}/${file.name}` : file.name)
  } else {
    await delFile(currentPath.value ? `${currentPath.value}/${file.name}` : file.name)
  }
  await loadFiles()
}

// 下载
async function handleDownload(file) {
  const filePath = currentPath.value ? `${currentPath.value}/${file.name}` : file.name
  await downloadFile(filePath, file.name)
}

function formatSize(bytes) {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return (bytes / Math.pow(1024, i)).toFixed(1) + ' ' + units[i]
}
</script>
```

### 自行管理 Token 续期

当你不使用 `<SpaceDrive />` 组件时，需要**自行实现 Token 定时续期**。推荐方式：

```javascript
import { isTokenExpiringSoon, ensureValidToken } from 'smh-space-drive'

// 每 30 秒检查一次，即将过期时自动续期
const timer = setInterval(() => {
  if (isTokenExpiringSoon()) {
    ensureValidToken().catch(err => console.error('Token 续期失败:', err))
  }
}, 30 * 1000)

// 应用卸载时清理
clearInterval(timer)
```

> `<SpaceDrive />` 组件内部已内置此逻辑，使用组件时无需手动处理。

### 服务层 API 详细说明

以下是每个文件操作 API 的详细参数签名、返回值和使用示例。

#### `getFileList(dirPath, options)` — 获取文件列表

```typescript
function getFileList(
  dirPath?: string,                    // 目录路径，空字符串表示根目录
  options?: { page?: number, pageSize?: number }  // 分页参数，默认 page=1, pageSize=100
): Promise<{
  contents: Array<{                    // 文件/目录列表
    name: string                       // 文件名
    type: 'file' | 'dir'               // 类型
    size?: number                      // 文件大小（字节），目录无此字段
    creationTime?: string              // 创建时间
    modificationTime?: string          // 修改时间
    // ...其他 SMH 返回字段
  }>
  totalNum: number                     // 总条数
  hasMore: boolean                     // 是否有更多
}>
```

**示例：**

```javascript
// 获取根目录第一页
const data = await getFileList('', { page: 1, pageSize: 50 })
console.log(data.contents)  // 文件列表
console.log(data.totalNum)  // 总数

// 获取子目录
const subData = await getFileList('docs/reports', { page: 1, pageSize: 100 })
```

#### `uploadFile(file, filePath, callbacks)` — 上传文件

支持分片上传、断点续传、秒传检测。如果目标目录不存在，会自动创建。

```typescript
function uploadFile(
  file: File,                          // 浏览器 File 对象
  filePath: string,                    // 上传目标路径，如 'docs/report.pdf'
  callbacks?: {
    onProgressCallback?: (percent: number, speed: number) => void  // 进度回调，percent 为 0-100，speed 单位 B/s
    onSuccessCallback?: (result: { id: string, name: string }) => void  // 上传成功回调
    onErrorCallback?: (error: any) => void                        // 上传失败回调
    onStateChangeCallback?: (state: string) => void               // 状态变更回调（如 'computing_hash' 秒传校验中）
  }
): Promise<object>                     // 返回上传 checkpoint 信息
```

**示例：**

```javascript
// 基础上传
await uploadFile(fileObject, 'docs/hello.pdf')

// 带进度回调的上传
await uploadFile(fileObject, 'images/photo.jpg', {
  onProgressCallback: (percent, speed) => {
    console.log(`进度: ${percent}%, 速度: ${(speed / 1024).toFixed(1)} KB/s`)
  },
  onSuccessCallback: ({ id, name }) => {
    console.log(`上传成功: ${name}`)
  },
  onErrorCallback: (err) => {
    console.error('上传失败:', err)
  },
  onStateChangeCallback: (state) => {
    if (state === 'computing_hash') console.log('正在校验秒传...')
  },
})
```

#### `delFile(filePath)` — 删除文件

```typescript
function delFile(filePath: string | string[]): Promise<boolean>  // 返回是否删除成功
```

**示例：**

```javascript
const success = await delFile('docs/old-report.pdf')
// 也支持数组形式的路径
const success2 = await delFile(['docs', 'old-report.pdf'])
```

#### `delDirectory(dirPath)` — 删除目录

```typescript
function delDirectory(dirPath: string | string[]): Promise<boolean>  // 返回是否删除成功
```

**示例：**

```javascript
const success = await delDirectory('docs/archive')
```

#### `createDirectory(dirPath)` — 创建文件夹

如果同名目录已存在，会自动重命名（追加后缀）。

```typescript
function createDirectory(dirPath: string | string[]): Promise<object>  // 返回创建结果
```

**示例：**

```javascript
await createDirectory('docs/new-folder')
```

#### `moveFile(fromPath, toPath)` — 移动文件

将文件从一个路径移动到另一个路径。如果目标路径已存在同名文件，会自动重命名。

```typescript
function moveFile(
  fromPath: string | string[],         // 原文件路径
  toPath: string | string[]            // 目标文件路径
): Promise<object>
```

**示例：**

```javascript
// 将文件从根目录移动到 docs 目录
await moveFile('report.pdf', 'docs/report.pdf')

// 数组形式
await moveFile(['uploads', 'photo.jpg'], ['images', 'photo.jpg'])
```

#### `moveDirectory(fromPath, toPath)` — 移动目录

将目录从一个路径移动到另一个路径。如果目标路径已存在同名目录，会自动重命名。

```typescript
function moveDirectory(
  fromPath: string | string[],         // 原目录路径
  toPath: string | string[]            // 目标目录路径
): Promise<object>
```

**示例：**

```javascript
await moveDirectory('temp/drafts', 'docs/drafts')
```

#### `renameFile(oldPath, newPath)` — 重命名文件

通过移动操作实现重命名。如果目标路径已存在同名文件，会提示冲突（`conflictResolutionStrategy: 'ask'`）。

```typescript
function renameFile(
  oldPath: string | string[],          // 原文件路径
  newPath: string | string[]           // 新文件路径
): Promise<object>
```

**示例：**

```javascript
await renameFile('docs/old-name.pdf', 'docs/new-name.pdf')
```

#### `renameDirectory(oldPath, newPath)` — 重命名目录

```typescript
function renameDirectory(
  oldPath: string | string[],          // 原目录路径
  newPath: string | string[]           // 新目录路径
): Promise<object>
```

**示例：**

```javascript
await renameDirectory('docs/old-folder', 'docs/new-folder')
```

#### `getFileInfo(filePath)` — 获取文件详情

```typescript
function getFileInfo(filePath: string | string[]): Promise<{
  name: string
  type: 'file' | 'dir'
  size?: number
  creationTime?: string
  modificationTime?: string
  // ...其他 SMH 返回字段
}>
```

**示例：**

```javascript
const info = await getFileInfo('docs/report.pdf')
console.log(`文件大小: ${info.size} 字节`)
```

#### `getPreview(filePath, isDoc?)` — 获取预览地址或内容

根据文件类型返回不同结果：
- **图片/视频**：返回带 `accessToken` 的直连 URL（字符串）
- **文档**（`isDoc=true`）：获取文件的 `cosUrl` 并返回文本内容

```typescript
function getPreview(
  filePath: string | string[],         // 文件路径
  isDoc?: boolean                      // 是否为文档类型，默认 false
): Promise<string | object>            // 图片/视频返回 URL 字符串，文档返回文本内容
```

**示例：**

```javascript
// 图片预览 URL
const imgUrl = await getPreview('images/photo.jpg')
img.src = imgUrl

// 文档内容
const textContent = await getPreview('docs/readme.md', true)
console.log(textContent)
```

#### `getDocPreviewUrl(filePath)` — 获取文档在线预览 URL

返回文档的 `cosUrl`，可用于 iframe 嵌入在线预览 Office 文档。

```typescript
function getDocPreviewUrl(filePath: string | string[]): Promise<string>  // cosUrl
```

**示例：**

```javascript
const previewUrl = await getDocPreviewUrl('docs/report.docx')
// 在 iframe 中预览
iframe.src = previewUrl
```

#### `downloadFile(filePath, fileName?)` — 下载文件

通过 SDK 的 `downloadByUrl` 方法获取文件的 `cosUrl`，并通过 `<a>` 标签触发浏览器原生下载。

```typescript
function downloadFile(
  filePath: string | string[],         // 文件路径
  fileName?: string                    // 自定义保存文件名，不传则使用远端文件名
): Promise<void>
```

**示例：**

```javascript
// 使用原始文件名下载
await downloadFile('docs/report.pdf')

// 自定义下载文件名
await downloadFile('docs/report.pdf', '2024年度报告.pdf')
```

#### `getFilePreviewUrlOrContent(file)` — 自动获取预览

根据文件扩展名自动判断类型，调用对应的预览方法：
- **图片/视频**（jpg、png、gif、mp4 等）→ 返回预览 URL
- **文档/文本**（json、txt、md、log、docx）→ 返回文本内容
- **其他类型** → 返回预览 URL

```typescript
function getFilePreviewUrlOrContent(file: {
  name: string                         // 文件名（用于判断扩展名）
  path?: string[]                      // 文件路径数组
}): Promise<string>
```

**示例：**

```javascript
const result = await getFilePreviewUrlOrContent({
  name: 'photo.jpg',
  path: ['images', 'photo.jpg'],
})
```

#### `getSpaceUsage()` — 获取空间使用量

```typescript
function getSpaceUsage(): Promise<{
  used: number   // 已使用容量（字节）
  total: number  // 总配额（字节）
} | null>        // 获取失败返回 null
```

**示例：**

```javascript
const usage = await getSpaceUsage()
if (usage) {
  const usedGB = (usage.used / 1024 / 1024 / 1024).toFixed(2)
  const totalGB = (usage.total / 1024 / 1024 / 1024).toFixed(2)
  console.log(`已用 ${usedGB} GB / 共 ${totalGB} GB`)
}
```

#### `resetClient()` — 重置 SDK Client

当 SMH 配置（`basePath`、`libraryId`、`spaceId`）发生变更时，需要调用此函数重置内部的 `SMHClient` 实例，下次 API 调用时会自动创建新实例。

```typescript
function resetClient(): void
```

**示例：**

```javascript
// 切换空间后重置 client
setSmhConfig({ spaceId: 'new-space-id', ... })
resetClient()
await initToken()
```

### 两种接入方式对比

| | 使用 `<SpaceDrive />` 组件 | 仅使用服务层 |
|---|---|---|
| **适用场景** | 快速接入，使用现成 UI | 需要完全自定义 UI 风格 |
| **UI 框架** | 必须使用 React | 任意框架（Vue、Angular、Svelte 等） |
| **Token 续期** | 自动（组件内置） | 需自行实现定时检查 |
| **错误提示** | 自动（内置 Toast） | 通过 `onError` 回调自定义 |
| **需要引入样式** | 是（`style.css`） | 否 |
| **代码量** | 最少，几行即可 | 较多，需自行构建 UI |

---

## Token 自动续期机制

UI Kit 内部实现了完整的 token 自动续期机制，接入方**无需手动管理 token 生命周期**：

```
时间线：
0h                              1h55m                    2h
|------- 正常使用 token ---------|--- 检测到即将过期 ---|--- 旧 token 过期 ---|
                                  ↓
                          自动调用 getAccessToken()
                          获取新 token，无缝续期
```

- **检查频率**：每 30 秒检查一次 token 状态
- **提前续期**：在 token 过期前 5 分钟触发续期
- **并发保护**：多个请求同时发现 token 过期时，只会触发一次续期
- **请求级保护**：每次 API 请求前都会检查 token 有效性

---

## 切换空间

当需要切换到不同的 `spaceId` 时，UI Kit 会自动处理：

1. 检测到 `spaceId` 变化后，自动清除旧 token
2. 重新调用 `getAccessToken` 获取新空间的 token
3. 重置内部 SDK client 实例

接入方只需更新传入的 `spaceId` prop 即可：

```jsx
<SpaceDrive
  basePath={basePath}
  libraryId={libraryId}
  spaceId={currentSpaceId}  // 切换此值即可
  getAccessToken={getAccessToken}
/>
```

---

## 样式引入

UI Kit 的样式需要在宿主项目中手动引入：

```javascript
// 在入口文件中引入
import 'smh-space-drive/dist/style.css'
```

UI Kit 内部所有 UI 均为原生实现，不依赖第三方 UI 组件库。

---

## 注意事项

1. **`librarySecret` 绝不能暴露给前端**，token 的生成必须在后端完成
2. **`getAccessToken` 函数引用需保持稳定**，建议使用 `useCallback` 包裹或定义在组件外部，避免不必要的重新初始化
3. **组件销毁时调用 `clearConfig()`**，清理内部状态，防止内存泄漏和状态残留
4. **`expiresAt` 必须是毫秒级时间戳**（`Date.now()` 格式），不是秒级
5. 组件默认占满父容器的 `100%` 高度，请确保父容器有合适的高度约束

---

## 相关 SDK 文档

| SDK | npm 包名 | 说明 |
|-----|---------|------|
| 前端 SDK | [`smh-js-sdk`](https://www.npmjs.com/package/smh-js-sdk) | 前端直连 SMH API，UI Kit 内部已集成 |
| 后端 SDK | [`smh-node-sdk`](https://www.npmjs.com/package/smh-node-sdk) | 后端管理 API，接入方需在后端安装使用 |

> 更多 SMH 服务信息请参考 [腾讯云智能媒体托管文档](https://cloud.tencent.com/document/product/1339)
