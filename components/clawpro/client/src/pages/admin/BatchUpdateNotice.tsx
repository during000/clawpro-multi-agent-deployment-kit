/**
 * BatchUpdateNotice - Agent 列表「批量更新」按钮上的红点 + 浮窗
 *
 * 触发条件：
 *   - 存在某个 agent 类型，其"启用镜像版本" ≠ "存量实例版本"
 *   - 数据来源：
 *     · 启用版本读 localStorage admin_images_v3（与 ImageManagement 保持一致）
 *     · 实例版本来自 props.claws
 *
 * 浮窗内容：
 *   - 列出所有"有版本不一致的实例"的 agent 类型
 *   - 每行两个动作：[推送提醒]（写 upgradePushStore）/ [批量升级]（关闭浮窗 + 触发外部已有的批量升级流程）
 */
import { useEffect, useMemo, useState } from "react";
import { Megaphone, RotateCcw, Info } from "lucide-react";
import { Button } from "@/components/ui/button";
import { StatusTag } from "@/components/ui/status-tag";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { toast } from "sonner";
import {
  listActivePushes,
  setActivePush,
  clearActivePush,
  type ActivePush,
  compareVersion,
} from "@/lib/upgradePushStore";

// ─── 类型 ───────────────────────────────────────────
interface ImageRowLite {
  id?: string;
  agentType: string;
  agentVersion: string;
  active: boolean;
  type: "public" | "custom";
  name: string;
}

interface ClawLite {
  id: string;
  name: string;
  agentType: string;       // OpenClaw / Hermes / LightclawACE
  version: string;
}

/** Claw.agentType → ImageManagement 内 agentType key */
const CLAW_TO_IMAGE_AGENT_TYPE: Record<string, string> = {
  OpenClaw: "OpenClaw",
  Hermes: "HermesAgent",
  LightclawACE: "LightClawACE",
  LocalAgent: "LocalAgent",
};

const AGENT_TYPE_DISPLAY: Record<string, string> = {
  OpenClaw: "OpenClaw",
  HermesAgent: "Hermes Agent",
  LightClawACE: "LightClaw ACE",
  LocalAgent: "本地 Agent",
};

export interface OutdatedTypeStat {
  /** ImageManagement 内的 agent 类型 key（OpenClaw / HermesAgent / LightClawACE） */
  agentType: string;
  /** 展示名 */
  agentTypeLabel: string;
  /** 启用版本 */
  enabledVersion: string;
  /** 启用镜像 id（用于在版本更新记录侧边栏精确命中时间线卡片） */
  enabledImageId?: string;
  /** 启用镜像名 */
  enabledImageName: string;
  /** 启用镜像来源 */
  imageSource: "public" | "custom";
  /** 旧版本实例数量 */
  outdatedCount: number;
}

// ─── 公开数据 hook（也供主组件红点使用） ─────────────
export function useOutdatedTypes(claws?: ClawLite[]): OutdatedTypeStat[] {
  const [tick, setTick] = useState(0);

  // 监听 ImageManagement 修改启用镜像（admin_images_v3）
  useEffect(() => {
    const onStorage = (e: StorageEvent) => {
      if (e.key === "admin_images_v3" || e.key === null) setTick((t) => t + 1);
    };
    window.addEventListener("storage", onStorage);
    return () => window.removeEventListener("storage", onStorage);
  }, []);

  return useMemo(() => {
    void tick;
    let images: ImageRowLite[] = [];
    try {
      const raw = localStorage.getItem("admin_images_v3");
      if (raw) images = JSON.parse(raw) as ImageRowLite[];
    } catch {
      /* ignore */
    }

    // 兜底：如果用户从未打开过 ImageManagement，localStorage 为空——
    // 用与 ImageManagement 一致的默认启用版本，保证红点可见
    if (images.length === 0) {
      images = [
        { id: "img-idzg74s9", agentType: "OpenClaw",     agentVersion: "2026.4.23", active: true, type: "public", name: "OpenClaw on Ubuntu 24.04" },
        { id: "img-al484uhr", agentType: "HermesAgent",  agentVersion: "0.12.0",    active: true, type: "public", name: "Hermes Agent on Ubuntu 24.04" },
        { id: "img-0dvlda3b", agentType: "LightClawACE", agentVersion: "0.1.8",     active: true, type: "public", name: "LightClaw ACE on TencentOS Server 4" },
      ];
    }

    // 没传 claws 时，用与 OpenClawMonitor 一致的 mock 实例统计
    const effectiveClaws = claws ?? FALLBACK_CLAWS;

    // 找出每个类型的启用版本
    const activeMap = new Map<string, ImageRowLite>();
    for (const i of images) {
      if (i.active) activeMap.set(i.agentType, i);
    }

    // 按 agent 类型聚合实例
    const result: OutdatedTypeStat[] = [];
    for (const [imgAgentType, image] of Array.from(activeMap.entries())) {
      // 找到所有 claws 中映射到此类型的实例
      const matchedInstances = effectiveClaws.filter((c) => {
        const mapped = CLAW_TO_IMAGE_AGENT_TYPE[c.agentType] ?? c.agentType;
        return mapped === imgAgentType;
      });
      if (matchedInstances.length === 0) continue;

      // 旧版本实例 = 实例版本 < 启用版本
      const outdated = matchedInstances.filter(
        (c) => compareVersion(c.version || "", image.agentVersion) < 0,
      );
      if (outdated.length === 0) continue;

      result.push({
        agentType: imgAgentType,
        agentTypeLabel: AGENT_TYPE_DISPLAY[imgAgentType] ?? imgAgentType,
        enabledVersion: image.agentVersion,
        enabledImageId: image.id,
        enabledImageName: image.name,
        imageSource: image.type,
        outdatedCount: outdated.length,
      });
    }
    return result.sort((a, b) => b.outdatedCount - a.outdatedCount);
  }, [claws, tick]);
}

