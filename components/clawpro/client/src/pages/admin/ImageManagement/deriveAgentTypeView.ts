/**
 * 派生层：扁平两行视图（公共行 + 自定义行）
 *
 * 心智简化：
 *   - 公共镜像：腾讯云提供的固定列表，每个都直接启用即可，没有"先关联再启用"两步
 *     · 派生层把已存在 ImageRow（active 状态真实）+ 平台候选（虚拟未存在）合并展示
 *     · UI 拨动虚拟项的 Switch → 主文件 handler 自动添加 ImageRow + 设 active=true
 *   - 自定义镜像：保留"导入"概念（客户从自己镜像列表里挑）；导入后启用就是 Switch
 *
 * 启用规则：
 *   - 同 Agent 类型只能有一个镜像启用（active=true）
 *   - 没有 active=true 的镜像 → 类型整体未启用
 */
import { AGENT_VERSIONS, getImageMeta } from "../VersionManagement/mockData";
import type { AgentTypeKey, AgentVersionInfo } from "../VersionManagement/mockData";

// ─── 输入类型（兼容现有 ImageRow） ────────────────────────────────────
export type ImageSource = "public" | "custom";

export interface ImageRow {
  id: string;
  name: string;
  status: string;
  type: ImageSource;
  agentType: string;
  agentVersion: string;
  os: string;
  createTime: string;
  active: boolean;
}

// ─── 单条镜像在视图里的轻量表达 ───────────────────────────────────────
export interface ViewImage {
  id: string;
  name: string;
  source: ImageSource;
  agentVersion: string;
  os: string;
  createTime: string;
  /** active=true 的那条镜像 */
  isEffective: boolean;
  status: string;
  /**
   * 是否为"虚拟"项（仅公共镜像）：
   *   true  → 平台提供但企业还未实际关联（images state 里没这条 ImageRow）
   *           拨亮 Switch 时主文件需要"添加 ImageRow + 启用"
   *   false → 已存在的 ImageRow，正常启用/停用
   */
  isVirtual?: boolean;
}

// ─── 行视图（公共/自定义共用） ────────────────────────────────────────
export interface RowView {
  /** 行内是否有任何镜像（公共：含虚拟候选；自定义：仅已导入） */
  hasAny: boolean;
  /** 行级是否启用（行内有 active=true 的真实镜像） */
  isEnabled: boolean;
  /** 当前启用镜像（仅启用时有值） */
  carrierImage?: ViewImage;
  /** 行内所有镜像（公共行可能含虚拟候选） */
  allImages: ViewImage[];
  /** 没填 agentVersion 的镜像（仅自定义） */
  orphanImages: ViewImage[];
}

// ─── 整个 Agent 类型的视图 ────────────────────────────────────────────
export interface AgentTypeView {
  agentType: string;
  isSystemPreset: boolean;
  publicRow: RowView | null;
  customRow: RowView;
  enabled: {
    isEnabled: boolean;
    source: ImageSource | null;
    version: string | null;
    imageName: string | null;
  };
  /**
   * 主表行展示的"当前选中镜像"：
   *   - 优先：当前启用的镜像（active=true）
   *   - 次：公共行第一条（含虚拟推荐）
   *   - 再次：自定义行第一条
   *   - 都没有 → undefined（自定义类型且未导入）
   */
  selectedImage?: ViewImage;
}

// ─── 系统预设 agentType key → AGENT_VERSIONS 里的 key ─────────────────
const IMG_TO_VERSION_KEY: Record<string, AgentTypeKey> = {
  OpenClaw: "OpenClaw",
  HermesAgent: "Hermes",
  LightClawACE: "LightclawACE",
};

const SYSTEM_PRESET_TYPES = new Set(Object.keys(IMG_TO_VERSION_KEY));

export function isSystemPresetAgentType(agentType: string): boolean {
  return SYSTEM_PRESET_TYPES.has(agentType);
}

