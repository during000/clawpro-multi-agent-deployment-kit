/**
 * 实例与组织配置对比抽屉（左右双栏卡片镜像版）
 *
 * 从 Agent 列表「组织」列的眼睛入口打开。
 * 逐配置项一张卡片，沿用 Agent 详情抽屉的卡片样式平铺：
 *   - 卡片左栏：实例当前配置
 *   - 卡片右栏：组织配置（与左栏一一对应镜像）
 *   - 卡片右上角：标记「符合 / 不符合」组织配置
 * 配置项严格对齐原型 instance-handling-mockup.html 视图⑧「配置对比」。
 */
import {
  Drawer,
  DrawerClose,
  DrawerContent,
  DrawerBody,
  DrawerHeader,
  DrawerTitle,
} from "@/components/ui/drawer";
import { Button } from "@/components/ui/button";
import { SurfaceInner } from "@/components/ui/Surface";
import { StatusTag } from "@/components/ui/status-tag";
import { CodeText, MetaText, PanelTitle } from "@/components/ui/Typography";
import { Check, AlertTriangle, ExternalLink, X } from "lucide-react";

/** 配置值的一行；diff=true 表示与组织配置不一致，需高亮。 */
export interface CompareLine {
  text: string;
  diff?: boolean;
}

/** 一个配置项的对比数据（实例当前配置 ↔ 组织配置）。 */
export interface CompareConfigItem {
  /** 配置项名称，如「模型」「Agent 工具 > 企业插件」 */
  category: string;
  /** 实例当前配置（多行） */
  instanceLines: CompareLine[];
  /** 组织配置（多行） */
  orgLines: CompareLine[];
  /** 是否符合组织配置 */
  isSame: boolean;
  /** 平台策略类配置项：迁移时将自动更改为新组织配置 */
  autoApply?: boolean;
}

interface ConfigCompareDrawerProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** 实例名称 */
  instanceName: string;
  /** 实例 ID */
  instanceId: string;
  /** 实例所属组织名称 */
  groupName: string;
  /** 是否整体符合组织配置 */
  isMatch: boolean;
  /** 逐项配置对比数据 */
  items: CompareConfigItem[];
}

/** 单栏配置值渲染：多行 + diff 行高亮 */
function ValueColumn({
  label,
  lines,
}: {
  label: string;
  lines: CompareLine[];
}) {
  return (
    <div className="px-4 py-3 min-w-0">
      <div className="text-[11px] text-[var(--text-weak)] mb-1.5">{label}</div>
      <div className="space-y-0.5">
        {lines.map((line, i) => (
          <div
            key={i}
            className={`text-sm leading-relaxed break-words ${
              line.diff
                ? "text-[var(--text-warning)] font-medium"
                : "text-[var(--text-secondary)]"
            }`}
          >
            {line.text}
          </div>
        ))}
      </div>
    </div>
  );
}

