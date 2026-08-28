/**
 * 配置对比弹窗（逐实例）
 *
 * 展示每个 Agent 实例「当前配置」与「迁移至的新组织配置」的逐项对比。
 * - 平台策略类配置项会在迁移时自动更改为新组织配置；其余项仅标注是否符合。
 * - 每个实例渲染一张卡片，卡片内为一张对比表。
 *
 * 严格遵循 ClawPro 设计规范：StatusTag（green/orange soft、4px）、Typography、
 * token 配色（--text-warning / --alert-warning-bg / --cp-border 等），无硬编码 hex、无渐变。
 */
import React from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { StatusTag } from "@/components/ui/status-tag";
import { CompactText, MetaText } from "@/components/ui/Typography";
import { Check, AlertTriangle, Minus } from "lucide-react";

/** 单个配置项的对比 */
export interface ConfigCompareItem {
  /** 配置项名称，如"模型"、"技能 > 技能安装来源" */
  category: string;
  /** 当前实例配置（多行用 \n 分隔） */
  currentValue: string;
  /** 新组织配置（多行用 \n 分隔） */
  newValue: string;
  /** 是否符合新组织配置 */
  isMatch: boolean;
  /** 该项不参与是否符合检查（如技能、企业插件、企业MCP），标注「不检查」且不高亮 */
  noCheck?: boolean;
  /** 是否为平台策略类（迁移时自动更改为新组织配置，整行高亮提示） */
  isPolicy?: boolean;
  /** 当前实例配置中需高亮的整行文本（与新组织不一致的项） */
  highlights?: string[];
}

/** 单个实例的对比卡片数据 */
export interface InstanceConfigCompare {
  instanceName: string;
  instanceId: string;
  items: ConfigCompareItem[];
}

interface ConfigDiffDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** 迁移至的新组织名称（用于表头第三列 & 说明） */
  newGroupName: string;
  /** 逐实例对比数据 */
  instances: InstanceConfigCompare[];
}

/** 渲染多行单元格，命中 highlights 的整行用 --text-warning 高亮 */
function PreCell({ value, highlights }: { value: string; highlights?: string[] }) {
  const lines = value.split("\n");
  const hl = highlights ?? [];
  return (
    <div className="whitespace-pre-line text-[13px] leading-[1.5] text-[var(--text-secondary)]">
      {lines.map((line, i) => {
        const isHl = hl.includes(line);
        return (
          <div
            key={i}
            className={isHl ? "text-amber-700 font-medium" : undefined}
          >
            {line || "\u00A0"}
          </div>
        );
      })}
    </div>
  );
}

