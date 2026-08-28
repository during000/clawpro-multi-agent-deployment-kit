/**
 * HelpPanel - 顶部导航的「使用指南」（右侧抽屉，非模态）
 *
 * 组件结构严格遵循 shadcn/ui Sheet 规范，与 NotificationPanel 保持一致：
 *   Sheet
 *   ├── SheetTrigger（HelpIcon 按钮）
 *   └── SheetContent
 *       ├── SheetHeader
 *       │   ├── SheetTitle + 关闭按钮
 *       │   └── SheetDescription (sr-only)
 *       └── 主体内容：Tabs（Segmented Control） + 文档正文滚动区
 *
 * 4 个 Tab 对应原 HelpDocs 页面的 4 张卡片：
 *   - 概念介绍（concept）
 *   - 功能特色（features）
 *   - 部署指引（deploy）
 *   - 进阶玩法（advanced）
 */
import { useState } from "react";
import { BookOpen, FileText, Rocket, Star, X } from "lucide-react";
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { Button } from "@/components/ui/button";
import NavIconButton from "./NavIconButton";
import { HelpIcon } from "./NavIcons";

type DocKey = "concept" | "features" | "deploy" | "advanced";

interface DocCategory {
  id: DocKey;
  icon: React.ComponentType<{ className?: string }>;
  color: string;
  title: string;
  tabLabel: string;
  content: string;
}

const DOC_CATEGORIES: DocCategory[] = [
  {
    id: "concept",
    icon: BookOpen,
    color: "from-blue-500 to-blue-600",
    title: "OpenClaw 概念介绍",
    tabLabel: "概念介绍",
    content: `# OpenClaw 概念介绍

## 什么是 Agent？

Agent 是一个开源的 AI Agent 框架，专为企业和个人用户设计，让你能够快速创建、部署和管理专属的 AI 智能助理。

## 核心概念

### Agent（智能体）
Agent 的核心是 Agent，即 AI 智能助理实例。每个 Agent 都有独立的配置，包括使用的大模型、接入的通道和安装的技能。

### 模型（Model）
模型是 Agent 的"大脑"，决定了 Agent 的智能水平和能力范围。Agent 支持接入多种主流大模型，包括腾讯云 DeepSeek、混元等。

### 通道（Channel）
通道是用户与 Agent 交互的入口，支持企业微信、飞书、钉钉、QQ 等主流即时通讯工具。

### 技能（Skill）
技能是 Agent 的能力扩展，通过安装不同的技能插件，Agent 可以执行搜索、文档处理、代码生成等特定任务。

## 工作原理

1. 用户通过 IM 工具发送消息
2. 消息通过通道传入 Agent
3. Agent 调用大模型处理请求
4. 如需要，调用相关技能执行任务
5. 将结果返回给用户`,
  },
  {
    id: "features",
    icon: Star,
    color: "from-purple-500 to-purple-600",
    title: "ClawPro平台的功能与特色",
    tabLabel: "功能特色",
    content: `# ClawPro平台的功能与特色

## 企业版核心优势

### 多成员协同
支持企业内多名成员各自创建和管理专属 Agent，统一在企业账号体系下管理，互不干扰。

### 企业级安全管控
- 完善的成员权限管理
- Tokens 配额控制
- 操作审计日志
- 确保 AI 使用在可控范围内

### 集中化配置管理
管理员可统一配置可用模型、通道和帮助文档，用户无需关心底层配置，专注于使用 AI 提升工作效率。

### 云端部署，24小时随时可用
部署在腾讯云服务器上，7×24 小时稳定运行，随时随地通过 IM 工具与你的 AI 助理对话。

### 一键配置，小白也能快速上手
极简的创建流程，只需输入名称即可创建 Agent，再按步骤配置通道，几分钟内即可拥有专属 AI 助理。

### 实时监控与审计
全面的运营监控面板，实时掌握 Agent 运行状态和 Tokens 消耗情况，操作记录全程可追溯。`,
  },
  {
    id: "deploy",
    icon: Rocket,
    color: "from-green-500 to-green-600",
    title: "部署 Agent 指引",
    tabLabel: "部署指引",
    content: `# 部署 Agent 指引

## 快速开始

### 第一步：创建 Agent

1. 登录企业版 Agent 平台
2. 进入「我的 Agent」页面
3. 点击「创建 Agent」按钮
4. 输入一个有意义的名称（如「工作助手」、「代码助手」）
5. 点击「创建」完成创建

### 第二步：配置模型

1. 点击刚创建的 Agent 卡片，进入详细配置页
2. 在「模型」面板中，选择你想使用的大模型（如腾讯云 DeepSeek）
3. 选择具体的模型版本（如 DeepSeek V3 0324）
4. 输入对应的 API Key
5. 点击「添加并应用」

> 提示：API Key 由企业管理员统一配置，如果你看到模型列表中已有可用模型，可以直接使用，无需额外配置。

### 第三步：配置通道

1. 在「通道」面板中，选择你想接入的 IM 工具（如企业微信、飞书、QQ）
2. 填入对应的配置信息（App ID、App Secret 等）
3. 点击「添加并应用」

### 第四步：安装技能（可选）

1. 在「技能」面板中，搜索你需要的技能
2. 点击「安装技能」完成安装

### 第五步：开始使用

配置完成后，你的 Agent 就会开始运行。打开你配置的 IM 工具，找到对应的机器人，开始对话吧！`,
  },
  {
    id: "advanced",
    icon: FileText,
    color: "from-orange-500 to-orange-600",
    title: "OpenClaw 进阶玩法",
    tabLabel: "进阶玩法",
    content: `# OpenClaw 进阶玩法

## 多模型切换

你可以为同一个 Agent 配置多个模型，根据不同的任务需求灵活切换：

- **日常对话**：使用轻量级模型，响应更快、成本更低
- **复杂推理**：切换到更强大的模型，获得更准确的答案
- **代码生成**：使用专门针对代码优化的模型

## 技能组合

通过组合不同的技能，可以打造功能强大的专业助手：

### 研究助手
- tavily-search：实时网络搜索
- summarize：内容摘要
- tencent-docs：文档处理

### 开发助手
- github：代码仓库管理
- agent-browser：网页浏览
- ai-ppt-generator：自动生成演示文稿

### 内容创作助手
- xhs-skill：小红书内容创作
- weather：天气信息获取
- notion：笔记管理

## 多通道策略

不同的通道适合不同的使用场景：

- **企业微信**：适合工作场景，与企业内部系统集成
- **飞书**：适合有飞书办公套件的团队
- **QQ**：适合个人使用和非正式沟通
- **钉钉**：适合阿里系企业用户

## 最佳实践

1. **为不同场景创建不同的 Agent**，避免一个助手承担过多职责
2. **定期检查 Tokens 消耗**，合理规划使用量
3. **及时更新技能版本**，获取最新功能和修复
4. **善用重启功能**，遇到异常时快速恢复`,
  },
];

