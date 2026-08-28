/**
 * publicImageRecords - ClawPro 公共镜像清单 + 历史更新记录
 *
 * 数据源：ClawPro 公共镜像更新记录.xlsx（产研真实历史发版）
 * 字段约定：
 *   - 每个公共镜像有一个 imageId（如 img-idzg74s9），下面挂多条更新记录
 *   - 每条更新记录是 firstRelease（首次上线）或 update（版本更新）两种之一
 *   - releaseDate 精确到日（yyyy-MM-dd）
 *   - update 类型只显示"更新到 vX.Y.Z 版本"，不展示更详细的内容
 *
 * 业务定义：
 *   - 镜像的"当前版本" = 该镜像最新一条记录的版本
 *   - "首次上线"会展示完整版本说明，"更新"只展示版本号
 */

export interface PublicImageRecord {
  /** 记录类型 */
  type: "firstRelease" | "update";
  /** 该次更新对应的版本号 */
  version: string;
  /** 发布日期（yyyy-MM-dd） */
  releaseDate: string;
  /** 该次发布的关键变更说明（可选，用于版本更新记录卡片二级展示） */
  description?: string;
}

export interface PublicImageMeta {
  /** 镜像 ID（短 ID，与 ImageManagement 里的 ImageRow.id 一致） */
  imageId: string;
  /** 镜像名（人类可读） */
  imageName: string;
  /** 操作系统 */
  os: string;
  /** 该镜像服务的 Agent 类型（与 ImageManagement 内 agentType key 一致） */
  agentType: "OpenClaw" | "HermesAgent" | "LightClawACE";
  /** 历史更新记录（按发布时间倒序：最新在前） */
  records: PublicImageRecord[];
}

export const PUBLIC_IMAGE_METAS: PublicImageMeta[] = [
  // ─── OpenClaw on Ubuntu 24.04 ────────────────────────────
  {
    imageId: "img-idzg74s9",
    imageName: "OpenClaw on Ubuntu 24.04",
    os: "Ubuntu 24.04 x86_64",
    agentType: "OpenClaw",
    records: [
      { type: "update",       version: "2026.4.23", releaseDate: "2026-05-11", description: "增强终端响应性能，修复偶发崩溃问题" },
      { type: "update",       version: "2026.4.23", releaseDate: "2026-04-30", description: "升级 memory-tencentdb 插件到 0.2.3" },
      { type: "update",       version: "2026.4.15", releaseDate: "2026-04-23", description: "Node 运行时升级到 22.22（该版本已回滚）" },
      { type: "update",       version: "2026.4.11", releaseDate: "2026-04-16", description: "通道适配器优化，提升大文件传输稳定性" },
      { type: "update",       version: "2026.4.2",  releaseDate: "2026-04-11", description: "接入 Mimo 模型，新增多模态调用支持" },
      { type: "update",       version: "2026.3.28", releaseDate: "2026-04-04", description: "鉴权链路重构，支持企业 SSO 单点登录" },
      { type: "update",       version: "2026.3.13", releaseDate: "2026-03-22", description: "插件子系统初版迭代修复 + 日志格式统一" },
      { type: "update",       version: "2026.3.13", releaseDate: "2026-03-21", description: "插件子系统初版上线" },
      { type: "firstRelease", version: "2026.3.8",  releaseDate: "2026-03-13", description: "首次上线，提供基础 Agent 能力" },
    ],
  },
  // ─── Hermes Agent on Ubuntu 24.04 ────────────────────────
  {
    imageId: "img-al484uhr",
    imageName: "Hermes Agent on Ubuntu 24.04",
    os: "Ubuntu 24.04 x86_64",
    agentType: "HermesAgent",
    records: [
      { type: "update",       version: "v0.12.0", releaseDate: "2026-05-09", description: "增强插件子系统，开放更多企业自建插件接入" },
      { type: "update",       version: "v0.10.0", releaseDate: "2026-04-21", description: "Gateway 启动修复，新增健康检查端点" },
      { type: "firstRelease", version: "v0.9.0",  releaseDate: "2026-04-16", description: "首次上线，提供 Hermes 网关与基础 Agent 能力" },
    ],
  },
  // ─── OpenClaw on TencentOS Server 4 ──────────────────────
  {
    imageId: "img-nmg7pw1r",
    imageName: "OpenClaw on TencentOS Server 4",
    os: "TencentOS Server 4 x86_64",
    agentType: "OpenClaw",
    records: [
      { type: "update",       version: "2026.4.23", releaseDate: "2026-04-30", description: "升级 memory-tencentdb 插件到 0.2.3" },
      { type: "firstRelease", version: "2026.4.11", releaseDate: "2026-04-16", description: "首次上线，适配 TencentOS Server 4 内核" },
    ],
  },
  // ─── Hermes Agent on TencentOS Server 4 ──────────────────
  {
    imageId: "img-ppz9gfjn",
    imageName: "Hermes Agent on TencentOS Server 4",
    os: "TencentOS Server 4 x86_64",
    agentType: "HermesAgent",
    records: [
      { type: "update",       version: "v0.12.0", releaseDate: "2026-05-09", description: "增强插件子系统，开放更多企业自建插件接入" },
      { type: "update",       version: "v0.10.0", releaseDate: "2026-04-22", description: "Gateway 启动修复，新增健康检查端点" },
      { type: "firstRelease", version: "v0.9.0",  releaseDate: "2026-04-16", description: "首次上线，适配 TencentOS Server 4 内核" },
    ],
  },
  // ─── OpenClaw on TencentOS Server 4 For Tencent ──────────
  {
    imageId: "img-pf18atu9",
    imageName: "OpenClaw on TencentOS Server 4 For Tencent",
    os: "TencentOS Server 4 (TKernel5) x86_64",
    agentType: "OpenClaw",
    records: [
      { type: "update",       version: "2026.4.23", releaseDate: "2026-04-30", description: "升级 memory-tencentdb 插件到 0.2.3" },
      { type: "update",       version: "2026.4.11", releaseDate: "2026-04-16", description: "通道适配器优化，提升大文件传输稳定性" },
      { type: "update",       version: "2026.4.2",  releaseDate: "2026-04-11", description: "接入 Mimo 模型，新增多模态调用支持" },
      { type: "firstRelease", version: "2026.3.13", releaseDate: "2026-04-04", description: "首次上线，适配 TKernel5 内核" },
    ],
  },
  // ─── LightClaw ACE on TencentOS Server 4 ─────────────────
  {
    imageId: "img-0dvlda3b",
    imageName: "LightClaw ACE on TencentOS Server 4",
    os: "TencentOS Server 4 x86_64",
    agentType: "LightClawACE",
    records: [
      { type: "update",       version: "v0.1.8", releaseDate: "2026-04-29", description: "多租户隔离修复，提升并发场景下的稳定性" },
      { type: "update",       version: "v0.1.5", releaseDate: "2026-04-27", description: "镜像精简，启动速度提升 30%" },
      { type: "firstRelease", version: "v0.1.1", releaseDate: "2026-04-16", description: "首次上线，提供 LightClaw ACE 轻量级 Agent" },
    ],
  },
];