function findLatestPlatformVersion(
  agentType: string,
): AgentVersionInfo | undefined {
  const key = IMG_TO_VERSION_KEY[agentType];
  if (!key) return undefined;
  return AGENT_VERSIONS.filter((v) => v.agentType === key).sort((a, b) =>
    b.releaseTime.localeCompare(a.releaseTime),
  )[0];
}

function toViewImage(row: ImageRow): ViewImage {
  return {
    id: row.id,
    name: row.name,
    source: row.type,
    agentVersion: row.agentVersion,
    os: row.os,
    createTime: row.createTime,
    isEffective: row.active,
    status: row.status,
    isVirtual: false,
  };
}

// ─── 公共镜像候选 mock：腾讯云提供的标准公共镜像清单 ──────────────────
/**
 * 这些是腾讯云为该 agentType 提供的公共镜像，固定不变。
 * 实际项目中由后端接口拉取，前端只是合并展示。
 */
export interface PublicImageCandidate {
  imageId: string;
  imageName: string;
  /** 当前最新版本（来自 ClawPro 平台元数据） */
  latestVersion: string;
  latestReleaseTime: string;
}

/** 每个系统 agent 类型对应的"安全加固版"虚拟镜像 ID（8 位规范格式） */
const HARDENED_IMAGE_ID: Record<string, string> = {
  OpenClaw: "img-tcocphard",
  HermesAgent: "img-tchmshard",
  LightClawACE: "img-tclachard",
};

export function getHardenedImageId(agentType: string): string | undefined {
  return HARDENED_IMAGE_ID[agentType];
}

function buildPublicCandidates(agentType: string): PublicImageCandidate[] {
  if (!isSystemPresetAgentType(agentType)) return [];
  const versionKey = IMG_TO_VERSION_KEY[agentType];
  if (!versionKey) return [];
  const meta = getImageMeta(versionKey);
  const latest = findLatestPlatformVersion(agentType);
  if (!meta || !latest) return [];

  const list: PublicImageCandidate[] = [
    {
      imageId: meta.imageId,
      imageName: meta.imageName,
      latestVersion: latest.version,
      latestReleaseTime: latest.releaseTime,
    },
  ];

  const hardenedId = HARDENED_IMAGE_ID[agentType];
  if (hardenedId) {
    list.push({
      imageId: hardenedId,
      imageName: `${meta.imageName} · 安全加固版`,
      latestVersion: latest.version,
      latestReleaseTime: latest.releaseTime,
    });
  }

  return list;
}

// ─── 公共行构建 ───────────────────────────────────────────────────────
function buildPublicRow(
  realImages: ImageRow[],
  agentType: string,
): RowView {
  const candidates = buildPublicCandidates(agentType);
  const realIds = new Set(realImages.map((i) => i.id));

  // 已存在的真实镜像
  const realViews = realImages.map(toViewImage);

  // 虚拟候选：候选清单里、还没在 realImages 里的
  const virtualViews: ViewImage[] = candidates
    .filter((c) => !realIds.has(c.imageId))
    .map((c) => ({
      id: c.imageId,
      name: c.imageName,
      source: "public" as ImageSource,
      agentVersion: c.latestVersion,
      os: "CentOS 7.9 64位",
      // mock：虚拟公共镜像的"接入时间"用一个相对早的固定时间
      // （真实接口下，这里来自接口返回的镜像首次接入时间）
      createTime: "2025-10-01 00:00:00",
      isEffective: false,
      status: "available",
      isVirtual: true,
    }));

  // 真实镜像没在候选清单里的（边界：客户曾经手动加过非标准公共镜像），仍展示
  const allImages = [...realViews, ...virtualViews];

  // 排序：纯按导入时间倒序（新的在上，老的在下；启用项不强制置顶）
  allImages.sort((a, b) => b.createTime.localeCompare(a.createTime));

  const activeReal = realImages.find((r) => r.active);
  const carrierImage = activeReal ? toViewImage(activeReal) : undefined;

  return {
    hasAny: allImages.length > 0,
    isEnabled: !!activeReal,
    carrierImage,
    allImages,
    orphanImages: [],
  };
}