// ─── 兜底实例 mock：当外部不传 claws 时使用（与 OpenClawMonitor MOCK_CLAWS 量级一致） ───
const FALLBACK_CLAWS: ClawLite[] = [
  // OpenClaw（12 个旧版本实例：版本 < 2026.4.23）
  { id: "fb-oc-1",  name: "Alice 的技术助手",   agentType: "OpenClaw", version: "2026.3.28" },
  { id: "fb-oc-2",  name: "Dave 的代码助手",    agentType: "OpenClaw", version: "2026.3.28" },
  { id: "fb-oc-3",  name: "Frank 的数据助手",   agentType: "OpenClaw", version: "2026.4.2"  },
  { id: "fb-oc-4",  name: "Henry 的销售助手",   agentType: "OpenClaw", version: "2026.3.28" },
  { id: "fb-oc-5",  name: "Jack 的会议助手",    agentType: "OpenClaw", version: "2026.3.28" },
  { id: "fb-oc-6",  name: "Leo 的项目助手",     agentType: "OpenClaw", version: "2026.4.2"  },
  { id: "fb-oc-7",  name: "Noah 的分析助手",    agentType: "OpenClaw", version: "2026.3.28" },
  { id: "fb-oc-8",  name: "Peter 的财务助手",   agentType: "OpenClaw", version: "2026.3.28" },
  { id: "fb-oc-9",  name: "Rachel 的 HR 助手",  agentType: "OpenClaw", version: "2026.3.28" },
  { id: "fb-oc-10", name: "Tina 的客服助手",    agentType: "OpenClaw", version: "2026.3.28" },
  { id: "fb-oc-11", name: "Victor 的技术助手",  agentType: "OpenClaw", version: "2026.3.28" },
  { id: "fb-oc-12", name: "GPULab 助手",        agentType: "OpenClaw", version: "2026.4.2"  },
  // Hermes（5 个旧版本：< v0.12.0）
  { id: "fb-h-1", name: "Bob 工作助手",   agentType: "Hermes", version: "v0.10.0" },
  { id: "fb-h-2", name: "Eve 的写作助手", agentType: "Hermes", version: "v0.9.0"  },
  { id: "fb-h-3", name: "Ivy 的客服助手", agentType: "Hermes", version: "v0.10.0" },
  { id: "fb-h-4", name: "Mia 的新助手",   agentType: "Hermes", version: "v0.9.0"  },
  { id: "fb-h-5", name: "Quinn 的法务助手", agentType: "Hermes", version: "v0.10.0" },
  // LightClaw ACE（3 个旧版本：< v0.1.8）
  { id: "fb-l-1", name: "Carol 的研究助手",     agentType: "LightclawACE", version: "v0.1.5" },
  { id: "fb-l-2", name: "Olivia 的运营助手",    agentType: "LightclawACE", version: "v0.1.1" },
  { id: "fb-l-3", name: "Sam 的产品助手",        agentType: "LightclawACE", version: "v0.1.5" },
];

// ─── 红点指示器（仅展示，无交互） ────────────────────
export function HasOutdatedIndicator({ outdated }: { outdated: OutdatedTypeStat[] }) {
  if (outdated.length === 0) return null;
  return (
    <span
      className="absolute -top-1 -right-1 w-2.5 h-2.5 bg-red-500 rounded-full ring-2 ring-white pointer-events-none"
      aria-label="有镜像更新可推送"
    />
  );
}