/**
 * 去掉 Markdown 内容开头与 Tab 文字重复的一级标题行：
 * 例如内容以「# OpenClaw 概念介绍\n\n」开头时，会被整体剔除，
 * 避免和 Tab 文字重复显示。
 */
const stripLeadingTitle = (content: string): string => {
  const lines = content.split("\n");
  if (lines[0]?.startsWith("# ")) {
    lines.shift();
    // 同时吃掉紧随其后的空行
    while (lines.length > 0 && lines[0].trim() === "") {
      lines.shift();
    }
  }
  return lines.join("\n");
};

/** 简易 Markdown 渲染（与 HelpDocs.tsx 保持一致的样式规则） */
const renderMarkdown = (content: string) =>
  content.split("\n").map((line, i) => {
    if (line.startsWith("# "))
      return (
        <h1 key={i} className="text-xl font-bold text-gray-900 mb-4">
          {line.slice(2)}
        </h1>
      );
    if (line.startsWith("## "))
      return (
        <h2
          key={i}
          className="text-base font-semibold text-gray-900 mt-5 mb-2.5"
        >
          {line.slice(3)}
        </h2>
      );
    if (line.startsWith("### "))
      return (
        <h3
          key={i}
          className="text-sm font-semibold text-gray-800 mt-4 mb-2"
        >
          {line.slice(4)}
        </h3>
      );
    if (line.startsWith("- "))
      return (
        <li key={i} className="ml-4 text-xs text-gray-600 mb-1">
          {line.slice(2)}
        </li>
      );
    if (line.startsWith("> "))
      return (
        <blockquote
          key={i}
          className="border-l-4 border-blue-200 pl-3 text-xs text-gray-500 italic my-2.5"
        >
          {line.slice(2)}
        </blockquote>
      );
    if (line.match(/^\d+\./))
      return (
        <p key={i} className="ml-4 text-xs text-gray-600 mb-1">
          {line}
        </p>
      );
    if (line.startsWith("**") && line.endsWith("**"))
      return (
        <p key={i} className="text-xs font-semibold text-gray-800 mb-1">
          {line.slice(2, -2)}
        </p>
      );
    if (line === "") return <div key={i} className="h-2" />;
    return (
      <p key={i} className="text-xs text-gray-600 mb-2 leading-relaxed">
        {line}
      </p>
    );
  });