export default function ConfigDiffDialog({
  open,
  onOpenChange,
  newGroupName,
  instances,
}: ConfigDiffDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="sm:max-w-3xl max-h-[85vh] flex flex-col"
        onOpenAutoFocus={(e) => e.preventDefault()}
      >
        <DialogHeader>
          <DialogTitle>配置对比</DialogTitle>
        </DialogHeader>

        {/* 内容滚动区：仅此区域滚动，标题与底部「关闭」固定 */}
        <div className="flex-1 min-h-0 overflow-y-auto -mx-6 px-6">
          <MetaText className="leading-[1.6]">
            以下是 Agent 实例当前配置与迁移至的新组织配置的对比。
            <span className="text-[var(--text-secondary)] font-medium">平台策略</span>
            类配置项会在迁移时自动更改为新组织配置，其余项仅标注是否符合；管理员可后续到 Agent
            列表查看实例与新组织的配置对比并调整配置项。
          </MetaText>

          <div className="space-y-3.5 py-2">
          {instances.length === 0 && (
            <div className="text-center py-8 space-y-1">
              <p className="text-xs text-[var(--text-weak)]">暂无可对比的实例</p>
              <p className="text-xs text-[var(--text-weak)]">本次操作未涉及存量 Agent 实例</p>
            </div>
          )}
          {instances.map((inst) => (
            <div
              key={inst.instanceId}
              className="rounded-[4px] border border-[var(--cp-border)] overflow-hidden"
            >
              {/* 卡片头：实例名 + ID */}
              <div className="px-3.5 py-2.5 bg-[var(--bg-grey-normal)] border-b border-[var(--cp-border)]">
                <span className="text-xs font-semibold text-[var(--text-secondary)]">
                  {inst.instanceName}
                </span>
                <span className="ml-1 text-[11px] text-[var(--text-weak)]">
                  ({inst.instanceId})
                </span>
              </div>

              {/* 对比表 */}
              <table className="w-full border-collapse">
                <thead>
                  <tr className="text-left">
                    <th className="w-[22%] px-3.5 py-2 text-[12px] font-medium text-[var(--text-muted)] border-b border-[var(--cp-border)]">
                      配置项
                    </th>
                    <th className="w-[28%] px-3.5 py-2 text-[12px] font-medium text-[var(--text-muted)] border-b border-[var(--cp-border)]">
                      当前实例配置
                    </th>
                    <th className="w-[28%] px-3.5 py-2 text-[12px] font-medium text-[var(--text-muted)] border-b border-[var(--cp-border)]">
                      新组织（{newGroupName}）
                    </th>
                    <th className="w-[22%] px-3.5 py-2 text-[12px] font-medium text-[var(--text-muted)] border-b border-[var(--cp-border)]">
                      是否符合新组织配置
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {inst.items.map((item, idx) => (
                    <tr
                      key={idx}
                      className={`align-top border-b border-[var(--cp-border)] last:border-b-0 ${item.isPolicy ? "bg-[var(--alert-warning-bg)]" : ""}`}
                    >
                      <td className="px-3.5 py-2">
                        <CompactText tone="primary" className="font-medium">
                          {item.category}
                        </CompactText>
                      </td>
                      <td className="px-3.5 py-2">
                        <PreCell
                          value={item.noCheck ? "-" : item.currentValue}
                          highlights={item.noCheck ? undefined : item.highlights}
                        />
                      </td>
                      <td className="px-3.5 py-2">
                        <PreCell value={item.noCheck ? "-" : item.newValue} />
                      </td>
                      <td className="px-3.5 py-2">
                        {item.noCheck ? (
                          <StatusTag mode="soft" variant="gray" icon={<Minus />}>
                            不检查
                          </StatusTag>
                        ) : item.isMatch ? (
                          <StatusTag mode="soft" variant="green" icon={<Check />}>
                            符合
                          </StatusTag>
                        ) : (
                          <StatusTag mode="soft" variant="orange" icon={<AlertTriangle />}>
                            不符合
                          </StatusTag>
                        )}
                        {item.isPolicy && !item.isMatch && (
                          <div className="mt-1 pl-1 text-[11px] text-[var(--text-muted)]">
                            将自动更改为新组织配置
                          </div>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ))}
          </div>
        </div>

        <DialogFooter>
          <Button variant="claw-outline" onClick={() => onOpenChange(false)}>
            关闭
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ─── Mock 逐实例对比数据（用于演示，对齐原型⑧「配置对比」） ─────────────────────

/** 实例 A 的配置项模板（含差异项） */
const INSTANCE_A_ITEMS: ConfigCompareItem[] = [
  {
    category: "计费模式",
    currentValue: "按量计费",
    newValue: "包年包月",
    isMatch: false,
    highlights: ["按量计费"],
  },
  {
    category: "模型",
    currentValue: "混元 TurboS Latest\nGPT-4o",
    newValue: "混元 TurboS Latest\nDeepSeek V3",
    isMatch: false,
    highlights: ["GPT-4o"],
  },
  {
    category: "通道",
    currentValue: "微信\n企业微信",
    newValue: "微信\n企业微信",
    isMatch: true,
  },
  {
    category: "技能",
    currentValue: "canvas\nclawhub\ndiagram-maker\nhealthcheck\nmeme-maker\nnode-connect",
    newValue:
      "canvas\nclawhub\ndiagram-maker\nhealthcheck\nmeme-maker\nnode-connect\nnode-inspect-debugger\nnotion\npython-debugpy",
    isMatch: true,
    noCheck: true,
  },
  {
    category: "技能 > 技能安装来源",
    currentValue: "git.techcorp.cn/skills",
    newValue: "git.techcorp.cn/skills",
    isMatch: true,
    noCheck: true,
  },
  {
    category: "Agent 工具 > 企业插件",
    currentValue: "Jira\nGitLab CI/CD\nFigma 设计稿同步",
    newValue: "Jira",
    isMatch: true,
    noCheck: true,
  },
  {
    category: "Agent 工具 > 企业MCP",
    currentValue: "文档解析\n代码仓库MCP",
    newValue: "文档解析\nAI推理MCP",
    isMatch: true,
    noCheck: true,
  },
  { category: "记忆", currentValue: "开启 Pro 版", newValue: "开启 Pro 版", isMatch: true },
  { category: "网盘", currentValue: "开启", newValue: "开启", isMatch: true },
  {
    category: "镜像",
    currentValue: "Openclaw",
    newValue: "Openclaw\nLighthouse ACE",
    isMatch: true,
  },
  {
    category: "网络 > 私有网络与子网",
    currentValue: "vpc-gn16sgnn\nsubnet-nvupa1uw",
    newValue: "vpc-gn16sgnn\nsubnet-nvupa1uw\nsubnet-a3bc82kx",
    isMatch: true,
  },
  {
    category: "网络 > 安全组",
    currentValue: "sg-tech4n7w",
    newValue: "sg-tech4n7w",
    isMatch: true,
  },
  {
    category: "网络 > 公网",
    currentValue: "公网 IP：已分配\n计费模式：按流量计费\n带宽上限：10 Mbps",
    newValue: "公网 IP：已分配\n计费模式：按流量计费\n带宽上限：10 Mbps",
    isMatch: true,
  },
  { category: "CLS 日志服务", currentValue: "开启", newValue: "开启", isMatch: true },
  { category: "AI Agent 安全", currentValue: "开启", newValue: "开启", isMatch: true },
  {
    category: "平台策略 > 用户配额",
    currentValue:
      "单用户 Agent 数量上限：60\n单用户 Tokens 上限：2026/06/09 00:00 – 2026/06/16 00:00，按日刷新 无限制",
    newValue:
      "单用户 Agent 数量上限：90\n单用户 Tokens 上限：2026/06/16 00:00 – 2026/06/25 00:00，按日刷新 无限制",
    isMatch: false,
    isPolicy: true,
    highlights: [
      "单用户 Agent 数量上限：60",
      "单用户 Tokens 上限：2026/06/09 00:00 – 2026/06/16 00:00，按日刷新 无限制",
    ],
  },
  {
    category: "平台策略 > 模型配额",
    currentValue: "全局 Tokens 上限：2026/06/01 17:26 – 无终止，按月刷新 无限制",
    newValue: "全局 Tokens 上限：2026/06/09 17:26 – 无终止，按月刷新 无限制",
    isMatch: false,
    isPolicy: true,
    highlights: ["全局 Tokens 上限：2026/06/01 17:26 – 无终止，按月刷新 无限制"],
  },
  {
    category: "平台策略 > 功能权限",
    currentValue:
      "允许配置模型 ✓\n允许配置通道 ✓\n允许进入终端 ✓\n允许访问面板 ✓\n允许使用对话视图 ✓\n允许访问云端浏览器 ✓\n允许使用龙虾医生 ✓\n允许查看模型额度 ✓\n允许添加自定义模型 ✓",
    newValue:
      "允许配置模型 ✓\n允许配置通道 ✓\n允许进入终端 ✗\n允许访问面板 ✓\n允许使用对话视图 ✓\n允许访问云端浏览器 ✓\n允许使用龙虾医生 ✓\n允许查看模型额度 ✓\n允许添加自定义模型 ✗",
    isMatch: false,
    isPolicy: true,
    highlights: ["允许进入终端 ✓", "允许添加自定义模型 ✓"],
  },
];

/** 实例 B 的配置项模板（多数符合） */
const INSTANCE_B_ITEMS: ConfigCompareItem[] = [
  {
    category: "计费模式",
    currentValue: "包年包月",
    newValue: "包年包月",
    isMatch: true,
  },
  {
    category: "模型",
    currentValue: "混元 TurboS Latest\nDeepSeek V3",
    newValue: "混元 TurboS Latest\nDeepSeek V3",
    isMatch: true,
  },
  { category: "通道", currentValue: "企业微信", newValue: "微信\n企业微信", isMatch: true },
  {
    category: "技能",
    currentValue:
      "canvas\nclawhub\ndiagram-maker\nhealthcheck\nmeme-maker\nnode-connect\nnode-inspect-debugger\nnotion\npython-debugpy",
    newValue:
      "canvas\nclawhub\ndiagram-maker\nhealthcheck\nmeme-maker\nnode-connect\nnode-inspect-debugger\nnotion\npython-debugpy",
    isMatch: true,
    noCheck: true,
  },
  {
    category: "技能 > 技能安装来源",
    currentValue: "git.techcorp.cn/skills",
    newValue: "git.techcorp.cn/skills",
    isMatch: true,
    noCheck: true,
  },
  {
    category: "Agent 工具 > 企业插件",
    currentValue: "Jira",
    newValue: "Jira",
    isMatch: true,
    noCheck: true,
  },
  {
    category: "Agent 工具 > 企业MCP",
    currentValue: "文档解析",
    newValue: "文档解析\nAI推理MCP",
    isMatch: true,
    noCheck: true,
  },
  { category: "记忆", currentValue: "开启 Pro 版", newValue: "开启 Pro 版", isMatch: true },
  { category: "网盘", currentValue: "开启", newValue: "开启", isMatch: true },
  {
    category: "镜像",
    currentValue: "Hermes Agent",
    newValue: "Openclaw\nLighthouse ACE",
    isMatch: false,
    highlights: ["Hermes Agent"],
  },
  {
    category: "网络 > 私有网络与子网",
    currentValue: "vpc-gn16sgnn\nsubnet-nvupa1uw",
    newValue: "vpc-gn16sgnn\nsubnet-nvupa1uw\nsubnet-a3bc82kx",
    isMatch: true,
  },
  {
    category: "网络 > 安全组",
    currentValue: "sg-tech4n7w",
    newValue: "sg-tech4n7w",
    isMatch: true,
  },
  {
    category: "网络 > 公网",
    currentValue: "公网 IP：已分配\n计费模式：按流量计费\n带宽上限：10 Mbps",
    newValue: "公网 IP：已分配\n计费模式：按流量计费\n带宽上限：10 Mbps",
    isMatch: true,
  },
  { category: "CLS 日志服务", currentValue: "开启", newValue: "开启", isMatch: true },
  { category: "AI Agent 安全", currentValue: "开启", newValue: "开启", isMatch: true },
  {
    category: "平台策略 > 用户配额",
    currentValue:
      "单用户 Agent 数量上限：60\n单用户 Tokens 上限：2026/06/09 00:00 – 2026/06/16 00:00，按日刷新 无限制",
    newValue:
      "单用户 Agent 数量上限：90\n单用户 Tokens 上限：2026/06/16 00:00 – 2026/06/25 00:00，按日刷新 无限制",
    isMatch: false,
    isPolicy: true,
    highlights: [
      "单用户 Agent 数量上限：60",
      "单用户 Tokens 上限：2026/06/09 00:00 – 2026/06/16 00:00，按日刷新 无限制",
    ],
  },
  {
    category: "平台策略 > 模型配额",
    currentValue: "全局 Tokens 上限：2026/06/01 17:26 – 无终止，按月刷新 无限制",
    newValue: "全局 Tokens 上限：2026/06/09 17:26 – 无终止，按月刷新 无限制",
    isMatch: false,
    isPolicy: true,
    highlights: ["全局 Tokens 上限：2026/06/01 17:26 – 无终止，按月刷新 无限制"],
  },
  {
    category: "平台策略 > 功能权限",
    currentValue:
      "允许配置模型 ✓\n允许配置通道 ✓\n允许进入终端 ✓\n允许访问面板 ✓\n允许使用对话视图 ✓\n允许访问云端浏览器 ✗\n允许使用龙虾医生 ✓\n允许查看模型额度 ✓\n允许添加自定义模型 ✓",
    newValue:
      "允许配置模型 ✓\n允许配置通道 ✓\n允许进入终端 ✗\n允许访问面板 ✓\n允许使用对话视图 ✓\n允许访问云端浏览器 ✓\n允许使用龙虾医生 ✓\n允许查看模型额度 ✓\n允许添加自定义模型 ✗",
    isMatch: false,
    isPolicy: true,
    highlights: ["允许进入终端 ✓", "允许访问云端浏览器 ✗", "允许添加自定义模型 ✓"],
  },
];

/**
 * 按"上一步实际选中/影响的实例"构建逐实例对比卡片数据。
 * 卡片数量与实例名/ID 来自真实选中实例；逐项配置对比为演示用 mock（按序交替模板）。
 */
export function buildMockInstanceCompare(
  instances: { instanceName: string; instanceId: string }[],
): InstanceConfigCompare[] {
  return instances.map((inst, idx) => ({
    instanceName: inst.instanceName,
    instanceId: inst.instanceId,
    items: idx % 2 === 0 ? INSTANCE_A_ITEMS : INSTANCE_B_ITEMS,
  }));
}

/** 静态 mock（兜底/演示用，对应原型⑧两个实例） */
export const MOCK_INSTANCE_COMPARE: InstanceConfigCompare[] = buildMockInstanceCompare([
  { instanceName: "代码评审助手", instanceId: "agt-9f3a1c" },
  { instanceName: "接口测试机器人", instanceId: "agt-7b22e0" },
]);