export default function ConfigCompareDrawer({
  open,
  onOpenChange,
  instanceName,
  instanceId,
  groupName,
  isMatch,
  items,
}: ConfigCompareDrawerProps) {
  const mismatchCount = items.filter((i) => !i.isSame).length;

  return (
    <Drawer open={open} onOpenChange={onOpenChange} direction="right">
      <DrawerContent className="data-[vaul-drawer-direction=right]:w-[960px] data-[vaul-drawer-direction=right]:sm:max-w-none max-w-[calc(100vw-24px)] h-full rounded-none bg-background p-0">
        {/* 抽屉头 */}
        <DrawerHeader className="flex flex-row items-center justify-between gap-4 p-4 bg-background text-left">
          <DrawerTitle asChild>
            <PanelTitle as="h2">实例与组织配置对比</PanelTitle>
          </DrawerTitle>
          <DrawerClose asChild>
            <Button
              size="sm"
              variant="ghost"
              className="h-7 w-7 p-0 text-[var(--text-title)] hover:text-[var(--cp-brand-black)]"
              aria-label="关闭"
            >
              <X className="w-4 h-4" />
            </Button>
          </DrawerClose>
        </DrawerHeader>

        <DrawerBody>
          <div className="p-4 space-y-5">
            {/* 实例身份信息（与 Agent 详情抽屉一致） */}
            <div className="min-w-0 space-y-1.5">
              <PanelTitle as="div" className="truncate leading-tight">{instanceName}</PanelTitle>
              <div className="flex items-center gap-2">
                <CodeText>{instanceId}</CodeText>
                <MetaText
                  as="button"
                  tone="brand"
                  className="inline-flex items-center gap-0.5 whitespace-nowrap hover:text-[var(--text-brand)]"
                  onClick={() => window.open(`https://console.cloud.tencent.com/cvm/instance/detail?rid=1&id=${instanceId}`, "_blank")}
                >
                  去腾讯云控制台管理
                  <ExternalLink className="w-3 h-3" />
                </MetaText>
              </div>
            </div>

            {/* 总体结论 */}
            <SurfaceInner className="px-4 py-3 flex items-center justify-between gap-3">
              <div className="flex flex-col min-w-0">
                <span className="text-sm font-semibold text-[var(--text-title)] leading-tight">组织配置</span>
                <span className="text-xs text-[var(--text-weak)] leading-tight mt-0.5 truncate">{groupName}</span>
              </div>
              {isMatch ? (
                <StatusTag mode="soft" variant="green" icon={<Check />}>符合组织配置</StatusTag>
              ) : (
                <StatusTag mode="soft" variant="orange" icon={<AlertTriangle />}>
                  {mismatchCount} 项不符合组织配置
                </StatusTag>
              )}
            </SurfaceInner>

            {/* 逐项卡片：每张卡片左右双栏镜像（实例当前配置 ↔ 组织配置） */}
            <div className="space-y-4">
              {items.map((item, idx) => (
                <div key={idx}>
                  {/* 配置项标题 + 符合/不符合（沿用详情抽屉「MetaText 标题 + 卡片」样式） */}
                  <div className="flex items-center justify-between mb-2 gap-3">
                    <MetaText as="div">{item.category}</MetaText>
                    {item.isSame ? (
                      <StatusTag mode="soft" variant="green" icon={<Check />}>符合</StatusTag>
                    ) : (
                      <StatusTag mode="soft" variant="orange" icon={<AlertTriangle />}>不符合</StatusTag>
                    )}
                  </div>
                  <SurfaceInner className="overflow-hidden">
                    <div className="grid grid-cols-2 divide-x divide-[var(--cp-border)]">
                      <ValueColumn label="实例当前配置" lines={item.instanceLines} />
                      <ValueColumn label={`组织配置（${groupName}）`} lines={item.orgLines} />
                    </div>
                    {item.autoApply && !item.isSame && (
                      <div className="px-4 py-2 border-t border-[var(--cp-border)] bg-[var(--bg-grey-normal)] text-xs text-[var(--text-muted)]">
                        平台策略类配置项将在迁移时自动更改为新组织配置
                      </div>
                    )}
                  </SurfaceInner>
                </div>
              ))}
            </div>
          </div>
        </DrawerBody>
      </DrawerContent>
    </Drawer>
  );
}

// ─── Mock 配置对比数据（严格对齐原型 instance-handling-mockup.html 视图⑧） ───

