/**
 * AgentTemplate - Agent 模板管理页
 * 管控端「云设备配置」组织下，供管理员统一管理和分发 Agent 模板
 *
 * ⚠️ 未接入路由：本组件当前未在 App.tsx 中引用，属保留备用的孤儿组件。
 * 路由 /admin/agent-template 实际渲染的是 ResourceManagement.tsx（资源管理 / 计费模式页）。
 * 修改本文件不会反映到任何线上页面，接入前请勿据此预览验证。
 */

import {
  Empty,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
  EmptyDescription,
} from "@/components/ui/empty";

export default function AgentTemplate() {
  return (
    <div className="page-enter flex h-full min-h-[calc(100vh-200px)] items-center justify-center">
      <Empty className="max-w-[590px] border-none">
        <EmptyHeader className="max-w-none">
          <EmptyMedia variant="hint" />
          <EmptyTitle>Agent 模板</EmptyTitle>
          <EmptyDescription>
            在此统一管理企业内可复用的 Agent 模板，包括预设的系统提示词、工具配置与模型参数。
            <br />
            管理员可发布模板供用户一键创建标准化 Agent，降低配置门槛，保障使用规范。
          </EmptyDescription>
        </EmptyHeader>
      </Empty>
    </div>
  );
}