// ─── 自定义行构建 ─────────────────────────────────────────────────────
function buildCustomRow(realImages: ImageRow[]): RowView {
  if (realImages.length === 0) {
    return { hasAny: false, isEnabled: false, allImages: [], orphanImages: [] };
  }

  const orphans = realImages
    .filter((r) => !r.agentVersion?.trim())
    .map(toViewImage)
    .sort((a, b) => b.createTime.localeCompare(a.createTime));

  const versioned = realImages.filter((r) => r.agentVersion?.trim());
  const activeRow = versioned.find((r) => r.active);

  let carrier: ViewImage | undefined;
  if (activeRow) {
    carrier = toViewImage(activeRow);
  } else if (versioned.length > 0) {
    const latest = [...versioned].sort((a, b) =>
      b.createTime.localeCompare(a.createTime),
    )[0];
    if (latest) carrier = toViewImage(latest);
  }

  // 列表用 allImages：纯按导入时间倒序（新在上）
  const allSorted = [...realImages].sort((a, b) =>
    b.createTime.localeCompare(a.createTime),
  );

  return {
    hasAny: realImages.length > 0,
    isEnabled: !!activeRow,
    carrierImage: carrier,
    allImages: allSorted.map(toViewImage),
    orphanImages: orphans,
  };
}

// ─── 主派生函数 ───────────────────────────────────────────────────────
export function deriveAgentTypeView(
  allImages: ImageRow[],
  agentType: string,
): AgentTypeView {
  const isSystemPreset = isSystemPresetAgentType(agentType);

  const typeImages = allImages.filter((i) => i.agentType === agentType);
  const publicImages = typeImages.filter((i) => i.type === "public");
  const customImages = typeImages.filter((i) => i.type === "custom");

  const publicRow: RowView | null = isSystemPreset
    ? buildPublicRow(publicImages, agentType)
    : null;

  const customRow: RowView = buildCustomRow(customImages);

  // 启用摘要
  let enabled: AgentTypeView["enabled"] = {
    isEnabled: false,
    source: null,
    version: null,
    imageName: null,
  };
  if (publicRow?.isEnabled && publicRow.carrierImage) {
    enabled = {
      isEnabled: true,
      source: "public",
      version: publicRow.carrierImage.agentVersion,
      imageName: publicRow.carrierImage.name,
    };
  } else if (customRow.isEnabled && customRow.carrierImage) {
    enabled = {
      isEnabled: true,
      source: "custom",
      version: customRow.carrierImage.agentVersion,
      imageName: customRow.carrierImage.name,
    };
  }

  // 主行展示对象：优先 active；否则公共首条（含虚拟）；再否则自定义首条
  let selectedImage: ViewImage | undefined;
  if (publicRow?.carrierImage) {
    selectedImage = publicRow.carrierImage;
  } else if (customRow.carrierImage) {
    selectedImage = customRow.carrierImage;
  } else if (publicRow && publicRow.allImages.length > 0) {
    selectedImage = publicRow.allImages[0];
  } else if (customRow.allImages.length > 0) {
    selectedImage = customRow.allImages[0];
  }

  return {
    agentType,
    isSystemPreset,
    publicRow,
    customRow,
    enabled,
    selectedImage,
  };
}

// ─── 工具：相对时间 ───────────────────────────────────────────────────
export function relativeTime(datetime: string): string {
  if (!datetime) return "—";
  const ts = new Date(datetime.replace(" ", "T")).getTime();
  if (Number.isNaN(ts)) return datetime;
  const diff = Date.now() - ts;
  if (diff < 0) return "刚刚";
  const min = Math.floor(diff / 60000);
  if (min < 1) return "刚刚";
  if (min < 60) return `${min} 分钟前`;
  const hr = Math.floor(min / 60);
  if (hr < 24) return `${hr} 小时前`;
  const day = Math.floor(hr / 24);
  if (day < 30) return `${day} 天前`;
  const mon = Math.floor(day / 30);
  if (mon < 12) return `${mon} 个月前`;
  return `${Math.floor(mon / 12)} 年前`;
}

export { IMG_TO_VERSION_KEY };