// ─── 提醒 Popover 主体 ─────────────────────────────
interface Props {
  outdated: OutdatedTypeStat[];
  /** 用户点击「批量升级」时的外部回调（关闭 popover 后由外部打开批量升级流程） */
  onBatchUpgrade?: (agentType: string) => void;
  /** 触发器内容（默认 = 红色感叹号小按钮，可被外部覆盖为整个"批量更新"按钮） */
  children?: React.ReactNode;
}

export default function BatchUpdateNotice({
  outdated,
  onBatchUpgrade,
  children,
}: Props) {
  const [open, setOpen] = useState(false);

  // 订阅活跃推送
  const [activePushes, setActivePushes] = useState<ActivePush[]>(() => listActivePushes());
  useEffect(() => {
    const refresh = () => setActivePushes(listActivePushes());
    window.addEventListener("upgrade-push-changed", refresh);
    window.addEventListener("storage", refresh);
    return () => {
      window.removeEventListener("upgrade-push-changed", refresh);
      window.removeEventListener("storage", refresh);
    };
  }, []);

  const findPush = (agentType: string, version: string): ActivePush | undefined =>
    activePushes.find((p) => p.agentType === agentType && p.version === version);

  const handlePush = (item: OutdatedTypeStat) => {
    const push: ActivePush = {
      agentType: item.agentType,
      agentTypeLabel: item.agentTypeLabel,
      version: item.enabledVersion,
      imageName: item.enabledImageName,
      imageSource: item.imageSource,
      pushedAt: nowStr(),
      pushedBy: "alice@acompany.com",
      message: `管理员推荐更新到 v${item.enabledVersion}`,
    };
    setActivePush(push);
    toast.success(
      `已向「${item.agentTypeLabel}」的 ${item.outdatedCount} 个旧版本 Agent 推送更新提醒`,
    );
  };

  const handleRevoke = (item: OutdatedTypeStat) => {
    clearActivePush(item.agentType);
    toast.success(`已撤回「${item.agentTypeLabel}」的推送提醒`);
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>{children}</PopoverTrigger>
      <PopoverContent align="end" sideOffset={8} className="w-[360px] p-0">
        <div className="px-5 py-4 border-b border-gray-200">
          <div className="flex items-center gap-2 text-[14px] font-semibold text-[#020617]">
            <Megaphone className="w-4 h-4 text-[#1447E6]" />
            镜像有新版本
          </div>
          <p className="text-[12px] text-[#737373] mt-1 leading-relaxed">
            以下 Agent 类型的存量 Agent 运行了旧版本，可推送提醒用户更新
          </p>
        </div>

        {outdated.length === 0 ? (
          <div className="py-10 text-center text-[13px] text-[#a3a3a3]">
            所有 Agent 都已运行最新启用版本
          </div>
        ) : (
          <ul className="max-h-[420px] overflow-y-auto divide-y divide-gray-200">
            {outdated.map((item) => {
              const push = findPush(item.agentType, item.enabledVersion);
              return (
                <li key={item.agentType} className="px-5 py-4">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className="text-[14px] font-medium text-[#020617]">
                        {item.agentTypeLabel}
                      </span>
                      <span className="text-[12px] text-[#1447E6] font-mono tabular-nums">
                        v{item.enabledVersion}
                      </span>
                      <StatusTag mode="fill" variant="blue">新版本上线</StatusTag>
                    </div>
                    <p className="text-[12px] text-[#737373] mt-1">
                      有 <span className="font-medium text-[#020617]">{item.outdatedCount}</span> 个 Agent 运行旧版本
                    </p>
                    {push && (
                      <div className="mt-2 flex items-center gap-2 flex-wrap">
                        <StatusTag mode="fill" variant="blue">正在提醒用户更新</StatusTag>
                        <button
                          onClick={() => handleRevoke(item)}
                          className="text-[12px] text-[#737373] hover:text-[#020617] transition-colors"
                        >
                          撤回
                        </button>
                      </div>
                    )}
                  </div>
                  {!push && (
                    <div className="mt-3 flex items-center justify-start">
                      <Button
                        size="sm"
                        variant="outline"
                        className="h-8 px-3 text-[13px]"
                        onClick={() => handlePush(item)}
                      >
                        推送提醒
                      </Button>
                    </div>
                  )}
                </li>
              );
            })}
          </ul>
        )}

        <div className="px-5 py-3 border-t border-gray-200 flex items-start gap-2">
          <Info className="w-3.5 h-3.5 text-[#a3a3a3] mt-0.5 shrink-0" />
          <p className="text-[12px] text-[#737373] leading-relaxed">
            "推送提醒"是软通知，用户在用户端会看到更新建议，自行决定更新时机
          </p>
        </div>
      </PopoverContent>
    </Popover>
  );
}

function nowStr(): string {
  const d = new Date();
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`;
}