export default function HelpPanel() {
  const [showPanel, setShowPanel] = useState(false);
  const [activeTab, setActiveTab] = useState<DocKey>("concept");

  return (
    <Sheet open={showPanel} onOpenChange={setShowPanel} modal={false}>
      <Tooltip>
        <TooltipTrigger asChild>
          <SheetTrigger asChild>
            <NavIconButton icon={<HelpIcon />} />
          </SheetTrigger>
        </TooltipTrigger>
        <TooltipContent side="bottom" sideOffset={6}>
          使用指南
        </TooltipContent>
      </Tooltip>

      <SheetContent
        side="right"
        showOverlay={false}
        className="!w-[420px] !max-w-none !top-[64px] !bottom-0 !h-[calc(100vh-64px)] p-0 flex flex-col gap-0 border-t [&>[data-slot=sheet-close]]:hidden"
      >
        {/* ───── shadcn 规范：SheetHeader > SheetTitle + SheetDescription ───── */}
        <SheetHeader className="px-5 pt-5 pb-4 border-b border-[#E5E5E5] gap-0 space-y-0">
          <div className="flex items-center justify-between">
            <SheetTitle className="text-base font-semibold text-[#0A0A0A]">
              使用指南
            </SheetTitle>
            <SheetClose asChild>
              <Button
                variant="ghost"
                size="icon-sm"
                aria-label="关闭"
                className="h-7 w-7 text-[#737373] hover:text-[#0A0A0A]"
              >
                <X className="w-4 h-4" />
              </Button>
            </SheetClose>
          </div>
          <SheetDescription className="sr-only">
            查看帮助文档，包括概念介绍、功能特色、部署指引、进阶玩法四大类。
          </SheetDescription>
        </SheetHeader>

        {/* ───── 主体内容：Tabs（Segmented Control）+ 滚动正文 ───── */}
        <Tabs
          value={activeTab}
          onValueChange={(v) => setActiveTab(v as DocKey)}
          className="flex flex-col flex-1 min-h-0 gap-0"
        >
          {/* §8.6 Tab 切换（Segmented Control，0522 胶囊版） */}
          <div className="px-3 pt-3 pb-0">
            <TabsList
              className="flex items-center gap-1 p-1 h-auto rounded-full w-full bg-muted"
            >
              {DOC_CATEGORIES.map((doc) => (
                <TabsTrigger
                  key={doc.id}
                  value={doc.id}
                  className="flex-1 rounded-full px-3 py-1 text-xs font-normal whitespace-nowrap text-muted-foreground hover:text-foreground data-[state=active]:bg-white data-[state=active]:text-foreground data-[state=active]:font-medium data-[state=active]:shadow-[var(--shadow-segment)] transition-colors flex items-center justify-center gap-1"
                >
                  {doc.tabLabel}
                </TabsTrigger>
              ))}
            </TabsList>
          </div>

          <div
            className="overflow-y-auto flex-1 min-h-0"
            style={{ scrollbarWidth: "thin", scrollbarColor: "#d1d5db #f3f4f6" }}
          >
            {DOC_CATEGORIES.map((doc) => (
              <TabsContent key={doc.id} value={doc.id} className="m-0">
                <div className="px-5 pt-4 pb-4">
                  <div className="prose prose-sm max-w-none text-gray-700 leading-relaxed">
                    {renderMarkdown(stripLeadingTitle(doc.content))}
                  </div>
                </div>
              </TabsContent>
            ))}
          </div>
        </Tabs>
      </SheetContent>
    </Sheet>
  );
}