// ─── 派生数据 ───────────────────────────────────────────────

/** 扁平化的更新记录（含镜像信息），按时间倒序 */
export interface FlatUpdateRecord extends PublicImageRecord {
  imageId: string;
  imageName: string;
  agentType: "OpenClaw" | "HermesAgent" | "LightClawACE";
  agentTypeLabel: string;
}

const AGENT_TYPE_LABEL_MAP: Record<string, string> = {
  OpenClaw: "OpenClaw",
  HermesAgent: "Hermes Agent",
  LightClawACE: "LightClaw ACE",
};

export function buildFlatUpdateRecords(): FlatUpdateRecord[] {
  const list: FlatUpdateRecord[] = [];
  for (const m of PUBLIC_IMAGE_METAS) {
    for (const r of m.records) {
      list.push({
        ...r,
        imageId: m.imageId,
        imageName: m.imageName,
        agentType: m.agentType,
        agentTypeLabel: AGENT_TYPE_LABEL_MAP[m.agentType] ?? m.agentType,
      });
    }
  }
  return list.sort((a, b) => b.releaseDate.localeCompare(a.releaseDate));
}

/** 取某个镜像的"当前版本"（其历史中最新一条） */
export function getCurrentVersion(imageId: string): string | null {
  const meta = PUBLIC_IMAGE_METAS.find((m) => m.imageId === imageId);
  if (!meta || meta.records.length === 0) return null;
  // records 已经按时间倒序
  return meta.records[0].version;
}

// ─── 兼容导出（原 UpdateRecordSidebar.tsx 已废弃，相关工具迁移至此） ───

/** @alias FlatUpdateRecord — 兼容历史命名（UpdateRecordsDrawer 使用） */
export type UpdateRecord = FlatUpdateRecord;

/** @alias buildFlatUpdateRecords — 兼容历史命名 */
export const buildUpdateRecords = buildFlatUpdateRecords;

/**
 * 统一展示版本号：
 *   - 已经带 v 前缀的（如 "v0.10.0"）保持不变
 *   - 不带 v 前缀的（如 "2026.4.23"）补一个 v
 */
export function formatVersion(version: string): string {
  if (!version) return "";
  return version.startsWith("v") || version.startsWith("V") ? version : `v${version}`;
}