/** 不符合实例的逐项对比（取原型「代码评审助手」一例） */
const MOCK_COMPARE_ITEMS_MISMATCH: CompareConfigItem[] = [
  {
    category: "模型",
    instanceLines: [{ text: "混元 TurboS Latest" }, { text: "GPT-4o", diff: true }],
    orgLines: [{ text: "混元 TurboS Latest" }, { text: "DeepSeek V3" }],
    isSame: false,
  },
  {
    category: "通道",
    instanceLines: [{ text: "微信" }, { text: "企业微信" }],
    orgLines: [{ text: "微信" }, { text: "企业微信" }],
    isSame: true,
  },
  {
    category: "技能",
    instanceLines: [
      { text: "canvas" }, { text: "clawhub" }, { text: "diagram-maker" },
      { text: "healthcheck" }, { text: "meme-maker" }, { text: "node-connect" },
    ],
    orgLines: [
      { text: "canvas" }, { text: "clawhub" }, { text: "diagram-maker" },
      { text: "healthcheck" }, { text: "meme-maker" }, { text: "node-connect" },
      { text: "node-inspect-debugger" }, { text: "notion" }, { text: "python-debugpy" },
    ],
    isSame: true,
  },
  {
    category: "技能 > 技能安装来源",
    instanceLines: [{ text: "git.techcorp.cn/skills" }],
    orgLines: [{ text: "git.techcorp.cn/skills" }],
    isSame: true,
  },
  {
    category: "Agent 工具 > 企业插件",
    instanceLines: [{ text: "Jira" }, { text: "GitLab CI/CD", diff: true }, { text: "Figma 设计稿同步", diff: true }],
    orgLines: [{ text: "Jira" }],
    isSame: false,
  },
  {
    category: "Agent 工具 > 企业MCP",
    instanceLines: [{ text: "文档解析" }, { text: "代码仓库MCP", diff: true }],
    orgLines: [{ text: "文档解析" }, { text: "AI推理MCP" }],
    isSame: false,
  },
  {
    category: "记忆",
    instanceLines: [{ text: "开启 Pro 版" }],
    orgLines: [{ text: "开启 Pro 版" }],
    isSame: true,
  },
  {
    category: "网盘",
    instanceLines: [{ text: "开启" }],
    orgLines: [{ text: "开启" }],
    isSame: true,
  },
  {
    category: "镜像",
    instanceLines: [{ text: "Openclaw" }],
    orgLines: [{ text: "Openclaw" }, { text: "Lighthouse ACE" }],
    isSame: true,
  },
  {
    category: "网络 > 私有网络与子网",
    instanceLines: [{ text: "vpc-gn16sgnn" }, { text: "subnet-nvupa1uw" }],
    orgLines: [{ text: "vpc-gn16sgnn" }, { text: "subnet-nvupa1uw" }, { text: "subnet-a3bc82kx" }],
    isSame: true,
  },
  {
    category: "网络 > 安全组",
    instanceLines: [{ text: "sg-tech4n7w" }],
    orgLines: [{ text: "sg-tech4n7w" }],
    isSame: true,
  },
  {
    category: "网络 > 公网",
    instanceLines: [{ text: "公网 IP：已分配" }, { text: "计费模式：按流量计费" }, { text: "带宽上限：10 Mbps" }],
    orgLines: [{ text: "公网 IP：已分配" }, { text: "计费模式：按流量计费" }, { text: "带宽上限：10 Mbps" }],
    isSame: true,
  },
  {
    category: "CLS 日志服务",
    instanceLines: [{ text: "开启" }],
    orgLines: [{ text: "开启" }],
    isSame: true,
  },
  {
    category: "AI Agent 安全",
    instanceLines: [{ text: "开启" }],
    orgLines: [{ text: "开启" }],
    isSame: true,
  },
  {
    category: "平台策略 > 用户配额",
    instanceLines: [
      { text: "单用户 Agent 数量上限：60", diff: true },
      { text: "单用户 Tokens 上限：2026/06/09 00:00 – 2026/06/16 00:00，按日刷新 无限制", diff: true },
    ],
    orgLines: [
      { text: "单用户 Agent 数量上限：90" },
      { text: "单用户 Tokens 上限：2026/06/16 00:00 – 2026/06/25 00:00，按日刷新 无限制" },
    ],
    isSame: false,
    autoApply: true,
  },
  {
    category: "平台策略 > 模型配额",
    instanceLines: [{ text: "全局 Tokens 上限：2026/06/01 17:26 – 无终止，按月刷新 无限制", diff: true }],
    orgLines: [{ text: "全局 Tokens 上限：2026/06/09 17:26 – 无终止，按月刷新 无限制" }],
    isSame: false,
    autoApply: true,
  },
  {
    category: "平台策略 > 功能权限",
    instanceLines: [
      { text: "允许配置模型 ✓" },
      { text: "允许配置通道 ✓" },
      { text: "允许进入终端 ✓", diff: true },
      { text: "允许访问面板 ✓" },
      { text: "允许使用对话视图 ✓" },
      { text: "允许访问云端浏览器 ✓" },
      { text: "允许使用龙虾医生 ✓" },
      { text: "允许查看模型额度 ✓" },
      { text: "允许添加自定义模型 ✓", diff: true },
    ],
    orgLines: [
      { text: "允许配置模型 ✓" },
      { text: "允许配置通道 ✓" },
      { text: "允许进入终端 ✗" },
      { text: "允许访问面板 ✓" },
      { text: "允许使用对话视图 ✓" },
      { text: "允许访问云端浏览器 ✓" },
      { text: "允许使用龙虾医生 ✓" },
      { text: "允许查看模型额度 ✓" },
      { text: "允许添加自定义模型 ✗" },
    ],
    isSame: false,
    autoApply: true,
  },
];

/**
 * 取某实例与所属组织的逐项配置对比（mock）。
 *   - isMatch=false：返回原型「代码评审助手」一例的差异数据；
 *   - isMatch=true ：所有项均一致（实例配置 = 组织配置）。
 */
export function getMockInstanceCompareItems(isMatch: boolean): CompareConfigItem[] {
  if (!isMatch) return MOCK_COMPARE_ITEMS_MISMATCH;
  // 全部符合：实例配置取组织配置，去除 diff 高亮与 autoApply 标记
  return MOCK_COMPARE_ITEMS_MISMATCH.map((item) => ({
    category: item.category,
    instanceLines: item.orgLines.map((l) => ({ text: l.text })),
    orgLines: item.orgLines.map((l) => ({ text: l.text })),
    isSame: true,
  }));
}
