/**
 * AgentOpsPageHeader - 「Agent 运维」组织下各页面的统一页眉
 *
 * 在 IA 重构后（2026-05），路由布局：
 *   /admin/agent-types       Agent 类型（已并入「Agent 配置」组织）
 *   /admin/agent-versions    Agent 版本（含 [版本更新][更新记录] 两 Tab）
 *   /admin/agent-commands    执行命令（含 [命令模板][执行记录] 两 Tab）
 *   /admin/agent-history     已下线，仅作兼容重定向
 *
 * 这些页面不再挂在侧边栏，仅作为「Agent 列表」入口的延展。
 * 因此页眉提供「返回 Agent 列表」入口，便于用户回到主入口。
 */
import { ArrowLeft } from "lucide-react";
import { Link } from "wouter";

interface Props {
  title: string;
  description: string;
  /** 返回链接，默认指向 Agent 列表 */
  backTo?: string;
  /** 返回按钮文案，默认「返回 Agent 列表」 */
  backLabel?: string;
}

export default function AgentOpsPageHeader({
  title,
  description,
  backTo = "/admin/openclaw-monitor",
  backLabel = "返回 Agent 列表",
}: Props) {
  return (
    <div className="mb-6">
      <Link href={backTo}>
        <button
          type="button"
          className="inline-flex items-center gap-1 text-xs text-[#525252] hover:text-[#355EF1] transition-colors mb-2 group"
        >
          <ArrowLeft className="w-3.5 h-3.5 group-hover:-translate-x-0.5 transition-transform" />
          {backLabel}
        </button>
      </Link>
      <h1 className="text-2xl font-bold text-[#0A0A0A]">{title}</h1>
      <p className="text-sm text-[#737373] mt-1">{description}</p>
    </div>
  );
}
